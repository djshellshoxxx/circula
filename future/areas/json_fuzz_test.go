package areas

import (
	"testing"
	"unicode/utf8"
)

func FuzzStrictRequest(f *testing.F) {
	for _, s := range []string{`{"ID":"a","Label":"North"}`, `{"ID":"a","id":"b"}`, `{"Label":"\ud800"}`, `{"Label":"\ud83d\ude00"}`, `null`, "{\"ID\":\"" + string([]byte{255}) + "\"}"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var out struct{ ID, Label string }
		e := strictJSON([]byte(s), &out)
		if e == nil && !utf8.ValidString(s) {
			t.Fatal("accepted invalid Unicode")
		}
	})
}
