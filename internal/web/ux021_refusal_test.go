package web

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// domainCodes parses the domain package and returns every declared Code value, so a code
// added later without a web mapping fails the table test below.
func domainCodes(t *testing.T) []domain.Code {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "domain", "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("domain sources: %v", err)
	}
	fset := token.NewFileSet()
	seen := map[domain.Code]bool{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, s := range g.Specs {
				vs := s.(*ast.ValueSpec)
				typed := false
				if id, ok := vs.Type.(*ast.Ident); ok && id.Name == "Code" {
					typed = true
				}
				for _, v := range vs.Values {
					lit, ok := v.(*ast.BasicLit)
					if call, isCall := v.(*ast.CallExpr); isCall {
						if fn, ok2 := call.Fun.(*ast.Ident); ok2 && fn.Name == "Code" && len(call.Args) == 1 {
							lit, ok = call.Args[0].(*ast.BasicLit)
							typed = ok
						}
					}
					if !typed || !ok || lit.Kind != token.STRING {
						continue
					}
					c, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatal(err)
					}
					seen[domain.Code(c)] = true
				}
			}
		}
	}
	out := make([]domain.Code, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// refusalRC is a request context for alice in w, as the network pages build it.
func refusalRC(w *world) *rc {
	s := newServer(w.env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: w.env.Clock})
	return &rc{s: s, ctx: w.env.Ctx, actor: w.alice, Net: w.net, Me: w.aliceM}
}

const internalDetail = "internal detail zq9"

// UX-020/UX-021: every domain error code a web form can meet is either a form refusal in
// plain words that says nothing changed and what to do next, or a page-level answer from fail.
// No code reaches the member as its machine name or with internal detail.
func TestRefusalCoversEveryCode(t *testing.T) {
	w := newWorld(t)
	c := refusalRC(w)
	type pageWant struct {
		status   int
		contains string // in the body, or in the Location header for a redirect
	}
	form := map[domain.Code]string{
		domain.CodeValidation:         "Fix the problems below",
		domain.CodeInsufficientCredit: "Available to spend is",
		domain.CodeAmountExceeds:      "largest single trade",
		domain.CodeCounterpartyInact:  "not active in this network",
		domain.CodeKeyReused:          "Reload the page and start again",
		domain.CodeKeyRequired:        "Reload the page and try again",
		domain.CodeInvalidState:       "Reload to see its current state",
		domain.CodeConflict:           "Choose a different one",
		domain.CodeBlocked:            "first blocker; second blocker",
		domain.CodeRateLimited:        "Wait a minute and try again",
		domain.CodeApprovalRequired:   "a second administrator approves",
	}
	pages := map[domain.Code]pageWant{
		domain.CodeUnauthenticated:        {http.StatusSeeOther, "/login"},
		domain.CodeForbidden:              {http.StatusForbidden, "You do not have access to this page."},
		domain.CodeNotFound:               {http.StatusNotFound, "We could not find that page."},
		domain.CodeNetworkFrozen:          {423, "Balances and history can still be viewed. Nothing was changed."},
		domain.CodeUnavailable:            {http.StatusServiceUnavailable, "Wait a few seconds and try again."},
		domain.CodeInternal:               {http.StatusInternalServerError, "quote the reference below"},
		domain.CodePasswordChangeRequired: {http.StatusSeeOther, "/account/password"},
		domain.CodeTermsRequired:          {http.StatusSeeOther, "/n/town/terms"},
		// Only the password-only service.Login returns it; the web signs in with LoginStep
		// (checked below), so a member never meets it. Should it ever escape, it is a plain 500.
		domain.CodeMFARequired: {http.StatusInternalServerError, "quote the reference below"},
	}
	codes := domainCodes(t)
	if len(codes) < 20 {
		t.Fatalf("parsed only %d codes: %v", len(codes), codes)
	}
	for _, code := range codes {
		err := &domain.Error{Code: code, Msg: internalDetail, Blockers: []string{"first blocker", "second blocker"},
			Fields: []domain.FieldError{{Field: "amount", Detail: "must be greater than zero"}}}
		sum, fields, ok := c.refuse(err)
		want, isForm := form[code]
		pw, isPage := pages[code]
		switch {
		case isForm == isPage:
			t.Errorf("%s: the table must list the code exactly once (form %v, page %v)", code, isForm, isPage)
			continue
		case isForm:
			if !ok {
				t.Errorf("%s: not mapped by refuse", code)
				continue
			}
			text := sum
			for _, f := range fields {
				text += " " + f.Text
			}
			for _, need := range []string{want, "Nothing changed."} {
				if !strings.Contains(text, need) {
					t.Errorf("%s: refusal %q lacks %q", code, text, need)
				}
			}
			if strings.Contains(text, string(code)) || strings.Contains(text, internalDetail) {
				t.Errorf("%s: refusal leaks the code or internal detail: %q", code, text)
			}
		default:
			if ok {
				t.Errorf("%s: refuse maps a page-level code as a form refusal: %q", code, sum)
				continue
			}
			req := httptest.NewRequest("POST", "/n/town/trades/new", nil)
			req.SetPathValue("slug", "town")
			rec := httptest.NewRecorder()
			c.s.fail(rec, req, err)
			body := rec.Body.String()
			if rec.Code != pw.status {
				t.Errorf("%s: fail status %d, want %d", code, rec.Code, pw.status)
			}
			if got := rec.Header().Get("Location") + body; !strings.Contains(got, pw.contains) {
				t.Errorf("%s: fail answer lacks %q: %s %s", code, pw.contains, rec.Header().Get("Location"), snippet(body))
			}
			if strings.Contains(body, internalDetail) || strings.Contains(body, ">"+string(code)+"<") {
				t.Errorf("%s: page leaks the code or internal detail", code)
			}
		}
	}
	for code := range form {
		if !containsCode(codes, code) {
			t.Errorf("table lists %s, which the domain no longer declares", code)
		}
	}
	for code := range pages {
		if !containsCode(codes, code) {
			t.Errorf("table lists %s, which the domain no longer declares", code)
		}
	}
	// mfa-required stays unreachable from the web only while no handler uses the password-only Login.
	for _, f := range webSources(t) {
		if src, _ := os.ReadFile(f); strings.Contains(string(src), "svc.Login(") {
			t.Errorf("%s calls service.Login, which can return mfa-required; map that code first", f)
		}
	}
}

