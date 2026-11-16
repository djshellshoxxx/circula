package service

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Administrator MFA must be enforced even when an action uses the decider gate.
func TestDeciderRejectsMissingAdministratorMFA(t *testing.T) {
	p := &principal{member: &domain.Member{Status: domain.MemberActive, Roles: []domain.Role{domain.RoleAdmin}}, mfaMissing: true}
	if err := p.requireDecider(); err == nil {
		t.Fatal("missing administrator MFA must be rejected")
	}
	p.mfaMissing = false
	if err := p.requireDecider(); err != nil {
		t.Fatalf("enrolled administrator should be allowed: %v", err)
	}
}
