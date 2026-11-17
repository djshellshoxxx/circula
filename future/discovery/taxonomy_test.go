package discovery_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

func TestCategoryAdminOnlyAndValidation(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.CreateCategory(ctx, alice, "Tools", d.TypeGoods); !errors.Is(err, d.ErrForbidden) {
		t.Fatal("member created category", err)
	}
	c, err := e.svc.CreateCategory(ctx, admin, "Tools", d.TypeGoods)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tools", "TOOLS", " Tôols "} {
		if _, err := e.svc.CreateCategory(ctx, admin, name, d.TypeGoods); !errors.Is(err, d.ErrDuplicate) {
			t.Fatalf("%q: %v", name, err)
		}
	}
	for _, bad := range []string{"", "x", strings.Repeat("a", 41), "a\x00b", "<script>x</script>", "evil‮name", "bad\xffutf"} {
		if _, err := e.svc.CreateCategory(ctx, admin, bad, d.TypeGoods); !errors.Is(err, d.ErrInvalid) {
			t.Fatalf("%q accepted: %v", bad, err)
		}
	}
	if _, err := e.svc.CreateCategory(ctx, admin, "Fine name", "weird"); !errors.Is(err, d.ErrInvalid) {
		t.Fatal("kind")
	}
	// same name allowed in another network
	if _, err := e.svc.CreateCategory(ctx, admin2, "Tools", d.TypeGoods); err != nil {
		t.Fatal(err)
	}
	// members only see enabled categories; admin sees all
	off := false
	if err := e.svc.UpdateCategory(ctx, admin, c.ID, nil, &off); err != nil {
		t.Fatal(err)
	}
	if l, _ := e.svc.ListCategories(ctx, alice); len(l) != 0 {
		t.Fatal("disabled category visible to member", l)
	}
	if l, _ := e.svc.ListCategories(ctx, admin); len(l) != 1 || l[0].Enabled {
		t.Fatal(l)
	}
	// cross-network update looks missing
	if err := e.svc.UpdateCategory(ctx, admin2, c.ID, nil, &off); !errors.Is(err, d.ErrNotFound) {
		t.Fatal(err)
	}
	if err := e.svc.UpdateCategory(ctx, alice, c.ID, nil, &off); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	newName := "Equipment"
	if err := e.svc.UpdateCategory(ctx, admin, c.ID, &newName, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCategoryLimitAndSeed(t *testing.T) {
	e := newEnv(t)
	e.cfg.MaxCategories = 3
	e.open()
	for _, n := range []string{"Aa", "Bb", "Cc"} {
		if _, err := e.svc.CreateCategory(ctx, admin, n, d.TypeAny); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.svc.CreateCategory(ctx, admin, "Dd", d.TypeAny); !errors.Is(err, d.ErrQuota) {
		t.Fatal(err)
	}
	e2 := newEnv(t)
	n, err := e2.svc.SeedDefaultCategories(ctx, admin)
	if err != nil || n != len(d.DefaultCategories) {
		t.Fatal(n, err)
	}
	n, err = e2.svc.SeedDefaultCategories(ctx, admin) // idempotent
	if err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err := e2.svc.SeedDefaultCategories(ctx, alice); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
}

func TestClassificationRules(t *testing.T) {
	e := newEnv(t)
	goods, _ := e.svc.CreateCategory(ctx, admin, "Goods cat", d.TypeGoods)
	svc, _ := e.svc.CreateCategory(ctx, admin, "Service cat", d.TypeService)
	other, _ := e.svc.CreateCategory(ctx, admin2, "Other net", d.TypeAny)
	area, _ := e.svc.CreateArea(ctx, admin, "North", nil)
	foreignArea, _ := e.svc.CreateArea(ctx, admin2, "South", nil)
	e.listing("n1", "L1", "alice", "Bike", "")

	ok := d.DiscoveryInput{CategoryID: goods.ID, Type: d.TypeGoods, Condition: "good", AreaID: area.ID, Availability: d.AvailAvailable}
	v, err := e.svc.UpdateDiscovery(ctx, alice, "L1", ok)
	if err != nil {
		t.Fatal(err)
	}
	if v.CategoryID != goods.ID || v.Version != 1 || v.Unclassified {
		t.Fatalf("%+v", v)
	}
	bad := map[string]d.DiscoveryInput{
		"foreign category": {CategoryID: other.ID, ExpectedVersion: 1},
		"unknown category": {CategoryID: "cat_nope", ExpectedVersion: 1},
		"foreign area":     {AreaID: foreignArea.ID, ExpectedVersion: 1},
		"kind mismatch":    {CategoryID: goods.ID, Type: d.TypeService, ExpectedVersion: 1},
		"service cond":     {CategoryID: svc.ID, Type: d.TypeService, Condition: "new", ExpectedVersion: 1},
		"bad condition":    {Type: d.TypeGoods, Condition: "pristine", ExpectedVersion: 1},
		"bad type":         {Type: "digital", ExpectedVersion: 1},
		"bad avail":        {Availability: "soon", ExpectedVersion: 1},
		"bad id":           {CategoryID: "x/../y", ExpectedVersion: 1},
		"stale version":    {CategoryID: goods.ID, Type: d.TypeGoods, ExpectedVersion: 0},
	}
	for name, in := range bad {
		_, err := e.svc.UpdateDiscovery(ctx, alice, "L1", in)
		if err == nil {
			t.Errorf("%s accepted", name)
		}
		if name == "stale version" && !errors.Is(err, d.ErrConflict) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// disabling a category keeps existing listings; new assignment fails
	off := false
	e.svc.UpdateCategory(ctx, admin, goods.ID, nil, &off)
	if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{CategoryID: goods.ID, Type: d.TypeGoods, ExpectedVersion: 1}); err != nil {
		t.Fatal("keeping existing disabled category must work:", err)
	}
	e.listing("n1", "L2", "bob", "Saw", "")
	if _, err := e.svc.UpdateDiscovery(ctx, bob, "L2", d.DiscoveryInput{CategoryID: goods.ID}); !errors.Is(err, d.ErrInvalid) {
		t.Fatal("disabled category assignable", err)
	}
	if r := e.mustSearch(alice, d.Filters{Category: goods.ID}); len(r.Items) != 1 {
		t.Fatal("existing listing must keep category", ids(r))
	}
	// non-owner cannot edit; stranger network sees nothing
	if _, err := e.svc.UpdateDiscovery(ctx, bob, "L1", d.DiscoveryInput{ExpectedVersion: 2}); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := e.svc.UpdateDiscovery(ctx, carol, "L1", d.DiscoveryInput{}); !errors.Is(err, d.ErrNotFound) {
		t.Fatal(err)
	}
	// admin may edit
	if _, err := e.svc.UpdateDiscovery(ctx, admin, "L1", d.DiscoveryInput{CategoryID: goods.ID, Type: d.TypeGoods, ExpectedVersion: 2}); err != nil {
		t.Fatal(err)
	}
}

func TestTagModeration(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bicycle", "")
	e.listing("n1", "L2", "bob", "Other", "")
	v, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{Tags: []string{"  Road Bike ", "road-bike", "Fixie", "ÉCO"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Tags) != 3 {
		t.Fatalf("normalize/dedupe: %+v", v.Tags)
	}
	// pending tags are invisible to others and not searchable
	vb, _ := e.svc.GetListing(ctx, bob, "L1")
	if len(vb.Tags) != 0 {
		t.Fatal("pending tag visible to other member", vb.Tags)
	}
	if r := e.mustSearch(bob, d.Filters{Tag: "fixie"}); len(r.Items) != 0 {
		t.Fatal("pending tag searchable")
	}
	if r := e.mustSearch(bob, d.Filters{Q: "fixie"}); len(r.Items) != 0 {
		t.Fatal("pending tag matched by text")
	}
	// member cannot moderate or see queue
	if err := e.svc.ModerateTag(ctx, bob, "fixie", d.TagApproved); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	pend, _ := e.svc.ListTags(ctx, admin, d.TagPending)
	if len(pend) != 3 {
		t.Fatal(pend)
	}
	if l, _ := e.svc.ListTags(ctx, bob, d.TagPending); len(l) != 0 {
		t.Fatal("member saw moderation queue")
	}
	if err := e.svc.ModerateTag(ctx, admin, "fixie", d.TagApproved); err != nil {
		t.Fatal(err)
	}
	if r := e.mustSearch(bob, d.Filters{Tag: "Fixie"}); len(r.Items) != 1 {
		t.Fatal("approved tag not searchable")
	}
	vb, _ = e.svc.GetListing(ctx, bob, "L1")
	if len(vb.Tags) != 1 || vb.Tags[0].Name != "fixie" {
		t.Fatal(vb.Tags)
	}
	// reject: disappears and cannot be reattached
	if err := e.svc.ModerateTag(ctx, admin, "fixie", d.TagRejected); err != nil {
		t.Fatal(err)
	}
	if r := e.mustSearch(bob, d.Filters{Tag: "fixie"}); len(r.Items) != 0 {
		t.Fatal("rejected tag searchable")
	}
	if _, err := e.svc.UpdateDiscovery(ctx, bob, "L2", d.DiscoveryInput{Tags: []string{"fixie"}}); !errors.Is(err, d.ErrRejected) {
		t.Fatal(err)
	}
	// admin-proposed tags are approved immediately
	if _, err := e.svc.UpdateDiscovery(ctx, admin, "L2", d.DiscoveryInput{Tags: []string{"curated"}}); err != nil {
		t.Fatal(err)
	}
	if r := e.mustSearch(alice, d.Filters{Tag: "curated"}); len(r.Items) != 1 {
		t.Fatal("admin tag not approved")
	}
	// tags are per network
	if l, _ := e.svc.ListTags(ctx, admin2, ""); len(l) != 0 {
		t.Fatal("tags leaked across networks", l)
	}
	// limits and bad tags
	many := []string{"aa", "bb", "cc", "dd", "ee", "ff", "gg", "hh", "ii"}
	if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{Tags: many, ExpectedVersion: 1}); !errors.Is(err, d.ErrInvalid) {
		t.Fatal(err)
	}
	for _, bad := range []string{"a", "x<y>", "emoji😀", strings.Repeat("z", 30), "a\x00b"} {
		if _, err := d.NormalizeTag(bad); err == nil {
			t.Errorf("tag %q accepted", bad)
		}
	}
}

func TestTagNetworkLimit(t *testing.T) {
	e := newEnv(t)
	e.cfg.MaxTagsPerNetwork = 2
	e.open()
	e.listing("n1", "L1", "alice", "x", "")
	if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{Tags: []string{"aa", "bb", "cc"}}); !errors.Is(err, d.ErrQuota) {
		t.Fatal(err)
	}
}

func TestConcurrentDuplicateCategory(t *testing.T) {
	e := newEnv(t)
	var wg sync.WaitGroup
	var ok, dup int32
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.svc.CreateCategory(ctx, admin, "Same name", d.TypeAny)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, d.ErrDuplicate):
				dup++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if ok != 1 || dup != 11 {
		t.Fatal(ok, dup)
	}
}

