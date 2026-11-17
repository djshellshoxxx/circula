package favorites_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/testkit"
	"github.com/djshellshoxxx/circula/future/favorites/memstore"
)

type storeFactory struct {
	name string
	new  func(t *testing.T) favorites.Store
}

// factories lists the stores every behavior test runs against. A build-tagged
// file appends the real SQLite store.
var factories = []storeFactory{{"mem", func(t *testing.T) favorites.Store { return memstore.New() }}}

type fixture struct {
	svc *favorites.Service
	dir *testkit.Directory
	st  favorites.Store
	clk *clock
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func newFixture(t *testing.T, f storeFactory, mod ...func(*favorites.Config)) *fixture {
	t.Helper()
	dir := testkit.NewDirectory()
	for _, n := range []string{"n1", "n2"} {
		for _, id := range []string{"alice", "bob", "carol", "dave", "erin"} {
			dir.Add(n, id)
		}
	}
	clk := &clock{t: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)}
	cfg := favorites.Config{Store: f.new(t), Policy: dir, Trades: dir, Clock: clk.Now}
	for _, m := range mod {
		m(&cfg)
	}
	svc, err := favorites.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{svc: svc, dir: dir, st: cfg.Store, clk: clk}
}

func eachStore(t *testing.T, fn func(t *testing.T, f storeFactory)) {
	for _, f := range factories {
		f := f
		t.Run(f.name, func(t *testing.T) { fn(t, f) })
	}
}

var (
	ctx   = context.Background()
	alice = favorites.Actor{Network: "n1", Member: "alice"}
	bob   = favorites.Actor{Network: "n1", Member: "bob"}
)

func targets(p favorites.Page) []string {
	out := []string{}
	for _, v := range p.Items {
		out = append(out, v.Target)
	}
	return out
}

func mustAdd(t *testing.T, s *favorites.Service, a favorites.Actor, target, nick string) {
	t.Helper()
	if _, _, err := s.Add(ctx, a, favorites.AddRequest{Target: target, Nickname: nick}); err != nil {
		t.Fatalf("add %s: %v", target, err)
	}
}

func TestAddListPinRenameRemove(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		s := fx.svc
		v, created, err := s.Add(ctx, alice, favorites.AddRequest{Target: "bob", Nickname: "  Bike guy  "})
		if err != nil || !created || v.Nickname != "Bike guy" || v.DisplayName != "Member bob" {
			t.Fatalf("add: %+v %v %v", v, created, err)
		}
		mustAdd(t, s, alice, "carol", "")
		mustAdd(t, s, alice, "dave", "")
		p, err := s.List(ctx, alice, favorites.ListOptions{})
		if err != nil || !reflect.DeepEqual(targets(p), []string{"bob", "carol", "dave"}) {
			t.Fatalf("list %v %v", targets(p), err)
		}
		if err := s.SetPinned(ctx, alice, "dave", true); err != nil {
			t.Fatal(err)
		}
		if err := s.SetPinned(ctx, alice, "dave", true); err != nil { // idempotent
			t.Fatal(err)
		}
		p2, _ := s.List(ctx, alice, favorites.ListOptions{})
		if !reflect.DeepEqual(targets(p2), []string{"dave", "bob", "carol"}) {
			t.Fatalf("pinned first: %v", targets(p2))
		}
		if p2.Revision <= p.Revision {
			t.Fatal("revision must advance on change")
		}
		if err := s.SetPinned(ctx, alice, "dave", false); err != nil {
			t.Fatal(err)
		}
		if err := s.SetNickname(ctx, alice, "carol", "C"); err != nil {
			t.Fatal(err)
		}
		if err := s.SetNickname(ctx, alice, "zed", "C"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("rename unknown: %v", err)
		}
		if ok, err := s.Remove(ctx, alice, "bob"); !ok || err != nil {
			t.Fatalf("remove %v %v", ok, err)
		}
		if ok, err := s.Remove(ctx, alice, "bob"); ok || err != nil {
			t.Fatalf("second remove %v %v", ok, err)
		}
		p3, _ := s.List(ctx, alice, favorites.ListOptions{})
		if !reflect.DeepEqual(targets(p3), []string{"carol", "dave"}) || p3.Items[0].Nickname != "C" {
			t.Fatalf("%+v", p3.Items)
		}
	})
}

