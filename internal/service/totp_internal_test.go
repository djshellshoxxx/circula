package service

import (
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B (SHA-1, secret "12345678901234567890"); the RFC lists
// 8-digit values, the 6-digit code is their last six digits.
func TestTOTPRFC6238Vectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	for _, c := range []struct {
		unix int64
		want string
	}{{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"}, {1234567890, "005924"}, {2000000000, "279037"}, {20000000000, "353130"}} {
		if got := hotp(secret, uint64(c.unix/30)); got != c.want { //nolint:gosec // test constants are positive
			t.Errorf("t=%d: got %s want %s", c.unix, got, c.want)
		}
	}
}

func TestMatchTOTPSkewAndReplay(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1111111109, 0)
	cur := totpStep(now)
	for _, c := range []struct {
		name string
		step int64
		last int64
		ok   bool
	}{
		{"current", cur, 0, true}, {"previous", cur - 1, 0, true}, {"next", cur + 1, 0, true},
		{"two back", cur - 2, 0, false}, {"two ahead", cur + 2, 0, false},
		{"replay same step", cur, cur, false}, {"older than last", cur - 1, cur, false},
	} {
		code := hotp(secret, uint64(c.step)) //nolint:gosec // positive
		step, ok := matchTOTP(secret, code, now, c.last)
		if ok != c.ok || (ok && step != c.step) {
			t.Errorf("%s: ok=%v step=%d", c.name, ok, step)
		}
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef", "12345 ", "-12345"} {
		if _, ok := matchTOTP(secret, bad, now, 0); ok {
			t.Errorf("malformed %q accepted", bad)
		}
	}
}

func TestRecoveryCodeFormat(t *testing.T) {
	codes, err := newRecoveryCodes()
	if err != nil || len(codes) != 10 {
		t.Fatal(err, len(codes))
	}
	seen := map[string]bool{}
	for _, c := range codes {
		n := normalizeCode(strings.ToLower(c))
		if len(c) != 11 || c[5] != '-' || !validRecoveryFormat(n) || seen[n] {
			t.Errorf("bad or duplicate code %q", c)
		}
		seen[n] = true
	}
	if hashRecoveryCode("a", "ABCDEFGHIJ") == hashRecoveryCode("b", "ABCDEFGHIJ") {
		t.Error("hash must be person-bound")
	}
}
