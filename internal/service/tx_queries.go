package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// TxView is a transaction with its ledger entries and status history.
type TxView struct {
	Tx      *domain.Transaction
	Entries []*domain.Entry
	History []*domain.StatusEvent
}

// canSeeTx: parties, administrators, and reviewers of an existing dispute.
func canSeeTx(tx store.Tx, p *principal, t *domain.Transaction) (bool, error) {
	if p.actor.System || t.IsParty(p.memberID()) {
		return true, nil
	}
	if p.requireAdmin() == nil {
		return true, nil
	}
	if p.requireDecider() == nil {
		ds, err := tx.ListDisputes(p.net.ID, store.DisputeFilter{TxID: t.ID, Limit: 1})
		return len(ds) > 0, err
	}
	return false, nil
}

// GetTransaction returns one transaction to a party, an administrator or a
// reviewer of a dispute on it. Anyone else, and any id from another network,
// gets the same not-found as a missing id (SEC-009, API-017).
func (s *Service) GetTransaction(ctx context.Context, actor domain.Actor, netID, txID string) (*TxView, error) {
	var out *TxView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		t, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return domain.NotFound("transaction")
		}
		if ok, err := canSeeTx(tx, p, t); err != nil {
			return err
		} else if !ok {
			return domain.NotFound("transaction")
		}
		v := &TxView{Tx: t}
		if v.Entries, err = tx.ListEntries(netID, txID); err != nil {
			return err
		}
		v.History, err = tx.ListStatusEvents(netID, txID)
		out = v
		return err
	})
	return out, err
}

// ListTransactions is the caller's own history, newest first, filterable by
// status. Suspended and departed members can read it (DOM-039).
func (s *Service) ListTransactions(ctx context.Context, actor domain.Actor, netID string, statuses []domain.TxStatus, pg Page) ([]*domain.Transaction, error) {
	var out []*domain.Transaction
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no history")
		}
		out, err = tx.ListTransactions(netID, store.TxFilter{MemberID: p.member.ID, Statuses: statuses, AfterID: pg.AfterID, Limit: clampPage(pg)})
		return err
	})
	return out, err
}

// AdminListTransactions lists every transaction of the network (administrators).
func (s *Service) AdminListTransactions(ctx context.Context, actor domain.Actor, netID string, kind domain.TxKind, statuses []domain.TxStatus, pg Page) ([]*domain.Transaction, error) {
	var out []*domain.Transaction
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		out, err = tx.ListTransactions(netID, store.TxFilter{Kind: kind, Statuses: statuses, AfterID: pg.AfterID, Limit: clampPage(pg)})
		return err
	})
	return out, err
}
