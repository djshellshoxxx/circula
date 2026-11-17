package discovery_test

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

func TestTextSearchFoldingAndInjection(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Café table", "solid oak")
	e.listing("n1", "L2", "alice", "Chair", "Das Stühle große Auswahl")
	e.listing("n1", "L3", "alice", "100% cotton_shirt", "")
	for q, want := range map[string]int{
		"cafe": 1, "CAFÉ": 1, "stuhle": 1, "GROSSE": 1, "oak solid": 1, "table oak": 1,
		"100%": 1, "cotton_shirt": 1, "%": 1, "_": 1, "zzz": 0,
		"' OR 1=1 --": 0, "\"; DROP TABLE listing_media; --": 0, "L1": 0,
	} {
		r, err := e.svc.Search(ctx, bob, d.Filters{Q: q}, "")
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		if len(r.Items) != want {
			t.Errorf("q=%q got %v want %d", q, ids(r), want)
		}
	}
	// tables survived the injection attempt
	if _, err := e.svc.ListCategories(ctx, bob); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Search(ctx, bob, d.Filters{Q: strings.Repeat("a", 201)}, ""); !errors.Is(err, d.ErrInvalid) {
		t.Fatal("long q accepted")
	}
	// terms are capped, not failed
	r, err := e.svc.Search(ctx, bob, d.Filters{Q: "a b c d e f g h i j k"}, "")
	if err != nil || len(r.Items) != 0 {
		t.Fatal(err, r.Items)
	}
}

func TestRelevanceAndRecentSorting(t *testing.T) {
	e := newEnv(t)
	now := e.clock.Now()
	e.listingAt("n1", "desc", "alice", "Garden", "a bike repair kit", d.StatusActive, now)
	e.listingAt("n1", "title", "alice", "Bike repair", "", d.StatusActive, now.Add(-time.Minute))
	e.listingAt("n1", "tag", "alice", "Wheels", "", d.StatusActive, now.Add(-2*time.Minute))
	e.listingAt("n1", "none", "alice", "Garden hose", "", d.StatusActive, now.Add(-3*time.Minute))
	e.clock.Advance(0)
	if _, err := e.svc.UpdateDiscovery(ctx, admin, "tag", d.DiscoveryInput{Tags: []string{"bike"}}); err != nil {
		t.Fatal(err)
	}
	got := ids(e.mustSearch(bob, d.Filters{Q: "bike", Sort: d.SortRelevance}))
	if !eq(got, []string{"title", "tag", "desc"}) {
		t.Fatalf("relevance order %v", got)
	}
	got = ids(e.mustSearch(bob, d.Filters{Q: "bike", Sort: d.SortRecent}))
	// admin edit touched "tag" to now; ties on updated resolved by id ascending
	if !eq(got, []string{"desc", "tag", "title"}) {
		t.Fatalf("recent order %v", got)
	}
	if got := ids(e.mustSearch(bob, d.Filters{})); len(got) != 4 {
		t.Fatal(got)
	}
	// phrase bonus ranks the contiguous title match first
	e.listing("n1", "p1", "alice", "bike big repair", "")
	e.listing("n1", "p2", "alice", "repair bike", "")
	e.clock.Advance(time.Second)
	got = ids(e.mustSearch(bob, d.Filters{Q: "bike repair", Sort: d.SortRelevance}))
	if got[0] != "title" {
		t.Fatalf("phrase bonus: %v", got)
	}
}

