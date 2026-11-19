package service_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-026 fixtures: small thresholds keep the bursts short.
func lowSignalThresholds(p *domain.Policy) {
	p.ActivitySignalsEnabled = true
	p.VelocitySignalProposals, p.VelocitySignalWindowMinutes = 3, 60
	p.ReciprocalSignalTrades, p.ReciprocalSignalWindowDays = 3, 30
}

// atPeriodStart moves the clock to the start of the next signal period.
func (w *world) atPeriodStart(t *testing.T, window time.Duration) time.Time {
	t.Helper()
	at := domain.SignalPeriodStart(w.Clock.Now(), window).Add(window)
	w.advanceTo(t, at)
	return at
}

func (w *world) proposeSmall(t *testing.T, by domain.Actor, to *domain.Member) *domain.Transaction {
	t.Helper()
	return tu.Must(w.propose(by, to, service.ProposerPays, 1))
}

// signalRows reads the stored signals directly, oldest first.
func signalRows(t *testing.T, e *tu.Env, netID string) []store.ActivitySignal {
	t.Helper()
	var out []store.ActivitySignal
	rows := tu.Must(e.Store.DB().Query(`SELECT id, kind, member_id, COALESCE(other_member_id,''), period_start, observed, threshold
		FROM activity_signals WHERE network_id = ? ORDER BY id`, netID))
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var s store.ActivitySignal
		var period int64
		tu.Must(0, rows.Scan(&s.ID, &s.Kind, &s.MemberID, &s.OtherMemberID, &period, &s.Observed, &s.Threshold))
		s.PeriodStart = time.UnixMilli(period).UTC()
		out = append(out, s)
	}
	return out
}

func signalNotices(t *testing.T, e *tu.Env, a domain.Actor, netID string) int {
	t.Helper()
	ns := tu.Must(e.Svc.ListNotifications(e.Ctx, a, netID, service.NotificationQuery{Page: service.Page{Limit: 500}}))
	n := 0
	for _, x := range ns {
		if x.Kind == service.NotifActivitySignal {
			n++
		}
	}
	return n
}

// SEC-026 acceptance 1: a seeded burst above the velocity threshold creates
// one signal per window (injected clock). The proposals themselves all
// succeed; the administrator is told once and the members are not told.
func TestSEC026VelocityBurstRaisesOneSignalPerWindow(t *testing.T) {
	w := newWorld(t, lowSignalThresholds)
	start := w.atPeriodStart(t, time.Hour)
	for i := 0; i < 10; i++ { // 10 proposals in ten minutes, threshold "more than 3"
		w.proposeSmall(t, w.a, w.bM)
		w.Clock.Advance(time.Minute)
	}
	got := signalRows(t, w.Env, w.net.ID)
	if len(got) != 1 {
		t.Fatalf("burst raised %d signals, want 1: %+v", len(got), got)
	}
	s := got[0]
	if s.Kind != domain.SignalVelocity || s.MemberID != w.aM.ID || s.OtherMemberID != "" || s.Observed != 4 || s.Threshold != 3 ||
		!s.PeriodStart.Equal(start) {
		t.Fatalf("signal %+v, want velocity for ana, observed 4 of threshold 3 in the period from %s", s, start)
	}
	if n := signalNotices(t, w.Env, w.admin, w.net.ID); n != 1 {
		t.Errorf("administrator received %d signal notices, want 1", n)
	}
	for _, m := range []domain.Actor{w.a, w.b} {
		if n := signalNotices(t, w.Env, m, w.net.ID); n != 0 {
			t.Errorf("a member received %d signal notices", n)
		}
	}
	// The next period: the sliding count still exceeds the threshold, so the
	// continuing burst raises exactly one more signal there.
	next := w.atPeriodStart(t, time.Hour)
	for i := 0; i < 5; i++ {
		w.proposeSmall(t, w.a, w.bM)
		w.Clock.Advance(time.Minute)
	}
	got = signalRows(t, w.Env, w.net.ID)
	if len(got) != 2 || !got[1].PeriodStart.Equal(next) {
		t.Fatalf("after the next period: %+v, want a second signal for %s", got, next)
	}
	if n := signalNotices(t, w.Env, w.admin, w.net.ID); n != 2 {
		t.Errorf("administrator received %d signal notices, want 2", n)
	}
	// Signals never block: every proposal is open and counted.
	if n := countRows(t, w.Env, "transactions"); n < 15 {
		t.Errorf("only %d transactions recorded", n)
	}
}

