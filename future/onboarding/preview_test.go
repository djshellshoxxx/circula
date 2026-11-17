package onboarding

import (
	"math"
	"testing"
)

func TestChecklistAndPractice(t *testing.T) {
	p := Progress{Version: 1}
	var e error
	p, e = p.Complete(Terms)
	if e != nil {
		t.Fatal(e)
	}
	p, _ = p.Complete(Terms)
	s, ok := p.Next()
	if !ok || s != Credit {
		t.Fatal(s)
	}
	r, e := PreviewTrade(0, 0, 100, 20, 80)
	if e != nil || !r.Practice || r.PayerAfter+r.PayeeAfter != 0 {
		t.Fatal(r, e)
	}
	if _, e = PreviewTrade(0, 0, 100, 20, 81); e == nil {
		t.Fatal("limit")
	}
	if _, e = PreviewTrade(0, math.MaxInt64, 100, 0, 1); e == nil {
		t.Fatal("overflow")
	}
	if _, e = PreviewTrade(math.MinInt64, 0, 100, 0, 1); e == nil {
		t.Fatal("underflow")
	}
}
