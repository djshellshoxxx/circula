package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// dispute returns the view of dispute id shown to the parties, reviewers and
// administrators (web page, API GET /disputes/{id}).
func (w *world) dispute(id string) *service.DisputeView {
	return tu.Must(w.Svc.GetDispute(w.Ctx, w.admin, w.net.ID, id))
}

// timelines7And14 sets the recommended dispute timelines (DEC-012): a 7-day
// response period and a 14-day re-review window, and room for several trades.
func timelines7And14(p *domain.Policy) {
	p.DefaultCreditLimit = 100000
	p.ResponsePeriodDays = 7
	p.ReReviewWindowDays = 14
}

// GOV-001, DEC-012, FR-076, FR-078 (AUD4-F03): dispute timelines change
// prospectively (10 section 2: "Dispute timelines ... Prospective"). An
// administrator who shortens the response period to 0 or the re-review window
// to 1 day after a dispute was opened changes nothing for that dispute: it
// keeps the response period and re-review window of the policy in force when
// it was opened. The enforced response deadline, the shown deadline and the
// overdue reminder agree. The finding's probe made an open dispute decidable
// with no response and no waiver, and stripped a decided dispute's re-review.
func TestDisputeTimelinesFixedAtOpen(t *testing.T) {
	w := newWorld(t, timelines7And14)
	rev, _ := w.reviewerAs("rita@example.org")
	pending := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 5000).ID))
	decided := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 3000).ID))
	opened := pending.OpenedAt
	if _, err := w.resolve(rev, pending.ID, domain.OutcomeUpheld, 0, "r"); err == nil {
		t.Fatal("control: decided inside the response period without a response or a waiver")
	}
	noErr(t, w.respond(w.b, decided.ID, "it was delivered"))
	decided = tu.Must(w.resolve(rev, decided.ID, domain.OutcomeDismissed, 0, "delivered"))

	w.Clock.Advance(time.Hour)
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.ResponsePeriodDays = 0; p.ReReviewWindowDays = 1 }, nil, ""))
	if p := w.netNow().Policy; p.ResponsePeriodDays != 0 || p.ReReviewWindowDays != 1 {
		t.Fatalf("precondition: the change is in force at once: %d/%d", p.ResponsePeriodDays, p.ReReviewWindowDays)
	}

	// (a) The open dispute keeps its 7-day response period: shown, reminded
	// and enforced alike.
	w.Clock.Advance(2 * day)
	if dl := w.dispute(pending.ID).Deadlines; !dl.ResponseDeadline.Equal(opened.Add(7*day)) || dl.ResponseOverdue {
		t.Errorf("(a) shown response deadline %s overdue=%v, want %s (7 days after opening, as enforced)",
			dl.ResponseDeadline, dl.ResponseOverdue, opened.Add(7*day))
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
		t.Errorf("(a) %d overdue reminders sent 2 days into a 7-day response period", n)
	}
	if _, err := w.resolve(rev, pending.ID, domain.OutcomeDismissed, 0, "r"); err == nil {
		t.Errorf("(a) response period removed for an open dispute: decided with no response and no waiver")
	}

	// (b) The decided dispute keeps its 14-day re-review window.
	w.Clock.Advance(day) // 3 days after the decision
	if c := w.dispute(decided.ID).Deadlines.ReReviewCloses; !c.Equal(decided.DecidedAt.Add(14 * day)) {
		t.Errorf("(b) shown re-review deadline %s, want %s (14 days after the decision, as enforced)", c, decided.DecidedAt.Add(14*day))
	}
	if err := w.rereview(w.a, decided.ID, "new evidence"); err != nil {
		t.Errorf("(b) re-review right stripped retroactively: %v", err)
	}

	// The open dispute becomes decidable exactly when its own period ends.
	w.advanceTo(t, opened.Add(7*day).Add(-time.Millisecond))
	if _, err := w.resolve(rev, pending.ID, domain.OutcomeDismissed, 0, "r"); err == nil {
		t.Errorf("decided 1ms before the response deadline without a response or a waiver")
	}
	w.advanceTo(t, opened.Add(7*day))
	if d, err := w.resolve(rev, pending.ID, domain.OutcomeDismissed, 0, "r"); err != nil || d.Status != domain.DisputeResolved {
		t.Fatalf("not decidable when its own response period ended: %v", err)
	}
}

// decidedDispute opens a dispute on a fresh trade, lets the other party
// respond and has rev decide it, so the decision needs no waiting period.
func (w *world) decidedDispute(t *testing.T, rev domain.Actor) *domain.Dispute {
	t.Helper()
	d := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	noErr(t, w.respond(w.b, d.ID, "it was delivered"))
	return tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "delivered"))
}