// SEC-026: window and period boundaries with the injected clock. The count
// covers the window that ends at the triggering proposal and excludes an
// event exactly one window earlier; a new period starts exactly on its boundary.
func TestSEC026VelocityClockAtWindowBoundary(t *testing.T) {
	t.Run("just inside the window", func(t *testing.T) {
		w := newWorld(t, lowSignalThresholds)
		start := w.atPeriodStart(t, time.Hour)
		for i := 0; i < 3; i++ {
			w.proposeSmall(t, w.a, w.bM)
		}
		w.advanceTo(t, start.Add(time.Hour-time.Millisecond))
		w.proposeSmall(t, w.a, w.bM)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 || got[0].Observed != 4 || !got[0].PeriodStart.Equal(start) {
			t.Fatalf("signals %+v, want one for 4 proposals in the period from %s", got, start)
		}
	})
	t.Run("exactly one window later", func(t *testing.T) {
		w := newWorld(t, lowSignalThresholds)
		start := w.atPeriodStart(t, time.Hour)
		for i := 0; i < 3; i++ {
			w.proposeSmall(t, w.a, w.bM)
		}
		w.advanceTo(t, start.Add(time.Hour))
		w.proposeSmall(t, w.a, w.bM)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
			t.Fatalf("proposals exactly one window apart raised %+v", got)
		}
	})
	t.Run("period boundary", func(t *testing.T) {
		w := newWorld(t, lowSignalThresholds)
		start := w.atPeriodStart(t, time.Hour)
		w.advanceTo(t, start.Add(time.Hour-4*time.Millisecond))
		for i := 0; i < 4; i++ {
			w.proposeSmall(t, w.a, w.bM)
			w.Clock.Advance(time.Millisecond)
		}
		if now := w.Clock.Now(); !now.Equal(start.Add(time.Hour)) {
			t.Fatalf("clock at %s", now)
		}
		w.proposeSmall(t, w.a, w.bM) // first instant of the next period
		w.Clock.Advance(time.Millisecond)
		w.proposeSmall(t, w.a, w.bM)
		got := signalRows(t, w.Env, w.net.ID)
		if len(got) != 2 || !got[0].PeriodStart.Equal(start) || !got[1].PeriodStart.Equal(start.Add(time.Hour)) || got[1].Observed != 5 {
			t.Fatalf("signals %+v, want one in each period, the second at the boundary with 5 proposals", got)
		}
	})
}

// SEC-026: windows end at the triggering event. Events recorded under a
// later wall-clock timestamp must not create a false signal after rollback.
func TestSEC026ClockRollbackExcludesFutureEvents(t *testing.T) {
	t.Run("velocity", func(t *testing.T) {
		w := newWorld(t, lowSignalThresholds)
		w.atPeriodStart(t, time.Hour)
		w.Clock.Advance(30 * time.Minute)
		for i := 0; i < 3; i++ {
			w.proposeSmall(t, w.a, w.bM)
		}
		w.Clock.Advance(-time.Minute)
		w.proposeSmall(t, w.a, w.bM)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
			t.Fatalf("future proposals raised a signal after clock rollback: %+v", got)
		}
		w.Clock.Advance(time.Minute)
		w.proposeSmall(t, w.a, w.bM)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 || got[0].Observed != 5 {
			t.Fatalf("events at the restored instant were not counted: %+v", got)
		}
	})
	t.Run("reciprocal", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) {
			lowSignalThresholds(p)
			p.VelocitySignalProposals = 1000
		})
		w.atPeriodStart(t, 30*24*time.Hour)
		w.Clock.Advance(time.Hour)
		w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10)
		w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10)
		w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 10)
		w.Clock.Advance(-time.Minute)
		w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
			t.Fatalf("future completions raised a signal after clock rollback: %+v", got)
		}
		w.Clock.Advance(time.Minute)
		w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 10)
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 || got[0].Kind != domain.SignalReciprocal || got[0].Observed != 5 {
			t.Fatalf("completions at the restored instant were not counted: %+v", got)
		}
		if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
			t.Fatalf("clock rollback changed ledger consistency: %+v", rep.Findings)
		}
	})
}

