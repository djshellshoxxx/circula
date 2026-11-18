package feed

import (
	"fmt"
	"testing"
	"time"
)

func allowAll(Source) bool { return true }

func testQuery(limit int) Query {
	return Query{Network: "n", Viewer: "member-A", Limit: limit}
}

func TestFeedPrivacyAndDedupe(t *testing.T) {
	c, _ := testCodec(t)
	now := time.Unix(1, 0)
	a := Source{"n", "listing", "1", "m", now, true, true}
	rows := []Source{a, a, {"n", "transaction", "tx", "m", now, true, true}, {"x", "listing", "x", "m", now, true, true}, {"n", "announcement", "2", "m", now, true, true}}
	res, e := Page(c, testQuery(1), rows, allowAll)
	if e != nil || len(res.Items) != 1 || res.Items[0].Kind != "announcement" || res.Next == "" {
		t.Fatal(res, e)
	}
	q := testQuery(10)
	q.Cursor = res.Next
	res, e = Page(c, q, rows, allowAll)
	if e != nil || len(res.Items) != 1 || res.Items[0].ID != "1" || res.Next != "" {
		t.Fatal(res, e)
	}
	res, e = Page(c, testQuery(10), rows, func(Source) bool { return false })
	if e != nil || len(res.Items) != 0 || res.Items == nil || res.Next != "" {
		t.Fatal("revoked", res, e)
	}
}

// feedRows returns n listings and n announcements in network "n", plus rows
// that must never appear: another network, hidden, inactive and trade rows.
func feedRows(n int) []Source {
	var rows []Source
	base := time.Unix(1_700_000_000, 0)
	for i := 0; i < n; i++ {
		at := base.Add(time.Duration(i/3) * time.Second) // ties exercise the kind/ID order
		rows = append(rows,
			Source{"n", "listing", fmt.Sprintf("l%03d", i), "m", at, true, true},
			Source{"n", "announcement", fmt.Sprintf("a%03d", i), "m", at, true, true},
			Source{"other", "listing", fmt.Sprintf("x%03d", i), "m", at, true, true},
			Source{"n", "listing", fmt.Sprintf("h%03d", i), "m", at, false, true},
			Source{"n", "listing", fmt.Sprintf("i%03d", i), "m", at, true, false},
			Source{"n", "transaction", fmt.Sprintf("t%03d", i), "m", at, true, true},
		)
	}
	return rows
}

// walk follows Next until the feed ends and returns every item in order. It
// requires exactly ceil(items/limit) pages: no trailing empty page.
func walk(t *testing.T, c *CursorCodec, q Query, rows []Source) []Source {
	t.Helper()
	var all []Source
	for pages := 1; ; pages++ {
		if pages > 100 {
			t.Fatal("pagination does not terminate")
		}
		res, err := Page(c, q, rows, allowAll)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Items) > q.Limit || (res.Next != "" && len(res.Items) != q.Limit) {
			t.Fatalf("page of %d items with next %q at limit %d", len(res.Items), res.Next, q.Limit)
		}
		all = append(all, res.Items...)
		if res.Next == "" {
			if want := (len(all) + q.Limit - 1) / q.Limit; pages != want && len(all) > 0 {
				t.Fatalf("%d items at limit %d took %d pages, want %d", len(all), q.Limit, pages, want)
			}
			return all
		}
		q.Cursor = res.Next
	}
}

func TestPageWalksWholeFeedWithSignedCursors(t *testing.T) {
	c, _ := testCodec(t)
	rows := feedRows(25)
	whole, err := Page(c, testQuery(100), rows, allowAll)
	if err != nil || len(whole.Items) != 50 || whole.Next != "" {
		t.Fatalf("single page: %d items, next %q, %v", len(whole.Items), whole.Next, err)
	}
	for _, limit := range []int{1, 7, 10, 49, 50} {
		got := walk(t, c, testQuery(limit), rows)
		if len(got) != len(whole.Items) {
			t.Fatalf("limit %d: %d items, want %d", limit, len(got), len(whole.Items))
		}
		for i := range got {
			if got[i] != whole.Items[i] {
				t.Fatalf("limit %d: item %d = %+v, want %+v", limit, i, got[i], whole.Items[i])
			}
			if i > 0 && !precedes(positionOf(got[i-1]), positionOf(got[i])) {
				t.Fatalf("limit %d: items %d and %d out of order", limit, i-1, i)
			}
		}
	}
}

