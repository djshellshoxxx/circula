package service

import (
	"context"
	"sort"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// maxReversalLinks is the store page size, not the total reversal count.
const maxReversalLinks = 500

// ReversalsOf returns the reversals posted against transaction txID, oldest first, so
// an original line can link to what reversed it (UX-017). It only reads. Whoever may
// see the original (GetTransaction) may see the list; anyone else, and any id from
// another network, gets the same not-found as a missing id (SEC-009).
func (s *Service) ReversalsOf(ctx context.Context, actor domain.Actor, netID, txID string) ([]*domain.Transaction, error) {
	var out []*domain.Transaction
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
		// ListTransactions is ordered by ID descending and caps each page at 500.
		// Fetch every page before sorting by ledger sequence; otherwise an original
		// with many small partial reversals silently loses history (UX-017).
		cursor := ""
		for {
			page, err := tx.ListTransactions(netID, store.TxFilter{
				OriginalTxID: t.ID, Kind: domain.KindReversal,
				AfterID: cursor, Limit: maxReversalLinks,
			})
			if err != nil {
				return err
			}
			out = append(out, page...)
			if len(page) < maxReversalLinks {
				break
			}
			cursor = page[len(page)-1].ID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].LedgerSeq != out[j].LedgerSeq {
			return out[i].LedgerSeq < out[j].LedgerSeq
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
