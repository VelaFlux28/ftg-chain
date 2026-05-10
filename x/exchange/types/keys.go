package types

// Key prefixes for the exchange store
var (
	WalletKeyPrefix     = []byte("Wallet/")
	OrderKeyPrefix      = []byte("Order/")
	TradeKeyPrefix      = []byte("Trade/")
	KYCKeyPrefix        = []byte("KYC/")
	PairKeyPrefix       = []byte("Pair/")
	DailyLimitKeyPrefix = []byte("DailyLimit/")
	ConfigKey           = []byte("Config")
	SequenceKey         = []byte("Sequence/")
)

func WalletKey(walletID string) []byte {
	return append(WalletKeyPrefix, []byte(walletID)...)
}

func OrderKey(orderID string) []byte {
	return append(OrderKeyPrefix, []byte(orderID)...)
}

func TradeKey(tradeID string) []byte {
	return append(TradeKeyPrefix, []byte(tradeID)...)
}

func KYCKey(applicationID string) []byte {
	return append(KYCKeyPrefix, []byte(applicationID)...)
}

func PairKey(pairID string) []byte {
	return append(PairKeyPrefix, []byte(pairID)...)
}

func DailyLimitKey(userID, date string) []byte {
	return append(DailyLimitKeyPrefix, []byte(userID+"/"+date)...)
}
