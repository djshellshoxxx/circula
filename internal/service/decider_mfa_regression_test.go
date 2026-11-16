package service

import "testing"

// Administrator MFA must be enforced even when an action uses the decider gate.
func TestDeciderRejectsMissingAdministratorMFA(t *testing.T) {
	p := &principal{mfaMissing: true}
	if err := p.requireDecider(); err == nil {
		t.Fatal("missing administrator MFA must be rejected")
	}
}
