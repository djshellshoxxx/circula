package web

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (b *browser) mustNotContain(p *pg, subs ...string) {
	b.t.Helper()
	for _, s := range subs {
		if strings.Contains(p.Body, s) {
			b.t.Errorf("page %s unexpectedly contains %q", p.URL, s)
		}
	}
}

// FR-067: the overview shows each count with its words and a link, and members are refused.
func TestAdminOverviewPage(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	// two disputes (one under review), a trade at the limit would need more setup: keep to disputes + invitation.
	tx1 := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	tx2 := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 200)
	for _, id := range []string{tx1.ID, tx2.ID} {
		tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{TxID: id, Category: "other",
			Statement: "x", IdempotencyKey: w.env.Key()}))
	}
	tu.Must2(w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{TTL: 24 * time.Hour}))
	adm := w.browser(t)
	adm.login("admin@example.org")
	p := adm.get(b + "/admin")
	adm.mustContain(p, "Needs attention", "0 membership requests are waiting.", "2 disputes are open or under review.",
		`href="/n/town/review"`, "0 disputes are decided but the reversal is not yet posted", "0 active members are at or beyond their credit limit",
		"1 invitation ends within 72 hours.", `href="/n/town/admin/invitations"`, "No ledger check has been recorded yet",
		"0 open proposals have waited more than 48 hours", "12:00 UTC")
	// a recorded check shows up
	adm.post(b+"/admin/check", p, nil)
	adm.mustContain(adm.get(b+"/admin"), "The last ledger check passed.")
	// members are refused
	al := w.browser(t)
	al.login("alice@example.org")
	if r := al.get(b + "/admin"); r.Status != 403 {
		t.Errorf("member overview status %d", r.Status)
	}
}

func TestAdminReversalUI(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 2500) // alice -25, bob +25
	adm := w.browser(t)
	adm.login("admin@example.org")

	// the trade page offers the reversal to an administrator who is not a party
	adm.mustContain(adm.get(b+"/trades/"+tx.ID), "Reverse this trade (administrator)")
	al := w.browser(t)
	al.login("alice@example.org")
	al.mustNotContain(al.get(b+"/trades/"+tx.ID), "Reverse this trade")
	if r := al.get(b + "/admin/reverse"); r.Status != 403 {
		t.Errorf("member reverse form status %d", r.Status)
	}

	f := adm.get(b + "/admin/reverse?tx=" + tx.ID)
	adm.mustContain(f, "Left to reverse", "25.00", "alice@example.org", "bob@example.org")
	form := func(extra url.Values) url.Values {
		v := url.Values{"tx": {tx.ID}, "amount": {""}, "reason": {"posted twice by mistake"}, "step": {"review"}}
		for k, x := range extra {
			v[k] = x
		}
		return v
	}
	// reason required
	bad := adm.post(b+"/admin/reverse", f, form(url.Values{"reason": {"  "}}))
	adm.mustContain(bad, `href="#f-reason"`, "a reason is required")
	if bad.Status != 422 {
		t.Errorf("status %d", bad.Status)
	}
	// unknown reference, too much, and a nonsense amount
	adm.mustContain(adm.post(b+"/admin/reverse", f, form(url.Values{"tx": {"nope"}})), "no trade with this reference")
	adm.mustContain(adm.post(b+"/admin/reverse", f, form(url.Values{"amount": {"30.00"}})), "at most 25.00 is left")
	adm.mustContain(adm.post(b+"/admin/reverse", f, form(url.Values{"amount": {"1.234"}})), "decimal places")
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 2500 {
		t.Fatalf("a refusal changed bob: %d", bal.Balance)
	}

	// review step: shows the effect, posts nothing
	rev := adm.post(b+"/admin/reverse", f, form(url.Values{"amount": {"10.00"}}))
	adm.mustContain(rev, "Nothing is posted until you confirm", "10.00 credits go back from bob@example.org to alice@example.org",
		"partial reversal", "posted twice by mistake", "Post reversal", "Go back and edit")
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 2500 {
		t.Fatalf("review posted early: %d", bal.Balance)
	}
	back := adm.post(b+"/admin/reverse", rev, form(url.Values{"amount": {"10.00"}, "step": {"back"}}))
	adm.mustContain(back, "Reverse a trade", "posted twice by mistake</textarea>")
	// confirm
	done := adm.post(b+"/admin/reverse", rev, form(url.Values{"amount": {"10.00"}, "step": {"confirm"}}))
	adm.mustContain(done, "Reversal posted: 10.00 credits went back from bob@example.org to alice@example.org", "Reversal")
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 1500 {
		t.Errorf("bob after partial = %d", bal.Balance)
	}
	// reversing the rest ends in Reversed; then nothing is left
	rev2 := adm.post(b+"/admin/reverse", f, form(nil))
	adm.mustContain(rev2, "15.00 credits go back", "becomes Reversed")
	adm.post(b+"/admin/reverse", rev2, form(url.Values{"step": {"confirm"}}))
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 0 {
		t.Errorf("bob after full = %d", bal.Balance)
	}
	adm.mustContain(adm.post(b+"/admin/reverse", f, form(nil)), "nothing is left to reverse on this trade")
	// an administrator who took part cannot reverse their own trade
	own := w.env.Trade(w.net.ID, w.admin, w.bob, w.adminM, w.bobM, 100)
	adm.mustContain(adm.post(b+"/admin/reverse", f, form(url.Values{"tx": {own.ID}})), "you took part in this trade")
	// a repeat submit of the same confirm form does not reverse twice
	other := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	f2 := adm.get(b + "/admin/reverse?tx=" + other.ID)
	r1 := adm.post(b+"/admin/reverse", f2, url.Values{"tx": {other.ID}, "reason": {"dup"}, "step": {"review"}})
	conf := url.Values{"tx": {other.ID}, "reason": {"dup"}, "step": {"confirm"}}
	adm.post(b+"/admin/reverse", r1, conf)
	adm.post(b+"/admin/reverse", r1, conf)
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 100 {
		t.Errorf("double submit: bob = %d (want 100: one reversal of the 500 plus the 100 trade)", bal.Balance)
	}
}

