package matching

import "testing"

func TestOptInScopeAndBlocks(t *testing.T) {
	rows := []Listing{{"o", "n", "a", "offer", "bike", true, true, true}, {"r", "n", "b", "request", "bike", true, true, true}, {"x", "other", "c", "request", "bike", true, true, true}, {"hidden", "n", "d", "request", "bike", true, false, true}, {"out", "n", "e", "request", "bike", true, true, false}}
	p, e := Suggest("n", rows, 10, func(a, b string) bool { return false })
	if e != nil || len(p) != 1 || p[0].Request != "r" {
		t.Fatal(p, e)
	}
	p, e = Suggest("n", rows, 10, func(a, b string) bool { return a == "r" })
	if e != nil || len(p) != 0 {
		t.Fatal("reverse block", p, e)
	}
	if _, e = Suggest("n", rows, 10, nil); e == nil {
		t.Fatal("missing policy")
	}
}
