package types

import "context"

// Query request/response types for the energymint module

type QueryCertificateRequest struct {
	ExternalId string `json:"external_id"`
}

type QueryCertificateResponse struct {
	Certificate Certificate `json:"certificate"`
}

type QueryTotalSupplyRequest struct{}

type QueryTotalSupplyResponse struct {
	TotalMintedUftg string `json:"total_minted_uftg"`
	TotalBackedMwh  string `json:"total_backed_mwh"`
	TotalFtgTokens  string `json:"total_ftg_tokens"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

type QueryIsAuthorizedMinterRequest struct {
	Address string `json:"address"`
}

type QueryIsAuthorizedMinterResponse struct {
	Authorized bool `json:"authorized"`
}

// QueryServer defines the energymint module's query service
type QueryServer interface {
	Certificate(ctx context.Context, req *QueryCertificateRequest) (*QueryCertificateResponse, error)
	TotalSupply(ctx context.Context, req *QueryTotalSupplyRequest) (*QueryTotalSupplyResponse, error)
	Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error)
	IsAuthorizedMinter(ctx context.Context, req *QueryIsAuthorizedMinterRequest) (*QueryIsAuthorizedMinterResponse, error)
}
