package domain

import (
	"math"
	"strconv"
	"strings"
)

// MaxDecimals is the platform maximum for the decimals policy (OQ-DOM-6).
const MaxDecimals = 4

// ParseAmount parses a non-negative decimal string such as "25", "25.5" or
// "25.50" into integer minor units. It never uses floating point (DOM-005).
// More fractional digits than decimals, signs, exponents, empty input and
// overflow are rejected. Zero parses successfully; callers enforce >0.
func ParseAmount(s string, decimals int) (int64, error) {
	if decimals < 0 || decimals > MaxDecimals {
		return 0, Invalid("decimals", "out of range")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, Invalid("amount", "required")
	}
	whole, frac, hasDot := strings.Cut(s, ".")
	if whole == "" || (hasDot && frac == "") {
		return 0, Invalid("amount", "not a number")
	}
	if !allDigits(whole) || !allDigits(frac) {
		return 0, Invalid("amount", "not a number")
	}
	if len(frac) > decimals {
		return 0, Invalid("amount", "too many decimal places")
	}
	frac += strings.Repeat("0", decimals-len(frac))
	digits := strings.TrimLeft(whole+frac, "0")
	if digits == "" {
		return 0, nil
	}
	if len(digits) > 19 {
		return 0, Invalid("amount", "too large")
	}
	v, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, Invalid("amount", "too large")
	}
	return v, nil
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// FormatAmount renders minor units as a decimal string with exactly
// `decimals` fractional digits. Handles negative values and math.MinInt64.
func FormatAmount(v int64, decimals int) string {
	neg := v < 0
	var u uint64
	if neg {
		u = uint64(-(v + 1)) + 1
	} else {
		u = uint64(v)
	}
	s := strconv.FormatUint(u, 10)
	if decimals > 0 {
		if len(s) <= decimals {
			s = strings.Repeat("0", decimals-len(s)+1) + s
		}
		s = s[:len(s)-decimals] + "." + s[len(s)-decimals:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// AddChecked adds two int64 values and reports overflow.
func AddChecked(a, b int64) (int64, bool) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}
