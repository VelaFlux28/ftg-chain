package types

import (
	"time"

	"cosmossdk.io/math"
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
	CertificateID  string         `json:"certificate_id"`
	ExternalID     string         `json:"external_id"`
	Source         string         `json:"source"`
	MwhQuantity    math.LegacyDec `json:"mwh_quantity"`
	GenerationDate time.Time      `json:"generation_date"`
	PurchaseDate   time.Time      `json:"purchase_date"`
	IPFSHash       string         `json:"ipfs_hash"`
	ProvenanceType string         `json:"provenance_type"`
	MintedTokens   math.Int       `json:"minted_tokens"`
	MintTimestamp  time.Time      `json:"mint_timestamp"`
	Owner          sdk.AccAddress `json:"owner"`
	Status         string         `json:"status"`
}

// MintRequest is the message to mint new FTG tokens from a certificate
type MintRequest struct {
	Sender            sdk.AccAddress `json:"sender"`
	Certificate       Certificate    `json:"certificate"`
	DestinationWallet sdk.AccAddress `json:"destination_wallet,omitempty"`
}

// MintResponse is returned after successful minting
type MintResponse struct {
	CertificateID string         `json:"certificate_id"`
	TokensMinted  math.Int       `json:"tokens_minted"`
	Recipient     sdk.AccAddress `json:"recipient"`
	TxHash        string         `json:"tx_hash"`
}

// GenesisState defines the energymint module's genesis state
type GenesisState struct {
	Params            Params        `json:"params"`
	Certificates      []Certificate `json:"certificates"`
	TotalMintedTokens math.Int      `json:"total_minted_tokens"`
	TotalBackedMwh    math.LegacyDec `json:"total_backed_mwh"`
}

// Params defines the parameters for the energymint module
type Params struct {
	AuthorizedMinters      []sdk.AccAddress `json:"authorized_minters"`
	MinCertificateMwh      math.LegacyDec   `json:"min_certificate_mwh"`
	RequireIPFS            bool             `json:"require_ipfs"`
	AcceptedProvenanceTypes []string        `json:"accepted_provenance_types"`
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		AuthorizedMinters:      []sdk.AccAddress{},
		MinCertificateMwh:      math.LegacyNewDecWithPrec(1, 0), // Minimum 1 MWh
		RequireIPFS:            true,
		AcceptedProvenanceTypes: []string{"proof_of_purchase", "proof_of_generation", "proof_of_storage"},
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:            DefaultParams(),
		Certificates:      []Certificate{},
		TotalMintedTokens: math.ZeroInt(),
		TotalBackedMwh:    math.LegacyZeroDec(),
	}
}
