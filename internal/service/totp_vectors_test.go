package service

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 4226/6238 vectors
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"net/url"
	"strings"
	"testing"
	"time"
)

// refHOTP is an independent RFC 4226 section 5.3 implementation, generalised
// to any hash and digit count, used to validate the RFC 6238 appendix B table
// (the production code is SHA-1 / 6 digits only, as authenticator apps assume).
func refHOTP(h func() hash.Hash, key []byte, counter uint64, digits int) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	m := hmac.New(h, key)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0xf
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, bin%mod)
}

// RFC 4226 appendix D: secret "12345678901234567890", counters 0..9.
func TestHOTPRFC4226AppendixD(t *testing.T) {
	secret := []byte("12345678901234567890")
	want := []string{"755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"}
	for c, w := range want {
		if got := hotp(secret, uint64(c)); got != w { //nolint:gosec // small non-negative
			t.Errorf("hotp counter %d: got %s want %s", c, got, w)
		}
		if got := refHOTP(func() hash.Hash { return sha1.New() }, secret, uint64(c), 6); got != w { //nolint:gosec // small non-negative
			t.Errorf("ref counter %d: got %s want %s", c, got, w)
		}
	}
	// Appendix D lists the full HMAC-SHA-1 for counter 0.
	m := hmac.New(sha1.New, secret)
	m.Write(make([]byte, 8))
	if got := hex.EncodeToString(m.Sum(nil)); got != "cc93cf18508d94934c64b65d8ba7667fb7cde4b0" {
		t.Errorf("HMAC-SHA-1 counter 0: %s", got)
	}
}

// RFC 6238 appendix B: 8-digit codes, 30 s step, T0 = 0, per-algorithm seeds.
func TestTOTPRFC6238AppendixB(t *testing.T) {
	seeds := map[string][]byte{
		"SHA1":   []byte("12345678901234567890"),
		"SHA256": []byte("12345678901234567890123456789012"),
		"SHA512": []byte("1234567890123456789012345678901234567890123456789012345678901234"),
	}
	newHash := map[string]func() hash.Hash{
		"SHA1":   func() hash.Hash { return sha1.New() }, //nolint:gosec // RFC vector
		"SHA256": sha256.New,
		"SHA512": sha512.New,
	}
	rows := []struct {
		unix                 int64
		sha1, sha256, sha512 string
	}{
		{59, "94287082", "46119246", "90693936"},
		{1111111109, "07081804", "68084774", "25091201"},
		{1111111111, "14050471", "67062674", "99943326"},
		{1234567890, "89005924", "91819424", "93441116"},
		{2000000000, "69279037", "90698825", "38618901"},
		{20000000000, "65353130", "77737706", "47863826"},
	}
	for _, r := range rows {
		step := uint64(r.unix / 30) //nolint:gosec // positive
		for alg, want := range map[string]string{"SHA1": r.sha1, "SHA256": r.sha256, "SHA512": r.sha512} {
			if got := refHOTP(newHash[alg], seeds[alg], step, 8); got != want {
				t.Errorf("%s t=%d: got %s want %s", alg, r.unix, got, want)
			}
		}
		// Production code: SHA-1, 6 digits = the low six digits of the 8-digit value.
		if got, want := hotp(seeds["SHA1"], step), r.sha1[2:]; got != want {
			t.Errorf("hotp t=%d: got %s want %s", r.unix, got, want)
		}
		if got := totpStep(time.Unix(r.unix, 0)); got != r.unix/30 {
			t.Errorf("totpStep(%d)=%d want %d", r.unix, got, r.unix/30)
		}
	}
}

// otpauth Key URI format (Google Authenticator wiki): otpauth://totp/Issuer:account?secret=BASE32&issuer=...
// with an unpadded upper-case base32 secret; the default parameters are spelled out.
func TestOTPAuthURIFormat(t *testing.T) {
	secret := []byte("12345678901234567890")
	u, err := url.Parse(otpauthURI("alice@example.org", secret))
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "otpauth" || u.Host != "totp" {
		t.Fatalf("scheme/type: %s://%s", u.Scheme, u.Host)
	}
	if got := strings.TrimPrefix(u.Path, "/"); got != "Circula:alice@example.org" {
		t.Errorf("label %q", got)
	}
	q := u.Query()
	if q.Get("secret") != "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" {
		t.Errorf("secret %q (want unpadded base32 of the seed)", q.Get("secret"))
	}
	for k, v := range map[string]string{"issuer": "Circula", "algorithm": "SHA1", "digits": "6", "period": "30"} {
		if q.Get(k) != v {
			t.Errorf("%s=%q want %q", k, q.Get(k), v)
		}
	}
	// A login with reserved characters must not break out of the label or query.
	u, err = url.Parse(otpauthURI("a b/c?d&e=f#g", secret))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("secret") == "" || u.Fragment != "" {
		t.Errorf("hostile login leaked into URI structure: %s", u)
	}
}

// The seed round-trips through the unpadded base32 form authenticator apps display.
func TestTOTPSecretBase32RoundTrip(t *testing.T) {
	s, err := newTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 20 {
		t.Fatalf("secret length %d, RFC 4226 R6 recommends 160 bits", len(s))
	}
	enc := b32.EncodeToString(s)
	back, err := b32.DecodeString(enc)
	if err != nil || string(back) != string(s) {
		t.Fatalf("round trip: %v", err)
	}
	if len(enc) != 32 {
		t.Errorf("20 bytes must encode to 32 unpadded characters, got %d", len(enc))
	}
}
