package service

import (
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// AuthenticationObserver is an optional Observer extension for refused
// credential operations (OPS-014). Callbacks run once after the operation's
// transactions finish, on the calling goroutine. They must be fast and must
// not call back into the Service. Neither callback receives credential or
// identity data. A throttled operation is reported only as throttled, not as
// a failed credential comparison.
type AuthenticationObserver interface {
	AuthenticationFailed(reason string)
	AuthenticationThrottled(reason string)
}

var authenticationFailureClasses = [...]string{
	"password", "mfa", "recovery", "session", "api_token",
	"password_proof", "mfa_management", "other",
}

// AuthenticationFailureClasses returns the fixed credential-operation
// classes used by failure and throttling metrics.
func AuthenticationFailureClasses() []string {
	return append([]string(nil), authenticationFailureClasses[:]...)
}

// NormalizeAuthenticationFailureClass prevents unexpected observer input
// from becoming an identity, credential or free-text metric label.
func NormalizeAuthenticationFailureClass(reason string) string {
	for _, known := range authenticationFailureClasses {
		if reason == known {
			return known
		}
	}
	return "other"
}

// observeAuthentication is called only by the outer credential operations.
// Login delegates to LoginStep and is intentionally not observed again.
// Storage failures, password-policy validation and authorization/state
// refusals are not failed credential proofs. MFA management represents a bad
// password or factor as a typed field validation error, including on HTTP 422.
func (s *Service) observeAuthentication(reason string, err error) {
	o, ok := s.obs.(AuthenticationObserver)
	if !ok || err == nil {
		return
	}
	reason = NormalizeAuthenticationFailureClass(reason)
	switch domain.CodeOf(err) {
	case domain.CodeUnauthenticated:
		o.AuthenticationFailed(reason)
	case domain.CodeRateLimited:
		o.AuthenticationThrottled(reason)
	case domain.CodeValidation:
		if reason != "mfa_management" {
			return
		}
		var de *domain.Error
		if !errors.As(err, &de) {
			return
		}
		for _, field := range de.Fields {
			if field.Field == "password" || field.Field == "code" {
				o.AuthenticationFailed(reason)
				return
			}
		}
	}
}