func containsCode(cs []domain.Code, c domain.Code) bool {
	for _, x := range cs {
		if x == c {
			return true
		}
	}
	return false
}

func webSources(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	return out
}

// validationField finds the field names the service and domain put in validation errors.
var validationField = regexp.MustCompile(`(?:Invalid|boundedText|cleanListingText|cleanLabel|add)\("([a-z_]+)"`)

// UX-021: a validation message names the field in words, keeps the service's description of
// the problem and suggests a correction, for every field the service can name.
func TestValidationMessagesNameFieldAndFix(t *testing.T) {
	w := newWorld(t)
	c := refusalRC(w)
	var srcs []string
	for _, dir := range []string{"service", "domain"} {
		fs, err := filepath.Glob(filepath.Join("..", dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		srcs = append(srcs, fs...)
	}
	fields := map[string]bool{}
	for _, f := range srcs {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range validationField.FindAllStringSubmatch(string(src), -1) {
			fields[m[1]] = true
		}
	}
	if len(fields) < 50 {
		t.Fatalf("found only %d validation fields", len(fields))
	}
	for f := range fields {
		if _, ok := fieldLabels[f]; !ok {
			t.Errorf("field %q has no plain label; the message would show %q", f, label(f))
			continue
		}
		_, fe, ok := c.refuse(domain.Invalid(f, "is not valid"))
		if !ok || len(fe) != 1 || fe[0].Field != f {
			t.Errorf("%s: refuse = %v %v", f, fe, ok)
			continue
		}
		text := fe[0].Text
		if !strings.HasPrefix(text, fieldLabels[f]+": is not valid. ") || !strings.HasSuffix(text, ".") ||
			len(text) <= len(fieldLabels[f]+": is not valid. ") {
			t.Errorf("%s: message %q does not name the field, the problem and a correction", f, text)
		}
	}

	cases := []struct {
		field, detail string
		noNet         bool
		want          string
	}{
		{"amount", "too many decimal places", false, "Amount: too many decimal places. Use at most 2 decimal places and try again."},
		{"amount", "not a number", false, "Amount: not a number. Enter digits only, for example 12.50."},
		{"amount", "not a number", true, "Amount: not a number. Check the value and try again."},
		{"amount", "too large", false, "Amount: too large. Enter a smaller amount and try again."},
		{"reason", "required", false, "Reason: required. Fill it in and try again."},
		{"category", "required, one of not_delivered, other", false, "Category: required, one of not_delivered, other. Fill it in and try again."},
		{"note", "too long", false, "Note: too long. Shorten it and try again."},
		{"password", "is too common; choose another", true, "Password: is too common; choose another. Choose a different one and try again."},
		{"password", "is not correct", true, "Password: is not correct. Type it again carefully and try again."},
		{"display_name", "must not contain control characters", false, "Display name: must not contain control characters. Remove hidden or special characters and try again."},
		{"counterparty_id", "cannot trade with yourself", false, "Counterparty: cannot trade with yourself. Choose another active member from the list."},
		{"direction", "must be pay or request", false, "Direction: must be pay or request. Choose whether you pay or are paid."},
		{"terms_version", "accept the network's current terms (version 2)", false, "Terms: accept the network's current terms (version 2). Read the terms shown, tick the box to accept them and try again."},
		{"title", "must be 1 to 120 characters", false, "Title: must be 1 to 120 characters. Check the value and try again."},
		{"time_zone", "must be an IANA time zone such as Europe/Berlin, or empty for UTC", true, "Time zone: must be an IANA time zone such as Europe/Berlin, or empty for UTC. Check the value and try again."},
	}
	for _, tc := range cases {
		cc := c
		if tc.noNet {
			cc = &rc{s: c.s, ctx: c.ctx, actor: c.actor}
		}
		sum, fe, ok := cc.refuse(domain.Invalid(tc.field, tc.detail))
		if !ok || len(fe) != 1 {
			t.Fatalf("%s/%s: refuse = %v %v", tc.field, tc.detail, fe, ok)
		}
		if fe[0].Text != tc.want {
			t.Errorf("%s/%s:\n got %q\nwant %q", tc.field, tc.detail, fe[0].Text, tc.want)
		}
		if sum != "Not saved. Fix the problems below. Nothing changed." {
			t.Errorf("summary %q", sum)
		}
	}
	// A validation error with no field to point at still says what to do.
	sum, fe, ok := c.refuse(domain.Errf(domain.CodeValidation, "amount overflows payee balance"))
	if !ok || len(fe) != 0 || sum != "Not done. The amount is too large to record on these accounts. Try a smaller amount. Nothing changed." {
		t.Errorf("overflow refusal = %q %v %v", sum, fe, ok)
	}
	if sum, _, _ := c.refuse(domain.Errf(domain.CodeValidation, "something")); strings.Contains(sum, "below") || !strings.Contains(sum, "Nothing changed.") {
		t.Errorf("field-less refusal points at fields that are not shown: %q", sum)
	}
}

// UX-020: approval-required used to fall through to a "No access" page. Removing a member whose
// settle-to-zero adjustment is above the dual-control threshold now refuses on the form.
func TestRemoveAboveDualControlThresholdRefusesOnForm(t *testing.T) {
	w := newWorld(t)
	ctx := w.env.Ctx
	b := w.base()
	_, adm2M := w.env.Member(w.admin, w.net, "admin2@example.org")
	testutil.Must(0, w.env.Svc.AssignRole(ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 2000)
	n := testutil.Must(w.env.Svc.GetNetwork(ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	pol.DepartureRule = domain.DepartSettleToZero
	now := 0
	testutil.Must(w.env.Svc.UpdatePolicy(ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "test",
		NoticeDays: &now, NoticeReason: "test fixture", IdempotencyKey: w.env.Key()}))
	if n := testutil.Must(w.env.Svc.GetNetwork(ctx, w.admin, w.net.ID)); n.Policy.DepartureRule != domain.DepartSettleToZero || n.Policy.DualControlThreshold != 1000 {
		t.Fatalf("fixture policy not in force: %+v", n.Policy)
	}

	adm := w.browser(t)
	adm.login("admin@example.org")
	f := adm.get(b + "/admin/members/" + w.bobM.ID + "/remove")
	p := adm.post(b+"/admin/members/"+w.bobM.ID+"/remove", f, url.Values{"reason_code": {"rule_breach"}, "reason": {"breach of network rules"}})
	if p.Status != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", p.Status)
	}
	adm.mustContain(p, `id="error-summary"`, "dual-control threshold", "a second administrator approves", "Nothing changed.")
	if strings.Contains(p.Body, "You do not have access") {
		t.Error("the refusal is shown as No access")
	}
	if m := testutil.Must(w.env.Svc.GetMember(ctx, w.admin, w.net.ID, w.bobM.ID)); m.Status != domain.MemberActive {
		t.Errorf("bob is %s after a refused removal", m.Status)
	}
	if got := testutil.Must(w.env.Svc.GetBalance(ctx, w.bob, w.net.ID)).Balance; got != 2000 {
		t.Errorf("bob's balance = %d after a refused removal", got)
	}
}
