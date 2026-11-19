package main

// HTTP acceptance scenario: the lifecycle of TestAcceptanceScenario driven
// through the handler stack that `circula serve` builds (newHandlers, wrapped
// by opsServer.routes), not through the service layer. Each person has one
// cookie jar and uses the web forms where a form exists and the JSON API with
// that same browser session where none does. The steps share state, so they
// run as ordered subtests; the restart step rebuilds the stack on a reopened
// database.

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

const (
	httpSlug        = "acceptance-net"
	httpSetupSecret = "acceptance-setup-secret-0123456789"
)

var (
	httpCSRFRe   = regexp.MustCompile(`name="csrf" value="([^"]+)"`)
	httpIdemRe   = regexp.MustCompile(`name="idem" value="([^"]+)"`)
	httpJoinRe   = regexp.MustCompile(`/join/([A-Za-z0-9_-]+)`)
	httpTradeRe  = regexp.MustCompile(`/trades/([A-Za-z0-9-]+)$`)
	httpErrorSum = `id="error-summary"`
)

// httpRun is the state shared by the ordered steps.
type httpRun struct {
	t       *testing.T
	ctx     context.Context
	cfg     *Config
	logger  *slog.Logger
	metrics *metrics
	st      *store.SQLite
	svc     *service.Service
	h       http.Handler // the serve stack; replaced on restart
	base    string

	founder, alice, bob, carol *webUser
	memberID                   map[string]string // display name -> member id
	tradeID, reversalID        string
	acceptPage                 *httpPage // Bob's accept review: its idem key is replayed in step 12
}

// webUser is one person's browser: a cookie jar and the CSRF token of the last
// HTML page they were shown (the API takes the same token in X-CSRF-Token).
type webUser struct {
	login  string
	client *http.Client
	csrf   string
}

// httpPage is a response: the final path after redirects, and the body.
type httpPage struct {
	status int
	path   string
	body   string
	header http.Header
}

func (p *httpPage) csrf() string {
	if m := httpCSRFRe.FindStringSubmatch(p.body); m != nil {
		return m[1]
	}
	return ""
}

func (p *httpPage) idem() string {
	if m := httpIdemRe.FindStringSubmatch(p.body); m != nil {
		return m[1]
	}
	return ""
}

// API views, decoded from the JSON the handlers return.
type httpTx struct {
	ID                 string  `json:"id"`
	Kind               string  `json:"kind"`
	Status             string  `json:"status"`
	PayerMemberID      *string `json:"payer_member_id"`
	PayeeMemberID      *string `json:"payee_member_id"`
	AmountMinor        string  `json:"amount_minor"`
	OriginalTxID       *string `json:"original_transaction_id"`
	ReversedTotalMinor string  `json:"reversed_total_minor"`
	PostedLedgerSeq    *int64  `json:"posted_ledger_seq"`
}

type httpAccount struct {
	BalanceMinor     string `json:"balance_minor"`
	CreditLimitMinor string `json:"credit_limit_minor"`
	ReservedMinor    string `json:"reserved_minor"`
	AvailableMinor   string `json:"available_minor"`
	Direction        string `json:"direction"`
}

type httpEntry struct {
	AmountMinor   string `json:"amount_minor"`
	TransactionID string `json:"transaction_id"`
	LedgerSeq     int64  `json:"ledger_seq"`
}

type httpMember struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type httpProblem struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
}

// resolveHTTPConfig applies the settings resolution `circula serve` uses, with
// the given environment, so the test runs with the operator's defaults.
func resolveHTTPConfig(t *testing.T, env map[string]string) *Config {
	t.Helper()
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	sf := bindSettingFlags(newFlagSet("serve", io.Discard))
	cfg, _, err := sf.Resolve(lookup, true)
	if err != nil {
		t.Fatalf("resolve settings: %v", err)
	}
	return cfg
}

func httpCLI(t *testing.T, args ...string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != exitOK {
		t.Fatalf("circula %s: exit %d: %s", args[0], code, errOut.String())
	}
	return out.String()
}

func (r *httpRun) must(err error, what string) {
	r.t.Helper()
	if err != nil {
		r.t.Fatalf("%s: %v", what, err)
	}
}