func TestAddRefusalsAreUniform(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		d := fx.dir
		d.Set(favorites.Member{ID: "hidden", Network: "n1", Active: true, Visible: false})
		d.Set(favorites.Member{ID: "gone", Network: "n1", Active: false, Visible: true})
		d.Set(favorites.Member{ID: "susp", Network: "n1", Active: true, Visible: true, Blocked: true})
		d.Block("n1", "alice", "carol")
		d.Set(favorites.Member{ID: "onlyn2", Network: "n2", Active: true, Visible: true})
		for _, target := range []string{"alice", "hidden", "gone", "susp", "carol", "onlyn2", "nobody", "", "bad id", "x\x00"} {
			_, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: target})
			if !errors.Is(err, favorites.ErrUnavailable) {
				t.Errorf("target %q: want ErrUnavailable got %v", target, err)
			}
		}
		// inactive or suspended owner
		d.Set(favorites.Member{ID: "ghost", Network: "n1", Active: false, Visible: true})
		if _, _, err := fx.svc.Add(ctx, favorites.Actor{Network: "n1", Member: "ghost"}, favorites.AddRequest{Target: "bob"}); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("inactive owner: %v", err)
		}
		if _, _, err := fx.svc.Add(ctx, favorites.Actor{Network: "", Member: "alice"}, favorites.AddRequest{Target: "bob"}); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("empty network: %v", err)
		}
		// owner unknown to the network
		if _, _, err := fx.svc.Add(ctx, favorites.Actor{Network: "n1", Member: "mallory"}, favorites.AddRequest{Target: "bob"}); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("unknown owner: %v", err)
		}
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		if len(p.Items) != 0 {
			t.Fatalf("nothing should be stored: %v", p.Items)
		}
	})
}

func TestNicknameValidation(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		bad := map[string]string{
			"newline": "a\nb", "nul": "a\x00b", "tab": "a\tb", "bidi override": "a\u202eb", "zero width": "a\u200bb",
			"bom": "a\ufeffb", "line sep": "a\u2028b", "c1": "a\u0085b", "invalid utf8": "a\xffb", "too long": string(make([]rune, 81)),
		}
		bad["too long"] = fmt.Sprintf("%081d", 0)
		for name, n := range bad {
			_, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "bob", Nickname: n})
			var ie *favorites.InputError
			if !errors.As(err, &ie) {
				t.Errorf("%s: want InputError got %v", name, err)
			}
		}
		ok := fmt.Sprintf("%080d", 0)
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "bob", Nickname: ok + "  "}); err != nil {
			t.Fatalf("80 runes after trim ok: %v", err)
		}
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "carol", Nickname: "Zoë 家"}); err != nil {
			t.Fatal(err)
		}
		if err := fx.svc.SetNickname(ctx, alice, "carol", "a\u202eb"); err == nil {
			t.Fatal("rename must validate")
		}
	})
}

func TestAddIdempotentAndConflict(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		req := favorites.AddRequest{Target: "bob", Nickname: "B"}
		if _, c, err := fx.svc.Add(ctx, alice, req); !c || err != nil {
			t.Fatal(c, err)
		}
		if _, c, err := fx.svc.Add(ctx, alice, req); c || err != nil {
			t.Fatalf("retry must succeed without creating: %v %v", c, err)
		}
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "bob", Nickname: "other"}); !errors.Is(err, favorites.ErrExists) {
			t.Fatalf("got %v", err)
		}
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		if len(p.Items) != 1 || p.Items[0].Nickname != "B" {
			t.Fatalf("%+v", p.Items)
		}
	})
}

