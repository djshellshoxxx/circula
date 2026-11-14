package service

import (
	"strings"
	"testing"
	"time"
)

// TOTP matching never panics, accepts only 6 ASCII digits, honours the +-1
// step window and the replay guard.
func FuzzMatchTOTP(f *testing.F) {
	secret := []byte("12345678901234567890")
	for _, c := range []string{"", "287082", "28708", "2870822", "abcdef", "２８７０８２", "-12345", " 28708", "000000", "287082\x00"} {
		f.Add(c, int64(59), int64(0))
	}
	f.Add("081804", int64(1111111109), int64(37037036))
	f.Add("000000", int64(-100000), int64(-5))
	f.Add("000000", int64(1<<62), int64(1<<62))
	f.Fuzz(func(t *testing.T, code string, unix, last int64) {
		now := time.Unix(unix%(1<<40), 0)
		step, ok := matchTOTP(secret, code, now, last)
		if !ok {
			return
		}
		if !isDigits(code, totpDigits) {
			t.Fatalf("accepted malformed code %q", code)
		}
		cur := totpStep(now)
		if step < cur-totpSkew || step > cur+totpSkew || step < 0 {
			t.Fatalf("step %d outside window of %d", step, cur)
		}
		if step <= last {
			t.Fatalf("replay: step %d <= last %d", step, last)
		}
		if hotp(secret, uint64(step)) != code { //nolint:gosec // step >= 0 asserted above
			t.Fatalf("code %q is not hotp(step %d)", code, step)
		}
		if _, again := matchTOTP(secret, code, now, step); again {
			t.Fatalf("code %q accepted twice", code)
		}
	})
}

// Codes for steps in the window are accepted, outside it refused, and never twice.
func TestTOTPWindowExactlyOnce(t *testing.T) {
	secret := []byte("0123456789abcdefghij")
	now := time.Unix(1_900_000_000, 0)
	cur := totpStep(now)
	for d := int64(-4); d <= 4; d++ {
		code := hotp(secret, uint64(cur+d)) //nolint:gosec // positive
		step, ok := matchTOTP(secret, code, now, 0)
		inWin := d >= -totpSkew && d <= totpSkew
		if ok != inWin {
			// a 6-digit code collides with a neighbouring step with p=1e-6
			if ok && step != cur+d {
				continue
			}
			t.Errorf("d=%d ok=%v want %v", d, ok, inWin)
		}
		if ok {
			if _, again := matchTOTP(secret, code, now, step); again {
				t.Errorf("d=%d: accepted after being consumed", d)
			}
		}
	}
}

func FuzzRecoveryCodeNormalise(f *testing.F) {
	for _, c := range []string{"", "ABCD-EFGH-IJKL-MNOP", "abcd efgh ijkl mnop", "ABCD-EFGH-2345-6722", "AAAA-AAAA-AAAA-AAAİ", "ß", "ǆ", "\x00", strings.Repeat("A", 100)} {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, c string) {
		norm, ok := normalizeRecoveryCode(c)
		if !ok {
			return
		}
		if len(norm) != recoveryCodeChars {
			t.Fatalf("len %d", len(norm))
		}
		for i := 0; i < len(norm); i++ {
			if (norm[i] < 'A' || norm[i] > 'Z') && (norm[i] < '2' || norm[i] > '7') {
				t.Fatalf("bad char in %q from %q", norm, c)
			}
		}
		again, ok2 := normalizeRecoveryCode(formatRecoveryCode(norm))
		if !ok2 || again != norm {
			t.Fatalf("format/normalise not stable: %q -> %q", norm, again)
		}
	})
}

// openSecret must reject garbage with an error, never panic.
func FuzzOpenSecret(f *testing.F) {
	s := &Service{cfg: Config{SecretKey: []byte("0123456789abcdef0123456789abcdef")}}
	sealed, _ := s.sealSecret("p1", []byte("12345678901234567890"))
	for _, v := range []string{"", "raw:", "raw:!!!", "v1:", "v1:AAAA", sealed, sealed[:len(sealed)-3], "x:y", "raw:GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"} {
		f.Add(v)
	}
	keyless := &Service{}
	f.Fuzz(func(t *testing.T, stored string) {
		_, _ = s.openSecret("p1", stored)
		_, _ = keyless.openSecret("p1", stored)
	})
}

func TestSealOpenRoundTripAndBinding(t *testing.T) {
	s := &Service{cfg: Config{SecretKey: []byte("0123456789abcdef0123456789abcdef")}}
	sec := []byte("12345678901234567890")
	st, err := s.sealSecret("alice", sec)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.openSecret("alice", st); err != nil || string(got) != string(sec) {
		t.Fatalf("roundtrip: %v %q", err, got)
	}
	if _, err := s.openSecret("mallory", st); err == nil {
		t.Fatal("sealed seed opened under another person id")
	}
}
