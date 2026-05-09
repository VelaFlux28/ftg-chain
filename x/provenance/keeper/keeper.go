package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/provenance/types"
)

// Keeper of the provenance store
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	logger   log.Logger
}

// NewKeeper creates a new provenance Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		logger:   logger,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// CreateProvenanceRecord creates an immutable provenance record for a minted token batch
func (k Keeper) CreateProvenanceRecord(
	ctx sdk.Context,
	certificateID string,
	tokenBatchStart uint64,
	tokenBatchEnd uint64,
	provenanceType string,
	energySource types.EnergySource,
	documents []types.DocumentHash,
	creator sdk.AccAddress,
) (*types.ProvenanceRecord, error) {
	k.Logger().Info("CreateProvenanceRecord",
		"certificate_id", certificateID,
		"token_batch", fmt.Sprintf("%d-%d", tokenBatchStart, tokenBatchEnd),
		"provenance_type", provenanceType,
	)

	params := k.GetParams(ctx)

	if params.RequireDocumentHash && len(documents) == 0 {
		return nil, fmt.Errorf("at least one document hash is required")
	}

	if uint32(len(documents)) > params.MaxDocumentsPerRecord {
		return nil, fmt.Errorf("too many documents: %d (max %d)", len(documents), params.MaxDocumentsPerRecord)
	}

	if !isAcceptedType(params.AcceptedEnergyTypes, energySource.Type) {
		return nil, fmt.Errorf("energy type %s is not accepted", energySource.Type)
	}

	for _, doc := range documents {
		if !isAcceptedType(params.AcceptedDocumentTypes, doc.DocumentType) {
			return nil, fmt.Errorf("document type %s is not accepted", doc.DocumentType)
		}
	}

	if k.HasProvenanceForCertificate(ctx, certificateID) {
		return nil, fmt.Errorf("provenance record already exists for certificate %s", certificateID)
	}

	record := types.ProvenanceRecord{
		RecordID:        k.GenerateRecordID(ctx),
		CertificateID:   certificateID,
		TokenBatchStart: tokenBatchStart,
		TokenBatchEnd:   tokenBatchEnd,
		ProvenanceType:  provenanceType,
		EnergySource:    energySource,
		DocumentHashes:  documents,
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       creator,
		ChainOfCustody: []types.CustodyEvent{
			{
				From:      nil,
				To:        creator,
				Timestamp: time.Now().UTC(),
				Reason:    "initial_mint",
			},
		},
	}

	k.SetProvenanceRecord(ctx, record)
	k.SetCertToProvenance(ctx, certificateID, record.RecordID)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provenance_created",
			sdk.NewAttribute("record_id", record.RecordID),
			sdk.NewAttribute("certificate_id", certificateID),
			sdk.NewAttribute("energy_type", energySource.Type),
			sdk.NewAttribute("provider", energySource.Provider),
			sdk.NewAttribute("mwh_quantity", energySource.MwhQuantity.String()),
			sdk.NewAttribute("documents_count", fmt.Sprintf("%d", len(documents))),
		),
	)

	return &record, nil
}

// GetProvenanceRecord retrieves a provenance record by ID
func (k Keeper) GetProvenanceRecord(ctx sdk.Context, recordID string) (types.ProvenanceRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ProvenanceRecordKey(recordID))
	if bz == nil {
		return types.ProvenanceRecord{}, false
	}
	var record types.ProvenanceRecord
	json.Unmarshal(bz, &record)
	return record, true
}

// SetProvenanceRecord stores a provenance record using JSON encoding
func (k Keeper) SetProvenanceRecord(ctx sdk.Context, record types.ProvenanceRecord) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&record)
	store.Set(types.ProvenanceRecordKey(record.RecordID), bz)
}

// HasProvenanceForCertificate checks if a provenance record exists for a certificate
func (k Keeper) HasProvenanceForCertificate(ctx sdk.Context, certID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.CertToProvenanceKey(certID))
}

// SetCertToProvenance maps a certificate ID to a provenance record ID
func (k Keeper) SetCertToProvenance(ctx sdk.Context, certID string, recordID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CertToProvenanceKey(certID), []byte(recordID))
}

// GetProvenanceByCertificate returns the provenance record for a certificate
func (k Keeper) GetProvenanceByCertificate(ctx sdk.Context, certID string) (types.ProvenanceRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CertToProvenanceKey(certID))
	if bz == nil {
		return types.ProvenanceRecord{}, false
	}
	return k.GetProvenanceRecord(ctx, string(bz))
}

// AddDocumentToRecord adds a new document hash to an existing provenance record
func (k Keeper) AddDocumentToRecord(ctx sdk.Context, recordID string, doc types.DocumentHash) error {
	record, found := k.GetProvenanceRecord(ctx, recordID)
	if !found {
		return fmt.Errorf("provenance record %s not found", recordID)
	}

	params := k.GetParams(ctx)
	if uint32(len(record.DocumentHashes)+1) > params.MaxDocumentsPerRecord {
		return fmt.Errorf("maximum documents reached for record %s", recordID)
	}

	record.DocumentHashes = append(record.DocumentHashes, doc)
	k.SetProvenanceRecord(ctx, record)
	return nil
}

// RecordCustodyTransfer records a transfer in the chain of custody
func (k Keeper) RecordCustodyTransfer(
	ctx sdk.Context,
	recordID string,
	from sdk.AccAddress,
	to sdk.AccAddress,
	reason string,
	txHash string,
) error {
	record, found := k.GetProvenanceRecord(ctx, recordID)
	if !found {
		return fmt.Errorf("provenance record %s not found", recordID)
	}

	event := types.CustodyEvent{
		From:      from,
		To:        to,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
		TxHash:    txHash,
	}

	record.ChainOfCustody = append(record.ChainOfCustody, event)
	k.SetProvenanceRecord(ctx, record)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"custody_transfer",
			sdk.NewAttribute("record_id", recordID),
			sdk.NewAttribute("from", from.String()),
			sdk.NewAttribute("to", to.String()),
			sdk.NewAttribute("reason", reason),
		),
	)

	return nil
}

// GenerateRecordID creates a unique provenance record ID
func (k Keeper) GenerateRecordID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ProvenanceCountKey))
	var count uint64
	if bz != nil {
		count = sdk.BigEndianToUint64(bz)
	}
	count++
	store.Set([]byte(types.ProvenanceCountKey), sdk.Uint64ToBigEndian(count))
	return fmt.Sprintf("FTG-PROV-%06d", count)
}

// GetParams returns the current module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.DefaultParams()
}

// helper
func isAcceptedType(accepted []string, t string) bool {
	for _, a := range accepted {
		if a == t {
			return true
		}
	}
	return false
}
