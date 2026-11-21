package service

import (
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

type authenticationClassObserver struct{ failed, throttled int }

func (*authenticationClassObserver) Posted(int)                       {}
func (*authenticationClassObserver) Replayed(int)                     {}
func (o *authenticationClassObserver) AuthenticationFailed(string)    { o.failed++ }
func (o *authenticationClassObserver) AuthenticationThrottled(string) { o.throttled++ }

// Only typed credential refusals count. Text that resembles an authentication
// error cannot turn a policy, state or storage failure into a credential metric.
func TestAuthenticationObservationUsesTypedCredentialRefusals(t *testing.T) {
	o := &authenticationClassObserver{}
	s := &Service{obs: o}
	for _, tc := range []struct {
		class string
		err   error
	}{
		{"password", errors.New("unauthenticated: password is not correct")},
		{"mfa_management", domain.Invalid("note", "password or code is not correct")},
		{"password_proof", domain.Invalid("password", "must satisfy the new password policy")},
		{"mfa", domain.ErrForbidden},
		{"mfa", domain.ErrInvalidState},
		{"password", domain.Errf(domain.CodeMFARequired, "second factor required")},
	} {
		s.observeAuthentication(tc.class, tc.err)
	}
	if o.failed != 0 || o.throttled != 0 {
		t.Fatal("unrelated failure counted as a credential refusal")
	}
	s.observeAuthentication("mfa_management", domain.Invalid("code", "fixed credential refusal"))
	s.observeAuthentication("mfa_management", domain.Invalid("password", "fixed credential refusal"))
	s.observeAuthentication("password", domain.ErrRateLimited)
	if o.failed != 2 || o.throttled != 1 {
		t.Fatalf("failed=%d throttled=%d", o.failed, o.throttled)
	}
	classes := AuthenticationFailureClasses()
	classes[0] = "changed-copy"
	if AuthenticationFailureClasses()[0] == "changed-copy" || NormalizeAuthenticationFailureClass("private-input") != "other" {
		t.Fatal("authentication metric vocabulary was expanded")
	}
}
