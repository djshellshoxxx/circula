package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type harness struct {
	t      *testing.T
	st     *MemStore
	tr     *Tracker
	h      *Handler
	mu     sync.Mutex
	actors map[string]Actor
	facts  Facts
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	hs := &harness{t: t, st: NewMemStore(), actors: map[string]Actor{
		"s-ana":   {NetworkID: "net-a", MemberID: "ana", Active: true},
		"s-ben":   {NetworkID: "net-a", MemberID: "ben", Active: true},
		"s-ana-b": {NetworkID: "net-b", MemberID: "ana", Active: true},
		"s-adm":   {NetworkID: "net-a", MemberID: "adm", Active: true, Admin: true},
		"s-sus":   {NetworkID: "net-a", MemberID: "sus", Active: false},
		"s-fr":    {NetworkID: "net-a", MemberID: "fred", Active: true, Locale: "fr-CA"},
	}}
	hs.tr = newTracker(t, hs.st)
	h, err := NewHandler(HandlerConfig{
		Tracker: hs.tr,
		Authenticate: func(r *http.Request) (Actor, error) {
			c, err := r.Cookie("sid")
			if err != nil {
				return Actor{}, ErrUnauthenticated
			}
			hs.mu.Lock()
			defer hs.mu.Unlock()
			a, ok := hs.actors[c.Value]
			if !ok {
				return Actor{}, ErrUnauthenticated
			}
			return a, nil
		},
		CSRF: HMACCSRF{Key: bytes.Repeat([]byte{7}, 32), Session: func(r *http.Request) string {
			c, _ := r.Cookie("sid")
			if c == nil {
				return ""
			}
			return c.Value
		}},
		Facts:    func(ctx2 context.Context, a Actor) (Facts, error) { return hs.facts, nil },
		BasePath: "/onboarding",
		StepLinks: map[Step]string{
			Terms: "/terms", Listing: "/governance/listings", Trade: "https://example.org/proposals",
		},
		PracticeURL: "https://example.org/practice/",
		Links:       LinkPolicy{PathPrefixes: []string{"/governance/", "/terms"}, Hosts: []string{"example.org"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	hs.h = h
	return hs
}

type req struct {
	method, path, sid, ctype string
	form                     url.Values
	body                     string
	hdr                      map[string]string
}

func (hs *harness) do(q req) *httptest.ResponseRecorder {
	hs.t.Helper()
	var body *strings.Reader
	switch {
	case q.body != "":
		body = strings.NewReader(q.body)
	case q.form != nil:
		body = strings.NewReader(q.form.Encode())
	default:
		body = strings.NewReader("")
	}
	r := httptest.NewRequest(q.method, "http://circula.test"+q.path, body)
	if q.sid != "" {
		r.AddCookie(&http.Cookie{Name: "sid", Value: q.sid})
	}
	if q.form != nil || q.body != "" {
		ct := q.ctype
		if ct == "" {
			ct = "application/x-www-form-urlencoded"
		}
		r.Header.Set("Content-Type", ct)
	}
	for k, v := range q.hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	hs.h.ServeHTTP(w, r)
	return w
}

func (hs *harness) get(sid, path string) *httptest.ResponseRecorder {
	return hs.do(req{method: "GET", path: path, sid: sid})
}

func csrfOf(t *testing.T, doc string) string {
	t.Helper()
	for _, e := range byName(parseHTML(t, doc), "input") {
		if e.Attrs["name"] == "csrf" {
			return e.Attrs["value"]
		}
	}
	t.Fatal("no csrf field")
	return ""
}

func (hs *harness) post(sid, path string, f url.Values) *httptest.ResponseRecorder {
	hs.t.Helper()
	pageHTML := hs.get(sid, "/onboarding/").Body.String()
	if f.Get("csrf") == "" && f.Get("noCSRF") == "" {
		f.Set("csrf", csrfOf(hs.t, pageHTML))
	}
	f.Del("noCSRF")
	return hs.do(req{method: "POST", path: path, sid: sid, form: f})
}

func progressForm(action, step string) url.Values {
	f := url.Values{"curriculum": {"1"}, "action": {action}}
	if step != "" {
		f.Set("step", step)
	}
	return f
}

func TestChecklistPageRendersAccessibly(t *testing.T) {
	hs := newHarness(t)
	w := hs.get("s-ana", "/onboarding/")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	doc := w.Body.String()
	els := checkA11y(t, doc)
	for _, hdr := range []string{"Cache-Control", "Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
		if w.Header().Get(hdr) == "" {
			t.Errorf("missing %s", hdr)
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Error(w.Header())
	}
	var items []el
	for _, e := range els {
		if e.Name == "li" && strings.HasPrefix(e.Attrs["id"], "step-") {
			items = append(items, e)
		}
	}
	if len(items) != int(StepCount) {
		t.Fatalf("%d task items", len(items))
	}
	// Upstream-derived semantics: link describedby hint+status; unique ids.
	for _, e := range byName(els, "a") {
		if strings.Contains(e.Attrs["class"], "onb-task-list__link") {
			d := strings.Fields(e.Attrs["aria-describedby"])
			if len(d) != 2 || !strings.HasSuffix(d[0], "-hint") || !strings.HasSuffix(d[1], "-status") {
				t.Errorf("describedby %v", d)
			}
		}
	}
	// Steps with no trusted link render as plain text (no <a>), as upstream does.
	if strings.Count(doc, `class="onb-task-list__link"`) != 3 {
		t.Errorf("expected 3 linked steps")
	}
	// Buttons carry the step name for screen readers.
	if !strings.Contains(doc, `Skip<span class="onb-visually-hidden">: Add an offer or a request</span>`) {
		t.Error("button lacks step context")
	}
	if strings.Contains(strings.ToLower(doc), "<script") || strings.Contains(doc, "style=") {
		t.Error("script/style attribute")
	}
	if strings.Contains(doc, "Practice —") && !strings.Contains(doc, "practice-help") {
		t.Error("unexpected practice banner on real page")
	}
	if !strings.Contains(doc, `href="https://example.org/practice/"`) {
		t.Error("practice link missing")
	}
	if strings.Contains(doc, "/onboarding/admin") {
		t.Error("admin link shown to member")
	}
}

func TestAuthenticationAndAuthorizationOnEveryRoute(t *testing.T) {
	hs := newHarness(t)
	routes := []struct{ m, p string }{{"GET", "/onboarding/"}, {"POST", "/onboarding/progress"}, {"GET", "/onboarding/admin"}, {"POST", "/onboarding/admin/copy"}, {"POST", "/onboarding/admin/metrics"}}
	for _, rt := range routes {
		w := hs.do(req{method: rt.m, path: rt.p, form: func() url.Values {
			if rt.m == "POST" {
				return url.Values{"x": {"y"}}
			}
			return nil
		}()})
		if w.Code != 401 {
			t.Errorf("%s %s anonymous: %d", rt.m, rt.p, w.Code)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s: auth failure cacheable", rt.m, rt.p)
		}
		w = hs.do(req{method: rt.m, path: rt.p, sid: "s-sus", form: func() url.Values {
			if rt.m == "POST" {
				return url.Values{"x": {"y"}}
			}
			return nil
		}()})
		if w.Code != 403 {
			t.Errorf("%s %s suspended: %d", rt.m, rt.p, w.Code)
		}
		w = hs.do(req{method: rt.m, path: rt.p, sid: "forged"})
		if w.Code != 401 {
			t.Errorf("%s %s forged session: %d", rt.m, rt.p, w.Code)
		}
	}
	if w := hs.get("s-ana", "/onboarding/admin"); w.Code != 403 {
		t.Errorf("member reached admin: %d", w.Code)
	}
	// authorization is re-evaluated per request: suspension takes effect immediately
	if w := hs.get("s-ben", "/onboarding/"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	hs.mu.Lock()
	a := hs.actors["s-ben"]
	a.Active = false
	hs.actors["s-ben"] = a
	hs.mu.Unlock()
	if w := hs.get("s-ben", "/onboarding/"); w.Code != 403 {
		t.Errorf("stale authorization honoured: %d", w.Code)
	}
}

func TestPostFlowSkipResumeAndAccountSwitch(t *testing.T) {
	hs := newHarness(t)
	w := hs.post("s-ana", "/onboarding/progress", progressForm("complete", "terms"))
	if w.Code != 303 || w.Header().Get("Location") != "/onboarding/#step-terms" {
		t.Fatal(w.Code, w.Header())
	}
	hs.post("s-ana", "/onboarding/progress", progressForm("skip", "partner"))
	doc := hs.get("s-ana", "/onboarding/").Body.String()
	if !strings.Contains(doc, "1 of 5 steps done") || !strings.Contains(doc, ">Skipped<") || !strings.Contains(doc, ">Marked as done<") {
		t.Fatalf("state not reflected:\n%s", doc)
	}
	if !strings.Contains(doc, `value="resume"`) {
		t.Error("no resume control for skipped step")
	}
	hs.post("s-ana", "/onboarding/progress", progressForm("resume", "partner"))
	if doc := hs.get("s-ana", "/onboarding/").Body.String(); strings.Contains(doc, ">Skipped<") {
		t.Error("resume failed")
	}
	// idempotent replay of a form (browser back + resubmit)
	hs.post("s-ana", "/onboarding/progress", progressForm("complete", "terms"))
	p, _ := hs.tr.Progress(ctx, hs.actors["s-ana"])
	if p.Revision != 3 || p.Completed != 1 {
		t.Fatalf("%+v", p)
	}
	// account switch: other member and same member id in another network are untouched
	for _, sid := range []string{"s-ben", "s-ana-b"} {
		doc := hs.get(sid, "/onboarding/").Body.String()
		if !strings.Contains(doc, "0 of 5 steps done") {
			t.Errorf("%s sees someone else's progress", sid)
		}
	}
	// dismiss and restore
	hs.post("s-ana", "/onboarding/progress", progressForm("dismiss", ""))
	doc = hs.get("s-ana", "/onboarding/").Body.String()
	if strings.Contains(doc, "onb-task-list__item") || !strings.Contains(doc, `value="restore"`) {
		t.Error("dismissed state")
	}
	checkA11y(t, doc)
	hs.post("s-ana", "/onboarding/progress", progressForm("restore", ""))
	if doc := hs.get("s-ana", "/onboarding/").Body.String(); !strings.Contains(doc, "onb-task-list__item") {
		t.Error("restore")
	}
}

func TestPostAttackSurface(t *testing.T) {
	hs := newHarness(t)
	token := csrfOf(t, hs.get("s-ana", "/onboarding/").Body.String())
	good := func() url.Values { f := progressForm("complete", "terms"); f.Set("csrf", token); return f }
	assertUnchanged := func(why string) {
		t.Helper()
		if p, _ := hs.tr.Progress(ctx, hs.actors["s-ana"]); p.Completed != 0 {
			t.Errorf("%s: state changed", why)
		}
	}
	cases := []struct {
		name string
		q    req
		want int
	}{
		{"no csrf", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: progressForm("complete", "terms")}, 403},
		{"wrong csrf", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := progressForm("complete", "terms"); f.Set("csrf", "x"); return f }()}, 403},
		{"csrf of other session", req{method: "POST", path: "/onboarding/progress", sid: "s-ben", form: good()}, 403},
		{"cross origin", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: good(), hdr: map[string]string{"Origin": "https://evil.example"}}, 403},
		{"null origin", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: good(), hdr: map[string]string{"Origin": "null"}}, 403},
		{"sec-fetch-site cross", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: good(), hdr: map[string]string{"Sec-Fetch-Site": "cross-site"}}, 403},
		{"json content type", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", body: `{"action":"complete"}`, ctype: "application/json"}, 415},
		{"multipart", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", body: "--b\r\n", ctype: "multipart/form-data; boundary=b"}, 415},
		{"query string action ignored", req{method: "POST", path: "/onboarding/progress?action=complete&step=terms&curriculum=1&csrf=" + token, sid: "s-ana", form: url.Values{"x": {"1"}}}, 403},
		{"oversize", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", body: "csrf=" + token + "&pad=" + strings.Repeat("a", 70<<10)}, 413},
		{"bad step", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := good(); f.Set("step", "../x"); return f }()}, 400},
		{"step missing", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := good(); f.Del("step"); return f }()}, 400},
		{"bad action", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := good(); f.Set("action", "delete"); return f }()}, 400},
		{"stale curriculum", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := good(); f.Set("curriculum", "9"); return f }()}, 409},
		{"non-numeric curriculum", req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: func() url.Values { f := good(); f.Set("curriculum", "x"); return f }()}, 400},
		{"GET on POST route", req{method: "GET", path: "/onboarding/progress", sid: "s-ana"}, 405},
		{"PUT on checklist", req{method: "PUT", path: "/onboarding/", sid: "s-ana"}, 405},
		{"unknown path", req{method: "GET", path: "/onboarding/nope", sid: "s-ana"}, 404},
		{"traversal", req{method: "GET", path: "/onboarding/assets/../admin", sid: "s-ana"}, 404},
		{"outside base", req{method: "GET", path: "/other", sid: "s-ana"}, 404},
	}
	for _, c := range cases {
		w := hs.do(c.q)
		if w.Code != c.want {
			t.Errorf("%s: got %d want %d", c.name, w.Code, c.want)
		}
		if strings.Contains(w.Body.String(), "<script") {
			t.Errorf("%s: script in error page", c.name)
		}
	}
	assertUnchanged("attack matrix")
	// the same token, honestly submitted, works
	if w := hs.do(req{method: "POST", path: "/onboarding/progress", sid: "s-ana", form: good()}); w.Code != 303 {
		t.Fatal(w.Code)
	}
}

