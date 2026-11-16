package service

import "testing"

// The administrator MFA gate must run even for decider-only routes.
func TestDeciderRejectsMissingAdministratorMFA(t *testing.T) {
 p := &principal{mfaMissing: true}
 if err := p.requireDecider(); err == nil || err.Error() != "forbidden: mfa-enrolment-required" {
  t.Fatalf("missing administrator MFA must be rejected, got %v", err)
 }
}
