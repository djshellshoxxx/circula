package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-09 / API-034 / API-035: page beyond 100 policy versions without losing
// predecessor values or withdrawal context at a page boundary.
func TestGovernancePolicyLogPagination(t *testing.T) {
	w := newWorld(t)
	const total = 107
	start := w.env.Clock.Now()
	err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error {
		g := tx.(store.GovRepo)
		for version := 2; version <= total; version++ {
			p := w.net.Policy
			p.CurrencyName = fmt.Sprintf("unit%d", version)
			at := start.Add(time.Duration(version) * time.Minute)
			effective := at
			if version == 53 {
				// The next immediate version withdraws this scheduled one.
				effective = at.Add(5 * 24 * time.Hour)
			}
			if err := g.AppendPolicyNotice(w.net.ID, version, p, w.adminM.ID, "policy fixture", at, effective, 0, ""); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	w.env.Clock.Advance(200 * time.Minute)
	want := testutil.Must(w.env.Svc.PolicyLog(w.env.Ctx, w.alice, w.net.ID))
	if len(want) != total {
		t.Fatalf("fixture has %d versions, want %d", len(want), total)
	}
	first := w.get(w.aliceT, "/network/policy-log")
	wantStatus(t, first, 200)
	if got := decode[page](t, first); len(got.Data) != 25 || got.NextCursor == nil {
		t.Fatalf("default policy page: %+v", got)
	}
	jsonText := func(v any) string {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	var cursor string
	var received []map[string]any
	for round := 0; round < 10; round++ {
		r := w.get(w.aliceT, "/network/policy-log?limit=37&cursor="+url.QueryEscape(cursor))
		wantStatus(t, r, 200)
		pg := decode[page](t, r)
		if len(pg.Data) > 37 {
			t.Fatalf("policy page exceeds requested size: %d", len(pg.Data))
		}
		received = append(received, pg.Data...)
		if round == 0 {
			// A newer version appears between requests. Keyset traversal must
			// still visit each initial version exactly once.
			err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error {
				at := w.env.Clock.Now().Add(time.Minute)
				return tx.(store.GovRepo).AppendPolicyNotice(w.net.ID, total+1, w.net.Policy, w.adminM.ID,
					"concurrent change", at, at, 0, "")
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if pg.NextCursor == nil {
			break
		}
		if *pg.NextCursor == cursor {
			t.Fatal("policy cursor did not advance")
		}
		cursor = *pg.NextCursor
	}
	if len(received) != len(want) {
		t.Fatalf("received %d policy versions, want %d", len(received), len(want))
	}
	for i, got := range received {
		e := want[i]
		if got["version"] != float64(e.Version) || got["withdrawn"] != e.Withdrawn || got["pending"] != e.Pending ||
			jsonText(got["changes"]) != jsonText(e.Changes) {
			t.Errorf("policy %d = %v, want %+v", i, got, e)
		}
		if len(e.Previous) > 0 && jsonText(got["previous"]) != jsonText(e.Previous) {
			t.Errorf("policy %d previous = %v, want %v", e.Version, got["previous"], e.Previous)
		}
	}
}

// A2-09: more than 200 challenges remain reachable; tied and backwards
// timestamps use the same stable (created_at, id) keyset as the SQL order.
func TestGovernanceChallengePagination(t *testing.T) {
	w := newWorld(t)
	const total = 213
	var want []store.Challenge
	insert := func(c store.Challenge) {
		t.Helper()
		if err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error { return tx.InsertChallenge(c) }); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < total; i++ {
		c := store.Challenge{ID: fmt.Sprintf("018f1000-0000-7000-8000-%012x", i), NetworkID: w.net.ID,
			MemberID: w.aliceM.ID, Kind: service.ChallengeCreditLimit, Reason: "fixture", Status: "requested",
			CreatedAt: w.env.Clock.Now().Add(-time.Duration(i%9) * time.Second)}
		insert(c)
		want = append(want, c)
	}
	// Another person's challenge must remain outside the member's pages.
	insert(store.Challenge{ID: "018f2000-0000-7000-8000-000000000000", NetworkID: w.net.ID,
		MemberID: w.bobM.ID, Kind: service.ChallengeCreditLimit, Reason: "other member", Status: "requested", CreatedAt: w.env.Clock.Now()})
	sort.Slice(want, func(i, j int) bool {
		if want[i].CreatedAt.Equal(want[j].CreatedAt) {
			return want[i].ID > want[j].ID
		}
		return want[i].CreatedAt.After(want[j].CreatedAt)
	})
	var cursor string
	var received []string
	for round := 0; round < 10; round++ {
		r := w.get(w.aliceT, "/challenges?status=requested&limit=41&cursor="+url.QueryEscape(cursor))
		wantStatus(t, r, 200)
		pg := decode[page](t, r)
		if len(pg.Data) > 41 {
			t.Fatalf("challenge page exceeds requested size: %d", len(pg.Data))
		}
		for _, c := range pg.Data {
			if c["member_id"] != w.aliceM.ID || c["status"] != "requested" {
				t.Errorf("challenge escaped filters: %v", c)
			}
			received = append(received, c["id"].(string))
		}
		if round == 0 {
			insert(store.Challenge{ID: "018f3000-0000-7000-8000-000000000000", NetworkID: w.net.ID,
				MemberID: w.aliceM.ID, Kind: service.ChallengeCreditLimit, Reason: "new arrival", Status: "requested",
				CreatedAt: w.env.Clock.Now().Add(time.Minute)})
		}
		if pg.NextCursor == nil {
			break
		}
		if *pg.NextCursor == cursor {
			t.Fatal("challenge cursor did not advance")
		}
		cursor = *pg.NextCursor
	}
	if len(received) != total {
		t.Fatalf("received %d challenges, want %d", len(received), total)
	}
	for i, id := range received {
		if id != want[i].ID {
			t.Errorf("challenge %d = %s, want %s", i, id, want[i].ID)
		}
	}
	if got := testutil.Must(w.env.Svc.ListChallenges(w.env.Ctx, w.alice, w.net.ID, "")); len(got) != 200 {
		t.Errorf("legacy service wrapper returned %d rows, want 200", len(got))
	}
}

func TestGovernancePaginationRejectsInvalidAndReboundCursors(t *testing.T) {
	w := newWorld(t)
	if err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error {
		if err := tx.(store.GovRepo).AppendPolicyNotice(w.net.ID, 2, w.net.Policy, w.adminM.ID,
			"fixture", w.env.Clock.Now(), w.env.Clock.Now(), 0, ""); err != nil {
			return err
		}
		for i := 0; i < 2; i++ {
			if err := tx.InsertChallenge(store.Challenge{ID: fmt.Sprintf("018f4000-0000-7000-8000-%012x", i),
				NetworkID: w.net.ID, MemberID: w.aliceM.ID, Kind: service.ChallengeCreditLimit, Reason: "fixture",
				Status: "requested", CreatedAt: w.env.Clock.Now()}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cursors := map[string]string{}
	for _, route := range []string{"/network/policy-log", "/challenges"} {
		for _, query := range []string{"limit=0", "limit=101", "limit=no", "unexpected=1"} {
			wantProblem(t, w.get(w.adminT, route+"?"+query), 400, "invalid-query")
		}
		wantProblem(t, w.get(w.adminT, route+"?cursor=broken"), 400, "invalid-cursor")
		r := w.get(w.adminT, route+"?limit=1")
		wantStatus(t, r, 200)
		pg := decode[page](t, r)
		if len(pg.Data) != 1 || pg.NextCursor == nil {
			t.Fatalf("cursor fixture %s: %+v", route, pg)
		}
		cursor := *pg.NextCursor
		cursors[route] = cursor
		wantProblem(t, w.get(w.adminT, route+"?cursor="+url.QueryEscape(cursor+"x")), 400, "invalid-cursor")
		wantProblem(t, w.get(w.aliceT, route+"?cursor="+url.QueryEscape(cursor)), 400, "invalid-cursor")
	}
	wantProblem(t, w.get(w.adminT, "/challenges?status=other"), 400, "invalid-query")
	wantProblem(t, w.get(w.adminT, "/challenges?status=decided&cursor="+url.QueryEscape(cursors["/challenges"])), 400, "invalid-cursor")
	wantProblem(t, w.get(w.adminT, "/challenges?cursor="+url.QueryEscape(cursors["/network/policy-log"])), 400, "invalid-cursor")
	other, _ := w.env.Network(w.admin, "other", nil)
	for route, cursor := range cursors {
		wantProblem(t, w.do(req{method: "GET", path: "/api/v1" + route + "?cursor=" + url.QueryEscape(cursor),
			token: w.adminT, hdr: map[string]string{"X-Network": other.ID}}), 400, "invalid-cursor")
	}
}