// SEC-026: a network that switched signals off raises none, and members of
// other networks are counted only in their own network.
func TestSEC026DisabledPolicyRaisesNothing(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { lowSignalThresholds(p); p.ActivitySignalsEnabled = false })
	for i := 0; i < 10; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	for i := 0; i < 3; i++ {
		w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1)
		w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 1)
	}
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
		t.Fatalf("disabled policy raised %+v", got)
	}
	if n := signalNotices(t, w.Env, w.admin, w.net.ID); n != 0 {
		t.Fatalf("disabled policy sent %d notices", n)
	}
}

// SEC-026: concurrent proposals cannot record two signals for one period:
// the database uniqueness decides inside each write unit.
func TestSEC026ConcurrentProposalsRaiseOneSignal(t *testing.T) {
	w := newWorld(t, lowSignalThresholds)
	w.atPeriodStart(t, time.Hour)
	const n = 12
	var wg sync.WaitGroup
	errs := make(chan error, n)
	keys := make([]string, n)
	for i := range keys {
		keys[i] = w.Key()
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID,
				Direction: service.ProposerPays, Amount: 1, IdempotencyKey: key})
			errs <- err
		}(keys[i])
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("a concurrent proposal failed: %v", err)
		}
	}
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 {
		t.Fatalf("%d concurrent proposals raised %d signals, want 1", n, len(got))
	}
	if c := signalNotices(t, w.Env, w.admin, w.net.ID); c != 1 {
		t.Fatalf("administrator received %d notices, want 1", c)
	}
}

// ledgerFingerprint describes the ledger without record ids: every entry in
// posting order with its member and amount, and every member's balance.
func ledgerFingerprint(t *testing.T, e *tu.Env, netID string) string {
	t.Helper()
	h := sha256.New()
	rows := tu.Must(e.Store.DB().Query(`SELECT e.ledger_seq, COALESCE(m.display_name, 'system'), e.amount
		FROM entries e JOIN accounts a ON a.id = e.account_id LEFT JOIN members m ON m.id = a.member_id
		WHERE e.network_id = ? ORDER BY e.ledger_seq, e.amount`, netID))
	for rows.Next() {
		var seq, amt int64
		var who string
		tu.Must(0, rows.Scan(&seq, &who, &amt))
		fmt.Fprintf(h, "%d|%s|%d\n", seq, who, amt)
	}
	_ = rows.Close()
	rows = tu.Must(e.Store.DB().Query(`SELECT COALESCE(m.display_name, 'system'), a.balance FROM accounts a
		LEFT JOIN members m ON m.id = a.member_id WHERE a.network_id = ? ORDER BY 1`, netID))
	for rows.Next() {
		var who string
		var bal int64
		tu.Must(0, rows.Scan(&who, &bal))
		fmt.Fprintf(h, "%s=%d\n", who, bal)
	}
	_ = rows.Close()
	return fmt.Sprintf("%x", h.Sum(nil))
}

func entryRows(t *testing.T, e *tu.Env, netID string) map[string]string {
	t.Helper()
	out := map[string]string{}
	rows := tu.Must(e.Store.DB().Query(`SELECT id, account_id, tx_id, amount, ledger_seq FROM entries WHERE network_id = ?`, netID))
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, acct, tx string
		var amt, seq int64
		tu.Must(0, rows.Scan(&id, &acct, &tx, &amt, &seq))
		out[id] = fmt.Sprintf("%s|%s|%d|%d", acct, tx, amt, seq)
	}
	return out
}

