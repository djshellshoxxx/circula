package feed

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// expectedProjection deliberately does not call the implementation's ordering,
// filtering, deduplication or cursor helpers. It is an oracle for synthetic,
// internally consistent snapshots only.
func expectedProjection(q Query, rows []Source, allowed func(Source) bool) []Source {
	items := make([]Source, 0)
	seen := make(map[[2]string]bool)
	for _, s := range rows {
		if s.Network != q.Network || !s.Visible || !s.Active || !allowed(s) {
			continue
		}
		selected := s.Kind == "listing" || s.Kind == "announcement"
		if len(q.Filter.Kinds) != 0 {
			selected = false
			for _, kind := range q.Filter.Kinds {
				selected = selected || s.Kind == kind
			}
		}
		key := [2]string{s.Kind, s.ID}
		if !selected || seen[key] {
			continue
		}
		seen[key] = true
		items = append(items, s)
	}
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if !left.Created.Equal(right.Created) {
			return left.Created.After(right.Created)
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.ID < right.ID
	})
	return items
}

func generatedSnapshot(seed uint64) []Source {
	rng := rand.New(rand.NewSource(int64(seed)))
	rows := make([]Source, 0, 300)
	for i, count := 0, rng.Intn(151); i < count; i++ {
		kind := []string{"listing", "announcement", "transaction"}[rng.Intn(3)]
		s := Source{
			Network: "n", Kind: kind, ID: fmt.Sprintf("item-%03d-λ", i),
			Member:  fmt.Sprintf("member-%d", rng.Intn(7)),
			Created: time.Unix(100+int64(rng.Intn(5)), int64(rng.Intn(3))).UTC(),
			Visible: rng.Intn(5) != 0, Active: rng.Intn(5) != 0,
		}
		if rng.Intn(5) == 0 {
			s.Network = "other-network"
		}
		rows = append(rows, s)
		if rng.Intn(3) == 0 {
			rows = append(rows, s)
		}
	}
	rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
	return rows
}

func snapshotWalk(codec *CursorCodec, q Query, rows []Source, allowed func(Source) bool) ([]Source, error) {
	result := make([]Source, 0)
	seenTokens := make(map[string]bool)
	for page := 0; page <= len(rows)+1; page++ {
		part, err := Page(codec, q, rows, allowed)
		if err != nil {
			return nil, err
		}
		if part.Items == nil || len(part.Items) > q.Limit {
			return nil, fmt.Errorf("invalid page length: %d (limit %d), nil=%v", len(part.Items), q.Limit, part.Items == nil)
		}
		result = append(result, part.Items...)
		if part.Next == "" {
			return result, nil
		}
		if len(part.Items) != q.Limit || seenTokens[part.Next] {
			return nil, fmt.Errorf("non-progressing or underfilled nonterminal page")
		}
		seenTokens[part.Next] = true
		q.Cursor = part.Next
		// Page size is intentionally changed throughout one walk. It is not
		// part of the filter binding and must not skip or duplicate an item.
		q.Limit = q.Limit%17 + 1
	}
	return nil, fmt.Errorf("pagination exceeded its finite source count")
}

func TestPageProjectionProperty(t *testing.T) {
	codec, _ := testCodec(t)
	var failure string
	check := func(seed uint64, page uint8) bool {
		rows := generatedSnapshot(seed)
		before := append([]Source(nil), rows...)
		q := testQuery(int(page%31) + 1)
		q.Filter.PreferenceRevision = "deterministic-preferences"
		switch seed % 4 {
		case 1:
			q.Filter.Kinds = []string{"listing", "listing"}
		case 2:
			q.Filter.Kinds = []string{"announcement"}
		case 3:
			q.Filter.Kinds = []string{"listing", "announcement", "listing"}
		}
		allowed := func(s Source) bool { return s.Member != "member-3" }
		want := expectedProjection(q, rows, allowed)
		got, err := snapshotWalk(codec, q, rows, allowed)
		if err != nil || !reflect.DeepEqual(got, want) {
			failure = fmt.Sprintf("seed %d: error %v; got %v; want %v", seed, err, got, want)
			return false
		}
		if len(rows) > 0 && !reflect.DeepEqual(rows, before) {
			failure = fmt.Sprintf("seed %d: paging mutated adapter input", seed)
			return false
		}
		for left, right := 0, len(rows)-1; left < right; left, right = left+1, right-1 {
			rows[left], rows[right] = rows[right], rows[left]
		}
		permuted, err := snapshotWalk(codec, q, rows, allowed)
		if err != nil || !reflect.DeepEqual(permuted, want) {
			failure = fmt.Sprintf("seed %d: row permutation changed projection: %v", seed, err)
			return false
		}
		return true
	}
	if err := quick.Check(check, &quick.Config{MaxCount: 300, Rand: rand.New(rand.NewSource(204))}); err != nil {
		t.Fatalf("%v\n%s", err, failure)
	}
}

