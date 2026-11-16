package contract

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"github.com/djshellshoxxx/circula/internal/web"
)

const equivalencePassword = "correct horse battery"

// Both adapters share the same deterministic service setup, clock and actors.
// Only two databases are needed for the complete sequential MVP workflow.
type adapterFixture struct {
	t        *testing.T
	env      *testutil.Env
	h        http.Handler
	isAPI    bool
	net      *domain.Network
	actors   map[string]domain.Actor
	sessions map[string]string
	ids      map[string]string
	seen     map[string]bool
	seenMu   sync.Mutex
}

func newAdapterFixture(t *testing.T, isAPI bool) *adapterFixture {
	t.Helper()
	e := testutil.NewEnv(t)
	f := &adapterFixture{t: t, env: e, isAPI: isAPI, actors: map[string]domain.Actor{}, sessions: map[string]string{}, ids: map[string]string{}, seen: map[string]bool{}}
	f.actors["admin"] = e.Person("admin")
	f.net, _ = e.Network(f.actors["admin"], "equivalence", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		f.actors[name], _ = e.Member(f.actors["admin"], f.net, name)
	}
	for _, name := range []string{"invitee", "pending", "denied"} {
		f.actors[name] = e.Person(name)
	}
	for _, name := range []string{"pending", "denied"} {
		testutil.Must(e.Svc.JoinNetwork(e.Ctx, f.actors[name], f.net.ID, name))
	}
	for _, name := range []string{"admin", "alice", "bob", "carol", "dave", "invitee", "pending", "denied"} {
		tok, _, err := e.Svc.Login(e.Ctx, name, equivalencePassword)
		if err != nil {
			t.Fatal(err)
		}
		f.sessions[name] = tok
	}
	for _, row := range f.rows(`SELECT p.login,m.id FROM members m JOIN persons p ON p.id=m.person_id ORDER BY p.login`) {
		f.ids[row[0].(string)] = row[1].(string)
	}
	if isAPI {
		f.h = api.NewHandlerWithOptions(e.Svc, api.Options{CSRFKey: csrfKey, Now: e.Clock.Now, IPPerMinute: -1, PrincipalPerMinute: -1})
		t.Cleanup(func() { api.CloseHandler(f.h) })
	} else {
		f.h = web.NewHandlerWith(e.Svc, web.Options{CSRFKey: csrfKey, Clock: e.Clock})
	}
	return f
}

func (f *adapterFixture) expand(s string) string {
	for name, id := range f.ids {
		s = strings.ReplaceAll(s, "{"+name+"}", id)
	}
	return strings.ReplaceAll(s, "{base}", "/n/equivalence")
}

func (f *adapterFixture) request(who, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	var raw string
	ctype := "application/json"
	if f.isAPI && body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			f.t.Fatal(err)
		}
		raw = f.expand(string(b))
		if method == "PATCH" {
			ctype = "application/merge-patch+json"
		}
	} else if body != nil {
		v := url.Values{}
		for k, val := range body.(map[string]string) {
			v.Set(k, f.expand(val))
		}
		raw, ctype = v.Encode(), "application/x-www-form-urlencoded"
	}
	r := httptest.NewRequest(method, f.expand(path), strings.NewReader(raw))
	if body != nil {
		r.Header.Set("Content-Type", ctype)
	}
	r.AddCookie(&http.Cookie{Name: httpx.SessionCookie, Value: f.sessions[who]})
	r.Header.Set("X-CSRF-Token", (httpx.CSRF{Key: csrfKey}).Token(f.sessions[who]))
	r.Header.Set("Origin", "http://example.com")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	f.seenMu.Lock()
	path = strings.ReplaceAll(path, "{base}", "/n/{slug}")
	f.seen[canonicalOperation(method+" "+strings.TrimPrefix(path, "/api/v1"))] = true
	f.seenMu.Unlock()
	return w
}

func (f *adapterFixture) requireStatus(w *httptest.ResponseRecorder, want int) {
	f.t.Helper()
	if w.Code != want {
		f.t.Fatalf("api=%t status=%d want=%d body=%s", f.isAPI, w.Code, want, trunc(w.Body.String()))
	}
}

