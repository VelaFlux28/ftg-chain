package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/burn/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the burn QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// BurnStats returns cumulative burn statistics
func (q queryServer) BurnStats(goCtx context.Context, req *types.QueryBurnStatsRequest) (*types.QueryBurnStatsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	stats := q.Keeper.GetBurnStats(ctx)

	return &types.QueryBurnStatsResponse{
		TotalBurned:            stats.TotalBurned.String(),
		TotalBurnEvents:        stats.TotalBurnEvents,
		MerchantProtocolBurned: stats.MerchantProtocolBurned.String(),
		ManualBurned:           stats.ManualBurned.String(),
		RedemptionBurned:       stats.RedemptionBurned.String(),
		TotalEnergyRevenueUsd:  stats.TotalEnergyRevenueUSD.String(),
	}, nil
}

// Params returns the current module parameters
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
