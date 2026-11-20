package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestReportSubjectPrivacyInHTMLQueue(t *testing.T) {
	w := newWorld(t)
	listing := testutil.Must(w.env.Svc.CreateListing(w.env.Ctx, w.bob, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Subject's listing", IdempotencyKey: w.env.Key(),
	}))
	var reports []string
	for _, target := range []struct{ typ, id string }{{service.ReportMember, w.bobM.ID}, {service.ReportListing, listing.ID}} {
		r := testutil.Must(w.env.Svc.CreateReport(w.env.Ctx, w.alice, w.net.ID, service.ReportInput{
			TargetType: target.typ, TargetID: target.id, Category: "other", Note: "Private HTML report fixture", IdempotencyKey: w.env.Key(),
		}))
		reports = append(reports, r.ID)
	}
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
	subject, independent := w.browser(t), w.browser(t)
	subject.login("bob@example.org")
	independent.login("admin@example.org")
	path := w.base() + "/admin/reports"
	queue := subject.get(path)
	if queue.Status != 200 {
		t.Fatalf("subject report queue status = %d", queue.Status)
	}
	for _, hidden := range append(reports, w.aliceM.ID, w.aliceM.DisplayName, "Private HTML report fixture") {
		if strings.Contains(queue.Body, hidden) {
			t.Fatal("subject HTML queue contained private report information")
		}
	}
	visible := independent.get(path)
	independent.mustContain(visible, reports[0], reports[1], w.aliceM.DisplayName, "Private HTML report fixture")
	before := testutil.Snapshot(t, w.env.Store.DB())
	refused := subject.post(path+"/"+reports[0]+"/handle", queue, url.Values{"note": {"Subject's request"}})
	if refused.Status != 404 {
		t.Fatalf("subject report handling status = %d", refused.Status)
	}
	if changed := before.Changed(testutil.Snapshot(t, w.env.Store.DB())); len(changed) != 0 {
		t.Fatalf("subject HTML handling changed durable state: %v", changed)
	}
}
