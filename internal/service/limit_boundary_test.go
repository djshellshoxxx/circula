package service_test

import (
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-004: the credit-limit boundary for limits of zero, one minor unit and a large value.
// A posting that lands exactly on -limit is accepted; one minor unit beyond is refused;
// a zero limit refuses any outgoing amount.
func TestQA004LimitBoundary(t *testing.T) {
	for _, limit := range []int64{0, 1, 1_000_000_00} {
		t.Run(fmt.Sprintf("limit_%d", limit), func(t *testing.T) {
			w := newWorld(t, func(p *domain.Policy) {
				p.DefaultCreditLimit = limit
				p.MaxTxAmount = 1_000_000_00 + 10
			})
			// One unit beyond the limit is refused at proposal and leaves nothing behind.
			_, err := w.propose(w.a, w.bM, service.ProposerPays, limit+1)
			wantCode(t, err, domain.ErrInsufficientCredit)
			if limit == 0 {
				// A zero limit refuses any negative balance, so even one unit is refused (checked above).
				if got := w.balance(w.a); got.Available != 0 || got.Balance != 0 {
					t.Errorf("zero limit: %+v", got)
				}
				return
			}
			// Exactly on the limit is accepted and posts; balance equals -limit.
			w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, limit)
			if got := w.balance(w.a); got.Balance != -limit || got.Available != 0 {
				t.Errorf("on the limit: %+v", got)
			}
			// Now one more minor unit is refused, and the balance is unchanged.
			_, err = w.propose(w.a, w.bM, service.ProposerPays, 1)
			wantCode(t, err, domain.ErrInsufficientCredit)
			if got := w.balance(w.a).Balance; got != -limit {
				t.Errorf("balance moved after refusal: %d", got)
			}
			if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
				t.Errorf("ledger: %v", rep.Findings)
			}
		})
	}
}
