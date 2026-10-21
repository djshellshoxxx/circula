package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) open(by domain.Actor, txID string) (*domain.Dispute, error) {
	return w.Svc.OpenDispute(w.Ctx, by, w.net.ID, service.OpenDisputeInput{TxID: txID, Category: "not_delivered",
		Statement: "never arrived", IdempotencyKey: w.Key()})
}

func (w *world) resolve(by domain.Actor, id string, o domain.DisputeOutcome, amt int64, why string) (*domain.Dispute, error) {
	return w.Svc.ResolveDispute(w.Ctx, by, w.net.ID, service.ResolveInput{DisputeID: id, Outcome: o, Amount: amt, Rationale: why, IdempotencyKey: w.Key()})
}

// reviewer adds an independent member holding only the reviewer role.
func (w *world) reviewer() (domain.Actor, *domain.Member) {
	r, rM := w.Member2(w.admin, "rev@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, rM.ID, domain.RoleReviewer))
	return r, rM
}

func (w *world) txStatus(id string) *domain.Transaction {
	return tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, id)).Tx
}

// DOM-017 + FR-071: dispute opens within the window, posts nothing.
func TestOpenDisputeWindowAndNoPostings(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DisputeWindowDays = 30 })
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	beforeA, beforeB := w.balance(w.a), w.balance(w.b)
	entriesBefore := w.entries(tx.ID)

	w.Clock.Advance(29 * 24 * time.Hour) // window - 1 day
	d := tu.Must(w.open(w.a, tx.ID))
	if d.Status != domain.DisputeOpen || w.txStatus(tx.ID).Status != domain.TxDisputed {
		t.Fatalf("dispute %s tx %s", d.Status, w.txStatus(tx.ID).Status)
	}
	if w.balance(w.a).Balance != beforeA.Balance || w.balance(w.b).Balance != beforeB.Balance || len(w.entries(tx.ID)) != len(entriesBefore) {
		t.Error("opening a dispute changed postings")
	}
	// a second dispute on the same transaction is refused
	_, err := w.open(w.b, tx.ID)
	wantCode(t, err, domain.ErrInvalidState)

	// window + 1 day in another transaction
	tx2 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	w.Clock.Advance(31 * 24 * time.Hour)
	_, err = w.open(w.a, tx2.ID)
	wantCode(t, err, domain.ErrInvalidState)
	if w.txStatus(tx2.ID).Status != domain.TxCompleted {
		t.Error("refused dispute changed status")
	}
}

func TestOpenDisputeRules(t *testing.T) {
	w := newWorld(t, nil)
	c, _ := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	pending := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 10))

	_, err := w.open(c, tx.ID) // non-party: same as missing
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.open(w.a, pending.ID) // not completed
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "nope", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	// DOM-039: suspended members cannot open disputes
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "review"))
	_, err = w.open(w.b, tx.ID)
	wantCode(t, err, domain.ErrForbidden)
	// reversal transactions cannot be disputed
	tu.Must(0, w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "ok"))
	rev := tu.Must(w.reverse(w.admin, tx.ID, 100, "partial"))
	_, err = w.open(w.a, rev.ID)
	wantCode(t, err, domain.ErrInvalidState)
}

