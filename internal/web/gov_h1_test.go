package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-009 through the pages: publish, prompt before a trade, accept, join with the terms shown.
func TestTermsPages(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	f := adm.get(b + "/admin/terms")
	adm.mustContain(f, "No terms are published yet")
	empty := adm.post(b+"/admin/terms", f, url.Values{"body": {" "}})
	adm.mustContain(empty, `id="error-summary"`)
	done := adm.post(b+"/admin/terms", f, url.Values{"body": {"Be kind.\n\nPay promptly."}})
	adm.mustContain(done, "Published version 1", "Be kind.")

	al := w.browser(t)
	al.login("alice@example.org")
	form := al.get(b + "/trades/new?to=" + w.bobM.ID)
	review := al.post(b+"/trades/new", form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"1.00"}, "step": {"review"}})
	sent := al.post(b+"/trades/new", review, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"1.00"}, "step": {"confirm"}})
	al.mustContain(sent, "Network rules and terms", "Pay promptly.", "Accept these terms")
	al.mustContain(al.get(b+"/history"), "History") // reading stays open
	al.mustContain(al.get(b+"/inbox"), "Updated network terms")
	// Not ticked: refused. Ticked: accepted and sent back to trading.
	page := al.get(b + "/terms")
	no := al.post(b+"/terms", page, url.Values{"version": {"1"}})
	al.mustContain(no, `id="error-summary"`, "tick the box")
	ok := al.post(b+"/terms", page, url.Values{"version": {"1"}, "accept": {"1"}})
	al.mustContain(ok, "You accepted version 1")
	if v := testutil.Must(w.env.Svc.CurrentTerms(w.env.Ctx, w.alice, w.net.ID)); v.NeedsAcceptance() {
		t.Error("still needs acceptance")
	}

	// Joining a network with terms shows them and needs the box.
	w.env.Person("dina@example.org")
	tok, _, err := w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	d := w.browser(t)
	d.login("dina@example.org")
	jf := d.get("/join/" + tok)
	d.mustContain(jf, "Network terms (version 1)", "Pay promptly.", "accept these terms")
	miss := d.post("/join/"+tok, jf, url.Values{"display_name": {"Dina"}, "terms_version": {"1"}})
	d.mustContain(miss, `id="error-summary"`, `href="#f-terms_version"`)
	if miss.Status != 422 {
		t.Errorf("status %d", miss.Status)
	}
	joined := d.post("/join/"+tok, jf, url.Values{"display_name": {"Dina"}, "terms_version": {"1"}, "accept_terms": {"1"}})
	d.mustContain(joined, "You have joined")
}

// GOV-002/003 through the pages: a scheduled change, the member log, cancellation.
func TestPolicyNoticePages(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	vals := func(limit, notice, nreason string) url.Values {
		return url.Values{"currency_name": {"credit"}, "default_credit_limit": {limit}, "max_tx_amount": {"1000000.00"},
			"signup_grant": {"0.00"}, "proposal_expiry_days": {"7"}, "dispute_window_days": {"30"}, "limit_warning_days": {"0"},
			"response_period_days": {"7"}, "re_review_window_days": {"14"}, "limit_warning_percent": {"80"},
			"join_mode": {"invite"}, "departure_rule": {"retain"}, "reason": {"tighten"}, "notice_days": {notice}, "notice_reason": {nreason}}
	}
	cur := w.env.Svc
	// Raise first (not adverse): immediate.
	if r := adm.post(b+"/admin/policy", adm.get(b+"/admin/policy"), vals("100.00", "", "")); r.Status >= 400 {
		t.Fatalf("raise: %d %s", r.Status, r.Body[strings.Index(r.Body, "error-summary"):][:600])
	}
	// Lower: 13 days refused, default scheduled.
	short := adm.post(b+"/admin/policy", adm.get(b+"/admin/policy"), vals("10.00", "13", ""))
	adm.mustContain(short, `id="error-summary"`, `href="#f-notice_days"`)
	sched := adm.post(b+"/admin/policy", adm.get(b+"/admin/policy"), vals("10.00", "", ""))
	adm.mustContain(sched, "Scheduled change", "Cancel the scheduled change", "Default credit limit: 100.00 to 10.00")
	n1 := testutil.Must(cur.GetNetwork(w.env.Ctx, w.admin, w.net.ID))
	if n1.Policy.DefaultCreditLimit != 10000 {
		t.Fatalf("limit in force = %d, want 10000 (the scheduled 1000 must wait)", n1.Policy.DefaultCreditLimit)
	}

	al := w.browser(t)
	al.login("alice@example.org")
	al.mustContain(al.get(b+"/policy"), "Scheduled", "Default credit limit: 100.00 to 10.00", "tighten", "takes effect")
	al.mustContain(al.get(b+"/inbox"), "Rule change announced")

	p := adm.get(b + "/admin/policy")
	can := adm.post(b+"/admin/policy/cancel", p, nil)
	adm.mustContain(can, "scheduled change was cancelled")
	if strings.Contains(can.Body, "Cancel the scheduled change") {
		t.Error("cancel button still shown")
	}
	w.env.Clock.Advance(20 * 24 * time.Hour)
	if n := testutil.Must(cur.GetNetwork(w.env.Ctx, w.admin, w.net.ID)); n.Policy.DefaultCreditLimit != 10000 {
		t.Errorf("cancelled change took effect: %d", n.Policy.DefaultCreditLimit)
	}
}

// FR-014 through the pages: blockers shown, then a confirmed departure.
func TestLeavePages(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	blocked := adm.get(b + "/leave")
	adm.mustContain(blocked, "You cannot leave yet", "last active administrator")
	if strings.Contains(blocked.Body, `action="/n/town/leave"`) {
		t.Error("the form is offered to a blocked member")
	}

	al := w.browser(t)
	al.login("alice@example.org")
	open := testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.bob, w.net.ID, service.ProposeInput{CounterpartyID: w.aliceM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.env.Key()}))
	al.mustContain(al.get(b+"/policy"), "Leave this network")
	f := al.get(b + "/leave")
	al.mustContain(f, "1 open trade proposal", "Leave this network", "Your balance is")
	no := al.post(b+"/leave", f, url.Values{"reason": {"moving"}})
	al.mustContain(no, `id="error-summary"`, "tick the box")
	done := al.post(b+"/leave", f, url.Values{"reason": {"moving"}, "confirm": {"1"}})
	al.mustContain(done, "You have left", "You have left this network")
	m := testutil.Must(w.env.Svc.MyMemberships(w.env.Ctx, w.alice))
	var status domain.MemberStatus
	for _, x := range m {
		status = x.Status
	}
	if status != domain.MemberDeparted {
		t.Errorf("status = %s", status)
	}
	if got := testutil.Must(w.env.Svc.GetTransaction(w.env.Ctx, w.bob, w.net.ID, open.ID)); got.Tx.Status != domain.TxCancelled {
		t.Errorf("open proposal = %s", got.Tx.Status)
	}
}
