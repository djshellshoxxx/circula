package domain

import (
	"testing"
	"unicode/utf8"
)

// RB-017. Seeds run under plain `go test`; `go test -fuzz=FuzzParseAmount`
// explores further.

func FuzzParseAmount(f *testing.F) {
	for _, s := range []string{"0", "25", "25.5", "25.50", "0.0001", "9223372036854775807", "92233720368547758.07",
		"", ".", "1.", ".5", "-1", "+1", "1e3", "１２", " 7 ", "00012", "1.23456", "99999999999999999999"} {
		f.Add(s, 2)
	}
	f.Add("1.2345", 4)
	f.Add("7", 0)
	f.Fuzz(func(t *testing.T, s string, d int) {
		v, err := ParseAmount(s, d)
		if err != nil {
			return
		}
		if v < 0 {
			t.Fatalf("ParseAmount(%q,%d) = %d negative", s, d, v)
		}
		// Round trip: the canonical rendering parses back to the same value.
		out := FormatAmount(v, d)
		back, err := ParseAmount(out, d)
		if err != nil || back != v {
			t.Fatalf("round trip %q -> %d -> %q -> %d (%v)", s, v, out, back, err)
		}
	})
}

func FuzzFormatAmount(f *testing.F) {
	for _, v := range []int64{0, 1, -1, 99, 100, 1<<63 - 1, -1 << 63} {
		f.Add(v, 2)
	}
	f.Fuzz(func(t *testing.T, v int64, d int) {
		if d < 0 || d > MaxDecimals {
			return
		}
		s := FormatAmount(v, d)
		if v >= 0 {
			back, err := ParseAmount(s, d)
			if err != nil || back != v {
				t.Fatalf("FormatAmount(%d,%d) = %q does not parse back (%d, %v)", v, d, s, back, err)
			}
		}
	})
}

func FuzzFoldName(f *testing.F) {
	for _, s := range []string{"Émile", "EMILE", "İstanbul", "ıstanbul", "ǅ", "ß", "", "á", "\xff\xfe", "Ｆｕｌｌ", "ﬃ"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := FoldName(s)
		if utf8.ValidString(s) && !utf8.ValidString(out) {
			t.Fatalf("FoldName(%q) = invalid UTF-8 %q", s, out)
		}
	})
}
