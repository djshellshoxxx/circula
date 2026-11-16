package favorites

import "testing"

func TestFreshUnpricedRepeats(t *testing.T) {
	a := Member{"a", "n", true, true, false}
	b := Member{"b", "n", true, true, false}
	h := HistoricalTrade{"n", "a", "b", "bike", true}
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
