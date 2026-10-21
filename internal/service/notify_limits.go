package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// LimitWarningPercent is the share of the credit limit in use at which a
// member is warned (UX-012, default proposed 80 percent; OQ-UX-02 is open, so
// it is a platform constant until the product owner makes it a network policy).
const LimitWarningPercent = 80

// limitWarnBatch bounds the members handled in one write unit.
const limitWarnBatch = 200

// LimitWarning reports whether the balance view has reached the warning
// threshold, and how much credit is in use. Credit is in use only while the
// balance is negative; a member without a limit is never warned.
func LimitWarning(b *BalanceView) (used int64, warn bool) {
	if b == nil || b.CreditLimit <= 0 || b.Balance >= 0 {
		return 0, false
	}
	used = -b.Balance
	return used, used*100 >= b.CreditLimit*LimitWarningPercent
}

// RunLimitWarnings notifies every active member whose used credit has reached
// LimitWarningPercent of their effective limit (UX-012). It is the body of a
// scheduled job: it takes no actor, spans all networks (closed ones are
// skipped), and is safe to run as often as wanted because a member gets at
// most one warning per ISO week (dedupe key). It returns the number of
// notifications created. Run it from the job runner, for example hourly.
func (s *Service) RunLimitWarnings(ctx context.Context) (int, error) {
	var ids []string
	if err := s.view(ctx, func(tx store.Tx) (err error) {
		ids, err = tx.ListNetworkIDs()
		return err
	}); err != nil {
		return 0, err
	}
	total := 0
	for _, netID := range ids {
		after := ""
		for {
			var n, scanned int
			var last string
			err := s.update(ctx, func(tx store.Tx) error {
				net, err := tx.GetNetwork(netID)
				if errors.Is(err, domain.ErrNotFound) {
					return nil
				}
				if err != nil {
					return err
				}
				if net.Status != domain.NetworkActive || net.Suspended() {
					return nil
				}
				ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive},
					AfterID: after, Limit: limitWarnBatch})
				if err != nil {
					return err
				}
				scanned = len(ms)
				now := s.now()
				year, week := now.ISOWeek()
				for _, m := range ms {
					last = m.ID
					acc, err := tx.GetAccountByMember(netID, m.ID)
					if err != nil {
						return err
					}
					b := &BalanceView{Balance: acc.Balance, CreditLimit: acc.EffectiveLimit(net.Policy.DefaultCreditLimit, now)}
					used, warn := LimitWarning(b)
					if !warn {
						continue
					}
					body := fmt.Sprintf("You are using %s of your %s %s credit limit.", fmtAmt(net, used), fmtAmt(net, b.CreditLimit), net.Policy.CurrencyName)
					created, err := tx.InsertNotification(store.Notification{ID: s.id(), NetworkID: netID, MemberID: m.ID,
						Kind: NotifLimitWarning, Title: "You are close to your credit limit", Body: body, Link: "/",
						DedupeKey: fmt.Sprintf("limit-warning:%d-W%02d", year, week), CreatedAt: now})
					if err != nil {
						return err
					}
					if created {
						n++
					}
				}
				return nil
			})
			if err != nil {
				return total, err
			}
			total += n
			if scanned < limitWarnBatch {
				break
			}
			after = last
		}
	}
	return total, nil
}
