package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DisputeQuery filters and pages the dispute list. Disputes come newest first; page forward by
// passing the last id as Page.AfterID.
type DisputeQuery struct {
	Statuses []domain.DisputeStatus // empty means every status
	TxID     string                 // only disputes on this transaction
	Page     Page
}

// FindDisputes is ListDisputes with a transaction filter and keyset paging. Reviewers and
// administrators see every dispute; other members see disputes on trades they are party to.
// Callers that must show all matches page until a short page comes back; there is no
// fixed window.
func (s *Service) FindDisputes(ctx context.Context, actor domain.Actor, netID string, q DisputeQuery) ([]*domain.Dispute, error) {
	var out []*domain.Dispute
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		f := store.DisputeFilter{Statuses: q.Statuses, TxID: q.TxID, AfterID: q.Page.AfterID, Limit: clampPage(q.Page)}
		if !p.actor.System && p.requireDecider() != nil {
			if p.member == nil || p.member.Status == domain.MemberPending {
				return domain.ErrForbidden
			}
			f.PartyID = p.member.ID
		}
		out, err = tx.ListDisputes(netID, f)
		return err
	})
	return out, err
}
