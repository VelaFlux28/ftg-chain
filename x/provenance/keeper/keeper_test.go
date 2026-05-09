package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/VelaFlux28/ftg-chain/x/provenance/keeper"
	"github.com/VelaFlux28/ftg-chain/x/provenance/types"
)

// setupProvenanceKeeper creates a test provenance keeper with an in-memory store
func setupProvenanceKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, storeKey, log.NewNopLogger())
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, ctx
}

func TestCreateProvenanceRecord_Success(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))

	energySource := types.EnergySource{
		Type:                  "solar",
		Provider:              "TerraPass",
		Location:              "California, USA",
		GenerationPeriodStart: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GenerationPeriodEnd:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		MwhQuantity:           math.LegacyNewDecWithPrec(617, 0),
		CertificateSerial:     "TERR-2025-617MWH",
		CertificateRegistry:   "Green-e",
	}

	docs := []types.DocumentHash{
		{
			IPFSHash:     "QmGenesisRECCertificate",
			DocumentType: "certificate",
			Description:  "Original REC certificate from TerraPass for 617 MWh",
			UploadedAt:   time.Now().UTC(),
			FileSize:     1024000,
			MimeType:     "application/pdf",
		},
	}

	record, err := k.CreateProvenanceRecord(
		ctx,
		"FTG-CERT-000001",
		1,
		61700,
		"proof_of_purchase",
		energySource,
		docs,
		creator,
	)

	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "FTG-PROV-000001", record.RecordID)
	require.Equal(t, "FTG-CERT-000001", record.CertificateID)
	require.Equal(t, uint64(1), record.TokenBatchStart)
	require.Equal(t, uint64(61700), record.TokenBatchEnd)
	require.Equal(t, "solar", record.EnergySource.Type)
	require.Equal(t, "TerraPass", record.EnergySource.Provider)
	require.Len(t, record.DocumentHashes, 1)
	require.Len(t, record.ChainOfCustody, 1)
	require.Equal(t, "initial_mint", record.ChainOfCustody[0].Reason)
}