func TestAdminCopyEscapingAndLinkPolicy(t *testing.T) {
	hs := newHarness(t)
	xss := `<script>alert(1)</script><img src=x onerror=alert(2)> "quoted" & 'single'`
	f := url.Values{"revision": {"0"}, "intro": {xss}, "governance": {"/governance/charter"}, "terms": {"https://example.org/terms"}}
	w := hs.post("s-adm", "/onboarding/admin/copy", f)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, sid := range []string{"s-ana", "s-adm"} {
		doc := hs.get(sid, "/onboarding/").Body.String()
		if strings.Contains(doc, "<script>alert") || strings.Contains(doc, "<img src=x") {
			t.Fatalf("%s: unescaped admin text:\n%s", sid, doc)
		}
		if !strings.Contains(doc, "&lt;script&gt;alert(1)&lt;/script&gt;") {
			t.Error("text not shown escaped")
		}
		els := checkA11y(t, doc)
		var hrefs []string
		for _, e := range byName(els, "a") {
			hrefs = append(hrefs, e.Attrs["href"])
		}
		joined := strings.Join(hrefs, " ")
		if !strings.Contains(joined, "/governance/charter") || !strings.Contains(joined, "https://example.org/terms") {
			t.Errorf("links missing: %v", hrefs)
		}
	}
	// Hostile links are rejected with an accessible error summary and nothing is saved.
	bad := []string{
		"javascript:alert(1)", "JaVaScRiPt:alert(1)", "data:text/html,<script>", "//evil.example/x", "https://evil.example/x",
		"http://example.org/terms", "https://example.org:8443/x", "https://user@example.org/x", "/other/place", "/governance/../admin",
		"/governance/%2e%2e/admin", "/governance\\evil", "/governance/ x", "/terms\x00", "vbscript:x", " /terms", "/terms\n", "https://example.org.evil.example/",
		"https://EXAMPLE.org@evil.example/", "/governance/‮x", strings.Repeat("/terms/a", 100),
	}
	for _, b := range bad {
		for _, field := range []string{"governance", "terms"} {
			f := url.Values{"revision": {"1"}, "intro": {"ok"}, "governance": {""}, "terms": {""}}
			f.Set(field, b)
			w := hs.post("s-adm", "/onboarding/admin/copy", f)
			if b == " /terms" || b == "/terms\n" {
				// surrounding whitespace is trimmed by the form handler; the trimmed value is a valid link
				continue
			}
			if w.Code != 422 {
				t.Errorf("%s=%q accepted: %d", field, b, w.Code)
				continue
			}
			doc := w.Body.String()
			checkA11y(t, doc)
			if !strings.Contains(doc, `role="alert"`) || !strings.Contains(doc, `href="#`+field+`"`) {
				t.Errorf("no error summary for %s", field)
			}
			if strings.Contains(doc, "javascript:") && strings.Contains(doc, `href="javascript:`) {
				t.Error("hostile href rendered")
			}
		}
	}
	// Overlong / control-character text is rejected too.
	for _, intro := range []string{strings.Repeat("a", MaxIntroRunes+1), "x\x00y", "x‮y", "bad\xff"} {
		f := url.Values{"revision": {"1"}, "intro": {intro}}
		if w := hs.post("s-adm", "/onboarding/admin/copy", f); w.Code != 422 {
			t.Errorf("intro %q accepted: %d", intro[:min(len(intro), 8)], w.Code)
		}
	}
	// Members cannot write copy; admin of another network cannot touch net-a.
	if w := hs.post("s-ana", "/onboarding/admin/copy", url.Values{"revision": {"1"}, "intro": {"mine"}}); w.Code != 403 {
		t.Errorf("member wrote copy: %d", w.Code)
	}
	hs.mu.Lock()
	hs.actors["s-adm-b"] = Actor{NetworkID: "net-b", MemberID: "adm2", Active: true, Admin: true}
	hs.mu.Unlock()
	hs.post("s-adm-b", "/onboarding/admin/copy", url.Values{"revision": {"0"}, "intro": {"net-b text"}})
	if doc := hs.get("s-ana", "/onboarding/").Body.String(); strings.Contains(doc, "net-b text") {
		t.Error("copy leaked across networks")
	}
	if doc := hs.get("s-ana-b", "/onboarding/").Body.String(); !strings.Contains(doc, "net-b text") {
		t.Error("net-b copy missing")
	}
	// Stale revision (two admins editing) -> 409, nothing lost silently.
	if w := hs.post("s-adm", "/onboarding/admin/copy", url.Values{"revision": {"0"}, "intro": {"late"}}); w.Code != 409 {
		t.Errorf("stale admin form: %d", w.Code)
	}
	// Copy never alters steps/credit: step text is fixed by the catalog.
	doc := hs.get("s-ana", "/onboarding/").Body.String()
	if !strings.Contains(doc, "A balance can go below zero, down to your credit limit.") {
		t.Error("credit explanation changed by copy")
	}
}

