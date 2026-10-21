package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) propose(from domain.Actor, to *domain.Member, amt int64) *domain.Transaction {
	return testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, from, w.net.ID, service.ProposeInput{CounterpartyID: to.ID,
		Direction: service.ProposerPays, Amount: amt, IdempotencyKey: w.env.Key()}))
}

// FR-038: badge with text, inbox list, waiting section, mark read.
func TestInboxAndBadge(t *testing.T) {
	w := newWorld(t)
	bob := w.browser(t)
	b := w.base()
	bob.login("bob@example.org")
	if p := bob.get(b); strings.Contains(p.Body, "unread") {
		t.Fatalf("badge shown without notifications")
	}

	tx := w.propose(w.alice, w.bobM, 500)
	home := bob.get(b)
	bob.mustContain(home, `href="/n/town/inbox"`, "Inbox", "1 unread")

	inbox := bob.get(b + "/inbox")
	bob.mustContain(inbox, "You have a trade proposal", "alice@example.org proposed a trade of 5.00 credit", "Unread",
		"Waiting for your action", "/trades/"+tx.ID, "Mark all as read")

	// mark the single entry read
	n := testutil.Must(w.env.Svc.ListNotifications(w.env.Ctx, w.bob, w.net.ID, service.NotificationQuery{}))
	after := bob.post(b+"/inbox/read", inbox, url.Values{"id": {n[0].ID}})
	bob.mustContain(after, "No unread notifications", "Read")
	if strings.Contains(after.Body, "unread</span>") {
		t.Error("badge still shown")
	}

	// other members' entries are untouched
	w.propose(w.alice, w.bobM, 100)
	alice := w.browser(t)
	alice.login("alice@example.org")
	ainbox := alice.get(b + "/inbox")
	alice.mustContain(ainbox, "No unread notifications")
	alice.post(b+"/inbox/read", ainbox, url.Values{})
	if c := testutil.Must(w.env.Svc.CountUnreadNotifications(w.env.Ctx, w.bob, w.net.ID)); c != 1 {
		t.Errorf("bob unread = %d after alice marked all", c)
	}
	// unread filter and mark all
	bob.mustContain(bob.get(b+"/inbox?show=unread"), "1 unread", "Show all")
	p := bob.post(b+"/inbox/read", bob.get(b+"/inbox"), url.Values{})
	bob.mustContain(p, "All notifications are marked as read", "No unread notifications")

	// anonymous visitors are sent to sign in
	if p := w.browser(t).get(b + "/inbox"); p.URL != "/login" {
		t.Errorf("anonymous inbox landed on %s", p.URL)
	}
	// a forged POST without the CSRF token is refused
	if p := bob.post(b+"/inbox/read", nil, url.Values{}); p.Status != 403 {
		t.Errorf("no-csrf status = %d", p.Status)
	}
}