func TestCreateProvenanceRecord_DuplicateCertificate(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	energySource := types.EnergySource{
		Type:     "solar",
		Provider: "TerraPass",
		Location: "California",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
	}
	docs := []types.DocumentHash{{
		IPFSHash:     "QmTest",
		DocumentType: "certificate",
		UploadedAt:   time.Now(),
	}}

	// First record should succeed
	_, err := k.CreateProvenanceRecord(ctx, "CERT-001", 1, 1000, "proof_of_purchase", energySource, docs, creator)
	require.NoError(t, err)

	// Duplicate should fail
	_, err = k.CreateProvenanceRecord(ctx, "CERT-001", 1001, 2000, "proof_of_purchase", energySource, docs, creator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists for certificate")
}

func TestCreateProvenanceRecord_MissingDocument(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	energySource := types.EnergySource{
		Type:     "solar",
		Provider: "TerraPass",
		Location: "California",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
	}

	// No documents — should fail because RequireDocumentHash is true by default
	_, err := k.CreateProvenanceRecord(ctx, "CERT-002", 1, 1000, "proof_of_purchase", energySource, nil, creator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one document hash is required")
}

func TestCreateProvenanceRecord_InvalidEnergyType(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	energySource := types.EnergySource{
		Type:     "nuclear", // Not in accepted list
		Provider: "NuclearCorp",
		Location: "France",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
	}
	docs := []types.DocumentHash{{
		IPFSHash:     "QmTest",
		DocumentType: "certificate",
		UploadedAt:   time.Now(),
	}}

	_, err := k.CreateProvenanceRecord(ctx, "CERT-003", 1, 1000, "proof_of_purchase", energySource, docs, creator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "energy type nuclear is not accepted")
}

func TestGetProvenanceByCertificate(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	energySource := types.EnergySource{
		Type:     "wind",
		Provider: "WindFarm Inc",
		Location: "Texas, USA",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(50, 0),
	}
	docs := []types.DocumentHash{{
		IPFSHash:     "QmWindCert",
		DocumentType: "certificate",
		UploadedAt:   time.Now(),
	}}

	created, err := k.CreateProvenanceRecord(ctx, "WIND-CERT-001", 1, 5000, "proof_of_generation", energySource, docs, creator)
	require.NoError(t, err)

	// Retrieve by certificate ID
	retrieved, found := k.GetProvenanceByCertificate(ctx, "WIND-CERT-001")
	require.True(t, found)
	require.Equal(t, created.RecordID, retrieved.RecordID)
	require.Equal(t, "wind", retrieved.EnergySource.Type)
}

func TestAddDocumentToRecord(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	energySource := types.EnergySource{
		Type:     "solar",
		Provider: "TerraPass",
		Location: "California",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
	}
	docs := []types.DocumentHash{{
		IPFSHash:     "QmOriginalCert",
		DocumentType: "certificate",
		UploadedAt:   time.Now(),
	}}

	record, err := k.CreateProvenanceRecord(ctx, "CERT-DOC-001", 1, 1000, "proof_of_purchase", energySource, docs, creator)
	require.NoError(t, err)
	require.Len(t, record.DocumentHashes, 1)

	// Add an audit report
	newDoc := types.DocumentHash{
		IPFSHash:     "QmAuditReport2026",
		DocumentType: "audit_report",
		Description:  "Third-party verification of REC certificates",
		UploadedAt:   time.Now().UTC(),
		FileSize:     2048000,
		MimeType:     "application/pdf",
	}

	err = k.AddDocumentToRecord(ctx, record.RecordID, newDoc)
	require.NoError(t, err)

	// Verify the document was added
	updated, found := k.GetProvenanceRecord(ctx, record.RecordID)
	require.True(t, found)
	require.Len(t, updated.DocumentHashes, 2)
	require.Equal(t, "QmAuditReport2026", updated.DocumentHashes[1].IPFSHash)
}

func TestRecordCustodyTransfer(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("nick-treasury-addr"))
	buyer := sdk.AccAddress([]byte("buyer-wallet-addr"))

	energySource := types.EnergySource{
		Type:     "solar",
		Provider: "TerraPass",
		Location: "California",
		GenerationPeriodStart: time.Now(),
		GenerationPeriodEnd:   time.Now(),
		MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
	}
	docs := []types.DocumentHash{{
		IPFSHash:     "QmCert",
		DocumentType: "certificate",
		UploadedAt:   time.Now(),
	}}

	record, err := k.CreateProvenanceRecord(ctx, "CERT-TRANSFER-001", 1, 1000, "proof_of_purchase", energySource, docs, creator)
	require.NoError(t, err)
	require.Len(t, record.ChainOfCustody, 1) // initial_mint

	// Record a sale transfer
	err = k.RecordCustodyTransfer(ctx, record.RecordID, creator, buyer, "sale", "TX-HASH-001")
	require.NoError(t, err)

	// Verify custody chain
	updated, found := k.GetProvenanceRecord(ctx, record.RecordID)
	require.True(t, found)
	require.Len(t, updated.ChainOfCustody, 2)
	require.Equal(t, "sale", updated.ChainOfCustody[1].Reason)
	require.Equal(t, "TX-HASH-001", updated.ChainOfCustody[1].TxHash)
}

func TestRecordIDGeneration(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	id1 := k.GenerateRecordID(ctx)
	id2 := k.GenerateRecordID(ctx)
	id3 := k.GenerateRecordID(ctx)

	require.Equal(t, "FTG-PROV-000001", id1)
	require.Equal(t, "FTG-PROV-000002", id2)
	require.Equal(t, "FTG-PROV-000003", id3)
}

func TestCreateProvenanceRecord_AllEnergyTypes(t *testing.T) {
	k, ctx := setupProvenanceKeeper(t)

	creator := sdk.AccAddress([]byte("creator-address"))
	acceptedTypes := []string{"solar", "wind", "hydro", "geothermal", "biomass", "mixed_renewable"}

	for i, energyType := range acceptedTypes {
		energySource := types.EnergySource{
			Type:     energyType,
			Provider: "Provider-" + energyType,
			Location: "Location-" + energyType,
			GenerationPeriodStart: time.Now(),
			GenerationPeriodEnd:   time.Now(),
			MwhQuantity: math.LegacyNewDecWithPrec(10, 0),
		}
		docs := []types.DocumentHash{{
			IPFSHash:     "Qm" + energyType,
			DocumentType: "certificate",
			UploadedAt:   time.Now(),
		}}

		certID := fmt.Sprintf("CERT-TYPE-%03d", i+1)
		record, err := k.CreateProvenanceRecord(ctx, certID, uint64(i*1000+1), uint64((i+1)*1000), "proof_of_purchase", energySource, docs, creator)
		require.NoError(t, err, "Failed for energy type: %s", energyType)
		require.Equal(t, energyType, record.EnergySource.Type)
	}
}