// need stops a step whose earlier step did not leave the state it depends on.
func (r *httpRun) need(ok bool, what string) {
	r.t.Helper()
	if !ok {
		r.t.Fatalf("prerequisite missing: %s", what)
	}
}

func clip(s string) string {
	if len(s) > 400 {
		return s[:400] + "..."
	}
	return s
}

func (r *httpRun) want(p *httpPage, status int, what string) {
	r.t.Helper()
	if p.status != status {
		r.t.Fatalf("%s: status %d, want %d: %s", what, p.status, status, clip(p.body))
	}
}

func (r *httpRun) contains(p *httpPage, subs ...string) {
	r.t.Helper()
	body := html.UnescapeString(p.body)
	for _, s := range subs {
		if !strings.Contains(body, s) && !strings.Contains(p.body, s) {
			r.t.Errorf("page %s (status %d) lacks %q: %s", p.path, p.status, s, clip(p.body))
		}
	}
}

// noError asserts a form was accepted: a success status and no error summary.
func (r *httpRun) noError(p *httpPage, what string) {
	r.t.Helper()
	if p.status >= 400 || strings.Contains(p.body, httpErrorSum) {
		r.t.Fatalf("%s: status %d with an error: %s", what, p.status, clip(p.body))
	}
}

func (r *httpRun) id(name string) string {
	r.t.Helper()
	id := r.memberID[name]
	r.need(id != "", "member id for "+name)
	return id
}

func (r *httpRun) newUser(login string) *webUser {
	r.t.Helper()
	jar, err := cookiejar.New(nil)
	r.must(err, "cookie jar")
	return &webUser{login: login, client: &http.Client{Jar: jar}}
}

// send performs one request as u. It records the CSRF token of any HTML page.
func (r *httpRun) send(u *webUser, method, path string, hdr map[string]string, body io.Reader) *httpPage {
	r.t.Helper()
	req, err := http.NewRequestWithContext(r.ctx, method, r.base+path, body)
	r.must(err, method+" "+path)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := u.client.Do(req)
	r.must(err, method+" "+path)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	r.must(err, "read "+path)
	p := &httpPage{status: resp.StatusCode, path: resp.Request.URL.Path, body: string(raw), header: resp.Header}
	if tok := p.csrf(); tok != "" {
		u.csrf = tok
	}
	return p
}

func (r *httpRun) get(u *webUser, path string) *httpPage {
	r.t.Helper()
	return r.send(u, http.MethodGet, path, nil, nil)
}

// postForm submits a form as a browser does: the CSRF token and idempotency
// key come from the page the form was shown on (from), and vals are the fields.
func (r *httpRun) postForm(u *webUser, path string, from *httpPage, vals url.Values) *httpPage {
	r.t.Helper()
	if vals == nil {
		vals = url.Values{}
	}
	token := u.csrf
	if from != nil && from.csrf() != "" {
		token = from.csrf()
	}
	if vals.Get("csrf") == "" {
		vals.Set("csrf", token)
	}
	if vals.Get("idem") == "" && from != nil && from.idem() != "" {
		vals.Set("idem", from.idem())
	}
	hdr := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	return r.send(u, http.MethodPost, path, hdr, strings.NewReader(vals.Encode()))
}

// api calls the JSON API with u's browser session (cookie plus X-CSRF-Token
// on unsafe methods). idem, when set, is sent as Idempotency-Key.
func (r *httpRun) api(u *webUser, method, path string, body any, idem string) *httpPage {
	r.t.Helper()
	hdr := map[string]string{"Accept": "application/json"}
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		r.must(err, "encode body")
		rd = bytes.NewReader(raw)
		hdr["Content-Type"] = "application/json"
	}
	if method != http.MethodGet {
		hdr["X-CSRF-Token"] = u.csrf
	}
	if idem != "" {
		hdr["Idempotency-Key"] = idem
	}
	return r.send(u, method, path, hdr, rd)
}

func (r *httpRun) decode(p *httpPage, v any) {
	r.t.Helper()
	if err := json.Unmarshal([]byte(p.body), v); err != nil {
		r.t.Fatalf("decode status %d body %q: %v", p.status, clip(p.body), err)
	}
}

func apiList[T any](r *httpRun, p *httpPage) []T {
	r.t.Helper()
	r.want(p, http.StatusOK, "list")
	var env struct {
		Data []T `json:"data"`
	}
	r.decode(p, &env)
	return env.Data
}

