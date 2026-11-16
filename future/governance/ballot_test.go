package governance

import (
	"testing"
	"time"
)

func TestBallotBoundaries(t *testing.T) {
	now := time.Unix(100, 0)
	p := Poll{"n", "p", now, now.Add(time.Hour), false, map[string]bool{"a": true, "b": true}}
	b, e := Cast(p, "n", "a", true, Yes, now, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Cast(p, "n", "a", true, No, now, []Ballot{b}); e == nil {
		t.Fatal("duplicate")
	}
	if _, e = Cast(p, "x", "a", true, Yes, now, nil); e == nil {
		t.Fatal("tenant")
	}
	if _, e = Cast(p, "n", "a", true, Yes, p.Closes, nil); e == nil {
		t.Fatal("late")
	}
	o, e := Tally(p, p.Closes, []Ballot{b})
	if e != nil || o.Yes != 1 || o.Eligible != 2 || !o.Advisory {
		t.Fatal(o, e)
	}
	if _, e = Tally(p, p.Closes, []Ballot{b, b}); e == nil {
		t.Fatal("double tally")
	}
}