func TestPageKindFilter(t *testing.T) {
	c, _ := testCodec(t)
	q := testQuery(7)
	q.Filter.Kinds = []string{"listing"}
	got := walk(t, c, q, feedRows(20))
	if len(got) != 20 {
		t.Fatalf("%d listings, want 20", len(got))
	}
	for _, s := range got {
		if s.Kind != "listing" || s.Network != "n" || !s.Visible || !s.Active {
			t.Fatalf("filtered feed shows %+v", s)
		}
	}
}

func TestPageCursorBoundToNetworkViewerAndFilter(t *testing.T) {
	c, _ := testCodec(t)
	rows := feedRows(10)
	first, err := Page(c, testQuery(3), rows, allowAll)
	if err != nil || first.Next == "" {
		t.Fatal(first, err)
	}
	for name, mutate := range map[string]func(*Query){
		"member B replays member A's cursor": func(q *Query) { q.Viewer = "member-B" },
		"other network":                      func(q *Query) { q.Network = "other" },
		"filter X cursor under filter Y":     func(q *Query) { q.Filter.Kinds = []string{"announcement"} },
		"preference revision changed":        func(q *Query) { q.Filter.PreferenceRevision = "rev-2" },
		"tampered cursor":                    func(q *Query) { q.Cursor = q.Cursor[:len(q.Cursor)-1] },
		"garbage cursor":                     func(q *Query) { q.Cursor = "garbage" },
	} {
		q := testQuery(3)
		q.Cursor = first.Next
		mutate(&q)
		if res, err := Page(c, q, rows, allowAll); err != ErrInvalidCursor || res.Items != nil || res.Next != "" {
			t.Fatalf("%s: %+v, %v; want ErrInvalidCursor and no items", name, res, err)
		}
	}
	// A limit change is not a filter change: the cursor stays valid.
	q := testQuery(5)
	q.Cursor = first.Next
	if _, err := Page(c, q, rows, allowAll); err != nil {
		t.Fatalf("limit change refused: %v", err)
	}
}

func TestPageExpiredCursor(t *testing.T) {
	c, clk := testCodec(t)
	rows := feedRows(5)
	first, err := Page(c, testQuery(2), rows, allowAll)
	if err != nil || first.Next == "" {
		t.Fatal(first, err)
	}
	q := testQuery(2)
	q.Cursor = first.Next
	clk.Set(testNow.Add(DefaultCursorTTL))
	if _, err := Page(c, q, rows, allowAll); err != ErrInvalidCursor {
		t.Fatalf("expired cursor: %v", err)
	}
	// The first page needs no cursor and still works.
	if _, err := Page(c, testQuery(2), rows, allowAll); err != nil {
		t.Fatal(err)
	}
}

func TestPageRotatedKeyContinuesPaging(t *testing.T) {
	clk := &clock{t: testNow}
	before := mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey1}), 0, clk.Now)
	during := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k1": testKey1, "k2": testKey2}), 0, clk.Now)
	rows := feedRows(5)
	first, err := Page(before, testQuery(3), rows, allowAll)
	if err != nil {
		t.Fatal(err)
	}
	q := testQuery(3)
	q.Cursor = first.Next
	second, err := Page(during, q, rows, allowAll)
	if err != nil || len(second.Items) != 3 || second.Next == "" {
		t.Fatalf("after rotation: %+v %v", second, err)
	}
	if kid, _, _ := splitToken(t, second.Next); kid != "k2" {
		t.Fatalf("next cursor signed with %q, want k2", kid)
	}
}

