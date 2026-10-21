package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-081: required content on every event.
func TestAuditEventContent(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{TargetID: tx.ID}))
	if len(evs) != 2 {
		t.Fatalf("events for tx = %d (proposed, completed)", len(evs))
	}
	for _, e := range evs {
		if e.ID == "" || e.NetworkID != w.net.ID || e.Seq == 0 || e.OccurredAt.IsZero() || e.ActorType == "" ||
			e.Action == "" || e.TargetType == "" || e.TargetID == "" || e.Hash == "" {
			t.Errorf("incomplete event %+v", e)
		}
	}
	if evs[0].Action != domain.ActTxProposed || evs[1].Action != domain.ActTxCompleted || evs[1].ActorID != w.bM.ID {
		t.Errorf("events = %s by %s, %s by %s", evs[0].Action, evs[0].ActorID, evs[1].Action, evs[1].ActorID)
	}
	if !strings.Contains(evs[1].SummaryAfter, `"amount":2500`) || !strings.Contains(evs[1].SummaryBefore, "proposed") {
		t.Errorf("summaries %q -> %q", evs[1].SummaryBefore, evs[1].SummaryAfter)
	}
	if evs[1].IdemRef == "" || strings.Contains(evs[1].IdemRef, "key-") {
		t.Errorf("idempotency reference %q must be a hash", evs[1].IdemRef)
	}
	// filters + keyset pagination
	all := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Limit: 2}))
	next := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Limit: 2, AfterSeq: all[1].Seq}))
	if len(all) != 2 || len(next) == 0 || next[0].Seq != all[1].Seq+1 {
		t.Errorf("pagination broken: %d, %v", len(all), next)
	}
	if got := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActTxCompleted, ActorID: w.bM.ID})); len(got) != 1 {
		t.Errorf("filter returned %d", len(got))
	}
}

// DAT-033 / SEC-019: members see only their own account's events, without other
// members' identifiers, request ids or hash material.
func TestAuditVisibilityForMembers(t *testing.T) {
	w := newWorld(t, nil)
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "internal reason"))

	mine := tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{}))
	if len(mine) == 0 {
		t.Fatal("member sees no events about own account")
	}
	for _, e := range mine {
		raw := strings.Join([]string{e.ActorID, e.SummaryBefore, e.SummaryAfter, e.Reason, strings.Join(e.SubjectIDs, ",")}, " ")
		for _, other := range []string{w.bM.ID, w.adminM.ID, cM.ID} {
			if strings.Contains(raw, other) && e.ActorID != other {
				t.Errorf("event %s leaks %s: %s", e.Action, other, raw)
			}
		}
		if e.Hash != "" || e.PrevHash != "" || e.RequestID != "" {
			t.Errorf("hash/request material exposed: %+v", e)
		}
		if e.Action == domain.ActMemberSuspended && (e.Reason != "" || e.ActorType != domain.ActorAdmin) {
			t.Errorf("suspension reason must be hidden and actor reduced to a role: %+v", e)
		}
	}
	// carol has nothing to do with the trade
	if got := tu.Must(w.Svc.ListAudit(w.Ctx, c, w.net.ID, service.AuditQuery{TargetID: tx.ID})); len(got) != 0 {
		t.Errorf("uninvolved member sees %d events", len(got))
	}
	// the counterparty sees the trade events but not those about A's suspension
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{})) {
		if e.Action == domain.ActMemberSuspended {
			t.Error("B sees A's suspension")
		}
	}
	// reviewers see dispute events only
	rev, _ := w.reviewer()
	d := tu.Must(w.open(w.b, tx.ID))
	_ = d
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, rev, w.net.ID, service.AuditQuery{})) {
		if e.TargetType != "dispute" {
			t.Errorf("reviewer sees %s event", e.TargetType)
		}
	}
	// an administrator's audit read is itself audited (DAT-034)
	before := len(tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActAuditRead})))
	after := len(tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActAuditRead})))
	if after != before+1 {
		t.Errorf("audit.read events %d -> %d", before, after)
	}
}

// FR-083 / LED-2 / DAT-011: the application has no path that edits or deletes
// ledger or audit rows, and the database refuses raw attempts.
func TestAppendOnlyThroughRawAccess(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	tu.Must(w.reverse(w.admin, tx.ID, 100, "partial"))
	db := w.Store.DB()
	for _, q := range []string{
		`UPDATE entries SET amount = amount + 1`, `DELETE FROM entries`,
		`UPDATE audit_events SET reason = 'x'`, `DELETE FROM audit_events`,
		`UPDATE transaction_status_events SET to_status = 'x'`, `DELETE FROM transaction_status_events`,
		`UPDATE transactions SET amount = 1`, `DELETE FROM transactions`,
		`UPDATE consistency_runs SET ok = 0`,
	} {
		tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)) // ensure consistency_runs has a row
		if _, err := db.Exec(q); err == nil {
			t.Errorf("%q succeeded", q)
		}
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("ledger damaged: %v", rep.Findings)
	}
}

// DAT-030: the audit chain detects silent edits (triggers are dropped here to
// simulate a database administrator).
func TestAuditChainDetectsTampering(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK || rep.AuditEvents == 0 {
		t.Fatalf("clean ledger reported %+v", rep)
	}
	db := w.Store.DB()
	if _, err := db.Exec(`DROP TRIGGER audit_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE audit_events SET summary_after = '{"amount":1}' WHERE action = 'transaction.completed'`); err != nil {
		t.Fatal(err)
	}
	rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if rep.OK || !strings.Contains(strings.Join(rep.Findings, " "), "hash") {
		t.Errorf("tampering not detected: %+v", rep)
	}
}

// LED-3: the check finds a corrupted cached balance; DAT-021: history reconstructable.
func TestVerifyDetectsBalanceCorruptionAndBalanceAt(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	cur := w.balance(w.b)
	if got := tu.Must(w.Svc.BalanceAt(w.Ctx, w.b, w.net.ID, 0)); got != cur.Balance {
		t.Errorf("latest sum %d != cached %d", got, cur.Balance)
	}
	if got := tu.Must(w.Svc.BalanceAt(w.Ctx, w.b, w.net.ID, 1)); got != 1000 {
		t.Errorf("balance at seq 1 = %d", got)
	}
	if got := tu.Must(w.Svc.BalanceAt(w.Ctx, w.b, w.net.ID, 2)); got != 1500 {
		t.Errorf("balance at seq 2 = %d", got)
	}
	if _, err := w.Store.DB().Exec(`UPDATE accounts SET balance = balance + 7 WHERE member_id = ?`, w.aM.ID); err != nil {
		t.Fatal(err)
	}
	rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if rep.OK || len(rep.Findings) < 2 { // account mismatch and non-zero network total
		t.Errorf("corruption not detected: %+v", rep)
	}
	if n := countRows(t, w.Env, "consistency_runs"); n != 1 {
		t.Errorf("consistency runs recorded = %d", n)
	}
}

// Operator tooling (CLI) verifies with the system actor and records the run.
func TestVerifyAsSystemActor(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, domain.SystemActor("cli-1"), w.net.ID))
	if !rep.OK || rep.LedgerSeq != 1 {
		t.Errorf("report %+v", rep)
	}
	ev := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActVerify}))
	if len(ev) != 1 || ev[0].ActorType != domain.ActorSystem || ev[0].RequestID != "cli-1" {
		t.Errorf("verify audit = %+v", ev)
	}
}