// GOV-001, DEC-012, FR-076, FR-078 (AUD4-F03): a change to the dispute
// timelines is prospective both ways. A dispute opened after the change uses
// the new response period and re-review window, whether shorter or longer; a
// longer period or window does not extend a dispute opened before it either.
func TestDisputeTimelineChangesApplyToLaterDisputes(t *testing.T) {
	w := newWorld(t, timelines7And14)
	rev, _ := w.reviewerAs("rita@example.org")
	old := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	oldDecided := w.decidedDispute(t, rev)
	t0 := old.OpenedAt

	// Lengthen to 30 / 60 days.
	w.Clock.Advance(time.Hour)
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.ResponsePeriodDays = 30; p.ReReviewWindowDays = 60 }, nil, ""))
	long := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	longDecided := w.decidedDispute(t, rev)
	if dl := w.dispute(long.ID).Deadlines; !dl.ResponseDeadline.Equal(long.OpenedAt.Add(30 * day)) {
		t.Errorf("later dispute: response deadline %s, want 30 days after opening", dl.ResponseDeadline)
	}
	if dl := w.dispute(old.ID).Deadlines; !dl.ResponseDeadline.Equal(t0.Add(7 * day)) {
		t.Errorf("earlier dispute: response deadline %s, want 7 days after opening (not extended)", dl.ResponseDeadline)
	}
	w.advanceTo(t, t0.Add(7*day))
	if _, err := w.resolve(rev, long.ID, domain.OutcomeDismissed, 0, "r"); err == nil {
		t.Errorf("later dispute decided 7 days into its 30-day response period")
	}
	if _, err := w.resolve(rev, old.ID, domain.OutcomeDismissed, 0, "r"); err != nil {
		t.Errorf("earlier dispute: its 7-day response period was extended by the later change: %v", err)
	}
	if c := w.dispute(oldDecided.ID).Deadlines.ReReviewCloses; !c.Equal(oldDecided.DecidedAt.Add(14 * day)) {
		t.Errorf("earlier dispute: shown re-review deadline %s, want 14 days after the decision", c)
	}
	if c := w.dispute(longDecided.ID).Deadlines.ReReviewCloses; !c.Equal(longDecided.DecidedAt.Add(60 * day)) {
		t.Errorf("later dispute: shown re-review deadline %s, want 60 days after the decision", c)
	}
	w.advanceTo(t, oldDecided.DecidedAt.Add(14*day).Add(time.Millisecond))
	if err := w.rereview(w.a, oldDecided.ID, "new evidence"); err == nil {
		t.Errorf("earlier dispute: its 14-day re-review window was extended by the later change")
	}
	if err := w.rereview(w.a, longDecided.ID, "new evidence"); err != nil {
		t.Errorf("later dispute: refused inside its 60-day re-review window: %v", err)
	}

	// Shorten to 0 / 1 day.
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.ResponsePeriodDays = 0; p.ReReviewWindowDays = 1 }, nil, ""))
	w.Clock.Advance(time.Hour)
	short := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	if dl := w.dispute(short.ID).Deadlines; !dl.ResponseDeadline.Equal(short.OpenedAt) {
		t.Errorf("later dispute: response deadline %s, want the opening instant (no waiting period)", dl.ResponseDeadline)
	}
	short = tu.Must(w.resolve(rev, short.ID, domain.OutcomeDismissed, 0, "r")) // no response, no waiver needed
	if c := w.dispute(short.ID).Deadlines.ReReviewCloses; !c.Equal(short.DecidedAt.Add(day)) {
		t.Errorf("later dispute: shown re-review deadline %s, want 1 day after the decision", c)
	}
	w.advanceTo(t, short.DecidedAt.Add(day).Add(time.Millisecond))
	if err := w.rereview(w.a, short.ID, "new evidence"); err == nil {
		t.Errorf("later dispute: re-review accepted after its 1-day window")
	}
}

// GOV-001, GOV-003, DEC-012 (AUD4-F03): a timeline change announced with a
// notice period is not in force before its date, so a dispute opened during
// the notice keeps the old response period and re-review window, also after
// the change takes effect. A dispute opened from that date uses the new ones.
func TestScheduledDisputeTimelineChangeWaitsForItsDate(t *testing.T) {
	w := newWorld(t, timelines7And14)
	rev, _ := w.reviewerAs("rita@example.org")
	t0 := w.Clock.Now()
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.ResponsePeriodDays = 0; p.ReReviewWindowDays = 1 }, iptr(10), ""))
	if p := w.netNow().Policy; p.ResponsePeriodDays != 7 || p.ReReviewWindowDays != 14 {
		t.Fatalf("precondition: the scheduled change is in force early: %d/%d", p.ResponsePeriodDays, p.ReReviewWindowDays)
	}

	w.Clock.Advance(day)
	during := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	duringDecided := w.decidedDispute(t, rev)
	if _, err := w.resolve(rev, during.ID, domain.OutcomeDismissed, 0, "r"); err == nil {
		t.Errorf("dispute opened during the notice was decided without a response or a waiver")
	}
	if dl := w.dispute(during.ID).Deadlines; !dl.ResponseDeadline.Equal(during.OpenedAt.Add(7 * day)) {
		t.Errorf("dispute opened during the notice: response deadline %s, want 7 days after opening", dl.ResponseDeadline)
	}

	w.advanceTo(t, t0.Add(10*day))
	if p := w.netNow().Policy; p.ResponsePeriodDays != 0 || p.ReReviewWindowDays != 1 {
		t.Fatalf("precondition: the scheduled change is in force on its date: %d/%d", p.ResponsePeriodDays, p.ReReviewWindowDays)
	}
	after := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	if _, err := w.resolve(rev, after.ID, domain.OutcomeDismissed, 0, "r"); err != nil {
		t.Errorf("dispute opened on the change's date keeps a response period: %v", err)
	}
	w.Clock.Advance(2 * day) // 11 days after the decision: inside 14, outside 1
	if err := w.rereview(w.a, duringDecided.ID, "new evidence"); err != nil {
		t.Errorf("dispute opened during the notice lost its re-review window when the change took effect: %v", err)
	}
}
