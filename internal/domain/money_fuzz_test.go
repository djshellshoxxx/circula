package domain

import (
	"math"
	"strings"
	"testing"
)

// QA-005: ParseAmount never panics, never wraps, and agrees with FormatAmount.
func FuzzParseAmount(f *testing.F) {
	for _, s := range []string{"", "0", "0.00", "25", "25.5", "25.50", "-1", "+1", "1e3", ".5", "5.", "1,5", " 7 ",
		"9223372036854775807", "92233720368547758.07", "92233720368547758.08", "99999999999999999999",
		"00000000000000000000000001", "1.23456", "\x00", "١٢٣", "1_000", "0x10", "1.2.3", "--1", "9223372036854775808"} {
		f.Add(s, 2)
		f.Add(s, 0)
		f.Add(s, MaxDecimals)
	}
	f.Fuzz(func(t *testing.T, s string, dec int) {
		dec %= 8 // exercises out-of-range decimals too
		v, err := ParseAmount(s, dec)
		if dec < 0 || dec > MaxDecimals {
			if err == nil {
				t.Fatalf("decimals %d accepted", dec)
			}
			return
		}
		if err != nil {
			if v != 0 {
				t.Fatalf("error with non-zero value %d for %q", v, s)
			}
			return
		}
		if v < 0 {
			t.Fatalf("negative result %d for %q", v, s)
		}
		// accepted input is only digits and at most one dot: no sign, exponent or separators
		for _, r := range strings.TrimSpace(s) {
			if (r < '0' || r > '9') && r != '.' {
				t.Fatalf("accepted %q", s)
			}
		}
		out := FormatAmount(v, dec)
		back, err := ParseAmount(out, dec)
		if err != nil || back != v {
			t.Fatalf("roundtrip %q -> %d -> %q -> %d (%v)", s, v, out, back, err)
		}
		if dec > 0 {
			i := strings.IndexByte(out, '.')
			if i < 1 || len(out)-i-1 != dec {
				t.Fatalf("format %q does not have %d decimals", out, dec)
			}
		} else if strings.Contains(out, ".") {
			t.Fatalf("format %q has a dot at 0 decimals", out)
		}
	})
}

// FormatAmount of any int64 (incl. MinInt64) is well formed, and non-negative values roundtrip.
func FuzzFormatAmount(f *testing.F) {
	for _, v := range []int64{0, 1, -1, 99, 100, -100, math.MaxInt64, math.MinInt64, math.MinInt64 + 1} {
		f.Add(v, 2)
	}
	f.Fuzz(func(t *testing.T, v int64, dec int) {
		dec = ((dec % 5) + 5) % 5
		s := FormatAmount(v, dec)
		if strings.HasPrefix(s, "-") != (v < 0) {
			t.Fatalf("sign of %d rendered %q", v, s)
		}
		body := strings.TrimPrefix(s, "-")
		if dec > 0 {
			i := strings.IndexByte(body, '.')
			if i < 1 || len(body)-i-1 != dec {
				t.Fatalf("%d/%d -> %q", v, dec, s)
			}
		}
		if v >= 0 {
			back, err := ParseAmount(s, dec)
			if err != nil || back != v {
				t.Fatalf("%d -> %q -> %d (%v)", v, s, back, err)
			}
		}
	})
}

func FuzzAddChecked(f *testing.F) {
	f.Add(int64(math.MaxInt64), int64(1))
	f.Add(int64(math.MinInt64), int64(-1))
	f.Add(int64(5), int64(-5))
	f.Fuzz(func(t *testing.T, a, b int64) {
		got, ok := AddChecked(a, b)
		wantOK := (b <= 0 || a <= math.MaxInt64-b) && (b >= 0 || a >= math.MinInt64-b)
		if ok != wantOK {
			t.Fatalf("%d+%d ok=%v", a, b, ok)
		}
		if ok && got != a+b {
			t.Fatalf("%d+%d=%d", a, b, got)
		}
		if !ok && got != 0 {
			t.Fatalf("overflow returned %d", got)
		}
	})
}