func TestDisputeDecisionNeedsConfirmation(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	d := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "quality", Statement: "bad", IdempotencyKey: w.env.Key()}))
	adm := w.browser(t)
	adm.login("admin@example.org")
	dv := adm.get(b + "/disputes/" + d.ID)
	adm.mustContain(dv, "Review decision")
	act := b + "/disputes/" + d.ID + "/resolve"
	// missing rationale, partial without amount, partial above remaining
	adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"upheld"}, "rationale": {" "}}), "Rationale")
	adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"partially_upheld"}, "rationale": {"r"}}), "enter the amount to reverse")
	adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"partially_upheld"}, "rationale": {"r"}, "amount": {"11.00"}}), "remaining reversible amount")
	// a post without step=confirm never records
	for _, step := range []string{"", "review", "bogus"} {
		adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"partially_upheld"}, "rationale": {"half"}, "amount": {"4.00"}, "step": {step}}),
			"Partly upheld: part of the trade is reversed", "4.00 credits go back from bob@example.org to alice@example.org")
	}
	if got := tu.Must(w.env.Svc.GetDispute(w.env.Ctx, w.admin, w.net.ID, d.ID)); got.Dispute.Status != domain.DisputeOpen {
		t.Fatalf("status after reviews = %s", got.Dispute.Status)
	}
	adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"dismissed"}, "rationale": {"fine"}}), "No credit moves")
	// back keeps the typed values
	adm.mustContain(adm.post(act, dv, url.Values{"outcome": {"upheld"}, "rationale": {"keep me"}, "step": {"back"}}), "Record a decision", "keep me")
	ok := adm.post(act, dv, url.Values{"outcome": {"partially_upheld"}, "rationale": {"half"}, "amount": {"4.00"}, "step": {"confirm"}})
	adm.mustContain(ok, "Decision recorded.", "Partly upheld: a reversal of 4.00 credits has been posted")
	if bal := tu.Must(w.env.Svc.GetBalance(w.env.Ctx, w.bob, w.net.ID)); bal.Balance != 600 {
		t.Errorf("bob = %d", bal.Balance)
	}
	// a party cannot decide even with a confirm
	d2tx := w.env.Trade(w.net.ID, w.bob, w.alice, w.bobM, w.aliceM, 100)
	d2 := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.bob, w.net.ID, service.OpenDisputeInput{TxID: d2tx.ID, Category: "other", Statement: "s", IdempotencyKey: w.env.Key()}))
	al := w.browser(t)
	al.login("alice@example.org")
	r := al.post(b+"/disputes/"+d2.ID+"/resolve", al.get(b+"/disputes/"+d2.ID), url.Values{"outcome": {"upheld"}, "rationale": {"r"}, "step": {"confirm"}})
	if r.Status == 200 && strings.Contains(r.Body, "Decision recorded") {
		t.Error("a party recorded a decision")
	}
}

