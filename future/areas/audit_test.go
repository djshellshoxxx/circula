package areas

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditSilentWritesRollback(t *testing.T) {
	for _, table := range []string{"future_areas", "future_listing_areas", "future_area_audit", "future_area_retries", "future_area_network_policy"} {
		t.Run(table, func(t *testing.T) {
			s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
			if table == "future_listing_areas" {
				createArea(t, s, "a", "North")
			}
			if _, e := s.DB.Exec(`CREATE TRIGGER ignored BEFORE INSERT ON ` + table + ` BEGIN SELECT RAISE(IGNORE); END`); e != nil {
				t.Fatal(e)
			}
			var e error
			switch table {
			case "future_listing_areas":
				_, e = s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0})
			case "future_area_network_policy":
				_, e = s.SetSharing(context.Background(), admin, "share", true, 0)
			default:
				_, e = s.Edit(context.Background(), admin, "new", Edit{"a", "North", true, false, 0})
			}
			if e == nil {
				t.Fatal("ignored write acknowledged")
			}
			var retries int
			if e = s.DB.QueryRow(`SELECT count(*) FROM future_area_retries WHERE key IN ('new','assign','share')`).Scan(&retries); e != nil || retries != 0 {
				t.Fatal("partial retry committed", retries, e)
			}
		})
	}
}
func TestAuditArchiveCleanupAndReplay(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	a := createArea(t, s, "a", "North")
	if _, e := s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(`CREATE TRIGGER ignored BEFORE UPDATE ON future_listing_areas BEGIN SELECT RAISE(IGNORE); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Edit(context.Background(), admin, "archive-fail", Edit{"a", "North", true, true, a.Revision}); e == nil {
		t.Fatal("archive cleanup failure acknowledged")
	}
	if _, e := s.DB.Exec(`DROP TRIGGER ignored`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Edit(context.Background(), admin, "archive", Edit{"a", "North", true, true, a.Revision}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e == nil {
		t.Fatal("archived assignment replay exposed")
	}
	if _, e := s.DB.Exec(`CREATE TRIGGER ignored BEFORE DELETE ON future_listing_areas BEGIN SELECT RAISE(IGNORE); END`); e != nil {
		t.Fatal(e)
	}
	if e := s.CleanupListing(context.Background(), owner, "clean", "own-l"); e == nil {
		t.Fatal("ignored cleanup acknowledged")
	}
}
func TestAuditSchemaAndCorruptReplay(t *testing.T) {
	db, e := sql.Open("sqlite", filepath.Join(t.TempDir(), "bad.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`PRAGMA foreign_keys=ON; CREATE TABLE future_areas(network TEXT,id TEXT)`); e != nil {
		t.Fatal(e)
	}
	if e = (Service{DB: db}).Migrate(context.Background()); e == nil {
		t.Fatal("incompatible schema accepted")
	}
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='future_area_schema'`).Scan(&n); e != nil || n != 0 {
		t.Fatal("migration partially initialized", n, e)
	}
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	createArea(t, s, "a", "North")
	if _, e = s.DB.Exec(`UPDATE future_area_retries SET result='{}'`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Edit(context.Background(), admin, "create-a", Edit{"a", "North", true, false, 0}); e == nil {
		t.Fatal("corrupt replay accepted")
	}
}

type cancelPolicy struct {
	base   Policy
	cancel context.CancelFunc
}

func (p cancelPolicy) Check(ctx context.Context, tx *sql.Tx, a Identity, action, object string) error {
	e := p.base.Check(ctx, tx, a, action, object)
	if action == "listing" {
		p.cancel()
	}
	return e
}
func TestAuditCanceledListingPolicy(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	createArea(t, s, "a", "North")
	if _, e := s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e != nil {
		t.Fatal(e)
	}
	for _, paged := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		copy := s
		copy.Policy = cancelPolicy{s.Policy, cancel}
		var e error
		if paged {
			_, e = copy.FilterPage(ctx, owner, "a", "", 100, []byte(strings.Repeat("k", 32)))
		} else {
			_, e = copy.Filter(ctx, owner, "a")
		}
		cancel()
		if !errors.Is(e, context.Canceled) {
			t.Fatal(paged, e)
		}
	}
}
func TestAuditStrictJSONRoutesAndBounds(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	w := Web{Service: s, Auth: func(*http.Request, bool) (Identity, error) { return admin, nil }}
	for _, body := range []string{`{"ID":"a","id":"b","Label":"North"}`, `{"ID":"a","Label":"\ud800"}`, "{\"ID\":\"a\",\"Label\":\"" + string([]byte{255}) + "\"}", strings.Repeat(" ", 65537)} {
		r := httptest.NewRequest("POST", "/admin", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "test")
		rec := httptest.NewRecorder()
		w.ServeHTTP(rec, r)
		if rec.Code != 400 && rec.Code != 413 {
			t.Fatal(rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{"//evil/admin", "/admin/", "/%61dmin", "/admin\\bad"} {
		r := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		w.ServeHTTP(rec, r)
		if rec.Code != 400 {
			t.Fatal(path, rec.Code)
		}
	}
	var out Edit
	if e := strictJSON([]byte(fmt.Sprintf(`{"ID":"a","Label":%q}`, "North")), &out); e != nil || out.Label != "North" {
		t.Fatal(out, e)
	}
}
func TestAuditCleanupRetiresCachedLocationAndRestorationDoesNotReplay(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	a := createArea(t, s, "a", "North")
	if _, e := s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e != nil {
		t.Fatal(e)
	}
	archived, e := s.Edit(context.Background(), admin, "archive", Edit{"a", "North", true, true, a.Revision})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Edit(context.Background(), admin, "restore", Edit{"a", "North", true, false, archived.Revision}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e == nil {
		t.Fatal("restoration replay returned active selection")
	}
	if e = s.CleanupListing(context.Background(), owner, "cleanup", "own-l"); e != nil {
		t.Fatal(e)
	}
	var data string
	if e = s.DB.QueryRow(`SELECT result FROM future_area_retries WHERE network='n' AND member='owner' AND key='assign'`).Scan(&data); e != nil || data != "null" {
		t.Fatal("location history retained", data, e)
	}
	if _, e = s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); !errors.Is(e, ErrUnavailable) {
		t.Fatal("tombstone replay", e)
	}
}
func TestAuditCleanupCachedReplyWriteFailureRollsBack(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	createArea(t, s, "a", "North")
	if _, e := s.Assign(context.Background(), owner, "assign", Assign{"own-l", "a", 0}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(`CREATE TRIGGER ignore_retirement BEFORE UPDATE ON future_area_retries BEGIN SELECT RAISE(IGNORE); END`); e != nil {
		t.Fatal(e)
	}
	if e := s.CleanupListing(context.Background(), owner, "cleanup", "own-l"); e == nil {
		t.Fatal("cache retirement failure acknowledged")
	}
	current, e := s.Current(context.Background(), owner, "own-l")
	if e != nil || !current.Active {
		t.Fatal("partial cleanup committed", current, e)
	}
}
func TestAuditMissingForeignKeysAndContentType(t *testing.T) {
	db, e := sql.Open("sqlite", filepath.Join(t.TempDir(), "bad.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = (Service{DB: db}).Migrate(context.Background()); e == nil {
		t.Fatal("foreign key requirement ignored")
	}
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	w := Web{Service: s, Auth: func(*http.Request, bool) (Identity, error) { return admin, nil }}
	r := httptest.NewRequest("POST", "/admin", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "text/plain")
	out := httptest.NewRecorder()
	w.ServeHTTP(out, r)
	if out.Code != 415 {
		t.Fatal(out.Code)
	}
}
func TestAuditSharingOptOutSurvivesQuotaAndStaleReplay(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	if _, e := s.SetSharing(context.Background(), admin, "enable", true, 0); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(`WITH RECURSIVE counts(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM counts WHERE x<10000) INSERT INTO future_area_retries(network,member,key,fingerprint,result,created) SELECT 'n','admin','full-'||x,'fixed','{}',0 FROM counts`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SetSharing(context.Background(), admin, "normal-disable", false, 1); e == nil {
		t.Fatal("quota fixture failed")
	}
	off, e := s.DisableSharing(context.Background(), admin)
	if e != nil || off.Enabled || off.Revision != 2 {
		t.Fatal(off, e)
	}
	again, e := s.DisableSharing(context.Background(), admin)
	if e != nil || again != off {
		t.Fatal(again, e)
	}
	if _, e = s.SetSharing(context.Background(), admin, "enable", true, 0); e == nil {
		t.Fatal("stale enable replay reported active")
	}
	exports, e := s.ExportSharing(context.Background(), owner)
	if e != nil || len(exports) != 0 {
		t.Fatal(exports, e)
	}
	if _, e = s.DisableSharing(context.Background(), owner); !errors.Is(e, ErrDenied) {
		t.Fatal("member changed network sharing", e)
	}
}
func TestAuditNativeSharingSafetyFormAndCLI(t *testing.T) {
	s := newService(t, filepath.Join(t.TempDir(), "db"), &fixturePolicy{})
	if _, e := s.SetSharing(context.Background(), admin, "on", true, 0); e != nil {
		t.Fatal(e)
	}
	web := Web{Service: s, CSRFToken: func(*http.Request) string { return "safe" }, Auth: func(r *http.Request, mutation bool) (Identity, error) {
		if mutation && (r.ParseForm() != nil || r.PostForm.Get("csrf") != "safe") {
			return Identity{}, ErrDenied
		}
		return admin, nil
	}}
	r := httptest.NewRequest("GET", "/admin", nil)
	out := httptest.NewRecorder()
	web.ServeHTTP(out, r)
	if out.Code != 200 || !strings.Contains(out.Body.String(), "Stop sharing immediately") {
		t.Fatal(out.Code, out.Body.String())
	}
	r = httptest.NewRequest("POST", "/disable-sharing", strings.NewReader("revision=0&csrf=safe"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	out = httptest.NewRecorder()
	web.ServeHTTP(out, r)
	if out.Code != 200 {
		t.Fatal(out.Code, out.Body.String())
	}
	var b strings.Builder
	if e := RunCLI(context.Background(), s, admin, []string{"unshare"}, &b); e != nil || !strings.Contains(b.String(), `"Enabled":false`) {
		t.Fatal(b.String(), e)
	}
}