func TestOwnerAndNetworkIsolation(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		s := fx.svc
		aliceN2 := favorites.Actor{Network: "n2", Member: "alice"}
		mustAdd(t, s, alice, "carol", "private A")
		mustAdd(t, s, bob, "carol", "private B")
		mustAdd(t, s, aliceN2, "dave", "private N2")
		pa, _ := s.List(ctx, alice, favorites.ListOptions{})
		pb, _ := s.List(ctx, bob, favorites.ListOptions{})
		pn, _ := s.List(ctx, aliceN2, favorites.ListOptions{})
		if len(pa.Items) != 1 || pa.Items[0].Nickname != "private A" || pb.Items[0].Nickname != "private B" ||
			len(pn.Items) != 1 || pn.Items[0].Target != "dave" {
			t.Fatalf("leak: %+v %+v %+v", pa.Items, pb.Items, pn.Items)
		}
		// carol (the target) sees nothing of either relation
		carol, _ := s.List(ctx, favorites.Actor{Network: "n1", Member: "carol"}, favorites.ListOptions{})
		if len(carol.Items) != 0 {
			t.Fatal("target sees relation")
		}
		// bob cannot remove or rename alice's favorite, even by naming the same target he also saved
		if _, err := s.Remove(ctx, bob, "carol"); err != nil {
			t.Fatal(err)
		}
		pa, _ = s.List(ctx, alice, favorites.ListOptions{})
		if len(pa.Items) != 1 {
			t.Fatal("bob's remove touched alice")
		}
		if err := s.SetPinned(ctx, bob, "carol", true); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("pin foreign: %v", err)
		}
		if err := s.SetNickname(ctx, aliceN2, "carol", "x"); !errors.Is(err, favorites.ErrNotFound) {
			t.Fatalf("cross-network rename: %v", err)
		}
		// no wildcard actor
		if _, err := s.List(ctx, favorites.Actor{}, favorites.ListOptions{}); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("nil actor: %v", err)
		}
	})
}

func TestLiveAuthorizationOverridesStoredFavorites(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		s := fx.svc
		for _, id := range []string{"bob", "carol", "dave", "erin"} {
			mustAdd(t, s, alice, id, id)
		}
		d := fx.dir
		m := d.Get("n1", "bob")
		m.Blocked = true
		d.Set(m) // suspended
		d.Block("n1", "alice", "carol")
		d.Remove("n1", "dave") // departed
		m = d.Get("n1", "erin")
		m.Visible = false
		d.Set(m)
		p, err := s.List(ctx, alice, favorites.ListOptions{})
		if err != nil || len(p.Items) != 0 {
			t.Fatalf("%v %v", p.Items, err)
		}
		// owner can still remove obsolete favorites
		for _, id := range []string{"bob", "carol", "dave", "erin"} {
			if ok, err := s.Remove(ctx, alice, id); !ok || err != nil {
				t.Fatalf("remove %s: %v %v", id, ok, err)
			}
		}
		// recovery: bob is restored and can be favorited again
		m = d.Get("n1", "bob")
		m.Blocked = false
		d.Set(m)
		mustAdd(t, s, alice, "bob", "")
		p, _ = s.List(ctx, alice, favorites.ListOptions{})
		if len(p.Items) != 1 {
			t.Fatal("restored")
		}
		// owner suspended: no list, no mutation
		o := d.Get("n1", "alice")
		o.Blocked = true
		d.Set(o)
		if _, err := s.List(ctx, alice, favorites.ListOptions{}); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("suspended owner list: %v", err)
		}
		if err := s.SetPinned(ctx, alice, "bob", true); !errors.Is(err, favorites.ErrUnavailable) {
			t.Fatalf("suspended owner pin: %v", err)
		}
	})
}

func TestPolicyFailureFailsClosed(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		mustAdd(t, fx.svc, alice, "bob", "")
		fx.dir.Fail.Store(true)
		if _, err := fx.svc.List(ctx, alice, favorites.ListOptions{}); !errors.Is(err, favorites.ErrPolicy) {
			t.Fatalf("list: %v", err)
		}
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "carol"}); !errors.Is(err, favorites.ErrPolicy) {
			t.Fatalf("add: %v", err)
		}
		if _, err := fx.svc.Remove(ctx, alice, "bob"); !errors.Is(err, favorites.ErrPolicy) {
			t.Fatalf("remove: %v", err)
		}
		if _, err := fx.svc.RepeatTrade(ctx, alice, "t1", ""); err == nil {
			t.Fatal("repeat")
		}
		fx.dir.Fail.Store(false)
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		if len(p.Items) != 1 {
			t.Fatal("state must be untouched by failures")
		}
	})
}

