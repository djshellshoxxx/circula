//go:build sqlite_cgo_test

package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/sqlitefixture"
	"github.com/djshellshoxxx/circula/future/favorites/internal/storetest"
	"github.com/djshellshoxxx/circula/future/favorites/internal/testkit"
	"github.com/djshellshoxxx/circula/future/favorites/sqlstore"
)

var ctx = context.Background()

func openMigrated(t testing.TB) (*sql.DB, string) {
	t.Helper()
	p := sqlitefixture.Path(t)
	db := sqlitefixture.Open(t, p)
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db, p
}

func TestConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) favorites.Store {
		db, _ := openMigrated(t)
		return sqlstore.New(db)
	})
}

func TestMigrateIdempotentAndRollback(t *testing.T) {
	p := sqlitefixture.Path(t)
	db := sqlitefixture.Open(t, p)
	if v, err := sqlstore.Version(ctx, db); err != nil || v != 0 {
		t.Fatalf("fresh version %d %v", v, err)
	}
	for i := 0; i < 3; i++ {
		if err := sqlstore.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
	}
	if v, _ := sqlstore.Version(ctx, db); v != sqlstore.SchemaVersion {
		t.Fatalf("version %d", v)
	}
	st := sqlstore.New(db)
	if err := st.Update(ctx, func(tx favorites.Tx) error {
		return tx.InsertFavorite(ctx, favorites.Entry{Favorite: favorites.Favorite{Network: "n", Owner: "a", Target: "b"}, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}); err != nil {
		t.Fatal(err)
	}
	// a database written by a newer module version is refused, not downgraded
	if _, err := db.Exec(`INSERT INTO favorites_schema_migrations(version, applied_at) VALUES (99, 'x')`); err != nil {
		t.Fatal(err)
	}
	if err := sqlstore.Migrate(ctx, db); !errors.Is(err, sqlstore.ErrNewerSchema) {
		t.Fatalf("newer schema: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM favorites_schema_migrations WHERE version=99`); err != nil {
		t.Fatal(err)
	}
	// documented rollback removes module tables only
	if _, err := db.Exec(`CREATE TABLE host_table(x)`); err != nil {
		t.Fatal(err)
	}
	if err := sqlstore.Rollback(ctx, db); err != nil {
		t.Fatal(err)
	}
	var names []string
	rows, _ := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		names = append(names, n)
	}
	rows.Close()
	if strings.Join(names, ",") != "host_table" {
		t.Fatalf("tables after rollback: %v", names)
	}
	// forward again after rollback
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if v, _ := sqlstore.Version(ctx, db); v != 1 {
		t.Fatal(v)
	}
}

func TestSchemaConstraintsRejectCorruptRows(t *testing.T) {
	db, _ := openMigrated(t)
	const ins = `INSERT INTO member_favorites(network_id, owner_member_id, target_member_id, nickname, pinned, position, created_at, updated_at) VALUES (?,?,?,?,?,?, 'c','u')`
	cases := map[string][]any{
		"self":         {"n", "a", "a", "", 0, 1},
		"empty net":    {"", "a", "b", "", 0, 1},
		"bad pinned":   {"n", "a", "b", "", 2, 1},
		"negative pos": {"n", "a", "b", "", 0, -1},
		"long nick":    {"n", "a", "b", strings.Repeat("x", 81), 0, 1},
		"null owner":   {"n", nil, "b", "", 0, 1},
	}
	for name, args := range cases {
		if _, err := db.Exec(ins, args...); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := db.Exec(ins, "n", "a", "b", "ok", 1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ins, "n", "a", "b", "ok", 1, 1); err == nil {
		t.Fatal("UNIQUE(network, owner, target) not enforced")
	}
	// same owner and target in a different network is a different relation
	if _, err := db.Exec(ins, "n2", "a", "b", "ok", 1, 1); err != nil {
		t.Fatal(err)
	}
}

func newService(t testing.TB, db *sql.DB, dir *testkit.Directory) *favorites.Service {
	s, err := favorites.NewService(favorites.Config{Store: sqlstore.New(db), Policy: dir, Enabled: dir.Enabled, Trades: dir})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func directory() *testkit.Directory {
	d := testkit.NewDirectory()
	for _, n := range []string{"n1", "n2"} {
		for _, id := range []string{"alice", "bob", "carol", "dave"} {
			d.Add(n, id)
		}
	}
	d.AddTrade("t1", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "bike", Completed: true})
	return d
}

// Restart recovery: everything the service wrote survives closing the
// database and reopening the same file with a brand new Service.
func TestRestartDurability(t *testing.T) {
	p := sqlitefixture.Path(t)
	dir := directory()
	alice := favorites.Actor{Network: "n1", Member: "alice"}
	db1, err := sql.Open(sqlitefixture.Driver, sqlitefixture.DSN(p))
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlstore.Migrate(ctx, db1); err != nil {
		t.Fatal(err)
	}
	s1 := newService(t, db1, dir)
	for _, id := range []string{"bob", "carol", "dave"} {
		if _, _, err := s1.Add(ctx, alice, favorites.AddRequest{Target: id, Nickname: "nick " + id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s1.SetPinned(ctx, alice, "dave", true); err != nil {
		t.Fatal(err)
	}
	pg, _ := s1.List(ctx, alice, favorites.ListOptions{})
	if _, err := s1.Reorder(ctx, alice, favorites.ReorderRequest{Revision: pg.Revision, Targets: []string{"carol", "bob"}}); err != nil {
		t.Fatal(err)
	}
	d1, err := s1.RepeatTrade(ctx, alice, "t1", "key-1")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s1.List(ctx, alice, favorites.ListOptions{})
	if err := db1.Close(); err != nil {
		t.Fatal(err)
	}

	db2 := sqlitefixture.Open(t, p)
	if err := sqlstore.Migrate(ctx, db2); err != nil { // re-running migrations on restart is a no-op
		t.Fatal(err)
	}
	s2 := newService(t, db2, dir)
	after, err := s2.List(ctx, alice, favorites.ListOptions{})
	if err != nil || fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("lost state across restart:\n%v\n%v\n%v", before, after, err)
	}
	if got := after.Items[0].Target + after.Items[1].Target + after.Items[2].Target; got != "dave"+"carol"+"bob" {
		t.Fatalf("order %s", got)
	}
	d2, err := s2.RepeatTrade(ctx, alice, "t1", "key-1")
	if err != nil || d2 != d1 {
		t.Fatalf("draft retry after restart must return same intent: %+v vs %+v (%v)", d2, d1, err)
	}
	// stale revision from before the restart still means what it meant
	if _, err := s2.Reorder(ctx, alice, favorites.ReorderRequest{Revision: pg.Revision, Targets: []string{"bob"}}); !errors.Is(err, favorites.ErrStale) {
		t.Fatalf("stale across restart: %v", err)
	}
}

// Two independent Store handles on one file (as a crash-restart overlap or a
// misconfigured second pool would produce) still keep uniqueness and cap.
func TestConcurrentWritersOnOneFile(t *testing.T) {
	_, p := openMigrated(t)
	dir := directory()
	for i := 0; i < 12; i++ {
		dir.Add("n1", fmt.Sprintf("p%d", i))
	}
	alice := favorites.Actor{Network: "n1", Member: "alice"}
	var svcs []*favorites.Service
	for i := 0; i < 3; i++ {
		db := sqlitefixture.Open(t, p)
		s, err := favorites.NewService(favorites.Config{Store: sqlstore.New(db), Policy: dir, Enabled: dir.Enabled, Trades: dir, MaxFavorites: 5})
		if err != nil {
			t.Fatal(err)
		}
		svcs = append(svcs, s)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	created := 0
	for i := 0; i < 36; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, c, err := svcs[i%3].Add(ctx, alice, favorites.AddRequest{Target: fmt.Sprintf("p%d", i%12)})
			if err != nil && !errors.Is(err, favorites.ErrLimit) {
				t.Errorf("add: %v", err)
			}
			if c {
				mu.Lock()
				created++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	pg, err := svcs[0].List(ctx, alice, favorites.ListOptions{})
	if err != nil || created != 5 || len(pg.Items) != 5 {
		t.Fatalf("created %d listed %d err %v", created, len(pg.Items), err)
	}
}

// The host's departure transaction can include the purge, and a rollback of
// that transaction leaves the favorites intact.
func TestPurgeMemberTxInsideHostTransaction(t *testing.T) {
	db, _ := openMigrated(t)
	dir := directory()
	s := newService(t, db, dir)
	for _, o := range []string{"alice", "bob", "carol"} {
		if _, _, err := s.Add(ctx, favorites.Actor{Network: "n1", Member: o}, favorites.AddRequest{Target: "dave"}); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sqlstore.PurgeMemberTx(ctx, tx, "n1", "dave")
	if err != nil || res.Favorites != 3 {
		t.Fatalf("%+v %v", res, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	pg, _ := s.List(ctx, favorites.Actor{Network: "n1", Member: "alice"}, favorites.ListOptions{})
	if len(pg.Items) != 1 {
		t.Fatal("rollback of host transaction must keep favorites")
	}
	tx, _ = db.BeginTx(ctx, nil)
	if _, err := sqlstore.PurgeMemberTx(ctx, tx, "n1", "dave"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM member_favorites`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d rows left", n)
	}
}

// SQL injection attempts through identifiers and text are plain data.
func TestHostileStringsAreData(t *testing.T) {
	db, _ := openMigrated(t)
	dir := directory()
	evil := `x');DROP/**/TABLE/**/member_favorites;--`
	dir.Add("n1", evil)
	s := newService(t, db, dir)
	alice := favorites.Actor{Network: "n1", Member: "alice"}
	if _, _, err := s.Add(ctx, alice, favorites.AddRequest{Target: evil, Nickname: `'; DELETE FROM member_favorites; --`}); err != nil {
		t.Fatal(err)
	}
	pg, err := s.List(ctx, alice, favorites.ListOptions{})
	if err != nil || len(pg.Items) != 1 || pg.Items[0].Target != evil {
		t.Fatalf("%+v %v", pg, err)
	}
}

func TestPersistedTextIsExactlyWhatWasValidated(t *testing.T) {
	db, _ := openMigrated(t)
	dir := directory()
	s := newService(t, db, dir)
	alice := favorites.Actor{Network: "n1", Member: "alice"}
	if _, _, err := s.Add(ctx, alice, favorites.AddRequest{Target: "bob", Nickname: "  Zoë 家  "}); err != nil {
		t.Fatal(err)
	}
	var nick string
	_ = db.QueryRow(`SELECT nickname FROM member_favorites`).Scan(&nick)
	if nick != "Zoë 家" {
		t.Fatalf("%q", nick)
	}
}