// SEC-026 acceptance 2: a reciprocal pattern above the threshold creates a
// signal and does not change any ledger entry. One-way trading, however
// frequent, is not a reciprocal pattern.
func TestSEC026ReciprocalPatternRaisesSignalWithoutLedgerChange(t *testing.T) {
	run := func(enabled bool) (*world, string) {
		w := newWorld(t, func(p *domain.Policy) { lowSignalThresholds(p); p.ActivitySignalsEnabled = enabled })
		for i := 0; i < 4; i++ { // one direction only: never a reciprocal pattern
			w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
			w.Clock.Advance(time.Hour)
		}
		if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
			t.Fatalf("one-way trades raised %+v", got)
		}
		before := entryRows(t, w.Env, w.net.ID)
		trade := w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 40) // the return trade makes the pattern
		after := entryRows(t, w.Env, w.net.ID)
		for id, row := range before {
			if after[id] != row {
				t.Errorf("entry %s changed: %q -> %q", id, row, after[id])
			}
		}
		if len(after) != len(before)+2 {
			t.Errorf("the return trade added %d entries, want its own 2", len(after)-len(before))
		}
		if trade.Status != domain.TxCompleted {
			t.Errorf("trade status %s", trade.Status)
		}
		return w, ledgerFingerprint(t, w.Env, w.net.ID)
	}
	w, withSignals := run(true)
	got := signalRows(t, w.Env, w.net.ID)
	if len(got) != 1 {
		t.Fatalf("reciprocal pattern raised %d signals, want 1", len(got))
	}
	lo, hi := w.aM.ID, w.bM.ID
	if hi < lo {
		lo, hi = hi, lo
	}
	if s := got[0]; s.Kind != domain.SignalReciprocal || s.MemberID != lo || s.OtherMemberID != hi || s.Observed != 5 || s.Threshold != 3 {
		t.Fatalf("signal %+v, want reciprocal for the ordered pair with 5 of threshold 3", s)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("ledger verification failed after a signal: %v", rep.Findings)
	}
	// More trades in the same period add no second signal.
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1)
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 {
		t.Fatalf("same period raised %d signals", len(got))
	}
	// The ledger is the same as in an identical network with signals off.
	_, without := run(false)
	if withSignals != without {
		t.Fatal("ledger entries or balances differ from the same trades with signals off")
	}
}

