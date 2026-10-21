package web

import (
	"io/fs"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

var (
	idAttrRe   = regexp.MustCompile(`(?i)\sid="([^"]+)"`)
	describeRe = regexp.MustCompile(`(?i)\saria-(?:describedby|labelledby)="([^"]*)"`)
	summaryRe  = regexp.MustCompile(`(?s)<div id="error-summary".*?</div>`)
	hrefRe     = regexp.MustCompile(`href="#([^"]+)"`)
	formRe     = regexp.MustCompile(`(?is)<form\b([^>]*)>(.*?)</form>`)
	inputRe    = regexp.MustCompile(`(?is)<input\b([^>]*)>`)
)

func attr(attrs, name string) string {
	if m := attrRe(name).FindStringSubmatch(attrs); m != nil {
		return m[1]
	}
	return ""
}

// checkForms adds the form-level checks that apply to the account-security,
// recovery, dual-control, break-glass and erasure pages: unique ids, valid
// aria references, error summary links that land on a control, autocomplete
// hints on credentials and codes, a CSRF field in every POST form, and
// no positive tabindex.
func checkForms(t *testing.T, path string, p *pg) {
	t.Helper()
	fail := func(f string, a ...any) { t.Helper(); t.Errorf("%s: "+f, append([]any{path}, a...)...) }
	b := p.Body
	ids := map[string]int{}
	for _, m := range idAttrRe.FindAllStringSubmatch(b, -1) {
		ids[m[1]]++
	}
	for id, n := range ids {
		if n > 1 {
			fail("duplicate id %q", id)
		}
	}
	for _, m := range describeRe.FindAllStringSubmatch(b, -1) {
		for _, id := range strings.Fields(m[1]) {
			if ids[id] == 0 {
				fail("aria reference to missing id %q", id)
			}
		}
	}
	if s := summaryRe.FindString(b); s != "" {
		links := hrefRe.FindAllStringSubmatch(s, -1)
		if len(links) == 0 && !strings.Contains(s, "<p>") {
			fail("error summary has neither message nor links")
		}
		for _, m := range links {
			if ids[m[1]] == 0 {
				fail("error summary links to #%s, which does not exist", m[1])
			}
		}
		if !strings.Contains(s, `role="alert"`) || !strings.Contains(s, `tabindex="-1"`) {
			fail("error summary is not announced or focusable")
		}
	}
	for _, m := range inputRe.FindAllStringSubmatch(b, -1) {
		a := m[1]
		name, typ, ac := attr(a, "name"), strings.ToLower(attr(a, "type")), attr(a, "autocomplete")
		switch {
		case typ == "password" && ac != "current-password" && ac != "new-password":
			fail("password input %q lacks current-password/new-password autocomplete", name)
		case typ == "password" && strings.Contains(name, "current") && ac != "current-password":
			fail("input %q must be current-password, is %q", name, ac)
		case typ == "password" && (name == "password2" || name == "new_password") && ac != "new-password":
			fail("input %q must be new-password, is %q", name, ac)
		case name == "code" && ac != "one-time-code":
			fail("code input lacks autocomplete=one-time-code (got %q)", ac)
		case name == "recovery_code" && ac != "off":
			fail("recovery code input should not be autofilled (got %q)", ac)
		}
		if strings.Contains(strings.ToLower(a), "onpaste") {
			fail("input %q blocks paste", name)
		}
	}
	for _, f := range formRe.FindAllStringSubmatch(b, -1) {
		if strings.EqualFold(attr(f[1], "method"), "post") && !strings.Contains(f[2], `name="csrf"`) {
			fail("POST form %q has no csrf field", attr(f[1], "action"))
		}
	}
	if regexp.MustCompile(`tabindex="[1-9]`).MatchString(b) {
		fail("positive tabindex")
	}
	// A page that can be reached by a person must not show raw error codes.
	if strings.Contains(b, "&lt;nil&gt;") || strings.Contains(b, "%!") {
		fail("template formatting leaked into the page")
	}
}

func checkPage(t *testing.T, path string, p *pg) {
	t.Helper()
	checkStructure(t, path, p)
	checkForms(t, path, p)
}

// Walks the pages added for MFA, recovery, dual control, break-glass,
// erasure and security notices, in their normal and their error states.
func TestSecurityPagesAccessibility(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx

	// Dual control with a queued request and a break-glass record.
	_, adm2M := w.env.Member(w.admin, w.net, "admin2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	n := testutil.Must(w.env.Svc.GetNetwork(ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dual", IdempotencyKey: w.env.Key()}))
	testutil.Must(w.env.Svc.OpenBreakGlass(ctx, domain.SystemActor("t"), service.OpenBreakGlassInput{Network: "town", Operator: "Olga",
		Reason: "restore after disk fault", TTL: time.Hour}))

	adm := w.browser(t)
	adm.login("admin@example.org")
	f := adm.get(b + "/admin/adjust")
	vals := url.Values{"member_id": {w.bobM.ID}, "direction": {"credit"}, "amount": {"25.00"}, "reason": {"big correction"}, "step": {"review"}}
	rev := adm.post(b+"/admin/adjust", f, vals)
	vals.Set("step", "confirm")
	queued := adm.post(b+"/admin/adjust", rev, vals)
	checkPage(t, "adjust queued", queued)
	checkPage(t, "pending list", adm.get(b+"/admin/adjustments/pending"))
	checkPage(t, "pending list approved", adm.get(b+"/admin/adjustments/pending?status=approved"))
	checkPage(t, "break-glass", adm.get(b+"/admin/break-glass"))
	two := w.browser(t)
	two.login("admin2@example.org")
	checkPage(t, "pending list as approver", two.get(b+"/admin/adjustments/pending"))

	// Erasure, admin and self, including both error states.
	testutil.Must(w.env.Svc.DepartMember(ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bobM.ID, Reason: "left"}))
	checkPage(t, "departed list", adm.get(b+"/admin/members?status=departed"))
	ef := adm.get(b + "/admin/members/" + w.bobM.ID + "/erase")
	checkPage(t, "erase form", ef)
	checkPage(t, "erase no confirm", adm.post(b+"/admin/members/"+w.bobM.ID+"/erase", ef, url.Values{"reason": {"request"}}))
	checkPage(t, "erase no reason", adm.post(b+"/admin/members/"+w.bobM.ID+"/erase", ef, url.Values{"confirm": {"yes"}}))
	bob := w.browser(t)
	bob.login("bob@example.org")
	sf := bob.get("/erase-my-data")
	checkPage(t, "self erase form", sf)
	checkPage(t, "self erase error", bob.post("/erase-my-data", sf, url.Values{}))

	// Recovery codes, account recovery, password change.
	al := w.browser(t)
	al.login("alice@example.org")
	cf := al.get("/account/recovery-codes")
	checkPage(t, "recovery codes", cf)
	checkPage(t, "recovery codes bad password", al.post("/account/recovery-codes", cf, url.Values{"password": {"nope nope nope"}}))
	checkPage(t, "recovery codes shown", al.post("/account/recovery-codes", cf, url.Values{"password": {testPW}}))
	pw := al.get("/account/password")
	checkPage(t, "password", pw)
	checkPage(t, "password errors", al.post("/account/password", pw, url.Values{"current_password": {"x"}, "password": {"short"}, "password2": {"other"}}))
	anon := w.browser(t)
	rc := anon.get("/recover")
	checkPage(t, "recover", rc)
	checkPage(t, "recover errors", anon.post("/recover", rc, url.Values{"login": {"alice@example.org"}, "recovery_code": {"AAAA-AAAA-AAAA-AAAA"},
		"password": {"a brand new passphrase"}, "password2": {"different"}}))
	checkPage(t, "recover wrong code", anon.post("/recover", rc, url.Values{"login": {"alice@example.org"}, "recovery_code": {"AAAA-AAAA-AAAA-AAAA"},
		"password": {"a brand new passphrase"}, "password2": {"a brand new passphrase"}}))
	checkPage(t, "reset confirm", adm.get(b+"/admin/members/"+w.aliceM.ID+"/reset-password"))
	rf := adm.get(b + "/admin/members/" + w.aliceM.ID + "/reset-password")
	checkPage(t, "reset error", adm.post(b+"/admin/members/"+w.aliceM.ID+"/reset-password", rf, url.Values{"reason": {""}}))
	checkPage(t, "reset done", adm.post(b+"/admin/members/"+w.aliceM.ID+"/reset-password", rf, url.Values{"reason": {"lost it all"}}))
}

