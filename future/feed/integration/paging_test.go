package feedintegration

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	feed "github.com/djshellshoxxx/circula/future/feed"
)

func codec(tb testing.TB) *feed.CursorCodec {
	tb.Helper()
	keys, err := feed.NewKeyRing("test-key", map[string][]byte{"test-key": []byte("synthetic-qualification-key-00000000000000000000")})
	if err != nil {
		tb.Fatal(err)
	}
	c, err := feed.NewCursorCodec(keys, feed.CodecOptions{Now: func() time.Time { return time.Unix(1_800_000_000, 123).UTC() }})
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

type countedStore struct {
	feed.Store
	rows, calls int
}

func (s *countedStore) QueryBatch(ctx context.Context, q feed.StoreQuery) ([]feed.StoredSource, error) {
	if q.Limit < 1 || q.Limit > feed.MaxStoreBatch {
		return nil, fmt.Errorf("query limit %d", q.Limit)
	}
	rows, err := s.Store.QueryBatch(ctx, q)
	s.rows += len(rows)
	s.calls++
	return rows, err
}
func allow(context.Context, feed.Source) (bool, error) { return true, nil }

func TestSQLiteSnapshotPaginationLargeHistories(t *testing.T) {
	for _, size := range []int{10_000, 100_000} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			store, db, _ := openStore(t)
			seed(t, db, size, false)
			counted := &countedStore{Store: store}
			c := codec(t)
			query := feed.Query{Network: "network", Viewer: "viewer", Limit: 100}
			seen := make(map[string]bool, size)
			var last feed.Source
			pages := 0
			for {
				counted.rows, counted.calls = 0, 0
				result, err := feed.PageFromStore(context.Background(), c, query, counted, allow)
				if err != nil {
					t.Fatal(err)
				}
				if counted.rows > feed.MaxPageCandidates || len(result.Items) > 100 {
					t.Fatalf("page exceeded bounds: candidates=%d items=%d", counted.rows, len(result.Items))
				}
				for _, s := range result.Items {
					if seen[s.ID] || last.ID != "" && !precedes(last, s) {
						t.Fatal("duplicate/out-of-order result", s)
					}
					seen[s.ID] = true
					last = s
				}
				pages++
				if result.Next == "" {
					break
				}
				if pages > size/100+1 {
					t.Fatal("nonterminating traversal")
				}
				query.Cursor = result.Next
			}
			if len(seen) != size || pages != (size+99)/100 {
				t.Fatalf("walk=%d items in %d pages; want %d", len(seen), pages, size)
			}
		})
	}
}

func TestSQLiteBoundedBacklogSparsePermissionsAndDeletions(t *testing.T) {
	ctx := context.Background()
	store, db, _ := openStore(t)
	seed(t, db, 3000, false)
	counted := &countedStore{Store: store}
	c := codec(t)
	query := feed.Query{Network: "network", Viewer: "viewer", Limit: 20}
	sparse := func(_ context.Context, s feed.Source) (bool, error) {
		v, err := strconv.Atoi(strings.TrimPrefix(s.ID, "source-"))
		return err == nil && v%97 == 0, nil
	}
	first, err := feed.PageFromStore(ctx, c, query, counted, sparse)
	if err != nil || len(first.Items) == 0 || first.Next == "" {
		t.Fatal(first, err)
	}
	if counted.rows != feed.MaxPageCandidates {
		t.Fatalf("sparse page read %d candidates, want exact cap", counted.rows)
	}
	state, err := c.OpenSnapshot(feed.Binding{Network: query.Network, Viewer: query.Viewer}, first.Next)
	if err != nil {
		t.Fatal(err)
	}
	// Both an already shown item and the private frontier can disappear. The
	// cursor must continue from its immutable position without reading them.
	lastShown := first.Items[len(first.Items)-1]
	if err := store.Delete(ctx, lastShown.Network, lastShown.Kind, lastShown.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, query.Network, state.Position.Kind, state.Position.ID); err != nil {
		t.Fatal(err)
	}
	removed := source(970)
	if err := store.Delete(ctx, removed.Network, removed.Kind, removed.ID); err != nil {
		t.Fatal(err)
	}
	// A stream of post-snapshot backdated records is read under the cap and
	// skipped after LIMIT. Filtering sequence in SQL could scan all 2,000.
	for i := 0; i < 2000; i++ {
		s := feed.Source{Network: query.Network, Kind: "listing", ID: fmt.Sprintf("late-%06d", i), Member: "member", Created: state.Position.Created.Add(-time.Nanosecond), Visible: true, Active: true}
		if err := store.Upsert(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	query.Cursor = first.Next
	seen := map[string]bool{}
	for _, s := range first.Items {
		seen[s.ID] = true
	}
	partial := 0
	for pages := 0; ; pages++ {
		if pages > 15 {
			t.Fatal("sparse/backlog traversal did not terminate")
		}
		counted.rows, counted.calls = 0, 0
		result, err := feed.PageFromStore(ctx, c, query, counted, sparse)
		if err != nil {
			t.Fatal(err)
		}
		if counted.rows > feed.MaxPageCandidates || counted.calls > 10 {
			t.Fatalf("work cap exceeded: %d rows, %d queries", counted.rows, counted.calls)
		}
		if result.Next != "" && len(result.Items) < query.Limit {
			partial++
		}
		for _, s := range result.Items {
			if seen[s.ID] || strings.HasPrefix(s.ID, "late-") || s.ID == removed.ID {
				t.Fatal("duplicate/new/deleted item", s)
			}
			seen[s.ID] = true
		}
		if result.Next == "" {
			break
		}
		query.Cursor = result.Next
	}
	if partial < 2 {
		t.Fatalf("expected bounded continuations through backlog, got %d", partial)
	}
	// 31 original multiples of 97 exist, with 970 deleted before it is read.
	if len(seen) != 30 {
		t.Fatalf("permitted stable sources seen=%d, want 30", len(seen))
	}
}

func BenchmarkSQLStorePageFromStore(b *testing.B) {
	for _, size := range []int{10_000, 100_000} {
		for _, tied := range []bool{false, true} {
			b.Run(fmt.Sprintf("events_%d/tied_%t", size, tied), func(b *testing.B) {
				store, db, _ := openStore(b)
				seed(b, db, size, tied)
				c := codec(b)
				p := position(source(size / 2))
				if tied {
					p.Created = time.Unix(1_700_000_000, 0).UTC()
					p.Kind = "listing"
				}
				query := feed.Query{Network: "network", Viewer: "viewer", Limit: 100}
				token, err := c.SealSnapshot(feed.Binding{Network: query.Network, Viewer: query.Viewer}, feed.SnapshotCursor{Position: p, HighWater: int64(size), IssuedAt: time.Unix(1_800_000_000, 123).UTC()})
				if err != nil {
					b.Fatal(err)
				}
				query.Cursor = token
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					result, err := feed.PageFromStore(context.Background(), c, query, store, allow)
					if err != nil || len(result.Items) != 100 || result.Next == "" {
						b.Fatalf("page = %d next=%t %v", len(result.Items), result.Next != "", err)
					}
				}
			})
		}
	}
}