// SEC-026 acceptance 3: signals are visible only to administrators and
// dispute reviewers of the same network, under their current roles and MFA
// rule. Plain members are refused, other networks' administrators get
// not-found, and a member never sees a signal about themselves.
func TestSEC026SignalsVisibleOnlyToSameNetworkReviewers(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	_, rc := enrol(t, e, w.admin)
	w.net, w.adminM = e.Network(w.admin, "town", lowSignalThresholds)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	rev, revM := e.Member(w.admin, w.net, "rev@example.org")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, revM.ID, domain.RoleReviewer))
	gone, goneM := e.Member(w.admin, w.net, "gone@example.org")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, goneM.ID, domain.RoleReviewer))
	tu.Must(0, e.Svc.SuspendMember(e.Ctx, w.admin, w.net.ID, goneM.ID, "fixture"))
	// ben is also a reviewer, and the subject of a signal.
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleReviewer))
	fran := e.Person("fran@example.org")
	enrol(t, e, fran)
	other, _ := e.Network(fran, "other", lowSignalThresholds)
	_, oaM := e.Member(fran, other, "oa@example.org")
	oa := domain.Actor{PersonID: oaM.PersonID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	_, obM := e.Member(fran, other, "ob@example.org")
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 4; i++ {
		w.proposeSmall(t, w.b, w.aM)
		tu.Must(e.Svc.ProposeTransaction(e.Ctx, oa, other.ID, service.ProposeInput{CounterpartyID: obM.ID,
			Direction: service.ProposerPays, Amount: 1, IdempotencyKey: e.Key()}))
	}
	if len(signalRows(t, e, w.net.ID)) != 1 || len(signalRows(t, e, other.ID)) != 1 {
		t.Fatal("fixture: want one signal in each network")
	}
	list := func(a domain.Actor) ([]service.ActivitySignalView, error) {
		return e.Svc.ListActivitySignals(e.Ctx, a, w.net.ID, service.ActivitySignalQuery{})
	}
	readOnly := w.admin
	readOnly.Scopes = []domain.Scope{domain.ScopeRead}
	for name, a := range map[string]domain.Actor{"administrator": w.admin, "reviewer": rev, "read-only token": readOnly} {
		got, err := list(a)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got) != 1 || got[0].MemberID != w.bM.ID || got[0].NetworkID != w.net.ID || got[0].MemberName == "" {
			t.Fatalf("%s sees %+v, want this network's one signal with the member's name", name, got)
		}
	}
	// The subject (a reviewer) does not see the signal about themselves.
	if got, err := list(w.b); err != nil || len(got) != 0 {
		t.Fatalf("subject reviewer sees %+v, %v", got, err)
	}
	for name, a := range map[string]domain.Actor{"plain member": w.a, "suspended reviewer": gone} {
		if _, err := list(a); domain.CodeOf(err) != domain.CodeForbidden {
			t.Errorf("%s: %v, want forbidden", name, err)
		}
	}
	scoped := w.admin
	scoped.NetworkScope = other.ID
	for name, a := range map[string]domain.Actor{"other network's administrator": fran, "token scoped to another network": scoped,
		"person of no network": e.Person("nobody@example.org")} {
		if _, err := list(a); domain.CodeOf(err) != domain.CodeNotFound {
			t.Errorf("%s: %v, want not-found", name, err)
		}
	}
	if got := tu.Must(e.Svc.ListActivitySignals(e.Ctx, fran, other.ID, service.ActivitySignalQuery{})); len(got) != 1 || got[0].NetworkID != other.ID {
		t.Fatalf("other network's administrator sees %+v", got)
	}
	if _, err := e.Svc.ListActivitySignals(e.Ctx, w.admin, w.net.ID, service.ActivitySignalQuery{Kind: "bogus"}); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("unknown kind: %v", err)
	}
	if _, err := e.Svc.ListActivitySignals(e.Ctx, w.admin, w.net.ID, service.ActivitySignalQuery{Page: service.Page{AfterID: "x' OR 1=1"}}); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("malformed cursor: %v", err)
	}
	// The administrator MFA rule applies as to other administrator reads.
	tu.Must(0, e.Svc.DisableMFA(e.Ctx, w.admin, pw, rc[0]))
	if _, err := list(w.admin); domain.CodeOf(err) != domain.CodeForbidden || !strings.Contains(err.Error(), "mfa-enrolment-required") {
		t.Fatalf("administrator without MFA: %v", err)
	}
	// The members concerned see nothing of it in their own audit view.
	for _, a := range []domain.Actor{w.a, w.b} {
		evs := tu.Must(e.Svc.ListAudit(e.Ctx, a, w.net.ID, service.AuditQuery{Limit: 500}))
		for _, ev := range evs {
			if ev.Action == domain.ActActivitySignalRaised {
				t.Fatal("a member's own audit view shows the signal")
			}
		}
	}
}

