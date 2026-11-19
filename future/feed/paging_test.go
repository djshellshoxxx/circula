package feed

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"
)

// fixtureStore is a reader-contract fixture. A separate integration module
// tests the production SQLStore against SQLite.
type fixtureStore struct {
	rows                      []StoredSource
	high                      int64
	queries, fetched, anchors int
	err                       error
	mutate                    func([]StoredSource) []StoredSource
}

func (s *fixtureStore) HighWater(context.Context, string) (int64, error) {
	s.anchors++
	return s.high, s.err
}
func (s *fixtureStore) QueryBatch(_ context.Context, q StoreQuery) ([]StoredSource, error) {
	s.queries++
	if s.err != nil {
		return nil, s.err
	}
	var out []StoredSource
	for _, row := range s.rows {
		p := positionOf(row.Source)
		if row.Source.Network != q.Network || !hasKind(q.Kinds, row.Source.Kind) || !row.Source.Visible || !row.Source.Active {
			continue
		}
		if q.After != nil && !precedes(*q.After, p) {
			continue
		}
		out = append(out, row)
		if len(out) == q.Limit {
			break
		}
	}
	if s.mutate != nil {
		out = s.mutate(out)
	}
	s.fetched += len(out)
	return out, nil
}
func basicStoreFixture(n int) *fixtureStore {
	s := &fixtureStore{high: int64(n)}
	for i := 0; i < n; i++ {
		s.rows = append(s.rows, StoredSource{Source: Source{"n", "listing", fmt.Sprintf("id-%06d", i), "owner", time.Unix(1700000000+int64(n-i)/3, 0), true, true}, Sequence: int64(i + 1)})
	}
	sort.Slice(s.rows, func(i, j int) bool { return precedes(positionOf(s.rows[i].Source), positionOf(s.rows[j].Source)) })
	return s
}
func allowStored(context.Context, Source) (bool, error) { return true, nil }

func TestPageFromStoreWalkAndSnapshot(t *testing.T) {
	c, _ := testCodec(t)
	s := basicStoreFixture(37)
	q := testQuery(7)
	first, err := PageFromStore(context.Background(), c, q, s, allowStored)
	if err != nil || len(first.Items) != 7 || first.Next == "" {
		t.Fatalf("first: %+v %v", first, err)
	}
	added := StoredSource{Source: Source{"n", "listing", "new-backdated", "owner", s.rows[15].Source.Created, true, true}, Sequence: 38}
	s.rows = append(s.rows, added)
	s.high = 38
	sort.Slice(s.rows, func(i, j int) bool { return precedes(positionOf(s.rows[i].Source), positionOf(s.rows[j].Source)) })
	got := append([]Source(nil), first.Items...)
	q.Cursor = first.Next
	for pages := 0; q.Cursor != ""; pages++ {
		if pages > 10 {
			t.Fatal("traversal did not terminate")
		}
		r, err := PageFromStore(context.Background(), c, q, s, allowStored)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, r.Items...)
		q.Cursor = r.Next
	}
	if len(got) != 37 || s.anchors != 1 {
		t.Fatalf("items=%d anchors=%d", len(got), s.anchors)
	}
	seen := map[string]bool{}
	for i, row := range got {
		if row.ID == added.Source.ID || seen[row.ID] {
			t.Fatalf("new/duplicate row: %+v", row)
		}
		seen[row.ID] = true
		if i > 0 && !precedes(positionOf(got[i-1]), positionOf(row)) {
			t.Fatal("order changed")
		}
	}
}
func TestPageFromStoreBoundedDenials(t *testing.T) {
	c, _ := testCodec(t)
	s := basicStoreFixture(2500)
	q := testQuery(1)
	allowed := func(_ context.Context, row Source) (bool, error) { return row.ID == "id-002200", nil }
	for page := 0; page < 3; page++ {
		before := s.fetched
		r, err := PageFromStore(context.Background(), c, q, s, allowed)
		if err != nil {
			t.Fatal(err)
		}
		if s.fetched-before > MaxPageCandidates {
			t.Fatalf("unbounded query fetched %d", s.fetched-before)
		}
		if page < 2 && (len(r.Items) != 0 || r.Next == "") {
			t.Fatalf("empty bounded page: %+v", r)
		}
		if page == 2 && (len(r.Items) != 1 || r.Items[0].ID != "id-002200" || r.Next != "") {
			t.Fatalf("last page: %+v", r)
		}
		q.Cursor = r.Next
	}
}
func TestPageFromStoreErrorsFailClosed(t *testing.T) {
	c, _ := testCodec(t)
	s := basicStoreFixture(20)
	s.err = errors.New("private SQL detail")
	r, err := PageFromStore(context.Background(), c, testQuery(3), s, allowStored)
	if err != ErrStore || r.Items != nil || r.Next != "" {
		t.Fatalf("store failure: %+v %v", r, err)
	}
	s.err = nil
	n := 0
	allowed := func(context.Context, Source) (bool, error) {
		n++
		if n == 2 {
			return false, errors.New("private auth detail")
		}
		return true, nil
	}
	r, err = PageFromStore(context.Background(), c, testQuery(3), s, allowed)
	if err != ErrAuthorization || r.Items != nil || r.Next != "" {
		t.Fatalf("authorization failure: %+v %v", r, err)
	}
}

type cancelReadStore struct {
	*fixtureStore
	cancel context.CancelFunc
	anchor bool
}

func (s cancelReadStore) HighWater(ctx context.Context, network string) (int64, error) {
	if s.anchor {
		s.cancel()
		return 0, nil
	}
	return s.fixtureStore.HighWater(ctx, network)
}

func (s cancelReadStore) QueryBatch(context.Context, StoreQuery) ([]StoredSource, error) {
	s.cancel()
	return nil, nil
}

func TestPageFromStoreCancellationDuringEmptyRead(t *testing.T) {
	c, _ := testCodec(t)
	for _, anchor := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		s := cancelReadStore{fixtureStore: basicStoreFixture(1), cancel: cancel, anchor: anchor}
		result, err := PageFromStore(ctx, c, testQuery(1), s, allowStored)
		cancel()
		if err != context.Canceled || result.Items != nil || result.Next != "" {
			t.Fatalf("anchor=%t: cancelled read returned %+v, %v", anchor, result, err)
		}
	}
}