func TestResolveOutcomes(t *testing.T) {
	t.Run("dismissed", func(t *testing.T) {
		w := newWorld(t, nil)
		rev, _ := w.reviewer()
		tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
		d := tu.Must(w.open(w.a, tx.ID))
		got := tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "work was delivered"))
		if got.Status != domain.DisputeResolved || got.Outcome != domain.OutcomeDismissed || got.ReviewerID == "" || got.Rationale == "" {
			t.Fatalf("dispute = %+v", got)
		}
		if w.txStatus(tx.ID).Status != domain.TxCompleted || w.balance(w.a).Balance != -1000 {
			t.Error("dismissal must restore completed and post nothing")
		}
	})
	t.Run("upheld", func(t *testing.T) {
		w := newWorld(t, nil)
		rev, _ := w.reviewer()
		tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
		d := tu.Must(w.open(w.a, tx.ID))
		tu.Must(w.Svc.StartDisputeReview(w.Ctx, rev, w.net.ID, d.ID))
		got := tu.Must(w.resolve(rev, d.ID, domain.OutcomeUpheld, 0, "not delivered"))
		if got.Status != domain.DisputeResolved || got.ReversalAmount != 1000 {
			t.Fatalf("dispute = %+v", got)
		}
		if w.txStatus(tx.ID).Status != domain.TxReversed || w.balance(w.a).Balance != 0 || w.balance(w.b).Balance != 0 {
			t.Error("upheld must fully reverse")
		}
	})
	t.Run("partially-upheld", func(t *testing.T) {
		w := newWorld(t, nil)
		rev, _ := w.reviewer()
		tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
		d := tu.Must(w.open(w.a, tx.ID))
		_, err := w.resolve(rev, d.ID, domain.OutcomePartiallyUpheld, 0, "x")
		wantCode(t, err, domain.ErrValidation)
		_, err = w.resolve(rev, d.ID, domain.OutcomePartiallyUpheld, 1001, "x")
		wantCode(t, err, domain.ErrValidation)
		tu.Must(w.resolve(rev, d.ID, domain.OutcomePartiallyUpheld, 400, "half delivered"))
		got := w.txStatus(tx.ID)
		if got.Status != domain.TxCompleted || got.ReversedTotal != 400 || w.balance(w.a).Balance != -600 {
			t.Errorf("tx %s reversed=%d balA=%d", got.Status, got.ReversedTotal, w.balance(w.a).Balance)
		}
	})
}

// DOM-038 / FR-072 / DOM-018: who may decide.
func TestDisputeDecisionAuthorization(t *testing.T) {
	w := newWorld(t, nil)
	rev, _ := w.reviewer()
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))

	_, err := w.resolve(w.a, d.ID, domain.OutcomeUpheld, 0, "pay me") // opener / plain member
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.resolve(w.b, d.ID, domain.OutcomeDismissed, 0, "no")
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "  ") // rationale mandatory
	wantCode(t, err, domain.ErrValidation)
	_, err = w.resolve(rev, d.ID, "maybe", 0, "x")
	wantCode(t, err, domain.ErrValidation)

	// an administrator who is a party cannot decide, even holding the reviewer role
	d2tx := w.Trade(w.net.ID, w.admin, w.b, w.adminM, w.bM, 100)
	d2 := tu.Must(w.open(w.b, d2tx.ID))
	_, err = w.resolve(w.admin, d2.ID, domain.OutcomeDismissed, 0, "my own trade")
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.StartDisputeReview(w.Ctx, w.admin, w.net.ID, d2.ID)
	wantCode(t, err, domain.ErrForbidden)

	// a reviewer who is a party is refused on that dispute but fine on others
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleReviewer))
	_, err = w.resolve(w.b, d2.ID, domain.OutcomeDismissed, 0, "x")
	wantCode(t, err, domain.ErrForbidden)
	// a reviewer who took part by reversing is conflicted (admin2 reversed, then cannot decide)
	adm2, adm2M := w.Member2(w.admin, "adm2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	tx3 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 200)
	tu.Must(w.reverse(adm2, tx3.ID, 50, "partial"))
	d3 := tu.Must(w.open(w.a, tx3.ID))
	_, err = w.resolve(adm2, d3.ID, domain.OutcomeDismissed, 0, "I reversed part of it myself")
	wantCode(t, err, domain.ErrForbidden)

	// the right person can; a closed dispute cannot be decided again
	tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "fine"))
	_, err = w.resolve(rev, d.ID, domain.OutcomeUpheld, 0, "changed my mind")
	wantCode(t, err, domain.ErrInvalidState)
}

func TestWithdrawDispute(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	_, err := w.Svc.WithdrawDispute(w.Ctx, w.b, w.net.ID, d.ID) // only the opener
	wantCode(t, err, domain.ErrNotFound)
	got := tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d.ID))
	if got.Status != domain.DisputeWithdrawn || w.txStatus(tx.ID).Status != domain.TxCompleted {
		t.Errorf("dispute %s tx %s", got.Status, w.txStatus(tx.ID).Status)
	}
	_, err = w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, d.ID)
	wantCode(t, err, domain.ErrInvalidState)
	// after withdrawal the trade can be disputed again
	tu.Must(w.open(w.a, tx.ID))
}