// WEB-02: the queue and the disputes list reach every dispute, and the dashboard every open proposal.
func TestDisputeListsAndDashboardAreNotCapped(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	const n = perPage + 3
	var ids []string
	for i := 0; i < n; i++ {
		tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1)
		d := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.bob, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", Statement: "s", IdempotencyKey: w.env.Key()}))
		ids = append(ids, d.ID)
	}
	// reviewer queue: two pages, no dispute lost or repeated, oldest last
	_, carolM := w.env.Member(w.admin, w.net, "carol@example.org")
	tu.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, carolM.ID, domain.RoleReviewer))
	rv := w.browser(t)
	rv.login("carol@example.org")
	seen := map[string]bool{}
	path := b + "/review"
	pages := 0
	for path != "" {
		p := rv.get(path)
		pages++
		for _, id := range ids {
			if strings.Contains(p.Body, "/disputes/"+id+`"`) {
				if seen[id] {
					t.Errorf("dispute %s listed twice", id)
				}
				seen[id] = true
			}
		}
		path = ""
		if i := strings.Index(p.Body, `Next page</a>`); i >= 0 {
			j := strings.LastIndex(p.Body[:i], `href="`)
			path = strings.ReplaceAll(p.Body[j+6:i-len(`">`)], "&amp;", "&")
		}
		if pages > 5 {
			t.Fatal("paging does not end")
		}
	}
	if len(seen) != n || pages != 2 {
		t.Errorf("queue reached %d of %d disputes in %d pages", len(seen), n, pages)
	}
	// filter by status shows none under review yet and rejects junk values quietly
	rv.mustContain(rv.get(b+"/review?status=under_review"), "The queue is empty")
	rv.mustContain(rv.get(b+"/review?status=garbage"), "Dispute: ")
	// the disputes list pages too: bob opened all of them
	bb := w.browser(t)
	bb.login("bob@example.org")
	first := bb.get(b + "/disputes")
	bb.mustContain(first, "Next page")
	if cnt := strings.Count(first.Body, "/disputes/0"); cnt != perPage {
		t.Errorf("first page lists %d disputes, want %d", cnt, perPage)
	}
	// a plain member cannot open the queue
	if p := bb.get(b + "/review"); p.Status != 403 {
		t.Errorf("member queue status %d", p.Status)
	}

	// dashboard: more open proposals than the old window of 100; the oldest still shows
	oldest := tu.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{CounterpartyID: w.bobM.ID,
		Direction: service.ProposerPays, Amount: 77, Note: "the oldest one", IdempotencyKey: w.env.Key()}))
	for i := 0; i < 110; i++ {
		tu.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{CounterpartyID: w.bobM.ID,
			Direction: service.ProposerPays, Amount: 1 + int64(i%50), IdempotencyKey: w.env.Key()}))
	}
	home := bb.get(b)
	bb.mustContain(home, "/trades/"+oldest.ID, "Trades waiting for you")
	if cnt := strings.Count(home.Body, "/trades/0"); cnt < 111 {
		t.Errorf("dashboard lists %d proposals, want at least 111", cnt)
	}
}

