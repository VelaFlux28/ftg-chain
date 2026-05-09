package types

import "context"

type QueryBurnStatsRequest struct{}

type QueryBurnStatsResponse struct {
	TotalBurned            string `json:"total_burned"`
	TotalBurnEvents        uint64 `json:"total_burn_events"`
	MerchantProtocolBurned string `json:"merchant_protocol_burned"`
	ManualBurned           string `json:"manual_burned"`
	RedemptionBurned       string `json:"redemption_burned"`
	TotalEnergyRevenueUsd  string `json:"total_energy_revenue_usd"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

type QueryServer interface {
	BurnStats(ctx context.Context, req *QueryBurnStatsRequest) (*QueryBurnStatsResponse, error)
	Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error)
}
