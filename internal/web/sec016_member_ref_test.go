package web

import (
	"html"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestMemberRefFormat(t *testing.T) {
	for in, want := range map[string]string{
		"018f0000-0000-7000-8000-00000000000a": "0000-000A",
		"0190c3b2-7d5e-7a4b-9f3e-2c1d0e9f8a7b": "0E9F-8A7B",
		"abc":                                  "ABC",
		"":                                     "",
	} {
		if got := memberRef(in); got != want {
			t.Errorf("memberRef(%q) = %q, want %q", in, got, want)
		}
	}
}

// SEC-016: display names are not unique identity, so every trade confirmation step shows a
// stable member reference beside each name, and two members with the same name differ there.
func TestConfirmScreensShowMemberRef(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx
	// carol joins under bob's display name.
	carol := w.env.Person("carol@example.org")
	tok, _ := testutil.Must2(w.env.Svc.CreateInvitation(ctx, w.admin, w.net.ID, service.InviteInput{}))
	carolM := testutil.Must(w.env.Svc.RedeemInvitation(ctx, carol, tok, w.bobM.DisplayName))
	if carolM.DisplayName != w.bobM.DisplayName || memberRef(carolM.ID) == memberRef(w.bobM.ID) {
		t.Fatalf("fixture: names %q/%q refs %q/%q", carolM.DisplayName, w.bobM.DisplayName, memberRef(carolM.ID), memberRef(w.bobM.ID))
	}
	ref := func(id string) string { return "(member reference " + memberRef(id) + ")" }

	al := w.browser(t)
	al.login("alice@example.org")
	form := al.get(b + "/trades/new")
	for _, m := range []struct{ id, other string }{{w.bobM.ID, carolM.ID}, {carolM.ID, w.bobM.ID}} {
		rv := al.post(b+"/trades/new", form, url.Values{"counterparty_id": {m.id}, "direction": {"pay"}, "amount": {"1.00"}, "step": {"review"}})
		al.mustContain(rv, "Review your trade", "<dt>With</dt><dd>"+html.EscapeString(w.bobM.DisplayName)+" "+ref(m.id)+"</dd>")
		if strings.Contains(rv.Body, ref(m.other)) {
			t.Errorf("review for %s shows the other member's reference", m.id)
		}
	}

	// Accept, decline and cancel confirmations show both parties with their references.
	in := testutil.Must(w.env.Svc.ProposeTransaction(ctx, carol, w.net.ID, service.ProposeInput{CounterpartyID: w.aliceM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.env.Key()}))
	for _, act := range []string{"accept", "decline"} {
		p := al.get(b + "/trades/" + in.ID + "/" + act)
		al.mustContain(p, "<dt>Payer</dt><dd>"+html.EscapeString(carolM.DisplayName)+" "+ref(carolM.ID)+"</dd>",
			"<dt>Payee</dt><dd>"+html.EscapeString(w.aliceM.DisplayName)+" "+ref(w.aliceM.ID)+"</dd>")
		if strings.Contains(p.Body, ref(w.bobM.ID)) {
			t.Errorf("%s: shows bob's reference for carol's proposal", act)
		}
	}
	cb := w.browser(t)
	cb.login("carol@example.org")
	cb.mustContain(cb.get(b+"/trades/"+in.ID+"/cancel"), ref(carolM.ID), ref(w.aliceM.ID))
	// An administrator cancelling someone else's proposal sees both references too.
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get(b+"/trades/"+in.ID+"/cancel"), ref(carolM.ID), ref(w.aliceM.ID), "Reason")
}

// Review fix (SEC-016): a reference shown only on the confirmation step cannot be checked
// against anything. The directory and the member page show it, a member sees their own to
// pass on, and the trade form's member list tells same-named members apart by it.
func TestMemberRefIsDiscoverable(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx
	carol := w.env.Person("carol@example.org")
	tok, _ := testutil.Must2(w.env.Svc.CreateInvitation(ctx, w.admin, w.net.ID, service.InviteInput{}))
	carolM := testutil.Must(w.env.Svc.RedeemInvitation(ctx, carol, tok, w.bobM.DisplayName))
	name := html.EscapeString(w.bobM.DisplayName)

	al := w.browser(t)
	al.login("alice@example.org")
	dir := al.get(b + "/directory")
	for _, id := range []string{w.bobM.ID, carolM.ID} {
		al.mustContain(dir, name+"</a> (member reference "+memberRef(id)+")")
		al.mustContain(al.get(b+"/members/"+id), "Member reference: <strong>"+memberRef(id)+"</strong>")
		form := al.get(b + "/trades/new")
		al.mustContain(form, `value="`+id+`"`, ">"+name+" (member reference "+memberRef(id)+")</option>")
	}
	al.mustContain(al.get(b+"/members/"+w.aliceM.ID), "Member reference: <strong>"+memberRef(w.aliceM.ID)+"</strong>", "Tell it to the members you trade with")
	// SEC-016 acceptance: member and transaction pages carry no-index directives, and an
	// anonymous visitor gets sign-in, not the directory or a member page.
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 10)
	for _, u := range []string{b + "/directory", b + "/members/" + w.bobM.ID, b + "/trades/" + tx.ID, b + "/history"} {
		p := al.get(u)
		if p.Status != 200 || !strings.Contains(p.Header.Get("X-Robots-Tag"), "noindex") || !strings.Contains(p.Body, `name="robots" content="noindex`) {
			t.Errorf("%s: status %d, no no-index directive", u, p.Status)
		}
		if a := w.browser(t).get(u); !strings.Contains(a.URL, "/login") || strings.Contains(a.Body, name+"</a> (member reference") {
			t.Errorf("anonymous %s: ended at %s (%d)", u, a.URL, a.Status)
		}
	}
	// A unique name keeps its plain label in the trade form.
	cb := w.browser(t)
	cb.login("carol@example.org")
	if f := cb.get(b + "/trades/new"); !strings.Contains(f.Body, ">"+html.EscapeString(w.aliceM.DisplayName)+"</option>") {
		t.Errorf("a unique name gained a reference in the member list: %s", snippet(f.Body))
	}
}
