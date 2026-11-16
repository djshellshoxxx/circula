package domain

import (
	"errors"
	"fmt"
)

// Code is a stable machine-readable error code. Values match the catalogue in
// docs/specs/07-api-contract.md section 7.2 where one exists.
type Code string

// Error codes returned by the service layer.
const (
	CodeValidation         Code = "validation-failed"
	CodeInsufficientCredit Code = "insufficient-available-credit"
	CodeAmountExceeds      Code = "amount-exceeds-policy"
	CodeCounterpartyInact  Code = "counterparty-not-active"
	CodeKeyReused          Code = "idempotency-key-reused"
	CodeKeyRequired        Code = "idempotency-key-required"
	CodeUnauthenticated    Code = "unauthenticated"
	CodeForbidden          Code = "forbidden"
	CodeNotFound           Code = "not-found"
	CodeInvalidState       Code = "invalid-transaction-state"
	CodeConflict           Code = "conflict"
	CodeBlocked            Code = "blocked"
	CodeRateLimited        Code = "rate-limited"
	// CodeNetworkFrozen: the network's ledger failed verification and the
	// network is read-only until an operator resolves it (CMD-07, OPS-033).
	CodeNetworkFrozen Code = "network-frozen"
	// CodeUnavailable is the retryable "try again" error: the database was busy
	// or locked past the busy timeout. No partial state was written. Adapters
	// map it to 503 with Retry-After.
	CodeUnavailable Code = "unavailable"
	CodeInternal    Code = "internal-error"
)

// FieldError describes one invalid input field.
type FieldError struct {
	Field  string `json:"field"`
	Detail string `json:"detail"`
}

// Error is the single error type returned for expected failures. Anything else
// returned by the service is an internal error.
type Error struct {
	Code     Code
	Msg      string
	Fields   []FieldError // for CodeValidation
	Blockers []string     // for CodeBlocked (departure, closure)
	Cause    error        // underlying error for logs; never shown to callers
}

// Unwrap exposes Cause to errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Cause }

// Error implements error.
func (e *Error) Error() string {
	if e.Msg == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Msg
}

// Is lets errors.Is(err, domain.ErrForbidden) match on Code only.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// Sentinels for errors.Is comparisons.
var (
	ErrValidation         = &Error{Code: CodeValidation}
	ErrInsufficientCredit = &Error{Code: CodeInsufficientCredit}
	ErrAmountExceeds      = &Error{Code: CodeAmountExceeds}
	ErrCounterpartyInact  = &Error{Code: CodeCounterpartyInact}
	ErrKeyReused          = &Error{Code: CodeKeyReused}
	ErrKeyRequired        = &Error{Code: CodeKeyRequired}
	ErrUnauthenticated    = &Error{Code: CodeUnauthenticated}
	ErrForbidden          = &Error{Code: CodeForbidden}
	ErrNotFound           = &Error{Code: CodeNotFound}
	ErrInvalidState       = &Error{Code: CodeInvalidState}
	ErrConflict           = &Error{Code: CodeConflict}
	ErrBlocked            = &Error{Code: CodeBlocked}
	ErrRateLimited        = &Error{Code: CodeRateLimited}
	ErrNetworkFrozen      = &Error{Code: CodeNetworkFrozen}
	ErrUnavailable        = &Error{Code: CodeUnavailable}
)

// IsRetryable reports whether err is a transient failure the caller may retry
// unchanged (database busy). Rate limits are not retryable immediately.
func IsRetryable(err error) bool { return CodeOf(err) == CodeUnavailable }

// Errf builds an Error with a formatted message.
func Errf(c Code, format string, a ...any) *Error {
	return &Error{Code: c, Msg: fmt.Sprintf(format, a...)}
}

// Invalid builds a validation error for a single field.
func Invalid(field, detail string) *Error {
	return &Error{Code: CodeValidation, Msg: field + ": " + detail,
		Fields: []FieldError{{Field: field, Detail: detail}}}
}

// NotFound builds a not-found error. Callers must use it for both missing and
// cross-tenant objects so existence is not revealed (API-017).
func NotFound(what string) *Error { return Errf(CodeNotFound, "%s not found", what) }

// Forbidden builds a permission error.
func Forbidden(format string, a ...any) *Error { return Errf(CodeForbidden, format, a...) }

// InvalidState builds a state-machine refusal.
func InvalidState(format string, a ...any) *Error { return Errf(CodeInvalidState, format, a...) }

// CodeOf returns the Code of err, or CodeInternal for foreign errors, "" for nil.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeInternal
}
