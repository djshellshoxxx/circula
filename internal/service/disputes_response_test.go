package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) reviewerAs(login string) (domain.Actor, *domain.Member) {
	r, rM := w.Member2(w.admin, login)
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, rM.ID, domain.RoleReviewer))
	return r, rM
}

func (w *world) respond(by domain.Actor, id, body string) error {
	_, err := w.Svc.RespondToDispute(w.Ctx, by, w.net.ID, service.RespondInput{DisputeID: id, Response: body, IdempotencyKey: w.Key()})
	return err
}

func (w *world) rereview(by domain.Actor, id, why string) (err error) {
	_, err = w.Svc.RequestReReview(w.Ctx, by, w.net.ID, service.ReReviewRequestInput{DisputeID: id, Reason: why, IdempotencyKey: w.Key()})
	return err
}

func (w *world) decideRR(by domain.Actor, id, outcome string, amt int64) error {
	_, err := w.Svc.DecideReReview(w.Ctx, by, w.net.ID, service.ReReviewDecideInput{DisputeID: id, Outcome: outcome, Amount: amt,
		Rationale: "second look", IdempotencyKey: w.Key()})
	return err
}

// FR-076: the other party responds once; opener, reviewer and admin see it; audited; opener notified.
func TestRespondToDispute(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	c, _ := w.Member2(w.admin, "carol@example.org")
	rev, _ := w.reviewerAs("rev@example.org")

	wantCode(t, w.respond(w.a, d.ID, "I am the opener"), domain.ErrForbidden) // opener cannot answer themselves
	wantCode(t, w.respond(c, d.ID, "not a party"), domain.ErrNotFound)
	wantCode(t, w.respond(w.b, d.ID, "   "), domain.ErrValidation)
	wantCode(t, w.respond(w.b, d.ID, strings.Repeat("x", 2001)), domain.ErrValidation)
	wantCode(t, w.respond(w.b, "missing", "hello"), domain.ErrNotFound)

	r := tu.Must(w.Svc.RespondToDispute(w.Ctx, w.b, w.net.ID, service.RespondInput{DisputeID: d.ID, Response: "  I delivered on Monday.  ", IdempotencyKey: "k-resp"}))
	if r.Body != "I delivered on Monday." || r.ResponderID != w.bM.ID {
		t.Fatalf("response = %+v", r)
	}
	// replay with the same key returns the same record; a second distinct response is refused
	again := tu.Must(w.Svc.RespondToDispute(w.Ctx, w.b, w.net.ID, service.RespondInput{DisputeID: d.ID, Response: "  I delivered on Monday.  ", IdempotencyKey: "k-resp"}))
	if again.ID != r.ID {
		t.Error("replay created a second response")
	}
	wantCode(t, w.respond(w.b, d.ID, "one more thing"), domain.ErrInvalidState)

	for name, who := range map[string]domain.Actor{"opener": w.a, "reviewer": rev, "admin": w.admin, "responder": w.b} {
		v := tu.Must(w.Svc.GetDispute(w.Ctx, who, w.net.ID, d.ID))
		if v.Response == nil || v.Response.Body != r.Body {
			t.Errorf("%s does not see the response: %+v", name, v.Response)
		}
	}
	if _, err := w.Svc.GetDispute(w.Ctx, c, w.net.ID, d.ID); err == nil {
		t.Error("outsider can read the dispute")
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActDisputeResponded}))
	if len(evs) != 1 || evs[0].TargetID != d.ID || evs[0].ActorID != w.bM.ID {
		t.Fatalf("audit = %+v", evs)
	}
	inbox := tu.Must(w.Svc.ListNotifications(w.Ctx, w.a, w.net.ID, service.NotificationQuery{}))
	if len(inbox) == 0 || inbox[0].Kind != service.NotifDisputeResponse {
		t.Errorf("opener not notified: %+v", inbox)
	}

	// after a decision no response can be added
	d2 := tu.Must(w.open(w.b, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500).ID))
	tu.Must(w.resolve(rev, d2.ID, domain.OutcomeDismissed, 0, "no action"))
	wantCode(t, w.respond(w.a, d2.ID, "too late"), domain.ErrInvalidState)
}

