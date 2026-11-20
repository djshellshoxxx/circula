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

// SEC-025 / FR-024 / PER-003: use the dedicated, current-authority projection;
// never return the administrator report DTO to a dispute reviewer.
func TestDisputeReportsAPIProjectionAndPagination(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	e := w.env
	reviewer, reviewerM := e.Member(w.admin, w.net, "reviewer")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	token := w.token(reviewer, domain.ScopeRead, domain.ScopeWrite)
	trade := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	dispute := tu.Must(e.Svc.OpenDispute(e.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Ordinary case", IdempotencyKey: e.Key(),
	}))
	report := func(actor domain.Actor, txID, category, note string) *service.Report {
		t.Helper()
		return tu.Must(e.Svc.CreateReport(e.Ctx, actor, w.net.ID, service.ReportInput{
			TargetType: service.ReportTransaction, TargetID: txID, Category: category, Note: note, IdempotencyKey: e.Key(),
		}))
	}
	const privateNote = "Reporter Alice asked for a private conversation"
	const privateHandling = "Administrator handling details about Alice"
	older := report(w.alice, trade.ID, "other", privateNote)
	tu.Must(e.Svc.HandleReport(e.Ctx, w.admin, w.net.ID, service.HandleReportInput{
		ReportID: older.ID, Note: privateHandling, IdempotencyKey: e.Key(),
	}))
	newer := report(w.bob, trade.ID, "harassment", "Reporter Bob's private details")
	otherTrade := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 200)
	unrelated := report(w.alice, otherTrade.ID, "scam", "Unrelated report")
	otherDispute := tu.Must(e.Svc.OpenDispute(e.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: otherTrade.ID, Category: "other", Statement: "Other case", IdempotencyKey: e.Key(),
	}))
	path := "/disputes/" + dispute.ID + "/reports"
	detail := w.get(token, "/disputes/"+dispute.ID)
	wantStatus(t, detail, 200)
	if got := decode[map[string]any](t, detail)["transaction_reports_url"]; got != "/api/v1"+path {
		t.Fatalf("independent reviewer omitted the capability link: %v", got)
	}
	before := tu.Snapshot(t, e.Store.DB())
	first := w.get(token, path+"?limit=1")
	wantStatus(t, first, 200)
	pg := decode[page](t, first)
	if len(pg.Data) != 1 || pg.Data[0]["id"] != newer.ID || pg.Data[0]["category"] != "harassment" || pg.Data[0]["status"] != "open" || len(pg.Data[0]) != 3 || pg.NextCursor == nil {
		t.Fatalf("wrong redacted first page: %+v", pg)
	}
	cursor := url.QueryEscape(*pg.NextCursor)
	second := w.get(token, path+"?limit=1&cursor="+cursor)
	wantStatus(t, second, 200)
	last := decode[page](t, second)
	if len(last.Data) != 1 || last.Data[0]["id"] != older.ID || last.Data[0]["status"] != "handled" || len(last.Data[0]) != 3 || last.NextCursor != nil {
		t.Fatalf("wrong redacted last page: %+v", last)
	}
	for _, response := range []string{first.Body.String(), second.Body.String()} {
		for _, hidden := range []string{privateNote, privateHandling, w.aliceM.ID, w.bobM.ID, w.adminM.ID, unrelated.ID,
			"reporter_member_id", "reporter_display_name", "description", "handling_note", "handled_by", "created_at", "handled_at"} {
			if strings.Contains(response, hidden) {
				t.Fatalf("reviewer projection contained private or unrelated field/content: %q", hidden)
			}
		}
	}
	if changed := before.Changed(tu.Snapshot(t, e.Store.DB())); len(changed) != 0 {
		t.Fatalf("reading report context changed durable state: %v", changed)
	}
	wantProblem(t, w.get(token, "/disputes/"+otherDispute.ID+"/reports?limit=1&cursor="+cursor), 400, "invalid-cursor")
	wantProblem(t, w.get(token, "/disputes?limit=1&cursor="+cursor), 400, "invalid-cursor")
	wantProblem(t, w.get(token, path+"?transaction_id="+otherTrade.ID), 400, "invalid-query")
	general := w.get(token, "/disputes?limit=1")
	wantStatus(t, general, 200)
	for _, hidden := range []string{older.ID, newer.ID, "transaction_reports_url", privateNote} {
		if strings.Contains(general.Body.String(), hidden) {
			t.Fatal("general dispute pagination included report context")
		}
	}
	wantProblem(t, w.get(token, "/admin/reports"), 403, "forbidden")
	wantProblem(t, w.post(token, "/admin/reports/"+newer.ID+"/handle", m{"note": "Reviewer request"}, w.key()), 403, "forbidden")
	tu.Must(0, e.Svc.RevokeRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	wantProblem(t, w.get(token, path+"?limit=1&cursor="+cursor), 403, "forbidden")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	wantStatus(t, w.get(token, path+"?limit=1&cursor="+cursor), 200)
	for _, party := range []struct {
		token string
		id    string
	}{{w.aliceT, w.aliceM.ID}, {w.bobT, w.bobM.ID}} {
		for _, role := range []domain.Role{domain.RoleReviewer, domain.RoleAdmin} {
			tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, party.id, role))
			refused := w.get(party.token, path)
			wantProblem(t, refused, 403, "forbidden")
			view := w.get(party.token, "/disputes/"+dispute.ID)
			wantStatus(t, view, 200)
			for _, hidden := range []string{"transaction_reports_url", older.ID, newer.ID, privateNote, privateHandling} {
				if strings.Contains(view.Body.String(), hidden) || strings.Contains(refused.Body.String(), hidden) {
					t.Fatal("party received report context after gaining a role")
				}
			}
		}
	}
	full := w.get(w.adminT, "/admin/reports")
	wantStatus(t, full, 200)
	if !strings.Contains(full.Body.String(), privateNote) || !strings.Contains(full.Body.String(), privateHandling) {
		t.Fatal("independent administrator lost full report details")
	}
	otherNet, _ := e.Network(reviewer, "reviewer-other-net", nil)
	wantProblem(t, w.do(req{method: "GET", path: "/api/v1" + path, token: token, hdr: map[string]string{"X-Network": otherNet.ID}}), 404, "not-found")
}
