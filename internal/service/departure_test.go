package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// departRefused asserts that the departure is refused as a state error and
// changes nothing: the leaver's balance view (balance, account state,
// reservations, ledger sequence) is as before.
func departRefused(t *testing.T, w *world, by domain.Actor, in service.DepartInput, leaver domain.Actor) {
	t.Helper()
	before := *w.balance(leaver)
	_, err := w.Svc.DepartMember(w.Ctx, by, w.net.ID, in)
	wantCode(t, err, domain.ErrInvalidState)
	if after := *w.balance(leaver); after != before {
		t.Fatalf("refused departure changed something: %+v -> %+v", before, after)
	}
}

// GOV-025 b, FR-068 (A1-01, was TestA1PoC_01): once the network is closed, a
// settle-to-zero departure that needs a settlement is refused, whoever starts
// it, so the ledger the final export certified stays as it was. A departure
// that posts nothing still completes.
func TestDepartureSettlementRefusedAfterClosure(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DepartureRule = domain.DepartSettleToZero })
	c, _ := w.Member2(w.admin, "carol@example.org") // balance 0: nothing to settle
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("end of pilot", 0))
	w.Clock.Advance(31 * 24 * time.Hour)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("finalised %d", n)
	}
	if st := w.netNow().Status; st != domain.NetworkClosed {
		t.Fatalf("status %s", st)
	}

	departRefused(t, w, w.a, service.DepartInput{}, w.a) // owes 500
	departRefused(t, w, w.b, service.DepartInput{}, w.b) // is owed 500
	departRefused(t, w, w.admin, service.DepartInput{MemberID: w.aM.ID, ReasonCode: domain.RemovalRuleBreach, Reason: "wind-down"}, w.a)
	tu.Must(w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{}))
	if st := w.balance(c).AccountState; st != domain.AccountClosed {
		t.Fatalf("departure with nothing to settle: account %s", st)
	}
	if chk := tu.Must(w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID)); !chk.Match {
		t.Fatalf("the certified final export no longer matches: %+v", chk)
	}
}

// GOV-025 b (A1-01): postings freeze at the closure date itself. One
// millisecond before it a settlement still posts; at the date, while the
// closure has not completed yet (closing), it is refused.
func TestDepartureSettlementFrozenFromClosureDate(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DepartureRule = domain.DepartSettleToZero })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	c := tu.Must(w.schedule("end of pilot", 0))

	w.Clock.Advance(c.ClosingAt.Sub(w.Clock.Now()) - time.Millisecond)
	tu.Must(w.Svc.DepartMember(w.Ctx, w.b, w.net.ID, service.DepartInput{}))
	if got := w.balance(w.b).Balance; got != 0 {
		t.Fatalf("balance settled before the closure date = %d", got)
	}

	w.Clock.Advance(time.Millisecond)
	if st := w.netNow().Status; st != domain.NetworkClosing {
		t.Fatalf("status at the closure date = %s", st)
	}
	departRefused(t, w, w.a, service.DepartInput{}, w.a)
}

// GOV-023 (A1-01, was TestA1PoC_01b): a suspended network accepts no
// postings, so a departure that needs a settlement is refused, voluntary or by
// an administrator, and the refusal also leaves the member's open proposals
// open. Leaving with nothing to settle still works, and the settlement posts
// once trading resumes.
func TestDepartureSettlementRefusedWhileSuspended(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DepartureRule = domain.DepartSettleToZero })
	c, _ := w.Member2(w.admin, "carol@example.org")
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	open := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 50))
	noErr(t, w.Svc.SuspendNetwork(w.Ctx, w.admin, w.net.ID, "security review"))

	departRefused(t, w, w.a, service.DepartInput{}, w.a)
	departRefused(t, w, w.admin, service.DepartInput{MemberID: w.aM.ID, ReasonCode: domain.RemovalRuleBreach, Reason: "left town"}, w.a)
	if got := w.txStatus(open.ID).Status; got != domain.TxProposed {
		t.Fatalf("refused departure left the open proposal %s", got)
	}
	tu.Must(w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{}))

	noErr(t, w.Svc.ResumeNetwork(w.Ctx, w.admin, w.net.ID, "review done"))
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
	if got := w.balance(w.a).Balance; got != 0 {
		t.Fatalf("balance settled after resumption = %d", got)
	}
	if got := w.txStatus(open.ID).Status; got != domain.TxCancelled {
		t.Fatalf("open proposal after the departure: %s", got)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("ledger not consistent: %v", rep.Findings)
	}
}

// GOV-020 (A1-01, was TestA1PoC_01c): with no active administrator the
// network is read-only and no value moves, so a departure that needs a
// settlement is refused.
func TestDepartureSettlementRefusedWhenAdminRequired(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DepartureRule = domain.DepartSettleToZero })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	// As in TestAdminRequiredReadOnly: the only administrator is no longer active.
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		t.Fatal(err)
	}
	if !tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		t.Fatal("precondition: administrator required")
	}
	departRefused(t, w, w.a, service.DepartInput{}, w.a)
}
