package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// rrStatus is the state of the one re-review of dispute id.
func (w *world) rrStatus(t *testing.T, id string) string {
	t.Helper()
	v := tu.Must(w.Svc.GetDispute(w.Ctx, w.admin, w.net.ID, id))
	if v.ReReview == nil {
		t.Fatalf("dispute %s has no re-review", id)
	}
	return v.ReReview.Status
}

// DEC-052, DOM-021, DOM-024 (A1-03, was TestA1PoC_03): a re-review of a
// dismissed dispute must not reverse a trade that a later dispute holds in
// posting_pending. It used to post, after which the pending reversal could
// never post (its amount exceeded what was left) or be resolved, so the trade
// stayed disputed and blocked departure and closure for good. The re-review is
// now refused, changing nothing, until the later dispute is decided.
func TestReReviewWaitsWhileAnotherDisputeIsPostingPending(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ReReviewWindowDays = 14 })
	r1, _ := w.reviewer()
	r2, _ := w.reviewerAs("rev2@example.org")
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000) // a=-1000 b=+1000
	d1 := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.resolve(r1, d1.ID, domain.OutcomeDismissed, 0, "no evidence"))
	w.Trade(w.net.ID, w.b, c, w.bM, cM, 10500) // b=-9500 (limit 10000)
	d2 := tu.Must(w.open(w.a, tx.ID))
	if d2r := tu.Must(w.resolve(r1, d2.ID, domain.OutcomeUpheld, 0, "upheld")); d2r.Status != domain.DisputePostingPending || d2r.ReversalAmount != 1000 {
		t.Fatalf("precondition: d2 = %s amount %d", d2r.Status, d2r.ReversalAmount)
	}
	noErr(t, w.rereview(w.a, d1.ID, "please look again"))

	seq := w.netNow().LedgerSeq
	wantCode(t, w.decideRR(r2, d1.ID, string(domain.OutcomePartiallyUpheld), 500), domain.ErrInvalidState)
	wantCode(t, w.decideRR(r2, d1.ID, string(domain.OutcomeUpheld), 0), domain.ErrInvalidState)
	if got := w.txStatus(tx.ID); got.Status != domain.TxDisputed || got.ReversedTotal != 0 || w.netNow().LedgerSeq != seq {
		t.Fatalf("refused re-review changed the trade: %s reversed %d seq %d -> %d", got.Status, got.ReversedTotal, seq, w.netNow().LedgerSeq)
	}
	if st := w.rrStatus(t, d1.ID); st != string(store.ReReviewRequested) {
		t.Fatalf("re-review after the refusal: %s", st)
	}

	// The pending reversal can complete once the payee has room.
	lim := int64(1_000_000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.bM.ID, Limit: &lim, Reason: "resolve dispute"}))
	if d := tu.Must(w.Svc.RetryDisputePosting(w.Ctx, r1, w.net.ID, d2.ID)); d.Status != domain.DisputeResolved {
		t.Fatalf("retry: dispute %s", d.Status)
	}
	if got := w.txStatus(tx.ID); got.Status != domain.TxReversed || got.ReversedTotal != 1000 {
		t.Fatalf("after the retry: %s reversed %d", got.Status, got.ReversedTotal)
	}
	if pv := tu.Must(w.Svc.PreviewDeparture(w.Ctx, w.b, w.net.ID)); len(pv.Blockers) != 0 {
		t.Fatalf("payee departure still blocked: %v", pv.Blockers)
	}
	// Nothing is left to reverse, so the re-review can only confirm.
	wantCode(t, w.decideRR(r2, d1.ID, string(domain.OutcomeUpheld), 0), domain.ErrInvalidState)
	noErr(t, w.decideRR(r2, d1.ID, service.ReReviewConfirmed, 0))
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("ledger not consistent: %v", rep.Findings)
	}
}

// FR-068, FR-075, DEC-052 (A1-03, was TestA1PoC_03b): a re-review must not
// fully reverse a trade while a later dispute on it is open. It used to, which
// left an open dispute on a reversed trade: the opener could not withdraw it
// and it no longer blocked closure. The re-review is now refused; the open
// dispute keeps blocking closure, can be withdrawn, and afterwards the
// re-review reverses the trade.
func TestReReviewCannotReverseUnderAnOpenDispute(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ReReviewWindowDays = 14 })
	r1, _ := w.reviewer()
	r2, _ := w.reviewerAs("rev2@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d1 := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.resolve(r1, d1.ID, domain.OutcomeDismissed, 0, "no evidence"))
	d2 := tu.Must(w.open(w.a, tx.ID))
	noErr(t, w.rereview(w.a, d1.ID, "look again"))

	wantCode(t, w.decideRR(r2, d1.ID, string(domain.OutcomeUpheld), 0), domain.ErrInvalidState)
	if st := w.txStatus(tx.ID).Status; st != domain.TxDisputed {
		t.Fatalf("trade after the refused re-review: %s", st)
	}
	_, err := w.schedule("end", 0)
	wantCode(t, err, domain.ErrBlocked)

	if d := tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d2.ID)); d.Status != domain.DisputeWithdrawn {
		t.Fatalf("withdraw: %s", d.Status)
	}
	noErr(t, w.decideRR(r2, d1.ID, string(domain.OutcomeUpheld), 0))
	if got := w.txStatus(tx.ID); got.Status != domain.TxReversed || got.ReversedTotal != 1000 {
		t.Fatalf("after the re-review: %s reversed %d", got.Status, got.ReversedTotal)
	}
	tu.Must(w.schedule("end", 0))
}
