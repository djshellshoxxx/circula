package service_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / FR-024 / PER-003: the independent review projection is read-only,
// uses the exact disputed transaction, and does not grant report handling.
func TestDisputeReportsSameTransactionAndReadOnly(t *testing.T) {
	w := newWorld(t, nil)
	reviewer, _ := w.reviewerAs("reviewer@example.org")
	trade := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	dispute := tu.Must(w.open(w.a, trade.ID))
	old := tu.Must(w.report(w.a, service.ReportTransaction, trade.ID, "other", "Reporter details stay private"))
	tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{
		ReportID: old, Note: "Handling details stay private", IdempotencyKey: w.Key(),
	}))
	otherTrade := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 200))
	tu.Must(w.report(w.a, service.ReportTransaction, otherTrade.ID, "scam", "Unrelated transaction"))
	tu.Must(w.report(w.a, service.ReportMember, w.bM.ID, "spam", "Member report"))
	tu.Must(w.report(w.a, service.ReportListing, w.listing(w.b, "Listing"), "spam", "Listing report"))
	newer := tu.Must(w.report(w.b, service.ReportTransaction, trade.ID, "harassment", "Other reporter details stay private"))
	want := []service.DisputeReport{{ID: newer, Category: "harassment", Status: "open"}, {ID: old, Category: "other", Status: "handled"}}
	before := tu.Snapshot(t, w.Store.DB())
	for _, actor := range []domain.Actor{reviewer, w.admin, domain.SystemActor("report-context-fixture")} {
		var got []service.DisputeReport
		after := ""
		for _, expected := range want {
			page := tu.Must(w.Svc.ListDisputeReports(w.Ctx, actor, w.net.ID, dispute.ID, service.Page{AfterID: after, Limit: 1}))
			if len(page) != 1 || page[0] != expected {
				t.Fatalf("related report page after %q = %+v, want %+v", after, page, expected)
			}
			got = append(got, page...)
			after = page[0].ID
		}
		if tail := tu.Must(w.Svc.ListDisputeReports(w.Ctx, actor, w.net.ID, dispute.ID, service.Page{AfterID: after, Limit: 1})); len(tail) != 0 {
			t.Fatal("report context pagination did not end")
		}
		if !reflect.DeepEqual(got, want) || !tu.Must(w.Svc.GetDispute(w.Ctx, actor, w.net.ID, dispute.ID)).CanReadTransactionReports {
			t.Fatal("independent authority lost its report context")
		}
	}
	_, err := w.Svc.ListReports(w.Ctx, reviewer, w.net.ID, "", service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.HandleReport(w.Ctx, reviewer, w.net.ID, service.HandleReportInput{ReportID: newer, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden)
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("report projection or refused handling changed durable state: %v", changed)
	}
}

func TestDisputeReportsRecheckRolesAndExcludeParties(t *testing.T) {
	w := newWorld(t, nil)
	reviewer, reviewerM := w.reviewerAs("reviewer@example.org")
	trade := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	dispute := tu.Must(w.open(w.a, trade.ID))
	tu.Must(w.report(w.a, service.ReportTransaction, trade.ID, "other", "Private report"))
	for _, party := range []struct {
		actor domain.Actor
		id    string
	}{{w.a, w.aM.ID}, {w.b, w.bM.ID}} {
		for _, role := range []domain.Role{domain.RoleReviewer, domain.RoleAdmin} {
			tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, party.id, role))
			before := tu.Snapshot(t, w.Store.DB())
			_, err := w.Svc.ListDisputeReports(w.Ctx, party.actor, w.net.ID, dispute.ID, service.Page{})
			wantCode(t, err, domain.ErrForbidden)
			view := tu.Must(w.Svc.GetDispute(w.Ctx, party.actor, w.net.ID, dispute.ID))
			if view.CanReadTransactionReports {
				t.Fatal("party gained report context through a reviewer or admin role")
			}
			if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
				t.Fatalf("party refusal changed durable state: %v", changed)
			}
		}
	}
	tu.Must(w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, dispute.ID, service.Page{}))
	tu.Must(0, w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	_, err := w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, dispute.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, reviewerM.ID, "Ordinary membership fixture"))
	_, err = w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, dispute.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	otherNet, _ := w.Network(w.admin, "other-report-context", nil)
	_, err = w.Svc.ListDisputeReports(w.Ctx, w.admin, otherNet.ID, dispute.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
}

func TestDisputeReportsExcludeReversalParticipant(t *testing.T) {
	w := newWorld(t, nil)
	trade := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{
		TxID: trade.ID, Amount: 20, Reason: "Ordinary partial correction", IdempotencyKey: w.Key(),
	}))
	dispute := tu.Must(w.open(w.a, trade.ID))
	tu.Must(w.report(w.a, service.ReportTransaction, trade.ID, "other", "Private report"))
	_, err := w.Svc.ListDisputeReports(w.Ctx, w.admin, w.net.ID, dispute.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	if view := tu.Must(w.Svc.GetDispute(w.Ctx, w.admin, w.net.ID, dispute.ID)); view.CanReadTransactionReports {
		t.Fatal("reversal participant received a report-context capability link")
	}
	reviewer, _ := w.reviewerAs("independent@example.org")
	if got := tu.Must(w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, dispute.ID, service.Page{})); len(got) != 1 {
		t.Fatal("independent reviewer lost the report")
	}
}

func TestDisputeReportsDoNotProjectUnvalidatedCategory(t *testing.T) {
	w := newWorld(t, nil)
	reviewer, _ := w.reviewerAs("reviewer@example.org")
	trade := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	dispute := tu.Must(w.open(w.a, trade.ID))
	id := tu.Must(w.report(w.a, service.ReportTransaction, trade.ID, "other", "Private report"))
	const legacyCategory = "legacy free-text category"
	if _, err := w.Store.DB().Exec("UPDATE reports SET category=? WHERE id=?", legacyCategory, id); err != nil {
		t.Fatal(err)
	}
	before := tu.Snapshot(t, w.Store.DB())
	rows, err := w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, dispute.ID, service.Page{})
	if domain.CodeOf(err) != domain.CodeInternal || rows != nil || strings.Contains(err.Error(), legacyCategory) || strings.Contains(err.Error(), id) {
		t.Fatalf("unvalidated stored text reached the projection: rows=%+v err=%v", rows, err)
	}
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("unavailable projection changed durable state: %v", changed)
	}
}