// SEC-026: every eligible administrator and reviewer is told, also beyond the
// first page of members, exactly once; the subject, plain members, inactive
// reviewers and other networks are not.
func TestSEC026SignalNoticesReachEveryReviewer(t *testing.T) {
	w := newWorld(t, lowSignalThresholds)
	rev, revM := w.Member(w.admin, w.net, "rev@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, revM.ID, domain.RoleReviewer))
	idle, idleM := w.Member(w.admin, w.net, "idle@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, idleM.ID, domain.RoleReviewer))
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, idleM.ID, "fixture"))
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin)) // the subject is an administrator too
	foreign := w.Person("foreign@example.org")
	foreignNet, _ := w.Network(foreign, "other-network", nil)
	var early domain.Actor
	// Valid ids that sort before every fixture member fill the first page.
	err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
		for i := 0; i < 501; i++ {
			pid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i)
			mid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i+1)
			if err := tx.InsertPerson(&domain.Person{ID: pid, Login: fmt.Sprintf("filler%d@example.org", i), CreatedAt: w.Clock.Now()}, "unused-fixture"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: mid, NetworkID: w.net.ID, PersonID: pid,
				DisplayName: "Filler", Status: domain.MemberActive, JoinMethod: "invitation", JoinedAt: w.Clock.Now()}); err != nil {
				return err
			}
			if i == 0 {
				early = domain.Actor{PersonID: pid, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
				if err := tx.GrantRole("00000000-0000-4000-9000-000000000001", w.net.ID, mid, domain.RoleReviewer, w.adminM.ID, w.Clock.Now()); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 4; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	cases := []struct {
		name  string
		actor domain.Actor
		want  int
	}{
		{"founder administrator after the first page", w.admin, 1},
		{"reviewer", rev, 1},
		{"reviewer on the first page", early, 1},
		{"subject administrator", w.a, 0},
		{"plain member", w.b, 0},
		{"suspended reviewer", idle, 0},
	}
	for _, c := range cases {
		if got := signalNotices(t, w.Env, c.actor, w.net.ID); got != c.want {
			t.Errorf("%s received %d notices, want %d", c.name, got, c.want)
		}
	}
	ns := tu.Must(w.Svc.ListNotifications(w.Ctx, w.admin, w.net.ID, service.NotificationQuery{Page: service.Page{Limit: 500}}))
	for _, n := range ns {
		if n.Kind == service.NotifActivitySignal {
			if strings.Contains(n.Title+n.Body, "ana") || strings.Contains(n.Title+n.Body, w.aM.DisplayName) || n.Link != "/review/activity-signals" {
				t.Errorf("notice reveals more than needed or links elsewhere: %+v", n)
			}
		}
	}
	if got := signalNotices(t, w.Env, foreign, foreignNet.ID); got != 0 {
		t.Errorf("administrator of another network received %d notices", got)
	}
}

// SEC-026: a failing signal evaluation never blocks, fails or rolls back the
// proposal or the trade. Its partial writes disappear with its savepoint and
// the failure is reported after the trade commits.
func TestSEC026SignalFailureDoesNotBlockTrade(t *testing.T) {
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	type failure struct {
		netID, kind string
		err         error
	}
	var mu sync.Mutex
	var reported []failure
	e.Svc.SetSignalFailureHandler(func(netID, kind string, err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, failure{netID, kind, err})
	})
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", lowSignalThresholds)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 3; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	auditBefore := countRows(t, e, "audit_events")
	notesBefore := countRows(t, e, "notifications")
	for _, op := range []string{"CountProposalsSince", "InsertActivitySignal", "InsertNotification#2"} {
		name, nth := op, 1
		if i := strings.IndexByte(op, '#'); i > 0 {
			name, nth = op[:i], 2 // the first notification is the counterparty's own trade notice
		}
		fs.Arm(name, nth)
		tx, err := w.propose(w.a, w.bM, service.ProposerPays, 1)
		fs.Disarm()
		if err != nil {
			t.Fatalf("%s: proposal failed: %v", op, err)
		}
		if v, err := w.Svc.GetTransaction(w.Ctx, w.b, w.net.ID, tx.ID); err != nil || v.Tx.Status != domain.TxProposed {
			t.Fatalf("%s: proposal not committed: %v", op, err)
		}
		if got := signalRows(t, e, w.net.ID); len(got) != 0 {
			t.Fatalf("%s: a failed evaluation left %+v", op, got)
		}
		// The trade wrote its proposal audit event and the counterparty's notice; nothing of the signal remains.
		auditBefore++
		notesBefore++
		if a, n := countRows(t, e, "audit_events"), countRows(t, e, "notifications"); a != auditBefore || n != notesBefore {
			t.Fatalf("%s: audit %d (want %d), notifications %d (want %d)", op, a, auditBefore, n, notesBefore)
		}
	}
	mu.Lock()
	if len(reported) != 3 {
		t.Fatalf("reported %d failures, want 3: %+v", len(reported), reported)
	}
	for _, r := range reported {
		if r.netID != w.net.ID || r.kind != domain.SignalVelocity || !errors.Is(r.err, tu.ErrInjected) {
			t.Errorf("reported %+v", r)
		}
	}
	reported = nil
	mu.Unlock()
	// A unit that fails after the evaluation rolls back and reports nothing.
	fs.Arm("InsertActivitySignal", 1)
	fs.Arm("PutIdem", 1)
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 1); !errors.Is(err, tu.ErrInjected) {
		t.Fatalf("idempotency failure: %v", err)
	}
	fs.Disarm()
	mu.Lock()
	if len(reported) != 0 {
		t.Fatalf("a rolled-back unit reported %+v", reported)
	}
	mu.Unlock()
	// Reciprocal evaluation failing at completion: the trade still posts.
	w2 := newWorldWith(t, lowSignalThresholds)
	for i := 0; i < 3; i++ {
		w2.Trade(w2.net.ID, w2.a, w2.b, w2.aM, w2.bM, 10)
	}
	w2.fs.Arm("CountCompletedTradesBetween", 1)
	trade := w2.Trade(w2.net.ID, w2.b, w2.a, w2.bM, w2.aM, 10)
	w2.fs.Disarm()
	if trade.Status != domain.TxCompleted || trade.LedgerSeq == 0 {
		t.Fatalf("trade not posted: %+v", trade)
	}
	if got := signalRows(t, w2.Env, w2.net.ID); len(got) != 0 {
		t.Fatalf("failed reciprocal evaluation left %+v", got)
	}
	// The same evaluation succeeds on the next completion.
	w2.Trade(w2.net.ID, w2.b, w2.a, w2.bM, w2.aM, 10)
	if got := signalRows(t, w2.Env, w2.net.ID); len(got) != 1 || got[0].Kind != domain.SignalReciprocal {
		t.Fatalf("after recovery: %+v", got)
	}
}

