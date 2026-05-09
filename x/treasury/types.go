package treasury

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "treasury"
	StoreKey   = ModuleName
)

type WalletType string

const (
	WalletTypeMinting   WalletType = "minting"
	WalletTypeCommitted WalletType = "committed"
	WalletTypeLiquidity WalletType = "liquidity"
)

type TreasuryWallet struct {
	Address sdk.AccAddress `json:"address"`
	Type    WalletType     `json:"type"`
	Label   string         `json:"label"`
	Owner   sdk.AccAddress `json:"owner"`
}

type Commitment struct {
	CommitmentID      string         `json:"commitment_id"`
	BuyerReference    string         `json:"buyer_reference"`
	Amount            math.Int       `json:"amount"`
	PricePerToken     math.LegacyDec `json:"price_per_token"`
	TotalValueUSD     math.LegacyDec `json:"total_value_usd"`
	Status            string         `json:"status"`
	CreatedAt         int64          `json:"created_at"`
	DeliveredAt       int64          `json:"delivered_at"`
	DestinationWallet sdk.AccAddress `json:"destination_wallet,omitempty"`
}

type TreasuryState struct {
	Wallets        []TreasuryWallet `json:"wallets"`
	Commitments    []Commitment     `json:"commitments"`
	TotalCommitted math.Int         `json:"total_committed"`
	TotalDelivered math.Int         `json:"total_delivered"`
}

type TreasuryConfig struct {
	MintingWalletAddress   sdk.AccAddress `json:"minting_wallet_address"`
	CommittedWalletAddress sdk.AccAddress `json:"committed_wallet_address"`
	LiquidityWalletAddress sdk.AccAddress `json:"liquidity_wallet_address"`
	OwnerAddress           sdk.AccAddress `json:"owner_address"`
}

type TransferToCommitted struct {
	Amount         math.Int       `json:"amount"`
	BuyerReference string         `json:"buyer_reference"`
	PricePerToken  math.LegacyDec `json:"price_per_token"`
}

type TransferToLiquidity struct {
	Amount math.Int `json:"amount"`
}

type DeliverCommitment struct {
	CommitmentID      string         `json:"commitment_id"`
	DestinationWallet sdk.AccAddress `json:"destination_wallet"`
}

type CancelCommitment struct {
	CommitmentID string `json:"commitment_id"`
	Reason       string `json:"reason"`
}
