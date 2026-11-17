package onboarding

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type pclient struct {
	t      *testing.T
	srv    *PracticeServer
	cookie *http.Cookie
	host   string
	hdr    map[string]string
}

func newPracticeServer(t *testing.T, mut func(*PracticeConfig)) *PracticeServer {
	t.Helper()
	cfg := PracticeConfig{AllowedHosts: []string{"practice.circula.test"}, ProductionHosts: []string{"circula.test"},
		ProductionCookieNames: []string{"circula_session", "__Host-circula_session"}, ExitURL: "/", Links: LinkPolicy{PathPrefixes: []string{"/"}}}
	if mut != nil {
		mut(&cfg)
	}
	s, err := NewPracticeServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func newClient(t *testing.T, s *PracticeServer) *pclient {
	return &pclient{t: t, srv: s, host: "practice.circula.test"}
}

func (c *pclient) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	c.t.Helper()
	var r *http.Request
	if form != nil {
		r = httptest.NewRequest(method, "http://"+c.host+path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, "http://"+c.host+path, nil)
	}
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	for k, v := range c.hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	c.srv.ServeHTTP(w, r)
	for _, ck := range w.Result().Cookies() {
		if ck.Name == c.srv.CookieName() {
			if ck.MaxAge < 0 {
				c.cookie = nil
			} else {
				c.cookie = ck
			}
		}
	}
	return w
}

func (c *pclient) get(path string) *httptest.ResponseRecorder { return c.do("GET", path, nil) }

// form reads the page at path and returns the first <form> (by index) hidden inputs plus csrf.
func (c *pclient) hidden(doc string) url.Values {
	v := url.Values{}
	for _, e := range byName(parseHTML(c.t, doc), "input") {
		if e.Attrs["type"] == "hidden" {
			v.Set(e.Attrs["name"], e.Attrs["value"])
		}
	}
	return v
}

func (c *pclient) start() {
	c.t.Helper()
	if w := c.do("POST", "/start", url.Values{}); w.Code != 303 || c.cookie == nil {
		c.t.Fatalf("start: %d %v", w.Code, w.Header())
	}
}

// postStep fetches a step page to obtain the csrf+nonce, overlays fields, posts.
func (c *pclient) postStep(page, action string, extra url.Values) *httptest.ResponseRecorder {
	c.t.Helper()
	doc := c.get(page).Body.String()
	f := c.hidden(doc)
	for k, v := range extra {
		f[k] = v
	}
	return c.do("POST", action, f)
}

func TestPracticeConfigValidation(t *testing.T) {
	ok := PracticeConfig{AllowedHosts: []string{"practice.x"}, ProductionHosts: []string{"app.x"}, ProductionCookieNames: []string{"sess"}}
	if _, err := NewPracticeServer(ok); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*PracticeConfig){
		"no hosts":           func(c *PracticeConfig) { c.AllowedHosts = nil },
		"upper host":         func(c *PracticeConfig) { c.AllowedHosts = []string{"Practice.x"} },
		"host with path":     func(c *PracticeConfig) { c.AllowedHosts = []string{"practice.x/evil"} },
		"same origin":        func(c *PracticeConfig) { c.ProductionHosts = []string{"practice.x"} },
		"same origin (case)": func(c *PracticeConfig) { c.ProductionHosts = []string{"PRACTICE.x"} },
		"cookie collides":    func(c *PracticeConfig) { c.CookieName = "sess" },
		"prefixed collides": func(c *PracticeConfig) {
			c.Secure = true
			c.CookieName = "sess"
			c.ProductionCookieNames = []string{"__Host-sess"}
		},
		"bad base": func(c *PracticeConfig) { c.BasePath = "x" },
		"bad exit": func(c *PracticeConfig) {
			c.ExitURL = "https://evil.example/"
			c.Links = LinkPolicy{Hosts: []string{"app.x"}}
		},
	} {
		c := ok
		mut(&c)
		if _, err := NewPracticeServer(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	s, _ := NewPracticeServer(PracticeConfig{AllowedHosts: []string{"p.x"}, Secure: true})
	if s.CookieName() != "__Host-practice_session" {
		t.Fatal(s.CookieName())
	}
}

// The practice constructor cannot receive production handles by construction.
func TestPracticeConfigCannotCarryHandles(t *testing.T) {
	typ := reflect.TypeOf(PracticeConfig{})
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		switch f.Type.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Chan, reflect.UnsafePointer:
			t.Errorf("field %s has handle-capable kind %s", f.Name, f.Type.Kind())
		case reflect.Func:
			if f.Name != "Now" {
				t.Errorf("unexpected func field %s", f.Name)
			}
		}
		if strings.Contains(f.Type.PkgPath(), "database/sql") || strings.Contains(f.Name, "DB") || strings.Contains(f.Name, "DSN") {
			t.Errorf("suspicious field %s", f.Name)
		}
	}
	if reflect.TypeOf(NewPracticeServer).NumIn() != 1 || reflect.TypeOf(NewPracticeServer).In(0) != typ {
		t.Fatal("NewPracticeServer must take only PracticeConfig")
	}
}