func TestAnnouncementsOnDashboard(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	al := w.browser(t)
	al.login("alice@example.org")
	al.mustNotContain(al.get(b), "Announcements")

	f := adm.get(b + "/admin/announcements")
	adm.mustContain(f, "Publish announcement", "No announcements yet")
	bad := adm.post(b+"/admin/announcements", f, url.Values{"title": {""}, "body": {"b"}})
	adm.mustContain(bad, `id="error-summary"`)
	if bad.Status != 422 {
		t.Errorf("status %d", bad.Status)
	}
	adm.mustContain(adm.post(b+"/admin/announcements", f, url.Values{"title": {"t"}, "body": {"b"}, "days": {"x"}}), "whole number from 0 to 365")
	ok := adm.post(b+"/admin/announcements", f, url.Values{"title": {"Market day"}, "body": {"Saturday <b>ten</b>"}})
	adm.mustContain(ok, "Announcement published", "Market day")
	adm.post(b+"/admin/announcements", adm.get(b+"/admin/announcements"), url.Values{"title": {"Next month"}, "body": {"later"}, "days": {"30"}})
	home := al.get(b)
	al.mustContain(home, "Announcements", "Market day", "Saturday &lt;b&gt;ten&lt;/b&gt;") // escaped
	al.mustNotContain(home, "Next month", "Saturday <b>")
	adm.mustContain(adm.get(b+"/admin/announcements"), "Next month", "starts")
	// members cannot reach the admin page or delete
	if r := al.get(b + "/admin/announcements"); r.Status != 403 {
		t.Errorf("member admin page %d", r.Status)
	}
	// delete
	as := tu.Must(w.env.Svc.AdminListAnnouncements(w.env.Ctx, w.admin, w.net.ID, service.Page{}))
	var marketID string
	for _, a := range as {
		if a.Title == "Market day" {
			marketID = a.ID
		}
	}
	if r := al.post(b+"/admin/announcements/"+marketID+"/delete", home, nil); r.Status != 403 {
		t.Errorf("member delete %d", r.Status)
	}
	adm.mustContain(adm.post(b+"/admin/announcements/"+marketID+"/delete", adm.get(b+"/admin/announcements"), nil), "Announcement deleted")
	al.mustNotContain(al.get(b), "Market day")
	if r := adm.post(b+"/admin/announcements/"+marketID+"/delete", adm.get(b+"/admin/announcements"), nil); r.Status != 404 {
		t.Errorf("second delete %d", r.Status)
	}
}

func TestTimeZoneDisplay(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100) // completed at 12:00 UTC
	al := w.browser(t)
	al.login("alice@example.org")
	al.mustContain(al.get(b+"/trades/"+tx.ID), "1 Mar 2026, 12:00 UTC", `datetime="2026-03-01T12:00:00Z"`)

	f := al.get("/account/time-zone")
	al.mustContain(f, "Time zone", "UTC")
	bad := al.post("/account/time-zone", f, url.Values{"time_zone": {"Mars/Base"}})
	if bad.Status != 422 {
		t.Errorf("bad zone status %d", bad.Status)
	}
	al.mustContain(bad, `id="error-summary"`, "Europe/Berlin")
	al.mustContain(al.post("/account/time-zone", f, url.Values{"time_zone": {"Asia/Kolkata"}}), "Time zone saved", "Asia/Kolkata", "17:30 IST")
	p := al.get(b + "/trades/" + tx.ID)
	al.mustContain(p, "1 Mar 2026, 17:30 IST", `datetime="2026-03-01T12:00:00Z"`)
	al.mustNotContain(p, "12:00 UTC")
	al.mustContain(al.get(b+"/history"), "17:30 IST")
	// another person is unaffected
	bb := w.browser(t)
	bb.login("bob@example.org")
	bb.mustContain(bb.get(b+"/trades/"+tx.ID), "12:00 UTC")
	// clearing goes back to UTC
	al.post("/account/time-zone", f, url.Values{"time_zone": {""}})
	al.mustContain(al.get(b+"/trades/"+tx.ID), "12:00 UTC")
	// the API-facing audit log stays UTC for the administrator whatever their zone
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.post("/account/time-zone", adm.get("/account/time-zone"), url.Values{"time_zone": {"America/New_York"}})
	adm.mustContain(adm.get(b+"/admin/audit"), "EST")
	if got := tu.Must(w.env.Svc.GetTimeZone(w.env.Ctx, w.admin)); got != "America/New_York" {
		t.Errorf("stored zone = %q", got)
	}
}

// A reviewer or administrator reaches the dispute from the disputed trade without being a party.
func TestTradeDetailLinksDisputeForDeciders(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	d := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", Statement: "s", IdempotencyKey: w.env.Key()}))
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get(b+"/trades/"+tx.ID), fmt.Sprintf(`href="/n/town/disputes/%s"`, d.ID))
	// not offered for a disputed trade as a reversal: it goes through the dispute
	adm.mustNotContain(adm.get(b+"/trades/"+tx.ID), "Reverse this trade")
	_, cM := w.env.Member(w.admin, w.net, "carol@example.org")
	_ = cM
	cb := w.browser(t)
	cb.login("carol@example.org")
	cb.mustNotContain(cb.get(b+"/trades/"+tx.ID), "/disputes/"+d.ID) // a plain member cannot even open the trade
}
