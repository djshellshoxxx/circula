package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// closureBlockers lists what stops a closure (FR-068, GOV-025): disputed
// transactions (a dispute stays disputed while posting is pending). Open
// proposals do not block: they are cancelled at the closure date (GOV-025 c).
func closureBlockers(tx store.Tx, netID string) ([]string, error) {
	var blockers []string
	props, err := tx.ListTransactions(netID, store.TxFilter{Statuses: []domain.TxStatus{domain.TxDisputed}, Limit: 500})
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
	OpenProposals   int      // proposals that will be cancelled at the closure date
	Closure         *domain.Closure
	NetworkStatus   domain.NetworkStatus
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
		out.Closure, out.NetworkStatus = p.net.Closure, p.net.Status
		props, err := tx.ListTransactions(netID, store.TxFilter{Statuses: []domain.TxStatus{domain.TxProposed}, Limit: 500})
		if err != nil {
			return err
		}
		out.OpenProposals = len(props)
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