type faultWorld struct {
	*world
	fs *tu.FaultStore
}

func newWorldWith(t *testing.T, mod func(*domain.Policy)) *faultWorld {
	t.Helper()
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", mod)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	return &faultWorld{world: w, fs: fs}
}

// SEC-026 / GOV-002 / GOV-005: threshold changes are policy changes: audited
// with before and after values, refused out of bounds with no version or
// audit event, applied prospectively from their effective time, and shown
// in the members' rules log (GOV-002).
func TestSEC026ThresholdChangesFollowPolicyRules(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ActivitySignalsEnabled = true }) // explicit activation, default thresholds
	change := func(mod func(*domain.Policy), notice *int) (*domain.Network, error) {
		pol := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).Policy
		mod(&pol)
		return w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "signal review",
			NoticeDays: notice, IdempotencyKey: w.Key()})
	}
	versions := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).PolicyVersion
	audits := countRows(t, w.Env, "audit_events")
	for _, bad := range []func(*domain.Policy){
		func(p *domain.Policy) { p.VelocitySignalProposals = 0 },
		func(p *domain.Policy) { p.VelocitySignalWindowMinutes = domain.MaxVelocityWindowMinutes + 1 },
		func(p *domain.Policy) { p.ReciprocalSignalTrades = domain.MaxSignalThreshold + 1 },
		func(p *domain.Policy) { p.ReciprocalSignalWindowDays = 0 },
	} {
		if _, err := change(bad, nil); domain.CodeOf(err) != domain.CodeValidation {
			t.Fatalf("out-of-bounds threshold: %v", err)
		}
	}
	if v := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).PolicyVersion; v != versions || countRows(t, w.Env, "audit_events") != audits {
		t.Fatal("a refused change wrote a version or an audit event")
	}
	// Scheduled with one day's notice: the old threshold stays in force until then.
	one := 1
	tu.Must(change(func(p *domain.Policy) { p.VelocitySignalProposals = 3 }, &one))
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 5; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 0 {
		t.Fatalf("a scheduled threshold applied early: %+v", got)
	}
	w.Clock.Advance(25 * time.Hour)
	for i := 0; i < 4; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 || got[0].Threshold != 3 {
		t.Fatalf("after the effective time: %+v, want one signal under threshold 3", got)
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActPolicyChanged, Limit: 50}))
	var found bool
	for _, ev := range evs {
		if strings.Contains(ev.SummaryBefore, `"velocity_signal_proposals":20`) && strings.Contains(ev.SummaryAfter, `"velocity_signal_proposals":3`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("no policy.changed event with the threshold's before and after values: %+v", evs)
	}
	for _, a := range []domain.Actor{w.a, w.admin} {
		for _, entries := range [][]service.PolicyLogEntry{
			tu.Must(w.Svc.PolicyLog(w.Ctx, a, w.net.ID)),
			tu.Must(w.Svc.PolicyLogPage(w.Ctx, a, w.net.ID, service.Page{Limit: 50})),
		} {
			shown := false
			for _, e := range entries {
				if e.Changes["velocity_signal_proposals"] == 3 && e.Previous["velocity_signal_proposals"] == 20 {
					shown = true
				}
			}
			if !shown {
				t.Fatal("member policy log omitted the previous and new signal threshold")
			}
		}
	}
	// Switching signals off applies at once and is audited too.
	tu.Must(change(func(p *domain.Policy) { p.ActivitySignalsEnabled = false }, nil))
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 5; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	if got := signalRows(t, w.Env, w.net.ID); len(got) != 1 {
		t.Fatalf("disabled signals still raised: %+v", got)
	}
}

