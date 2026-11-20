package analytics

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fixtureSource struct {
	snap       Snapshot
	allowAdmin bool
}

func (f fixtureSource) Read(_ context.Context, a Actor, admin bool, cutoff time.Time) (Snapshot, error) {
	if a.Network != "n" || a.Member == "" || admin && !f.allowAdmin {
		return Snapshot{}, errors.New("denied")
	}
	snap := f.snap
	snap.AsOf = cutoff
	return snap, nil
}
func snapshotFixture(now time.Time) Snapshot {
	s := Snapshot{Revision: "r1", AsOf: now}
	at := time.Date(now.Year(), now.Month()-1, 15, 12, 0, 0, 0, now.Location())
	for i := 0; i < 5; i++ {
		a, b := string(rune('a'+i)), string(rune('f'+i))
		id := a + b
		s.Trades = append(s.Trades, Completed{Trade: Trade{id, "n", a, b, 10, at, true}})
		s.Postings = append(s.Postings, Posting{id + "d", "n", id, a, -10}, Posting{id + "c", "n", id, b, 10})
		s.Balances = append(s.Balances, Balance{Member: a, Amount: -10, Network: "n"}, Balance{Member: b, Amount: 10, Network: "n"})
		s.Offers = append(s.Offers, Offer{Member: a, Active: true, Updated: now.AddDate(0, 0, -60), Network: "n", ID: "offer-" + a, Visible: true})
	}
	return s
}
func TestDashboardLedgerStockFlowAndOwn(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	service := DashboardService{Source: fixtureSource{snap, true}, Now: func() time.Time { return now }, Zone: time.UTC, K: 5}
	d, e := service.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || len(d.Flows) != 1 || d.Flows[0].Gross != 50 || d.Flows[0].Trades != 5 || d.Stock.Positive != 50 || d.Stock.Negative != -50 || d.Stock.Suppressed || d.DormantOffers != 5 {
		t.Fatal(d, e)
	}
	own, e := service.Own(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || len(own) != 1 || own[0].Outgoing != 10 || own[0].Incoming != 0 || own[0].Trades != 1 {
		t.Fatal(own, e)
	}
	var b bytes.Buffer
	d.Revision = "<script>"
	if e = RenderDashboard(&b, d); e != nil || strings.Contains(b.String(), "<script>") {
		t.Fatal(b.String(), e)
	}
	b.Reset()
	if e = Export(&b, d); e != nil || !strings.Contains(b.String(), "50,0,50,5,10") {
		t.Fatal(b.String(), e)
	}
}
func TestReversalAndSuppression(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	snap := snapshotFixture(now)
	original := snap.Trades[0]
	rev := Trade{"rev", "n", original.Payee, original.Payer, 10, original.PostedAt.Add(time.Hour), true}
	snap.Trades = append(snap.Trades, Completed{Trade: rev, ReversalOf: original.ID})
	snap.Postings = append(snap.Postings, Posting{"rd", "n", "rev", rev.Payer, -10}, Posting{"rc", "n", "rev", rev.Payee, 10})
	service := DashboardService{Source: fixtureSource{snap, true}, Now: func() time.Time { return now }, Zone: time.UTC, K: 5}
	d, e := service.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !d.Flows[0].Suppressed || d.Flows[0].Net != 0 || d.Flows[0].Reversed != 0 {
		t.Fatal(d, e)
	}
	snap.Trades = snap.Trades[:1]
	snap.Postings = snap.Postings[:2]
	service.Source = fixtureSource{snap, true}
	d, e = service.Admin(context.Background(), Actor{"n", "a"}, "month", 1)
	if e != nil || !d.Flows[0].Suppressed || d.Flows[0].Gross != 0 {
		t.Fatal(d, e)
	}
}
func TestAuthLedgerAndPeriodFailures(t *testing.T) {
	zone, e := time.LoadLocation("America/Vancouver")
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, zone)
	periods, e := Periods(now, zone, "week", 3)
	if e != nil || periods[2].To.Weekday() != time.Monday || !periods[0].To.Equal(periods[1].From) {
		t.Fatal(periods, e)
	}
	snap := snapshotFixture(now)
	service := DashboardService{Source: fixtureSource{snap, false}, Now: func() time.Time { return now }, Zone: zone, K: 5}
	if _, e = service.Admin(context.Background(), Actor{"n", "a"}, "month", 1); e == nil {
		t.Fatal("ordinary admin access")
	}
	snap.Postings[0].Delta = -9
	service.Source = fixtureSource{snap, true}
	if _, e = service.Admin(context.Background(), Actor{"n", "a"}, "month", 1); e == nil {
		t.Fatal("ledger mismatch")
	}
	if _, e = Periods(now, zone, "day", 1); e == nil {
		t.Fatal("arbitrary slices")
	}
}
