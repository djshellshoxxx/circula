package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func TestLoginTradeFlow(t *testing.T) {
	w := newWorld(t)
	alice, bob := w.browser(t), w.browser(t)
	b := w.base()

	// Anonymous users are sent to sign in.
	if p := alice.get(b); p.URL != "/login" {
		t.Fatalf("anonymous landed on %s", p.URL)
	}
	// Wrong password gives one generic message.
	f := alice.get("/login")
	bad := alice.post("/login", f, url.Values{"login": {"alice@example.org"}, "password": {"nope nope nope nope"}})
	alice.mustContain(bad, "The login or password is not correct", `id="error-summary"`)
	if bad.Status != 401 {
		t.Errorf("bad login status %d", bad.Status)
	}

	home := alice.login("alice@example.org")
	alice.mustContain(home, "Available to spend", "0.00", "Level", "Pending")

	// Propose: review step first, nothing posted yet.
	form := alice.get(b + "/trades/new?to=" + w.bobM.ID)
	alice.mustContain(form, `value="`+w.bobM.ID+`" selected`, "inputmode=\"decimal\"")
	review := alice.post(b+"/trades/new", form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"25.00"}, "note": {"garden help"}, "step": {"review"}})
	alice.mustContain(review, "Review your trade", "You pay bob@example.org 25.00 credits", "Go back and edit", "Send trade",
		"−25.00", "Available to spend after sending")
	txs, _ := w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{})
	if len(txs) != 0 {
		t.Fatal("review step must not create a trade")
	}
	// Go back keeps the values.
	back := alice.post(b+"/trades/new", review, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"25.00"}, "note": {"garden help"}, "step": {"back"}})
	alice.mustContain(back, `value="25.00"`, "garden help")

	sent := alice.post(b+"/trades/new", review, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"25.00"}, "note": {"garden help"}, "step": {"confirm"}})
	alice.mustContain(sent, "Trade sent to bob@example.org", "Waiting for bob@example.org to respond", "Cancel this proposal")
	// Replaying the same confirm (double click) creates no second trade.
	alice.post(b+"/trades/new", review, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"},
		"amount": {"25.00"}, "note": {"garden help"}, "step": {"confirm"}})
	txs, _ = w.env.Svc.ListTransactions(w.env.Ctx, w.alice, w.net.ID, nil, service.Page{})
	if len(txs) != 1 {
		t.Fatalf("want 1 trade after replay, got %d", len(txs))
	}
	id := txs[0].ID

	// Alice's home: pending section, reserved amount, balance unchanged.
	home = alice.get(b)
	alice.mustContain(home, "Reserved for pending outgoing trades", "25.00", "Waiting for bob@example.org to respond")

	// Bob sees it waiting, reviews, accepts.
	bob.login("bob@example.org")
	bh := bob.get(b)
	bob.mustContain(bh, "Trades waiting for you", "Waiting for you to respond")
	rev := bob.get(b + "/trades/" + id + "/accept")
	bob.mustContain(rev, "Review before accepting", "+25.00", "Your balance after posting", "Confirm and accept", "Go back")
	done := bob.post(b+"/trades/"+id+"/accept", rev, nil)
	bob.mustContain(done, "Trade completed.", "Your balance is now +25.00", "In credit", "Completed", "Raise a dispute")

	// Alice: in debt, with sign and word, and a history line stating direction.
	ah := alice.get(b)
	alice.mustContain(ah, "−25.00", "In debt", "In debt by 25.00 credits", "You are using 25.00 of your 100.00 credit limit")
	hist := alice.get(b + "/history")
	alice.mustContain(hist, "You paid bob@example.org 25.00 credits", "−25.00", "Completed", "Ledger reference")
	bh2 := bob.get(b + "/history")
	bob.mustContain(bh2, "You received 25.00 credits from alice@example.org", "+25.00")

	// Dispute with review step.
	df := alice.get(b + "/trades/" + id + "/dispute")
	dr := alice.post(b+"/trades/"+id+"/dispute", df, url.Values{"category": {"quality"}, "statement": {"Not as described"}, "step": {"review"}})
	alice.mustContain(dr, "Review your dispute", "Not as described", "does not change")
	dd := alice.post(b+"/trades/"+id+"/dispute", dr, url.Values{"category": {"quality"}, "statement": {"Not as described"}, "step": {"confirm"}})
	alice.mustContain(dd, "Dispute opened", "Not as described", "Withdraw this dispute")

	// Admin reviewer queue shows it and can decide.
	_, carolM := w.env.Member(w.admin, w.net, "carol@example.org")
	if err := w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, carolM.ID, domain.RoleReviewer); err != nil {
		t.Fatal(err)
	}
	adm := w.browser(t)
	adm.login("carol@example.org")
	q := adm.get(b + "/review")
	adm.mustContain(q, "Dispute: Not as described or poor quality", "Open, waiting for a reviewer")
	ds, _ := w.env.Svc.ListDisputes(w.env.Ctx, w.admin, w.net.ID, nil, service.Page{})
	dv := adm.get(b + "/disputes/" + ds[0].ID)
	adm.mustContain(dv, "Record a decision", "Start review")
	res := adm.post(b+"/disputes/"+ds[0].ID+"/resolve", dv, url.Values{"outcome": {"upheld"}, "rationale": {"Agreed"}})
	adm.mustContain(res, "Upheld: a reversal of 25.00 credits has been posted", "Agreed")

	// Logout ends the session.
	hp := alice.get(b)
	out := alice.post("/logout", hp, nil)
	alice.mustContain(out, "You are signed out")
	if p := alice.get(b); p.URL != "/login" {
		t.Errorf("after logout landed on %s", p.URL)
	}
}

