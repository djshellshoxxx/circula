package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-019 scenario (OPS-028, FR-038): one proposal is declined and another
// expires after the configured period. Neither posts an entry, the statuses
// are declined and expired, each closure is audited, and the parties are
// told: the proposer of the declined one, and both parties of the expired one
// (the expiry job acts for nobody, so nobody is skipped). Running the sweep
// again sends no second notice (OPS-027).
func TestQA019DeclineAndExpiryScenario(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ProposalExpiryDays = 3 })
	count := func(q string, a ...any) int {
		t.Helper()
		var n int
		if err := w.Store.DB().QueryRow(q, a...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	kindCount := func(a domain.Actor, kind string) int {
		n := 0
		for _, k := range w.inbox(a) {
			if k == kind {
				n++
			}
		}
		return n
	}

	declined := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 700))
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: declined.ID, IdempotencyKey: w.Key()}))

	expiring := tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 400)) // B asks A to pay

	// Just before the configured period nothing expires.
	w.Clock.Advance(3*24*time.Hour - time.Minute)
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("expiry-job"), 100)); n != 0 {
		t.Fatalf("expired %d before the period ended", n)
	}
	w.Clock.Advance(2 * time.Minute)
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("expiry-job"), 100)); n != 1 {
		t.Fatalf("expired %d, want 1", n)
	}
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("expiry-job"), 100)); n != 0 {
		t.Fatalf("second sweep expired %d, want 0", n)
	}

	for id, want := range map[string]domain.TxStatus{declined.ID: domain.TxDeclined, expiring.ID: domain.TxExpired} {
		if got := w.txStatus(id).Status; got != want {
			t.Errorf("tx %s status %s, want %s", id, got, want)
		}
		if n := count(`SELECT COUNT(*) FROM entries WHERE tx_id = ?`, id); n != 0 {
			t.Errorf("tx %s has %d entries", id, n)
		}
	}
	if n := count(`SELECT COUNT(*) FROM entries`); n != 0 {
		t.Errorf("%d entries posted", n)
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action = ? AND target_id = ?`, domain.ActTxDeclined, declined.ID); n != 1 {
		t.Errorf("decline audit events = %d", n)
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action = ? AND target_id = ?`, domain.ActTxExpired, expiring.ID); n != 1 {
		t.Errorf("expiry audit events = %d", n)
	}

	// Notices: the proposer hears of the decline, the decliner does not.
	if n := kindCount(w.a, service.NotifTradeDeclined); n != 1 {
		t.Errorf("proposer decline notices = %d, inbox %v", n, w.inbox(w.a))
	}
	if n := kindCount(w.b, service.NotifTradeDeclined); n != 0 {
		t.Errorf("decliner notified of own decline: %v", w.inbox(w.b))
	}
	// Both parties hear of the expiry, exactly once each.
	for name, a := range map[string]domain.Actor{"payer": w.a, "proposer": w.b} {
		if n := kindCount(a, service.NotifTradeExpired); n != 1 {
			t.Errorf("%s expiry notices = %d, inbox %v", name, n, w.inbox(a))
		}
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("ledger: %v", rep.Findings)
	}
}
