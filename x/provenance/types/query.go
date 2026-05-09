package types

// Query request/response types for the provenance module

// QueryProvenanceRecordRequest is the request to query a provenance record
type QueryProvenanceRecordRequest struct {
	RecordId string `json:"record_id"`
}

// QueryProvenanceRecordResponse contains a provenance record
type QueryProvenanceRecordResponse struct {
	Record ProvenanceRecord `json:"record"`
}

// QueryProvenanceByCertificateRequest queries provenance by certificate ID
type QueryProvenanceByCertificateRequest struct {
	CertificateId string `json:"certificate_id"`
}

// QueryProvenanceByCertificateResponse contains the provenance for a certificate
type QueryProvenanceByCertificateResponse struct {
	Record ProvenanceRecord `json:"record"`
}

// QueryParamsRequest is the request to query module parameters
type QueryParamsRequest struct{}

// QueryParamsResponse contains the module parameters
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// QueryServer defines the provenance module's query service
type QueryServer interface {
	ProvenanceRecord(ctx interface{}, req *QueryProvenanceRecordRequest) (*QueryProvenanceRecordResponse, error)
	ProvenanceByCertificate(ctx interface{}, req *QueryProvenanceByCertificateRequest) (*QueryProvenanceByCertificateResponse, error)
	Params(ctx interface{}, req *QueryParamsRequest) (*QueryParamsResponse, error)
}
