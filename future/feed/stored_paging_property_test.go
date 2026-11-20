package feed

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// propertyStore is intentionally a test-only, in-memory oracle. Production
// bounds and indexes are tested against SQLite separately. It returns current
// candidate rows without filtering their insertion sequence, matching Store's
// bounded-candidate contract rather than pretending a snapshot is immutable.
type propertyStore struct {
	mu            sync.Mutex
	rows          []StoredSource
	high          int64
	highCalls     int
	batchCalls    int
	returned      int
	requested     []int
	batchOverride func(StoreQuery) ([]StoredSource, error)
}

func (s *propertyStore) HighWater(ctx context.Context, network string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.highCalls++
	return s.high, nil
}

func (s *propertyStore) QueryBatch(ctx context.Context, q StoreQuery) ([]StoredSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batchCalls++
	s.requested = append(s.requested, q.Limit)
	if s.batchOverride != nil {
		rows, err := s.batchOverride(q)
		s.returned += len(rows)
		return rows, err
	}
	rows := make([]StoredSource, 0)
	for _, item := range s.rows {
		source := item.Source
		if source.Network != q.Network || !source.Visible || !source.Active {
			continue
		}
		kindMatches := false
		for _, kind := range q.Kinds {
			kindMatches = kindMatches || source.Kind == kind
		}
		if !kindMatches || (q.After != nil && !sourceFollows(source, *q.After)) {
			continue
		}
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i].Source, rows[j].Source
		if !left.Created.Equal(right.Created) {
			return left.Created.After(right.Created)
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.ID < right.ID
	})
	if len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}
	s.returned += len(rows)
	return rows, nil
}

func sourceFollows(source Source, after Cursor) bool {
	if !source.Created.Equal(after.Created) {
		return source.Created.Before(after.Created)
	}
	if source.Kind != after.Kind {
		return source.Kind > after.Kind
	}
	return source.ID > after.ID
}

func storeFixture(n int) *propertyStore {
	s := &propertyStore{high: int64(n)}
	for i := 0; i < n; i++ {
		s.rows = append(s.rows, StoredSource{
			Source: Source{
				Network: "n", Kind: []string{"announcement", "listing"}[i%2],
				ID: fmt.Sprintf("source-%06d", i), Member: "member",
				Created: time.Unix(1_700_000_000-int64(i/6), int64(i%2)).UTC(),
				Visible: true, Active: true,
			},
			Sequence: int64(i + 1),
		})
	}
	return s
}

func permitStored(context.Context, Source) (bool, error) { return true, nil }

func storedWalk(t *testing.T, codec *CursorCodec, q Query, store *propertyStore, allowed func(context.Context, Source) (bool, error)) []Source {
	t.Helper()
	result := make([]Source, 0)
	seen := make(map[[2]string]bool)
	for page := 0; page < len(store.rows)+10; page++ {
		previousReads := store.returned
		previousCalls := store.batchCalls
		part, err := PageFromStore(context.Background(), codec, q, store, allowed)
		if err != nil {
			t.Fatal(err)
		}
		if len(part.Items) > q.Limit || part.Items == nil {
			t.Fatalf("invalid page: %+v", part)
		}
		if count := store.returned - previousReads; count > 1000 {
			t.Fatalf("page read %d candidates, exceeds 1000-candidate budget", count)
		}
		if calls := store.batchCalls - previousCalls; calls > 11 {
			t.Fatalf("page queried %d batches, exceeds finite query budget", calls)
		}
		for _, source := range part.Items {
			key := [2]string{source.Kind, source.ID}
			if seen[key] {
				t.Fatalf("source emitted twice: %v", key)
			}
			seen[key] = true
			result = append(result, source)
		}
		if part.Next == "" {
			for _, requested := range store.requested {
				if requested < 1 || requested > 101 {
					t.Fatalf("unbounded candidate request: %d", requested)
				}
			}
			return result
		}
		q.Cursor = part.Next
		q.Limit = q.Limit%19 + 1
	}
	t.Fatal("store pagination failed to terminate")
	return nil
}