func TestPageRevocationBetweenPages(t *testing.T) {
	c, _ := testCodec(t)
	rows := feedRows(4)
	first, err := Page(c, testQuery(2), rows, allowAll)
	if err != nil {
		t.Fatal(err)
	}
	q := testQuery(100)
	q.Cursor = first.Next
	revoked := map[string]bool{}
	whole, _ := Page(c, testQuery(100), rows, allowAll)
	for _, s := range whole.Items[2:4] {
		revoked[s.ID] = true
	}
	// Revoking the cursor's own item does not break paging either.
	revoked[first.Items[1].ID] = true
	res, err := Page(c, q, rows, func(s Source) bool { return !revoked[s.ID] })
	if err != nil || len(res.Items) != len(whole.Items)-4 {
		t.Fatalf("%d items, %v; want %d", len(res.Items), err, len(whole.Items)-4)
	}
	for _, s := range res.Items {
		if revoked[s.ID] {
			t.Fatalf("revoked item %s shown", s.ID)
		}
	}
}

func TestPageInvalidQuery(t *testing.T) {
	c, _ := testCodec(t)
	rows := feedRows(2)
	for name, tc := range map[string]struct {
		c       *CursorCodec
		q       Query
		rows    []Source
		allowed func(Source) bool
	}{
		"nil codec":         {nil, testQuery(1), rows, allowAll},
		"nil authorization": {c, testQuery(1), rows, nil},
		"limit 0":           {c, testQuery(0), rows, allowAll},
		"limit 101":         {c, testQuery(101), rows, allowAll},
		"empty network":     {c, Query{Viewer: "v", Limit: 1}, rows, allowAll},
		"empty viewer":      {c, Query{Network: "n", Limit: 1}, rows, allowAll},
		"private kind":      {c, Query{Network: "n", Viewer: "v", Limit: 1, Filter: Filter{Kinds: []string{"transaction"}}}, rows, allowAll},
		"too many rows":     {c, testQuery(1), make([]Source, maxRows+1), allowAll},
	} {
		if res, err := Page(tc.c, tc.q, tc.rows, tc.allowed); err != ErrInvalidQuery || res.Items != nil {
			t.Fatalf("%s: %+v, %v; want ErrInvalidQuery", name, res, err)
		}
	}
	at := time.Unix(5, 0)
	for name, bad := range map[string]Source{
		"empty id":    {"n", "listing", "", "m", at, true, true},
		"zero time":   {"n", "listing", "x", "m", time.Time{}, true, true},
		"long id":     {"n", "listing", string(make([]byte, maxSourceIDLen+1)), "m", at, true, true},
		"non-UTF8 id": {"n", "announcement", "\xff", "m", at, true, true},
	} {
		if _, err := Page(c, testQuery(10), []Source{bad}, allowAll); err != ErrInvalidSource {
			t.Fatalf("%s: %v, want ErrInvalidSource", name, err)
		}
	}
}

// failingSigner verifies normally but cannot sign: Page must fail rather than
// return a page without a usable next cursor.
type failingSigner struct{ *KeyRing }

func (failingSigner) SigningKey() (string, []byte) { return "", nil }

func TestPageFailsClosedWhenSigningFails(t *testing.T) {
	c := mustCodec(t, failingSigner{mustRing(t, "k1", map[string][]byte{"k1": testKey1})}, 0, func() time.Time { return testNow })
	if _, err := Page(c, testQuery(1), feedRows(2), allowAll); err == nil {
		t.Fatal("page returned without a signed next cursor")
	}
	// A last page needs no cursor and still works.
	if res, err := Page(c, testQuery(100), feedRows(2), allowAll); err != nil || res.Next != "" || len(res.Items) != 4 {
		t.Fatalf("%+v %v", res, err)
	}
}
