package web

import "testing"

func TestUnitForPlural(t *testing.T) {
	for _, c := range []struct {
		name string
		v    int64
		dec  int
		want string
	}{
		{"Credit", 1, 0, "Credit"}, {"Credit", -1, 0, "Credit"}, {"Credit", 0, 0, "Credits"},
		{"Credit", 10, 1, "Credit"}, {"Credit", -10, 1, "Credit"}, {"Credit", 11, 1, "Credits"},
		{"Credit", 100, 2, "Credit"}, {"Credit", -100, 2, "Credit"}, {"Credit", 250, 2, "Credits"},
		{"Credit", 10000, 4, "Credit"}, {"Credit", -10000, 4, "Credit"}, {"Credit", 10001, 4, "Credits"},
		{"Credit", 0, 4, "Credits"}, {"Credit", 9007199254740993, 2, "Credits"},
		{"Credits", 250, 2, "Credits"}, {"Hours", 100, 2, "Hours"}, {"HOURS", 250, 2, "HOURS"},
		{"Zoë 100% $&", 250, 2, "Zoë 100% $&s"}, {"<img src=x>", 250, 2, "<img src=x>s"},
	} {
		if got := unitFor(c.name, c.v, c.dec); got != c.want {
			t.Errorf("unitFor(%q,%d,%d)=%q want %q", c.name, c.v, c.dec, got, c.want)
		}
	}
}
