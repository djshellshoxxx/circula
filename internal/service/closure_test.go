package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

var sysJob = domain.SystemActor("job-test")

func (w *world) schedule(reason string, days int) (*domain.Closure, error) {
	return w.Svc.ScheduleClosure(w.Ctx, w.admin, w.net.ID, service.ClosureInput{Reason: reason, NoticeDays: days})
}

func (w *world) netNow() *domain.Network {
	return tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
}

// FR-068 / GOV-025 a: unresolved disputes block the announcement; proposals do not.
func TestClosureBlockersPreviewAndNotice(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	open := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	d := tu.Must(w.open(w.a, tx.ID))

	pv := tu.Must(w.Svc.PreviewClosure(w.Ctx, w.admin, w.net.ID))
	if len(pv.Blockers) != 1 || pv.NonZeroBalances != 2 || pv.OpenProposals != 1 {
		t.Fatalf("preview = %+v", pv)
	}
	if _, err := w.Svc.PreviewClosure(w.Ctx, w.a, w.net.ID); err == nil {
		t.Fatal("member previewed a closure")
	}
	_, err := w.schedule("end", 0)
	wantCode(t, err, domain.ErrBlocked)

	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d.ID))
	_, err = w.schedule("end", 29) // DEC-025: 30 days minimum
	wantCode(t, err, domain.ErrValidation)
	c := tu.Must(w.schedule("end", 0))
	if c.NoticeDays != 30 || !c.ClosingAt.Equal(w.Clock.Now().AddDate(0, 0, 30)) {
		t.Fatalf("closure = %+v", c)
	}
	if !hasKind(w.inbox(w.b), service.NotifClosureScheduled) {
		t.Fatal("member not notified of the closure date")
	}
	_, err = w.schedule("again", 0)
	wantCode(t, err, domain.ErrInvalidState)
	if !hasKind(auditActions(w), service.ActClosureScheduled) {
		t.Fatal("scheduling not audited")
	}

	// Until the date trading continues.
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 50))

	w.Clock.Advance(30*24*time.Hour + time.Hour)
	_, err = w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState) // frozen at the date, with no job having run
	if w.netNow().Status != domain.NetworkClosing {
		t.Fatalf("status = %s", w.netNow().Status)
	}
	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("read while closing: %v", err)
	}
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("finalised %d", n)
	}
	if got := w.txStatus(open.ID).Status; got != domain.TxCancelled {
		t.Fatalf("open proposal is %s", got) // GOV-025 c
	}
	if !hasKind(w.inbox(w.b), service.NotifNetworkClosed) {
		t.Fatal("member not notified of closure")
	}
	_, err = w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("second run finalised %d", n)
	}
}

// GOV-025 e: the final export checksum matches the recomputed value; the
// reconciliation summary is zero.
func TestClosureFinalExportChecksum(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("end", 0))
	w.Clock.Advance(31 * 24 * time.Hour)
	tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
	c := w.netNow().Closure
	if c == nil || !c.Done() || len(c.FinalSHA256) != 64 || c.FinalEntriesSum != 0 || c.FinalBalanceTotal != 0 {
		t.Fatalf("closure = %+v", c)
	}
	chk := tu.Must(w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID))
	if !chk.Match || chk.Stored != c.FinalSHA256 {
		t.Fatalf("check = %+v", chk)
	}
	if _, err := w.Svc.VerifyClosureExport(w.Ctx, w.a, w.net.ID); err == nil {
		t.Fatal("member verified the closure export")
	}
	// Read and export access continue during the period.
	tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportFull, service.FormatJSON, &service.MemSink{}))
	tu.Must(w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, service.FormatJSON, &service.MemSink{}))
	if !hasKind(auditActions(w), service.ActClosureExported) {
		t.Fatal("final export not audited")
	}
}

// GOV-025: the admin can cancel before the date, not after it.
func TestClosureCancel(t *testing.T) {
	w := newWorld(t, nil)
	tu.Must(w.schedule("end", 40))
	wantCode(t, w.Svc.CancelClosure(w.Ctx, w.admin, w.net.ID, ""), domain.ErrValidation)
	wantCode(t, w.Svc.CancelClosure(w.Ctx, w.a, w.net.ID, "no"), domain.ErrForbidden)
	noErr(t, w.Svc.CancelClosure(w.Ctx, w.admin, w.net.ID, "changed our minds"))
	if w.netNow().Closure != nil || !hasKind(w.inbox(w.b), service.NotifClosureCancelled) {
		t.Fatal("closure not cancelled or members not told")
	}
	wantCode(t, w.Svc.CancelClosure(w.Ctx, w.admin, w.net.ID, "again"), domain.ErrInvalidState)
	tu.Must(w.schedule("end", 30))
	w.Clock.Advance(31 * 24 * time.Hour)
	wantCode(t, w.Svc.CancelClosure(w.Ctx, w.admin, w.net.ID, "too late"), domain.ErrInvalidState)
}