func TestAreasCoarseAndDistance(t *testing.T) {
	e := newEnv(t)
	van, _ := e.svc.CreateArea(ctx, admin, "Vancouver", &d.GeoCell{Lat10: 492, Lon10: -1231})
	bby, _ := e.svc.CreateArea(ctx, admin, "Burnaby", &d.GeoCell{Lat10: 492, Lon10: -1230})
	vic, _ := e.svc.CreateArea(ctx, admin, "Victoria", &d.GeoCell{Lat10: 484, Lon10: -1234})
	nogeo, _ := e.svc.CreateArea(ctx, admin, "Remote", nil)
	if _, err := e.svc.CreateArea(ctx, alice, "Hack", nil); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateArea(ctx, admin, "Bad", &d.GeoCell{Lat10: 9999, Lon10: 0}); !errors.Is(err, d.ErrInvalid) {
		t.Fatal(err)
	}
	for i, a := range []d.Area{van, bby, vic, nogeo} {
		id := []string{"A", "B", "C", "D"}[i]
		e.listing("n1", id, "alice", "Item "+id, "")
		if _, err := e.svc.UpdateDiscovery(ctx, alice, id, d.DiscoveryInput{AreaID: a.ID}); err != nil {
			t.Fatal(err)
		}
	}
	if got := ids(e.mustSearch(bob, d.Filters{Area: van.ID})); len(got) != 1 || got[0] != "A" {
		t.Fatal(got)
	}
	got := ids(e.mustSearch(bob, d.Filters{NearArea: van.ID, WithinKM: 30}))
	if len(got) != 2 {
		t.Fatalf("near 30km: %v", got)
	}
	got = ids(e.mustSearch(bob, d.Filters{NearArea: van.ID, WithinKM: 200}))
	if len(got) != 3 {
		t.Fatalf("near 200km: %v", got)
	}
	if _, err := e.svc.Search(ctx, bob, d.Filters{NearArea: nogeo.ID, WithinKM: 10}, ""); !errors.Is(err, d.ErrInvalid) {
		t.Fatal("area without centroid usable as origin", err)
	}
	if _, err := e.svc.Search(ctx, bob, d.Filters{NearArea: van.ID}, ""); !errors.Is(err, d.ErrInvalid) {
		t.Fatal(err)
	}
	// other network's area ids match nothing and are not an oracle
	if got := ids(e.mustSearch(carol, d.Filters{Area: van.ID})); len(got) != 0 {
		t.Fatal(got)
	}
	// no coordinates ever surface in a view
	v, _ := e.svc.GetListing(ctx, bob, "A")
	if strings.Contains(strings.ToLower(strings.Join([]string{v.AreaID, v.Title}, " ")), "492") {
		t.Fatal("coordinate leakage")
	}
}
