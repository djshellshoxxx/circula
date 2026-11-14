package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-013 (GOV-013, GOV-014, FR-071, FR-076, FR-078): every dispute deadline is
// elapsed-time arithmetic on UTC instants. Driven by the injected clock, the
// raise window, the response deadline, the decision target and the re-review
// window each fall exactly N*24h after their start, whether the window spans a
// daylight-saving change (EU, US, southern hemisphere) or the end of the year,
// and whatever zone the injected clock reports its instants in. One
// millisecond either side of each boundary is checked.

// zonedClock reports the inner clock's instants in loc, as a host clock in a
// daylight-saving zone would.
type zonedClock struct {
	inner domain.Clock
	loc   *time.Location
}

func (z zonedClock) Now() time.Time { return z.inner.Now().In(z.loc) }

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name) // the service package embeds time/tzdata
	if err != nil {
		t.Fatalf("zone %s: %v", name, err)
	}
	return loc
}

// newDeadlineWorld is newWorld with the policy windows set to the defaults and
// the service clock reporting its instants in loc.
func newDeadlineWorld(t *testing.T, loc *time.Location) *world {
	t.Helper()
	def := domain.DefaultPolicy()
	return newWorldCfg(t, func(p *domain.Policy) {
		p.ResponsePeriodDays = def.ResponsePeriodDays
		p.DisputeWindowDays = def.DisputeWindowDays
		p.ReReviewWindowDays = def.ReReviewWindowDays
	}, func(c *service.Config) { c.Clock = zonedClock{inner: c.Clock, loc: loc} })
}

// advanceTo moves the fake clock forward to at exactly.
func (w *world) advanceTo(t *testing.T, at time.Time) {
	t.Helper()
	d := at.Sub(w.Clock.Now())
	if d < 0 {
		t.Fatalf("clock at %s cannot move back to %s", w.Clock.Now(), at)
	}
	w.Clock.Advance(d)
}

func days(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

func notificationBodies(t *testing.T, w *world, a domain.Actor, kind string) []string {
	t.Helper()
	var out []string
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, a, w.net.ID, service.NotificationQuery{})) {
		if n.Kind == kind {
			out = append(out, n.Body)
		}
	}
	return out
}

func wantErrMention(t *testing.T, err error, target *domain.Error, date string) {
	t.Helper()
	wantCode(t, err, target)
	if !strings.Contains(err.Error(), date) {
		t.Fatalf("refusal %q does not name the UTC date %s", err, date)
	}
}

var deadlineZones = []string{"UTC", "Europe/Berlin", "America/New_York", "Australia/Sydney"}

func TestDisputeDeadlinesAcrossDSTAndYearBoundary(t *testing.T) {
	cases := []struct {
		name   string
		opened time.Time // UTC instant the disputes are opened
	}{
		// Response window spans the US spring change (8 Mar 2026); the
		// re-review window spans the EU (29 Mar) and Sydney (5 Apr) changes.
		{"US spring forward", time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)},
		// Response window spans the EU spring change (29 Mar 2026, 01:00 UTC).
		{"EU spring forward", time.Date(2026, 3, 26, 0, 30, 0, 0, time.UTC)},
		// Response window spans the EU autumn change (25 Oct 2026); the
		// decision target spans the US autumn change (1 Nov 2026).
		{"EU fall back", time.Date(2026, 10, 20, 22, 30, 0, 0, time.UTC)},
		// Response window spans the US autumn change (1 Nov 2026, 06:00 UTC).
		{"US fall back", time.Date(2026, 10, 29, 5, 30, 0, 0, time.UTC)},
		// Decision target spans the end of 2026.
		{"decision target crosses new year", time.Date(2026, 12, 20, 12, 0, 0, 0, time.UTC)},
		// Response window spans the end of 2026.
		{"response deadline crosses new year", time.Date(2026, 12, 28, 12, 0, 0, 0, time.UTC)},
		// Opened in the last millisecond of 2026 UTC, when Sydney already reads 2027.
		{"last millisecond of the year", time.Date(2026, 12, 31, 23, 59, 59, int(999*time.Millisecond), time.UTC)},
	}
	for _, c := range cases {
		for _, zone := range deadlineZones {
			t.Run(c.name+"/"+zone, func(t *testing.T) {
				checkDisputeDeadlines(t, newDeadlineWorld(t, mustZone(t, zone)), c.opened)
			})
		}
	}
}

