//go:build sqlite_cgo_test

// Test-only real-SQLite coverage using github.com/mattn/go-sqlite3 (cgo, MIT).
// This does NOT qualify the production driver (modernc.org/sqlite) the core
// uses; it exercises real SQL semantics (CAS, transactions, constraints,
// restart durability) for the adapter.
package onboarding

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func openDB(t testing.TB, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func openSQL(t *testing.T) Store {
	t.Helper()
	db := openDB(t, filepath.Join(t.TempDir(), "o.db"))
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	s, err := NewSQLStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSQLStoreContract(t *testing.T) { storeContract(t, openSQL) }

func TestSQLMigrateAdditiveAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "h.db"))
	// A pre-existing "host" schema with data, standing in for the core ledger.
	if _, err := db.Exec(`CREATE TABLE ledger_entries (id INTEGER PRIMARY KEY, amount INTEGER NOT NULL);
		INSERT INTO ledger_entries VALUES (1, 100), (2, -100);`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ { // idempotent
		if err := Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	var n, sum int
	if err := db.QueryRow(`SELECT count(*), sum(amount) FROM ledger_entries`).Scan(&n, &sum); err != nil || n != 2 || sum != 0 {
		t.Fatal(n, sum, err)
	}
	var v int
	if err := db.QueryRow(`SELECT count(*) FROM onboarding_schema_migrations`).Scan(&v); err != nil || v != 1 {
		t.Fatal(v, err)
	}
	// Rollback removes only module tables.
	s, _ := NewSQLStore(db)
	s.SaveProgress(ctx, Key{"n", "m"}, 0, Progress{Version: 1}, nil, t0)
	if err := Rollback(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM ledger_entries`).Scan(&n); err != nil || n != 2 {
		t.Fatal("rollback touched host table", n, err)
	}
	var names int
	db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name LIKE '%onboarding%'`).Scan(&names)
	if names != 0 {
		t.Fatalf("%d onboarding objects remain", names)
	}
	// Re-migrate after rollback works (forward-only recreation).
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
}

func TestSQLRestartDurability(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "r.db")
	db := openDB(t, path)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	st, _ := NewSQLStore(db)
	tr, _ := NewTracker(TrackerConfig{Store: st, Now: func() time.Time { return t0 }, Links: LinkPolicy{PathPrefixes: []string{"/terms"}}})
	me := Actor{NetworkID: "n", MemberID: "m", Active: true}
	admin := Actor{NetworkID: "n", MemberID: "adm", Active: true, Admin: true}
	if _, err := tr.Apply(ctx, me, 1, Action{ActComplete, Credit}); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.Apply(ctx, me, 1, Action{ActSkip, Partner}); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.SaveCopy(ctx, admin, Copy{Intro: "Welcome", TermsURL: "/terms"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := tr.SetMetrics(ctx, admin, true); err != nil {
		t.Fatal(err)
	}
	db.Close() // "restart"

	db2 := openDB(t, path)
	if err := Migrate(ctx, db2); err != nil { // migrating again on restart is safe
		t.Fatal(err)
	}
	st2, _ := NewSQLStore(db2)
	tr2, _ := NewTracker(TrackerConfig{Store: st2, Links: LinkPolicy{PathPrefixes: []string{"/terms"}}})
	p, err := tr2.Progress(ctx, me)
	if err != nil || !p.IsCompleted(Credit) || !p.IsSkipped(Partner) || p.Revision != 2 {
		t.Fatalf("%+v %v", p, err)
	}
	c, err := tr2.Copy(ctx, me)
	if err != nil || c.Intro != "Welcome" {
		t.Fatalf("%+v %v", c, err)
	}
	if on, _ := tr2.MetricsEnabled(ctx, me); !on {
		t.Fatal("metrics setting lost")
	}
}

func TestSQLConcurrentTabsLoseNothing(t *testing.T) {
	ctx := context.Background()
	s := openSQL(t)
	tr, _ := NewTracker(TrackerConfig{Store: s, Now: func() time.Time { return t0 }})
	me := Actor{NetworkID: "n", MemberID: "m", Active: true}
	var wg sync.WaitGroup
	for rep := 0; rep < 4; rep++ {
		for st := Terms; st < StepCount; st++ {
			wg.Add(1)
			go func(st Step) {
				defer wg.Done()
				if _, err := tr.Apply(ctx, me, 1, Action{ActComplete, st}); err != nil {
					t.Error(err)
				}
			}(st)
		}
	}
	wg.Wait()
	p, err := tr.Progress(ctx, me)
	if err != nil || !p.Done() {
		t.Fatalf("lost a step: %+v %v", p, err)
	}
}

func TestSQLConcurrentCountersExact(t *testing.T) {
	ctx := context.Background()
	s := openSQL(t)
	tr, _ := NewTracker(TrackerConfig{Store: s, Now: func() time.Time { return t0 }})
	admin := Actor{NetworkID: "n", MemberID: "adm", Active: true, Admin: true}
	if err := tr.SetMetrics(ctx, admin, true); err != nil {
		t.Fatal(err)
	}
	const members = 12
	var wg sync.WaitGroup
	for i := 0; i < members; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a := Actor{NetworkID: "n", MemberID: "m" + string(rune('a'+i)), Active: true}
			for _, act := range []Action{{ActShare, 0}, {ActComplete, Terms}, {ActComplete, Terms}} {
				if _, err := tr.Apply(ctx, a, 1, act); err != nil {
					t.Error(err)
				}
			}
		}(i)
	}
	wg.Wait()
	agg, err := tr.Aggregate(ctx, admin)
	if err != nil || len(agg) != int(StepCount) || agg[Terms].Completed != members || agg[Terms].Suppressed {
		t.Fatalf("%+v %v", agg, err)
	}
	if !agg[Credit].Suppressed {
		t.Fatal("small cohort must be suppressed")
	}
}

