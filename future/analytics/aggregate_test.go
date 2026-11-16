package analytics

import (
	"math"
	"testing"
	"time"
)

func TestAggregationSuppressionAndOverflow(t *testing.T) {
	now := time.Unix(100, 0)
	rows := []Trade{{"1", "n", "a", "b", 10, now, true}, {"2", "n", "b", "c", 20, now, true}, {"3", "n", "c", "d", 30, now, true}, {"4", "n", "d", "e", 40, now, true}, {"foreign", "x", "a", "e", 900, now, true}}
	r, e := Aggregate("n", now, now.Add(time.Hour), 5, rows)
	if e != nil || r.GrossTurnover != 100 || r.TradeCount != 4 || r.ActiveTraders != 5 {
		t.Fatal(r, e)
	}
	r, e = Aggregate("n", now, now.Add(time.Hour), 5, rows[:1])
	if e != nil || !r.Suppressed || r.GrossTurnover != 0 || r.TradeCount != 0 || r.ActiveTraders != 0 {
		t.Fatal("suppression leak", r, e)
	}
	rows[0].Amount = math.MaxInt64
	if _, e = Aggregate("n", now, now.Add(time.Hour), 5, rows); e == nil {
		t.Fatal("overflow")
	}
}