func TestFiltersCombine(t *testing.T) {
	e := newEnv(t)
	goods, _ := e.svc.CreateCategory(ctx, admin, "Goods", d.TypeGoods)
	svc, _ := e.svc.CreateCategory(ctx, admin, "Services", d.TypeService)
	e.listing("n1", "g1", "alice", "Drill", "")
	e.listing("n1", "g2", "alice", "Saw", "")
	e.listing("n1", "s1", "bob", "Plumbing", "")
	e.listing("n1", "legacy", "bob", "Old listing", "")
	e.svc.UpdateDiscovery(ctx, alice, "g1", d.DiscoveryInput{CategoryID: goods.ID, Type: d.TypeGoods, Condition: "good", Availability: d.AvailAvailable})
	e.svc.UpdateDiscovery(ctx, alice, "g2", d.DiscoveryInput{CategoryID: goods.ID, Type: d.TypeGoods, Condition: "fair", Availability: d.AvailLimited})
	e.svc.UpdateDiscovery(ctx, bob, "s1", d.DiscoveryInput{CategoryID: svc.ID, Type: d.TypeService, Availability: d.AvailUnavailable})
	cases := []struct {
		f    d.Filters
		want int
	}{
		{d.Filters{}, 4},
		{d.Filters{Category: goods.ID}, 2},
		{d.Filters{Type: d.TypeService}, 1},
		{d.Filters{Condition: "fair"}, 1},
		{d.Filters{Availability: d.AvailLimited}, 1},
		{d.Filters{Availability: d.AvailAvailable}, 2}, // legacy (unspecified) counts as available
		{d.Filters{Availability: d.AvailUnavailable}, 1},
		{d.Filters{Category: goods.ID, Condition: "good"}, 1},
		{d.Filters{Kind: d.KindRequest}, 0},
		{d.Filters{Category: "nonexistent"}, 0},
	}
	for i, c := range cases {
		if got := e.mustSearch(carolInN1(), c.f); len(got.Items) != c.want {
			t.Errorf("case %d %+v: %v want %d", i, c.f, ids(got), c.want)
		}
	}
	// unclassified legacy listing searchable and flagged
	v, err := e.svc.GetListing(ctx, bob, "legacy")
	if err != nil || !v.Unclassified {
		t.Fatal(v.Unclassified, err)
	}
	for _, bad := range []d.Filters{{Type: "x"}, {Kind: "x"}, {Condition: "x"}, {Availability: "x"}, {Sort: "x"}, {Scope: "x"}, {Limit: 1000}, {Category: "a b"}, {WithinKM: 99999, NearArea: "x"}} {
		if _, err := e.svc.Search(ctx, bob, bad, ""); !errors.Is(err, d.ErrInvalid) {
			t.Errorf("%+v accepted: %v", bad, err)
		}
	}
}

func carolInN1() d.Actor { return d.Actor{ID: "dave", Network: "n1", Role: d.RoleMember, Active: true} }

func TestVisibilityMatrix(t *testing.T) {
	e := newEnv(t)
	now := e.clock.Now()
	e.listing("n1", "pub", "alice", "Public item", "")
	e.listingAt("n1", "paused", "alice", "Paused item", "", d.StatusPaused, now)
	e.listingAt("n1", "arch", "alice", "Archived item", "", d.StatusArchived, now)
	e.listingAt("n1", "susp", "alice", "Suspended item", "", d.StatusSuspended, now)
	rec := e.listing("n1", "gone", "alice", "Departed owner item", "")
	rec.OwnerActive = false
	if err := e.svc.SyncListing(ctx, rec); err != nil {
		t.Fatal(err)
	}
	e.listing("n1", "exp", "alice", "Expiring item", "")
	e.listing("n1", "future", "alice", "Future item", "")
	e.svc.UpdateDiscovery(ctx, alice, "exp", d.DiscoveryInput{ExpiresAt: now.Add(time.Hour)})
	e.svc.UpdateDiscovery(ctx, alice, "future", d.DiscoveryInput{AvailableFrom: now.Add(time.Hour), ExpiresAt: now.Add(48 * time.Hour)})
	e.listing("n2", "other", "carol", "Public item", "")

	if got := ids(e.mustSearch(bob, d.Filters{})); !eq(got, []string{"exp", "pub"}) && !eq(got, []string{"pub", "exp"}) {
		t.Fatalf("member view %v", got)
	}
	if got := ids(e.mustSearch(carol, d.Filters{})); !eq(got, []string{"other"}) {
		t.Fatalf("n2 view %v", got)
	}
	if got := e.mustSearch(alice, d.Filters{Scope: d.ScopeMine}); len(got.Items) != 7 {
		t.Fatalf("owner mine %v", ids(got))
	}
	if got := e.mustSearch(bob, d.Filters{Scope: d.ScopeMine}); len(got.Items) != 0 {
		t.Fatalf("bob mine %v", ids(got))
	}
	if got := e.mustSearch(admin, d.Filters{Scope: d.ScopeAll}); len(got.Items) != 7 {
		t.Fatalf("admin all %v", ids(got))
	}
	if got := e.mustSearch(admin2, d.Filters{Scope: d.ScopeAll}); !eq(ids(got), []string{"other"}) {
		t.Fatalf("admin2 all %v", ids(got))
	}
	if _, err := e.svc.Search(ctx, bob, d.Filters{Scope: d.ScopeAll}, ""); !errors.Is(err, d.ErrForbidden) {
		t.Fatal("member used admin scope", err)
	}
	// direct fetch rules
	for _, id := range []string{"paused", "arch", "susp", "gone", "future", "other", "missing", "../x"} {
		if _, err := e.svc.GetListing(ctx, bob, id); !errors.Is(err, d.ErrNotFound) {
			t.Errorf("bob GetListing(%s) = %v", id, err)
		}
	}
	for _, id := range []string{"paused", "arch", "susp", "gone", "future"} {
		if _, err := e.svc.GetListing(ctx, alice, id); err != nil {
			t.Errorf("owner GetListing(%s) = %v", id, err)
		}
		if _, err := e.svc.GetListing(ctx, admin, id); err != nil {
			t.Errorf("admin GetListing(%s) = %v", id, err)
		}
		if _, err := e.svc.GetListing(ctx, admin2, id); !errors.Is(err, d.ErrNotFound) {
			t.Errorf("foreign admin GetListing(%s) = %v", id, err)
		}
	}
	// expiry boundary: at exactly expires_at the listing is no longer public
	e.clock.Advance(time.Hour - time.Second)
	if _, err := e.svc.GetListing(ctx, bob, "exp"); err != nil {
		t.Fatal("expired one second early", err)
	}
	e.clock.Advance(time.Second)
	if _, err := e.svc.GetListing(ctx, bob, "exp"); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("expiry boundary not honored", err)
	}
	v, _ := e.svc.GetListing(ctx, alice, "exp")
	if !v.Expired || !v.Stale {
		t.Fatal(v.Expired, v.Stale)
	}
	if _, err := e.svc.GetListing(ctx, bob, "future"); err != nil {
		t.Fatal("available_from reached", err)
	}
	// inactive / malformed actors are denied everything
	for _, a := range []d.Actor{{ID: "x", Network: "n1", Active: false}, {Network: "n1", Active: true}, {ID: "x", Active: true}} {
		if _, err := e.svc.Search(ctx, a, d.Filters{}, ""); !errors.Is(err, d.ErrForbidden) {
			t.Errorf("%+v: %v", a, err)
		}
		if _, err := e.svc.GetListing(ctx, a, "pub"); !errors.Is(err, d.ErrForbidden) {
			t.Errorf("%+v: %v", a, err)
		}
	}
}

