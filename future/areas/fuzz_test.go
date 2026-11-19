package areas

import (
	"testing"
	"unicode/utf8"
)

func FuzzValidate(f *testing.F) {
	for _, s := range []string{"North", "<script>", "\u202e", "\u2028", string([]byte{255})} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, label string) {
		a := Area{"n", "id", label, true}
		e := Validate(a)
		if e == nil && (!utf8.ValidString(label) || utf8.RuneCountInString(label) > 100) {
			t.Fatal("accepted invalid label")
		}
	})
}
