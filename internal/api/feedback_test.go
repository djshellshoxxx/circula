package api_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-076 / FR-078 over HTTP.
func TestDisputeResponseAndReReviewAPI(t *testing.T) {
	w := newWorld(t)
	rev2, rev2M := w.env.Member(w.admin, w.net, "rev2")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, rev2M.ID, domain.RoleReviewer))
	rev2T := w.token(rev2, "read", "write")

	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := decode[map[string]any](t, w.post(w.aliceT, "/disputes", m{"transaction_id": tx.ID, "category": "quality", "statement": "broken"}, w.key()))["id"].(string)

	// response: other party only, bounded, once
	wantProblem(t, w.post(w.aliceT, "/disputes/"+d+"/response", m{"response": "mine"}, w.key()), 403, "forbidden")
	wantProblem(t, w.post(w.bobT, "/disputes/"+d+"/response", m{"response": " "}, w.key()), 422, "validation-failed")
	wantProblem(t, w.post(w.bobT, "/disputes/"+d+"/response", m{"response": "x"}, ""), 400, "idempotency-key-required")
	rec := w.post(w.bobT, "/disputes/"+d+"/response", m{"response": "It worked when delivered."}, w.key())
	wantStatus(t, rec, 201)
	if got := decode[map[string]any](t, rec); got["response"] != "It worked when delivered." || got["responder_member_id"] != w.bobM.ID {
		t.Fatalf("response = %v", got)
	}
	wantProblem(t, w.post(w.bobT, "/disputes/"+d+"/response", m{"response": "again"}, w.key()), 409, "invalid-transaction-state")
	for _, tok := range []string{w.aliceT, w.adminT, w.bobT} {
		det := decode[map[string]any](t, w.get(tok, "/disputes/"+d))
		r, _ := det["response"].(map[string]any)
		if r["response"] != "It worked when delivered." {
			t.Fatalf("detail response = %v", det["response"])
		}
		if det["re_review"] != nil {
			t.Fatalf("re_review = %v", det["re_review"])
		}
	}

	// re-review needs a decision first
	wantProblem(t, w.post(w.aliceT, "/disputes/"+d+"/re-review", m{"reason": "early"}, w.key()), 409, "invalid-transaction-state")
	wantStatus(t, w.post(w.adminT, "/disputes/"+d+"/resolve", m{"outcome": "no_action", "reason": "no evidence"}, w.key()), 200)
	wantProblem(t, w.post(w.bobT, "/disputes/"+d+"/response", m{"response": "late"}, w.key()), 409, "invalid-transaction-state")

	wantProblem(t, w.post(w.aliceT, "/disputes/"+d+"/re-review", m{"reason": ""}, w.key()), 422, "validation-failed")
	rec = w.post(w.aliceT, "/disputes/"+d+"/re-review", m{"reason": "I have photos"}, w.key())
	wantStatus(t, rec, 201)
	if got := decode[map[string]any](t, rec); got["status"] != "requested" || got["requested_by"] != w.aliceM.ID {
		t.Fatalf("re-review = %v", got)
	}
	wantProblem(t, w.post(w.bobT, "/disputes/"+d+"/re-review", m{"reason": "me too"}, w.key()), 409, "invalid-transaction-state")

	// original reviewer (admin) and a party cannot decide; a different reviewer can
	dec := m{"outcome": "upheld", "reason": "photos are convincing"}
	wantProblem(t, w.post(w.adminT, "/disputes/"+d+"/re-review/decision", dec, w.key()), 403, "forbidden")
	wantProblem(t, w.post(w.aliceT, "/disputes/"+d+"/re-review/decision", dec, w.key()), 403, "forbidden")
	wantProblem(t, w.post(rev2T, "/disputes/"+d+"/re-review/decision", m{"outcome": "bogus", "reason": "x"}, w.key()), 422, "validation-failed")
	wantProblem(t, w.post(rev2T, "/disputes/"+d+"/re-review/decision", m{"outcome": "partially_upheld", "reason": "x", "amount_minor": "99999"}, w.key()), 422, "validation-failed")
	rec = w.post(rev2T, "/disputes/"+d+"/re-review/decision", dec, w.key())
	wantStatus(t, rec, 200)
	if got := decode[map[string]any](t, rec); got["status"] != "decided" || got["outcome"] != "upheld" || got["reversal_amount_minor"] != "1000" {
		t.Fatalf("decision = %v", got)
	}
	det := decode[map[string]any](t, w.get(w.bobT, "/disputes/"+d))
	if det["outcome"] != "dismissed" || det["re_review"].(map[string]any)["outcome"] != "upheld" {
		t.Fatalf("original must stay dismissed with the linked re-review: %v", det)
	}
	if a := w.acct(w.aliceT); a.BalanceMinor != "0" {
		t.Fatalf("alice after re-review = %+v", a)
	}
}