// SEC-026 abuse case: switching signals off or loosening them sends an inbox
// notice to other administrators and reviewers; members can read the rules log.
func TestSEC026SettingChangesNotifyOtherReviewers(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ActivitySignalsEnabled = true })
	admin2, admin2M := w.Member(w.admin, w.net, "second@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, admin2M.ID, domain.RoleAdmin))
	rev, revM := w.Member(w.admin, w.net, "rev@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, revM.ID, domain.RoleReviewer))
	notices := func(a domain.Actor) int {
		n := 0
		for _, x := range tu.Must(w.Svc.ListNotifications(w.Ctx, a, w.net.ID, service.NotificationQuery{Page: service.Page{Limit: 500}})) {
			if x.Kind == service.NotifActivitySignalSettings {
				n++
			}
		}
		return n
	}
	update := func(mod func(*domain.Policy)) {
		pol := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).Policy
		mod(&pol)
		tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "r", IdempotencyKey: w.Key()}))
	}
	update(func(p *domain.Policy) { p.LimitWarningPercent = 70 }) // not a signal setting
	update(func(p *domain.Policy) { p.ActivitySignalsEnabled = false })
	update(func(p *domain.Policy) { p.ReciprocalSignalTrades = 500 })
	for name, c := range map[string]struct {
		a    domain.Actor
		want int
	}{"other administrator": {admin2, 2}, "reviewer": {rev, 2}, "the administrator who changed it": {w.admin, 0}, "member": {w.a, 0}} {
		if got := notices(c.a); got != c.want {
			t.Errorf("%s received %d settings notices, want %d", name, got, c.want)
		}
	}
}

// SEC-026 / SEC-018: erasing a departed member keeps the signal rows, which
// hold ids and counts only, and the queue then shows the anonymised name.
func TestSEC026ErasureKeepsSignalIDsOnly(t *testing.T) {
	w := newWorld(t, lowSignalThresholds)
	tu.Must(0, func() error {
		_, err := w.Store.DB().Exec(`UPDATE members SET display_name = ? WHERE id = ?`, anaName, w.aM.ID)
		return err
	}())
	w.atPeriodStart(t, time.Hour)
	for i := 0; i < 4; i++ {
		w.proposeSmall(t, w.a, w.bM)
	}
	before := signalRows(t, w.Env, w.net.ID)
	if len(before) != 1 {
		t.Fatalf("fixture signals %+v", before)
	}
	if got := tu.Must(w.Svc.ListActivitySignals(w.Ctx, w.admin, w.net.ID, service.ActivitySignalQuery{})); got[0].MemberName != anaName {
		t.Fatalf("queue name %q", got[0].MemberName)
	}
	w.depart(w.admin, w.aM)
	tu.Must(w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "requested"))
	after := signalRows(t, w.Env, w.net.ID)
	if len(after) != 1 || after[0] != before[0] {
		t.Fatalf("erasure changed signals: %+v -> %+v", before, after)
	}
	got := tu.Must(w.Svc.ListActivitySignals(w.Ctx, w.admin, w.net.ID, service.ActivitySignalQuery{}))
	if len(got) != 1 || !strings.HasPrefix(got[0].MemberName, "Former member") {
		t.Fatalf("after erasure the queue shows %+v", got)
	}
	if hits := dbMatches(t, w.Store.DB(), anaName); len(hits) != 0 {
		t.Fatalf("erased name still stored in %v", hits)
	}
}
