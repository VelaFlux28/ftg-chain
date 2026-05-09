package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "ftgburn"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
	BurnAddress = "ftg1burn000000000000000000000000000000000"
)

// BurnEvent records a token burn on-chain
type BurnEvent struct {
	BurnID       string         `json:"burn_id"`
	Amount       math.Int       `json:"amount"`
	Source       string         `json:"source"`
	Sender       sdk.AccAddress `json:"sender"`
	Timestamp    time.Time      `json:"timestamp"`
	TxHash       string         `json:"tx_hash"`
	MerchantData *MerchantBurnData `json:"merchant_data,omitempty"`
}

// MerchantBurnData contains details specific to Automatic Merchant Protocol burns
type MerchantBurnData struct {
	EnergyRevenueUSD math.LegacyDec `json:"energy_revenue_usd"`
	DEXPurchasePrice math.LegacyDec `json:"dex_purchase_price"`
	VaultID          string         `json:"vault_id,omitempty"`
	GridOperator     string         `json:"grid_operator,omitempty"`
}

// BurnRequest is the message to burn FTG tokens
type BurnRequest struct {
	Sender       sdk.AccAddress    `json:"sender"`
	Amount       math.Int          `json:"amount"`
	Source       string            `json:"source"`
	MerchantData *MerchantBurnData `json:"merchant_data,omitempty"`
}

// BurnStats tracks cumulative burn statistics
type BurnStats struct {
	TotalBurned            math.Int       `json:"total_burned"`
	TotalBurnEvents        uint64         `json:"total_burn_events"`
	MerchantProtocolBurned math.Int       `json:"merchant_protocol_burned"`
	ManualBurned           math.Int       `json:"manual_burned"`
	RedemptionBurned       math.Int       `json:"redemption_burned"`
	TotalEnergyRevenueUSD  math.LegacyDec `json:"total_energy_revenue_usd"`
}

// GenesisState defines the burn module's genesis state
type GenesisState struct {
	Params     Params      `json:"params"`
	BurnEvents []BurnEvent `json:"burn_events"`
	Stats      BurnStats   `json:"stats"`
}

// Params defines the parameters for the burn module
type Params struct {
	MerchantProtocolEnabled     bool             `json:"merchant_protocol_enabled"`
	MerchantBurnPercentage      math.LegacyDec   `json:"merchant_burn_percentage"`
	MinBurnAmount               math.Int         `json:"min_burn_amount"`
	AuthorizedMerchantAddresses []sdk.AccAddress `json:"authorized_merchant_addresses"`
}

// DefaultParams returns default module parameters
func DefaultParams() Params {
	return Params{
		MerchantProtocolEnabled:     true,
		MerchantBurnPercentage:      math.LegacyNewDecWithPrec(100, 0), // 100%
		MinBurnAmount:               math.NewInt(1_000_000),            // Minimum 1 FTG
		AuthorizedMerchantAddresses: []sdk.AccAddress{},
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:     DefaultParams(),
		BurnEvents: []BurnEvent{},
		Stats: BurnStats{
			TotalBurned:            math.ZeroInt(),
			TotalBurnEvents:        0,
			MerchantProtocolBurned: math.ZeroInt(),
			ManualBurned:           math.ZeroInt(),
			RedemptionBurned:       math.ZeroInt(),
			TotalEnergyRevenueUSD:  math.LegacyZeroDec(),
		},
	}
}
