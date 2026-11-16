package areas

import "testing"

func TestOptionalScopeAndVisibility(t *testing.T) {
	rows := []Area{{"n", "north", "North", true}, {"x", "foreign", "Elsewhere", true}, {"n", "hidden", "Hidden", false}}
	a, e := Resolve("n", "", rows)
	if a != nil || e != nil {
		t.Fatal(a, e)
	}
	a, e = Resolve("n", "north", rows)
	if e != nil || a.Label != "North" {
		t.Fatal(a, e)
	}
	for _, id := range []string{"foreign", "hidden", "unknown"} {
		if _, e = Resolve("n", id, rows); e == nil {
			t.Fatal(id)
		}
	}
	if e = Validate(Area{"n", "x", "Bad\nLabel", true}); e == nil {
		t.Fatal("label")
	}
}
