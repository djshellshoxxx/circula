package onboarding

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// purgeCutoffContract checks the retention contract against time.Time.Before.
// Fixtures go through SaveProgress so SQL rows use the adapter's real encoding.
func purgeCutoffContract(t *testing.T, open func(*testing.T) Store) {
	t.Helper()
	base := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	cutoffs := []struct {
		name   string
		before time.Time
	}{
		{"whole_second", base},
		{"first_nanosecond", base.Add(time.Nanosecond)},
		{"tenth_second", base.Add(100 * time.Millisecond)},
		{"tenth_plus_nanosecond", base.Add(100*time.Millisecond + time.Nanosecond)},
		{"half_second", base.Add(500 * time.Millisecond)},
		{"submillisecond", base.Add(123456789 * time.Nanosecond)},
		{"last_nanosecond", base.Add(time.Second - time.Nanosecond)},
		{"midnight", time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)},
		{"new_year", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"leap_midnight", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
		{"offset_cutoff", base.Add(500 * time.Millisecond).In(time.FixedZone("cutoff", 5*3600+45*60))},
		{"before_epoch", time.Date(1969, 12, 31, 23, 59, 59, 500000000, time.UTC)},
	}
	offsets := []time.Duration{
		-time.Second, -500 * time.Millisecond, -time.Nanosecond,
		0, time.Nanosecond, 500 * time.Millisecond, time.Second,
	}
	for _, tc := range cutoffs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := open(t)
			type fixture struct {
				key   Key
				saved Progress
				when  time.Time
			}
			var fixtures []fixture
			var wantRemoved int64
			for networkIndex, network := range []string{"net-a", "net-b"} {
				if err := s.SetMetricsEnabled(ctx, network, true, "admin", tc.before); err != nil {
					t.Fatal(err)
				}
				if _, err := s.SaveCopy(ctx, Copy{NetworkID: network, Version: 1, Intro: network, UpdatedBy: "admin"}, 0, tc.before); err != nil {
					t.Fatal(err)
				}
				for i := range offsets {
					index := i
					if networkIndex == 1 {
						index = len(offsets) - 1 - i
					}
					when := tc.before.Add(offsets[index])
					key := Key{network, fmt.Sprintf("member-%d", i)}
					saved, err := s.SaveProgress(ctx, key, 0, Progress{Version: 1, Completed: 1, Share: true}, []Step{Terms}, when.In(time.FixedZone("write", -7*3600)))
					if err != nil {
						t.Fatal(err)
					}
					fixtures = append(fixtures, fixture{key, saved, when})
					if when.Before(tc.before) {
						wantRemoved++
					}
				}
			}
			removed, err := s.PurgeStale(ctx, tc.before)
			if err != nil {
				t.Fatal(err)
			}
			if removed != wantRemoved {
				t.Errorf("removed %d rows, want %d before %s", removed, wantRemoved, tc.before.Format(time.RFC3339Nano))
			}
			for _, f := range fixtures {
				got, exists, err := s.LoadProgress(ctx, f.key)
				if err != nil {
					t.Fatal(err)
				}
				wantExists := !f.when.Before(tc.before)
				if exists != wantExists {
					t.Errorf("%v updated %s: exists=%t, want %t at cutoff %s", f.key, f.when.Format(time.RFC3339Nano), exists, wantExists, tc.before.Format(time.RFC3339Nano))
				}
				if exists && wantExists {
					if !got.UpdatedAt.Equal(f.saved.UpdatedAt) {
						t.Errorf("%v timestamp changed: got %s want %s", f.key, got.UpdatedAt, f.saved.UpdatedAt)
					}
					got.UpdatedAt = f.saved.UpdatedAt
					if got != f.saved {
						t.Errorf("%v progress changed: got %+v want %+v", f.key, got, f.saved)
					}
				}
			}
			if n, err := s.PurgeStale(ctx, tc.before); err != nil || n != 0 {
				t.Errorf("repeated purge: removed %d, error %v", n, err)
			}
			// Retention is global across networks, but removes progress only.
			// Admin copy, metric opt-in and anonymous counters must survive.
			for _, network := range []string{"net-a", "net-b"} {
				copy, exists, err := s.LoadCopy(ctx, network, 1)
				if err != nil || !exists || copy.Intro != network || copy.Revision != 1 || !copy.UpdatedAt.Equal(tc.before) {
					t.Errorf("%s copy changed: %+v, exists=%t, error %v", network, copy, exists, err)
				}
				if enabled, err := s.MetricsEnabled(ctx, network); err != nil || !enabled {
					t.Errorf("%s metrics setting changed: enabled=%t, error %v", network, enabled, err)
				}
				counts, err := s.MetricCounts(ctx, network, 1)
				want := [StepCount]int64{}
				want[Terms] = int64(len(offsets))
				if err != nil || counts != want {
					t.Errorf("%s counters changed: got %v, want %v, error %v", network, counts, want, err)
				}
			}
		})
	}
	t.Run("curriculum_versions", func(t *testing.T) {
		ctx := context.Background()
		s := open(t)
		for _, network := range []string{"net-a", "net-b"} {
			key := Key{network, "same-member"}
			for version, when := range map[int]time.Time{
				1: base.Add(-time.Nanosecond),
				2: base,
			} {
				if _, err := s.SaveProgress(ctx, key, 0, Progress{Version: version}, nil, when); err != nil {
					t.Fatal(err)
				}
			}
		}
		if n, err := s.PurgeStale(ctx, base); err != nil || n != 2 {
			t.Errorf("first purge: removed %d, error %v", n, err)
		}
		for _, network := range []string{"net-a", "net-b"} {
			got, exists, err := s.LoadProgress(ctx, Key{network, "same-member"})
			if err != nil || !exists || got.Version != 2 || !got.UpdatedAt.Equal(base) {
				t.Errorf("%s equal-cutoff curriculum changed: %+v, exists=%t, error %v", network, got, exists, err)
			}
		}
		if n, err := s.PurgeStale(ctx, base.Add(time.Nanosecond)); err != nil || n != 2 {
			t.Errorf("later purge: removed %d, error %v", n, err)
		}
		for _, network := range []string{"net-a", "net-b"} {
			if _, exists, err := s.LoadProgress(ctx, Key{network, "same-member"}); err != nil || exists {
				t.Errorf("%s stale curriculum survived: exists=%t, error %v", network, exists, err)
			}
		}
	})
}

func TestMemPurgeStaleChronology(t *testing.T) {
	purgeCutoffContract(t, func(*testing.T) Store { return NewMemStore() })
}
