package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "provenance"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

// ProvenanceRecord is the immutable on-chain record linking a token to its energy source
// This is the audit trail that proves every FTG token is backed by real energy
type ProvenanceRecord struct {
	// RecordID is the unique on-chain identifier
	RecordID string `json:"record_id"`

	// CertificateID links to the energymint module's certificate
	CertificateID string `json:"certificate_id"`

	// TokenBatchStart is the first token ID in this batch
	TokenBatchStart uint64 `json:"token_batch_start"`

	// TokenBatchEnd is the last token ID in this batch
	TokenBatchEnd uint64 `json:"token_batch_end"`

	// ProvenanceType classifies the energy source
	ProvenanceType string `json:"provenance_type"`

	// EnergySource contains details about the energy origin
	EnergySource EnergySource `json:"energy_source"`

	// DocumentHashes contains all IPFS hashes for supporting documents
	DocumentHashes []DocumentHash `json:"document_hashes"`

	// CreatedAt is when this record was created
	CreatedAt time.Time `json:"created_at"`

	// CreatedBy is the address that created this record
	CreatedBy sdk.AccAddress `json:"created_by"`

	// ChainOfCustody tracks ownership transfers of the underlying certificate
	ChainOfCustody []CustodyEvent `json:"chain_of_custody"`
}

// EnergySource describes where the energy came from
type EnergySource struct {
	// Type: "solar", "wind", "hydro", "geothermal", "biomass", "mixed_renewable"
	Type string `json:"type"`

	// Provider is the utility or generator name
	Provider string `json:"provider"`

	// Location is the geographic location (state/country)
	Location string `json:"location"`

	// GenerationPeriodStart is when energy generation began
	GenerationPeriodStart time.Time `json:"generation_period_start"`

	// GenerationPeriodEnd is when energy generation ended
	GenerationPeriodEnd time.Time `json:"generation_period_end"`

	// MwhQuantity is the total energy in MWh
	MwhQuantity math.LegacyDec `json:"mwh_quantity"`

	// CertificateSerial is the external certificate serial number
	CertificateSerial string `json:"certificate_serial"`

	// CertificateRegistry is where the cert is registered (e.g., "Green-e", "M-RETS", "WREGIS")
	CertificateRegistry string `json:"certificate_registry"`

	// VaultID is set if energy comes from a Strategic Energy Vault
	VaultID string `json:"vault_id,omitempty"`

	// MCSDeviceID is set if energy was metered by a patented MCS device
	MCSDeviceID string `json:"mcs_device_id,omitempty"`
}

// DocumentHash links an IPFS document to this provenance record
type DocumentHash struct {
	// IPFSHash is the IPFS CID (Content Identifier)
	IPFSHash string `json:"ipfs_hash"`

	// DocumentType classifies the document
	// "certificate" = Original REC certificate
	// "invoice" = Purchase invoice
	// "meter_reading" = MCS meter reading
	// "audit_report" = Third-party audit
	// "photo" = Physical asset photo
	DocumentType string `json:"document_type"`

	// Description is a human-readable description
	Description string `json:"description"`

	// UploadedAt is when the document was uploaded
	UploadedAt time.Time `json:"uploaded_at"`

	// FileSize in bytes
	FileSize uint64 `json:"file_size"`

	// MimeType of the document
	MimeType string `json:"mime_type"`
}

// CustodyEvent tracks a transfer in the chain of custody
type CustodyEvent struct {
	// From is the previous custodian
	From sdk.AccAddress `json:"from"`

	// To is the new custodian
	To sdk.AccAddress `json:"to"`

	// Timestamp is when the transfer occurred
	Timestamp time.Time `json:"timestamp"`

	// Reason describes why the transfer happened
	// "initial_mint", "sale", "transfer", "redemption"
	Reason string `json:"reason"`

	// TxHash is the transaction that recorded this transfer
	TxHash string `json:"tx_hash"`
}

// TokenLineage allows any token holder to trace their token back to the original energy source
type TokenLineage struct {
	// TokenID is the specific token being traced
	TokenID uint64 `json:"token_id"`

	// CurrentOwner is who holds the token now
	CurrentOwner sdk.AccAddress `json:"current_owner"`

	// ProvenanceRecordID links to the full provenance record
	ProvenanceRecordID string `json:"provenance_record_id"`

	// OriginalMintTx is the transaction that created this token
	OriginalMintTx string `json:"original_mint_tx"`

	// TransferHistory is the complete ownership history
	TransferHistory []CustodyEvent `json:"transfer_history"`
}

// GenesisState defines the provenance module's genesis state
type GenesisState struct {
	Params  Params             `json:"params"`
	Records []ProvenanceRecord `json:"records"`
}

// Params defines the parameters for the provenance module
type Params struct {
	// RequireDocumentHash determines if at least one document hash is required
	RequireDocumentHash bool `json:"require_document_hash"`

	// MaxDocumentsPerRecord limits documents per provenance record
	MaxDocumentsPerRecord uint32 `json:"max_documents_per_record"`

	// AcceptedDocumentTypes lists valid document types
	AcceptedDocumentTypes []string `json:"accepted_document_types"`

	// AcceptedEnergyTypes lists valid energy source types
	AcceptedEnergyTypes []string `json:"accepted_energy_types"`
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		RequireDocumentHash:   true,
		MaxDocumentsPerRecord: 20,
		AcceptedDocumentTypes: []string{
			"certificate", "invoice", "meter_reading", "audit_report", "photo",
		},
		AcceptedEnergyTypes: []string{
			"solar", "wind", "hydro", "geothermal", "biomass", "mixed_renewable",
		},
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:  DefaultParams(),
		Records: []ProvenanceRecord{},
	}
}
