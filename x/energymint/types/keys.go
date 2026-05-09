package types

const (
	// Module store key prefixes
	CertificateKeyPrefix      = "Certificate/value/"
	CertificateCountKey       = "Certificate/count/"
	TotalMintedKey            = "TotalMinted/"
	TotalBackedMwhKey         = "TotalBackedMwh/"
	AuthorizedMinterKeyPrefix = "AuthorizedMinter/"
	ParamsKey                 = "Params/"
)

// CertificateKey returns the store key for a certificate by ID
func CertificateKey(certID string) []byte {
	return []byte(CertificateKeyPrefix + certID)
}

// AuthorizedMinterKey returns the store key for an authorized minter
func AuthorizedMinterKey(addr string) []byte {
	return []byte(AuthorizedMinterKeyPrefix + addr)
}
