package types

import (
	"time"
)

const (
	ModuleName = "exchange"
	StoreKey   = ModuleName
)

// OrderSide represents buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the order type
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus represents the order lifecycle
type OrderStatus string

const (
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusExpired   OrderStatus = "expired"
)

// KYCStatus represents KYC verification state
type KYCStatus string

const (
	KYCStatusNone     KYCStatus = "none"
	KYCStatusPending  KYCStatus = "pending"
	KYCStatusApproved KYCStatus = "approved"
	KYCStatusRejected KYCStatus = "rejected"
)

// KYCTier represents access levels based on verification
type KYCTier int

const (
	KYCTierNone   KYCTier = 0 // No access
	KYCTierBasic  KYCTier = 1 // Email + phone verified, $1K daily limit
	KYCTierFull   KYCTier = 2 // ID verified, $50K daily limit
	KYCTierInst   KYCTier = 3 // Institutional, unlimited + OTC access
)

// TradingPair represents a tradable pair on the exchange
type TradingPair struct {
	ID             string `json:"id"`
	BaseDenom      string `json:"base_denom"`       // e.g., "uftg"
	QuoteDenom     string `json:"quote_denom"`      // e.g., "uusdc"
	BaseSymbol     string `json:"base_symbol"`      // e.g., "FTG"
	QuoteSymbol    string `json:"quote_symbol"`     // e.g., "USDC"
	MinOrderSize   int64  `json:"min_order_size"`   // Minimum order in base denom
	PriceIncrement int64  `json:"price_increment"`  // Minimum price tick (in quote denom)
	SizeIncrement  int64  `json:"size_increment"`   // Minimum size tick (in base denom)
	MakerFee       int64  `json:"maker_fee_bps"`    // Fee in basis points (e.g., 10 = 0.10%)
	TakerFee       int64  `json:"taker_fee_bps"`    // Fee in basis points (e.g., 25 = 0.25%)
	Active         bool   `json:"active"`
}

// ExchangeWallet represents a user's exchange wallet
type ExchangeWallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Address   string    `json:"address"`    // On-chain address
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	KYCTier   KYCTier   `json:"kyc_tier"`
	Balances  map[string]int64 `json:"balances"` // denom → amount
	Locked    map[string]int64 `json:"locked"`   // denom → locked in open orders
	Active    bool      `json:"active"`
}

// Order represents a limit/market order on the book
type Order struct {
	ID            string      `json:"id"`
	WalletID      string      `json:"wallet_id"`
	UserID        string      `json:"user_id"`
	PairID        string      `json:"pair_id"`
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	Price         int64       `json:"price"`          // In quote denom per 1 base unit (for limit)
	Quantity      int64       `json:"quantity"`       // Total quantity in base denom
	FilledQty     int64       `json:"filled_qty"`     // Already filled
	RemainingQty  int64       `json:"remaining_qty"`  // Remaining to fill
	Status        OrderStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	ExpiresAt     *time.Time  `json:"expires_at,omitempty"`
}

// Trade represents a matched trade between two orders
type Trade struct {
	ID          string    `json:"id"`
	PairID      string    `json:"pair_id"`
	MakerOrder  string    `json:"maker_order_id"`
	TakerOrder  string    `json:"taker_order_id"`
	MakerWallet string    `json:"maker_wallet_id"`
	TakerWallet string    `json:"taker_wallet_id"`
	Side        OrderSide `json:"side"`         // Taker's side
	Price       int64     `json:"price"`        // Execution price
	Quantity    int64     `json:"quantity"`     // Quantity traded
	MakerFee    int64     `json:"maker_fee"`   // Fee charged to maker
	TakerFee    int64     `json:"taker_fee"`   // Fee charged to taker
	ExecutedAt  time.Time `json:"executed_at"`
	BlockHeight int64     `json:"block_height"`
}

// KYCApplication represents a user's KYC submission
type KYCApplication struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	WalletID       string    `json:"wallet_id"`
	RequestedTier  KYCTier   `json:"requested_tier"`
	CurrentTier    KYCTier   `json:"current_tier"`
	Status         KYCStatus `json:"status"`
	// Basic tier fields
	Email          string    `json:"email,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	Country        string    `json:"country,omitempty"`
	// Full tier fields
	FullName       string    `json:"full_name,omitempty"`
	DateOfBirth    string    `json:"date_of_birth,omitempty"`
	IDType         string    `json:"id_type,omitempty"`         // passport, drivers_license, national_id
	IDNumber       string    `json:"id_number,omitempty"`
	IDDocumentHash string    `json:"id_document_hash,omitempty"` // IPFS hash of ID document
	ProofOfAddress string    `json:"proof_of_address,omitempty"` // IPFS hash
	// Institutional tier fields
	CompanyName    string    `json:"company_name,omitempty"`
	CompanyRegNo   string    `json:"company_reg_no,omitempty"`
	CompanyCountry string    `json:"company_country,omitempty"`
	// AML fields
	AMLRiskScore   int       `json:"aml_risk_score"`    // 0-100
	PEPCheck       bool      `json:"pep_check"`         // Politically Exposed Person
	SanctionsCheck bool      `json:"sanctions_check"`   // OFAC/EU sanctions
	// Timestamps
	SubmittedAt    time.Time `json:"submitted_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	RejectedReason string    `json:"rejected_reason,omitempty"`
}

// DailyLimit tracks a user's daily trading volume for compliance
type DailyLimit struct {
	UserID      string `json:"user_id"`
	Date        string `json:"date"` // YYYY-MM-DD
	BuyVolume   int64  `json:"buy_volume_usdc"`
	SellVolume  int64  `json:"sell_volume_usdc"`
	TotalVolume int64  `json:"total_volume_usdc"`
}

// OrderBookLevel represents a price level in the order book
type OrderBookLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
	Orders   int   `json:"order_count"`
}

// OrderBookSnapshot represents the current state of the order book
type OrderBookSnapshot struct {
	PairID    string           `json:"pair_id"`
	Bids      []OrderBookLevel `json:"bids"`  // Sorted descending by price
	Asks      []OrderBookLevel `json:"asks"`  // Sorted ascending by price
	LastPrice int64            `json:"last_price"`
	Timestamp time.Time        `json:"timestamp"`
}

// ExchangeConfig holds exchange-wide configuration
type ExchangeConfig struct {
	// Fee collection address
	FeeCollector string `json:"fee_collector"`
	// Whether the exchange is accepting new orders
	TradingEnabled bool `json:"trading_enabled"`
	// Whether new registrations are open
	RegistrationOpen bool `json:"registration_open"`
	// AML threshold (USDC) that triggers enhanced due diligence
	AMLThreshold int64 `json:"aml_threshold_usdc"`
	// Maximum daily volume per tier (in USDC micro-units)
	TierLimits map[KYCTier]int64 `json:"tier_limits"`
}
