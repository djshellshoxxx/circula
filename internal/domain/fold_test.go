package domain

import "testing"

func TestFoldName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Émile", "emile"}, {"EMILE", "emile"}, {"émile", "emile"}, // precomposed, upper, combining mark
		{"İstanbul", "istanbul"}, {"ISTANBUL", "istanbul"}, {"ıstanbul", "istanbul"}, // Turkish dotted/dotless i
		{"Straße", "strasse"}, {"STRASSE", "strasse"},
		{"Ångström", "angstrom"}, {"Ｆｕｌｌ", "full"}, // compatibility forms
		{"Ñandú", "nandu"}, {"Łukasz", "łukasz"}, // ł has no decomposition: kept, only case-folded
		{"", ""}, {"日本語", "日本語"}, {"ﬁsh", "fish"},
	} {
		if got := FoldName(c.in); got != c.want {
			t.Errorf("FoldName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if FoldName("a") == FoldName("b") {
		t.Error("distinct letters folded together")
	}
}