func TestRespondSuspendedPartyMayAnswerButNotPending(t *testing.T) {
	w := newWorld(t, nil)
	d := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "review"))
	if err := w.respond(w.b, d.ID, "still my side"); err != nil {
		t.Fatalf("suspended responder refused: %v", err)
	}
}

// FR-078: one re-review, by a party, within the window, by a different reviewer; original untouched.
func TestReReviewHappyPath(t *testing.T) {
	w := newWorld(t, nil)
	rev1, rev1M := w.reviewerAs("rev1@example.org")
	rev2, rev2M := w.reviewerAs("rev2@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))

	wantCode(t, w.rereview(w.a, d.ID, "premature"), domain.ErrInvalidState) // not decided yet
	tu.Must(w.resolve(rev1, d.ID, domain.OutcomeDismissed, 0, "evidence thin"))

	c, _ := w.Member2(w.admin, "carol@example.org")
	wantCode(t, w.rereview(c, d.ID, "not mine"), domain.ErrNotFound)
	wantCode(t, w.rereview(w.a, d.ID, " "), domain.ErrValidation)
	wantCode(t, w.rereview(rev1, d.ID, "reviewer is not a party"), domain.ErrNotFound)
	wantCode(t, w.decideRR(rev2, d.ID, "upheld", 0), domain.ErrInvalidState) // nothing requested yet

	rr := tu.Must(w.Svc.RequestReReview(w.Ctx, w.a, w.net.ID, service.ReReviewRequestInput{DisputeID: d.ID, Reason: "new evidence", IdempotencyKey: w.Key()}))
	if rr.Status != "requested" || rr.RequestedBy != w.aM.ID {
		t.Fatalf("re-review = %+v", rr)
	}
	// once only, by either party
	wantCode(t, w.rereview(w.a, d.ID, "again"), domain.ErrInvalidState)
	wantCode(t, w.rereview(w.b, d.ID, "me too"), domain.ErrInvalidState)

	// the original reviewer, parties and ordinary members cannot decide
	wantCode(t, w.decideRR(rev1, d.ID, "upheld", 0), domain.ErrForbidden)
	wantCode(t, w.decideRR(w.a, d.ID, "upheld", 0), domain.ErrForbidden)
	wantCode(t, w.decideRR(c, d.ID, "upheld", 0), domain.ErrForbidden)
	wantCode(t, w.decideRR(rev2, d.ID, "maybe", 0), domain.ErrValidation)
	wantCode(t, w.decideRR(rev2, d.ID, "partially_upheld", 5000), domain.ErrValidation)
	wantCode(t, w.decideRR(rev2, d.ID, "confirmed", 5), domain.ErrValidation)

	aBefore, bBefore := w.balance(w.a).Balance, w.balance(w.b).Balance
	tu.Must(0, w.decideRR(rev2, d.ID, "upheld", 0))
	if got := w.balance(w.a).Balance; got != aBefore+1000 {
		t.Errorf("payer balance %d, want %d", got, aBefore+1000)
	}
	if got := w.balance(w.b).Balance; got != bBefore-1000 {
		t.Errorf("payee balance %d, want %d", got, bBefore-1000)
	}
	if got := w.txStatus(tx.ID); got.Status != domain.TxReversed || got.ReversedTotal != 1000 {
		t.Errorf("tx = %+v", got)
	}
	wantCode(t, w.decideRR(rev2, d.ID, "upheld", 0), domain.ErrInvalidState) // already decided

	v := tu.Must(w.Svc.GetDispute(w.Ctx, w.b, w.net.ID, d.ID))
	if v.Dispute.Outcome != domain.OutcomeDismissed || v.Dispute.ReviewerID != rev1M.ID {
		t.Errorf("original decision was edited: %+v", v.Dispute)
	}
	if v.ReReview == nil || v.ReReview.Outcome != "upheld" || v.ReReview.ReviewerID != rev2M.ID || v.ReReview.ReversalAmount != 1000 {
		t.Errorf("linked re-review = %+v", v.ReReview)
	}
	wantCode(t, w.rereview(w.b, d.ID, "after decided"), domain.ErrInvalidState)
	if n := len(tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActDisputeReReviewDecide}))); n != 1 {
		t.Errorf("decision audit events = %d", n)
	}
}