// A dispute still unresolved at the date keeps the network closing until it is resolved.
func TestClosureWaitsForDispute(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("end", 0))
	d := tu.Must(w.open(w.a, tx.ID))
	w.Clock.Advance(31 * 24 * time.Hour)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("closed with an open dispute: %d", n)
	}
	if w.netNow().Status != domain.NetworkClosing || !hasKind(w.inbox(w.admin), service.NotifClosureBlocked) {
		t.Fatal("network should be closing with admins told")
	}
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d.ID))
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("finalised %d", n)
	}
}

// GOV-025 f: after the read-only period the members are anonymised through
// AnonymiseMember; the ledger is untouched.
func TestClosureErasureAfterReadOnlyPeriod(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("end", 0))
	w.Clock.Advance(31 * 24 * time.Hour)
	tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
	before := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))

	w.Clock.Advance(89 * 24 * time.Hour)
	if n := tu.Must(w.Svc.EraseClosedNetworks(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("erased inside the period: %d", n)
	}
	w.Clock.Advance(2 * 24 * time.Hour)
	if n := tu.Must(w.Svc.EraseClosedNetworks(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("erased %d networks", n)
	}
	rows, err := w.Store.DB().Query(`SELECT display_name FROM members WHERE network_id = ?`, w.net.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	count := 0
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		count++
		if !strings.HasPrefix(name, "Former member") {
			t.Fatalf("member %q not anonymised", name)
		}
	}
	if count != 3 {
		t.Fatalf("members = %d", count)
	}
	after := tu.Must(w.Svc.VerifyLedger(w.Ctx, sysJob, w.net.ID))
	if len(after.Findings) != len(before.Findings) {
		t.Fatalf("ledger findings changed: %v", after.Findings)
	}
	var erasedAudit int
	_ = w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = ?`, w.net.ID, service.ActClosureErased).Scan(&erasedAudit)
	if tu.Must(w.Svc.GetNetwork(w.Ctx, sysJob, w.net.ID)).Closure.ErasedAt.IsZero() || erasedAudit != 1 {
		t.Fatal("erasure not recorded")
	}
	if n := tu.Must(w.Svc.EraseClosedNetworks(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("second run erased %d", n)
	}
	if _, err := w.Svc.EraseClosedNetworks(w.Ctx, w.admin); err == nil {
		t.Fatal("non-system actor ran the erasure")
	}
}

// GOV-020: a network with no active administrator is read-only and visible as such.
func TestAdminRequiredReadOnly(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	if w.netNow().AdminRequired {
		t.Fatal("admin required with an active administrator")
	}
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID))
	if !n.AdminRequired {
		t.Fatal("members cannot see the administrator-required state")
	}
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 10)
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.Svc.ProposeTransaction(w.Ctx, w.b, w.net.ID, service.ProposeInput{CounterpartyID: w.aM.ID, Direction: service.ProposerIsPaid, Amount: 10, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrInvalidState)
	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("balance read: %v", err)
	}
	tu.Must(w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, service.FormatJSON, &service.MemSink{}))
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='active' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 10))
}

// GOV-014: deadlines are visible; a missed response deadline decides nothing and is reminded once.
func TestDisputeDeadlinesAndReminders(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ResponsePeriodDays = 7 })
	rev, _ := w.reviewer()
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	d := tu.Must(w.open(w.a, tx.ID))
	v := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, d.ID))
	if !v.Deadlines.ResponseDeadline.Equal(d.OpenedAt.AddDate(0, 0, 7)) || !v.Deadlines.DecisionTarget.Equal(d.OpenedAt.AddDate(0, 0, 21)) ||
		v.Deadlines.ResponseOverdue || v.Deadlines.DecisionOverdue {
		t.Fatalf("deadlines = %+v", v.Deadlines)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("early reminders: %d", n)
	}
	w.Clock.Advance(8 * 24 * time.Hour)
	v = tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, d.ID))
	if !v.Deadlines.ResponseOverdue || v.Deadlines.DecisionOverdue || v.Dispute.Status != domain.DisputeOpen {
		t.Fatalf("after the response deadline: %+v status %s", v.Deadlines, v.Dispute.Status)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("response reminders: %d", n)
	}
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("reminder repeated: %d", n)
	}
	if !hasKind(w.inbox(rev), service.NotifDisputeOverdue) || hasKind(w.inbox(w.a), service.NotifDisputeOverdue) {
		t.Fatal("reminder should reach reviewers, not parties")
	}
	w.Clock.Advance(14 * 24 * time.Hour)
	if n := tu.Must(w.Svc.RemindOverdueDisputes(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("decision reminders: %d", n)
	}
	// Silence did not decide the case: the reviewer still assesses it.
	r := tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "no evidence"))
	if r.Status != domain.DisputeResolved || r.Outcome != domain.OutcomeDismissed {
		t.Fatalf("dispute = %+v", r)
	}
	if _, err := w.Svc.RemindOverdueDisputes(w.Ctx, w.admin); err == nil {
		t.Fatal("non-system actor ran the reminders")
	}
}

func (w *world) challenge(by domain.Actor, kind, subject string) (string, error) {
	c, err := w.Svc.OpenChallenge(w.Ctx, by, w.net.ID, service.ChallengeInput{Kind: kind, SubjectID: subject, Reason: "this is unfair"})
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (w *world) decideChallenge(by domain.Actor, id, outcome, why string) error {
	_, err := w.Svc.DecideChallenge(w.Ctx, by, w.net.ID, service.ChallengeDecision{ID: id, Outcome: outcome, Rationale: why})
	return err
}

// GOV-016: a removed member challenges; someone who was not involved decides, with reasons.
func TestChallengeRemovalAndLimit(t *testing.T) {
	w := newWorld(t, nil)
	rev, _ := w.reviewer()
	// Nothing to challenge yet.
	_, err := w.challenge(w.b, service.ChallengeRemoval, "")
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.challenge(w.b, service.ChallengeCreditLimit, "")
	wantCode(t, err, domain.ErrInvalidState)

	tu.Must(w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bM.ID, Reason: "complaints"}))
	id := tu.Must(w.challenge(w.b, service.ChallengeRemoval, ""))
	if !hasKind(w.inbox(rev), service.NotifChallengeAsked) {
		t.Fatal("deciders not told")
	}
	wantCode(t, w.decideChallenge(w.b, id, service.ChallengeUpheld, "me"), domain.ErrForbidden)       // the challenger
	wantCode(t, w.decideChallenge(w.admin, id, service.ChallengeUpheld, "same"), domain.ErrForbidden) // took the original decision
	wantCode(t, w.decideChallenge(w.a, id, service.ChallengeUpheld, "x"), domain.ErrForbidden)        // no reviewer role
	wantCode(t, w.decideChallenge(rev, id, service.ChallengeUpheld, ""), domain.ErrValidation)        // reasons required
	noErr(t, w.decideChallenge(rev, id, service.ChallengeUpheld, "no grounds were recorded"))
	wantCode(t, w.decideChallenge(rev, id, service.ChallengeRejected, "again"), domain.ErrInvalidState)
	if !hasKind(w.inbox(w.b), service.NotifChallengeDecided) {
		t.Fatal("member not told")
	}
	acts := auditActions(w)
	if !hasKind(acts, service.ActChallengeOpened) || !hasKind(acts, service.ActChallengeDecided) {
		t.Fatalf("audit = %v", acts)
	}
	mine := tu.Must(w.Svc.ListChallenges(w.Ctx, w.b, w.net.ID, ""))
	if len(mine) != 1 || mine[0].Outcome != service.ChallengeUpheld {
		t.Fatalf("mine = %+v", mine)
	}
	if got := tu.Must(w.Svc.ListChallenges(w.Ctx, w.a, w.net.ID, "")); len(got) != 0 {
		t.Fatal("a member saw another member's challenge")
	}

	// A credit-limit reduction applied to the member.
	lim := int64(100)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lim, Reason: "risk"}))
	id = tu.Must(w.challenge(w.a, service.ChallengeCreditLimit, ""))
	noErr(t, w.decideChallenge(rev, id, service.ChallengeRejected, "limit stands"))
}