// DOM-021 / FR-074: upheld reversal blocked by LED-5 -> posting_pending, nothing posts;
// after an audited time-limited limit override the retry posts.
func TestUpheldReversalBlockedThenOverride(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	rev, _ := w.reviewer()
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, c, w.bM, cM, 5000) // B spent it all: -2500 at its limit
	d := tu.Must(w.open(w.a, tx.ID))
	seqBefore := w.balance(w.b).LedgerSeq

	got := tu.Must(w.resolve(rev, d.ID, domain.OutcomeUpheld, 0, "not delivered"))
	if got.Status != domain.DisputePostingPending || got.Outcome != domain.OutcomeUpheld {
		t.Fatalf("dispute = %+v", got)
	}
	if w.txStatus(tx.ID).Status != domain.TxDisputed || w.balance(w.b).LedgerSeq != seqBefore || w.balance(w.b).Balance != -2500 {
		t.Error("blocked reversal must not post or change status")
	}
	// resolve is not the way out of posting_pending, and retry still fails
	_, err := w.resolve(rev, d.ID, domain.OutcomeUpheld, 0, "again")
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.Svc.RetryDisputePosting(w.Ctx, rev, w.net.ID, d.ID)
	wantCode(t, err, domain.ErrInsufficientCredit)
	// a departure is blocked while a dispute is pending (DOM-024)
	_, err = w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{})
	wantCode(t, err, domain.ErrBlocked)

	big := int64(10000)
	exp := w.Clock.Now().Add(24 * time.Hour)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.bM.ID, Limit: &big, ExpiresAt: &exp, Reason: "dispute DOM-021"}))
	done := tu.Must(w.Svc.RetryDisputePosting(w.Ctx, rev, w.net.ID, d.ID))
	if done.Status != domain.DisputeResolved || w.txStatus(tx.ID).Status != domain.TxReversed {
		t.Fatalf("dispute %s tx %s", done.Status, w.txStatus(tx.ID).Status)
	}
	if w.balance(w.a).Balance != 0 || w.balance(w.b).Balance != -5000 {
		t.Errorf("balances A=%d B=%d", w.balance(w.a).Balance, w.balance(w.b).Balance)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
	v := tu.Must(w.Svc.GetDispute(w.Ctx, rev, w.net.ID, d.ID))
	if len(v.Events) < 3 || v.PayerName == "" || v.PayeeName == "" {
		t.Errorf("dispute view = %+v", v)
	}
}

func TestDisputeVisibility(t *testing.T) {
	w := newWorld(t, nil)
	c, _ := w.Member2(w.admin, "carol@example.org")
	rev, _ := w.reviewer()
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	if _, err := w.Svc.GetDispute(w.Ctx, c, w.net.ID, d.ID); err == nil {
		t.Error("outsider must not read a dispute")
	} else {
		wantCode(t, err, domain.ErrNotFound)
	}
	for _, who := range []domain.Actor{w.a, w.b, rev, w.admin} {
		tu.Must(w.Svc.GetDispute(w.Ctx, who, w.net.ID, d.ID))
	}
	if l := tu.Must(w.Svc.ListDisputes(w.Ctx, c, w.net.ID, nil, service.Page{})); len(l) != 0 {
		t.Errorf("outsider lists %d disputes", len(l))
	}
	if l := tu.Must(w.Svc.ListDisputes(w.Ctx, w.b, w.net.ID, nil, service.Page{})); len(l) != 1 {
		t.Errorf("party lists %d disputes", len(l))
	}
	// reviewers may read the transaction under dispute but not other transactions
	tu.Must(w.Svc.GetTransaction(w.Ctx, rev, w.net.ID, tx.ID))
	other := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 5)
	if _, err := w.Svc.GetTransaction(w.Ctx, rev, w.net.ID, other.ID); err == nil {
		t.Error("reviewer must not read transactions without a dispute")
	}
}