// FR-076 and FR-078 through the dispute page.
func TestDisputeResponseAndReReviewPages(t *testing.T) {
	w := newWorld(t)
	rev1, rev1M := w.env.Member(w.admin, w.net, "rev1@example.org")
	rev2, rev2M := w.env.Member(w.admin, w.net, "rev2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, rev1M.ID, domain.RoleReviewer))
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, rev2M.ID, domain.RoleReviewer))
	_, _ = rev1, rev2
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := testutil.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "quality",
		Statement: "broken", IdempotencyKey: w.env.Key()}))
	path := w.base() + "/disputes/" + d.ID

	alice, bob := w.browser(t), w.browser(t)
	r1, r2 := w.browser(t), w.browser(t)
	alice.login("alice@example.org")
	bob.login("bob@example.org")
	r1.login("rev1@example.org")
	r2.login("rev2@example.org")

	// bob (other party) sees the form and the dispute in "waiting for you"; alice (opener) does not
	bob.mustContain(bob.get(w.base()+"/inbox"), "A dispute was opened on a trade with you")
	bp := bob.get(path)
	bob.mustContain(bp, "Add your response", `action="/n/town/disputes/`+d.ID+`/respond"`)
	if strings.Contains(alice.get(path).Body, "Add your response") {
		t.Error("opener is offered the response form")
	}
	// empty response is refused in place
	bad := bob.post(path+"/respond", bp, url.Values{"response": {"  "}})
	bob.mustContain(bad, "There is a problem", "Add your response")
	if bad.Status != 422 {
		t.Errorf("status = %d", bad.Status)
	}
	ok := bob.post(path+"/respond", bp, url.Values{"response": {"It worked when I gave it to you."}})
	bob.mustContain(ok, "Your response was added", "It worked when I gave it to you.", "Response from bob@example.org")
	if strings.Contains(ok.Body, "Add your response") {
		t.Error("form still offered after responding")
	}
	alice.mustContain(alice.get(path), "It worked when I gave it to you.")
	r1.mustContain(r1.get(path), "It worked when I gave it to you.")
	bob.mustContain(bob.get(w.base()+"/inbox"), "Nothing is waiting for you")

	// re-review not offered before a decision
	if strings.Contains(alice.get(path).Body, "Ask for a re-review") {
		t.Error("re-review offered before a decision")
	}
	rp := r1.get(path)
	r1.mustContain(r1.post(path+"/resolve", rp, url.Values{"outcome": {"dismissed"}, "rationale": {"not enough evidence"}, "step": {"confirm"}}), "Dismissed")

	ap := alice.get(path)
	alice.mustContain(ap, "Ask for a re-review", "Request re-review")
	if strings.Contains(bob.get(path).Body, "Add your response") {
		t.Error("response offered after a decision")
	}
	req := alice.post(path+"/rereview", ap, url.Values{"reason": {"I have photos"}})
	alice.mustContain(req, "Re-review requested", "Re-review requested, waiting for a different reviewer.", "I have photos")
	if strings.Contains(req.Body, "Request re-review") {
		t.Error("second request offered")
	}

	// original reviewer is not offered the decision form; the other reviewer is
	if strings.Contains(r1.get(path).Body, "Record the re-review decision") {
		t.Error("original reviewer offered the re-review decision")
	}
	r2p := r2.get(path)
	r2.mustContain(r2p, "Record the re-review decision")
	// original reviewer posting anyway is refused
	forced := r1.post(path+"/rereview-decide", r1.get(path), url.Values{"outcome": {"upheld"}, "rationale": {"x"}})
	if forced.Status != 403 {
		t.Errorf("original reviewer forced status = %d", forced.Status)
	}
	done := r2.post(path+"/rereview-decide", r2p, url.Values{"outcome": {"upheld"}, "rationale": {"photos convince me"}})
	r2.mustContain(done, "Re-review decision recorded", "Re-review decided, upheld", "photos convince me", "Dismissed")
	if strings.Contains(done.Body, "Record the re-review decision") {
		t.Error("decision form offered after deciding")
	}
	if bal := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID)); bal.Balance != 0 {
		t.Errorf("alice balance = %d", bal.Balance)
	}
	// parties are told
	alice.mustContain(alice.get(w.base()+"/inbox"), "The re-review of a dispute is decided")
}