func TestStoredPaginationInsertionFenceAndLiveRevocation(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(180)
	q := testQuery(7)
	all := make([]Source, 0, len(store.rows))
	for _, row := range store.rows {
		all = append(all, row.Source)
	}
	ordered := expectedProjection(q, all, allowAll)
	first, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || !reflect.DeepEqual(first.Items, ordered[:7]) || first.Next == "" {
		t.Fatalf("first page: %+v, %v", first, err)
	}
	removed := make(map[[2]string]bool)
	for _, i := range []int{6, 8, 51, 90, 150} {
		source := ordered[i]
		removed[[2]string{source.Kind, source.ID}] = true
	}
	for i := range store.rows {
		source := &store.rows[i].Source
		if removed[[2]string{source.Kind, source.ID}] {
			source.Visible = false
		}
	}
	// A late insertion older than the cursor must still be excluded from
	// this traversal. A first-page-only timestamp fence cannot ensure this.
	for i, at := range []time.Time{time.Unix(1_700_000_100, 0), ordered[40].Created, time.Unix(1, 0)} {
		store.high++
		store.rows = append(store.rows, StoredSource{
			Source:   Source{"n", "listing", fmt.Sprintf("new-insert-%d", i), "member", at, true, true},
			Sequence: store.high,
		})
	}
	q.Cursor = first.Next
	want := make([]Source, 0)
	for _, source := range ordered[7:] {
		if !removed[[2]string{source.Kind, source.ID}] {
			want = append(want, source)
		}
	}
	got := storedWalk(t, codec, q, store, permitStored)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("insert/deletion changed snapshot traversal\ngot %v\nwant %v", got, want)
	}
	if store.highCalls != 1 {
		t.Fatalf("snapshot replaced during traversal: %d high-water reads", store.highCalls)
	}
}

func TestStoredPaginationDeniedHistoryKeepsFiniteProgress(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(3005)
	q := testQuery(2)
	all := make([]Source, 0, len(store.rows))
	for _, row := range store.rows {
		all = append(all, row.Source)
	}
	ordered := expectedProjection(q, all, allowAll)
	permitted := map[[2]string]bool{
		{ordered[1205].Kind, ordered[1205].ID}: true,
		{ordered[2405].Kind, ordered[2405].ID}: true,
		{ordered[3001].Kind, ordered[3001].ID}: true,
	}
	authorizations := 0
	allowed := func(_ context.Context, source Source) (bool, error) {
		authorizations++
		return permitted[[2]string{source.Kind, source.ID}], nil
	}
	first, err := PageFromStore(context.Background(), codec, q, store, allowed)
	if err != nil || len(first.Items) != 0 || first.Next == "" {
		t.Fatalf("denied prefix must stop with a continuation: %+v, %v", first, err)
	}
	if store.returned > 1000 || authorizations > 1000 {
		t.Fatalf("denied prefix exceeded work bound: %d rows, %d authorization checks", store.returned, authorizations)
	}
	q.Cursor = first.Next
	got := storedWalk(t, codec, q, store, allowed)
	want := []Source{ordered[1205], ordered[2405], ordered[3001]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bounded filtering skipped or duplicated an allowed row\ngot %v\nwant %v", got, want)
	}
}

func TestStoredPaginationRejectsMalformedAdapters(t *testing.T) {
	codec, _ := testCodec(t)
	valid := storeFixture(2).rows
	sort.Slice(valid, func(i, j int) bool { return valid[i].Source.Created.After(valid[j].Source.Created) })
	for name, mutate := range map[string]func([]StoredSource) []StoredSource{
		"wrong network":      func(rows []StoredSource) []StoredSource { rows[0].Source.Network = "other"; return rows },
		"private kind":       func(rows []StoredSource) []StoredSource { rows[0].Source.Kind = "transaction"; return rows },
		"empty identifier":   func(rows []StoredSource) []StoredSource { rows[0].Source.ID = ""; return rows },
		"zero sequence":      func(rows []StoredSource) []StoredSource { rows[0].Sequence = 0; return rows },
		"negative sequence":  func(rows []StoredSource) []StoredSource { rows[0].Sequence = -1; return rows },
		"zero creation":      func(rows []StoredSource) []StoredSource { rows[0].Source.Created = time.Time{}; return rows },
		"duplicate position": func(rows []StoredSource) []StoredSource { return []StoredSource{rows[0], rows[0]} },
		"duplicate identity with different time": func(rows []StoredSource) []StoredSource {
			rows[1].Source.Kind, rows[1].Source.ID = rows[0].Source.Kind, rows[0].Source.ID
			rows[1].Source.Created = rows[0].Source.Created.Add(-time.Nanosecond)
			return rows
		},
		"duplicate insertion sequence": func(rows []StoredSource) []StoredSource {
			rows[1].Sequence = rows[0].Sequence
			return rows
		},
		"oversized owner": func(rows []StoredSource) []StoredSource {
			rows[0].Source.Member = strings.Repeat("m", 257)
			return rows
		},
		"out of order": func(rows []StoredSource) []StoredSource {
			sort.Slice(rows, func(i, j int) bool { return rows[i].Source.Created.Before(rows[j].Source.Created) })
			return rows
		},
	} {
		t.Run(name, func(t *testing.T) {
			rows := mutate(append([]StoredSource(nil), valid...))
			store := &propertyStore{high: 2, batchOverride: func(StoreQuery) ([]StoredSource, error) { return rows, nil }}
			result, err := PageFromStore(context.Background(), codec, testQuery(2), store, permitStored)
			if err == nil || result.Items != nil || result.Next != "" {
				t.Fatalf("malformed store returned result: %+v, %v", result, err)
			}
		})
	}
	store := &propertyStore{high: 2, batchOverride: func(q StoreQuery) ([]StoredSource, error) {
		return make([]StoredSource, q.Limit+1), nil
	}}
	if result, err := PageFromStore(context.Background(), codec, testQuery(1), store, permitStored); err == nil || result.Items != nil || result.Next != "" {
		t.Fatalf("oversized store batch returned %+v, %v", result, err)
	}
}

