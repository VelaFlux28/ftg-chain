package types

import (
	"encoding/binary"
)

const (
	// Module store key prefixes
	ProvenanceRecordKeyPrefix = "ProvenanceRecord/value/"
	ProvenanceCountKey        = "ProvenanceRecord/count/"
	TokenLineageKeyPrefix     = "TokenLineage/value/"
	CertToProvenanceKeyPrefix = "CertToProvenance/"
)

// ProvenanceRecordKey returns the store key for a provenance record by ID
func ProvenanceRecordKey(recordID string) []byte {
	return []byte(ProvenanceRecordKeyPrefix + recordID)
}

// TokenLineageKey returns the store key for a token's lineage
func TokenLineageKey(tokenID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, tokenID)
	return append([]byte(TokenLineageKeyPrefix), bz...)
}

// CertToProvenanceKey maps a certificate ID to its provenance record
func CertToProvenanceKey(certID string) []byte {
	return []byte(CertToProvenanceKeyPrefix + certID)
}