// MFA pages need an enrolled person; kept separate so the clock moves cleanly.
func TestMFAPagesAccessibility(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	checkPage(t, "security off", b.get("/account/security"))
	e1 := b.get("/account/security/mfa/enrol")
	checkPage(t, "enrol start", e1)
	e2 := b.post("/account/security/mfa/enrol", e1, nil)
	checkPage(t, "enrol key", e2)
	secret := secretRe.FindStringSubmatch(e2.Body)[1]
	e3 := b.post("/account/security/mfa/enrol/confirm", e2, url.Values{"code": {"000000"}})
	checkPage(t, "enrol bad code", e3)
	e4 := b.post("/account/security/mfa/enrol/confirm", e3, url.Values{"code": {webTOTP(t, secret, w.env.Clock.Now())}})
	checkPage(t, "enrol codes", e4)
	checkPage(t, "security on with notices", b.get("/account/security"))
	checkPage(t, "enrol when on", b.get("/account/security/mfa/enrol"))
	sec := b.get("/account/security")
	dis := b.post("/account/security/mfa/disable", sec, url.Values{"password": {testPW}, "code": {"1"}})
	checkPage(t, "disable bad", dis)
	// The two code fields on this page have distinct ids and the error lands on the one submitted.
	if !strings.Contains(dis.Body, `id="e-disable_code"`) || strings.Contains(dis.Body, `id="e-regen_code"`) ||
		!strings.Contains(dis.Body, `<label for="f-regen_code">`) || !strings.Contains(dis.Body, `<label for="f-disable_code">`) {
		t.Error("code fields on the security page are not told apart")
	}
	checkPage(t, "recovery regenerate bad", b.post("/account/security/mfa/recovery", sec, url.Values{"code": {"1"}}))
	w.env.Clock.Advance(31 * time.Second)

	c := w.browser(t)
	step := c.post("/login", c.get("/login"), url.Values{"login": {"alice@example.org"}, "password": {testPW}})
	checkPage(t, "login mfa", step)
	checkPage(t, "login mfa wrong", c.post("/login/mfa", step, url.Values{"challenge": {challengeOf(t, step)}, "code": {"111111"}}))
}

