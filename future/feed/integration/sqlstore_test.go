package feedintegration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	feed "github.com/djshellshoxxx/circula/future/feed"
	"modernc.org/sqlite"
)

func openStore(tb testing.TB) (*feed.SQLStore, *sql.DB, string) {
	tb.Helper()
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(tb.TempDir(), "feed.db"))}
	u.RawQuery = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		tb.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	tb.Cleanup(func() { db.Close() })
	store, err := feed.NewSQLStore(db)
	if err != nil {
		tb.Fatal(err)
	}
	if err := store.InitSchema(context.Background()); err != nil {
		tb.Fatal(err)
	}
	return store, db, u.String()
}

func source(id int) feed.Source {
	kind := "listing"
	if id%2 == 0 {
		kind = "announcement"
	}
	return feed.Source{Network: "network", Kind: kind, ID: fmt.Sprintf("source-%09d", id), Member: "member", Created: time.Unix(1_700_000_000+int64(id/3), int64(id%2)).UTC(), Visible: true, Active: true}
}

func position(s feed.Source) feed.Cursor {
	return feed.Cursor{Created: s.Created, Kind: s.Kind, ID: s.ID}
}
func precedes(a, b feed.Source) bool {
	if !a.Created.Equal(b.Created) {
		return a.Created.After(b.Created)
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.ID < b.ID
}

func TestSQLiteStoreLifecycleAndRestart(t *testing.T) {
	ctx := context.Background()
	store, db, uri := openStore(t)
	if err := store.InitSchema(ctx); err != nil {
		t.Fatal("repeat schema", err)
	}
	if water, err := store.HighWater(ctx, "network"); err != nil || water != 0 {
		t.Fatal(water, err)
	}
	s := source(1)
	for i := 0; i < 2; i++ {
		if err := store.Upsert(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101})
	if err != nil || len(rows) != 1 || rows[0].Source != s {
		t.Fatalf("idempotent source = %+v, %v", rows, err)
	}
	sequence := rows[0].Sequence
	changed := s
	changed.Created = changed.Created.Add(time.Nanosecond)
	if err := store.Upsert(ctx, changed); err != feed.ErrInvalidSource {
		t.Fatalf("changed creation accepted: %v", err)
	}
	s.Visible = false
	if err := store.Upsert(ctx, s); err != nil {
		t.Fatal(err)
	}
	if rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101}); err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	s.Visible = true
	s.Member = "new-member"
	if err := store.Upsert(ctx, s); err != nil {
		t.Fatal(err)
	}
	rows, err = store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101})
	if err != nil || len(rows) != 1 || rows[0].Sequence != sequence || rows[0].Source.Member != "new-member" {
		t.Fatal(rows, err)
	}
	other := source(2)
	other.Network = "other-network"
	if err := store.Upsert(ctx, other); err != nil {
		t.Fatal(err)
	}
	if water, err := store.HighWater(ctx, "network"); err != nil || water != sequence {
		t.Fatal("network water", water, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := sql.Open("sqlite", uri)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	store, err = feed.NewSQLStore(reopened)
	if err != nil {
		t.Fatal(err)
	}
	rows, err = store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101})
	if err != nil || len(rows) != 1 || rows[0].Sequence != sequence {
		t.Fatal("restart", rows, err)
	}
	for i := 0; i < 2; i++ {
		if err := store.Delete(ctx, s.Network, s.Kind, s.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Upsert(ctx, s); err != feed.ErrInvalidSource {
		t.Fatalf("deleted source resurrected: %v", err)
	}
	unknown := source(99)
	if err := store.Delete(ctx, unknown.Network, unknown.Kind, unknown.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(ctx, unknown); err != feed.ErrInvalidSource {
		t.Fatalf("delayed source resurrected: %v", err)
	}
	if rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101}); err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}