func (r *httpRun) myAccount(u *webUser) httpAccount {
	r.t.Helper()
	p := r.api(u, http.MethodGet, "/api/v1/accounts/me", nil, "")
	r.want(p, http.StatusOK, "GET /api/v1/accounts/me as "+u.login)
	var a httpAccount
	r.decode(p, &a)
	return a
}

func (r *httpRun) adminAccount(u *webUser, memberID string) httpAccount {
	r.t.Helper()
	p := r.api(u, http.MethodGet, "/api/v1/admin/accounts/"+memberID, nil, "")
	r.want(p, http.StatusOK, "GET /api/v1/admin/accounts/{id}")
	var a httpAccount
	r.decode(p, &a)
	return a
}

func (r *httpRun) entries(u *webUser) []httpEntry {
	r.t.Helper()
	return apiList[httpEntry](r, r.api(u, http.MethodGet, "/api/v1/accounts/me/entries", nil, ""))
}

func (r *httpRun) txs(u *webUser) []httpTx {
	r.t.Helper()
	return apiList[httpTx](r, r.api(u, http.MethodGet, "/api/v1/transactions", nil, ""))
}

func (r *httpRun) tx(u *webUser, id string) httpTx {
	r.t.Helper()
	p := r.api(u, http.MethodGet, "/api/v1/transactions/"+id, nil, "")
	r.want(p, http.StatusOK, "GET /api/v1/transactions/{id} as "+u.login)
	var t httpTx
	r.decode(p, &t)
	return t
}

// entriesFor returns the entries of one transaction in a member's ledger.
func entriesFor(es []httpEntry, txID string) []httpEntry {
	var out []httpEntry
	for _, e := range es {
		if e.TransactionID == txID {
			out = append(out, e)
		}
	}
	return out
}

// rebuild builds the serve stack over the current service and store, the way
// runServe does: the same edge policy and CSRF key for the API and the UI.
func (r *httpRun) rebuild() {
	r.t.Helper()
	apiH, webH, err := newHandlers(r.svc, r.cfg, r.logger)
	r.must(err, "build handlers")
	svc := r.svc
	ops := &opsServer{cfg: r.cfg, db: r.st.DB(), logger: r.logger, metrics: r.metrics, maint: newMaintenance(r.cfg),
		frozen: func() int { return len(svc.FrozenNetworks()) }}
	ops.ready.Store(true)
	r.h = ops.routes(apiH, webH)
}

// tradeForm is the propose form for a payment from the signed-in member to bob.
func tradeForm(counterpartyID, amount, step string) url.Values {
	return url.Values{"counterparty_id": {counterpartyID}, "direction": {"pay"}, "amount": {amount},
		"note": {"acceptance: garden help"}, "step": {step}}
}

