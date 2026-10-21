package web

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

var (
	controlRe = regexp.MustCompile(`(?is)<(input|select|textarea)\b([^>]*)>`)
	attrRe    = func(name string) *regexp.Regexp { return regexp.MustCompile(`(?i)\b` + name + `="([^"]*)"`) }
	headingRe = regexp.MustCompile(`(?i)<h([1-6])\b`)
	tagRe     = regexp.MustCompile(`(?s)<[^>]*>`)
)

// checkStructure applies structural WCAG-oriented checks to one rendered page.
func checkStructure(t *testing.T, path string, p *pg) {
	t.Helper()
	fail := func(f string, a ...any) { t.Errorf("%s: "+f, append([]any{path}, a...)...) }
	b := p.Body
	if !strings.HasPrefix(b, "<!doctype html>") {
		fail("no doctype")
	}
	if !strings.Contains(b, `<html lang="en">`) {
		fail("html lang missing")
	}
	if m := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(b); m == nil || !strings.HasSuffix(m[1], "Circula") {
		fail("title missing or not unique-looking: %v", m)
	} else if h := title(p); h != "" && !strings.Contains(html.UnescapeString(m[1]), html.UnescapeString(h)) {
		fail("title %q does not contain h1 %q", m[1], h)
	}
	if n := len(regexp.MustCompile(`(?i)<h1\b`).FindAllString(b, -1)); n != 1 {
		fail("want exactly one h1, got %d", n)
	}
	skip := strings.Index(b, `class="skip" href="#main"`)
	if skip < 0 || skip > strings.Index(b, "<header") || !strings.Contains(b, `<main id="main"`) {
		fail("skip link must come first and target main")
	}
	for _, l := range []string{"<header", "<main", "<footer"} {
		if !strings.Contains(b, l) {
			fail("landmark %s missing", l)
		}
	}
	if strings.Contains(b, "<nav") && !strings.Contains(b, `<nav aria-label=`) {
		fail("nav without a name")
	}
	// Heading outline never skips a level.
	prev := 0
	for _, m := range headingRe.FindAllStringSubmatch(b, -1) {
		lv, _ := strconv.Atoi(m[1])
		if prev != 0 && lv > prev+1 {
			fail("heading level jumps from h%d to h%d", prev, lv)
		}
		prev = lv
	}
	// Every visible form control has a programmatic label.
	for _, loc := range controlRe.FindAllStringSubmatchIndex(b, -1) {
		tag, attrs := b[loc[2]:loc[3]], b[loc[4]:loc[5]]
		if m := attrRe("type").FindStringSubmatch(attrs); m != nil && (m[1] == "hidden" || m[1] == "submit") {
			continue
		}
		before := b[:loc[0]]
		wrapped := strings.LastIndex(before, "<label") > strings.LastIndex(before, "</label>")
		if m := attrRe("id").FindStringSubmatch(attrs); m != nil {
			hasFor := regexp.MustCompile(`<label[^>]*\bfor="` + regexp.QuoteMeta(m[1]) + `"`).MatchString(b)
			if !hasFor && !wrapped {
				fail("%s#%s has no label", tag, m[1])
			}
			continue
		}
		if !wrapped {
			fail("%s without id is not wrapped in a label: %s", tag, attrs)
		}
	}
	for _, m := range regexp.MustCompile(`(?i)<label[^>]*\bfor="([^"]+)"`).FindAllStringSubmatch(b, -1) {
		if !strings.Contains(b, `id="`+m[1]+`"`) {
			fail("label for=%s points nowhere", m[1])
		}
	}
	// Tables: header cells with scope; images: alt.
	if strings.Contains(b, "<table") && !strings.Contains(b, `scope="col"`) {
		fail("table without column headers")
	}
	if regexp.MustCompile(`(?i)<img\b(?:[^>]*)>`).MatchString(b) && !strings.Contains(b, "alt=") {
		fail("image without alt")
	}
	// Buttons and links have text.
	for _, m := range regexp.MustCompile(`(?is)<(a|button)\b[^>]*>(.*?)</(a|button)>`).FindAllStringSubmatch(b, -1) {
		if strings.TrimSpace(tagRe.ReplaceAllString(m[2], "")) == "" {
			fail("empty %s", m[1])
		}
	}
	// Copy rules (UX-050): no value words.
	text := strings.ToLower(html.UnescapeString(tagRe.ReplaceAllString(b, " ")))
	text = strings.NewReplacer("api tokens", "", "api token", "").Replace(text) // the credential kind, not a unit of value
	for _, bad := range []string{"money", "cash", "coins", "tokens", "investment"} {
		if strings.Contains(text, bad) {
			fail("forbidden word %q", bad)
		}
	}
	// No inline script/style, no external resources.
	if regexp.MustCompile(`(?i)<script\b[^>]*>[^<]`).MatchString(b) || strings.Contains(b, " style=") || regexp.MustCompile(`(src|href)="https?://`).MatchString(b) {
		fail("inline or external code/style")
	}
	if !strings.Contains(b, `name="viewport"`) {
		fail("no viewport meta")
	}
}

