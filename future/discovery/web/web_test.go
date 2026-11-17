package web_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/imgsan"
	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore/sqlitetest"
	"github.com/djshellshoxxx/circula/future/discovery/web"
)

var ctx = context.Background()

var actors = map[string]d.Actor{
	"alice": {ID: "alice", Network: "n1", Role: d.RoleMember, Active: true, CSRF: "csrf-alice"},
	"bob":   {ID: "bob", Network: "n1", Role: d.RoleMember, Active: true, CSRF: "csrf-bob"},
	"adm":   {ID: "adm", Network: "n1", Role: d.RoleAdmin, Active: true, CSRF: "csrf-adm"},
	"carol": {ID: "carol", Network: "n2", Role: d.RoleMember, Active: true, CSRF: "csrf-carol"},
	"ghost": {ID: "ghost", Network: "n1", Role: d.RoleMember, Active: false, CSRF: "csrf-ghost"},
}

type fixture struct {
	t     *testing.T
	db    *sql.DB
	svc   *d.Service
	h     http.Handler
	clock *testkit.Clock
	dir   string
}

func newFixture(t *testing.T, base string) *fixture {
	t.Helper()
	if !sqlitetest.Available() {
		t.Skip("run with -tags=sqlite_cgo_test")
	}
	f := &fixture{t: t, clock: testkit.NewClock(), dir: t.TempDir()}
	db, err := sqlitetest.OpenFile(filepath.Join(f.dir, "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	f.db = db
	blobs, _ := d.NewFSBlobs(filepath.Join(f.dir, "blobs"))
	f.svc, err = d.NewService(sqlstore.New(db), blobs, d.Config{CursorKeys: [][]byte{bytes.Repeat([]byte{7}, 32)}, Now: f.clock.Now, UploadsPerHour: 1000},
		imgsan.Limits{FullMaxEdge: 500, ThumbMaxEdge: 100, MaxInputBytes: 200 << 10})
	if err != nil {
		t.Fatal(err)
	}
	f.h, err = web.New(web.Config{Service: f.svc, BasePath: base, MaxMediaBody: 300 << 10, Logf: t.Logf,
		Auth: web.AuthFunc(func(r *http.Request) (d.Actor, error) {
			a, ok := actors[r.Header.Get("X-Test-User")]
			if !ok {
				return d.Actor{}, fmt.Errorf("no session")
			}
			return a, nil
		})})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) listing(network, id, owner, title string) {
	f.t.Helper()
	now := f.clock.Now()
	if err := f.svc.SyncListing(ctx, d.ListingRecord{ID: id, Network: network, OwnerID: owner, Kind: d.KindOffer, Title: title, Description: "desc of " + id,
		Status: d.StatusActive, OwnerActive: true, Created: now.Add(-time.Hour), Updated: now}); err != nil {
		f.t.Fatal(err)
	}
}

type req struct {
	method, path string
	user         string
	body         io.Reader
	headers      map[string]string
}

func (f *fixture) do(r req) *httptest.ResponseRecorder {
	f.t.Helper()
	hr := httptest.NewRequest(r.method, r.path, r.body)
	if r.user != "" {
		hr.Header.Set("X-Test-User", r.user)
	}
	for k, v := range r.headers {
		hr.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, hr)
	return w
}

func (f *fixture) get(user, path string) *httptest.ResponseRecorder {
	return f.do(req{method: "GET", path: path, user: user})
}

func (f *fixture) jsonReq(method, path, user, csrf string, body any, extra map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	b, _ := json.Marshal(body)
	h := map[string]string{"Content-Type": "application/json"}
	if csrf != "" {
		h["X-CSRF-Token"] = csrf
	}
	for k, v := range extra {
		h[k] = v
	}
	return f.do(req{method: method, path: path, user: user, body: bytes.NewReader(b), headers: h})
}

func (f *fixture) form(path, user string, vals url.Values) *httptest.ResponseRecorder {
	f.t.Helper()
	if vals.Get("csrf") == "" && actors[user].CSRF != "" {
		vals.Set("csrf", actors[user].CSRF)
	}
	return f.do(req{method: "POST", path: path, user: user, body: strings.NewReader(vals.Encode()), headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}})
}

func multipartBody(t *testing.T, csrf, alt, filename string, data []byte) (io.Reader, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	if csrf != "" {
		mw.WriteField("csrf", csrf)
	}
	mw.WriteField("alt", alt)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "image/svg+xml") // lies must be ignored
	pw, _ := mw.CreatePart(h)
	pw.Write(data)
	mw.Close()
	return &b, mw.FormDataContentType()
}

func (f *fixture) upload(path, user string, csrfInForm bool, alt string, data []byte, apiHeaders bool) *httptest.ResponseRecorder {
	f.t.Helper()
	csrf := ""
	if csrfInForm {
		csrf = actors[user].CSRF
	}
	body, ct := multipartBody(f.t, csrf, alt, "../../evil.svg", data)
	h := map[string]string{"Content-Type": ct}
	if apiHeaders {
		h["X-CSRF-Token"] = actors[user].CSRF
	}
	return f.do(req{method: "POST", path: path, user: user, body: body, headers: h})
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("bad json %q: %v", w.Body.String(), err)
	}
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status %d want %d; body: %.300s", w.Code, code, w.Body.String())
	}
}

type prob struct {
	Type, Title, Detail, Field string
	Status                     int
}

func wantProblem(t *testing.T, w *httptest.ResponseRecorder, code int) prob {
	t.Helper()
	wantStatus(t, w, code)
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content type %q", ct)
	}
	var p prob
	decode(t, w, &p)
	if p.Status != code || p.Type == "" || p.Title == "" {
		t.Fatalf("incomplete problem %+v", p)
	}
	return p
}

func (f *fixture) classes(t *testing.T) (goods d.Category) {
	var err error
	if goods, err = f.svc.CreateCategory(ctx, actors["adm"], "Tools", d.TypeGoods); err != nil {
		t.Fatal(err)
	}
	return goods
}
