package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message types for the burn module
const (
	TypeMsgBurnTokens  = "burn_tokens"
	TypeMsgMerchantBurn = "merchant_burn"
	TypeMsgAddMerchant  = "add_merchant"
)

// MsgBurnTokens is the transaction message to burn FTG tokens
type MsgBurnTokens struct {
	Sender string `json:"sender"`
	Amount string `json:"amount"` // in uftg
	Source string `json:"source"` // "manual" or "redemption"
}

// MsgBurnTokensResponse is the response from a successful burn
type MsgBurnTokensResponse struct {
	BurnId       string `json:"burn_id"`
	AmountBurned string `json:"amount_burned"`
}

// MsgMerchantBurn is the Automatic Merchant Protocol burn message
// This is triggered by the Liquidity Smart Contract after purchasing FTG on the DEX
type MsgMerchantBurn struct {
	Sender           string `json:"sender"`            // Liquidity contract address
	TokensPurchased  string `json:"tokens_purchased"`  // uftg purchased on DEX
	EnergyRevenueUsd string `json:"energy_revenue_usd"` // USD revenue that triggered this
	DexPurchasePrice string `json:"dex_purchase_price"` // Price per FTG on DEX
	VaultId          string `json:"vault_id,omitempty"`
	GridOperator     string `json:"grid_operator,omitempty"`
}

// MsgMerchantBurnResponse is the response from a merchant burn
type MsgMerchantBurnResponse struct {
	BurnId       string `json:"burn_id"`
	AmountBurned string `json:"amount_burned"`
}

// MsgAddMerchant is the governance message to add an authorized merchant
type MsgAddMerchant struct {
	Authority       string `json:"authority"`
	MerchantAddress string `json:"merchant_address"`
}

// MsgAddMerchantResponse is the response from adding a merchant
type MsgAddMerchantResponse struct{}

// MsgServer defines the burn module's gRPC message service
type MsgServer interface {
	BurnTokens(ctx interface{}, msg *MsgBurnTokens) (*MsgBurnTokensResponse, error)
	MerchantBurn(ctx interface{}, msg *MsgMerchantBurn) (*MsgMerchantBurnResponse, error)
	AddMerchant(ctx interface{}, msg *MsgAddMerchant) (*MsgAddMerchantResponse, error)
}

// ValidateBasic performs basic validation on MsgBurnTokens
func (msg *MsgBurnTokens) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return err
	}

	if msg.Amount == "" {
		return ErrInvalidBurn("amount cannot be empty")
	}

	if msg.Source != "manual" && msg.Source != "redemption" {
		return ErrInvalidBurn("source must be 'manual' or 'redemption'")
	}

	return nil
}

// ValidateBasic performs basic validation on MsgMerchantBurn
func (msg *MsgMerchantBurn) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return err
	}

	if msg.TokensPurchased == "" {
		return ErrInvalidBurn("tokens_purchased cannot be empty")
	}

	if msg.EnergyRevenueUsd == "" {
		return ErrInvalidBurn("energy_revenue_usd cannot be empty")
	}

	return nil
}

// GetSigners returns the expected signers for MsgBurnTokens
func (msg *MsgBurnTokens) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

// GetSigners returns the expected signers for MsgMerchantBurn
func (msg *MsgMerchantBurn) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