func TestPageStructure(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	w.env.Trade(w.net.ID, w.bob, w.alice, w.bobM, w.aliceM, 200)
	pend, err := w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{CounterpartyID: w.bobM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: "k-struct"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.env.Svc.CreateListing(w.env.Ctx, w.alice, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike repair", IdempotencyKey: "l1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.env.Svc.OpenDispute(w.env.Ctx, w.bob, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "quality", Statement: "meh", IdempotencyKey: "d1"}); err != nil {
		t.Fatal(err)
	}
	ds, _ := w.env.Svc.ListDisputes(w.env.Ctx, w.admin, w.net.ID, nil, service.Page{})

	public := []string{"/login", "/register", "/help", "/join/sometoken", "/no/such/page"}
	br := w.browser(t)
	for _, path := range public {
		checkStructure(t, path, br.get(path))
	}

	member := []string{"/", b, b + "/trades", b + "/trades/new?to=" + w.bobM.ID, b + "/trades/" + tx.ID, b + "/trades/" + pend.ID + "/cancel",
		b + "/history", b + "/directory?q=bo", b + "/members/" + w.bobM.ID, b + "/listings", b + "/listings?mine=1", b + "/listings/new",
		b + "/disputes", b + "/disputes/" + ds[0].ID, "/setup", b + "/join", b + "/nothing"}
	a := w.browser(t)
	a.login("alice@example.org")
	for _, path := range member {
		checkStructure(t, path, a.get(path))
	}
	// Bob has the pending proposal to accept.
	bb := w.browser(t)
	bb.login("bob@example.org")
	for _, path := range []string{b + "/trades/" + pend.ID + "/accept", b + "/trades/" + pend.ID + "/decline", b + "/trades/" + pend.ID, b + "/trades/" + w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 300).ID + "/dispute"} {
		checkStructure(t, path, bb.get(path))
	}

	adm := w.browser(t)
	adm.login("admin@example.org")
	for _, path := range []string{b + "/admin", b + "/admin/members", b + "/admin/members?status=active", b + "/admin/members/" + w.bobM.ID + "/suspend",
		b + "/admin/members/" + w.bobM.ID + "/limit", b + "/admin/invitations", b + "/admin/policy", b + "/admin/adjust", b + "/admin/audit",
		b + "/admin/exports", b + "/review"} {
		checkStructure(t, path, adm.get(path))
	}
	// Review and error states.
	f := adm.get(b + "/admin/adjust")
	rev := adm.post(b+"/admin/adjust", f, url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"5"}, "reason": {"fix"}, "step": {"review"}})
	checkStructure(t, "adjust review", rev)
	errp := adm.post(b+"/admin/adjust", f, url.Values{"step": {"review"}})
	checkStructure(t, "adjust errors", errp)
	checkStructure(t, "login error", br.post("/login", br.get("/login"), url.Values{"login": {"x"}, "password": {"y"}}))
	chk := adm.post(b+"/admin/check", adm.get(b+"/admin/audit"), nil)
	checkStructure(t, "check result", chk)
}

func TestFormLabelsAndAutocomplete(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	p := b.get("/login")
	b.mustContain(p, `autocomplete="username"`, `autocomplete="current-password"`, `<label for="f-login">Login</label>`, `type="password"`)
	if strings.Contains(p.Body, "onpaste") {
		t.Error("paste must be allowed")
	}
	r := b.get("/register")
	b.mustContain(r, `autocomplete="new-password"`, "At least 12 characters")
}
