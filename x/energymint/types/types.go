package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "energymint"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// TokenDenom is the denomination of the FTG token
	TokenDenom = "uftg" // 1 FTG = 1,000,000 uftg (micro-FTG)

	// KwhPerToken defines the energy backing per token
	// 1 FTG = 10 kWh
	KwhPerToken = 10

	// TokensPerMwh defines how many tokens 1 MWh backs
	// 1 MWh = 1000 kWh / 10 kWh per token = 100 tokens
	TokensPerMwh = 100

	// MicroFTGPerToken is the conversion factor
	// 1 FTG = 1,000,000 uftg
	MicroFTGPerToken = 1_000_000
)

// Certificate represents a Renewable Energy Certificate uploaded to the chain
type Certificate struct {
	// CertificateID is the unique on-chain identifier
	CertificateID string `json:"certificate_id"`

	// ExternalID is the external certificate serial (e.g., TerraPass serial number)
	ExternalID string `json:"external_id"`

	// Source is the certificate provider (e.g., "TerraPass", "Green-e", utility name)
	Source string `json:"source"`

	// MwhQuantity is the energy quantity in MWh
	MwhQuantity sdk.Dec `json:"mwh_quantity"`

	// GenerationDate is when the energy was originally generated
	GenerationDate time.Time `json:"generation_date"`

	// PurchaseDate is when the certificate was acquired
	PurchaseDate time.Time `json:"purchase_date"`

	// IPFSHash is the IPFS CID of the original certificate document
	IPFSHash string `json:"ipfs_hash"`

	// ProvenanceType classifies the backing source
	// "proof_of_purchase" = REC certificate
	// "proof_of_generation" = MCS-metered vault energy
	// "proof_of_storage" = Strategic Energy Vault (charged Power Cell)
	ProvenanceType string `json:"provenance_type"`

	// MintedTokens is the number of FTG tokens minted from this certificate
	MintedTokens sdk.Int `json:"minted_tokens"`

	// MintTimestamp is when the tokens were minted
	MintTimestamp time.Time `json:"mint_timestamp"`

	// Owner is the address that uploaded the certificate (treasury)
	Owner sdk.AccAddress `json:"owner"`

	// Status: "active", "partially_redeemed", "fully_redeemed", "burned"
	Status string `json:"status"`
}

// MintRequest is the message to mint new FTG tokens from a certificate
type MintRequest struct {
	// Sender is the treasury address initiating the mint
	Sender sdk.AccAddress `json:"sender"`

	// Certificate contains the REC/vault certificate data
	Certificate Certificate `json:"certificate"`

	// DestinationWallet is where the minted tokens should land
	// If empty, defaults to sender (treasury)
	DestinationWallet sdk.AccAddress `json:"destination_wallet,omitempty"`
}

// MintResponse is returned after successful minting
type MintResponse struct {
	// CertificateID is the on-chain certificate record ID
	CertificateID string `json:"certificate_id"`

	// TokensMinted is the number of FTG tokens created
	TokensMinted sdk.Int `json:"tokens_minted"`

	// Recipient is the address that received the tokens
	Recipient sdk.AccAddress `json:"recipient"`

	// TxHash is the transaction hash
	TxHash string `json:"tx_hash"`
}

// GenesisState defines the energymint module's genesis state
type GenesisState struct {
	// Params defines module parameters
	Params Params `json:"params"`

	// Certificates is the list of all registered certificates
	Certificates []Certificate `json:"certificates"`

	// TotalMintedTokens is the total supply minted through this module
	TotalMintedTokens sdk.Int `json:"total_minted_tokens"`

	// TotalBackedMwh is the total MWh backing all minted tokens
	TotalBackedMwh sdk.Dec `json:"total_backed_mwh"`
}

// Params defines the parameters for the energymint module
type Params struct {
	// AuthorizedMinters is the list of addresses allowed to mint (treasury wallets)
	AuthorizedMinters []sdk.AccAddress `json:"authorized_minters"`

	// MinCertificateMwh is the minimum MWh for a single certificate upload
	MinCertificateMwh sdk.Dec `json:"min_certificate_mwh"`

	// RequireIPFS determines if IPFS hash is mandatory for minting
	RequireIPFS bool `json:"require_ipfs"`

	// AcceptedProvenanceTypes lists which provenance types are currently active
	AcceptedProvenanceTypes []string `json:"accepted_provenance_types"`
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		AuthorizedMinters:      []sdk.AccAddress{},
		MinCertificateMwh:      sdk.NewDecWithPrec(1, 0), // Minimum 1 MWh
		RequireIPFS:            true,
		AcceptedProvenanceTypes: []string{"proof_of_purchase", "proof_of_generation", "proof_of_storage"},
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:            DefaultParams(),
		Certificates:      []Certificate{},
		TotalMintedTokens: sdk.ZeroInt(),
		TotalBackedMwh:    sdk.ZeroDec(),
	}
}