func TestSQLConstraintsAndCorruptRows(t *testing.T) {
	ctx := context.Background()
	db := openDB(t, filepath.Join(t.TempDir(), "c.db"))
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	s, _ := NewSQLStore(db)
	for _, q := range []string{
		`INSERT INTO member_onboarding_progress VALUES ('n','m',1,32,0,0,0,1,'2026-11-16T00:00:00Z')`,
		`INSERT INTO member_onboarding_progress VALUES ('n','m',1,1,1,0,0,1,'2026-11-16T00:00:00Z')`,
		`INSERT INTO member_onboarding_progress VALUES ('n','m',0,1,0,0,0,1,'2026-11-16T00:00:00Z')`,
		`INSERT INTO member_onboarding_progress VALUES ('n','m',1,1,0,2,0,1,'2026-11-16T00:00:00Z')`,
		`INSERT INTO member_onboarding_progress VALUES ('','m',1,1,0,0,0,1,'2026-11-16T00:00:00Z')`,
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("constraint missed: %s", q)
		}
	}
	// Bad timestamp (tampered row) is reported, not trusted.
	if _, err := db.Exec(`INSERT INTO member_onboarding_progress VALUES ('n','m',1,1,0,0,0,1,'yesterday')`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.LoadProgress(ctx, Key{"n", "m"}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("%v", err)
	}
	// Cancelled context does not write.
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.SaveProgress(cctx, Key{"n", "z"}, 0, Progress{Version: 1}, nil, t0); err == nil {
		t.Fatal("write with cancelled context")
	}
	if _, ok, _ := s.LoadProgress(ctx, Key{"n", "z"}); ok {
		t.Fatal("cancelled write persisted")
	}
}

func TestSQLInjectionIdentifiersRejected(t *testing.T) {
	s := openSQL(t)
	for _, bad := range []string{"x'; DROP TABLE member_onboarding_progress;--", "a b", "", "é", "../x"} {
		if _, _, err := s.LoadProgress(context.Background(), Key{bad, "m"}); !errors.Is(err, ErrInvalidIdentifier) {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

// Acceptance (automated part): running the whole practice flow leaves a real
// host database file byte-identical. The practice server holds no handle to it,
// so this is a regression net for accidental coupling, not proof about the real
// Circula ledger and audit log, which stays an integration-time gate.
func TestPracticeFlowLeavesHostDatabaseByteIdentical(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "host.db")
	db, err := sql.Open("sqlite3", "file:"+path+"?_journal_mode=DELETE&_busy_timeout=10000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE ledger_entries (id INTEGER PRIMARY KEY, account TEXT, amount INTEGER);
		CREATE TABLE admin_audit (id INTEGER PRIMARY KEY, event TEXT);
		INSERT INTO ledger_entries VALUES (1,'a',100),(2,'b',-100);
		INSERT INTO admin_audit VALUES (1,'created');`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	st, _ := NewSQLStore(db)
	if _, err := st.SaveProgress(ctx, Key{"n", "m"}, 0, Progress{Version: 1, Completed: 1}, nil, t0); err != nil {
		t.Fatal(err)
	}
	db.Close()
	before := fileSum(t, path)

	s := newPracticeServer(t, nil)
	c := newClient(t, s)
	c.start()
	c.postStep("/step/terms", "/step/terms", nil)
	c.postStep("/step/credit", "/step/credit", nil)
	c.postStep("/step/listing", "/step/listing", url.Values{"kind": {"offer"}, "title": {"Tutoring"}, "amount": {"20"}})
	c.postStep("/step/partner", "/step/partner", url.Values{"partner": {"practice-bo"}})
	doc := c.get("/step/trade?listing=practice-listing-1").Body.String()
	c.do("POST", "/step/trade/propose", c.hidden(doc))
	doc = c.get("/step/trade").Body.String()
	for _, form := range strings.Split(doc, "<form") {
		if strings.Contains(form, `name="accept" value="1"`) {
			c.do("POST", "/proposal/respond", c.hidden("<form"+form))
		}
	}
	if l := s.sessions[c.cookie.Value].world.Snapshot().Learner(); l.Balance != -30 {
		t.Fatalf("practice flow did not complete: %+v", l)
	}
	c.get("/export.json")
	c.get("/export.csv")

	if after := fileSum(t, path); after != before {
		t.Fatal("host database changed during practice flow")
	}
}

func fileSum(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}