func TestReReviewConfirmedAndPartial(t *testing.T) {
	w := newWorld(t, nil)
	rev1, _ := w.reviewerAs("rev1@example.org")
	rev2, _ := w.reviewerAs("rev2@example.org")

	// confirmed: nothing is posted
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.resolve(rev1, d.ID, domain.OutcomeDismissed, 0, "no"))
	tu.Must(w.Svc.RequestReReview(w.Ctx, w.b, w.net.ID, service.ReReviewRequestInput{DisputeID: d.ID, Reason: "check", IdempotencyKey: w.Key()}))
	before := w.balance(w.a).Balance
	tu.Must(0, w.decideRR(rev2, d.ID, "confirmed", 0))
	if w.balance(w.a).Balance != before || w.txStatus(tx.ID).Status != domain.TxCompleted {
		t.Error("confirming changed postings")
	}

	// a fully upheld original leaves nothing to reverse: only confirmed is possible
	tx2 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 700)
	d2 := tu.Must(w.open(w.a, tx2.ID))
	tu.Must(w.resolve(rev1, d2.ID, domain.OutcomeUpheld, 0, "yes"))
	tu.Must(w.Svc.RequestReReview(w.Ctx, w.b, w.net.ID, service.ReReviewRequestInput{DisputeID: d2.ID, Reason: "unfair", IdempotencyKey: w.Key()}))
	wantCode(t, w.decideRR(rev2, d2.ID, "upheld", 0), domain.ErrInvalidState)
	wantCode(t, w.decideRR(rev2, d2.ID, "partially_upheld", 100), domain.ErrValidation)
	tu.Must(0, w.decideRR(rev2, d2.ID, "confirmed", 0))

	// partial: reverses only the amount asked
	tx3 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 900)
	d3 := tu.Must(w.open(w.a, tx3.ID))
	tu.Must(w.resolve(rev1, d3.ID, domain.OutcomeDismissed, 0, "no"))
	tu.Must(w.Svc.RequestReReview(w.Ctx, w.a, w.net.ID, service.ReReviewRequestInput{DisputeID: d3.ID, Reason: "half", IdempotencyKey: w.Key()}))
	tu.Must(0, w.decideRR(rev2, d3.ID, "partially_upheld", 300))
	if got := w.txStatus(tx3.ID); got.ReversedTotal != 300 || got.Status != domain.TxCompleted {
		t.Errorf("tx3 = %+v", got)
	}
}

func TestReReviewWindow(t *testing.T) {
	w := newWorld(t, nil)
	rev, _ := w.reviewerAs("rev@example.org")
	d := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "no"))
	w.Clock.Advance((service.ReReviewWindowDays + 1) * 24 * time.Hour)
	wantCode(t, w.rereview(w.a, d.ID, "late"), domain.ErrInvalidState)

	d2 := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100).ID))
	tu.Must(w.resolve(rev, d2.ID, domain.OutcomeDismissed, 0, "no"))
	w.Clock.Advance((service.ReReviewWindowDays - 1) * 24 * time.Hour)
	if err := w.rereview(w.a, d2.ID, "in time"); err != nil {
		t.Fatalf("within the window: %v", err)
	}
}

func TestReReviewBlockedByLimitChangesNothing(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	rev1, _ := w.reviewerAs("rev1@example.org")
	rev2, _ := w.reviewerAs("rev2@example.org")
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	w.Trade(w.net.ID, w.b, c, w.bM, cM, 5000) // b spends it all: -2500
	d := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.resolve(rev1, d.ID, domain.OutcomeDismissed, 0, "no"))
	tu.Must(w.Svc.RequestReReview(w.Ctx, w.a, w.net.ID, service.ReReviewRequestInput{DisputeID: d.ID, Reason: "pay me", IdempotencyKey: w.Key()}))
	wantCode(t, w.decideRR(rev2, d.ID, "upheld", 0), domain.ErrInsufficientCredit)
	v := tu.Must(w.Svc.GetDispute(w.Ctx, w.admin, w.net.ID, d.ID))
	if v.ReReview.Status != "requested" || v.Tx.ReversedTotal != 0 {
		t.Errorf("failed decision changed state: %+v %+v", v.ReReview, v.Tx)
	}
}
