package analytics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// This fixture exercises the complete verified snapshot path, not an unchecked
// aggregation helper. Five evenly represented pairs avoid dominance suppression.
func performanceService(events int) DashboardService {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	base := append([]Completed(nil), snap.Trades...)
	snap.Trades = make([]Completed, 0, events)
	snap.Postings = make([]Posting, 0, events*2)
	for i := range snap.Balances {
		snap.Balances[i].Amount *= int64(events / 5)
	}
	for i := 0; i < events; i++ {
		row := base[i%5]
		row.ID = fmt.Sprintf("trade-%d", i)
		snap.Trades = append(snap.Trades, row)
		snap.Postings = append(snap.Postings,
			Posting{row.ID + "d", "n", row.ID, row.Payer, -row.Amount},
			Posting{row.ID + "c", "n", row.ID, row.Payee, row.Amount})
	}
	return DashboardService{Source: fixtureSource{snap, true}, Now: func() time.Time { return now }, Zone: time.UTC, K: 5}
}

func checkPerformanceDashboard(t testing.TB, result Dashboard, events int) {
	t.Helper()
	if len(result.Flows) != 12 {
		t.Fatalf("unexpected period count %d", len(result.Flows))
	}
	flow := result.Flows[11]
	want := int64(events * 10)
	if flow.Suppressed || flow.Trades != events || flow.Traders != 10 || flow.Gross != want || flow.Net != want || flow.Reversed != 0 {
		t.Fatalf("incorrect large-fixture flow: %+v", flow)
	}
	if result.Stock.Suppressed || result.Stock.Positive != want || result.Stock.Negative != -want || result.DormantOffers != 5 {
		t.Fatalf("incorrect large-fixture stock/dormant result: %+v / %d", result.Stock, result.DormantOffers)
	}
}

func TestPerformanceFixtureAccounting(t *testing.T) {
	s := performanceService(100)
	d, err := s.Admin(context.Background(), Actor{"n", "a"}, "month", 12)
	if err != nil {
		t.Fatal(err)
	}
	checkPerformanceDashboard(t, d, 100)
	own, err := s.Own(context.Background(), Actor{"n", "a"}, "month", 12)
	if err != nil || len(own) != 12 || own[11].Outgoing != 200 || own[11].Opening != 0 || own[11].Closing != -200 {
		t.Fatalf("incorrect own trend: %+v, %v", own, err)
	}
}

func BenchmarkAdminVerifiedSnapshot(b *testing.B) {
	for _, events := range []int{10000, 100000} {
		b.Run(fmt.Sprintf("events-%d/periods-12", events), func(b *testing.B) {
			s := performanceService(events)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d, err := s.Admin(context.Background(), Actor{"n", "a"}, "month", 12)
				if err != nil {
					b.Fatal(err)
				}
				checkPerformanceDashboard(b, d, events)
			}
		})
	}
}
