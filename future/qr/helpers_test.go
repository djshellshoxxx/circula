package qr_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/qrfixture"
)

const origin = "https://circula.example"

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}
func (c *clock) Add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type env struct {
	t     *testing.T
	path  string
	db    *sql.DB
	store *qr.Store
	fx    *qrfixture.Fixture
	svc   *qr.Service
	clk   *clock
	cfg   qr.Config
}

var secret = []byte("0123456789abcdef0123456789abcdef-test-secret")

func openDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	if !haveDriver {
		t.Skip("no SQLite driver linked: run with -tags sqlite_cgo_test (see VALIDATION.md)")
	}
	db, err := sql.Open(driverName, "file:"+path+dsnOpts)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { db.Close() })
	return db
}

// newEnv builds a migrated database with two networks:
// n1 "Alpha": a (Alice), b (Bob), c (Carol), d (Dan), all limit 1000.
// n2 "Beta": x (Xena).
func newEnv(t *testing.T, mod ...func(*qr.Config)) *env {
	t.Helper()
	e := &env{t: t, path: filepath.Join(t.TempDir(), "qr.db"), fx: &qrfixture.Fixture{},
		clk: &clock{t: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)}}
	e.db = openDB(t, e.path)
	ctx := context.Background()
	must(t, qrfixture.Migrate(ctx, e.db))
	must(t, qrfixture.AddNetwork(ctx, e.db, "n1", "Alpha", true))
	must(t, qrfixture.AddNetwork(ctx, e.db, "n2", "Beta", true))
	for _, m := range [][2]string{{"a", "Alice"}, {"b", "Bob"}, {"c", "Carol"}, {"d", "Dan"}} {
		must(t, qrfixture.AddMember(ctx, e.db, "n1", m[0], m[1], 1000))
	}
	must(t, qrfixture.AddMember(ctx, e.db, "n2", "x", "Xena", 1000))
	e.store = qr.NewStore(e.db)
	must(t, e.store.Migrate(ctx))
	e.cfg = qr.Config{Origin: origin, AliasSecret: secret, Now: e.clk.Now, NetworkEnabled: allEnabled}
	for _, f := range mod {
		f(&e.cfg)
	}
	e.svc = e.newService(e.store)
	return e
}

func (e *env) newService(st *qr.Store) *qr.Service {
	e.t.Helper()
	s, err := qr.NewService(e.cfg, st, e.fx)
	if err != nil {
		e.t.Fatal(err)
	}
	return s
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func p(net, member string) qr.Principal { return qr.Principal{NetworkID: net, MemberID: member} }

func (e *env) counts() qrfixture.Counts {
	e.t.Helper()
	c, err := qrfixture.Count(context.Background(), e.db)
	must(e.t, err)
	return c
}

func (e *env) accept(net, tx, actor string) error {
	return e.store.Update(context.Background(), func(ctx context.Context, q qr.DBTX) error {
		return e.fx.Accept(ctx, q, net, tx, actor)
	})
}

func (e *env) scalar(query string, args ...any) int64 {
	e.t.Helper()
	var n int64
	must(e.t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

func allEnabled(context.Context, qr.DBTX, string) (bool, error) { return true, nil }