// FR-024 through the pages.
func TestReportPages(t *testing.T) {
	w := newWorld(t)
	l := testutil.Must(w.env.Svc.CreateListing(w.env.Ctx, w.bob, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike repair",
		IdempotencyKey: w.env.Key()}))
	alice, bob, admin := w.browser(t), w.browser(t), w.browser(t)
	alice.login("alice@example.org")
	bob.login("bob@example.org")
	admin.login("admin@example.org")
	b := w.base()

	alice.mustContain(alice.get(b+"/listings"), "report?type=listing&amp;id="+l.ID)
	alice.mustContain(alice.get(b+"/members/"+w.bobM.ID), "Report bob@example.org")

	form := alice.get(b + "/report?type=listing&id=" + l.ID)
	alice.mustContain(form, "Listing: Bike repair", "does not hide, suspend or change anything", "Send report")
	sent := alice.post(b+"/report", form, url.Values{"type": {"listing"}, "id": {l.ID}, "category": {"scam"}, "note": {"looks fake"}})
	alice.mustContain(sent, "Report sent", "they are not told who reported")

	// duplicate is explained
	dup := alice.post(b+"/report", form, url.Values{"type": {"listing"}, "id": {l.ID}, "category": {"spam"}, "idem": {"other-key"}})
	alice.mustContain(dup, "You already have an open report about this")
	if dup.Status != 422 {
		t.Errorf("duplicate status = %d", dup.Status)
	}
	// invalid category
	badc := alice.post(b+"/report", form, url.Values{"type": {"listing"}, "id": {l.ID}, "category": {"nope"}, "idem": {"k3"}})
	alice.mustContain(badc, "There is a problem")
	// own listing and unknown targets are not reportable
	if p := bob.get(b + "/report?type=listing&id=" + l.ID); p.Status != 404 {
		t.Errorf("own listing status = %d", p.Status)
	}
	if p := alice.get(b + "/report?type=member&id=nope"); p.Status != 404 {
		t.Errorf("unknown target status = %d", p.Status)
	}
	if p := alice.get(b + "/report?type=bogus&id=x"); p.Status != 404 {
		t.Errorf("bad type status = %d", p.Status)
	}

	// queue is administrators only and names the reporter
	if p := bob.get(b + "/admin/reports"); p.Status != 403 {
		t.Errorf("member queue status = %d", p.Status)
	}
	if p := alice.post(b+"/admin/reports/x/handle", alice.get(b), url.Values{}); p.Status != 403 {
		t.Errorf("member handle status = %d", p.Status)
	}
	q := admin.get(b + "/admin/reports")
	admin.mustContain(q, "Reported by alice@example.org", "Bike repair", "Looks like a scam", "looks fake", "Mark handled")
	admin.mustContain(admin.get(b+"/admin"), "/admin/reports")

	rs := testutil.Must(w.env.Svc.ListReports(w.env.Ctx, w.admin, w.net.ID, "open", service.Page{}))
	done := admin.post(b+"/admin/reports/"+rs[0].ID+"/handle", q, url.Values{"note": {"removed"}})
	admin.mustContain(done, "Report marked as handled", "No reports to show")
	admin.mustContain(admin.get(b+"/admin/reports?status=handled"), "Handled", "removed")
	alice.mustContain(alice.get(b+"/inbox"), "Your report was reviewed")
	if strings.Contains(bob.get(b+"/inbox").Body, "report was reviewed") {
		t.Error("reported member told about the report")
	}
}

// UX-012 inline notice on Home and the propose form.
func TestLimitNoticePages(t *testing.T) {
	w := newWorld(t)
	alice := w.browser(t)
	alice.login("alice@example.org")
	b := w.base()
	if strings.Contains(alice.get(b).Body, "Credit limit notice") {
		t.Fatal("notice shown with no credit used")
	}
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 7900) // default limit 100.00: 79 percent
	if strings.Contains(alice.get(b).Body, "Credit limit notice") {
		t.Fatal("notice shown below the threshold")
	}
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100) // 80 percent
	alice.mustContain(alice.get(b), "Credit limit notice", "You are using 80.00 of your 100.00 credit limit")
	alice.mustContain(alice.get(b+"/trades/new"), "Credit limit notice")
	bob := w.browser(t)
	bob.login("bob@example.org")
	if strings.Contains(bob.get(b).Body, "Credit limit notice") {
		t.Error("creditor warned")
	}
	// the background job leaves one inbox entry
	if n := testutil.Must(w.env.Svc.RunLimitWarnings(w.env.Ctx)); n != 1 {
		t.Fatalf("job notified %d", n)
	}
	alice.mustContain(alice.get(b+"/inbox"), "You are close to your credit limit")
}