func TestPaginationStableAndBounded(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		for i := 0; i < 9; i++ {
			id := fmt.Sprintf("m%02d", i)
			fx.dir.Add("n1", id)
			mustAdd(t, fx.svc, alice, id, "")
		}
		_ = fx.svc.SetPinned(ctx, alice, "m07", true)
		var got []string
		cur := ""
		pages := 0
		for {
			p, err := fx.svc.List(ctx, alice, favorites.ListOptions{Limit: 4, Cursor: cur})
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, targets(p)...)
			pages++
			if p.Next == "" {
				break
			}
			cur = p.Next
		}
		want := []string{"m07", "m00", "m01", "m02", "m03", "m04", "m05", "m06", "m08"}
		if !reflect.DeepEqual(got, want) || pages != 3 {
			t.Fatalf("got %v pages %d", got, pages)
		}
		for _, bad := range []string{"!!!", "bm90anNvbg", string(make([]byte, 600))} {
			if _, err := fx.svc.List(ctx, alice, favorites.ListOptions{Cursor: bad}); err == nil {
				t.Errorf("cursor %q accepted", bad)
			}
		}
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{Limit: 100000})
		if len(p.Items) != 9 {
			t.Fatal("limit clamp")
		}
		// A cursor naming another owner's member never reveals anything: scoping is by actor.
		pb, _ := fx.svc.List(ctx, bob, favorites.ListOptions{Cursor: fmt.Sprint("")})
		if len(pb.Items) != 0 {
			t.Fatal("bob sees alice")
		}
	})
}

func TestReorder(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		s := fx.svc
		for _, id := range []string{"bob", "carol", "dave", "erin"} {
			mustAdd(t, s, alice, id, "")
		}
		mustAdd(t, s, bob, "alice", "bobs")
		p, _ := s.List(ctx, alice, favorites.ListOptions{})
		before := targets(p)
		// stale
		if _, err := s.Reorder(ctx, alice, favorites.ReorderRequest{Revision: p.Revision - 1, Targets: []string{"erin"}}); !errors.Is(err, favorites.ErrStale) {
			t.Fatalf("stale: %v", err)
		}
		for name, tg := range map[string][]string{
			"duplicate": {"erin", "erin"}, "unknown": {"zed"}, "foreign": {"alice"}, "mixed": {"erin", "bob", "nobody"},
			"oversized": make([]string, 5000),
		} {
			if _, err := s.Reorder(ctx, alice, favorites.ReorderRequest{Revision: p.Revision, Targets: tg}); !errors.Is(err, favorites.ErrInvalidOrder) {
				t.Errorf("%s: %v", name, err)
			}
		}
		pp, _ := s.List(ctx, alice, favorites.ListOptions{})
		if !reflect.DeepEqual(targets(pp), before) || pp.Revision != p.Revision {
			t.Fatalf("failed reorder changed state: %v rev %d->%d", targets(pp), p.Revision, pp.Revision)
		}
		rev, err := s.Reorder(ctx, alice, favorites.ReorderRequest{Revision: p.Revision, Targets: []string{"erin", "carol"}})
		if err != nil || rev <= p.Revision {
			t.Fatal(rev, err)
		}
		pp, _ = s.List(ctx, alice, favorites.ListOptions{})
		if !reflect.DeepEqual(targets(pp), []string{"erin", "carol", "bob", "dave"}) {
			t.Fatalf("%v", targets(pp))
		}
		// replaying the old revision is now stale (no double-apply)
		if _, err := s.Reorder(ctx, alice, favorites.ReorderRequest{Revision: p.Revision, Targets: []string{"dave"}}); !errors.Is(err, favorites.ErrStale) {
			t.Fatalf("replay: %v", err)
		}
		pb, _ := s.List(ctx, bob, favorites.ListOptions{})
		if len(pb.Items) != 1 || pb.Items[0].Order != 1 {
			t.Fatalf("bob untouched: %+v", pb.Items)
		}
	})
}

func TestLimitAndPruneOfDepartedTargets(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f, func(c *favorites.Config) { c.MaxFavorites = 3 })
		for _, id := range []string{"bob", "carol", "dave"} {
			mustAdd(t, fx.svc, alice, id, "")
		}
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "erin"}); !errors.Is(err, favorites.ErrLimit) {
			t.Fatalf("limit: %v", err)
		}
		fx.dir.Remove("n1", "carol") // departed: hidden stale row must not block adding
		if _, _, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: "erin"}); err != nil {
			t.Fatalf("prune then add: %v", err)
		}
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		if !reflect.DeepEqual(targets(p), []string{"bob", "dave", "erin"}) {
			t.Fatalf("%v", targets(p))
		}
		if _, err := favorites.NewService(favorites.Config{Store: memstore.New(), Policy: fx.dir, MaxFavorites: favorites.HardMaxFavorites + 1}); err == nil {
			t.Fatal("hard cap")
		}
	})
}