func TestValidationErrorSummary(t *testing.T) {
	w := newWorld(t)
	a := w.browser(t)
	a.login("alice@example.org")
	b := w.base()
	form := a.get(b + "/trades/new")
	p := a.post(b+"/trades/new", form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"1.234"}, "step": {"review"}, "note": {"keep me"}})
	a.mustContain(p, `id="error-summary"`, `role="alert"`, `href="#f-amount"`, `id="f-amount"`, "at most 2 decimal places", "keep me",
		`aria-invalid="true"`, `id="e-amount"`)
	if p.Status != 422 {
		t.Errorf("status %d", p.Status)
	}
	// Over the available credit: the refusal says nothing changed and names the available amount.
	rev := a.post(b+"/trades/new", form, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"500.00"}, "step": {"review"}})
	a.mustContain(rev, "more than your available to spend")
	conf := a.post(b+"/trades/new", rev, url.Values{"counterparty_id": {w.bobM.ID}, "direction": {"pay"}, "amount": {"500.00"}, "step": {"confirm"}})
	a.mustContain(conf, "Not sent. This trade would take your available credit below zero.", "Available to spend is 100.00", "Nothing changed")
}

func TestJoinByInvitationLink(t *testing.T) {
	w := newWorld(t)
	tok, _, err := w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	b := w.browser(t)
	anon := b.get("/join/" + tok)
	b.mustContain(anon, "Sign in", "Create an account")
	reg := b.get("/register?next=/join/" + tok)
	p := b.post("/register", reg, url.Values{"login": {"carol@example.org"}, "password": {testPW}, "password2": {testPW}, "next": {"/join/" + tok}})
	b.mustContain(p, "Accept your invitation", "Your display name")
	j := b.post("/join/"+tok, p, url.Values{"display_name": {"Carol"}})
	b.mustContain(j, "You have joined Net town", "Available to spend")
	// Second use fails with a plain message.
	d := w.browser(t)
	reg2 := d.get("/register")
	d.post("/register", reg2, url.Values{"login": {"dave@example.org"}, "password": {testPW}, "password2": {testPW}})
	jf := d.get("/join/" + tok)
	again := d.post("/join/"+tok, jf, url.Values{"display_name": {"Dave"}})
	d.mustContain(again, "expired, already used or not valid")
}

func TestSetupFirstNetwork(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	reg := b.get("/register")
	b.post("/register", reg, url.Values{"login": {"founder@example.org"}, "password": {testPW}, "password2": {testPW}})
	f := b.get("/setup")
	b.mustContain(f, "Create a network", "Credit unit name")
	p := b.post("/setup", f, url.Values{"name": {"Fresh"}, "slug": {"fresh"}, "display_name": {"Fran"}, "currency_name": {"hour"},
		"decimals": {"1"}, "default_credit_limit": {"10.0"}, "signup_grant": {"2.5"}, "join_mode": {"approval"}})
	b.mustContain(p, "Network created and open", "10.0 hours", "Available to spend", "Network admin")
	dup := b.post("/setup", f, url.Values{"name": {"Fresh2"}, "slug": {"fresh"}, "display_name": {"Fran"}, "currency_name": {"hour"},
		"decimals": {"1"}, "default_credit_limit": {"10.0"}, "signup_grant": {"0"}, "join_mode": {"invite"}})
	b.mustContain(dup, "already in use")
	if !strings.Contains(dup.Body, `id="f-slug"`) {
		t.Error("slug field missing")
	}
}
