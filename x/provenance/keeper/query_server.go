package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/provenance/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the provenance QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// ProvenanceRecord returns a single provenance record by ID
func (q queryServer) ProvenanceRecord(goCtx context.Context, req *types.QueryProvenanceRecordRequest) (*types.QueryProvenanceRecordResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	record, found := q.Keeper.GetProvenanceRecord(ctx, req.RecordId)
	if !found {
		return nil, fmt.Errorf("provenance record %s not found", req.RecordId)
	}

	return &types.QueryProvenanceRecordResponse{
		Record: record,
	}, nil
}

// ProvenanceByCertificate returns the provenance record for a given certificate
func (q queryServer) ProvenanceByCertificate(goCtx context.Context, req *types.QueryProvenanceByCertificateRequest) (*types.QueryProvenanceByCertificateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	record, found := q.Keeper.GetProvenanceByCertificate(ctx, req.CertificateId)
	if !found {
		return nil, fmt.Errorf("no provenance record for certificate %s", req.CertificateId)
	}

	return &types.QueryProvenanceByCertificateResponse{
		Record: record,
	}, nil
}

// Params returns the current module parameters
func (q queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := q.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}
