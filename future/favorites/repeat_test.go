package favorites

import "testing"

func TestFreshUnpricedRepeats(t *testing.T) {
	a := Member{ID: "a", Network: "n", Active: true, Visible: true}
	b := Member{ID: "b", Network: "n", Active: true, Visible: true}
	h := HistoricalTrade{Network: "n", Payer: "a", Payee: "b", Description: "bike", Completed: true}
	x, e := Repeat(a, b, h)
	if e != nil {
		t.Fatal(e)
	}
	y, e := Repeat(a, b, h)
	if e != nil || x.Intent == y.Intent || x.Amount != nil || y.Amount != nil {
		t.Fatal(x, y, e)
	}
	b.Network = "other"
	if _, e = Repeat(a, b, h); e == nil {
		t.Fatal("cross network")
	}
	b.Network = "n"
	b.Blocked = true
	if _, e = Repeat(a, b, h); e == nil {
		t.Fatal("blocked")
	}
	if _, e = Save(a, a, ""); e == nil {
		t.Fatal("self favorite")
	}
}
