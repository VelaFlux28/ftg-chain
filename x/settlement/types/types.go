package types

import "time"

const (
	ModuleName = "settlement"
	StoreKey   = ModuleName
)

// PaymentMethod defines how USDC enters/exits the system
type PaymentMethod string

const (
	PaymentMethodBankWire    PaymentMethod = "bank_wire"     // ACH/SWIFT → Circle → USDC
	PaymentMethodCircleMint  PaymentMethod = "circle_mint"   // Direct Circle API mint
	PaymentMethodNobleIBC    PaymentMethod = "noble_ibc"     // IBC transfer from Noble chain
	PaymentMethodStablecoin  PaymentMethod = "stablecoin"    // Direct USDC deposit (ERC-20/SPL)
	PaymentMethodStripe      PaymentMethod = "stripe"        // Stripe fiat on-ramp
)

// SettlementStatus tracks the lifecycle of a settlement
type SettlementStatus string

const (
	StatusPending    SettlementStatus = "pending"     // Created, awaiting funding
	StatusFunded     SettlementStatus = "funded"      // USDC received in escrow
	StatusExecuting  SettlementStatus = "executing"   // Trade being settled
	StatusCompleted  SettlementStatus = "completed"   // FTG delivered, USDC released
	StatusFailed     SettlementStatus = "failed"      // Settlement failed
	StatusRefunded   SettlementStatus = "refunded"    // USDC returned to buyer
	StatusDisputed   SettlementStatus = "disputed"    // Under review
)

// EscrowAccount holds USDC in trust during trade settlement
type EscrowAccount struct {
	ID              string           `json:"id"`
	TradeID         string           `json:"trade_id"`
	BuyerWalletID   string           `json:"buyer_wallet_id"`
	SellerWalletID  string           `json:"seller_wallet_id"`
	AmountUSDC      int64            `json:"amount_usdc"`       // in uusdc (1 USDC = 1,000,000 uusdc)
	FeeUSDC         int64            `json:"fee_usdc"`
	Status          SettlementStatus `json:"status"`
	PaymentMethod   PaymentMethod    `json:"payment_method"`
	PaymentRef      string           `json:"payment_ref"`       // External payment reference
	CreatedAt       time.Time        `json:"created_at"`
	FundedAt        *time.Time       `json:"funded_at,omitempty"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`
	ExpiresAt       time.Time        `json:"expires_at"`        // Auto-refund if not funded
}

// SettlementRecord is the permanent record of a completed settlement
type SettlementRecord struct {
	ID              string           `json:"id"`
	EscrowID        string           `json:"escrow_id"`
	TradeID         string           `json:"trade_id"`
	BuyerWalletID   string           `json:"buyer_wallet_id"`
	SellerWalletID  string           `json:"seller_wallet_id"`
	FTGAmount       int64            `json:"ftg_amount"`        // uftg delivered
	USDCAmount      int64            `json:"usdc_amount"`       // uusdc paid
	PricePerFTG     int64            `json:"price_per_ftg"`     // uusdc per 1M uftg
	FeeCollected    int64            `json:"fee_collected"`     // uusdc
	PaymentMethod   PaymentMethod    `json:"payment_method"`
	BlockHeight     int64            `json:"block_height"`
	SettledAt       time.Time        `json:"settled_at"`
}

// DepositIntent represents a user's intent to deposit USDC
type DepositIntent struct {
	ID              string        `json:"id"`
	WalletID        string        `json:"wallet_id"`
	UserID          string        `json:"user_id"`
	Amount          int64         `json:"amount"`           // uusdc
	PaymentMethod   PaymentMethod `json:"payment_method"`
	Status          string        `json:"status"`           // pending, confirmed, failed, expired
	PaymentRef      string        `json:"payment_ref"`      // Stripe session ID, wire ref, etc.
	DepositAddress  string        `json:"deposit_address"`  // For crypto deposits
	CreatedAt       time.Time     `json:"created_at"`
	ConfirmedAt     *time.Time    `json:"confirmed_at,omitempty"`
	ExpiresAt       time.Time     `json:"expires_at"`
}

// WithdrawalRequest represents a user's request to withdraw USDC
type WithdrawalRequest struct {
	ID              string        `json:"id"`
	WalletID        string        `json:"wallet_id"`
	UserID          string        `json:"user_id"`
	Amount          int64         `json:"amount"`           // uusdc
	PaymentMethod   PaymentMethod `json:"payment_method"`
	Status          string        `json:"status"`           // pending, processing, completed, failed
	DestinationRef  string        `json:"destination_ref"`  // Bank account, crypto address
	CreatedAt       time.Time     `json:"created_at"`
	ProcessedAt     *time.Time    `json:"processed_at,omitempty"`
}

// SettlementConfig holds settlement module configuration
type SettlementConfig struct {
	// Escrow settings
	EscrowTimeoutHours  int   `json:"escrow_timeout_hours"`
	MinSettlementUSDC   int64 `json:"min_settlement_usdc"`    // Minimum trade size
	MaxSettlementUSDC   int64 `json:"max_settlement_usdc"`    // Maximum single trade

	// Fee structure
	SettlementFeeBPS    int   `json:"settlement_fee_bps"`     // Basis points on settlement
	WithdrawalFeeFlat   int64 `json:"withdrawal_fee_flat"`    // Flat fee per withdrawal (uusdc)
	WithdrawalFeeBPS    int   `json:"withdrawal_fee_bps"`     // Variable fee

	// Payment methods enabled
	BankWireEnabled     bool  `json:"bank_wire_enabled"`
	StripeEnabled       bool  `json:"stripe_enabled"`
	CryptoEnabled       bool  `json:"crypto_enabled"`
	NobleIBCEnabled     bool  `json:"noble_ibc_enabled"`

	// Circle API (for USDC minting)
	CircleAPIEnabled    bool  `json:"circle_api_enabled"`
}

// DefaultSettlementConfig returns testnet defaults
func DefaultSettlementConfig() *SettlementConfig {
	return &SettlementConfig{
		EscrowTimeoutHours:  24,
		MinSettlementUSDC:   500_000,       // $0.50 minimum
		MaxSettlementUSDC:   1_000_000_000_000, // $1M maximum
		SettlementFeeBPS:    10,             // 0.10%
		WithdrawalFeeFlat:   1_000_000,     // $1.00 flat
		WithdrawalFeeBPS:    5,             // 0.05%
		BankWireEnabled:     false,          // Not yet
		StripeEnabled:       true,           // Via Stripe on-ramp
		CryptoEnabled:       true,           // Direct USDC deposit
		NobleIBCEnabled:     false,          // Future: IBC bridge
		CircleAPIEnabled:    false,          // Future: Circle API
	}
}
