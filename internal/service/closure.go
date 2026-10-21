package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// closureBlockers lists what stops a closure (FR-068): proposed, disputed and
// posting-pending transactions.
func closureBlockers(tx store.Tx, netID string) ([]string, error) {
	var blockers []string
	props, err := tx.ListTransactions(netID, store.TxFilter{Statuses: []domain.TxStatus{domain.TxProposed, domain.TxDisputed}, Limit: 500})
	if err != nil {
		return nil, err
	}
	for _, t := range props {
		blockers = append(blockers, string(t.Status)+" transaction "+t.ID)
	}
	return blockers, nil
}

// ClosurePreview is what an administrator sees before confirming a closure.
type ClosurePreview struct {
	Blockers        []string // closure is refused while any exist
	NonZeroBalances int      // member accounts that will close with a balance (warning only, RSK-020)
	Members         int      // members who will be notified
}

// PreviewClosure reports the blockers and the balances that would remain. It
// changes nothing. Administrator only.
func (s *Service) PreviewClosure(ctx context.Context, actor domain.Actor, netID string) (*ClosurePreview, error) {
	out := &ClosurePreview{}
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if out.Blockers, err = closureBlockers(tx, netID); err != nil {
			return err
		}
		after := ""
		for {
			ms, err := tx.ListMembers(netID, store.MemberFilter{AfterID: after, Limit: 500})
			if err != nil {
				return err
			}
			for _, m := range ms {
				if m.Status == domain.MemberActive {
					out.Members++
				}
				if a, err := tx.GetAccountByMember(netID, m.ID); err == nil && a.Balance != 0 {
					out.NonZeroBalances++
				}
			}
			if len(ms) < 500 {
				return nil
			}
			after = ms[len(ms)-1].ID
		}
	})
	return out, err
}
