package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / FR-024: ordinary parties can choose a report from their dispute
// without typing references. The selected case remains visible after errors,
// and neither this entry point nor the summary grants reporter/subject access.
func TestReportDisputeContextHTMLJourney(t *testing.T) {
	w := newWorld(t)
	e := w.env
	listing := tu.Must(e.Svc.CreateListing(e.Ctx, w.bob, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Related listing title", IdempotencyKey: e.Key(),
	}))
	trade := tu.Must(e.Svc.ProposeTransaction(e.Ctx, w.alice, w.net.ID, service.ProposeInput{
		CounterpartyID: w.bobM.ID, Direction: service.ProposerPays, Amount: 100, ListingID: listing.ID, IdempotencyKey: e.Key(),
	}))
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, w.bob, w.net.ID, service.TxRef{TxID: trade.ID, IdempotencyKey: e.Key()}))
	dispute := tu.Must(e.Svc.OpenDispute(e.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Ordinary dispute", IdempotencyKey: e.Key(),
	}))
	_, reviewerM := e.Member(w.admin, w.net, "context-reviewer@example.org")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, reviewerM.ID, domain.RoleReviewer))
	alice, bob, admin, reviewer := w.browser(t), w.browser(t), w.browser(t), w.browser(t)
	alice.login("alice@example.org")
	bob.login("bob@example.org")
	admin.login("admin@example.org")
	reviewer.login("context-reviewer@example.org")
	casePath := w.base() + "/disputes/" + dispute.ID
	memberPath := w.base() + "/report?type=member&id=" + w.bobM.ID + "&dispute_id=" + dispute.ID
	listingPath := w.base() + "/report?type=listing&id=" + listing.ID + "&dispute_id=" + dispute.ID
	detail := alice.get(casePath)
	alice.mustContain(detail, memberPath, listingPath, "Report the other member about this dispute", "Report the listing about this dispute")
	if page := bob.get(casePath); strings.Contains(page.Body, "Report the listing about this dispute") {
		t.Fatal("listing owner received an action to report their own listing")
	}
	if page := reviewer.get(casePath); strings.Contains(page.Body, "Report the other member about this dispute") || strings.Contains(page.Body, "Report the listing about this dispute") {
		t.Fatal("nonparty reviewer received party-only report actions")
	}
	form := alice.get(memberPath)
	alice.mustContain(form, "This report is linked to your dispute with "+w.bobM.DisplayName, "opened on 1 Mar 2026", casePath,
		`name="dispute_id" value="`+dispute.ID+`"`, "Independent reviewers of this dispute can see the report category")
	invalid := alice.post(w.base()+"/report", form, url.Values{
		"type": {"member"}, "id": {w.bobM.ID}, "dispute_id": {dispute.ID}, "category": {"other"}, "note": {strings.Repeat("n", 501)},
	})
	if invalid.Status != 422 {
		t.Fatalf("ordinary form validation status = %d", invalid.Status)
	}
	alice.mustContain(invalid, "This report is linked to your dispute with "+w.bobM.DisplayName, casePath,
		`name="dispute_id" value="`+dispute.ID+`"`)
	const privateMemberNote = "Private case-related member note"
	sent := alice.post(w.base()+"/report", invalid, url.Values{
		"type": {"member"}, "id": {w.bobM.ID}, "dispute_id": {dispute.ID}, "category": {"other"}, "note": {privateMemberNote},
	})
	alice.mustContain(sent, "Report sent", "Dispute")
	if sent.URL != casePath {
		t.Fatalf("case report did not return to its dispute: %s", sent.URL)
	}
	// A historical listing relation stays usable without exposing a title
	// hidden by the ordinary listing reader.
	archived := domain.ListingArchived
	tu.Must(e.Svc.UpdateListing(e.Ctx, w.bob, w.net.ID, service.ListingUpdate{ListingID: listing.ID, Status: &archived}))
	listingForm := alice.get(listingPath)
	alice.mustContain(listingForm, "The listing attached to this disputed trade", "View related dispute")
	if strings.Contains(listingForm.Body, listing.Title) {
		t.Fatal("case form disclosed an archived listing title")
	}
	const privateListingNote = "Private case-related listing note"
	alice.mustContain(alice.post(w.base()+"/report", listingForm, url.Values{
		"type": {"listing"}, "id": {listing.ID}, "dispute_id": {dispute.ID}, "category": {"scam"}, "note": {privateListingNote},
	}), "Report sent")
	queue := admin.get(w.base() + "/admin/reports")
	admin.mustContain(queue, casePath, "View related dispute", privateMemberNote, privateListingNote)
	reports := tu.Must(e.Svc.ListReports(e.Ctx, w.admin, w.net.ID, "", service.Page{}))
	if len(reports) != 2 || reports[0].DisputeID != dispute.ID || reports[1].DisputeID != dispute.ID {
		t.Fatal("HTML report submissions lost the selected case")
	}
	summary := reviewer.get(casePath + "/reports")
	reviewer.mustContain(summary, "Reports for this dispute", reports[0].ID, reports[1].ID)
	for _, hidden := range []string{privateMemberNote, privateListingNote, w.aliceM.ID, w.bobM.ID, listing.ID, listing.Title} {
		if strings.Contains(summary.Body, hidden) {
			t.Fatalf("reviewer page disclosed private report target or text: %q", hidden)
		}
	}
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleReviewer))
	tu.Must(0, e.Svc.AssignRole(e.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleAdmin))
	if page := bob.get(casePath + "/reports"); page.Status != 403 {
		t.Fatal("subject roles granted report summary access")
	}
	for _, page := range []*pg{bob.get(casePath), bob.get(w.base() + "/admin/reports")} {
		for _, hidden := range []string{reports[0].ID, reports[1].ID, privateMemberNote, privateListingNote} {
			if strings.Contains(page.Body, hidden) {
				t.Fatal("subject page disclosed the reports")
			}
		}
	}
	if page := admin.get(memberPath); page.Status != 404 {
		t.Fatal("nonparty administrator gained a case-linked creation form")
	}
	// Existing context-free form entry stays valid and does not acquire a case.
	_, otherM := e.Member(w.admin, w.net, "other-context-target@example.org")
	plain := alice.get(w.base() + "/report?type=member&id=" + otherM.ID)
	if plain.Status != 200 || strings.Contains(plain.Body, `name="dispute_id"`) || strings.Contains(plain.Body, "This report is linked") {
		t.Fatal("context-free report form changed")
	}
}
