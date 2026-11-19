package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestRecoveredCleanupChronologicalBoundaries(t *testing.T) {
	for _, cutoff := range []string{
		"2026-01-01T12:00:00Z",
		"2026-01-01T12:00:00.5Z",
		"2026-01-01T12:00:00.500000001Z",
		"2026-01-01T00:00:00Z",
		"2025-12-31T23:59:59.999999999Z",
	} {
		t.Run(cutoff, func(t *testing.T) {
			s, actor := fixture(t, filepath.Join(t.TempDir(), "retention.db"))
			ctx := context.Background()
			before := at(cutoff)
			samples := []time.Time{
				before.Add(-time.Second), before.Add(-500 * time.Millisecond), before.Add(-time.Nanosecond),
				before, before.Add(time.Nanosecond), before.Add(500 * time.Millisecond), before.Add(time.Second),
			}
			for _, network := range []string{actor.Network, "other"} {
				r := create(t, s, Actor{network, actor.Member}, "fixture")
				response, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				for i, created := range samples {
					id, stamp := fmt.Sprintf("retention-%d", i), created.UTC().Format(time.RFC3339Nano)
					if _, err = s.DB.Exec(`INSERT INTO future_template_reminders(network,id,owner,template,created) VALUES(?,?,?,?,?)`, network, id, actor.Member, r.ID, stamp); err != nil {
						t.Fatal(err)
					}
					if _, err = s.DB.Exec(`INSERT INTO future_template_audit(network,owner,template,action,at) VALUES(?,?,?,?,?)`, network, actor.Member, r.ID, id, stamp); err != nil {
						t.Fatal(err)
					}
					if _, err = s.DB.Exec(`INSERT INTO future_template_operations(network,owner,key,fingerprint,response,created) VALUES(?,?,?,?,?,?)`, network, actor.Member, id, digest(id), string(response), stamp); err != nil {
						t.Fatal(err)
					}
				}
				if _, err = s.DB.Exec(`INSERT INTO future_template_firings(network,template,window,scheduled,state) VALUES(?,?,?,?,?)`, network, r.ID, "retention-tombstone", samples[0].UTC().Format(time.RFC3339Nano), "reminder"); err != nil {
					t.Fatal(err)
				}
			}
			// The caller may supply any zone; comparison is by the exact instant.
			if err := s.Cleanup(ctx, actor.Network, before.In(time.FixedZone("cutoff", 90*60))); err != nil {
				t.Fatal(err)
			}
			for _, network := range []string{actor.Network, "other"} {
				for i, created := range samples {
					id := fmt.Sprintf("retention-%d", i)
					wantExpired := network == actor.Network && created.Before(before)
					for _, table := range []struct{ name, key string }{{"future_template_reminders", "id"}, {"future_template_audit", "action"}} {
						var remaining bool
						if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM `+table.name+` WHERE network=? AND `+table.key+`=?)`, network, id).Scan(&remaining); err != nil {
							t.Fatal(err)
						}
						if remaining == wantExpired {
							t.Errorf("%s network=%s created=%s cutoff=%s remaining=%v wantExpired=%v", table.name, network, created.Format(time.RFC3339Nano), cutoff, remaining, wantExpired)
						}
					}
					var response, fingerprint string
					if err := s.DB.QueryRow(`SELECT response,fingerprint FROM future_template_operations WHERE network=? AND owner=? AND key=?`, network, actor.Member, id).Scan(&response, &fingerprint); err != nil {
						t.Fatalf("operation tombstone lost: %v", err)
					}
					if (response == "") != wantExpired || fingerprint != digest(id) {
						t.Errorf("operation network=%s created=%s cutoff=%s retired=%v wantExpired=%v fingerprint=%s", network, created.Format(time.RFC3339Nano), cutoff, response == "", wantExpired, fingerprint)
					}
				}
				var tombstones int
				if err := s.DB.QueryRow(`SELECT count(*) FROM future_template_firings WHERE network=? AND window='retention-tombstone'`, network).Scan(&tombstones); err != nil || tombstones != 1 {
					t.Fatalf("firing tombstone changed: count=%d err=%v", tombstones, err)
				}
			}
		})
	}
}

func TestRecoveredInboxChronologicalOrder(t *testing.T) {
	for _, times := range [][]string{
		{"2026-11-16T12:00:00Z", "2026-11-16T12:00:00.000000001Z", "2026-11-16T12:00:00.5Z", "2026-11-16T12:00:00.5Z"},
		{"2026-11-16T12:00:00.5Z", "2026-11-16T12:00:00.500000001Z", "2026-11-16T12:00:00.500000002Z", "2026-11-16T12:00:00.500000002Z"},
	} {
		t.Run(times[0], func(t *testing.T) {
			s, actor := fixture(t, filepath.Join(t.TempDir(), "inbox.db"))
			ctx := context.Background()
			now := at(times[0])
			s.Now = func() time.Time { return now }
			var want []Reminder
			for i, stamp := range times {
				now = at(stamp)
				r := create(t, s, actor, fmt.Sprintf("inbox-%d", i))
				result, err := s.RunDue(ctx, actor.Network, "", 20)
				if err != nil || result.Fired != 1 {
					t.Fatal(result, err)
				}
				window, due, err := r.Calendar.Window(actor.Network, r.ID, now)
				if err != nil || !due {
					t.Fatal(window, due, err)
				}
				want = append(want, Reminder{ID: window.Key, Network: actor.Network, Owner: actor.Member, Template: r.ID, Scheduled: window.Scheduled, Created: now})
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].Created.Equal(want[j].Created) {
					return want[i].ID < want[j].ID
				}
				return want[i].Created.After(want[j].Created)
			})
			// A newer reminder in another tenant must not affect the result.
			now = now.Add(time.Second)
			create(t, s, Actor{"other", actor.Member}, "foreign")
			if result, err := s.RunDue(ctx, "other", "", 20); err != nil || result.Fired != 1 {
				t.Fatal(result, err)
			}
			for _, limit := range []int{1, 2, 100} {
				rows, err := s.Inbox(ctx, actor, limit)
				count := min(limit, len(want))
				if err != nil || len(rows) != count {
					t.Fatalf("limit=%d count=%d want=%d err=%v", limit, len(rows), count, err)
				}
				for i, row := range rows {
					if row.ID != want[i].ID || row.Template != want[i].Template || !row.Created.Equal(want[i].Created) {
						t.Errorf("limit=%d row=%d got=%+v want=%+v", limit, i, row, want[i])
					}
				}
			}
			if _, err := s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network=? AND member=?`, actor.Network, actor.Member); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Inbox(ctx, actor, 1); !errors.Is(err, ErrDenied) {
				t.Fatal("inbox stopped checking current authorization", err)
			}
		})
	}
}
