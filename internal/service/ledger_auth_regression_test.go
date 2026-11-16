package service

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestReverseRejectsAdministratorRequiredNetwork(t *testing.T) {
	net := &domain.Network{Status: domain.NetworkActive, AdminRequired: true}
	s := &Service{}
	_, err := s.reverse(nil, net, nil, 1, "test", "", "")
	if err == nil {
		t.Fatal("reversal must reject administrator-required network before accessing transaction")
	}
}