func TestPageConcurrentIndependentViewers(t *testing.T) {
	codec, _ := testCodec(t)
	rows := generatedSnapshot(123456)
	before := append([]Source(nil), rows...)
	const workers, repeats = 16, 40
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			q := testQuery(worker%7 + 1)
			q.Viewer = fmt.Sprintf("viewer-%02d", worker)
			blocked := fmt.Sprintf("member-%d", worker%7)
			allowed := func(s Source) bool { return s.Member != blocked }
			want := expectedProjection(q, rows, allowed)
			for repeat := 0; repeat < repeats; repeat++ {
				got, err := snapshotWalk(codec, q, rows, allowed)
				if err != nil || !reflect.DeepEqual(got, want) {
					errors <- fmt.Errorf("viewer %d iteration %d: inconsistent result: %v", worker, repeat, err)
					return
				}
				page, err := Page(codec, q, rows, allowed)
				if err != nil {
					errors <- err
					return
				}
				if page.Next != "" {
					other := q
					other.Viewer += "-other"
					other.Cursor = page.Next
					result, err := Page(codec, other, rows, allowAll)
					if err != ErrInvalidCursor || result.Items != nil || result.Next != "" {
						errors <- fmt.Errorf("viewer %d: cross-viewer replay returned %+v, %v", worker, result, err)
						return
					}
				}
			}
		}(worker)
	}
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if !reflect.DeepEqual(rows, before) {
		t.Fatal("concurrent paging mutated shared adapter input")
	}
}

func TestPageRevocationAndNewHeadProperty(t *testing.T) {
	codec, _ := testCodec(t)
	for seed := uint64(0); seed < 100; seed++ {
		rows := generatedSnapshot(seed)
		q := testQuery(3)
		want := expectedProjection(q, rows, allowAll)
		first, err := Page(codec, q, rows, allowAll)
		if err != nil {
			t.Fatal(err)
		}
		if first.Next == "" {
			continue
		}
		// A new, strictly newer head must not shift an existing keyset page.
		rows = append(rows, Source{"n", "listing", "new-head", "member", time.Unix(1_000, 0), true, true})
		remaining := make([]Source, 0)
		revoked := make(map[[2]string]bool)
		for i, item := range want[3:] {
			if i%3 == 0 {
				revoked[[2]string{item.Kind, item.ID}] = true
			} else {
				remaining = append(remaining, item)
			}
		}
		// The row that anchors the cursor may disappear as well.
		revoked[[2]string{first.Items[2].Kind, first.Items[2].ID}] = true
		q.Cursor = first.Next
		got, err := snapshotWalk(codec, q, rows, func(s Source) bool { return !revoked[[2]string{s.Kind, s.ID}] })
		if err != nil || !reflect.DeepEqual(got, remaining) {
			t.Fatalf("seed %d: head insert/revocation changed remaining order: %v\ngot %v\nwant %v", seed, err, got, remaining)
		}
	}
}

func FuzzPagePermutation(f *testing.F) {
	codec, _ := testCodec(f)
	for _, seed := range []uint64{0, 1, 3, 9, 204, 123456, 1<<63 - 1} {
		f.Add(seed, uint8(seed%100))
	}
	f.Fuzz(func(t *testing.T, seed uint64, limit uint8) {
		rows := generatedSnapshot(seed)
		q := testQuery(int(limit%100) + 1)
		want := expectedProjection(q, rows, allowAll)
		got, err := snapshotWalk(codec, q, rows, allowAll)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("projection differs from oracle: %v\ngot %v\nwant %v", err, got, want)
		}
	})
}
