package discovery_test

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/imgsan"
	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore/sqlitetest"
)

var ctx = context.Background()

type env struct {
	t     *testing.T
	dir   string
	db    *sql.DB
	store *sqlstore.Store
	blobs *d.FSBlobs
	svc   *d.Service
	clock *testkit.Clock
	cfg   d.Config
	img   imgsan.Limits
}

var (
	alice  = d.Actor{ID: "alice", Network: "n1", Role: d.RoleMember, Active: true, CSRF: "csrf-alice"}
	bob    = d.Actor{ID: "bob", Network: "n1", Role: d.RoleMember, Active: true, CSRF: "csrf-bob"}
	admin  = d.Actor{ID: "adm", Network: "n1", Role: d.RoleAdmin, Active: true, CSRF: "csrf-adm"}
	carol  = d.Actor{ID: "carol", Network: "n2", Role: d.RoleMember, Active: true, CSRF: "csrf-carol"}
	admin2 = d.Actor{ID: "adm2", Network: "n2", Role: d.RoleAdmin, Active: true, CSRF: "csrf-adm2"}
)

func testKey(b byte) []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = b
	}
	return k
}

func newEnv(t *testing.T) *env {
	t.Helper()
	if !sqlitetest.Available() {
		t.Skip("run with -tags=sqlite_cgo_test (real SQLite)")
	}
	dir := t.TempDir()
	e := &env{t: t, dir: dir, clock: testkit.NewClock()}
	e.cfg = d.Config{CursorKeys: [][]byte{testKey(1)}, Now: e.clock.Now, UploadsPerHour: 1000}
	e.img = imgsan.Limits{FullMaxEdge: 600, ThumbMaxEdge: 120}
	e.open()
	return e
}

// open (re)opens the database and filesystem blob store: simulates a restart.
func (e *env) open() {
	e.t.Helper()
	if e.db != nil {
		e.db.Close()
	}
	db, err := sqlitetest.OpenFile(filepath.Join(e.dir, "d.db"))
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { db.Close() })
	if err := sqlstore.Migrate(ctx, db); err != nil {
		e.t.Fatal(err)
	}
	e.db = db
	e.store = sqlstore.New(db)
	e.blobs, err = d.NewFSBlobs(filepath.Join(e.dir, "blobs"))
	if err != nil {
		e.t.Fatal(err)
	}
	e.svc, err = d.NewService(e.store, e.blobs, e.cfg, e.img)
	if err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) listing(network, id, owner, title, desc string) d.ListingRecord {
	e.t.Helper()
	return e.listingAt(network, id, owner, title, desc, d.StatusActive, e.clock.Now())
}

func (e *env) listingAt(network, id, owner, title, desc, status string, updated time.Time) d.ListingRecord {
	e.t.Helper()
	rec := d.ListingRecord{ID: id, Network: network, OwnerID: owner, Kind: d.KindOffer, Title: title, Description: desc,
		Status: status, OwnerActive: true, Created: updated.Add(-time.Hour), Updated: updated}
	if err := e.svc.SyncListing(ctx, rec); err != nil {
		e.t.Fatal(err)
	}
	return rec
}

func (e *env) mustSearch(a d.Actor, f d.Filters) d.SearchResult {
	e.t.Helper()
	r, err := e.svc.Search(ctx, a, f, "")
	if err != nil {
		e.t.Fatal(err)
	}
	return r
}

func ids(r d.SearchResult) []string {
	out := make([]string, len(r.Items))
	for i, it := range r.Items {
		out[i] = it.Listing.ID
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(root, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !de.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