func TestAdminMetricsPage(t *testing.T) {
	hs := newHarness(t)
	doc := hs.get("s-adm", "/onboarding/admin").Body.String()
	checkA11y(t, doc)
	if !strings.Contains(doc, "Counting is off.") {
		t.Fatal("metrics must default off")
	}
	if w := hs.post("s-ana", "/onboarding/admin/metrics", url.Values{"enabled": {"1"}}); w.Code != 403 {
		t.Fatal("member enabled metrics", w.Code)
	}
	if w := hs.post("s-adm", "/onboarding/admin/metrics", url.Values{"enabled": {"maybe"}}); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := hs.post("s-adm", "/onboarding/admin/metrics", url.Values{"enabled": {"1"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	for i := 0; i < 5; i++ {
		sid := fmt.Sprintf("s-m%d", i)
		hs.mu.Lock()
		hs.actors[sid] = Actor{NetworkID: "net-a", MemberID: fmt.Sprintf("m%d", i), Active: true}
		hs.mu.Unlock()
		hs.post(sid, "/onboarding/progress", progressForm("share", ""))
		hs.post(sid, "/onboarding/progress", progressForm("complete", "credit"))
	}
	doc = hs.get("s-adm", "/onboarding/admin").Body.String()
	checkA11y(t, doc)
	if !strings.Contains(doc, "Counting is on.") || !strings.Contains(doc, `<td class="onb-num">5</td>`) || !strings.Contains(doc, "Hidden (fewer than 5)") {
		t.Fatalf("aggregate table wrong:\n%s", doc)
	}
	for _, leak := range []string{"m0", "m1", "ana", "ben"} {
		if strings.Contains(doc, ">"+leak+"<") {
			t.Errorf("member identifier %q in aggregate page", leak)
		}
	}
	hs.post("s-adm", "/onboarding/admin/metrics", url.Values{"enabled": {"0"}})
	doc = hs.get("s-adm", "/onboarding/admin").Body.String()
	if strings.Contains(doc, "<table") {
		t.Error("counts shown after disabling")
	}
}

func TestFactsAreSeparateFromTutorialProgress(t *testing.T) {
	hs := newHarness(t)
	hs.facts = Facts{TermsAccepted: true, HasProposal: true}
	doc := hs.get("s-ana", "/onboarding/").Body.String()
	if strings.Count(doc, "Done in your account") != 2 || !strings.Contains(doc, "2 of 5 steps done") {
		t.Fatalf("facts not displayed")
	}
	if p, _ := hs.tr.Progress(ctx, hs.actors["s-ana"]); p.Completed != 0 {
		t.Fatal("facts leaked into tutorial progress")
	}
}

func TestFiveGuidedActionsReachTheTradeStep(t *testing.T) {
	hs := newHarness(t)
	actions := 0
	for {
		doc := hs.get("s-ben", "/onboarding/").Body.String()
		var next string
		for _, e := range parseHTML(t, doc) {
			if e.Name == "li" && strings.Contains(e.Attrs["class"], "onb-task-list__item--next") {
				next = strings.TrimPrefix(e.Attrs["id"], "step-")
			}
		}
		if next == "" {
			break
		}
		if w := hs.post("s-ben", "/onboarding/progress", progressForm("complete", next)); w.Code != 303 {
			t.Fatal(w.Code)
		}
		actions++
		if actions > 10 {
			t.Fatal("loop")
		}
	}
	if actions != 5 {
		t.Fatalf("guided actions = %d", actions)
	}
	if doc := hs.get("s-ben", "/onboarding/").Body.String(); !strings.Contains(doc, "All steps are done") {
		t.Error("completion message")
	}
}

func TestTranslationsAndLocaleFallback(t *testing.T) {
	for key := range en {
		if fr[key] == "" {
			t.Errorf("fr missing %q", key)
		}
	}
	for key := range fr {
		if en[key] == "" {
			t.Errorf("en missing %q", key)
		}
	}
	count := func(s string) string { return fmt.Sprint(strings.Count(s, "%d"), strings.Count(s, "%s")) }
	for k, v := range en {
		if count(v) != count(fr[k]) {
			t.Errorf("format verbs differ for %q", k)
		}
	}
	for _, tag := range []string{"fr", "FR_ca", "fr-CA", "en-GB", "", "zz", "../../x", "fr\x00"} {
		got := pickLocale(tag)
		if got != "fr" && got != "en" {
			t.Errorf("%q -> %q", tag, got)
		}
	}
	hs := newHarness(t)
	doc := hs.get("s-fr", "/onboarding/").Body.String()
	checkA11y(t, doc)
	if !strings.Contains(doc, `<html lang="fr">`) || !strings.Contains(doc, "Liste de bienvenue") || strings.Contains(doc, "Welcome checklist") {
		t.Error("French not served")
	}
	// A French admin text page still escapes.
	hs.mu.Lock()
	hs.actors["s-fr-adm"] = Actor{NetworkID: "net-a", MemberID: "fadm", Active: true, Admin: true, Locale: "fr"}
	hs.mu.Unlock()
	d := hs.get("s-fr-adm", "/onboarding/admin").Body.String()
	checkA11y(t, d)
	if !strings.Contains(d, "Texte et liens d&#39;accueil") {
		t.Error("French admin page")
	}
}

func TestCSSServedAndSafe(t *testing.T) {
	hs := newHarness(t)
	w := hs.get("", "/onboarding/assets/onboarding.css")
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/css; charset=utf-8" || w.Header().Get("ETag") == "" {
		t.Fatal(w.Code, w.Header())
	}
	css := w.Body.String()
	for _, bad := range []string{"@import", "url(", "expression(", "javascript:", "http://", "https://"} {
		if strings.Contains(css, bad) {
			t.Errorf("css contains %q", bad)
		}
	}
	if !strings.Contains(css, "prefers-reduced-motion") || !strings.Contains(css, "forced-colors") || !strings.Contains(css, "focus-visible") {
		t.Error("css lacks reduced-motion/forced-colors/focus rules")
	}
	r := httptest.NewRequest("GET", "/onboarding/assets/onboarding.css", nil)
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	w2 := httptest.NewRecorder()
	hs.h.ServeHTTP(w2, r)
	if w2.Code != 304 {
		t.Fatal(w2.Code)
	}
	if w := hs.do(req{method: "POST", path: "/onboarding/assets/onboarding.css", form: url.Values{}}); w.Code != 405 {
		t.Fatal(w.Code)
	}
}

func TestNewHandlerRejectsBadConfig(t *testing.T) {
	base := HandlerConfig{Tracker: newTracker(t, NewMemStore()), Authenticate: func(*http.Request) (Actor, error) { return Actor{}, nil }, CSRF: HMACCSRF{}, BasePath: "/o"}
	if _, err := NewHandler(base); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*HandlerConfig){
		"no tracker":   func(c *HandlerConfig) { c.Tracker = nil },
		"no auth":      func(c *HandlerConfig) { c.Authenticate = nil },
		"no csrf":      func(c *HandlerConfig) { c.CSRF = nil },
		"empty base":   func(c *HandlerConfig) { c.BasePath = "" },
		"slash base":   func(c *HandlerConfig) { c.BasePath = "/o/" },
		"proto base":   func(c *HandlerConfig) { c.BasePath = "//evil" },
		"bad step":     func(c *HandlerConfig) { c.StepLinks = map[Step]string{Terms: "javascript:alert(1)"} },
		"open step":    func(c *HandlerConfig) { c.StepLinks = map[Step]string{Terms: "https://evil.example/"} },
		"bad range":    func(c *HandlerConfig) { c.StepLinks = map[Step]string{StepCount: "/x"} },
		"bad practice": func(c *HandlerConfig) { c.PracticeURL = "https://evil.example/" },
	} {
		c := base
		mut(&c)
		if _, err := NewHandler(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// Empty HMAC CSRF never validates (fails closed).
	if (HMACCSRF{}).Verify(Actor{}, httptest.NewRequest("GET", "/", nil), "") {
		t.Fatal("zero HMACCSRF verifies")
	}
	if (HMACCSRF{Key: make([]byte, 31), Session: func(*http.Request) string { return "s" }}).Token(Actor{}, httptest.NewRequest("GET", "/", nil)) != "" {
		t.Fatal("short key accepted")
	}
}

func TestCSRFTokenBoundToActorAndSession(t *testing.T) {
	c := HMACCSRF{Key: bytes.Repeat([]byte{1}, 32), Session: func(r *http.Request) string { return r.Header.Get("S") }}
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.Header.Set("S", "one")
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.Header.Set("S", "two")
	a := Actor{NetworkID: "n", MemberID: "m"}
	tok := c.Token(a, r1)
	if !c.Verify(a, r1, tok) || c.Verify(a, r2, tok) || c.Verify(Actor{NetworkID: "n", MemberID: "x"}, r1, tok) || c.Verify(Actor{NetworkID: "o", MemberID: "m"}, r1, tok) {
		t.Fatal("token binding")
	}
}

// ---- JSON API ----

func (hs *harness) api(method, sid, body, csrf string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://circula.test/api/v1/me/onboarding-progress", strings.NewReader(body))
	if sid != "" {
		r.AddCookie(&http.Cookie{Name: "sid", Value: sid})
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	hs.h.APIHandler().ServeHTTP(w, r)
	return w
}

func (hs *harness) token(sid string) string {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sid", Value: sid})
	return hs.h.cfg.CSRF.Token(hs.actors[sid], r)
}

func TestJSONAPI(t *testing.T) {
	hs := newHarness(t)
	if w := hs.api("GET", "", "", "", nil); w.Code != 401 || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatal(w.Code, w.Header())
	}
	if w := hs.api("GET", "s-sus", "", "", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w := hs.api("GET", "s-ana", "", "", nil)
	var got apiProgress
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != 200 || got.Authoritative || len(got.Steps) != 5 || got.Next != "terms" || got.Curriculum != 1 {
		t.Fatalf("%d %s %v", w.Code, w.Body.String(), err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("api cacheable")
	}
	tok := hs.token("s-ana")
	body := `{"curriculum":1,"action":"complete","step":"terms"}`
	if w := hs.api("PATCH", "s-ana", body, "", nil); w.Code != 403 {
		t.Fatalf("PATCH without csrf: %d", w.Code)
	}
	if w := hs.api("PATCH", "s-ana", body, tok, map[string]string{"Origin": "https://evil.example"}); w.Code != 403 {
		t.Fatalf("cross origin: %d", w.Code)
	}
	for i := 0; i < 2; i++ { // idempotent
		w = hs.api("PATCH", "s-ana", body, tok, nil)
		got = apiProgress{}
		json.Unmarshal(w.Body.Bytes(), &got)
		if w.Code != 200 || !got.Steps[0].Completed || got.Next != "credit" || got.Revision != 1 {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
	bad := map[string]struct {
		body string
		want int
	}{
		"unknown field":   {`{"curriculum":1,"action":"complete","step":"terms","admin":true}`, 400},
		"bad json":        {`{`, 400},
		"trailing":        {`{"curriculum":1,"action":"skip","step":"credit"} {}`, 400},
		"bad step":        {`{"curriculum":1,"action":"complete","step":"x"}`, 400},
		"bad action":      {`{"curriculum":1,"action":"drop","step":"credit"}`, 400},
		"stale":           {`{"curriculum":2,"action":"complete","step":"credit"}`, 409},
		"huge":            {`{"curriculum":1,"action":"complete","step":"` + strings.Repeat("a", 70<<10) + `"}`, 413},
		"wrong type":      {`{"curriculum":"1","action":"complete","step":"credit"}`, 400},
		"network in body": {`{"curriculum":1,"action":"complete","step":"credit","network_id":"net-b"}`, 400},
	}
	for name, c := range bad {
		if w := hs.api("PATCH", "s-ana", c.body, tok, nil); w.Code != c.want || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s: %d (%s)", name, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("PATCH", "/", strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "sid", Value: "s-ana"})
	r.Header.Set("X-CSRF-Token", tok)
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	hs.h.APIHandler().ServeHTTP(w, r)
	if w.Code != 415 {
		t.Errorf("content type: %d", w.Code)
	}
	if w := hs.api("DELETE", "s-ana", "", "", nil); w.Code != 405 || w.Header().Get("Allow") == "" {
		t.Errorf("delete: %d", w.Code)
	}
	// still isolated
	if p, _ := hs.tr.Progress(ctx, hs.actors["s-ben"]); p.Completed != 0 {
		t.Fatal("api leaked to other member")
	}
	if p, _ := hs.tr.Progress(ctx, hs.actors["s-ana"]); p.Completed != 1 || p.Skipped != 0 {
		t.Fatalf("%+v", p)
	}
}
