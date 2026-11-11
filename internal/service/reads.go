package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// GetMember returns one member by id, so adapters never scan lists to find one
// (API-04).
//   - the caller's own membership, any status: the full record;
//   - an administrator (or System): the full record of any member of the network;
//   - an active member looking at another ACTIVE member: the directory subset
//     only (ID, NetworkID, DisplayName, Status), exactly what ListDirectory shows
//     (FR-020, SEC-016);
//   - everything else, including other networks' ids: not-found.
func (s *Service) GetMember(ctx context.Context, actor domain.Actor, netID, memberID string) (*domain.Member, error) {
	var out *domain.Member
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		m, err := tx.GetMember(netID, memberID)
		if err != nil {
			return domain.NotFound("member")
		}
		switch {
		case p.actor.System || p.memberID() == m.ID || p.requireAdmin() == nil:
			out = m
		case p.requireActive() == nil && m.Status == domain.MemberActive:
			out = &domain.Member{ID: m.ID, NetworkID: m.NetworkID, DisplayName: m.DisplayName, Status: m.Status}
		default:
			return domain.NotFound("member")
		}
		return nil
	})
	return out, err
}

// GetListing returns one listing by id. Active members see active listings of
// active owners and all of their own; anything else, including other networks'
// ids, is not-found (FR-023, FR-025).
func (s *Service) GetListing(ctx context.Context, actor domain.Actor, netID, listingID string) (*domain.Listing, error) {
	var out *domain.Listing
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		l, err := tx.GetListing(netID, listingID)
		if err != nil {
			return domain.NotFound("listing")
		}
		if l.MemberID != p.member.ID {
			owner, err := tx.GetMember(netID, l.MemberID)
			if err != nil || l.Status != domain.ListingActive || owner.Status != domain.MemberActive {
				return domain.NotFound("listing")
			}
		}
		out = l
		return nil
	})
	return out, err
}

// GetDisputeForTx returns the most recent dispute on a transaction to the
// people allowed to see it (parties, reviewers, administrators); everyone else,
// and a transaction without disputes, gets not-found.
func (s *Service) GetDisputeForTx(ctx context.Context, actor domain.Actor, netID, txID string) (*DisputeView, error) {
	var out *DisputeView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		ds, err := tx.ListDisputes(netID, store.DisputeFilter{TxID: txID, Limit: 1})
		if err != nil {
			return err
		}
		if len(ds) == 0 {
			return domain.NotFound("dispute")
		}
		v, err := s.disputeView(tx, p.net, ds[0])
		if err != nil {
			return err
		}
		if !p.actor.System && !v.Tx.IsParty(p.memberID()) && p.requireDecider() != nil {
			return domain.NotFound("dispute")
		}
		out = v
		return nil
	})
	return out, err
}

// EntryQuery pages the caller's ledger entries, newest first by ledger_seq.
// BeforeSeq is the keyset cursor: 0 starts at the newest; pass the LedgerSeq of
// the last row returned to get the next page. Limit 0 means 100, max 500.
type EntryQuery struct {
	BeforeSeq int64
	Limit     int
}

// EntryRow is one ledger entry with the transaction it belongs to.
type EntryRow struct {
	Entry *domain.Entry
	Tx    *domain.Transaction
}

// ListEntries lists the caller's own ledger entries ordered by ledger_seq
// descending (API-045, API-03). Postings, not proposals, define the order, so
// it is stable even though transaction ids are assigned at proposal time. A
// page's last Entry.LedgerSeq is the next BeforeSeq. Suspended and departed
// members can read their own history (DOM-039); pending members and System
// actors have no account and are refused.
func (s *Service) ListEntries(ctx context.Context, actor domain.Actor, netID string, q EntryQuery) ([]EntryRow, error) {
	var out []EntryRow
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no history")
		}
		acc, err := tx.GetAccountByMember(netID, p.member.ID)
		if err != nil {
			return err
		}
		es, err := tx.ListAccountEntries(netID, acc.ID, q.BeforeSeq, clampPage(Page{Limit: q.Limit}))
		if err != nil {
			return err
		}
		for _, e := range es {
			t, err := tx.GetTransaction(netID, e.TxID)
			if err != nil {
				return err
			}
			out = append(out, EntryRow{Entry: e, Tx: t})
		}
		return nil
	})
	return out, err
}

// PendingQuery selects open proposals for the caller.
type PendingQuery struct {
	// AwaitingMe limits to proposals the caller must answer (made by the other
	// party). False also includes the caller's own open proposals.
	AwaitingMe bool
	Page       Page // keyset on transaction id, newest first
}

// ListPendingForActor lists the caller's open (proposed, unexpired) trades in
// one filtered, keyset-paged query, so a dashboard never misses an old proposal
// behind a fixed window (WEB-02, FR-053). Page by passing the last id as
// Page.AfterID; there is no row cap other than the page size (max 500).
func (s *Service) ListPendingForActor(ctx context.Context, actor domain.Actor, netID string, q PendingQuery) ([]*domain.Transaction, error) {
	var out []*domain.Transaction
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no history")
		}
		f := store.TxFilter{MemberID: p.member.ID, Statuses: []domain.TxStatus{domain.TxProposed}, Kind: domain.KindTrade,
			LiveAt: s.now(), AfterID: q.Page.AfterID, Limit: clampPage(q.Page)}
		if q.AwaitingMe {
			f.NotCreatedBy = p.member.ID
		}
		out, err = tx.ListTransactions(netID, f)
		return err
	})
	return out, err
}
