package web

import "testing"

func TestUnitForPlural(t *testing.T) {
	for _, c := range []struct {
		name string
		v    int64
		want string
	}{{"Credit", 100, "Credit"}, {"Credit", 250, "Credits"}, {"Credits", 250, "Credits"}, {"Hours", 100, "Hours"}} {
		if got := unitFor(c.name, c.v, 2); got != c.want {
			t.Errorf("unitFor(%q,%d)=%q want %q", c.name, c.v, got, c.want)
		}
	}
}