// seed writes realistic immutable reference rows in one committed transaction.
// The production paging path still goes through SQLStore/database/sql.
func seed(tb testing.TB, db *sql.DB, count int, tied bool) {
	tb.Helper()
	tx, err := db.Begin()
	if err != nil {
		tb.Fatal(err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO fnet204_sources(network,kind,source_id,member,order_time,visible,active) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		tb.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < count; i++ {
		s := source(i)
		if tied {
			s.Created = time.Unix(1_700_000_000, 0).UTC()
		}
		if _, err := stmt.Exec(s.Network, s.Kind, s.ID, s.Member, -s.Created.UnixNano(), s.Visible, s.Active); err != nil {
			tb.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		tb.Fatal(err)
	}
}

func TestSQLiteLargeHistoryTraversal(t *testing.T) {
	for _, size := range []int{10_000, 100_000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			store, db, _ := openStore(t)
			seed(t, db, size, false)
			ctx := context.Background()
			water, err := store.HighWater(ctx, "network")
			if err != nil || water != int64(size) {
				t.Fatal(water, err)
			}
			var after *feed.Cursor
			seen := make(map[string]bool, size)
			var last feed.Source
			for {
				rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", After: after, Limit: 101})
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) == 0 {
					break
				}
				if len(rows) > 101 {
					t.Fatal("unbounded result")
				}
				for _, row := range rows {
					if seen[row.Source.ID] || row.Sequence > water || last.ID != "" && !precedes(last, row.Source) {
						t.Fatalf("duplicate/order/water violation: %+v", row)
					}
					seen[row.Source.ID] = true
					last = row.Source
				}
				p := position(last)
				after = &p
			}
			if len(seen) != size {
				t.Fatalf("walk saw %d, want %d", len(seen), size)
			}
			filtered, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Kinds: []string{"listing"}, Limit: 101})
			if err != nil || len(filtered) != 101 {
				t.Fatal(filtered, err)
			}
			for _, row := range filtered {
				if row.Source.Kind != "listing" {
					t.Fatal(row)
				}
			}
			if rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "absent", Limit: 101}); err != nil || len(rows) != 0 {
				t.Fatal(rows, err)
			}
		})
	}
}

func TestSQLiteHighWaterAndChangesBetweenBatches(t *testing.T) {
	ctx := context.Background()
	store, db, _ := openStore(t)
	seed(t, db, 100, false)
	water, err := store.HighWater(ctx, "network")
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	after := position(first[len(first)-1].Source)
	newest := source(1000)
	if err := store.Upsert(ctx, newest); err != nil {
		t.Fatal(err)
	}
	backdated := source(1001)
	backdated.Created = time.Unix(1_600_000_000, 0).UTC()
	if err := store.Upsert(ctx, backdated); err != nil {
		t.Fatal(err)
	}
	removed := source(80)
	if err := store.Delete(ctx, removed.Network, removed.Kind, removed.ID); err != nil {
		t.Fatal(err)
	}
	rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", After: &after, Limit: 101})
	if err != nil {
		t.Fatal(err)
	}
	old, late := 0, 0
	for _, row := range rows {
		if row.Source.ID == removed.ID || row.Source.ID == newest.ID {
			t.Fatal("removed/newer returned", row)
		}
		if row.Sequence > water {
			late++
		} else {
			old++
		}
	}
	if old != 89 || late != 1 {
		t.Fatalf("snapshot candidates old=%d late=%d; want 89/1", old, late)
	}
}

type queryTrace struct {
	mu    sync.Mutex
	query string
	args  []driver.NamedValue
}
type traceConnector struct {
	uri   string
	trace *queryTrace
}

func (c traceConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := (&sqlite.Driver{}).Open(c.uri)
	if err != nil {
		return nil, err
	}
	return &traceConn{conn, c.trace}, nil
}
func (c traceConnector) Driver() driver.Driver { return &sqlite.Driver{} }

type traceConn struct {
	driver.Conn
	trace *queryTrace
}