func TestStoredPaginationRejectsStalledAdapter(t *testing.T) {
	codec, _ := testCodec(t)
	rows := storeFixture(2).rows
	sort.Slice(rows, func(i, j int) bool { return rows[i].Source.Created.After(rows[j].Source.Created) })
	store := &propertyStore{high: 2, batchOverride: func(StoreQuery) ([]StoredSource, error) { return rows, nil }}
	q := testQuery(1)
	first, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || first.Next == "" {
		t.Fatalf("initial progress fixture: %+v, %v", first, err)
	}
	q.Cursor = first.Next
	result, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != ErrInvalidSource || result.Items != nil || result.Next != "" {
		t.Fatalf("store ignored exclusive keyset without rejection: %+v, %v", result, err)
	}
}

func TestStoredPaginationNewInsertionsNeedNoAuthorization(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(3)
	q := testQuery(1)
	first, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || first.Next == "" {
		t.Fatalf("initial snapshot fixture: %+v, %v", first, err)
	}
	// Remove all remaining snapshot rows, then fill the continuation region
	// with new insertions. They are outside the fence even if the viewer
	// could otherwise see them. No source authorization read is necessary.
	store.rows = nil
	for i := 0; i < 1200; i++ {
		store.rows = append(store.rows, StoredSource{
			Source:   Source{"n", "listing", fmt.Sprintf("later-%06d", i), "member", time.Unix(100-int64(i/20), 0), true, true},
			Sequence: int64(4 + i),
		})
	}
	store.high = 1203
	q.Cursor = first.Next
	got := storedWalk(t, codec, q, store, func(context.Context, Source) (bool, error) {
		t.Error("post-snapshot insertion reached authorization")
		return true, nil
	})
	if len(got) != 0 {
		t.Fatalf("post-snapshot insertions appeared: %v", got)
	}
}

func TestStoredPaginationAuthorizationFailureDiscardsPage(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(5)
	calls := 0
	failure := errors.New("authorization database unavailable: private fixture detail")
	result, err := PageFromStore(context.Background(), codec, testQuery(5), store, func(context.Context, Source) (bool, error) {
		calls++
		if calls == 2 {
			return false, failure
		}
		return true, nil
	})
	if err != ErrAuthorization || result.Items != nil || result.Next != "" {
		t.Fatalf("authorization failure returned a partial page: %+v, %v", result, err)
	}
}

func TestStoredPaginationErrorsAndCancellation(t *testing.T) {
	codec, _ := testCodec(t)
	t.Run("store error hides private diagnostics", func(t *testing.T) {
		store := &propertyStore{high: 1, batchOverride: func(StoreQuery) ([]StoredSource, error) {
			return nil, errors.New("private schema and network diagnostic")
		}}
		result, err := PageFromStore(context.Background(), codec, testQuery(1), store, permitStored)
		if err != ErrStore || result.Items != nil || result.Next != "" {
			t.Fatalf("store error contract: %+v, %v", result, err)
		}
	})
	t.Run("already cancelled request performs no reads", func(t *testing.T) {
		store := storeFixture(3)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := PageFromStore(ctx, codec, testQuery(1), store, permitStored)
		if err != context.Canceled || result.Items != nil || result.Next != "" || store.highCalls != 0 || store.batchCalls != 0 {
			t.Fatalf("cancelled request performed work: %+v, %v, %d/%d reads", result, err, store.highCalls, store.batchCalls)
		}
	})
	t.Run("authorization cancellation discards collected items", func(t *testing.T) {
		store := storeFixture(3)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		result, err := PageFromStore(ctx, codec, testQuery(3), store, func(context.Context, Source) (bool, error) {
			calls++
			if calls == 2 {
				cancel()
			}
			return true, nil
		})
		if err != context.Canceled || result.Items != nil || result.Next != "" {
			t.Fatalf("cancelled authorization returned a partial page: %+v, %v", result, err)
		}
	})
}