// FR-024 over HTTP.
func TestReportsAPI(t *testing.T) {
	w := newWorld(t)
	rec := w.post(w.bobT, "/listings", m{"kind": "offer", "title": "Bike repair"}, w.key())
	wantStatus(t, rec, 201)
	lid := decode[map[string]any](t, rec)["id"].(string)

	body := m{"target_type": "listing", "target_id": lid, "category": "spam", "description": "fake"}
	wantProblem(t, w.post(w.aliceT, "/reports", body, ""), 400, "idempotency-key-required")
	wantProblem(t, w.post(w.aliceT, "/reports", m{"target_type": "listing", "target_id": lid, "category": "bogus"}, w.key()), 422, "validation-failed")
	wantProblem(t, w.post(w.bobT, "/reports", body, w.key()), 422, "validation-failed") // own listing
	wantProblem(t, w.post(w.aliceT, "/reports", m{"target_type": "listing", "target_id": "nope", "category": "spam"}, w.key()), 404, "not-found")
	wantProblem(t, w.post(w.aliceT, "/reports", m{"target_type": "listing", "target_id": lid, "category": "spam", "bogus": 1}, w.key()), 400, "invalid-body")
	rec = w.post(w.aliceT, "/reports", body, w.key())
	wantStatus(t, rec, 201)
	id := decode[map[string]any](t, rec)["id"].(string)
	wantProblem(t, w.post(w.aliceT, "/reports", body, w.key()), 409, "conflict")
	wantStatus(t, w.post(w.aliceT, "/reports", m{"target_type": "member", "target_id": w.bobM.ID, "category": "harassment"}, w.key()), 201)

	// queue: administrators only, with the reporter named
	wantProblem(t, w.get(w.aliceT, "/admin/reports"), 403, "forbidden")
	wantProblem(t, w.get(w.bobT, "/admin/reports"), 403, "forbidden")
	wantProblem(t, w.get(w.adminT, "/admin/reports?status=weird"), 400, "invalid-query")
	pg := decode[page](t, w.get(w.adminT, "/admin/reports?status=open&limit=1"))
	if len(pg.Data) != 1 || pg.NextCursor == nil {
		t.Fatalf("page 1 = %+v", pg)
	}
	pg2 := decode[page](t, w.get(w.adminT, "/admin/reports?status=open&limit=1&cursor="+*pg.NextCursor))
	if len(pg2.Data) != 1 || pg2.Data[0]["id"] == pg.Data[0]["id"] || pg2.Data[0]["reporter_display_name"] != "alice" {
		t.Fatalf("page 2 = %+v", pg2)
	}

	wantProblem(t, w.post(w.aliceT, "/admin/reports/"+id+"/handle", m{}, w.key()), 403, "forbidden")
	wantProblem(t, w.post(w.adminT, "/admin/reports/nope/handle", m{}, w.key()), 404, "not-found")
	rec = w.post(w.adminT, "/admin/reports/"+id+"/handle", m{"note": "listing removed"}, w.key())
	wantStatus(t, rec, 200)
	if got := decode[map[string]any](t, rec); got["status"] != "handled" || got["handled_by"] != w.adminM.ID {
		t.Fatalf("handled = %v", got)
	}
	wantProblem(t, w.post(w.adminT, "/admin/reports/"+id+"/handle", m{}, w.key()), 409, "invalid-transaction-state")
	if h := decode[page](t, w.get(w.adminT, "/admin/reports?status=handled")); len(h.Data) != 1 {
		t.Fatalf("handled queue = %+v", h)
	}
	// reporter is told; the reported member is not
	nt := decode[page](t, w.get(w.aliceT, "/notifications"))
	if len(nt.Data) != 1 || nt.Data[0]["kind"] != "report.handled" {
		t.Fatalf("alice inbox = %+v", nt)
	}
	if bt := decode[page](t, w.get(w.bobT, "/notifications")); len(bt.Data) != 0 {
		t.Fatalf("bob inbox = %+v", bt)
	}
}

