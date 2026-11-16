package service

import (
 "testing"
 "github.com/djshellshoxxx/circula/internal/domain"
)

func TestReverseRejectsAdministratorRequiredNetwork(t *testing.T) {
 net := &domain.Network{Status: domain.NetworkActive, AdminRequired: true}
 s := &Service{}
 _, err := s.reverse(nil, net, nil, 1, "test", "", "")
 if err == nil { t.Fatal("reversal must reject administrator-required network before touching transaction") }
}

func TestDeciderRequiresAdminMFA(t *testing.T) {
 p := &principal{mfaMissing: true}
 // The decider must refuse missing administrator MFA before role-based processing.
 if err := p.requireDecider(); err == nil { t.Fatal("missing required admin MFA accepted") }
}
