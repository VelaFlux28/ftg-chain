package types

import "fmt"

// Custom error types for the burn module

type InvalidBurnError struct {
	Reason string
}

func (e InvalidBurnError) Error() string {
	return fmt.Sprintf("invalid burn: %s", e.Reason)
}

func ErrInvalidBurn(reason string) error {
	return InvalidBurnError{Reason: reason}
}

type UnauthorizedMerchantError struct {
	Address string
}

func (e UnauthorizedMerchantError) Error() string {
	return fmt.Sprintf("unauthorized merchant: %s", e.Address)
}

func ErrUnauthorizedMerchant(addr string) error {
	return UnauthorizedMerchantError{Address: addr}
}

type MerchantProtocolDisabledError struct{}

func (e MerchantProtocolDisabledError) Error() string {
	return "merchant protocol is currently disabled"
}

func ErrMerchantProtocolDisabled() error {
	return MerchantProtocolDisabledError{}
}

type InsufficientBalanceError struct {
	Has  string
	Need string
}

func (e InsufficientBalanceError) Error() string {
	return fmt.Sprintf("insufficient balance: has %s, needs %s", e.Has, e.Need)
}

func ErrInsufficientBalance(has, need string) error {
	return InsufficientBalanceError{Has: has, Need: need}
}