// FR-038 over HTTP.
func TestNotificationsAPI(t *testing.T) {
	w := newWorld(t)
	for i := 0; i < 3; i++ {
		wantStatus(t, w.post(w.aliceT, "/transactions", m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "100"}, w.key()), 201)
	}
	cnt := func(tok string) float64 {
		return decode[map[string]any](t, w.get(tok, "/notifications/unread-count"))["unread"].(float64)
	}
	if cnt(w.bobT) != 3 || cnt(w.aliceT) != 0 {
		t.Fatalf("unread bob=%v alice=%v", cnt(w.bobT), cnt(w.aliceT))
	}
	pg := decode[page](t, w.get(w.bobT, "/notifications?limit=2"))
	if len(pg.Data) != 2 || pg.NextCursor == nil || pg.Data[0]["kind"] != "trade.proposed" || pg.Data[0]["read"] != false {
		t.Fatalf("page = %+v", pg)
	}
	if !strings.HasPrefix(pg.Data[0]["link"].(string), "/trades/") {
		t.Errorf("link = %v", pg.Data[0]["link"])
	}
	rest := decode[page](t, w.get(w.bobT, "/notifications?limit=2&cursor="+*pg.NextCursor))
	if len(rest.Data) != 1 || rest.NextCursor != nil {
		t.Fatalf("rest = %+v", rest)
	}
	wantProblem(t, w.get(w.bobT, "/notifications?unread=maybe"), 400, "invalid-query")

	first := pg.Data[0]["id"].(string)
	wantProblem(t, w.post(w.bobT, "/notifications/read", m{}, ""), 422, "validation-failed")
	wantProblem(t, w.post(w.bobT, "/notifications/read", m{"ids": []string{first}, "all": true}, ""), 422, "validation-failed")
	// another member's id is ignored
	if got := decode[map[string]any](t, w.post(w.aliceT, "/notifications/read", m{"ids": []string{first}}, ""))["marked"]; got != float64(0) {
		t.Fatalf("alice marked bob's: %v", got)
	}
	if got := decode[map[string]any](t, w.post(w.bobT, "/notifications/read", m{"ids": []string{first}}, ""))["marked"]; got != float64(1) {
		t.Fatalf("marked = %v", got)
	}
	if cnt(w.bobT) != 2 {
		t.Fatalf("unread = %v", cnt(w.bobT))
	}
	un := decode[page](t, w.get(w.bobT, "/notifications?unread=true"))
	if len(un.Data) != 2 {
		t.Fatalf("unread list = %+v", un)
	}
	if got := decode[map[string]any](t, w.post(w.bobT, "/notifications/read", m{"all": true}, ""))["marked"]; got != float64(2) {
		t.Fatalf("mark all = %v", got)
	}
	if cnt(w.bobT) != 0 {
		t.Fatal("still unread")
	}
	// auth required; read-only tokens may list but not mark
	wantProblem(t, w.get("", "/notifications"), 401, "unauthenticated")
	ro := w.token(w.bob, "read")
	wantStatus(t, w.get(ro, "/notifications"), 200)
	wantProblem(t, w.post(ro, "/notifications/read", m{"all": true}, ""), 403, "forbidden")
}
