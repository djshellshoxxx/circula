package service_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025: either member party may report the transaction. The report and its
// replay preserve transaction terms, history, ledger and dispute state.
func TestTransactionReportsPreserveTransactionAndReplay(t *testing.T) {
	w := newWorld(t, nil)
	tx := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	for _, by := range []domain.Actor{w.a, w.b} {
		before := tu.Snapshot(t, w.Store.DB())
		prior := tu.Must(w.Svc.GetTransaction(w.Ctx, by, w.net.ID, tx.ID))
		in := service.ReportInput{TargetType: service.ReportTransaction, TargetID: tx.ID,
			Category: "other", Note: "Ordinary transaction report", IdempotencyKey: w.Key()}
		filed := tu.Must(w.Svc.CreateReport(w.Ctx, by, w.net.ID, in))
		after := tu.Snapshot(t, w.Store.DB())
		if filed.TargetType != service.ReportTransaction || filed.TargetID != tx.ID || filed.Status != "open" {
			t.Fatalf("wrong report target: %+v", filed)
		}
		if before.Ledger != after.Ledger || after.AuditEvents != before.AuditEvents+1 {
			t.Fatal("filing a transaction report changed the ledger or missed its audit")
		}
		if got := tu.Must(w.Svc.GetTransaction(w.Ctx, by, w.net.ID, tx.ID)); !reflect.DeepEqual(prior, got) {
			t.Fatal("filing a report changed the transaction or its history")
		}
		if disputes := tu.Must(w.Svc.FindDisputes(w.Ctx, by, w.net.ID, service.DisputeQuery{TxID: tx.ID})); len(disputes) != 0 {
			t.Fatal("filing a report created a dispute")
		}
		replayed := tu.Must(w.Svc.CreateReport(w.Ctx, by, w.net.ID, in))
		if replayed.ID != filed.ID || len(after.Changed(tu.Snapshot(t, w.Store.DB()))) != 0 {
			t.Fatal("durable report replay changed state")
		}
		in.IdempotencyKey = w.Key()
		_, err := w.Svc.CreateReport(w.Ctx, by, w.net.ID, in)
		wantCode(t, err, domain.ErrConflict)
	}
}

// SEC-025 imposes party membership, not a dispute filing deadline or posting
// state. A two-member reversal also has an unambiguous other member.
func TestTransactionReportsDoNotInheritDisputeRestrictions(t *testing.T) {
	for _, state := range []string{"proposed", "cancelled", "completed_after_window", "reversal"} {
		t.Run(state, func(t *testing.T) {
			w := newWorld(t, nil)
			tx := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
			switch state {
			case "cancelled":
				tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: tx.ID, IdempotencyKey: w.Key()}))
			case "completed_after_window":
				tu.Must(w.accept(w.b, tx.ID))
				w.Clock.Advance(365 * 24 * time.Hour)
			case "reversal":
				tu.Must(w.accept(w.b, tx.ID))
				tx = tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{
					TxID: tx.ID, Amount: 100, Reason: "Ordinary reversal", IdempotencyKey: w.Key(),
				}))
			}
			tu.Must(w.report(w.a, service.ReportTransaction, tx.ID, "other", "Report fixture"))
		})
	}
}

func TestTransactionReportsRequireRelatedTwoMemberTarget(t *testing.T) {
	w := newWorld(t, nil)
	tx := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	other, _ := w.Member(w.admin, w.net, "unrelated@example.org")
	for _, actor := range []domain.Actor{other, w.admin} {
		before := tu.Snapshot(t, w.Store.DB())
		_, err := w.report(actor, service.ReportTransaction, tx.ID, "other", "Unrelated fixture")
		wantCode(t, err, domain.ErrNotFound)
		if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
			t.Fatalf("unrelated report refusal changed state: %v", changed)
		}
	}
	otherNet, _ := w.Network(w.a, "other-net", nil)
	before := tu.Snapshot(t, w.Store.DB())
	_, err := w.Svc.CreateReport(w.Ctx, w.a, otherNet.ID, service.ReportInput{
		TargetType: service.ReportTransaction, TargetID: tx.ID, Category: "other", IdempotencyKey: w.Key(),
	})
	wantCode(t, err, domain.ErrNotFound)
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("cross-network report refusal changed state: %v", changed)
	}
	adjustment := tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{
		MemberID: w.aM.ID, Amount: 1, Direction: service.AdjustCredit, Reason: "System-account fixture", IdempotencyKey: w.Key(),
	}))
	before = tu.Snapshot(t, w.Store.DB())
	_, err = w.report(w.a, service.ReportTransaction, adjustment.ID, "other", "Unsupported subject fixture")
	wantCode(t, err, domain.ErrValidation)
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("undefined report subject refusal changed state: %v", changed)
	}
}
