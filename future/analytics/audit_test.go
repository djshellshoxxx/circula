package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type sourceFunc func(context.Context, Actor, bool, time.Time) (Snapshot, error)

func (f sourceFunc) Read(ctx context.Context, a Actor, admin bool, cutoff time.Time) (Snapshot, error) {
	return f(ctx, a, admin, cutoff)
}
func auditedService(snap Snapshot, now time.Time) DashboardService {
	return DashboardService{Source: fixtureSource{snap, true}, Now: func() time.Time { return now }, Zone: time.UTC, K: 5}
}
func addEvent(s *Snapshot, t Completed) {
	s.Trades = append(s.Trades, t)
	s.Postings = append(s.Postings, Posting{t.ID + "d", t.Network, t.ID, t.Payer, -t.Amount}, Posting{t.ID + "c", t.Network, t.ID, t.Payee, t.Amount})
}
func TestAuditDominanceCannotBeSplitAndStockComplement(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	for i := range snap.Trades {
		snap.Trades[i].Payer = "a"
		snap.Postings[i*2].Member = "a"
	}
	s := auditedService(snap, now)
	d, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !d.Flows[0].Suppressed || !d.Stock.Suppressed || d.Stock.Positive != 0 {
		t.Fatal("split participant dominance", d, e)
	}
	snap = snapshotFixture(now)
	snap.Balances[0].Amount = -1000
	snap.Balances[1].Amount = 1000
	s = auditedService(snap, now)
	d, e = s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !d.Stock.Suppressed {
		t.Fatal("dominant stock", d, e)
	}
	snap = snapshotFixture(now)
	for i := 0; i < 30; i++ {
		snap.Offers = append(snap.Offers, Offer{Member: "a", Active: true, Updated: now.AddDate(0, 0, -60), Network: "n", ID: fmt.Sprintf("extra-%d", i), Visible: true})
	}
	s = auditedService(snap, now)
	d, e = s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !d.DormantSuppressed || d.DormantOffers != 0 {
		t.Fatal("dominant offer count", d, e)
	}
}
func TestAuditRevisionOnlyUsesReleasedData(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	s := auditedService(snap, now)
	first, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil {
		t.Fatal(e)
	}
	snap.Revision = "global-revision-PRIVATE-OTHER-TENANT"
	s = auditedService(snap, now)
	again, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("internal revision leaked", first, again, e)
	}
	var html bytes.Buffer
	RenderDashboard(&html, again)
	if strings.Contains(html.String(), snap.Revision) {
		t.Fatal("raw database revision exposed")
	}
}
func TestAuditScopedOffersAndBalancesAndCancellation(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for _, mutate := range []func(*Snapshot){func(s *Snapshot) { s.Balances[0].Network = "other" }, func(s *Snapshot) { s.Offers[0].Network = "other" }, func(s *Snapshot) { s.Offers = append(s.Offers, s.Offers[0]) }, func(s *Snapshot) { s.Trades[0].Payer = "unknown"; s.Postings[0].Member = "unknown" }, func(s *Snapshot) { s.Scale = 9 }, func(s *Snapshot) { s.Offers[0].Updated = now.Add(time.Hour) }} {
		snap := snapshotFixture(now)
		mutate(&snap)
		if _, e := auditedService(snap, now).Admin(context.Background(), Actor{"n", "a"}, "month", 1); e == nil {
			t.Fatal("malformed snapshot accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := auditedService(snapshotFixture(now), now)
	s.Source = sourceFunc(func(_ context.Context, _ Actor, _ bool, cut time.Time) (Snapshot, error) {
		snap := snapshotFixture(now)
		snap.AsOf = cut
		cancel()
		return snap, nil
	})
	if _, e := s.Admin(ctx, Actor{"n", "a"}, "month", 1); !errors.Is(e, context.Canceled) {
		t.Fatal("late callback cancellation", e)
	}
	s = auditedService(snapshotFixture(now), now)
	calls := 0
	s.Now = func() time.Time { calls++; return now }
	if _, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1); e != nil || calls != 1 {
		t.Fatal("multiple clock snapshots", calls, e)
	}
}
func TestAuditPartialReversalsAdjustmentsAndOwnOpening(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	snap.Balances = append(snap.Balances, Balance{Member: "system", Network: "n", System: true})
	original := snap.Trades[0]
	original.PostedAt = original.PostedAt.AddDate(0, -1, 0)
	snap.Trades[0] = original
	addEvent(&snap, Completed{Trade: Trade{"partial1", "n", original.Payee, original.Payer, 4, time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC), true}, ReversalOf: original.ID, Kind: "reversal"})
	addEvent(&snap, Completed{Trade: Trade{"partial2", "n", original.Payee, original.Payer, 6, time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), true}, ReversalOf: original.ID, Kind: "reversal"})
	addEvent(&snap, Completed{Trade: Trade{"grant", "n", "system", "a", 3, time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC), true}, Kind: "grant"})
	addEvent(&snap, Completed{Trade: Trade{"adjustment", "n", "a", "system", 2, time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC), true}, Kind: "adjustment"})
	s := auditedService(snap, now)
	own, e := s.Own(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || len(own) != 1 || own[0].Incoming != 13 || own[0].Outgoing != 2 || own[0].Closing != -10 || own[0].Opening != -21 || !own[0].HasPostings {
		t.Fatal(own, e)
	}
	addEvent(&snap, Completed{Trade: Trade{"extra", "n", original.Payee, original.Payer, 1, time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC), true}, ReversalOf: original.ID, Kind: "reversal"})
	if _, e = auditedService(snap, now).Admin(context.Background(), Actor{"n", "a"}, "month", 1); e == nil {
		t.Fatal("over-reversal accepted")
	}
}
func TestAuditExactIntegerOutputAndNoJSOwn(t *testing.T) {
	n := int64(1<<53) + 1
	b, e := json.Marshal(Flow{Gross: n, Reversed: 2, Net: n - 2})
	if e != nil || !bytes.Contains(b, []byte(`"Gross":"9007199254740993"`)) {
		t.Fatal(string(b), e)
	}
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	s := auditedService(snapshotFixture(now), now)
	h := Handler{s, func(*http.Request) (Actor, error) { return Actor{"n", "a"}, nil }}
	for _, test := range []struct {
		path     string
		status   int
		contains string
	}{{"/?view=own&format=html", 200, "My credit trend"}, {"/?view=own&format=csv", 200, "opening,incoming,outgoing,closing"}, {"/?format=json&format=html", 400, "Duplicate query"}, {"/?count=0", 400, "Invalid closed period"}, {"/?format=javascript", 400, "Invalid format"}, {"/?period=%zz", 400, "Invalid query"}} {
		r := httptest.NewRequest("GET", test.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != test.status || !strings.Contains(w.Body.String(), test.contains) {
			t.Fatal(test, w.Code, w.Body.String())
		}
	}
	var page bytes.Buffer
	d, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil {
		t.Fatal(e)
	}
	RenderDashboard(&page, d)
	for _, expected := range []string{"Outstanding balance stock", "Dormant offers", "decimal scale"} {
		if !strings.Contains(page.String(), expected) {
			t.Fatal("missing no-JS content", expected)
		}
	}
}
func TestAuditOrdinarySystemAccountAndNoPostingDistinction(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	snap.Balances = append(snap.Balances, Balance{Member: "z", Network: "n"}, Balance{Member: "system", Network: "n", System: true})
	addEvent(&snap, Completed{Trade: Trade{"grant", "n", "system", "z", 10, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), true}, Kind: "grant"})
	s := auditedService(snap, now)
	d, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || d.Flows[0].Gross != 50 || d.Flows[0].Trades != 5 {
		t.Fatal("grant included in turnover", d, e)
	}
	own, e := s.Own(context.Background(), Actor{"n", "z"}, "month", 2)
	if e != nil || len(own) != 2 || own[0].HasPostings || !own[1].HasPostings {
		t.Fatal(own, e)
	}
	if _, e = s.Own(context.Background(), Actor{"n", "missing"}, "month", 1); e == nil {
		t.Fatal("unresolved own account")
	}
}
func TestAuditPermutationOverflowAndReadConcurrency(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	s := auditedService(snap, now)
	first, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil {
		t.Fatal(e)
	}
	for i, j := 0, len(snap.Balances)-1; i < j; i, j = i+1, j-1 {
		snap.Balances[i], snap.Balances[j] = snap.Balances[j], snap.Balances[i]
	}
	again, e := auditedService(snap, now).Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("input-order effect", e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	snap = snapshotFixture(now)
	snap.Trades[0].Amount = math.MaxInt64
	snap.Postings[0].Delta = -math.MaxInt64
	snap.Postings[1].Delta = math.MaxInt64
	if _, e = auditedService(snap, now).Admin(context.Background(), Actor{"n", "a"}, "month", 1); e == nil {
		t.Fatal("overflow accepted")
	}
}
func FuzzClosedPeriods(f *testing.F) {
	f.Add(uint16(2026), uint8(11), uint8(16), uint8(1), true)
	f.Fuzz(func(t *testing.T, y uint16, m, d, n uint8, weekly bool) {
		year := 1980 + int(y%150)
		kind := "month"
		if weekly {
			kind = "week"
		}
		now := time.Date(year, time.Month(m%12+1), int(d%28+1), 12, 0, 0, 0, time.UTC)
		rows, e := Periods(now, time.UTC, kind, int(n%12+1))
		if e != nil {
			t.Fatal(e)
		}
		for i, p := range rows {
			if !p.From.Before(p.To) || p.To.After(now) || i > 0 && !rows[i-1].To.Equal(p.From) {
				t.Fatal(rows)
			}
		}
	})
}
func TestAuditLaterPartialReversalPeriod(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	originals := append([]Completed(nil), snap.Trades...)
	for i, original := range originals {
		snap.Trades[i].PostedAt = original.PostedAt.AddDate(0, -1, 0)
		addEvent(&snap, Completed{Trade: Trade{ID: "reversal-" + original.ID, Network: "n", Payer: original.Payee, Payee: original.Payer, Amount: 4, PostedAt: original.PostedAt.Add(time.Hour), Completed: true}, ReversalOf: original.ID, Kind: "reversal"})
		snap.Balances[i*2].Amount = -6
		snap.Balances[i*2+1].Amount = 6
	}
	s := auditedService(snap, now)
	d, e := s.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || d.Flows[0].Suppressed || d.Flows[0].Gross != 0 || d.Flows[0].Reversed != 20 || d.Flows[0].Net != -20 || d.Flows[0].Trades != 0 || d.Flows[0].Reversals != 5 || d.Stock.Positive != 30 {
		t.Fatal(d, e)
	}
}
func TestAuditDominanceThresholdDoesNotRoundUp(t *testing.T) {
	if !dominant(map[string]int64{"a": 17}, 21) || dominant(map[string]int64{"a": 16}, 20) {
		t.Fatal("80 percent threshold rounded incorrectly")
	}
}
func TestAuditCSVMetadataFormulaAndWriterFailure(t *testing.T) {
	var b bytes.Buffer
	d := Dashboard{Timezone: "=unsafe()", Scale: 2, Flows: []Flow{{Period: Period{From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}, Gross: 50}}}
	if e := Export(&b, d); e != nil || !strings.Contains(b.String(), "decimal_scale,timezone") || !strings.Contains(b.String(), "'=unsafe()") {
		t.Fatal(b.String(), e)
	}
	if e := Export(brokenWriter{}, d); e == nil {
		t.Fatal("writer error swallowed")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
