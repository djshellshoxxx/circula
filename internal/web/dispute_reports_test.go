package web

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / FR-024 / PER-003: independent reviewer pages contain only
// structured report summaries, while party pages expose no report link.
func TestDisputeReportsHTMLProjectionAndPagination(t *testing.T) {
	w := newWorld(t)
	e := w.env
	_, reviewerM := e.Member(w.admin, w.net, "reviewer@example.org")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	trade := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	dispute := tu.Must(e.Svc.OpenDispute(e.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Ordinary dispute statement", IdempotencyKey: e.Key(),
	}))
	const privateNote = "Alice asked for a confidential personal conversation"
	const privateHandling = "Private follow-up details for Alice"
	var ids []string
	for i := 0; i <= perPage; i++ {
		if i > 0 && i%service.MaxReportsPerDay == 0 {
			e.Clock.Advance(24*time.Hour + time.Second)
		}
		report := tu.Must(e.Svc.CreateReport(e.Ctx, w.alice, w.net.ID, service.ReportInput{
			TargetType: service.ReportTransaction, TargetID: trade.ID, Category: "other", Note: privateNote, IdempotencyKey: e.Key(),
		}))
		ids = append(ids, report.ID)
		if i < perPage {
			tu.Must(e.Svc.HandleReport(e.Ctx, w.admin, w.net.ID, service.HandleReportInput{
				ReportID: report.ID, Note: privateHandling, IdempotencyKey: e.Key(),
			}))
		}
	}
	otherTx := e.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 200)
	unrelated := tu.Must(e.Svc.CreateReport(e.Ctx, w.alice, w.net.ID, service.ReportInput{
		TargetType: service.ReportTransaction, TargetID: otherTx.ID, Category: "scam", Note: "Unrelated report", IdempotencyKey: e.Key(),
	}))
	reader := w.browser(t)
	reader.login("reviewer@example.org")
	path := w.base() + "/disputes/" + dispute.ID + "/reports"
	detail := reader.get(w.base() + "/disputes/" + dispute.ID)
	reader.mustContain(detail, path, "Read transaction report summaries")
	before := tu.Snapshot(t, e.Store.DB())
	first := reader.get(path)
	reader.mustContain(first, "Transaction reports", "Reporter identities and report notes are available only to independent administrators.",
		ids[len(ids)-1], path+"?after="+ids[1])
	last := reader.get(path + "?after=" + ids[1])
	reader.mustContain(last, ids[0], "Back to dispute")
	for _, page := range []*pg{first, last} {
		if page.Status != 200 {
			t.Fatalf("report projection status = %d", page.Status)
		}
		for _, hidden := range []string{privateNote, privateHandling, w.aliceM.ID, w.bobM.ID, w.adminM.ID, unrelated.ID,
			"alice@example.org", "bob@example.org", "/admin/reports/"} {
			if strings.Contains(page.Body, hidden) {
				t.Fatalf("reviewer HTML contained private or unrelated report content: %q", hidden)
			}
		}
	}
	if strings.Contains(first.Body, ids[0]) || strings.Contains(last.Body, path+"?after=") {
		t.Fatal("report context pagination skipped or repeated its boundary")
	}
	if changed := before.Changed(tu.Snapshot(t, e.Store.DB())); len(changed) != 0 {
		t.Fatalf("report projection changed durable state: %v", changed)
	}
	tu.Must(0, e.Svc.RevokeRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	if refused := reader.get(path + "?after=" + ids[1]); refused.Status != 403 {
		t.Fatalf("revoked reviewer resumed report pagination: %d", refused.Status)
	}
	for _, party := range []struct {
		login string
		id    string
	}{{"alice@example.org", w.aliceM.ID}, {"bob@example.org", w.bobM.ID}} {
		tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, party.id, domain.RoleReviewer))
		tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, party.id, domain.RoleAdmin))
		browser := w.browser(t)
		browser.login(party.login)
		view := browser.get(w.base() + "/disputes/" + dispute.ID)
		if view.Status != 200 || strings.Contains(view.Body, path) || strings.Contains(view.Body, "Read transaction report summaries") {
			t.Fatal("party dispute page gained a report-context link from its roles")
		}
		refused := browser.get(path)
		if refused.Status != 403 || strings.Contains(refused.Body, privateNote) || strings.Contains(refused.Body, ids[0]) {
			t.Fatal("party with review/admin roles gained the report projection")
		}
	}
	administrator := w.browser(t)
	administrator.login("admin@example.org")
	administrator.mustContain(administrator.get(w.base()+"/admin/reports?status=all"), privateNote, privateHandling)
}