func seed(e *env, n int) []string {
	now := e.clock.Now()
	var order []string
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("L%02d", i)
		e.listingAt("n1", id, "alice", "Bike "+id, "", d.StatusActive, now.Add(-time.Duration(i/2)*time.Minute)) // pairs share timestamps
		order = append(order, id)
	}
	return order
}

func walk(t *testing.T, e *env, a d.Actor, f d.Filters, between func(page int)) []string {
	t.Helper()
	var all []string
	cursor := ""
	for page := 0; page < 50; page++ {
		r, err := e.svc.Search(ctx, a, f, cursor)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, ids(r)...)
		if r.Next == "" {
			return all
		}
		cursor = r.Next
		if between != nil {
			between(page)
		}
	}
	t.Fatal("pagination did not terminate")
	return nil
}

func TestPaginationDeterministicAndComplete(t *testing.T) {
	e := newEnv(t)
	want := seed(e, 25)
	for _, sortBy := range []string{d.SortRecent, d.SortRelevance} {
		f := d.Filters{Q: "bike", Limit: 10, Sort: sortBy}
		got := walk(t, e, bob, f, nil)
		if !eq(got, want) {
			t.Fatalf("%s: %v want %v", sortBy, got, want)
		}
		full := ids(e.mustSearch(bob, d.Filters{Q: "bike", Limit: 50, Sort: sortBy}))
		if !eq(full, want) {
			t.Fatal("single page differs", full)
		}
	}
}

func TestPaginationStableUnderWrites(t *testing.T) {
	e := newEnv(t)
	seed(e, 25)
	touched := map[string]bool{}
	got := walk(t, e, bob, d.Filters{Limit: 10}, func(page int) {
		if page != 0 {
			return
		}
		// concurrent writes after the first page was served
		e.clock.Advance(time.Minute)
		e.listing("n1", "NEWEST", "alice", "Bike brand new", "") // insert
		e.listingAt("n1", "L15", "alice", "Bike L15 edited", "", d.StatusActive, e.clock.Now())
		touched["L15"] = true // update moves it above the cursor
		e.listingAt("n1", "L20", "alice", "Bike L20", "", d.StatusArchived, e.clock.Now())
		touched["L20"] = true // hidden mid-walk
	})
	seen := map[string]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("duplicate %s in %v", id, got)
		}
		seen[id] = true
		if id == "NEWEST" {
			t.Fatal("post-snapshot insert leaked into continuing pages")
		}
		if id == "L20" {
			t.Fatal("archived-mid-walk listing leaked")
		}
	}
	if len(got) < 20 {
		t.Fatalf("lost rows: %v", got)
	}
	// a fresh search sees the new state
	fresh := ids(e.mustSearch(bob, d.Filters{Limit: 50}))
	if fresh[0] != "NEWEST" && fresh[0] != "L15" {
		t.Fatal(fresh[:3])
	}
}

