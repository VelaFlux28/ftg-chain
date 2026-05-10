package types

import "errors"

var (
	ErrWalletNotFound       = errors.New("exchange wallet not found")
	ErrWalletExists         = errors.New("wallet already exists for this user")
	ErrWalletInactive       = errors.New("wallet is inactive/frozen")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrInsufficientKYC      = errors.New("insufficient KYC tier for this operation")
	ErrDailyLimitExceeded   = errors.New("daily trading limit exceeded for your KYC tier")
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderCancelled       = errors.New("order already cancelled")
	ErrOrderFilled          = errors.New("order already filled")
	ErrInvalidOrderSide     = errors.New("invalid order side (must be buy or sell)")
	ErrInvalidOrderType     = errors.New("invalid order type (must be limit or market)")
	ErrInvalidPrice         = errors.New("invalid price (must be positive)")
	ErrInvalidQuantity      = errors.New("invalid quantity (must be positive)")
	ErrBelowMinOrder        = errors.New("order below minimum size")
	ErrPairNotFound         = errors.New("trading pair not found")
	ErrPairInactive         = errors.New("trading pair is not active")
	ErrTradingDisabled      = errors.New("trading is currently disabled")
	ErrRegistrationClosed   = errors.New("new registrations are currently closed")
	ErrKYCAlreadyPending    = errors.New("KYC application already pending review")
	ErrKYCAlreadyApproved   = errors.New("already approved at this tier or higher")
	ErrSanctionedEntity     = errors.New("entity appears on sanctions list — cannot proceed")
	ErrPEPRequiresReview    = errors.New("politically exposed person — requires enhanced review")
	ErrSelfTrade            = errors.New("self-trading is not permitted")
)