func TestStoredPaginationExpiryRejectsBeforeStoreRead(t *testing.T) {
	codec, clock := testCodec(t)
	store := storeFixture(5)
	q := testQuery(1)
	first, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || first.Next == "" {
		t.Fatalf("first page: %+v, %v", first, err)
	}
	beforeHigh, beforeBatch := store.highCalls, store.batchCalls
	clock.Set(testNow.Add(DefaultCursorTTL))
	q.Cursor = first.Next
	result, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != ErrInvalidCursor || result.Items != nil || result.Next != "" {
		t.Fatalf("expired traversal returned %+v, %v", result, err)
	}
	if store.highCalls != beforeHigh || store.batchCalls != beforeBatch {
		t.Fatal("expired cursor triggered database work")
	}
}

func TestStoredPaginationContinuationDoesNotRenewLifetime(t *testing.T) {
	codec, clock := testCodec(t)
	store := storeFixture(12)
	q := testQuery(1)
	first, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || first.Next == "" {
		t.Fatalf("first page: %+v, %v", first, err)
	}
	clock.Set(testNow.Add(DefaultCursorTTL - time.Second))
	q.Cursor = first.Next
	second, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != nil || second.Next == "" {
		t.Fatalf("continuation immediately before expiry: %+v, %v", second, err)
	}
	state, err := codec.OpenSnapshot(Binding{Network: q.Network, Viewer: q.Viewer, Filter: q.Filter}, second.Next)
	if err != nil || !state.IssuedAt.Equal(testNow) {
		t.Fatalf("continuation reset original issue time: %+v, %v", state, err)
	}
	clock.Set(testNow.Add(DefaultCursorTTL))
	q.Cursor = second.Next
	result, err := PageFromStore(context.Background(), codec, q, store, permitStored)
	if err != ErrInvalidCursor || result.Items != nil || result.Next != "" {
		t.Fatalf("continued traversal outlived original TTL: %+v, %v", result, err)
	}
}

func TestStoredPaginationPrivateFrontierIsEncrypted(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(1005)
	for i := range store.rows {
		store.rows[i].Source.ID = fmt.Sprintf("private-unread-source-%06d", i)
	}
	q := testQuery(1)
	result, err := PageFromStore(context.Background(), codec, q, store, func(context.Context, Source) (bool, error) { return false, nil })
	if err != nil || len(result.Items) != 0 || result.Next == "" {
		t.Fatalf("private continuation fixture: %+v, %v", result, err)
	}
	for _, segment := range strings.Split(result.Next, ".") {
		decoded, _ := base64.RawURLEncoding.DecodeString(segment)
		if strings.Contains(segment, "private-unread-source") || bytes.Contains(decoded, []byte("private-unread-source")) || bytes.Contains(decoded, []byte(`"iat"`)) {
			t.Fatalf("private scan frontier exposes plaintext: %q", segment)
		}
	}
	position, err := codec.OpenSnapshot(Binding{Network: q.Network, Viewer: q.Viewer, Filter: q.Filter}, result.Next)
	if err != nil || !strings.HasPrefix(position.Position.ID, "private-unread-source-") {
		t.Fatalf("test did not exercise a hidden position: %+v, %v", position, err)
	}
}

func TestStoredPaginationConcurrentViewers(t *testing.T) {
	codec, _ := testCodec(t)
	store := storeFixture(95)
	all := make([]Source, 0, len(store.rows))
	for _, row := range store.rows {
		all = append(all, row.Source)
	}
	const workers = 12
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			q := testQuery(worker%13 + 1)
			q.Viewer = fmt.Sprintf("parallel-viewer-%d", worker)
			q.Filter.Kinds = []string{[]string{"announcement", "listing"}[worker%2]}
			want := expectedProjection(q, all, allowAll)
			got := make([]Source, 0)
			for pages := 0; ; pages++ {
				if pages > len(all) {
					errors <- fmt.Errorf("viewer %d failed to terminate", worker)
					return
				}
				part, err := PageFromStore(context.Background(), codec, q, store, permitStored)
				if err != nil {
					errors <- fmt.Errorf("viewer %d: %w", worker, err)
					return
				}
				got = append(got, part.Items...)
				if part.Next == "" {
					break
				}
				other := q
				other.Viewer += "-other"
				other.Cursor = part.Next
				result, err := PageFromStore(context.Background(), codec, other, store, permitStored)
				if err != ErrInvalidCursor || result.Items != nil || result.Next != "" {
					errors <- fmt.Errorf("viewer %d: stored cursor replay accepted", worker)
					return
				}
				q.Cursor = part.Next
			}
			if !reflect.DeepEqual(got, want) {
				errors <- fmt.Errorf("viewer %d: concurrent projection changed", worker)
			}
		}(worker)
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