func TestAcceptanceScenarioHTTP(t *testing.T) {
	t.Setenv("CIRCULA_BACKUP_KEY", scenarioBackupKey)
	dir := t.TempDir()
	db := filepath.Join(dir, "circula.db")

	// The listener is created first so the public base URL can name its port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	cfg := resolveHTTPConfig(t, map[string]string{
		"CIRCULA_DATABASE":        db,
		"CIRCULA_PUBLIC_BASE_URL": base,
		"CIRCULA_COOKIE_SECURE":   "false", // plain http on loopback
		"CIRCULA_SETUP_SECRET":    httpSetupSecret,
		"CIRCULA_BACKUP_KEY":      scenarioBackupKey,
	})
	r := &httpRun{ctx: context.Background(), cfg: cfg, base: base, memberID: map[string]string{},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics: newMetrics(time.Now().UTC())}
	srv := &httptest.Server{Listener: ln, Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.h.ServeHTTP(w, req)
	})}}
	srv.Start()
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		if r.st != nil {
			_ = r.st.Close()
		}
	})

	// run gives each step its own *testing.T; the steps are sequential.
	run := func(name string, fn func(t *testing.T)) {
		t.Run(name, func(t *testing.T) {
			r.t = t
			fn(t)
		})
	}

	run("01 start with an initialized sqlite database", func(t *testing.T) {
		st, err := store.Open(r.ctx, db)
		r.must(err, "create database")
		r.must(st.Close(), "close bootstrap handle")
		out := httpCLI(t, "migrate", "-dry-run", "-database", db)
		if !strings.Contains(out, "schema is up to date") {
			t.Fatalf("schema not current after initialisation: %q", out)
		}
		r.st, r.svc, err = openDatabase(r.ctx, cfg)
		r.must(err, "start application")
		r.rebuild()
	})

	run("02 the founder registers, signs in and creates the network through the setup form", func(t *testing.T) {
		f := r.newUser("founder@example.org")
		r.founder = f
		reg := r.get(f, "/register")
		p := r.postForm(f, "/register", reg, url.Values{"login": {f.login}, "password": {scenarioPassword}, "password2": {scenarioPassword}})
		r.noError(p, "register founder")
		r.noError(r.postForm(f, "/login", r.get(f, "/login"), url.Values{"login": {f.login}, "password": {scenarioPassword}}), "sign in founder")
		setup := r.get(f, "/setup")
		r.contains(setup, "Create a network", "Credit unit name")
		created := r.postForm(f, "/setup", setup, url.Values{"name": {"Acceptance network"}, "slug": {httpSlug},
			"display_name": {"Founder"}, "currency_name": {"credit"}, "decimals": {"2"},
			"default_credit_limit": {domain.FormatAmount(scenarioDefault, 2)}, "signup_grant": {"0.00"},
			"join_mode": {"invite"}, "setup_secret": {httpSetupSecret}})
		r.contains(created, "Network created and open")
		r.want(r.get(f, "/n/"+httpSlug+"/"), http.StatusOK, "founder home after setup")
	})

	run("03 invite Alice, Bob and Carol and have each redeem the invitation", func(t *testing.T) {
		r.need(r.founder != nil, "network created")
		invites := "/n/" + httpSlug + "/admin/invitations"
		for _, name := range []string{"Alice", "Bob", "Carol"} {
			login := strings.ToLower(name) + "@example.org"
			page := r.get(r.founder, invites)
			made := r.postForm(r.founder, invites, page, url.Values{"days": {"7"}})
			m := httpJoinRe.FindStringSubmatch(made.body)
			if m == nil {
				t.Fatalf("no invitation link for %s: %s", name, clip(made.body))
			}
			tok := m[1]
			u := r.newUser(login)
			next := url.Values{"next": {"/join/" + tok}}
			reg := r.get(u, "/register?next=/join/"+tok)
			vals := url.Values{"login": {login}, "password": {scenarioPassword}, "password2": {scenarioPassword}}
			for k, v := range next {
				vals[k] = v
			}
			r.noError(r.postForm(u, "/register", reg, vals), "register "+name)
			// Registration never signs in: the person signs in, then lands on the invitation.
			lf := r.get(u, "/login?next=/join/"+tok)
			vals = url.Values{"login": {login}, "password": {scenarioPassword}}
			for k, v := range next {
				vals[k] = v
			}
			accept := r.postForm(u, "/login", lf, vals)
			r.contains(accept, "Accept your invitation")
			joined := r.postForm(u, "/join/"+tok, accept, url.Values{"display_name": {name}})
			r.contains(joined, "You have joined")
			r.contains(r.postForm(u, "/logout", joined, nil), "You are signed out")
			switch name {
			case "Alice":
				r.alice = u
			case "Bob":
				r.bob = u
			case "Carol":
				r.carol = u
			}
		}
		for _, m := range apiList[httpMember](r, r.api(r.founder, http.MethodGet, "/api/v1/members", nil, "")) {
			r.memberID[m.DisplayName] = m.ID
		}
		for _, name := range []string{"Alice", "Bob", "Carol"} {
			if r.memberID[name] == "" {
				t.Errorf("%s is not an active member after joining", name)
			}
		}
	})

	run("04 set credit limits", func(t *testing.T) {
		r.need(r.memberID["Alice"] != "", "members joined")
		members := r.get(r.founder, "/n/"+httpSlug+"/admin/members")
		limitPath := func(name string) string { return "/n/" + httpSlug + "/admin/members/" + r.id(name) + "/limit" }
		p := r.postForm(r.founder, limitPath("Alice"), members, url.Values{
			"amount": {domain.FormatAmount(scenarioAliceLimit, 2)}, "reason": {"acceptance scenario: Alice's limit"}})
		r.contains(p, "Set credit limit: done for Alice.")
		if got := r.adminAccount(r.founder, r.id("Alice")).CreditLimitMinor; got != "5000" {
			t.Errorf("Alice credit limit = %s, want 5000", got)
		}
		if got := r.adminAccount(r.founder, r.id("Bob")).CreditLimitMinor; got != "10000" {
			t.Errorf("Bob credit limit = %s, want the network default 10000", got)
		}
	})

	run("05 Alice signs in through the form", func(t *testing.T) {
		r.need(r.alice != nil, "Alice joined")
		p := r.postForm(r.alice, "/login", r.get(r.alice, "/login"), url.Values{"login": {r.alice.login}, "password": {scenarioPassword}})
		r.noError(p, "sign in Alice")
		r.contains(p, "Available to spend")
		if a := r.myAccount(r.alice); a.CreditLimitMinor != "5000" || a.BalanceMinor != "0" {
			t.Errorf("Alice account after sign-in = %+v", a)
		}
	})

	run("06 Alice proposes a trade to Bob through the web form", func(t *testing.T) {
		r.need(r.alice != nil && r.memberID["Bob"] != "", "Alice signed in")
		bob := r.id("Bob")
		path := "/n/" + httpSlug + "/trades/new"
		form := r.get(r.alice, path+"?to="+bob)
		r.contains(form, `value="`+bob+`" selected`)
		review := r.postForm(r.alice, path, form, tradeForm(bob, domain.FormatAmount(scenarioTrade, 2), "review"))
		r.contains(review, "Review your trade", "You pay Bob 30.00 credits")
		r.contains(r.postForm(r.alice, path, review, tradeForm(bob, domain.FormatAmount(scenarioTrade, 2), "confirm")),
			"Trade sent to")
		// A repeated submit of the same confirm (double click) creates no second trade.
		r.postForm(r.alice, path, review, tradeForm(bob, domain.FormatAmount(scenarioTrade, 2), "confirm"))
		list := r.txs(r.alice)
		if len(list) != 1 || list[0].Status != "proposed" || list[0].AmountMinor != "3000" {
			t.Fatalf("after proposal Alice lists %+v, want one proposed trade of 3000", list)
		}
		r.tradeID = list[0].ID
		if a := r.myAccount(r.alice); a.BalanceMinor != "0" || a.ReservedMinor != "3000" {
			t.Errorf("proposal moved Alice's balance or did not reserve: %+v", a)
		}
	})

	run("07 Bob signs in and accepts through the web form", func(t *testing.T) {
		r.need(r.tradeID != "", "trade proposed")
		p := r.postForm(r.bob, "/login", r.get(r.bob, "/login"), url.Values{"login": {r.bob.login}, "password": {scenarioPassword}})
		r.noError(p, "sign in Bob")
		acceptPath := "/n/" + httpSlug + "/trades/" + r.tradeID + "/accept"
		rev := r.get(r.bob, acceptPath)
		r.contains(rev, "Review before accepting", "+30.00", "Confirm and accept")
		done := r.postForm(r.bob, acceptPath, rev, nil)
		r.contains(done, "Trade completed.")
		r.acceptPage = rev
		if tx := r.tx(r.bob, r.tradeID); tx.Status != "completed" || tx.PostedLedgerSeq == nil {
			t.Errorf("trade after acceptance = %+v, want completed and posted", tx)
		}
	})

	run("08 the trade posts exactly once", func(t *testing.T) {
		r.need(r.tradeID != "", "trade accepted")
		ae := entriesFor(r.entries(r.alice), r.tradeID)
		be := entriesFor(r.entries(r.bob), r.tradeID)
		if len(ae) != 1 || len(be) != 1 {
			t.Fatalf("trade entries: Alice %d, Bob %d; want one posting in each ledger", len(ae), len(be))
		}
		if ae[0].AmountMinor != "-3000" || be[0].AmountMinor != "3000" {
			t.Errorf("posted amounts Alice %s Bob %s, want -3000 and 3000", ae[0].AmountMinor, be[0].AmountMinor)
		}
		if ae[0].LedgerSeq != be[0].LedgerSeq {
			t.Errorf("both sides must share one ledger sequence: Alice %d Bob %d", ae[0].LedgerSeq, be[0].LedgerSeq)
		}
		completed := 0
		for _, tx := range r.txs(r.bob) {
			if tx.ID == r.tradeID && tx.Status == "completed" {
				completed++
			}
		}
		if completed != 1 {
			t.Errorf("trade appears %d times as completed in Bob's list, want 1", completed)
		}
	})

	run("09 both balances move by equal and opposite amounts", func(t *testing.T) {
		r.need(r.bob != nil, "trade accepted")
		a, b := r.myAccount(r.alice), r.myAccount(r.bob)
		if a.BalanceMinor != "-3000" || a.Direction != "in_debt" || a.ReservedMinor != "0" || a.AvailableMinor != "2000" {
			t.Errorf("Alice after acceptance = %+v", a)
		}
		if b.BalanceMinor != "3000" || b.Direction != "in_credit" || b.AvailableMinor != "13000" {
			t.Errorf("Bob after acceptance = %+v", b)
		}
	})

	run("10 the member balances sum to zero", func(t *testing.T) {
		r.need(r.carol != nil, "members joined")
		sum := int64(0)
		for _, a := range []httpAccount{r.myAccount(r.founder), r.adminAccount(r.founder, r.id("Alice")),
			r.adminAccount(r.founder, r.id("Bob")), r.adminAccount(r.founder, r.id("Carol"))} {
			n, err := strconv.ParseInt(a.BalanceMinor, 10, 64)
			if err != nil {
				t.Fatalf("balance %q: %v", a.BalanceMinor, err)
			}
			sum += n
		}
		if sum != 0 {
			t.Errorf("sum of member balances = %d, want 0", sum)
		}
	})

	run("11 Carol gets 403 or 404 on Alice's trade", func(t *testing.T) {
		r.need(r.tradeID != "", "trade exists")
		r.postForm(r.carol, "/login", r.get(r.carol, "/login"), url.Values{"login": {r.carol.login}, "password": {scenarioPassword}})
		page := r.get(r.carol, "/n/"+httpSlug+"/trades/"+r.tradeID)
		if page.status != http.StatusForbidden && page.status != http.StatusNotFound {
			t.Errorf("Carol's trade page status %d, want 403 or 404", page.status)
		}
		if strings.Contains(page.body, "30.00") {
			t.Errorf("Carol's trade page leaks the amount")
		}
		api := r.api(r.carol, http.MethodGet, "/api/v1/transactions/"+r.tradeID, nil, "")
		if api.status != http.StatusForbidden && api.status != http.StatusNotFound {
			t.Errorf("Carol's API status %d, want 403 or 404", api.status)
		}
		for _, tx := range r.txs(r.carol) {
			if tx.ID == r.tradeID {
				t.Errorf("Carol's transaction list includes Alice and Bob's trade")
			}
		}
	})

	run("12 a repeated acceptance does not post again", func(t *testing.T) {
		r.need(r.acceptPage != nil, "Bob's accept form was shown")
		before := r.myAccount(r.bob).BalanceMinor
		acceptPath := "/n/" + httpSlug + "/trades/" + r.tradeID + "/accept"
		// A repeated form submit: same form, same idempotency key.
		again := r.postForm(r.bob, acceptPath, r.acceptPage, nil)
		r.contains(again, "Trade completed.")
		// The same key over the API returns the original completed trade. The key
		// was first used on the HTML form, so the API's own replay cache (API-022,
		// internal/api/idem.go) does not hold it and the header is not set; the
		// service's durable idempotency returns the original result instead, so
		// nothing posts again (checked below).
		key := r.acceptPage.idem()
		replay := r.api(r.bob, http.MethodPost, "/api/v1/transactions/"+r.tradeID+"/accept", nil, key)
		r.want(replay, http.StatusOK, "API accept with the same Idempotency-Key")
		var got httpTx
		r.decode(replay, &got)
		if got.ID != r.tradeID || got.Status != "completed" {
			t.Errorf("API replay returned %+v, want the completed trade %s", got, r.tradeID)
		}
		// A new key on a completed trade is refused as an invalid state.
		fresh := r.api(r.bob, http.MethodPost, "/api/v1/transactions/"+r.tradeID+"/accept", nil, "acceptance-repeat-new-key")
		var prob httpProblem
		r.decode(fresh, &prob)
		if fresh.status != http.StatusConflict || prob.Code != "invalid-transaction-state" {
			t.Errorf("accept with a new key = %d %q, want 409 invalid-transaction-state", fresh.status, prob.Code)
		}
		if n := len(entriesFor(r.entries(r.alice), r.tradeID)); n != 1 {
			t.Errorf("trade has %d postings in Alice's ledger after repeats, want 1", n)
		}
		if after := r.myAccount(r.bob).BalanceMinor; after != before {
			t.Errorf("Bob's balance changed from %s to %s on a repeated acceptance", before, after)
		}
	})

	run("13 a proposal above Alice's available credit is refused", func(t *testing.T) {
		r.need(r.tradeID != "", "trade exists")
		bob := r.id("Bob")
		path := "/n/" + httpSlug + "/trades/new"
		form := r.get(r.alice, path)
		review := r.postForm(r.alice, path, form, tradeForm(bob, "25.00", "review"))
		refused := r.postForm(r.alice, path, review, tradeForm(bob, "25.00", "confirm"))
		if refused.status != http.StatusUnprocessableEntity {
			t.Errorf("over-limit confirm status %d, want 422", refused.status)
		}
		r.contains(refused, "Not sent.", "Nothing changed")
		// The API gives the machine-readable code for the same refusal.
		api := r.api(r.alice, http.MethodPost, "/api/v1/transactions", map[string]any{"counterparty_member_id": bob,
			"direction": "pay", "amount_minor": "2500", "description": "over the limit"}, "alice-over-limit")
		var prob httpProblem
		r.decode(api, &prob)
		if api.status != http.StatusUnprocessableEntity || prob.Code != "insufficient-available-credit" {
			t.Errorf("API over-limit = %d %q, want 422 insufficient-available-credit", api.status, prob.Code)
		}
		if n := len(r.txs(r.alice)); n != 1 {
			t.Errorf("Alice lists %d transactions after refusals, want 1", n)
		}
		if a := r.myAccount(r.alice); a.BalanceMinor != "-3000" || a.ReservedMinor != "0" {
			t.Errorf("refused proposal changed Alice's account: %+v", a)
		}
	})

	run("14 an administrator reverses part of the posted trade", func(t *testing.T) {
		r.need(r.tradeID != "", "trade exists")
		reversePath := "/n/" + httpSlug + "/admin/reverse"
		origBefore := r.tx(r.founder, r.tradeID)
		entriesBefore := entriesFor(r.entries(r.alice), r.tradeID)
		form := r.get(r.founder, reversePath+"?tx="+r.tradeID)
		vals := func(step string) url.Values {
			return url.Values{"tx": {r.tradeID}, "amount": {domain.FormatAmount(scenarioPartial, 2)},
				"reason": {"acceptance scenario: partial reversal"}, "step": {step}}
		}
		review := r.postForm(r.founder, reversePath, form, vals("review"))
		r.noError(review, "reversal review")
		done := r.postForm(r.founder, reversePath, review, vals("confirm"))
		r.contains(done, "Reversal posted")
		m := httpTradeRe.FindStringSubmatch(done.path)
		if m == nil {
			t.Fatalf("reversal did not redirect to the new trade, landed on %s", done.path)
		}
		r.reversalID = m[1]
		rev := r.tx(r.founder, r.reversalID)
		if rev.Kind != string(domain.KindReversal) || rev.OriginalTxID == nil || *rev.OriginalTxID != r.tradeID {
			t.Errorf("reversal = %+v, want kind reversal referencing %s", rev, r.tradeID)
		}
		if rev.AmountMinor != "1000" {
			t.Errorf("reversal amount %s, want 1000", rev.AmountMinor)
		}
		orig := r.tx(r.founder, r.tradeID)
		if orig.AmountMinor != origBefore.AmountMinor || orig.Kind != origBefore.Kind || orig.Status != origBefore.Status ||
			*orig.PayerMemberID != *origBefore.PayerMemberID || *orig.PayeeMemberID != *origBefore.PayeeMemberID {
			t.Errorf("original trade changed: before %+v after %+v", origBefore, orig)
		}
		if orig.ReversedTotalMinor != "1000" {
			t.Errorf("original reversed total %s, want 1000", orig.ReversedTotalMinor)
		}
		entriesAfter := entriesFor(r.entries(r.alice), r.tradeID)
		if len(entriesAfter) != len(entriesBefore) || len(entriesAfter) != 1 || entriesAfter[0] != entriesBefore[0] {
			t.Errorf("original postings changed by the reversal: before %+v after %+v", entriesBefore, entriesAfter)
		}
		if n := len(entriesFor(r.entries(r.alice), r.reversalID)); n != 1 {
			t.Errorf("reversal has %d postings in Alice's ledger, want 1", n)
		}
		if a, b := r.myAccount(r.alice), r.myAccount(r.bob); a.BalanceMinor != "-2000" || b.BalanceMinor != "2000" {
			t.Errorf("balances after reversal: Alice %s Bob %s, want -2000 and 2000", a.BalanceMinor, b.BalanceMinor)
		}
	})

	run("15 a plain member cannot post an administrative adjustment", func(t *testing.T) {
		r.need(r.bob != nil && r.memberID["Alice"] != "", "members joined")
		adjust := "/n/" + httpSlug + "/admin/adjust"
		home := r.get(r.bob, "/n/"+httpSlug+"/")
		p := r.postForm(r.bob, adjust, home, url.Values{"member_id": {r.id("Alice")}, "direction": {"credit"},
			"amount": {"5.00"}, "reason": {"acceptance scenario: Bob is not an administrator"}, "step": {"confirm"}})
		if p.status != http.StatusForbidden {
			t.Errorf("member adjustment form status %d, want 403", p.status)
		}
		if g := r.get(r.bob, adjust); g.status != http.StatusForbidden {
			t.Errorf("member adjustment page status %d, want 403", g.status)
		}
		api := r.api(r.bob, http.MethodPost, "/api/v1/admin/adjustments", map[string]any{"member_id": r.id("Alice"),
			"direction": "credit", "amount_minor": "500", "reason": "acceptance scenario: Bob is not an administrator"}, "bob-adjust-1")
		var prob httpProblem
		r.decode(api, &prob)
		if api.status != http.StatusForbidden || prob.Code != "forbidden" {
			t.Errorf("API adjustment by a member = %d %q, want 403 forbidden", api.status, prob.Code)
		}
		if a, b := r.myAccount(r.alice), r.myAccount(r.bob); a.BalanceMinor != "-2000" || b.BalanceMinor != "2000" {
			t.Errorf("refused adjustment changed balances: Alice %s Bob %s", a.BalanceMinor, b.BalanceMinor)
		}
	})

	run("16 restart the application", func(t *testing.T) {
		r.need(r.st != nil, "application running")
		r.must(r.st.Close(), "close store")
		var err error
		r.st, r.svc, err = openDatabase(r.ctx, cfg)
		r.must(err, "reopen application")
		r.rebuild()
	})

	run("17 balances, history and sign-ins persist across the restart", func(t *testing.T) {
		r.need(r.reversalID != "", "reversal posted before restart")
		a, b := r.myAccount(r.alice), r.myAccount(r.bob)
		if a.BalanceMinor != "-2000" || b.BalanceMinor != "2000" {
			t.Errorf("balances after restart: Alice %s Bob %s", a.BalanceMinor, b.BalanceMinor)
		}
		if tx := r.tx(r.founder, r.tradeID); tx.ReversedTotalMinor != "1000" || tx.AmountMinor != "3000" {
			t.Errorf("original trade after restart = %+v", tx)
		}
		ids := map[string]bool{}
		for _, tx := range r.txs(r.alice) {
			ids[tx.ID] = true
		}
		if !ids[r.tradeID] || !ids[r.reversalID] {
			t.Errorf("Alice's history after restart lacks the trade or its reversal")
		}
		if n := len(entriesFor(r.entries(r.alice), r.tradeID)); n != 1 {
			t.Errorf("trade postings after restart = %d, want 1", n)
		}
		// The browser sessions survive: the cookies were issued before the restart.
		hist := r.get(r.alice, "/n/"+httpSlug+"/history")
		r.want(hist, http.StatusOK, "Alice history after restart")
	})

	run("18 verify the ledger with the CLI after the restart", func(t *testing.T) {
		// httpCLI fails the step on a non-zero exit (exitLedgerFailed included).
		out := httpCLI(t, "verify-ledger", "-database", db)
		if want := "network " + httpSlug + ": ok"; !strings.Contains(out, want) {
			t.Errorf("verify-ledger output lacks %q: %q", want, out)
		}
	})
}
