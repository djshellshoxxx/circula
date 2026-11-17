package discovery_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

func TestRenewalSemantics(t *testing.T) {
	e := newEnv(t)
	rec := e.listing("n1", "L1", "alice", "Bike", "")
	now := e.clock.Now()
	// renew a listing that has no discovery row yet
	v, err := e.svc.Renew(ctx, alice, "L1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := now.Add(30 * 24 * time.Hour); !v.ExpiresAt.Equal(want) || v.RenewCount != 1 || v.Version != 1 {
		t.Fatalf("%+v", v)
	}
	// original identity and trade-anchoring fields are untouched
	if v.ID != rec.ID || !v.Created.Equal(rec.Created) || v.OwnerID != rec.OwnerID {
		t.Fatal("renew changed listing identity")
	}
	// renewal extends from the existing expiry when still in the future, capped
	v, err = e.svc.Renew(ctx, alice, "L1", 1)
	if err != nil || !v.ExpiresAt.Equal(now.Add(60*24*time.Hour)) {
		t.Fatal(v.ExpiresAt, err)
	}
	for i := int64(2); i < 20; i++ {
		if v, err = e.svc.Renew(ctx, alice, "L1", i); err != nil {
			t.Fatal(err)
		}
	}
	if v.ExpiresAt.After(now.Add(365 * 24 * time.Hour)) {
		t.Fatal("expiry beyond cap:", v.ExpiresAt)
	}
	// stale token, other members, foreign network, archived
	if _, err := e.svc.Renew(ctx, alice, "L1", 3); !errors.Is(err, d.ErrConflict) {
		t.Fatal(err)
	}
	if _, err := e.svc.Renew(ctx, bob, "L1", 20); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	if _, err := e.svc.Renew(ctx, carol, "L1", 20); !errors.Is(err, d.ErrNotFound) {
		t.Fatal(err)
	}
	rec.Status = d.StatusArchived
	e.svc.SyncListing(ctx, rec)
	if _, err := e.svc.Renew(ctx, alice, "L1", 20); !errors.Is(err, d.ErrInvalid) {
		t.Fatal("archived listing renewed:", err)
	}
	// suspended or departed owner cannot renew
	e2 := newEnv(t)
	r2 := e2.listing("n1", "L9", "alice", "Bike", "")
	r2.Status = d.StatusSuspended
	e2.svc.SyncListing(ctx, r2)
	if _, err := e2.svc.Renew(ctx, alice, "L9", 0); !errors.Is(err, d.ErrInvalid) {
		t.Fatal(err)
	}
}

func TestStaleListingsAndRenewalRestoresVisibility(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{ExpiresAt: e.clock.Now().Add(24 * time.Hour)})
	e.clock.Advance(48 * time.Hour)
	if got := ids(e.mustSearch(bob, d.Filters{})); len(got) != 0 {
		t.Fatal("expired listing public:", got)
	}
	mine := e.mustSearch(alice, d.Filters{Scope: d.ScopeMine})
	if len(mine.Items) != 1 {
		t.Fatal(ids(mine))
	}
	v, _ := e.svc.GetListing(ctx, alice, "L1")
	if !v.Stale || !v.Expired {
		t.Fatal("expected stale+expired flags")
	}
	if _, err := e.svc.Renew(ctx, alice, "L1", v.Version); err != nil {
		t.Fatal(err)
	}
	if got := ids(e.mustSearch(bob, d.Filters{})); len(got) != 1 {
		t.Fatal("renewed listing not visible:", got)
	}
	// stale-by-age flag without expiry
	e.clock.Advance(100 * 24 * time.Hour)
	e.cfg.Now = e.clock.Now
	v, _ = e.svc.GetListing(ctx, alice, "L1")
	if !v.Stale {
		t.Fatal("old listing should be flagged stale")
	}
}

func TestAvailabilityValidation(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	now := e.clock.Now()
	for name, in := range map[string]d.DiscoveryInput{
		"past expiry":          {ExpiresAt: now.Add(-time.Hour)},
		"now expiry":           {ExpiresAt: now},
		"far expiry":           {ExpiresAt: now.Add(400 * 24 * time.Hour)},
		"from after expiry":    {AvailableFrom: now.Add(5 * time.Hour), ExpiresAt: now.Add(2 * time.Hour)},
		"from equals expiry":   {AvailableFrom: now.Add(2 * time.Hour), ExpiresAt: now.Add(2 * time.Hour)},
		"far available_from":   {AvailableFrom: now.Add(900 * 24 * time.Hour)},
		"bad availability val": {Availability: "maybe"},
	} {
		if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", in); !errors.Is(err, d.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestConcurrentRenewalsOneWinner(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, conflict := 0, 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.svc.Renew(ctx, alice, "L1", 0)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, d.ErrConflict):
				conflict++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if ok != 1 || conflict != 11 {
		t.Fatalf("ok=%d conflict=%d", ok, conflict)
	}
	v, _ := e.svc.GetListing(ctx, alice, "L1")
	if v.RenewCount != 1 {
		t.Fatal(v.RenewCount)
	}
}

func TestConcurrentSearchAndWrites(t *testing.T) {
	e := newEnv(t)
	seed(e, 30)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				id := "W" + string(rune('a'+w)) + time.Duration(i).String()
				rec := d.ListingRecord{ID: id, Network: "n1", OwnerID: "alice", Kind: d.KindOffer, Title: "Bike " + id, Status: d.StatusActive,
					OwnerActive: true, Created: e.clock.Now(), Updated: e.clock.Now()}
				if err := e.svc.SyncListing(ctx, rec); err != nil {
					t.Error(err)
					return
				}
			}
		}(w)
	}
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 15; i++ {
				seen := map[string]bool{}
				cursor := ""
				for p := 0; p < 10; p++ {
					res, err := e.svc.Search(ctx, bob, d.Filters{Q: "bike", Limit: 7}, cursor)
					if err != nil {
						t.Error(err)
						return
					}
					for _, id := range ids(res) {
						if seen[id] {
							t.Errorf("duplicate %s under concurrent writes", id)
							return
						}
						seen[id] = true
					}
					if res.Next == "" {
						break
					}
					cursor = res.Next
				}
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	// readers finish first, then stop writers
	done := make(chan struct{})
	go func() { time.Sleep(1500 * time.Millisecond); close(stop); close(done) }()
	wg.Wait()
	<-done
}