type adapterStep struct {
	key, who, apiMethod, apiPath, webMethod, webPath string
	json                                             map[string]any
	form                                             map[string]string
	apiStatus, webStatus                             int
	review                                           bool
	// A balance shown by the adjustment confirmation is an audited read.
	// The equivalent API operation is explicitly GET account then POST adjust.
	apiAccountReads []string
}

var hiddenInput = regexp.MustCompile(`<input\b[^>]*type="hidden"[^>]*>`)
var inputAttribute = regexp.MustCompile(`([a-z_]+)="([^"]*)"`)

func (f *adapterFixture) prepare(s adapterStep) map[string]string {
	vals := map[string]string{}
	for k, v := range s.form {
		vals[k] = v
	}
	vals["idem"] = "equivalence-" + s.key
	if f.isAPI && s.review {
		// Match the HTML review's disclosed account reads before the command
		// baseline as well as the final confirmation's reads below.
		for _, member := range s.apiAccountReads {
			f.requireStatus(f.request(s.who, "GET", "/api/v1/admin/accounts/"+member, nil, nil), 200)
		}
	}
	if !f.isAPI && s.review {
		vals["step"] = "review"
		before := f.ledger()
		w := f.request(s.who, "POST", s.webPath, vals, nil)
		f.requireStatus(w, 200)
		if !reflect.DeepEqual(before, f.ledger()) {
			f.t.Fatal("HTML review changed domain state before confirmation")
		}
		for _, tag := range hiddenInput.FindAllString(w.Body.String(), -1) {
			attrs := map[string]string{}
			for _, a := range inputAttribute.FindAllStringSubmatch(tag, -1) {
				attrs[a[1]] = html.UnescapeString(a[2])
			}
			if attrs["name"] != "" {
				vals[attrs["name"]] = attrs["value"]
			}
		}
		vals["step"] = "confirm"
	}
	return vals
}

func (f *adapterFixture) execute(s adapterStep, vals map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	if f.isAPI {
		for _, member := range s.apiAccountReads {
			f.requireStatus(f.request(s.who, "GET", "/api/v1/admin/accounts/"+member, nil, nil), 200)
		}
		w := f.request(s.who, s.apiMethod, "/api/v1"+s.apiPath, s.json, map[string]string{"Idempotency-Key": "equivalence-" + s.key, "If-Match": "*"})
		want := s.apiStatus
		if want == 0 {
			want = 200
		}
		f.requireStatus(w, want)
		return w
	}
	body := any(vals)
	if s.webMethod == "GET" {
		body = nil
	}
	w := f.request(s.who, s.webMethod, s.webPath, body, nil)
	want := s.webStatus
	if want == 0 {
		if s.webMethod == "GET" {
			want = 200
		} else {
			want = 303
		}
	}
	f.requireStatus(w, want)
	return w
}