func TestCursorBinding(t *testing.T) {
	e := newEnv(t)
	seed(e, 15)
	f := d.Filters{Q: "bike", Limit: 5}
	r, err := e.svc.Search(ctx, bob, f, "")
	if err != nil || r.Next == "" {
		t.Fatal(err)
	}
	cur := r.Next
	bad := func(name string, a d.Actor, f d.Filters, c string) {
		t.Helper()
		if _, err := e.svc.Search(ctx, a, f, c); !errors.Is(err, d.ErrBadCursor) {
			t.Errorf("%s: %v", name, err)
		}
	}
	bad("other actor", alice, f, cur)
	bad("other network", carol, f, cur)
	g := f
	g.Limit = 6
	bad("other limit", bob, g, cur)
	g = f
	g.Q = "bik"
	bad("other query", bob, g, cur)
	g = f
	g.Category = "x"
	bad("other filter", bob, g, cur)
	g = f
	g.Sort = d.SortRecent
	bad("other sort", bob, g, cur)
	body, sig, _ := strings.Cut(cur, ".")
	bad("tampered payload", bob, f, body+"x."+sig)
	bad("tampered sig", bob, f, body+"."+sig[:len(sig)-2]+"AA")
	bad("garbage", bob, f, "not-a-cursor")
	bad("empty sig", bob, f, body+".")
	bad("oversize", bob, f, strings.Repeat("A", 2000))
	// valid cursor works and expires
	if _, err := e.svc.Search(ctx, bob, f, cur); err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(16 * time.Minute)
	bad("expired", bob, f, cur)
	// forged cursor signed with a different key
	e2 := newEnv(t)
	e2.cfg.CursorKeys = [][]byte{testKey(9)}
	e2.open()
	seed(e2, 15)
	r2, _ := e2.svc.Search(ctx, bob, f, "")
	bad("foreign key", bob, f, r2.Next)
}

func TestCursorKeyRotationAndRestart(t *testing.T) {
	e := newEnv(t)
	seed(e, 15)
	f := d.Filters{Limit: 5}
	r, _ := e.svc.Search(ctx, bob, f, "")
	// restart + rotate: new primary key, old key still verifies
	e.cfg.CursorKeys = [][]byte{testKey(2), testKey(1)}
	e.open()
	r2, err := e.svc.Search(ctx, bob, f, r.Next)
	if err != nil || len(r2.Items) != 5 {
		t.Fatal(err, ids(r2))
	}
	// after dropping the old key the old cursor stops verifying
	e.cfg.CursorKeys = [][]byte{testKey(2)}
	e.open()
	if _, err := e.svc.Search(ctx, bob, f, r.Next); !errors.Is(err, d.ErrBadCursor) {
		t.Fatal(err)
	}
	if _, err := d.NewService(e.store, e.blobs, d.Config{CursorKeys: [][]byte{[]byte("short")}}, e.img); err == nil {
		t.Fatal("short key accepted")
	}
	if _, err := d.NewService(e.store, e.blobs, d.Config{}, e.img); err == nil {
		t.Fatal("missing key accepted")
	}
}

func TestSavedViewURLs(t *testing.T) {
	cfg := d.Config{}
	f, cur, err := d.ParseFilters(url.Values{"q": {"bike"}, "category": {"cat_1"}, "sort": {"recent"}, "limit": {"10"}, "cursor": {"abc"}}, cfg)
	if err != nil || cur != "abc" {
		t.Fatal(err, cur)
	}
	enc := f.Canonical(cfg)
	if strings.Contains(enc, "cursor") || strings.Contains(enc, "abc") {
		t.Fatal("cursor in saved view:", enc)
	}
	f2, _, err := d.ParseFilters(mustQuery(t, enc), cfg)
	if err != nil || f2 != f {
		t.Fatalf("round trip: %+v vs %+v (%v)", f2, f, err)
	}
	if f.Canonical(cfg) != f2.Canonical(cfg) {
		t.Fatal("non-canonical")
	}
	for _, bad := range []string{"token=abc", "csrf=1", "session_id=1", "api_key=1", "q=a&q=b", "bogus=1", "limit=0", "limit=x", "within_km=5", "near=a", "tag=%3Cb%3E", "sort=zzz"} {
		if _, _, err := d.ParseFilters(mustQuery(t, bad), cfg); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	// defaults are omitted so equal views share one URL
	g, _, _ := d.ParseFilters(url.Values{}, cfg)
	if g.Canonical(cfg) != "" {
		t.Fatal(g.Canonical(cfg))
	}
}

func mustQuery(t *testing.T, s string) url.Values {
	v, err := url.ParseQuery(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