func (c *traceConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.trace.mu.Lock()
	c.trace.query = q
	c.trace.args = append([]driver.NamedValue(nil), args...)
	c.trace.mu.Unlock()
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func TestSQLiteQueryPlansSeekUsingBoundedIndexes(t *testing.T) {
	_, db, uri := openStore(t)
	seed(t, db, 100_000, true)
	trace := &queryTrace{}
	tracedDB := sql.OpenDB(traceConnector{uri, trace})
	defer tracedDB.Close()
	store, err := feed.NewSQLStore(tracedDB)
	if err != nil {
		t.Fatal(err)
	}
	pos := position(source(90001))
	pos.Created = time.Unix(1_700_000_000, 0).UTC()
	for _, kinds := range [][]string{nil, {"listing"}} {
		if _, err := store.QueryBatch(context.Background(), feed.StoreQuery{Network: "network", Kinds: kinds, After: &pos, Limit: 101}); err != nil {
			t.Fatal(err)
		}
		trace.mu.Lock()
		statement := trace.query
		args := make([]any, len(trace.args))
		for i, a := range trace.args {
			args[i] = a.Value
		}
		trace.mu.Unlock()
		plan, err := db.Query("EXPLAIN QUERY PLAN "+statement, args...)
		if err != nil {
			t.Fatal(err)
		}
		var details []string
		for plan.Next() {
			var id, parent, unused int
			var detail string
			if err := plan.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			details = append(details, detail)
		}
		if err := plan.Close(); err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(details, "; ")
		t.Log(joined)
		if !strings.Contains(joined, "SEARCH") || !strings.Contains(joined, "order_time") || strings.Contains(joined, "TEMP B-TREE") || strings.Contains(joined, "SCAN ") {
			t.Fatalf("unbounded query plan: %s", joined)
		}
		want := "fnet204_sources_all_order"
		if len(kinds) == 1 {
			want = "fnet204_sources_kind_order"
		}
		if !strings.Contains(joined, want) {
			t.Fatalf("wrong index: %s", joined)
		}
	}
}

func TestSQLiteConcurrentReadersAndUpdates(t *testing.T) {
	store, db, _ := openStore(t)
	seed(t, db, 1000, false)
	ctx := context.Background()
	var wg sync.WaitGroup
	failures := make(chan error, 4)
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if worker == 0 {
					s := source(i)
					s.Active = i%2 == 0
					if err := store.Upsert(ctx, s); err != nil {
						failures <- err
						return
					}
				}
				rows, err := store.QueryBatch(ctx, feed.StoreQuery{Network: "network", Limit: 101})
				if err != nil {
					failures <- err
					return
				}
				for j, row := range rows {
					if !row.Source.Active || !row.Source.Visible || j > 0 && !precedes(rows[j-1].Source, row.Source) {
						failures <- fmt.Errorf("bad concurrent batch")
						return
					}
				}
			}
		}(worker)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.QueryBatch(cancelled, feed.StoreQuery{Network: "network", Limit: 101}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query = %v", err)
	}
}

// BenchmarkSQLStorePage reports actual SQLite query latency, Go heap bytes and
// allocations per bounded page at the required history sizes. Fixture setup is
// excluded. Deep equal-timestamp seeks exercise the worst tie-group boundary.
func BenchmarkSQLStorePage(b *testing.B) {
	for _, size := range []int{10_000, 100_000} {
		for _, tied := range []bool{false, true} {
			name := fmt.Sprintf("events_%d/tied_%t", size, tied)
			b.Run(name, func(b *testing.B) {
				store, db, _ := openStore(b)
				seed(b, db, size, tied)
				p := position(source(size / 2))
				if tied {
					p.Created = time.Unix(1_700_000_000, 0).UTC()
					p.Kind = "listing"
				}
				request := feed.StoreQuery{Network: "network", After: &p, Limit: 101}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					rows, err := store.QueryBatch(context.Background(), request)
					if err != nil || len(rows) != 101 {
						b.Fatalf("page = %d, %v", len(rows), err)
					}
				}
			})
		}
	}
}

func TestSQLitePageHeapBounds(t *testing.T) {
	for _, size := range []int{10_000, 100_000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			store, db, _ := openStore(t)
			seed(t, db, size, false)
			request := feed.StoreQuery{Network: "network", Limit: 101}
			for i := 0; i < 10; i++ {
				if _, err := store.QueryBatch(context.Background(), request); err != nil {
					t.Fatal(err)
				}
			}
			runtime.GC()
			residentBefore := residentBytes()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			for i := 0; i < 100; i++ {
				if _, err := store.QueryBatch(context.Background(), request); err != nil {
					t.Fatal(err)
				}
			}
			elapsed := time.Since(start)
			runtime.ReadMemStats(&after)
			residentAfter := residentBytes()
			allocated := (after.TotalAlloc - before.TotalAlloc) / 100
			allocations := (after.Mallocs - before.Mallocs) / 100
			t.Logf("events=%d rows/page=101 latency=%s allocated_bytes/page=%d allocations/page=%d heap_live_before=%d heap_live_after=%d resident_before=%d resident_after=%d", size, elapsed/100, allocated, allocations, before.HeapAlloc, after.HeapAlloc, residentBefore, residentAfter)
			if allocated > 512*1024 || allocations > 10_000 {
				t.Fatalf("unbounded page allocation: %d bytes, %d allocs", allocated, allocations)
			}
		})
	}
}

// Resident bytes include SQLite's native allocations as well as the Go heap.
// Linux supplies this diagnostic; zero means unavailable on another platform.
func residentBytes() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "VmRSS:" {
			kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				return kilobytes * 1024
			}
		}
	}
	return 0
}
