package areas

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Fixture policy mirrors the required transactional principal/listing boundary.
type fixturePolicy struct {
	mu                  sync.Mutex
	revoked, mfaMissing bool
	fail                bool
}

func (p *fixturePolicy) Check(_ context.Context, _ *sql.Tx, a Identity, action, listing string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.revoked || p.fail || a.Network != "n" || a.Member != "owner" && a.Member != "admin" {
		return errors.New("denied")
	}
	if action == "admin" && (a.Member != "admin" || p.mfaMissing) {
		return errors.New("admin MFA required")
	}
	if (action == "assign" || action == "cleanup") && !strings.HasPrefix(listing, "own-") {
		return errors.New("not listing owner")
	}
	if action == "listing" && strings.HasPrefix(listing, "hidden") {
		return errors.New("not visible")
	}
	return nil
}
func newService(t *testing.T, path string, p *fixturePolicy) Service {
	t.Helper()
	db, e := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	s := Service{db, p, func() time.Time { return time.Unix(100, 0) }}
	if e = s.Migrate(context.Background()); e != nil {
		t.Fatal(e)
	}
	return s
}

var admin = Identity{"n", "admin"}
var owner = Identity{"n", "owner"}

func createArea(t *testing.T, s Service, id, label string) Record {
	t.Helper()
	r, e := s.Edit(context.Background(), admin, "create-"+id, Edit{id, label, true, false, 0})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestPersistenceAssignmentsLifecyclePrivacy(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "areas.db")
	p := &fixturePolicy{}
	s := newService(t, path, p)
	a := createArea(t, s, "a", "North")
	createArea(t, s, "b", "South")
	for i, area := range []string{"a", "a", "b", ""} {
		_, e := s.Assign(ctx, owner, fmt.Sprintf("assign%d", i), Assign{fmt.Sprintf("own-%d", i), area, 0})
		if e != nil {
			t.Fatal(e)
		}
	}
	ids, e := s.Filter(ctx, owner, "a")
	if e != nil || strings.Join(ids, ",") != "own-0,own-1" {
		t.Fatal(ids, e)
	}
	sharing, e := s.ExportSharing(ctx, owner)
	if e != nil || len(sharing) != 0 {
		t.Fatal("default export", sharing, e)
	}
	if _, e = s.SetSharing(ctx, admin, "share", true, 0); e != nil {
		t.Fatal(e)
	}
	sharing, e = s.ExportSharing(ctx, owner)
	if e != nil || len(sharing) != 2 {
		t.Fatal(sharing, e)
	}
	// Same operation, same result; changing payload with the key conflicts.
	first, e := s.Assign(ctx, owner, "assign0", Assign{"own-0", "a", 0})
	if e != nil || first.Revision != 1 {
		t.Fatal(first, e)
	}
	if _, e = s.Assign(ctx, owner, "assign0", Assign{"own-0", "b", 0}); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if _, e = s.Edit(ctx, admin, "archive-a", Edit{"a", a.Label, true, true, a.Revision}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Filter(ctx, owner, "a"); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	if _, e = s.Edit(ctx, admin, "restore-a", Edit{"a", a.Label, true, false, a.Revision + 1}); e != nil {
		t.Fatal(e)
	}
	ids, e = s.Filter(ctx, owner, "a")
	if e != nil || len(ids) != 0 {
		t.Fatal("restoration resurrected selections", ids, e)
	}
	if e = s.DB.Close(); e != nil {
		t.Fatal(e)
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	data, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(backup, data, 0600); e != nil {
		t.Fatal(e)
	}
	s = newService(t, backup, p)
	catalog, e := s.Catalog(ctx, owner)
	if e != nil || len(catalog) != 2 {
		t.Fatal(catalog, e)
	}
	if _, e = s.Assign(ctx, owner, "again", Assign{"own-0", "a", 2}); e != nil {
		t.Fatal("restored revision", e)
	}
	if e = s.CleanupListing(ctx, owner, "cleanup", "own-0"); e != nil {
		t.Fatal(e)
	}
	ids, e = s.Filter(ctx, owner, "a")
	if e != nil || len(ids) != 0 {
		t.Fatal(ids, e)
	}
}
func TestFreshAuthIsolationConcurrencyAndRollback(t *testing.T) {
	ctx := context.Background()
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "A")
	for _, actor := range []Identity{owner, {"other", "admin"}, {"n", "stranger"}} {
		if _, e := s.Edit(ctx, actor, "forbidden", Edit{"x", "X", true, false, 0}); e == nil {
			t.Fatal(actor)
		}
	}
	p.mu.Lock()
	p.mfaMissing = true
	p.mu.Unlock()
	if _, e := s.Edit(ctx, admin, "no-mfa", Edit{"x", "X", true, false, 0}); e == nil {
		t.Fatal("MFA bypass")
	}
	p.mu.Lock()
	p.mfaMissing = false
	p.mu.Unlock()
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := s.Assign(ctx, owner, fmt.Sprintf("race%d", i), Assign{"own-race", "a", 0})
			if e == nil {
				mu.Lock()
				success++
				mu.Unlock()
			} else if !errors.Is(e, ErrConflict) {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	if success != 1 {
		t.Fatal(success)
	}
	if _, e := s.Assign(ctx, owner, "foreign-listing", Assign{"someone-else", "a", 0}); e == nil {
		t.Fatal("listing ownership")
	}
	if _, e := s.Assign(ctx, owner, "foreign-area", Assign{"own-x", "foreign", 0}); e == nil {
		t.Fatal("foreign area")
	}
	if _, e := s.DB.Exec(`CREATE TRIGGER reject_area_audit BEFORE INSERT ON future_area_audit BEGIN SELECT RAISE(ABORT,'injected failure'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Assign(ctx, owner, "rollback", Assign{"own-rollback", "a", 0}); e == nil {
		t.Fatal("expected audit failure")
	}
	var n int
	if e := s.DB.QueryRow(`SELECT count(*) FROM future_listing_areas WHERE listing='own-rollback'`).Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	p.mu.Lock()
	p.revoked = true
	p.mu.Unlock()
	if _, e := s.Assign(ctx, owner, "race0", Assign{"own-race", "a", 0}); e == nil {
		t.Fatal("retry bypassed revocation")
	}
	if _, e := s.Catalog(ctx, owner); e == nil {
		t.Fatal("catalog bypassed revocation")
	}
}
func TestHTTPFormsJSONCLIBoundaries(t *testing.T) {
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "<script>alert(1)</script>")
	auth := func(r *http.Request, mutation bool) (Identity, error) {
		if r.Header.Get("Authorization") != "fixture" {
			return Identity{}, errors.New("denied")
		}
		if mutation && r.Header.Get("X-CSRF") != "valid" {
			if e := r.ParseForm(); e != nil || r.PostForm.Get("csrf") != "valid" {
				return Identity{}, errors.New("csrf")
			}
		}
		if r.Header.Get("Admin") == "true" {
			return admin, nil
		}
		return owner, nil
	}
	web := Web{s, auth, func(*http.Request) string { return "valid" }, []byte(strings.Repeat("k", 32))}
	for _, c := range []struct {
		method, path, body, auth, csrf, content, admin string
		status                                         int
	}{
		{"GET", "/catalog", "", "", "", "", "", 403}, {"GET", "/catalog", "", "fixture", "", "", "", 200}, {"GET", "/admin", "", "fixture", "", "", "", 403}, {"GET", "/admin", "", "fixture", "", "", "true", 200},
		{"POST", "/assign", `{"Listing":"own-1","AreaID":"a","Revision":0}`, "fixture", "", "application/json", "", 403},
		{"POST", "/assign", `{"Listing":"own-1","AreaID":"a","Revision":0,"latitude":1.234}`, "fixture", "valid", "application/json", "", 400},
		{"POST", "/assign", `{"Listing":"own-1","AreaID":"a","Revision":0}`, "fixture", "valid", "application/json", "", 200},
		{"POST", "/assign", url.Values{"csrf": {"valid"}, "listing": {"own-2"}, "area": {"a"}, "revision": {"0"}, "key": {"form1"}}.Encode(), "fixture", "", "application/x-www-form-urlencoded", "", 200},
	} {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		r.Header.Set("Authorization", c.auth)
		r.Header.Set("X-CSRF", c.csrf)
		r.Header.Set("Content-Type", c.content)
		r.Header.Set("Idempotency-Key", "http1")
		r.Header.Set("Admin", c.admin)
		r.Header.Set("Accept", "text/html")
		out := httptest.NewRecorder()
		web.ServeHTTP(out, r)
		if out.Code != c.status {
			t.Fatalf("%+v status %d %s", c, out.Code, out.Body.String())
		}
		if out.Header().Get("Cache-Control") != "no-store" || strings.Contains(out.Body.String(), "<script>") {
			t.Fatal("privacy/XSS headers")
		}
	}
	var b bytes.Buffer
	if e := RunCLI(context.Background(), s, owner, []string{"catalog"}, &b); e != nil || !strings.Contains(b.String(), "a") {
		t.Fatal(b.String(), e)
	}
	for _, label := range []string{"bad\u2028line", "bad\u2029paragraph", "bad\u202eright", "bad\nline", string([]byte{255})} {
		if e := Validate(Area{"n", "a", label, true}); e == nil {
			t.Fatal(label)
		}
	}
}

func TestSharingRevocationAndStrictRequestBoundary(t *testing.T) {
	ctx := context.Background()
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "A")
	if _, e := s.SetSharing(ctx, owner, "share", true, 0); e == nil {
		t.Fatal("member policy write")
	}
	v, e := s.GetSharing(ctx, admin)
	if e != nil || v.Enabled || v.Revision != 0 {
		t.Fatal(v, e)
	}
	v, e = s.SetSharing(ctx, admin, "share", true, 0)
	if e != nil || !v.Enabled || v.Revision != 1 {
		t.Fatal(v, e)
	}
	if _, e = s.SetSharing(ctx, admin, "share-bad", false, 0); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if _, e = s.SetSharing(ctx, admin, "unshare", false, 1); e != nil {
		t.Fatal(e)
	}
	out, e := s.ExportSharing(ctx, owner)
	if e != nil || len(out) != 0 {
		t.Fatal(out, e)
	}
	// Composite references reject a cross-network association even outside service.
	if _, e = s.DB.Exec(`INSERT INTO future_listing_areas(network,listing,area_id,active,revision,updated) VALUES('other','own-x','a',1,1,0)`); e == nil {
		t.Fatal("foreign key scope")
	}
	auth := func(r *http.Request, m bool) (Identity, error) { return admin, nil }
	web := Web{s, auth, func(*http.Request) string { return "fixture" }, []byte(strings.Repeat("k", 32))}
	cases := []struct {
		path, body, kind string
		code             int
	}{{"/admin", `{"ID":"a","Label":"A","Revision":1,"Visible":true,"Archived":false} {}`, "application/json", 400}, {"/admin", `{"ID":"b","Label":"` + strings.Repeat("x", 70000) + `","Visible":true}`, "application/json", 413}, {"/assign", "listing=own-x&area=a&area=b&revision=0&key=dup", "application/x-www-form-urlencoded", 400}}
	for _, c := range cases {
		r := httptest.NewRequest("POST", c.path, strings.NewReader(c.body))
		r.Header.Set("Content-Type", c.kind)
		r.Header.Set("Idempotency-Key", "strict")
		w := httptest.NewRecorder()
		web.ServeHTTP(w, r)
		if w.Code != c.code {
			t.Fatal(c.path, w.Code, w.Body.String())
		}
	}
}

func TestEncryptedActorBoundPaging(t *testing.T) {
	ctx := context.Background()
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "A")
	for i := 0; i < 4; i++ {
		if _, e := s.Assign(ctx, owner, fmt.Sprintf("page%d", i), Assign{fmt.Sprintf("own-%d", i), "a", 0}); e != nil {
			t.Fatal(e)
		}
	}
	key := []byte(strings.Repeat("k", 32))
	first, e := s.FilterPage(ctx, owner, "a", "", 2, key)
	if e != nil || len(first.Listings) != 2 || first.Next == "" {
		t.Fatal(first, e)
	}
	next, e := s.FilterPage(ctx, owner, "a", first.Next, 2, key)
	if e != nil || strings.Join(next.Listings, ",") != "own-2,own-3" || next.Next != "" {
		t.Fatal(next, e)
	}
	for _, a := range []Identity{admin, {"other", "owner"}} {
		if _, e = s.FilterPage(ctx, a, "a", first.Next, 2, key); e == nil {
			t.Fatal("cursor binding", a)
		}
	}
	if _, e = s.FilterPage(ctx, owner, "a", first.Next+"tampered", 2, key); e == nil {
		t.Fatal("cursor tamper")
	}
	s.Now = func() time.Time { return time.Unix(100, 0).Add(15 * time.Minute) }
	if _, e = s.FilterPage(ctx, owner, "a", first.Next, 2, key); e == nil {
		t.Fatal("cursor expiry")
	}
}

func TestCurrentAssociationNoJSAndPagedCLI(t *testing.T) {
	ctx := context.Background()
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "North")
	current, e := s.Current(ctx, owner, "own-x")
	if e != nil || current.Revision != 0 || current.AreaID != "" {
		t.Fatal(current, e)
	}
	if _, e = s.Assign(ctx, owner, "select", Assign{"own-x", "a", 0}); e != nil {
		t.Fatal(e)
	}
	current, e = s.Current(ctx, owner, "own-x")
	if e != nil || current.AreaID != "a" || current.Revision != 1 {
		t.Fatal(current, e)
	}
	if _, e = s.Current(ctx, owner, "someone-else"); !errors.Is(e, ErrDenied) {
		t.Fatal("owner boundary", e)
	}
	web := Web{s, func(*http.Request, bool) (Identity, error) { return owner, nil }, func(*http.Request) string { return "fixture" }, []byte(strings.Repeat("k", 32))}
	for _, path := range []string{"/catalog?listing=own-x", "/filter?area=a", "/assignment?listing=own-x"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Accept", "text/html")
		out := httptest.NewRecorder()
		web.ServeHTTP(out, r)
		if out.Code != 200 || !strings.Contains(out.Body.String(), "own-x") {
			t.Fatal(path, out.Code, out.Body.String())
		}
		for _, forbidden := range []string{"navigator.geolocation", "<script", "https://", "latitude", "longitude"} {
			if strings.Contains(out.Body.String(), forbidden) {
				t.Fatal("location/browser dependency", forbidden)
			}
		}
	}
	var out bytes.Buffer
	if e = RunPagedCLI(ctx, s, owner, []byte(strings.Repeat("k", 32)), []string{"filter", "a"}, &out); e != nil || !strings.Contains(out.String(), "own-x") {
		t.Fatal(out.String(), e)
	}
	if _, e = s.Edit(ctx, admin, "hide", Edit{"a", "North", false, false, 1}); e != nil {
		t.Fatal(e)
	}
	current, e = s.Current(ctx, owner, "own-x")
	if e != nil || current.AreaID != "" || current.Revision != 1 {
		t.Fatal("hidden area leak", current, e)
	}
}

func TestCleanupSurvivesRetryQuota(t *testing.T) {
	ctx := context.Background()
	p := &fixturePolicy{}
	s := newService(t, filepath.Join(t.TempDir(), "db"), p)
	createArea(t, s, "a", "A")
	if _, e := s.Assign(ctx, owner, "assign", Assign{"own-x", "a", 0}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(`WITH RECURSIVE seq(x) AS(SELECT 1 UNION ALL SELECT x+1 FROM seq WHERE x<10000) INSERT INTO future_area_retries(network,member,key,fingerprint,result,created) SELECT 'n','owner','quota-'||x,'x','{}',0 FROM seq`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Assign(ctx, owner, "blocked", Assign{"own-y", "a", 0}); e == nil {
		t.Fatal("quota not applied")
	}
	for i := 0; i < 2; i++ {
		if e := s.CleanupListing(ctx, owner, "cleanup", "own-x"); e != nil {
			t.Fatal("cleanup blocked by quota", e)
		}
	}
	var count int
	if e := s.DB.QueryRow(`SELECT count(*) FROM future_area_audit WHERE action='cleanup' AND object='own-x'`).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
}
