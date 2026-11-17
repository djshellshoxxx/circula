package onboarding

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

// storeContract is run against every Store implementation.
func storeContract(t *testing.T, open func(t *testing.T) Store) {
	ctx := context.Background()
	t.Run("progress CAS and isolation", func(t *testing.T) {
		s := open(t)
		k := Key{"net-a", "m1"}
		if _, ok, err := s.LoadProgress(ctx, k); ok || err != nil {
			t.Fatal(ok, err)
		}
		p, err := s.SaveProgress(ctx, k, 0, Progress{Version: 1, Completed: 1}, nil, t0)
		if err != nil || p.Revision != 1 || !p.UpdatedAt.Equal(t0) {
			t.Fatalf("%+v %v", p, err)
		}
		// second insert at same version loses
		if _, err := s.SaveProgress(ctx, k, 0, Progress{Version: 1, Completed: 2}, nil, t0); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		// stale revision loses
		if _, err := s.SaveProgress(ctx, k, 7, Progress{Version: 1, Completed: 3}, nil, t0); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		// update of absent version with nonzero revision loses
		if _, err := s.SaveProgress(ctx, k, 1, Progress{Version: 2}, nil, t0); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		p, err = s.SaveProgress(ctx, k, 1, Progress{Version: 1, Completed: 3, Skipped: 4, Dismissed: true, Share: true}, nil, t0.Add(time.Minute))
		if err != nil || p.Revision != 2 {
			t.Fatalf("%+v %v", p, err)
		}
		got, ok, err := s.LoadProgress(ctx, k)
		if err != nil || !ok || got.Completed != 3 || got.Skipped != 4 || !got.Dismissed || !got.Share || got.Revision != 2 || !got.UpdatedAt.Equal(t0.Add(time.Minute)) {
			t.Fatalf("%+v %v", got, err)
		}
		// other network / other member see nothing
		for _, o := range []Key{{"net-b", "m1"}, {"net-a", "m2"}} {
			if _, ok, _ := s.LoadProgress(ctx, o); ok {
				t.Fatalf("leak to %v", o)
			}
		}
		// invalid states are refused at the store too
		if _, err := s.SaveProgress(ctx, k, 2, Progress{Version: 1, Completed: 1, Skipped: 1}, nil, t0); !errors.Is(err, ErrInvalidState) {
			t.Fatal(err)
		}
		if _, err := s.SaveProgress(ctx, Key{"bad id", "m"}, 0, Progress{Version: 1}, nil, t0); !errors.Is(err, ErrInvalidIdentifier) {
			t.Fatal(err)
		}
	})

	t.Run("latest version wins and delete", func(t *testing.T) {
		s := open(t)
		k := Key{"n", "m"}
		s.SaveProgress(ctx, k, 0, Progress{Version: 1, Completed: 1}, nil, t0)
		s.SaveProgress(ctx, k, 0, Progress{Version: 2, Completed: 2}, nil, t0)
		got, _, _ := s.LoadProgress(ctx, k)
		if got.Version != 2 {
			t.Fatalf("%+v", got)
		}
		if err := s.DeleteMember(ctx, k); err != nil {
			t.Fatal(err)
		}
		if _, ok, _ := s.LoadProgress(ctx, k); ok {
			t.Fatal("not deleted")
		}
	})

	t.Run("concurrent CAS exactly one winner", func(t *testing.T) {
		s := open(t)
		k := Key{"n", "m"}
		var wins, conflicts atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := s.SaveProgress(ctx, k, 0, Progress{Version: 1, Completed: 1 << (i % 5)}, nil, t0)
				switch {
				case err == nil:
					wins.Add(1)
				case errors.Is(err, ErrConflict):
					conflicts.Add(1)
				default:
					t.Error(err)
				}
			}(i)
		}
		wg.Wait()
		if wins.Load() != 1 || conflicts.Load() != 7 {
			t.Fatalf("wins=%d conflicts=%d", wins.Load(), conflicts.Load())
		}
	})

	t.Run("purge stale", func(t *testing.T) {
		s := open(t)
		s.SaveProgress(ctx, Key{"n", "old"}, 0, Progress{Version: 1}, nil, t0)
		s.SaveProgress(ctx, Key{"n", "new"}, 0, Progress{Version: 1}, nil, t0.Add(48*time.Hour))
		n, err := s.PurgeStale(ctx, t0.Add(24*time.Hour))
		if err != nil || n != 1 {
			t.Fatal(n, err)
		}
		if _, ok, _ := s.LoadProgress(ctx, Key{"n", "old"}); ok {
			t.Fatal("old survived")
		}
		if _, ok, _ := s.LoadProgress(ctx, Key{"n", "new"}); !ok {
			t.Fatal("new purged")
		}
	})

	t.Run("copy CAS", func(t *testing.T) {
		s := open(t)
		if _, ok, err := s.LoadCopy(ctx, "n", 1); ok || err != nil {
			t.Fatal(ok, err)
		}
		c, err := s.SaveCopy(ctx, Copy{NetworkID: "n", Version: 1, Intro: "hi", UpdatedBy: "admin"}, 0, t0)
		if err != nil || c.Revision != 1 {
			t.Fatal(c, err)
		}
		if _, err := s.SaveCopy(ctx, Copy{NetworkID: "n", Version: 1, UpdatedBy: "admin"}, 0, t0); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		if _, err := s.SaveCopy(ctx, Copy{NetworkID: "n", Version: 1, Intro: "x", UpdatedBy: "admin"}, 5, t0); !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
		c, err = s.SaveCopy(ctx, Copy{NetworkID: "n", Version: 1, Intro: "two", TermsURL: "/terms", UpdatedBy: "admin2"}, 1, t0.Add(time.Hour))
		if err != nil || c.Revision != 2 {
			t.Fatal(c, err)
		}
		got, ok, _ := s.LoadCopy(ctx, "n", 1)
		if !ok || got.Intro != "two" || got.TermsURL != "/terms" || got.UpdatedBy != "admin2" || got.Revision != 2 {
			t.Fatalf("%+v", got)
		}
		if _, ok, _ := s.LoadCopy(ctx, "other", 1); ok {
			t.Fatal("copy leaked across networks")
		}
	})

	t.Run("metrics default off, opt-in only, deleted on disable", func(t *testing.T) {
		s := open(t)
		k := Key{"n", "m"}
		if on, _ := s.MetricsEnabled(ctx, "n"); on {
			t.Fatal("default must be off")
		}
		// disabled: no counting even with member consent
		s.SaveProgress(ctx, k, 0, Progress{Version: 1, Completed: 1, Share: true}, []Step{Terms}, t0)
		if c, _ := s.MetricCounts(ctx, "n", 1); c != [StepCount]int64{} {
			t.Fatalf("counted while disabled: %v", c)
		}
		if err := s.SetMetricsEnabled(ctx, "n", true, "admin", t0); err != nil {
			t.Fatal(err)
		}
		// enabled but member did not consent: no counting
		s.SaveProgress(ctx, Key{"n", "m2"}, 0, Progress{Version: 1, Completed: 1}, []Step{Terms}, t0)
		if c, _ := s.MetricCounts(ctx, "n", 1); c != [StepCount]int64{} {
			t.Fatalf("counted without consent: %v", c)
		}
		// enabled + consent: counted once per transition
		s.SaveProgress(ctx, k, 1, Progress{Version: 1, Completed: 3, Share: true}, []Step{Credit}, t0)
		c, _ := s.MetricCounts(ctx, "n", 1)
		if c[Credit] != 1 || c[Terms] != 0 {
			t.Fatalf("%v", c)
		}
		// other network unaffected
		if c, _ := s.MetricCounts(ctx, "other", 1); c != [StepCount]int64{} {
			t.Fatal("leak")
		}
		s.SetMetricsEnabled(ctx, "n", false, "admin", t0)
		if c, _ := s.MetricCounts(ctx, "n", 1); c != [StepCount]int64{} {
			t.Fatalf("counters survived disable: %v", c)
		}
	})
}

func TestMemStoreContract(t *testing.T) {
	storeContract(t, func(t *testing.T) Store { return NewMemStore() })
}
