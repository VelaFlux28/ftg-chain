package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgBurnTokens   = "burn_tokens"
	TypeMsgMerchantBurn = "merchant_burn"
	TypeMsgAddMerchant  = "add_merchant"
)

type MsgBurnTokens struct {
	Sender string `json:"sender"`
	Amount string `json:"amount"`
	Source string `json:"source"`
}

type MsgBurnTokensResponse struct {
	BurnId       string `json:"burn_id"`
	AmountBurned string `json:"amount_burned"`
}

type MsgMerchantBurn struct {
	Sender           string `json:"sender"`
	TokensPurchased  string `json:"tokens_purchased"`
	EnergyRevenueUsd string `json:"energy_revenue_usd"`
	DexPurchasePrice string `json:"dex_purchase_price"`
	VaultId          string `json:"vault_id,omitempty"`
	GridOperator     string `json:"grid_operator,omitempty"`
}

type MsgMerchantBurnResponse struct {
	BurnId       string `json:"burn_id"`
	AmountBurned string `json:"amount_burned"`
}

type MsgAddMerchant struct {
	Authority       string `json:"authority"`
	MerchantAddress string `json:"merchant_address"`
}

type MsgAddMerchantResponse struct{}

type MsgServer interface {
	BurnTokens(ctx context.Context, msg *MsgBurnTokens) (*MsgBurnTokensResponse, error)
	MerchantBurn(ctx context.Context, msg *MsgMerchantBurn) (*MsgMerchantBurnResponse, error)
	AddMerchant(ctx context.Context, msg *MsgAddMerchant) (*MsgAddMerchantResponse, error)
}

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

func (msg *MsgBurnTokens) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg *MsgMerchantBurn) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