// Every template file is covered by a source-level check, so a page added
// later cannot skip the basics even before a flow test renders it.
func TestTemplateSources(t *testing.T) {
	rawInput := regexp.MustCompile(`(?is)<(input|select|textarea)\b([^>]*)>`)
	files, err := fs.Glob(assets, "templates/*.html")
	if err != nil || len(files) < 40 {
		t.Fatalf("templates not found: %v %d", err, len(files))
	}
	for _, name := range files {
		raw, err := fs.ReadFile(assets, name)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		fail := func(f string, a ...any) { t.Errorf("%s: "+f, append([]any{name}, a...)...) }
		if strings.Contains(src, "<h1") && name != "templates/base.html" {
			fail("page templates must not emit their own h1 (base does)")
		}
		if regexp.MustCompile(`(?i)<h[3-6]`).MatchString(src) && !regexp.MustCompile(`(?i)<h2`).MatchString(src) {
			fail("h3 or lower without an h2 before it")
		}
		if regexp.MustCompile(`(?i)\s(on[a-z]+|style)=`).MatchString(src) {
			fail("inline handler or style (CSP)")
		}
		if regexp.MustCompile(`(?i)<(b|i|font|center|blink|marquee)\b`).MatchString(src) {
			fail("presentational element")
		}
		for _, m := range rawInput.FindAllStringSubmatch(src, -1) {
			a := m[2]
			typ := strings.ToLower(attr(a, "type"))
			if typ == "hidden" || typ == "submit" {
				continue
			}
			if typ == "password" {
				fail("raw password input; use the field template (autocomplete, label, errors): %s", a)
			}
			if !strings.Contains(src, "<label") {
				fail("control without any label in the template: %s", a)
			}
		}
	}
}
