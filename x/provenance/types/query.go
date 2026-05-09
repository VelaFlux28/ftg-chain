package types

import "context"

type QueryProvenanceRecordRequest struct {
	RecordId string `json:"record_id"`
}

type QueryProvenanceRecordResponse struct {
	Record ProvenanceRecord `json:"record"`
}

type QueryProvenanceByCertificateRequest struct {
	CertificateId string `json:"certificate_id"`
}

type QueryProvenanceByCertificateResponse struct {
	Record ProvenanceRecord `json:"record"`
}

type QueryParamsRequest struct{}

type QueryParamsResponse struct {
	Params Params `json:"params"`
}

type QueryServer interface {
	ProvenanceRecord(ctx context.Context, req *QueryProvenanceRecordRequest) (*QueryProvenanceRecordResponse, error)
	ProvenanceByCertificate(ctx context.Context, req *QueryProvenanceByCertificateRequest) (*QueryProvenanceByCertificateResponse, error)
	Params(ctx context.Context, req *QueryParamsRequest) (*QueryParamsResponse, error)
}
