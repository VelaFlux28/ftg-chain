package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/energymint/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the energymint QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Certificate returns a single certificate by external ID
func (q queryServer) Certificate(goCtx context.Context, req *types.QueryCertificateRequest) (*types.QueryCertificateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cert, found := q.Keeper.GetCertificate(ctx, req.ExternalId)
	if !found {
		return nil, fmt.Errorf("certificate %s not found", req.ExternalId)
	}

	return &types.QueryCertificateResponse{
		Certificate: cert,
	}, nil
}

// TotalSupply returns the total minted supply and backing
func (q queryServer) TotalSupply(goCtx context.Context, req *types.QueryTotalSupplyRequest) (*types.QueryTotalSupplyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	totalMinted := q.Keeper.GetTotalMinted(ctx)
	totalBackedMwh := q.Keeper.GetTotalBackedMwh(ctx)

	return &types.QueryTotalSupplyResponse{
		TotalMintedUftg: totalMinted.String(),
		TotalBackedMwh:  totalBackedMwh.String(),
		TotalFtgTokens:  totalMinted.Quo(math.NewInt(types.MicroFTGPerToken)).String(),
	}, nil
}

// Params returns the current module parameters
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// IsAuthorizedMinter checks if an address is authorized to mint
func (q queryServer) IsAuthorizedMinter(goCtx context.Context, req *types.QueryIsAuthorizedMinterRequest) (*types.QueryIsAuthorizedMinterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	authorized := q.Keeper.IsAuthorizedMinter(ctx, addr)
	return &types.QueryIsAuthorizedMinterResponse{
		Authorized: authorized,
	}, nil
}