func TestMemberDepartedCleanupBeyondOnePage(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f, func(c *favorites.Config) { c.MaxFavorites = 1000 })
		const n = 260 // larger than any page size
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("o%03d", i)
			fx.dir.Add("n1", id)
			mustAdd(t, fx.svc, favorites.Actor{Network: "n1", Member: id}, "dave", "")
		}
		fx.dir.Add("n2", "o000") // same id in other network
		mustAdd(t, fx.svc, favorites.Actor{Network: "n2", Member: "o000"}, "dave", "keep")
		for i := 0; i < 5; i++ {
			fx.dir.Add("n1", fmt.Sprintf("t%d", i))
			mustAdd(t, fx.svc, alice, fmt.Sprintf("t%d", i), "")
		}
		mustAdd(t, fx.svc, alice, "dave", "")
		if _, err := fx.svc.DraftFromFavorite(ctx, alice, "dave", "k1"); err != nil {
			t.Fatal(err)
		}
		fx.dir.Remove("n1", "dave")
		res, err := fx.svc.MemberDeparted(ctx, "n1", "dave")
		if err != nil || res.Favorites != n+1 || res.Intents != 1 {
			t.Fatalf("%+v %v", res, err)
		}
		fx.dir.Add("n1", "dave") // even if re-created, old relations are gone
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		for _, it := range p.Items {
			if it.Target == "dave" {
				t.Fatal("favorite to departed member survived")
			}
		}
		if len(p.Items) != 5 {
			t.Fatalf("unrelated rows lost: %d", len(p.Items))
		}
		p, _ = fx.svc.List(ctx, favorites.Actor{Network: "n2", Member: "o000"}, favorites.ListOptions{})
		if len(p.Items) != 1 {
			t.Fatal("other network purged")
		}
		// owner departs: their own list goes, revision/state too
		res, err = fx.svc.MemberDeparted(ctx, "n1", "alice")
		if err != nil || res.Favorites != 5 {
			t.Fatalf("%+v %v", res, err)
		}
		if _, err := fx.svc.MemberDeparted(ctx, "", "alice"); err == nil {
			t.Fatal("empty network")
		}
	})
}

func TestConcurrentAddsRespectUniquenessAndCap(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f, func(c *favorites.Config) { c.MaxFavorites = 3 })
		for i := 0; i < 10; i++ {
			fx.dir.Add("n1", fmt.Sprintf("p%d", i))
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		created := 0
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, c, err := fx.svc.Add(ctx, alice, favorites.AddRequest{Target: fmt.Sprintf("p%d", i%10)})
				if err != nil && !errors.Is(err, favorites.ErrLimit) {
					t.Errorf("unexpected %v", err)
				}
				if c {
					mu.Lock()
					created++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		if created != 3 || len(p.Items) != 3 {
			t.Fatalf("created %d listed %d", created, len(p.Items))
		}
		seen := map[string]bool{}
		for _, it := range p.Items {
			if seen[it.Target] {
				t.Fatal("duplicate")
			}
			seen[it.Target] = true
		}
	})
}

func TestConcurrentReorderOnlyOneWinsPerRevision(t *testing.T) {
	eachStore(t, func(t *testing.T, f storeFactory) {
		fx := newFixture(t, f)
		for _, id := range []string{"bob", "carol", "dave"} {
			mustAdd(t, fx.svc, alice, id, "")
		}
		p, _ := fx.svc.List(ctx, alice, favorites.ListOptions{})
		var wg sync.WaitGroup
		var ok, stale int32
		var mu sync.Mutex
		for _, first := range []string{"bob", "carol", "dave", "bob", "carol", "dave"} {
			wg.Add(1)
			go func(first string) {
				defer wg.Done()
				_, err := fx.svc.Reorder(ctx, alice, favorites.ReorderRequest{Revision: p.Revision, Targets: []string{first}})
				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					ok++
				case errors.Is(err, favorites.ErrStale):
					stale++
				default:
					t.Errorf("%v", err)
				}
			}(first)
		}
		wg.Wait()
		if ok != 1 || stale != 5 {
			t.Fatalf("ok=%d stale=%d", ok, stale)
		}
	})
}