func TestPracticeServerSourceHasNoProductionReach(t *testing.T) {
	forbidden := []string{"database/sql", "os", "os/exec", "io/ioutil", "syscall", "unsafe", "plugin", "reflect", "github.com", "golang.org"}
	af, err := parser.ParseFile(token.NewFileSet(), "practice_http.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, im := range af.Imports {
		p := strings.Trim(im.Path.Value, `"`)
		for _, bad := range forbidden {
			if p == bad || strings.HasPrefix(p, bad+"/") {
				t.Errorf("practice_http.go imports %s", p)
			}
		}
		if p == "net" || p == "net/rpc" || p == "net/smtp" || p == "net/http/httputil" {
			t.Errorf("practice_http.go imports raw network package %s", p)
		}
	}
}

func TestPracticeFullFlowLabelledAndAccessible(t *testing.T) {
	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	// Landing page before a session: banner already present.
	w := c.get("/")
	if w.Code != 200 || !strings.Contains(w.Body.String(), PracticeMarker) {
		t.Fatal(w.Code)
	}
	checkA11y(t, w.Body.String())
	if w.Header().Get("X-Practice") != "synthetic" || w.Header().Get("Cache-Control") != "no-store" {
		t.Error(w.Header())
	}
	c.start()
	ck := c.cookie
	if !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode || ck.Path != "/" || ck.Domain != "" || ck.MaxAge <= 0 || len(ck.Value) < 40 {
		t.Fatalf("cookie attrs %+v", ck)
	}
	// Every page carries the banner, passes structural a11y, and is not cacheable.
	for _, p := range []string{"/", "/step/terms", "/step/credit", "/step/listing", "/step/partner", "/step/trade"} {
		w := c.get(p)
		if w.Code != 200 {
			t.Fatalf("%s: %d", p, w.Code)
		}
		d := w.Body.String()
		if !strings.Contains(d, `id="practice-banner"`) || !strings.Contains(d, PracticeMarker) || !strings.Contains(d, `<span class="onb-visually-hidden">Warning</span>`) {
			t.Errorf("%s: banner missing", p)
		}
		if !strings.Contains(d, "<title>"+PracticeMarker) {
			t.Errorf("%s: title lacks practice marker", p)
		}
		if w.Header().Get("X-Practice") != "synthetic" {
			t.Errorf("%s: header", p)
		}
		checkA11y(t, d)
	}
	// 1 terms, 2 credit
	if w := c.postStep("/step/terms", "/step/terms", nil); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := c.postStep("/step/credit", "/step/credit", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	// 3 listing: invalid first (accessible errors), then valid
	w = c.postStep("/step/listing", "/step/listing", url.Values{"kind": {"offer"}, "title": {""}, "amount": {"abc"}})
	if w.Code != 422 || !strings.Contains(w.Body.String(), `role="alert"`) || !strings.Contains(w.Body.String(), `href="#title"`) || !strings.Contains(w.Body.String(), `href="#amount"`) {
		t.Fatalf("listing errors: %d", w.Code)
	}
	checkA11y(t, w.Body.String())
	if !strings.Contains(w.Body.String(), PracticeMarker) {
		t.Error("error page lost banner")
	}
	if w := c.postStep("/step/listing", "/step/listing", url.Values{"kind": {"offer"}, "title": {"Piano lessons"}, "amount": {"20"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	// 4 partner
	if w := c.postStep("/step/partner", "/step/partner", url.Values{"partner": {"practice-bo"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	// 5 propose: preview then send
	doc := c.get("/step/trade?listing=practice-listing-1").Body.String()
	checkA11y(t, doc)
	for _, want := range []string{"Balance preview: before and after", "This is only a preview. Nothing has changed yet.", "below zero. That is allowed", "reserves 30 credits"} {
		if !strings.Contains(doc, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
	f := c.hidden(doc)
	if w := c.do("POST", "/step/trade/propose", f); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	doc = c.get("/step/trade").Body.String()
	checkA11y(t, doc)
	if !strings.Contains(doc, "Proposal sent: not paid yet") || !strings.Contains(doc, "A proposal is not a payment. Your balance has not changed. 30 credits are reserved") {
		t.Fatal("pending explanation missing")
	}
	if snap := s.sessions[c.cookie.Value].world.Snapshot(); snap.Learner().Balance != 0 || snap.Learner().Reserved != 30 {
		t.Fatalf("%+v", snap.Learner())
	}
	// accept (scripted reply)
	var acc url.Values
	for _, form := range strings.Split(doc, "<form") {
		if strings.Contains(form, `name="accept" value="1"`) {
			acc = c.hidden("<form" + form)
		}
	}
	if acc == nil {
		t.Fatal("no accept form")
	}
	if w := c.do("POST", "/proposal/respond", acc); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	// duplicate submit of the same form (double click) is harmless
	c.do("POST", "/proposal/respond", acc)
	snap := s.sessions[c.cookie.Value].world.Snapshot()
	if snap.Learner().Balance != -30 || snap.Learner().Reserved != 0 || !snap.Progress.Done() {
		t.Fatalf("%+v %+v", snap.Learner(), snap.Progress)
	}
	home := c.get("/").Body.String()
	if !strings.Contains(home, "You finished the practice steps") || strings.Count(home, ">Done<") != 5 {
		t.Error("completion not shown")
	}
	checkA11y(t, c.get("/step/trade").Body.String())
	// exports are labelled
	jw := c.get("/export.json")
	var ex exportDoc
	if err := json.Unmarshal(jw.Body.Bytes(), &ex); err != nil || !ex.Practice || ex.Notice != PracticeMarker || len(ex.Members) != 4 {
		t.Fatal(err, jw.Body.String())
	}
	for _, m := range ex.Members {
		if !m.Practice || !strings.HasPrefix(m.ID, "practice-") {
			t.Errorf("unlabelled %+v", m)
		}
	}
	if !strings.Contains(jw.Header().Get("Content-Disposition"), "PRACTICE") {
		t.Error("export filename")
	}
	cw := c.get("/export.csv")
	for i, line := range strings.Split(strings.TrimSpace(cw.Body.String()), "\n") {
		if i > 0 && !strings.HasPrefix(line, "PRACTICE,") {
			t.Errorf("csv line %d unlabelled: %s", i, line)
		}
	}
	// reset puts everything back; exit deletes the world and cookie
	if w := c.postStep("/", "/reset", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	if snap := s.sessions[c.cookie.Value].world.Snapshot(); snap.Progress.Completed != 0 || snap.Learner().Balance != 0 {
		t.Fatal("reset")
	}
	if w := c.postStep("/", "/exit", nil); w.Code != 303 || w.Header().Get("Location") != "/" || c.cookie != nil || s.SessionCount() != 0 {
		t.Fatalf("exit: %d %v %d", w.Code, w.Header(), s.SessionCount())
	}
}

func TestPracticeSessionIsolationAndCredentialSeparation(t *testing.T) {
	s := newPracticeServer(t, nil)
	a, b := newClient(t, s), newClient(t, s)
	a.start()
	b.start()
	if a.cookie.Value == b.cookie.Value {
		t.Fatal("shared session token")
	}
	a.postStep("/step/terms", "/step/terms", nil)
	if s.sessions[b.cookie.Value].world.Snapshot().Progress.Completed != 0 {
		t.Fatal("progress leaked between practice sessions")
	}
	// A production session cookie is never a practice credential, never read, never echoed.
	c := newClient(t, s)
	r := httptest.NewRequest("GET", "http://practice.circula.test/step/terms", nil)
	r.AddCookie(&http.Cookie{Name: "circula_session", Value: strings.Repeat("A", 48)})
	r.AddCookie(&http.Cookie{Name: "__Host-circula_session", Value: a.cookie.Value})
	r.Header.Set("Authorization", "Bearer real-token")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/" || len(w.Result().Cookies()) != 0 {
		t.Fatalf("production credentials accepted by practice: %d %v", w.Code, w.Header())
	}
	// A practice token presented under a production cookie name or via header grants nothing.
	r = httptest.NewRequest("GET", "http://practice.circula.test/export.json", nil)
	r.Header.Set("Authorization", "Bearer "+a.cookie.Value)
	r.Header.Set("X-Session", a.cookie.Value)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatal("practice token accepted outside its cookie", w.Code)
	}
	// Forged / malformed / oversized cookies behave like no session.
	for _, v := range []string{"x", strings.Repeat("a", 50), strings.Repeat("a", 5000), "../../etc/passwd", a.cookie.Value + "x"} {
		c.cookie = &http.Cookie{Name: s.CookieName(), Value: v}
		if w := c.get("/export.json"); w.Code != 303 {
			t.Errorf("cookie %q: %d", v[:min(len(v), 10)], w.Code)
		}
	}
	// Wrong host (rebinding / production host) -> 421, nothing served.
	for _, h := range []string{"circula.test", "evil.example", "practice.circula.test.evil.example", ""} {
		c.host = h
		if w := c.get("/"); w.Code != 421 {
			t.Errorf("host %q: %d", h, w.Code)
		}
	}
}

func TestPracticePostGuards(t *testing.T) {
	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	c.start()
	doc := c.get("/step/terms").Body.String()
	good := c.hidden(doc)
	clone := func(mut func(url.Values)) url.Values {
		v := url.Values{}
		for k, vv := range good {
			v[k] = vv
		}
		mut(v)
		return v
	}
	if w := c.do("POST", "/step/terms", clone(func(v url.Values) { v.Del("csrf") })); w.Code != 403 {
		t.Errorf("no csrf: %d", w.Code)
	}
	if w := c.do("POST", "/step/terms", clone(func(v url.Values) { v.Set("csrf", "nope") })); w.Code != 403 {
		t.Errorf("bad csrf: %d", w.Code)
	}
	if w := c.do("POST", "/step/terms", clone(func(v url.Values) { v.Set("cmd", "zz") })); w.Code != 400 {
		t.Errorf("bad cmd: %d", w.Code)
	}
	c.hdr = map[string]string{"Origin": "https://evil.example"}
	if w := c.do("POST", "/step/terms", good); w.Code != 403 {
		t.Errorf("cross origin: %d", w.Code)
	}
	c.hdr = map[string]string{"Sec-Fetch-Site": "cross-site"}
	if w := c.do("POST", "/step/terms", good); w.Code != 403 {
		t.Errorf("cross site: %d", w.Code)
	}
	c.hdr = nil
	if w := c.do("POST", "/step/terms?csrf="+good.Get("csrf")+"&cmd="+good.Get("cmd"), url.Values{}); w.Code != 403 {
		t.Errorf("query-string csrf: %d", w.Code)
	}
	big := clone(func(v url.Values) { v.Set("pad", strings.Repeat("a", 20<<10)) })
	if w := c.do("POST", "/step/terms", big); w.Code != 413 {
		t.Errorf("oversize: %d", w.Code)
	}
	r := httptest.NewRequest("POST", "http://practice.circula.test/step/terms", strings.NewReader(`{"a":1}`))
	r.AddCookie(c.cookie)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Errorf("json: %d", w.Code)
	}
	// POST without any session (other than /start) is refused.
	n := newClient(t, s)
	if w := n.do("POST", "/reset", good); w.Code != 403 {
		t.Errorf("sessionless reset: %d", w.Code)
	}
	if w := n.do("POST", "/exit", good); w.Code != 403 {
		t.Errorf("sessionless exit: %d", w.Code)
	}
	if w := c.do("GET", "/start", nil); w.Code != 405 {
		t.Errorf("GET start: %d", w.Code)
	}
	if w := c.do("DELETE", "/", nil); w.Code != 405 {
		t.Errorf("DELETE: %d", w.Code)
	}
	for _, p := range []string{"/nope", "/step/other", "/step/../export.json", "/proposal"} {
		if w := c.get(p); w.Code != 404 && w.Code != 303 {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	// None of the rejected requests changed state.
	if s.sessions[c.cookie.Value].world.Snapshot().Progress.Completed != 0 {
		t.Fatal("rejected request changed state")
	}
	// Repeated start with an existing session keeps the same world.
	old := c.cookie.Value
	c.do("POST", "/start", url.Values{})
	if c.cookie.Value != old || s.SessionCount() != 1 {
		t.Fatal("start replaced the session")
	}
}

func TestPracticeEscapingAndCSVInjection(t *testing.T) {
	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	c.start()
	evil := `<script>alert(1)</script>`
	// HTML-significant characters are legal text and must be escaped everywhere.
	if w := c.postStep("/step/listing", "/step/listing", url.Values{"kind": {"offer"}, "title": {evil}, "amount": {"5"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	for _, p := range []string{"/step/listing", "/step/partner", "/step/trade", "/"} {
		d := c.get(p).Body.String()
		if strings.Contains(d, "<script>") {
			t.Errorf("%s: unescaped title", p)
		}
		checkA11y(t, d)
	}
	if w := c.postStep("/step/listing", "/step/listing", url.Values{"kind": {"offer"}, "title": {"=HYPERLINK(\"http://evil\")"}, "amount": {"5"}}); w.Code != 303 {
		t.Fatal(w.Code)
	}
	csv := c.get("/export.csv").Body.String()
	if !strings.Contains(csv, `'=HYPERLINK`) || strings.Contains(csv, ",=HYPERLINK") || strings.Contains(csv, `,"=HYPERLINK`) {
		t.Fatalf("csv formula injection:\n%s", csv)
	}
	// Bad inputs return 422 without creating anything.
	before := len(s.sessions[c.cookie.Value].world.Snapshot().Listings)
	for _, f := range []url.Values{
		{"kind": {"offer"}, "title": {"x"}, "amount": {"-1"}}, {"kind": {"offer"}, "title": {"x"}, "amount": {"10001"}},
		{"kind": {"gift"}, "title": {"x"}, "amount": {"1"}}, {"kind": {"offer"}, "title": {strings.Repeat("é", 61)}, "amount": {"1"}},
		{"kind": {"offer"}, "title": {"a\x00b"}, "amount": {"1"}}, {"kind": {"offer"}, "title": {"x"}, "amount": {"99999999999999999999"}},
	} {
		if w := c.postStep("/step/listing", "/step/listing", f); w.Code != 422 {
			t.Errorf("%v: %d", f, w.Code)
		}
	}
	if len(s.sessions[c.cookie.Value].world.Snapshot().Listings) != before {
		t.Fatal("invalid listing created")
	}
	// Tampered IDs.
	for _, f := range []url.Values{{"partner": {"real-member-1"}}, {"partner": {"practice-ana"}}, {"partner": {""}}, {"partner": {"practice-../x"}}} {
		w := c.postStep("/step/partner", "/step/partner", f)
		if w.Code == 303 {
			t.Errorf("accepted %v", f)
		}
	}
	for _, id := range []string{"1", "practice-listing-1\x00", "practice-listing-999", "practice-ana"} {
		d := c.get("/step/trade?listing=" + url.QueryEscape(id)).Body.String()
		checkA11y(t, d)
		if strings.Contains(d, "preview-heading") && id != "practice-listing-1" {
			t.Errorf("preview shown for %q", id)
		}
	}
}

func TestPracticeCapacityAndPendingGuards(t *testing.T) {
	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	c.start()
	world := s.sessions[c.cookie.Value].world
	world.mu.Lock()
	world.listings = append(world.listings, PracticeListing{PracticeID{"practice-listing-9"}, PracticeID{"practice-bo"}, KindOffer, "Too dear", 101})
	world.mu.Unlock()
	doc := c.get("/step/trade?listing=practice-listing-9").Body.String()
	if !strings.Contains(doc, "beyond the credit limit") || strings.Contains(doc, "Send proposal") {
		t.Fatalf("over-limit preview must not offer to send:\n%s", doc)
	}
	// forcing the POST anyway is refused by the engine
	f := c.hidden(c.get("/step/trade?listing=practice-listing-1").Body.String())
	f.Set("listing", "practice-listing-9")
	w := c.do("POST", "/step/trade/propose", f)
	if w.Code != 422 || !strings.Contains(w.Body.String(), PracticeMarker) {
		t.Fatalf("%d", w.Code)
	}
	checkA11y(t, w.Body.String())
	// Respond to a non-existent / foreign proposal id.
	f = c.hidden(c.get("/step/terms").Body.String())
	f.Set("proposal", "practice-proposal-77")
	f.Set("accept", "1")
	f.Set("cmd", nonce())
	if w := c.do("POST", "/proposal/respond", f); w.Code != 409 {
		t.Errorf("unknown proposal: %d", w.Code)
	}
	f.Set("accept", "maybe")
	if w := c.do("POST", "/proposal/respond", f); w.Code != 400 {
		t.Errorf("bad accept: %d", w.Code)
	}
}

func TestPracticeLimitsAndExpiry(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	s := newPracticeServer(t, func(c *PracticeConfig) {
		c.MaxSessions, c.SessionTTL, c.Now = 3, time.Hour, func() time.Time { return now }
	})
	var cs []*pclient
	for i := 0; i < 3; i++ {
		c := newClient(t, s)
		c.start()
		cs = append(cs, c)
	}
	over := newClient(t, s)
	if w := over.do("POST", "/start", url.Values{}); w.Code != 503 || !strings.Contains(w.Body.String(), PracticeMarker) {
		t.Fatalf("session cap: %d", w.Code)
	}
	now = now.Add(2 * time.Hour) // all expired
	if w := cs[0].get("/export.json"); w.Code != 303 {
		t.Fatalf("expired session still valid: %d", w.Code)
	}
	over.start() // sweep frees capacity
	if s.SessionCount() != 1 {
		t.Fatalf("sessions=%d", s.SessionCount())
	}
}

func TestPracticeConcurrentSessions(t *testing.T) {
	s := newPracticeServer(t, nil)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := newClient(t, s)
			c.start()
			c.postStep("/step/terms", "/step/terms", nil)
			c.postStep("/step/credit", "/step/credit", nil)
			doc := c.get("/step/trade?listing=practice-listing-1").Body.String()
			c.do("POST", "/step/trade/propose", c.hidden(doc))
			c.get("/export.json")
			c.get("/")
		}()
	}
	wg.Wait()
	if s.SessionCount() != 12 {
		t.Fatal(s.SessionCount())
	}
}

func TestPracticeTranslated(t *testing.T) {
	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	c.hdr = map[string]string{"Accept-Language": "fr-CA,fr;q=0.9,en;q=0.8"}
	c.start()
	for _, p := range []string{"/", "/step/terms", "/step/credit", "/step/listing", "/step/partner", "/step/trade?listing=practice-listing-1"} {
		d := c.get(p).Body.String()
		checkA11y(t, d)
		if !strings.Contains(d, `<html lang="fr">`) || !strings.Contains(d, "Entraînement — crédits fictifs, aucun vrai échange") || !strings.Contains(d, ">Avertissement<") {
			t.Errorf("%s: not French / banner missing", p)
		}
	}
	// Pages never leak untranslated message keys.
	for _, p := range []string{"/", "/step/listing"} {
		if d := c.get(p).Body.String(); strings.Contains(d, "pr_") || strings.Contains(d, "&lt;no value&gt;") || strings.Contains(d, "%!") {
			t.Errorf("%s: missing translation placeholder", p)
		}
	}
}

func TestPracticeCreditExampleMatchesFixtures(t *testing.T) {
	cy, _ := NewWorld().Snapshot().MemberByID(PracticeID{"practice-cy"})
	for loc, cat := range catalogs {
		want := map[string]string{"en": "-40", "fr": "-40"}[loc]
		if !strings.Contains(cat["pr_credit_example"], want) || cy.Balance != -40 || cy.Limit != 100 || cy.Available() != 60 || !strings.Contains(cat["pr_credit_example"], "60") {
			t.Errorf("%s: credit example out of sync with fixture", loc)
		}
	}
}
