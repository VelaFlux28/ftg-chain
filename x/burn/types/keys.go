package types

const (
	// Module store key prefixes
	BurnEventKeyPrefix          = "BurnEvent/value/"
	BurnEventCountKey           = "BurnEvent/count/"
	BurnStatsKey                = "BurnStats/"
	AuthorizedMerchantKeyPrefix = "AuthorizedMerchant/"
)

// BurnEventKey returns the store key for a burn event by ID
func BurnEventKey(burnID string) []byte {
	return []byte(BurnEventKeyPrefix + burnID)
}

// AuthorizedMerchantKey returns the store key for an authorized merchant address
func AuthorizedMerchantKey(addr string) []byte {
	return []byte(AuthorizedMerchantKeyPrefix + addr)
}
