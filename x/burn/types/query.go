package types

// Query request/response types for the burn module

// QueryBurnStatsRequest is the request to query burn statistics
type QueryBurnStatsRequest struct{}

// QueryBurnStatsResponse contains cumulative burn statistics
type QueryBurnStatsResponse struct {
	TotalBurned            string `json:"total_burned"`
	TotalBurnEvents        uint64 `json:"total_burn_events"`
	MerchantProtocolBurned string `json:"merchant_protocol_burned"`
	ManualBurned           string `json:"manual_burned"`
	RedemptionBurned       string `json:"redemption_burned"`
	TotalEnergyRevenueUsd  string `json:"total_energy_revenue_usd"`
}

// QueryParamsRequest is the request to query module parameters
type QueryParamsRequest struct{}

// QueryParamsResponse contains the module parameters
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// QueryServer defines the burn module's query service
type QueryServer interface {
	BurnStats(ctx interface{}, req *QueryBurnStatsRequest) (*QueryBurnStatsResponse, error)
	Params(ctx interface{}, req *QueryParamsRequest) (*QueryParamsResponse, error)
}
