package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / API-022: context survives cached/durable replay, remains exact
// through reviewer pagination, and grants no handling or subject access.
func TestReportDisputeContextAPIAndReplay(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	e := w.env
	listing := tu.Must(e.Svc.CreateListing(e.Ctx, w.bob, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Related listing", IdempotencyKey: e.Key(),
	}))
	trade := tu.Must(e.Svc.ProposeTransaction(e.Ctx, w.alice, w.net.ID, service.ProposeInput{
		CounterpartyID: w.bobM.ID, Direction: service.ProposerPays, Amount: 100, ListingID: listing.ID, IdempotencyKey: e.Key(),
	}))
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, w.bob, w.net.ID, service.TxRef{TxID: trade.ID, IdempotencyKey: e.Key()}))
	open := func(actor domain.Actor) *domain.Dispute {
		return tu.Must(e.Svc.OpenDispute(e.Ctx, actor, w.net.ID, service.OpenDisputeInput{
			TxID: trade.ID, Category: "other", Statement: "Ordinary case", IdempotencyKey: e.Key(),
		}))
	}
	first := open(w.alice)
	tu.Must(e.Svc.WithdrawDispute(e.Ctx, w.alice, w.net.ID, first.ID))
	second := open(w.bob)
	const privateNote = "Private member report note"
	body := m{"target_type": "member", "target_id": w.bobM.ID, "category": "other", "description": privateNote, "dispute_id": first.ID}
	key := w.key()
	filed := w.post(w.aliceT, "/reports", body, key)
	wantStatus(t, filed, 201)
	memberReport := decode[map[string]any](t, filed)
	memberID := memberReport["id"].(string)
	if memberReport["dispute_id"] != first.ID || memberReport["related_dispute_url"] != "/api/v1/disputes/"+first.ID || memberReport["related_disputes_url"] != nil {
		t.Fatalf("wrong explicit case relation: %+v", memberReport)
	}
	listingResult := w.post(w.aliceT, "/reports", m{"target_type": "listing", "target_id": listing.ID, "category": "scam", "dispute_id": second.ID}, w.key())
	wantStatus(t, listingResult, 201)
	listingID := decode[map[string]any](t, listingResult)["id"].(string)
	transactionResult := w.post(w.aliceT, "/reports", m{"target_type": "transaction", "target_id": trade.ID, "category": "other"}, w.key())
	wantStatus(t, transactionResult, 201)
	transaction := decode[map[string]any](t, transactionResult)
	transactionID := transaction["id"].(string)
	if transaction["dispute_id"] != nil || transaction["related_dispute_url"] != nil || transaction["related_disputes_url"] != "/api/v1/disputes?transaction_id="+trade.ID {
		t.Fatal("context-free transaction report changed its established relation")
	}
	reviewer, reviewerM := e.Member(w.admin, w.net, "context-reviewer")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	token := w.token(reviewer, domain.ScopeRead, domain.ScopeWrite)
	for _, tc := range []struct{ caseID, contextReport string }{{first.ID, memberID}, {second.ID, listingID}} {
		path := "/disputes/" + tc.caseID + "/reports"
		page1 := w.get(token, path+"?limit=1")
		wantStatus(t, page1, 200)
		pg := decode[page](t, page1)
		if len(pg.Data) != 1 || pg.Data[0]["id"] != transactionID || len(pg.Data[0]) != 3 || pg.NextCursor == nil {
			t.Fatalf("wrong first context page: %+v", pg)
		}
		page2 := w.get(token, path+"?limit=1&cursor="+url.QueryEscape(*pg.NextCursor))
		wantStatus(t, page2, 200)
		last := decode[page](t, page2)
		if len(last.Data) != 1 || last.Data[0]["id"] != tc.contextReport || len(last.Data[0]) != 3 || last.NextCursor != nil {
			t.Fatalf("explicit case was not filtered before paging: %+v", last)
		}
		for _, hidden := range []string{privateNote, w.aliceM.ID, w.bobM.ID, "target_type", "target_id", "dispute_id", "description", "reporter_member_id"} {
			if strings.Contains(page1.Body.String()+page2.Body.String(), hidden) {
				t.Fatalf("reviewer context contained a private target or identity field: %q", hidden)
			}
		}
	}
	wantProblem(t, w.get(token, "/disputes/"+first.ID+"/reports?dispute_id="+second.ID), 400, "invalid-query")
	wantProblem(t, w.get(token, "/admin/reports"), 403, "forbidden")
	wantProblem(t, w.post(token, "/admin/reports/"+memberID+"/handle", m{}, w.key()), 403, "forbidden")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleReviewer))
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
	wantProblem(t, w.get(w.bobT, "/disputes/"+first.ID+"/reports"), 403, "forbidden")
	wantProblem(t, w.post(w.bobT, "/admin/reports/"+memberID+"/handle", m{}, w.key()), 404, "not-found")
	queue := w.get(w.bobT, "/admin/reports")
	wantStatus(t, queue, 200)
	if len(decode[page](t, queue).Data) != 0 {
		t.Fatal("case context disclosed a subject's reports through the full queue")
	}
	adminQueue := w.get(w.adminT, "/admin/reports")
	wantStatus(t, adminQueue, 200)
	if !strings.Contains(adminQueue.Body.String(), "/api/v1/disputes/"+first.ID) || !strings.Contains(adminQueue.Body.String(), privateNote) {
		t.Fatal("independent administrator lost the case relation or full details")
	}
	wantStatus(t, w.get(w.adminT, "/disputes/"+first.ID), 200)
	tu.Must(e.Svc.HandleReport(e.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: memberID, IdempotencyKey: e.Key()}))
	before := tu.Snapshot(t, e.Store.DB())
	cached := w.post(w.aliceT, "/reports", body, key)
	wantStatus(t, cached, 201)
	if cached.Body.String() != filed.Body.String() || cached.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatal("cached context report did not return its original response")
	}
	old := w.h
	w.h = api.NewHandler(e.Svc)
	api.CloseHandler(old)
	durable := w.post(w.aliceT, "/reports", body, key)
	wantStatus(t, durable, 201)
	replayed := decode[map[string]any](t, durable)
	if replayed["id"] != memberID || replayed["dispute_id"] != first.ID || replayed["status"] != "handled" {
		t.Fatal("durable replay lost its original identity/context or re-executed creation")
	}
	changedBody := m{"target_type": "member", "target_id": w.bobM.ID, "category": "other", "description": privateNote, "dispute_id": second.ID}
	wantProblem(t, w.post(w.aliceT, "/reports", changedBody, key), 422, "idempotency-key-reused")
	if changed := before.Changed(tu.Snapshot(t, e.Store.DB())); len(changed) != 0 {
		t.Fatalf("cached/durable replay changed durable state: %v", changed)
	}
}
