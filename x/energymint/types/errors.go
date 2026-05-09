package types

import "fmt"

// Custom error types for the energymint module

type InvalidCertificateError struct {
	Reason string
}

func (e InvalidCertificateError) Error() string {
	return fmt.Sprintf("invalid certificate: %s", e.Reason)
}

func ErrInvalidCertificate(reason string) error {
	return InvalidCertificateError{Reason: reason}
}

type UnauthorizedMinterError struct {
	Address string
}

func (e UnauthorizedMinterError) Error() string {
	return fmt.Sprintf("unauthorized minter: %s", e.Address)
}

func ErrUnauthorizedMinter(addr string) error {
	return UnauthorizedMinterError{Address: addr}
}

type DuplicateCertificateError struct {
	ExternalID string
}

func (e DuplicateCertificateError) Error() string {
	return fmt.Sprintf("certificate already registered: %s", e.ExternalID)
}

func ErrDuplicateCertificate(externalID string) error {
	return DuplicateCertificateError{ExternalID: externalID}
}

type InsufficientMwhError struct {
	Provided string
	Minimum  string
}

func (e InsufficientMwhError) Error() string {
	return fmt.Sprintf("insufficient MWh: provided %s, minimum %s", e.Provided, e.Minimum)
}

func ErrInsufficientMwh(provided, minimum string) error {
	return InsufficientMwhError{Provided: provided, Minimum: minimum}
}
