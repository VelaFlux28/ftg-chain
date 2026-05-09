package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/burn/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the burn MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// BurnTokens handles the MsgBurnTokens transaction
// Any token holder can burn their own tokens (manual burn)
func (m msgServer) BurnTokens(goCtx context.Context, msg *types.MsgBurnTokens) (*types.MsgBurnTokensResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	amount, ok := math.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	burnEvent, err := m.Keeper.BurnTokens(ctx, sender, amount, msg.Source, nil)
	if err != nil {
		return nil, err
	}

	return &types.MsgBurnTokensResponse{
		BurnId:       burnEvent.BurnID,
		AmountBurned: burnEvent.Amount.String(),
	}, nil
}

// MerchantBurn handles the Automatic Merchant Protocol burn
// Only authorized merchant addresses can call this
func (m msgServer) MerchantBurn(goCtx context.Context, msg *types.MsgMerchantBurn) (*types.MsgMerchantBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	tokensPurchased, ok := math.NewIntFromString(msg.TokensPurchased)
	if !ok {
		return nil, fmt.Errorf("invalid tokens_purchased: %s", msg.TokensPurchased)
	}

	energyRevenue, err := math.LegacyNewDecFromStr(msg.EnergyRevenueUsd)
	if err != nil {
		return nil, fmt.Errorf("invalid energy_revenue_usd: %w", err)
	}

	dexPrice, err := math.LegacyNewDecFromStr(msg.DexPurchasePrice)
	if err != nil {
		return nil, fmt.Errorf("invalid dex_purchase_price: %w", err)
	}

	burnEvent, err := m.Keeper.MerchantProtocolBurn(
		ctx,
		sender,
		energyRevenue,
		dexPrice,
		tokensPurchased,
		msg.VaultId,
		msg.GridOperator,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgMerchantBurnResponse{
		BurnId:       burnEvent.BurnID,
		AmountBurned: burnEvent.Amount.String(),
	}, nil
}

// AddMerchant handles adding a new authorized merchant address (governance only)
func (m msgServer) AddMerchant(goCtx context.Context, msg *types.MsgAddMerchant) (*types.MsgAddMerchantResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.authority {
		return nil, fmt.Errorf("unauthorized: only governance can add merchants")
	}

	merchantAddr, err := sdk.AccAddressFromBech32(msg.MerchantAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid merchant address: %w", err)
	}

	m.Keeper.AddAuthorizedMerchant(ctx, merchantAddr)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"merchant_added",
			sdk.NewAttribute("merchant_address", msg.MerchantAddress),
			sdk.NewAttribute("added_by", msg.Authority),
		),
	)

	return &types.MsgAddMerchantResponse{}, nil
}
