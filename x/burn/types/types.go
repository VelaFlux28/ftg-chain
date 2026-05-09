package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "ftgburn"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// BurnAddress is the null address where burned tokens are sent
	// This is the cosmos equivalent of Ethereum's 0x0 address
	BurnAddress = "ftg1burn000000000000000000000000000000000"
)

// BurnEvent records a token burn on-chain
type BurnEvent struct {
	// BurnID is the unique identifier for this burn
	BurnID string `json:"burn_id"`

	// Amount is the number of uftg burned
	Amount sdk.Int `json:"amount"`

	// Source describes why the burn happened
	// "merchant_protocol" = Automatic Merchant Protocol (energy sales → buy → burn)
	// "manual" = Manual burn by token holder
	// "redemption" = Token redeemed for physical energy delivery
	Source string `json:"source"`

	// Sender is the address that initiated the burn
	Sender sdk.AccAddress `json:"sender"`

	// Timestamp is when the burn occurred
	Timestamp time.Time `json:"timestamp"`

	// TxHash is the transaction hash
	TxHash string `json:"tx_hash"`

	// MerchantData contains optional data for merchant protocol burns
	MerchantData *MerchantBurnData `json:"merchant_data,omitempty"`
}

// MerchantBurnData contains details specific to Automatic Merchant Protocol burns
type MerchantBurnData struct {
	// EnergyRevenueUSD is the energy sale revenue that triggered this burn
	EnergyRevenueUSD sdk.Dec `json:"energy_revenue_usd"`

	// DEXPurchasePrice is the price per FTG token paid on the DEX
	DEXPurchasePrice sdk.Dec `json:"dex_purchase_price"`

	// VaultID identifies which vault generated the revenue
	VaultID string `json:"vault_id,omitempty"`

	// GridOperator is the utility/grid that paid for the energy
	GridOperator string `json:"grid_operator,omitempty"`
}

// BurnRequest is the message to burn FTG tokens
type BurnRequest struct {
	// Sender is the address burning tokens
	Sender sdk.AccAddress `json:"sender"`

	// Amount is the number of uftg to burn
	Amount sdk.Int `json:"amount"`

	// Source classifies the burn type
	Source string `json:"source"`

	// MerchantData is optional data for merchant protocol burns
	MerchantData *MerchantBurnData `json:"merchant_data,omitempty"`
}

// BurnStats tracks cumulative burn statistics
type BurnStats struct {
	// TotalBurned is the total uftg ever burned
	TotalBurned sdk.Int `json:"total_burned"`

	// TotalBurnEvents is the count of burn transactions
	TotalBurnEvents uint64 `json:"total_burn_events"`

	// MerchantProtocolBurned is uftg burned via the Automatic Merchant Protocol
	MerchantProtocolBurned sdk.Int `json:"merchant_protocol_burned"`

	// ManualBurned is uftg burned manually by holders
	ManualBurned sdk.Int `json:"manual_burned"`

	// RedemptionBurned is uftg burned via energy redemption
	RedemptionBurned sdk.Int `json:"redemption_burned"`

	// TotalEnergyRevenueUSD is cumulative energy revenue that drove merchant burns
	TotalEnergyRevenueUSD sdk.Dec `json:"total_energy_revenue_usd"`
}

// GenesisState defines the burn module's genesis state
type GenesisState struct {
	Params     Params      `json:"params"`
	BurnEvents []BurnEvent `json:"burn_events"`
	Stats      BurnStats   `json:"stats"`
}

// Params defines the parameters for the burn module
type Params struct {
	// MerchantProtocolEnabled determines if automatic burns are active
	MerchantProtocolEnabled bool `json:"merchant_protocol_enabled"`

	// MerchantBurnPercentage is the % of energy revenue used to buy and burn
	// Default: 100% (all energy revenue goes to buy-and-burn)
	MerchantBurnPercentage sdk.Dec `json:"merchant_burn_percentage"`

	// MinBurnAmount is the minimum uftg that can be burned in a single tx
	MinBurnAmount sdk.Int `json:"min_burn_amount"`

	// AuthorizedMerchantAddresses are addresses allowed to trigger merchant burns
	AuthorizedMerchantAddresses []sdk.AccAddress `json:"authorized_merchant_addresses"`
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		MerchantProtocolEnabled:     true,
		MerchantBurnPercentage:      sdk.NewDecWithPrec(100, 0), // 100%
		MinBurnAmount:               sdk.NewInt(1_000_000),      // Minimum 1 FTG
		AuthorizedMerchantAddresses: []sdk.AccAddress{},
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:     DefaultParams(),
		BurnEvents: []BurnEvent{},
		Stats: BurnStats{
			TotalBurned:            sdk.ZeroInt(),
			TotalBurnEvents:        0,
			MerchantProtocolBurned: sdk.ZeroInt(),
			ManualBurned:           sdk.ZeroInt(),
			RedemptionBurned:       sdk.ZeroInt(),
			TotalEnergyRevenueUSD:  sdk.ZeroDec(),
		},
	}
}
