package sqlstore_test

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore/sqlitetest"
)

var ctx = context.Background()

func tables(t *testing.T, db *sql.DB) map[string]bool {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out[n] = true
	}
	return out
}

// Existing (core) listings must survive migration and rollback untouched.
func TestMigrationPreservesExistingRowsAndRollsBack(t *testing.T) {
	db := sqlitetest.Open(t, "")
	// stand-in for the core listings table (shape copied from core 0001_init.sql)
	if _, err := db.Exec(`CREATE TABLE listings (id TEXT PRIMARY KEY, network_id TEXT NOT NULL, member_id TEXT NOT NULL, type TEXT NOT NULL,
	  title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', price INTEGER, status TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		if _, err := db.Exec(`INSERT INTO listings VALUES (?,?,?,?,?,?,?,?,?,?)`, "L"+string(rune('A'+i%26))+string(rune('a'+i/26)), "n1", "m1", "offer", "Title", "desc", nil, "active", 1, 2); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := func() string {
		rows, _ := db.Query(`SELECT id||title||status||created_at FROM listings ORDER BY id`)
		defer rows.Close()
		var b strings.Builder
		for rows.Next() {
			var s string
			rows.Scan(&s)
			b.WriteString(s)
		}
		return b.String()
	}
	before := snapshot()
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := sqlstore.Migrate(ctx, db); err != nil { // idempotent
		t.Fatal(err)
	}
	tb := tables(t, db)
	for _, want := range []string{"listing_media", "listing_tags", "listing_category", "listing_availability", "coarse_area", "taxonomy_category", "taxonomy_tag"} {
		if !tb[want] {
			t.Errorf("missing table %s", want)
		}
	}
	if snapshot() != before {
		t.Fatal("migration changed core listings")
	}
	if err := sqlstore.Rollback(ctx, db); err != nil {
		t.Fatal(err)
	}
	tb = tables(t, db)
	if tb["listing_media"] || tb["coarse_area"] || tb["discovery_revision"] {
		t.Fatal("rollback left module tables:", tb)
	}
	if !tb["listings"] || snapshot() != before {
		t.Fatal("rollback damaged core listings")
	}
	if err := sqlstore.Rollback(ctx, db); err != nil { // nothing left to roll back
		t.Fatal(err)
	}
	// re-apply after rollback works and the legacy rows are still there
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if snapshot() != before {
		t.Fatal("re-migration changed core listings")
	}
}

func TestIdempotencyStore(t *testing.T) {
	db := sqlitetest.Open(t, "")
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := sqlstore.New(db)
	now := time.Unix(1_800_000_000, 0)
	r, err := s.IdemBegin(ctx, "n1", "a", "k1", "h1", now)
	if err != nil || r.State != "new" {
		t.Fatal(r, err)
	}
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k1", "h1", now); r.State != "inflight" {
		t.Fatal(r)
	}
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k1", "OTHER", now); r.State != "mismatch" {
		t.Fatal(r)
	}
	if err := s.IdemFinish(ctx, "n1", "a", "k1", 201, "application/json", []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k1", "h1", now); r.State != "replay" || r.Status != 201 || string(r.Body) != `{"ok":true}` {
		t.Fatal(r)
	}
	// scoped by network and actor
	for _, c := range [][2]string{{"n2", "a"}, {"n1", "b"}} {
		if r, _ = s.IdemBegin(ctx, c[0], c[1], "k1", "h1", now); r.State != "new" {
			t.Fatal("key leaked across scope", c, r)
		}
	}
	// abort lets a failed attempt retry
	s.IdemBegin(ctx, "n1", "a", "k2", "h", now)
	s.IdemAbort(ctx, "n1", "a", "k2")
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k2", "h", now); r.State != "new" {
		t.Fatal(r)
	}
	// stuck in-flight keys are taken over; old keys expire
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k2", "h", now.Add(6*time.Minute)); r.State != "new" {
		t.Fatal("stale inflight not recovered", r)
	}
	if r, _ = s.IdemBegin(ctx, "n1", "a", "k1", "h1", now.Add(25*time.Hour)); r.State != "new" {
		t.Fatal("expired key not recycled", r)
	}
	// exactly one winner among concurrent identical begins
	var wg sync.WaitGroup
	var mu sync.Mutex
	news := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := s.IdemBegin(ctx, "n1", "z", "race", "h", now)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			if r.State == "new" {
				news++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if news != 1 {
		t.Fatal("concurrent begins:", news)
	}
}

// Every repository read/write must be network scoped even when a caller
// hands over another network's identifiers.
func TestRepositoryNetworkScoping(t *testing.T) {
	db := sqlitetest.Open(t, "")
	sqlstore.Migrate(ctx, db)
	s := sqlstore.New(db)
	now := time.Unix(1_800_000_000, 0).UTC()
	rec := d.ListingRecord{ID: "L1", Network: "n1", OwnerID: "a", Kind: d.KindOffer, Title: "Bike", Status: d.StatusActive, OwnerActive: true, Created: now, Updated: now}
	if err := s.UpsertListing(ctx, rec); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetListing(ctx, "n2", "L1"); err != d.ErrNotFound {
		t.Fatal(err)
	}
	if _, err := s.SaveDiscovery(ctx, d.SaveInput{Network: "n2", ListingID: "L1", ActorID: "x", Now: now, MaxNetworkTags: 5}); err != d.ErrNotFound {
		t.Fatal(err)
	}
	if _, err := s.Renew(ctx, "n2", "L1", "x", 0, now.Add(time.Hour), now); err != d.ErrNotFound {
		t.Fatal(err)
	}
	page, err := s.Search(ctx, d.SearchParams{Network: "n2", Viewer: "x", Now: now, Scope: d.ScopeAll, Limit: 10})
	if err != nil || len(page.Items) != 0 {
		t.Fatal(page, err)
	}
	// owner is immutable
	rec.OwnerID = "other"
	if err := s.UpsertListing(ctx, rec); err == nil {
		t.Fatal("owner change accepted")
	}
	// the same id may exist independently in another network
	rec.Network, rec.OwnerID = "n2", "z"
	if err := s.UpsertListing(ctx, rec); err != nil {
		t.Fatal(err)
	}
}
