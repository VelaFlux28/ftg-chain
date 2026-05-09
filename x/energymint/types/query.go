package types

// Query request/response types for the energymint module

// QueryCertificateRequest is the request to query a certificate
type QueryCertificateRequest struct {
	ExternalId string `json:"external_id"`
}

// QueryCertificateResponse is the response containing a certificate
type QueryCertificateResponse struct {
	Certificate Certificate `json:"certificate"`
}

// QueryTotalSupplyRequest is the request to query total supply stats
type QueryTotalSupplyRequest struct{}

// QueryTotalSupplyResponse contains the total supply information
type QueryTotalSupplyResponse struct {
	TotalMintedUftg string `json:"total_minted_uftg"`
	TotalBackedMwh  string `json:"total_backed_mwh"`
	TotalFtgTokens  string `json:"total_ftg_tokens"`
}

// QueryParamsRequest is the request to query module parameters
type QueryParamsRequest struct{}

// QueryParamsResponse contains the module parameters
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// QueryIsAuthorizedMinterRequest checks if an address can mint
type QueryIsAuthorizedMinterRequest struct {
	Address string `json:"address"`
}

// QueryIsAuthorizedMinterResponse contains the authorization status
type QueryIsAuthorizedMinterResponse struct {
	Authorized bool `json:"authorized"`
}

// QueryServer defines the energymint module's query service
type QueryServer interface {
	Certificate(ctx interface{}, req *QueryCertificateRequest) (*QueryCertificateResponse, error)
	TotalSupply(ctx interface{}, req *QueryTotalSupplyRequest) (*QueryTotalSupplyResponse, error)
	Params(ctx interface{}, req *QueryParamsRequest) (*QueryParamsResponse, error)
	IsAuthorizedMinter(ctx interface{}, req *QueryIsAuthorizedMinterRequest) (*QueryIsAuthorizedMinterResponse, error)
}