func (f *adapterFixture) rows(q string, args ...any) [][]any {
	f.t.Helper()
	r, err := f.env.Store.DB().Query(q, args...)
	if err != nil {
		f.t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	cols, err := r.Columns()
	if err != nil {
		f.t.Fatal(err)
	}
	out := [][]any{}
	for r.Next() {
		v := make([]any, len(cols))
		dest := make([]any, len(cols))
		for i := range v {
			dest[i] = &v[i]
		}
		if err := r.Scan(dest...); err != nil {
			f.t.Fatal(err)
		}
		for i, x := range v {
			if b, ok := x.([]byte); ok {
				v[i] = string(b)
			}
		}
		out = append(out, v)
	}
	if err := r.Err(); err != nil {
		f.t.Fatal(err)
	}
	return out
}

// Opaque entity IDs are normalized by table and creation order. Amounts,
// ledger sequence, status, actor, timestamps, reasons and complete before/after
// audit summaries remain asserted. Request IDs and idempotency references are
// transport metadata; audit hashes necessarily bind those different values.
func (f *adapterFixture) normalize(v any) string {
	replacements := []string{}
	for _, table := range []string{"networks", "persons", "members", "accounts", "transactions", "listings", "disputes", "invitations", "announcements", "consistency_runs", "api_tokens"} {
		for i, row := range f.rows("SELECT id FROM " + table + " ORDER BY id") {
			replacements = append(replacements, row[0].(string), fmt.Sprintf("%s#%d", table, i+1))
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		f.t.Fatal(err)
	}
	return strings.NewReplacer(replacements...).Replace(string(b))
}

func (f *adapterFixture) ledger() map[string][][]any {
	out := map[string][][]any{}
	queries := map[string]string{
		"networks":     `SELECT id,status,ledger_seq FROM networks ORDER BY id`,
		"accounts":     `SELECT id,member_id,is_system,balance,limit_override,limit_override_until FROM accounts ORDER BY id`,
		"transactions": `SELECT id,kind,status,payer_account_id,payee_account_id,amount,listing_id,note,reason,original_tx_id,created_by,accepted_by,reversed_total,proposed_at,expires_at,completed_at,ledger_seq,related_tx_id FROM transactions ORDER BY id`,
		"entries":      `SELECT tx_id,account_id,amount,ledger_seq,posted_at FROM entries ORDER BY ledger_seq,account_id`,
		"status":       `SELECT tx_id,from_status,to_status,actor_id,occurred_at FROM transaction_status_events ORDER BY tx_id,id`,
		"members":      `SELECT id,person_id,display_name,status,default_flag FROM members ORDER BY id`,
		"listings":     `SELECT id,member_id,type,title,description,price,status FROM listings ORDER BY id`,
		"disputes":     `SELECT id,tx_id,opened_by,category,statement,status,outcome,reversal_amount,reviewer_id,rationale FROM disputes ORDER BY id`,
		"policies":     `SELECT network_id,version,policy_json,reason,effective_at FROM network_policies ORDER BY network_id,version`,
	}
	for k, q := range queries {
		out[k] = f.rows(q)
	}
	return out
}

func (f *adapterFixture) auditMark() int64 {
	return f.rows("SELECT COALESCE(MAX(rowid),0) FROM audit_events")[0][0].(int64)
}
func (f *adapterFixture) auditAfter(mark int64) [][]any {
	return f.rows(`SELECT actor_type,actor_id,action,target_type,target_id,summary_before,summary_after,reason,occurred_at,
		(SELECT json_group_array(member_id) FROM (SELECT member_id FROM audit_subjects WHERE audit_id=a.id ORDER BY member_id))
		FROM audit_events a WHERE rowid>? ORDER BY rowid`, mark)
}

type adapterPair struct {
	t        *testing.T
	api, web *adapterFixture
}

func newAdapterPair(t *testing.T) *adapterPair {
	return &adapterPair{t, newAdapterFixture(t, true), newAdapterFixture(t, false)}
}

func (p *adapterPair) step(s adapterStep) [2]*httptest.ResponseRecorder {
	p.t.Helper()
	var result [2]*httptest.ResponseRecorder
	p.t.Run(s.key, func(t *testing.T) {
		p.api.t, p.web.t = t, t
		ap, wp := p.api.auditMark(), p.web.auditMark()
		av, wv := p.api.prepare(s), p.web.prepare(s)
		if a, w := p.api.normalize(p.api.auditAfter(ap)), p.web.normalize(p.web.auditAfter(wp)); a != w {
			t.Errorf("review audit outcomes differ\nAPI %s\nHTML %s", a, w)
		}
		am, wm := p.api.auditMark(), p.web.auditMark()
		result[0], result[1] = p.api.execute(s, av), p.web.execute(s, wv)
		if a, w := p.api.normalize(p.api.ledger()), p.web.normalize(p.web.ledger()); a != w {
			t.Errorf("domain outcomes differ\nAPI %s\nHTML %s", a, w)
		}
		if a, w := p.api.normalize(p.api.auditAfter(am)), p.web.normalize(p.web.auditAfter(wm)); a != w {
			t.Errorf("audit outcomes differ\nAPI %s\nHTML %s", a, w)
		}
	})
	p.api.t, p.web.t = p.t, p.t
	return result
}

func (p *adapterPair) remember(alias, table, where string, args ...any) {
	p.t.Helper()
	for _, f := range []*adapterFixture{p.api, p.web} {
		rows := f.rows("SELECT id FROM "+table+" WHERE "+where, args...)
		if len(rows) != 1 {
			p.t.Fatalf("lookup %s returned %d rows", alias, len(rows))
		}
		f.ids[alias] = rows[0][0].(string)
	}
}

// Keep map rendering deterministic in failure reports and inventory checks.
func sortedKeys(m map[string]bool) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