func checkDisputeDeadlines(t *testing.T, w *world, opened time.Time) {
	pol := w.net.Policy
	rev, _ := w.reviewer()
	w.advanceTo(t, opened)
	trA := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	trB := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	silent := tu.Must(w.open(w.a, trA.ID))   // the other party never responds
	answered := tu.Must(w.open(w.a, trB.ID)) // the other party responds in time
	if !silent.OpenedAt.Equal(opened) || silent.OpenedAt.Location() != time.UTC {
		t.Fatalf("opened at %s (%s), want %s UTC", silent.OpenedAt, silent.OpenedAt.Location(), opened)
	}

	respDeadline := opened.Add(days(pol.ResponsePeriodDays))
	target := respDeadline.Add(days(domain.DecisionTargetDays))
	v := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, silent.ID))
	dl := v.Deadlines
	if !dl.ResponseDeadline.Equal(respDeadline) || dl.ResponseDeadline.Sub(v.Dispute.OpenedAt) != days(pol.ResponsePeriodDays) {
		t.Fatalf("response deadline %s, want %s (exactly %d x 24h after opening)", dl.ResponseDeadline, respDeadline, pol.ResponsePeriodDays)
	}
	if !dl.DecisionTarget.Equal(target) || dl.DecisionTarget.Sub(dl.ResponseDeadline) != days(domain.DecisionTargetDays) {
		t.Fatalf("decision target %s, want %s", dl.DecisionTarget, target)
	}
	if dl.ResponseDeadline.Location() != time.UTC || dl.DecisionTarget.Location() != time.UTC {
		t.Fatalf("deadlines not in UTC: %s / %s", dl.ResponseDeadline.Location(), dl.DecisionTarget.Location())
	}

	// One millisecond before the response deadline: nothing is overdue, a
	// decision without a response still needs a waiver.
	w.advanceTo(t, respDeadline.Add(-time.Millisecond))
	if err := w.respond(w.b, answered.ID, "I delivered on time."); err != nil {
		t.Fatalf("response before the deadline: %v", err)
	}
	if dl := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, silent.ID)).Deadlines; dl.ResponseOverdue || dl.DecisionOverdue {
		t.Fatalf("overdue 1ms before the deadline: %+v", dl)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("%d reminder(s) before the response deadline", n)
	}
	_, err := w.resolve(rev, silent.ID, domain.OutcomeDismissed, 0, "no evidence")
	wantErrMention(t, err, domain.ErrInvalidState, respDeadline.Format("2006-01-02"))

	// One millisecond after: the silent case is overdue and reminded once, in
	// UTC dates; the answered case is not overdue.
	w.advanceTo(t, respDeadline.Add(time.Millisecond))
	if dl := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, silent.ID)).Deadlines; !dl.ResponseOverdue || dl.DecisionOverdue {
		t.Fatalf("1ms after the response deadline: %+v", dl)
	}
	if dl := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, answered.ID)).Deadlines; dl.ResponseOverdue {
		t.Fatalf("answered dispute reported without a response: %+v", dl)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("response reminders = %d, want 1 (the silent case only)", n)
	}
	bodies := notificationBodies(t, w, rev, service.NotifDisputeOverdue)
	if len(bodies) != 1 || !strings.Contains(bodies[0], respDeadline.Format("2 Jan 2006")) {
		t.Fatalf("reminder bodies %q do not name the UTC deadline %s", bodies, respDeadline.Format("2 Jan 2006"))
	}

	// Decision target, both sides.
	w.advanceTo(t, target.Add(-time.Millisecond))
	if dl := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, silent.ID)).Deadlines; dl.DecisionOverdue {
		t.Fatalf("decision overdue 1ms before the target: %+v", dl)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("%d reminder(s) before the decision target", n)
	}
	w.advanceTo(t, target.Add(time.Millisecond))
	if dl := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, answered.ID)).Deadlines; !dl.DecisionOverdue {
		t.Fatalf("decision not overdue 1ms after the target: %+v", dl)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 2 {
		t.Fatalf("decision reminders = %d, want 2", n)
	}

	// Both are decided now; the re-review (appeal) window is the decision
	// instant plus exactly ReReviewWindowDays x 24h, inclusive.
	dSilent := tu.Must(w.resolve(rev, silent.ID, domain.OutcomeDismissed, 0, "no evidence"))
	dAnswered := tu.Must(w.resolve(rev, answered.ID, domain.OutcomeDismissed, 0, "delivered"))
	if dSilent.DecidedAt == nil || !dSilent.DecidedAt.Equal(target.Add(time.Millisecond)) || dAnswered.DecidedAt == nil {
		t.Fatalf("decided at %v, want %s", dSilent.DecidedAt, target.Add(time.Millisecond))
	}
	rrCloses := dSilent.DecidedAt.Add(days(pol.ReReviewWindowDays))
	w.advanceTo(t, rrCloses)
	if err := w.rereview(w.a, silent.ID, "please look again"); err != nil {
		t.Fatalf("re-review at the last instant of the window: %v", err)
	}
	w.advanceTo(t, rrCloses.Add(time.Millisecond))
	wantErrMention(t, w.rereview(w.a, answered.ID, "please look again"), domain.ErrInvalidState, rrCloses.Format("2006-01-02"))
}

// FR-071: the raise window is CompletedAt plus exactly DisputeWindowDays x 24h,
// inclusive, across daylight-saving changes and the year end.
func TestDisputeRaiseWindowAcrossDSTAndYearBoundary(t *testing.T) {
	cases := []struct {
		name      string
		completed time.Time
	}{
		{"spans EU and US spring changes", time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)},
		{"spans EU and US autumn changes", time.Date(2026, 10, 10, 1, 30, 0, 0, time.UTC)},
		{"spans the year end", time.Date(2026, 12, 15, 23, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		for _, zone := range deadlineZones {
			t.Run(c.name+"/"+zone, func(t *testing.T) {
				w := newDeadlineWorld(t, mustZone(t, zone))
				w.advanceTo(t, c.completed)
				trA := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
				trB := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
				if trA.CompletedAt == nil || !trA.CompletedAt.Equal(c.completed) {
					t.Fatalf("completed at %v, want %s", trA.CompletedAt, c.completed)
				}
				closes := c.completed.Add(days(w.net.Policy.DisputeWindowDays))
				w.advanceTo(t, closes)
				if _, err := w.open(w.a, trA.ID); err != nil {
					t.Fatalf("dispute at the last instant of the window: %v", err)
				}
				w.advanceTo(t, closes.Add(time.Millisecond))
				_, err := w.open(w.a, trB.ID)
				wantErrMention(t, err, domain.ErrInvalidState, closes.Format("2006-01-02"))
				if st := w.txStatus(trB.ID).Status; st != domain.TxCompleted {
					t.Fatalf("refused dispute changed the transaction to %s", st)
				}
			})
		}
	}
}
