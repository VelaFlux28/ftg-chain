package treasury

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "treasury"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// WalletType defines the purpose of a treasury wallet
type WalletType string

const (
	// WalletTypeMinting — receives all newly minted tokens from energymint module
	WalletTypeMinting WalletType = "minting"

	// WalletTypeCommitted — holds tokens earmarked for specific buyers (OTC deals, pre-sales)
	// These tokens are OFF the public market
	WalletTypeCommitted WalletType = "committed"

	// WalletTypeLiquidity — tokens listed on the DEX for public purchase
	// Only tokens in this wallet are available for open market trading
	WalletTypeLiquidity WalletType = "liquidity"
)

// TreasuryWallet represents a segregated wallet within the treasury
type TreasuryWallet struct {
	// Address is the on-chain address of this wallet
	Address sdk.AccAddress `json:"address"`

	// Type classifies the wallet's purpose
	Type WalletType `json:"type"`

	// Label is a human-readable name
	Label string `json:"label"`

	// Owner is the address authorized to operate this wallet
	Owner sdk.AccAddress `json:"owner"`
}

// Commitment represents tokens earmarked for a specific buyer
type Commitment struct {
	// CommitmentID is the unique identifier
	CommitmentID string `json:"commitment_id"`

	// BuyerReference is an internal reference for the buyer (not on-chain identity)
	BuyerReference string `json:"buyer_reference"`

	// Amount is the number of uftg committed
	Amount sdk.Int `json:"amount"`

	// PricePerToken is the agreed price in USD (for record-keeping)
	PricePerToken sdk.Dec `json:"price_per_token"`

	// TotalValueUSD is the total deal value
	TotalValueUSD sdk.Dec `json:"total_value_usd"`

	// Status: "pending", "funded", "delivered", "cancelled"
	Status string `json:"status"`

	// CreatedAt is when the commitment was created
	CreatedAt int64 `json:"created_at"`

	// DeliveredAt is when tokens were sent to the buyer (0 if not yet)
	DeliveredAt int64 `json:"delivered_at"`

	// DestinationWallet is where tokens will be sent upon delivery
	DestinationWallet sdk.AccAddress `json:"destination_wallet,omitempty"`
}

// TreasuryState holds the overall treasury status
type TreasuryState struct {
	// Wallets is the list of all treasury wallets
	Wallets []TreasuryWallet `json:"wallets"`

	// Commitments is the list of all active commitments
	Commitments []Commitment `json:"commitments"`

	// TotalCommitted is the total uftg currently committed to buyers
	TotalCommitted sdk.Int `json:"total_committed"`

	// TotalDelivered is the total uftg delivered to buyers historically
	TotalDelivered sdk.Int `json:"total_delivered"`
}

// TreasuryConfig holds the configuration for treasury operations
type TreasuryConfig struct {
	// MintingWalletAddress is where newly minted tokens land
	MintingWalletAddress sdk.AccAddress `json:"minting_wallet_address"`

	// CommittedWalletAddress holds tokens reserved for OTC buyers
	CommittedWalletAddress sdk.AccAddress `json:"committed_wallet_address"`

	// LiquidityWalletAddress holds tokens available for DEX trading
	LiquidityWalletAddress sdk.AccAddress `json:"liquidity_wallet_address"`

	// OwnerAddress is the master authority (Nick's key)
	OwnerAddress sdk.AccAddress `json:"owner_address"`
}

// Operations defines the treasury operations available

// TransferToCommitted moves tokens from Minting → Committed wallet
// Used when an OTC deal is confirmed and tokens need to be reserved
type TransferToCommitted struct {
	Amount         sdk.Int `json:"amount"`
	BuyerReference string  `json:"buyer_reference"`
	PricePerToken  sdk.Dec `json:"price_per_token"`
}

// TransferToLiquidity moves tokens from Minting → Liquidity wallet
// Used when tokens should be made available for public DEX trading
type TransferToLiquidity struct {
	Amount sdk.Int `json:"amount"`
}

// DeliverCommitment moves tokens from Committed → Buyer's wallet
// Used when an OTC buyer has paid and is ready to receive tokens
type DeliverCommitment struct {
	CommitmentID      string         `json:"commitment_id"`
	DestinationWallet sdk.AccAddress `json:"destination_wallet"`
}

// CancelCommitment returns tokens from Committed → Minting wallet
// Used when an OTC deal falls through
type CancelCommitment struct {
	CommitmentID string `json:"commitment_id"`
	Reason       string `json:"reason"`
}
