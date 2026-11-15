package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/histbal"
	"github.com/djshellshoxxx/circula/internal/store"
)

var _ histbal.Reader = (*Service)(nil)

// HistoricalBalance is the caller's own posted balance at the instant asOf
// (FR-054, DAT-050, API-083). Everything is read in one view transaction, so
// the timestamp sum and the high-water ledger_seq come from the same snapshot.
// The cutoff is the stored posted_at, inclusive; it is never translated to a
// ledger_seq prefix. Pending proposals have no entries and so never count.
//
// asOf must already be at millisecond precision and not after the service
// clock; the service refuses rather than rounds, so a caller that skipped
// histbal.ParseInstant still cannot get a silently altered instant.
func (s *Service) HistoricalBalance(ctx context.Context, actor domain.Actor, netID string, asOf time.Time) (histbal.Result, error) {
	asOf = asOf.UTC()
	if asOf.Nanosecond()%int(time.Millisecond) != 0 {
		return histbal.Result{}, &histbal.InputError{Reason: histbal.ReasonPrecision}
	}
	if asOf.After(s.now()) {
		return histbal.Result{}, &histbal.InputError{Reason: histbal.ReasonFuture}
	}
	var out histbal.Result
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		// Own account only; suspended and departed members keep read access (DOM-039).
		if p.member == nil || p.member.Status == domain.MemberPending {
			return domain.Forbidden("no account")
		}
		acc, err := tx.GetAccountByMember(netID, p.member.ID)
		if err != nil {
			return err
		}
		sum, n, err := tx.SumEntriesAsOf(acc.ID, asOf)
		if err != nil {
			return err
		}
		seq, err := tx.LedgerSeq(netID)
		if err != nil {
			return err
		}
		out = histbal.Result{AsOf: asOf, BalanceMinor: sum, Decimals: p.net.Policy.Decimals,
			LedgerSeq: seq, HasPostingsByAsOf: n > 0}
		return nil
	})
	return out, err
}
