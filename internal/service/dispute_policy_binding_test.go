package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-001 / AUD4-F03: an appended policy must never replace the policy of
// an existing dispute, even if wall-clock timestamps do not establish order.
func TestDisputeTimelinesSurviveEqualOrReversedPolicyClocks(t *testing.T) {
	for _, shift := range []struct {
		name string
		by   time.Duration
	}{{"same millisecond", 0}, {"clock rolled backward", -time.Hour}} {
		t.Run(shift.name, func(t *testing.T) {
			w := newWorld(t, timelines7And14)
			rev, _ := w.reviewerAs("rita@example.org")
			w.Clock.Advance(day) // rollback stays after the initial network policy
			pending := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
			decided := w.decidedDispute(t, rev)
			w.Clock.Advance(shift.by)
			tu.Must(w.setPolicy(func(p *domain.Policy) {
				p.ResponsePeriodDays = 0
				p.ReReviewWindowDays = 1
			}, nil, ""))

			// A fresh dispute uses the new policy in the same write order,
			// even though its timestamp can precede the old dispute's timestamp.
			fresh := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
			if dl := w.dispute(fresh.ID).Deadlines; !dl.ResponseDeadline.Equal(fresh.OpenedAt) {
				t.Errorf("new dispute did not use the new zero-day period: %s", dl.ResponseDeadline)
			}
			if _, err := w.resolve(rev, fresh.ID, domain.OutcomeDismissed, 0, "new policy"); err != nil {
				t.Fatalf("new dispute cannot be decided under its own policy: %v", err)
			}

			w.Clock.Advance(2*day - shift.by)
			wantResponse := pending.OpenedAt.Add(7 * day)
			if dl := w.dispute(pending.ID).Deadlines; !dl.ResponseDeadline.Equal(wantResponse) || dl.ResponseOverdue {
				t.Errorf("existing response period changed: %+v, want %s", dl, wantResponse)
			}
			if _, err := w.resolve(rev, pending.ID, domain.OutcomeDismissed, 0, "no waiver"); err == nil {
				t.Error("existing dispute decided before its original response deadline")
			}
			if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
				t.Errorf("sent %d overdue reminders before the original response deadline", n)
			}
			wantReview := decided.DecidedAt.Add(14 * day)
			if got := w.dispute(decided.ID).Deadlines.ReReviewCloses; !got.Equal(wantReview) {
				t.Errorf("existing re-review window changed: %s, want %s", got, wantReview)
			}
			if err := w.rereview(w.a, decided.ID, "new evidence"); err != nil {
				t.Errorf("original re-review window was lost: %v", err)
			}
		})
	}
}
