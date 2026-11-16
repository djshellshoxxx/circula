package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// NetworkAccess is a read-authorized snapshot for HTTP adapters. Member has
// effective roles, including live break-glass grants. The capability flags
// additionally enforce current membership status and required administrator MFA.
type NetworkAccess struct {
	Network   *domain.Network
	Member    *domain.Member
	CanAdmin  bool
	CanReview bool
}

// GetNetworkAccess resolves the caller's current authority in one snapshot.
// It uses read scope so frozen networks remain available for investigation.
// Commands must still enforce their own write and operation-specific guards.
func (s *Service) GetNetworkAccess(ctx context.Context, actor domain.Actor, netID string) (*NetworkAccess, error) {
	var out *NetworkAccess
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		out = &NetworkAccess{Network: p.net, Member: p.member, CanAdmin: p.requireAdmin() == nil,
			CanReview: p.requireDecider() == nil}
		return nil
	})
	return out, err
}

// AuthorizeAdminRead gates adapter-owned administrative representations, such
// as an in-memory consistency result, using the same authority as service reads.
func (s *Service) AuthorizeAdminRead(ctx context.Context, actor domain.Actor, netID string) error {
	return s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		return p.requireAdmin()
	})
}
