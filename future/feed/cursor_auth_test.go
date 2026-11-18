package feed

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testKey1  = bytes.Repeat([]byte{0x11}, 32)
	testKey2  = bytes.Repeat([]byte{0x22}, 32)
	testNow   = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	testPos   = Cursor{Created: time.Unix(1234, 34).UTC(), Kind: "listing", ID: "item-1"}
	testBindA = Binding{Network: "network-A", Viewer: "member-A", Filter: Filter{PreferenceRevision: "rev-1"}}
)

// clock is a settable test clock.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

func mustRing(t testing.TB, current string, keys map[string][]byte) *KeyRing {
	t.Helper()
	r, err := NewKeyRing(current, keys)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func mustCodec(t testing.TB, keys KeySet, ttl time.Duration, now func() time.Time) *CursorCodec {
	t.Helper()
	c, err := NewCursorCodec(keys, CodecOptions{TTL: ttl, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testCodec(t testing.TB) (*CursorCodec, *clock) {
	clk := &clock{t: testNow}
	return mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey1}), 0, clk.Now), clk
}

func mustSeal(t testing.TB, c *CursorCodec, b Binding, pos Cursor) string {
	t.Helper()
	tok, err := c.Seal(b, pos)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// wantInvalid asserts the uniform refusal: the ErrInvalidCursor value itself,
// not a wrapped or differently worded error.
func wantInvalid(t *testing.T, name string, _ Cursor, err error) {
	t.Helper()
	if err != ErrInvalidCursor {
		t.Fatalf("%s: err = %v, want exactly ErrInvalidCursor", name, err)
	}
}

func wantPos(t *testing.T, name string, got Cursor, err error, want Cursor) {
	t.Helper()
	if err != nil || !got.Created.Equal(want.Created) || got.Kind != want.Kind || got.ID != want.ID {
		t.Fatalf("%s: got %+v, %v; want %+v", name, got, err, want)
	}
}

// signedToken builds a token with a valid MAC over an arbitrary payload, to
// reach the checks that run after MAC verification.
func signedToken(t *testing.T, key []byte, kid string, b Binding, payload []byte) string {
	t.Helper()
	_, d, err := b.canonical()
	if err != nil {
		t.Fatal(err)
	}
	return kid + "." + b64.EncodeToString(payload) + "." + b64.EncodeToString(cursorMAC(key, kid, b, d, payload))
}

func splitToken(t *testing.T, tok string) (kid string, payload, sig []byte) {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q has %d segments, want 3", tok, len(parts))
	}
	p, err1 := b64.DecodeString(parts[1])
	s, err2 := b64.DecodeString(parts[2])
	if err1 != nil || err2 != nil {
		t.Fatalf("token %q does not decode: %v %v", tok, err1, err2)
	}
	return parts[0], p, s
}

func TestCursorRoundTrip(t *testing.T) {
	c, _ := testCodec(t)
	for _, pos := range []Cursor{
		testPos,
		{Created: time.Unix(1, 1), Kind: "announcement", ID: "a"},
		{Created: time.Date(2200, 1, 2, 3, 4, 5, 999999999, time.UTC), Kind: "listing", ID: "018f0000-0000-7000-8000-000000000042"},
		{Created: time.Unix(99, 0), Kind: "listing", ID: "unicode ✓ <&>   \"quoted\" \x01"},
		{Created: time.Unix(99, 0), Kind: "listing", ID: strings.Repeat("<", maxSourceIDLen)}, // worst-case JSON escaping
	} {
		tok := mustSeal(t, c, testBindA, pos)
		if len(tok) > maxTokenLen || !isTokenText(tok) || strings.Count(tok, ".") != 2 {
			t.Fatalf("minted token %q is not a bounded 3-segment base64url token", tok)
		}
		got, err := c.Open(testBindA, tok)
		wantPos(t, "round trip", got, err, pos)
		if got.Created.Location() != time.UTC {
			t.Fatalf("opened time is not UTC: %v", got.Created)
		}
	}
}

func TestCursorTokenOmitsBoundIdentifiers(t *testing.T) {
	c, _ := testCodec(t)
	b := Binding{Network: "net-secret-7f3a", Viewer: "person-secret-91c2", Filter: Filter{Kinds: []string{"listing"}, PreferenceRevision: "pref-secret-55d0"}}
	tok := mustSeal(t, c, b, testPos)
	kid, payload, sig := splitToken(t, tok)
	if kid != "k1" || len(sig) != sha256.Size {
		t.Fatalf("kid %q, MAC length %d", kid, len(sig))
	}
	for _, secret := range []string{b.Network, b.Viewer, b.Filter.PreferenceRevision, "secret"} {
		if strings.Contains(tok, secret) || bytes.Contains(payload, []byte(secret)) {
			t.Fatalf("token or payload exposes %q: %s", secret, payload)
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 5 || fields["v"] == nil || fields["k"] == nil || fields["i"] == nil || fields["t"] == nil || fields["iat"] == nil {
		t.Fatalf("payload fields = %s, want exactly v, k, i, t, iat", payload)
	}
	if want := `"iat":` + itoa64(testNow.Unix()); !strings.Contains(string(payload), want) {
		t.Fatalf("payload %s lacks issue time %s from the injected clock", payload, want)
	}
}

func itoa64(n int64) string { b, _ := json.Marshal(n); return string(b) }

func TestCursorBinding(t *testing.T) {
	c, _ := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)
	with := func(f func(*Binding)) Binding { b := testBindA; f(&b); return b }

	for name, b := range map[string]Binding{
		"wrong network":             with(func(b *Binding) { b.Network = "network-B" }),
		"network prefix":            with(func(b *Binding) { b.Network = "network-" }),
		"member A cursor, member B": with(func(b *Binding) { b.Viewer = "member-B" }),
		"viewer moved to network":   with(func(b *Binding) { b.Network, b.Viewer = "network-Amember-A", "" }),
		"field boundary shift":      with(func(b *Binding) { b.Network, b.Viewer = "network-Amem", "ber-A" }),
		"filter X, filter Y":        with(func(b *Binding) { b.Filter.Kinds = []string{"listing"} }),
		"other kind filter":         with(func(b *Binding) { b.Filter.Kinds = []string{"announcement"} }),
		"preference revision":       with(func(b *Binding) { b.Filter.PreferenceRevision = "rev-2" }),
		"no preference revision":    with(func(b *Binding) { b.Filter.PreferenceRevision = "" }),
		"invalid filter kind":       with(func(b *Binding) { b.Filter.Kinds = []string{"transaction"} }),
		"empty network":             with(func(b *Binding) { b.Network = "" }),
		"empty viewer":              with(func(b *Binding) { b.Viewer = "" }),
	} {
		got, err := c.Open(b, tok)
		wantInvalid(t, name, got, err)
	}

	// Filters that select the same items are the same filter.
	for name, f := range map[string][]string{
		"both kinds":         {"listing", "announcement"},
		"both kinds, sorted": {"announcement", "listing"},
		"repeated":           {"listing", "announcement", "listing"},
	} {
		got, err := c.Open(with(func(b *Binding) { b.Filter.Kinds = f }), tok)
		wantPos(t, name, got, err, testPos)
	}

	// The same holds in the other direction for a narrowed filter.
	listingsA := with(func(b *Binding) { b.Filter.Kinds = []string{"listing"} })
	tok2 := mustSeal(t, c, listingsA, testPos)
	got, err := c.Open(with(func(b *Binding) { b.Filter.Kinds = []string{"listing", "listing"} }), tok2)
	wantPos(t, "repeated narrowed filter", got, err, testPos)
	got, err = c.Open(testBindA, tok2)
	wantInvalid(t, "narrowed cursor under unfiltered query", got, err)
}

func TestCursorTamperedMACSameLength(t *testing.T) {
	c, _ := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)
	kid, payload, sig := splitToken(t, tok)
	for i := range sig {
		for _, bit := range []byte{0x01, 0x80} {
			bad := append([]byte(nil), sig...)
			bad[i] ^= bit
			forged := kid + "." + b64.EncodeToString(payload) + "." + b64.EncodeToString(bad)
			// Same length, same alphabet, decodes to 32 bytes: only the MAC
			// comparison can refuse it.
			if len(forged) != len(tok) || !isTokenText(forged) {
				t.Fatalf("tampered token changed shape: %q vs %q", forged, tok)
			}
			got, err := c.Open(testBindA, forged)
			wantInvalid(t, "flipped MAC byte", got, err)
		}
	}
	for i := range payload {
		bad := append([]byte(nil), payload...)
		bad[i] ^= 0x01
		forged := kid + "." + b64.EncodeToString(bad) + "." + b64.EncodeToString(sig)
		if len(forged) != len(tok) {
			t.Fatalf("tampered payload changed length")
		}
		got, err := c.Open(testBindA, forged)
		wantInvalid(t, "flipped payload byte", got, err)
	}
	// The key ID is MAC input too: relabelling a token to another ID that
	// holds the same secret (an aliased key) is refused.
	aliased := mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey1, "k2": testKey1}), 0, func() time.Time { return testNow })
	got, err := aliased.Open(testBindA, "k2"+tok[len(kid):])
	wantInvalid(t, "relabelled key id", got, err)
	if got, err = c.Open(testBindA, tok); err != nil {
		t.Fatalf("untampered token refused: %+v %v", got, err)
	}
}

func TestCursorMalformedInputs(t *testing.T) {
	c, _ := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)
	kid, payload, sig := splitToken(t, tok)
	p64, s64 := b64.EncodeToString(payload), b64.EncodeToString(sig)
	coreShaped := p64 + "." + base64.RawURLEncoding.EncodeToString(hmacSHA256(testKey1, payload))

	cases := map[string]string{
		"empty":                    "",
		"dot":                      ".",
		"two dots":                 "..",
		"three dots":               "...",
		"key id only":              kid,
		"no MAC":                   kid + "." + p64,
		"no MAC segment":           kid + "." + p64 + ".",
		"no payload":               kid + ".." + s64,
		"no key id":                "." + p64 + "." + s64,
		"core two-segment format":  coreShaped,
		"four segments":            tok + "." + s64,
		"trailing dot":             tok + ".",
		"appended char":            tok + "A",
		"padding":                  tok + "=",
		"padded payload":           kid + "." + p64 + "=." + s64,
		"line break":               kid + "." + p64 + "\n." + s64,
		"CRLF in MAC":              kid + "." + p64 + "." + s64[:10] + "\r\n" + s64[10:],
		"space":                    " " + tok,
		"std base64 alphabet":      strings.NewReplacer("-", "+", "_", "/").Replace(tok) + "+/",
		"percent-encoded":          strings.Replace(tok, ".", "%2E", 1),
		"NUL":                      "\x00.\x00.\x00",
		"non-ASCII":                "k1.✓." + s64,
		"upper-cased":              strings.ToUpper(tok),
		"key id too long":          strings.Repeat("k", maxKeyIDLen+1) + "." + p64 + "." + s64,
		"short MAC":                kid + "." + p64 + "." + s64[:len(s64)-2],
		"long MAC":                 kid + "." + p64 + "." + b64.EncodeToString(append(append([]byte(nil), sig...), 0)),
		"non-canonical MAC bits":   kid + "." + p64 + "." + nonCanonical(s64),
		"extra payload char":       kid + "." + p64 + "A." + s64,
		"invalid base64 length":    kid + ".A." + s64,
		"oversize":                 kid + "." + strings.Repeat("A", maxTokenLen) + "." + s64,
		"garbage":                  "not-a-cursor",
		"JSON instead of token":    `{"v":2,"k":"listing","i":"item-1"}`,
		"MAC under core-style key": kid + "." + p64 + "." + base64.RawURLEncoding.EncodeToString(hmacSHA256(testKey1, payload)),
	}
	if len(payload)%3 != 0 { // only then does the payload encoding have unused bits
		cases["non-canonical payload"] = kid + "." + nonCanonical(p64) + "." + s64
	}
	for i := 0; i < len(tok); i++ {
		cases["truncated to "+itoa64(int64(i))] = tok[:i]
	}
	for name, in := range cases {
		got, err := c.Open(testBindA, in)
		wantInvalid(t, name, got, err)
	}

	// Authentic MACs over payloads this codec would never mint.
	iat := itoa64(testNow.Unix())
	for name, p := range map[string]string{
		"not JSON":         `not json`,
		"JSON array":       `[2,"listing","item-1",1,1]`,
		"version 1":        `{"v":1,"k":"listing","i":"item-1","t":1234000000034,"iat":` + iat + `}`,
		"version 3":        `{"v":3,"k":"listing","i":"item-1","t":1234000000034,"iat":` + iat + `}`,
		"private kind":     `{"v":2,"k":"transaction","i":"tx-1","t":1234000000034,"iat":` + iat + `}`,
		"empty id":         `{"v":2,"k":"listing","i":"","t":1234000000034,"iat":` + iat + `}`,
		"long id":          `{"v":2,"k":"listing","i":"` + strings.Repeat("x", maxSourceIDLen+1) + `","t":1,"iat":` + iat + `}`,
		"zero time":        `{"v":2,"k":"listing","i":"item-1","t":0,"iat":` + iat + `}`,
		"negative time":    `{"v":2,"k":"listing","i":"item-1","t":-5,"iat":` + iat + `}`,
		"no issue time":    `{"v":2,"k":"listing","i":"item-1","t":1234000000034}`,
		"zero issue time":  `{"v":2,"k":"listing","i":"item-1","t":1234000000034,"iat":0}`,
		"float issue time": `{"v":2,"k":"listing","i":"item-1","t":1234000000034,"iat":1.5}`,
		"string time":      `{"v":2,"k":"listing","i":"item-1","t":"1234","iat":` + iat + `}`,
	} {
		got, err := c.Open(testBindA, signedToken(t, testKey1, "k1", testBindA, []byte(p)))
		wantInvalid(t, "authentic MAC, "+name, got, err)
	}
	// Control: the same construction over a minted payload is accepted.
	got, err := c.Open(testBindA, signedToken(t, testKey1, "k1", testBindA, payload))
	wantPos(t, "control", got, err, testPos)
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}

// nonCanonical changes the last character of a raw base64url string so that
// it keeps its decoded bytes but sets unused trailing bits.
func nonCanonical(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, s[len(s)-1])
	return s[:len(s)-1] + string(alphabet[last|1])
}

func TestNonCanonicalHelperKeepsBytes(t *testing.T) {
	// The lenient decoder accepts the altered string as the same bytes; only
	// strict decoding refuses it. This proves the malformed-input case above
	// exercises strictness rather than a decoding error.
	s := b64.EncodeToString(sha256.New().Sum(nil)) // 32 bytes: 43 chars, 2 unused bits
	alt := nonCanonical(s)
	lenient, err := base64.RawURLEncoding.DecodeString(alt)
	want, _ := b64.DecodeString(s)
	if alt == s || err != nil || !bytes.Equal(lenient, want) {
		t.Fatalf("helper does not produce a non-canonical encoding: %q %q %v", s, alt, err)
	}
	if _, err := b64.DecodeString(alt); err == nil {
		t.Fatal("strict decoder accepted non-canonical encoding")
	}
}

func TestCursorExpiry(t *testing.T) {
	c, clk := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)

	for _, tc := range []struct {
		name string
		at   time.Time
		ok   bool
	}{
		{"just issued", testNow, true},
		{"one second before expiry", testNow.Add(DefaultCursorTTL - time.Second), true},
		{"at expiry", testNow.Add(DefaultCursorTTL), false},
		{"long expired", testNow.Add(30 * 24 * time.Hour), false},
		{"verifier clock 59s behind", testNow.Add(-59 * time.Second), true},
		{"verifier clock 1m behind", testNow.Add(-maxIssueSkew), true},
		{"issued in the future", testNow.Add(-maxIssueSkew - time.Second), false},
	} {
		clk.Set(tc.at)
		got, err := c.Open(testBindA, tok)
		if tc.ok {
			wantPos(t, tc.name, got, err, testPos)
		} else {
			wantInvalid(t, tc.name, got, err)
		}
	}

	// The verifier's configured TTL applies to cursors already issued.
	clk.Set(testNow)
	short := mustCodec(t, c.keys, 5*time.Minute, clk.Now)
	clk.Set(testNow.Add(5*time.Minute - time.Second))
	got, err := short.Open(testBindA, tok)
	wantPos(t, "within configured TTL", got, err, testPos)
	clk.Set(testNow.Add(5 * time.Minute))
	got, err = short.Open(testBindA, tok)
	wantInvalid(t, "after configured TTL", got, err)

	// A clock before 1970 cannot issue cursors, and a non-positive issue time
	// is refused even where the TTL alone would not refuse it.
	clk.Set(time.Unix(0, 0))
	if _, err := c.Seal(testBindA, testPos); err == nil {
		t.Fatal("sealed with a non-positive issue time")
	}
	clk.Set(time.Unix(30, 0))
	for _, iat := range []int64{0, -10} {
		p, _ := json.Marshal(cursorPayload{V: cursorVersion, K: "listing", I: "item-1", T: 1, Iat: iat})
		got, err := c.Open(testBindA, signedToken(t, testKey1, "k1", testBindA, p))
		wantInvalid(t, "non-positive issue time", got, err)
	}
}

func TestNewCursorCodecOptions(t *testing.T) {
	ring := mustRing(t, "k1", map[string][]byte{"k1": testKey1})
	c := mustCodec(t, ring, 0, nil)
	if c.ttl != DefaultCursorTTL || c.now == nil {
		t.Fatalf("defaults not applied: ttl %v", c.ttl)
	}
	if mustCodec(t, ring, MaxCursorTTL, nil).ttl != MaxCursorTTL || mustCodec(t, ring, time.Second, nil).ttl != time.Second {
		t.Fatal("bounds not inclusive")
	}
	for name, opt := range map[string]CodecOptions{
		"negative TTL":  {TTL: -time.Minute},
		"sub-second":    {TTL: time.Millisecond},
		"above maximum": {TTL: MaxCursorTTL + time.Second},
	} {
		if _, err := NewCursorCodec(ring, opt); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if _, err := NewCursorCodec(nil, CodecOptions{}); err == nil {
		t.Fatal("nil key set accepted")
	}
	// A real clock issues cursors that open.
	tok := mustSeal(t, c, testBindA, testPos)
	got, err := c.Open(testBindA, tok)
	wantPos(t, "real clock", got, err, testPos)
}

func TestCursorKeyRotation(t *testing.T) {
	clk := &clock{t: testNow}
	before := mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey1}), 0, clk.Now)
	during := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k1": testKey1, "k2": testKey2}), 0, clk.Now)
	after := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k2": testKey2}), 0, clk.Now)
	sameIDOtherKey := mustCodec(t, mustRing(t, "k1", map[string][]byte{"k1": testKey2}), 0, clk.Now)

	old := mustSeal(t, before, testBindA, testPos)
	fresh := mustSeal(t, during, testBindA, testPos)
	if kid, _, _ := splitToken(t, fresh); kid != "k2" {
		t.Fatalf("rotated codec signs with %q, want k2", kid)
	}
	got, err := during.Open(testBindA, old)
	wantPos(t, "rotated key still verifies", got, err, testPos)
	got, err = during.Open(testBindA, fresh)
	wantPos(t, "new key verifies", got, err, testPos)
	got, err = after.Open(testBindA, old)
	wantInvalid(t, "retired key id", got, err)
	got, err = after.Open(testBindA, fresh)
	wantPos(t, "new key after retirement", got, err, testPos)
	got, err = before.Open(testBindA, fresh)
	wantInvalid(t, "unknown key id", got, err)
	got, err = sameIDOtherKey.Open(testBindA, old)
	wantInvalid(t, "same key id, different key", got, err)

	// An unknown key ID is refused even when the MAC matches the placeholder
	// key that keeps its verification work equal.
	p, _ := json.Marshal(cursorPayload{V: cursorVersion, K: "listing", I: "item-1", T: 1, Iat: testNow.Unix()})
	got, err = before.Open(testBindA, signedToken(t, placeholderKey[:], "k9", testBindA, p))
	wantInvalid(t, "unknown key id, placeholder MAC", got, err)
}

// shortKeySet misbehaves: it hands out keys below the minimum length.
type shortKeySet struct{}

func (shortKeySet) SigningKey() (string, []byte)          { return "k1", []byte("short") }
func (shortKeySet) VerificationKey(string) ([]byte, bool) { return []byte("short"), true }

func TestCursorRefusesWeakOrMissingKeys(t *testing.T) {
	c := mustCodec(t, shortKeySet{}, 0, func() time.Time { return testNow })
	if _, err := c.Seal(testBindA, testPos); err == nil {
		t.Fatal("sealed with a short key")
	}
	// A short verification key is treated as unknown, even with a valid MAC
	// over a payload that would otherwise be accepted.
	p, _ := json.Marshal(cursorPayload{V: cursorVersion, K: "listing", I: "item-1", T: 1, Iat: testNow.Unix()})
	got, err := c.Open(testBindA, signedToken(t, []byte("short"), "k1", testBindA, p))
	wantInvalid(t, "short verification key", got, err)

	// A typed-nil KeyRing fails closed instead of panicking.
	var nilRing *KeyRing
	nc := mustCodec(t, nilRing, 0, func() time.Time { return testNow })
	if _, err := nc.Seal(testBindA, testPos); err == nil {
		t.Fatal("sealed with a nil key ring")
	}
	good, _ := testCodec(t)
	got, err = nc.Open(testBindA, mustSeal(t, good, testBindA, testPos))
	wantInvalid(t, "nil key ring", got, err)

	var nilCodec *CursorCodec
	if _, err := nilCodec.Seal(testBindA, testPos); err == nil {
		t.Fatal("nil codec sealed")
	}
	got, err = nilCodec.Open(testBindA, "k1.e30.AAAA")
	wantInvalid(t, "nil codec", got, err)
}

func TestNewKeyRing(t *testing.T) {
	for name, tc := range map[string]struct {
		current string
		keys    map[string][]byte
	}{
		"no keys":            {"k1", nil},
		"current not in set": {"k3", map[string][]byte{"k1": testKey1}},
		"short key":          {"k1", map[string][]byte{"k1": testKey1[:31]}},
		"short retired key":  {"k1", map[string][]byte{"k1": testKey1, "k0": []byte("x")}},
		"empty id":           {"", map[string][]byte{"": testKey1}},
		"dot in id":          {"k.1", map[string][]byte{"k.1": testKey1}},
		"slash in id":        {"k/1", map[string][]byte{"k/1": testKey1}},
		"id too long":        {strings.Repeat("k", 33), map[string][]byte{strings.Repeat("k", 33): testKey1}},
	} {
		if _, err := NewKeyRing(tc.current, tc.keys); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	// Keys are copied: later changes to the caller's slice do not matter.
	k := append([]byte(nil), testKey1...)
	ring := mustRing(t, "Key_2026-11", map[string][]byte{"Key_2026-11": k})
	c := mustCodec(t, ring, 0, func() time.Time { return testNow })
	tok := mustSeal(t, c, testBindA, testPos)
	k[0] ^= 0xff
	got, err := c.Open(testBindA, tok)
	wantPos(t, "copied key", got, err, testPos)
}

func TestCursorDomainSeparation(t *testing.T) {
	c, _ := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)
	kid, payload, sig := splitToken(t, tok)
	_, d, _ := testBindA.canonical()

	// A feed MAC is not a MAC of the bare payload (the core construction),
	// nor of the same fields under another label.
	if hmac.Equal(sig, hmacSHA256(testKey1, payload)) {
		t.Fatal("feed MAC equals the core-style MAC of the payload")
	}
	for _, label := range []string{"", "circula/future/feed-cursor/v1", "circula/core/cursor"} {
		other := labelledMAC(label, testKey1, kid, testBindA, d, payload)
		if hmac.Equal(sig, other) {
			t.Fatalf("label %q produces the same MAC", label)
		}
		got, err := c.Open(testBindA, kid+"."+b64.EncodeToString(payload)+"."+b64.EncodeToString(other))
		wantInvalid(t, "MAC under label "+label, got, err)
	}

	// A core-format cursor minted with the same secret is refused.
	core, _ := json.Marshal(map[string]string{"p": "item-1", "f": "/api/v1/feed|network-A|member-A|"})
	got, err := c.Open(testBindA, b64.EncodeToString(core)+"."+b64.EncodeToString(hmacSHA256(testKey1, core)))
	wantInvalid(t, "core cursor", got, err)
}

func TestSealRejectsInvalidInputs(t *testing.T) {
	c, _ := testCodec(t)
	at := testPos.Created
	for name, tc := range map[string]struct {
		b   Binding
		pos Cursor
	}{
		"private kind":         {testBindA, Cursor{Created: at, Kind: "transaction", ID: "x"}},
		"empty kind":           {testBindA, Cursor{Created: at, ID: "x"}},
		"empty id":             {testBindA, Cursor{Created: at, Kind: "listing"}},
		"id too long":          {testBindA, Cursor{Created: at, Kind: "listing", ID: strings.Repeat("x", maxSourceIDLen+1)}},
		"id not UTF-8":         {testBindA, Cursor{Created: at, Kind: "listing", ID: "\xff"}},
		"zero time":            {testBindA, Cursor{Kind: "listing", ID: "x"}},
		"epoch":                {testBindA, Cursor{Created: time.Unix(0, 0), Kind: "listing", ID: "x"}},
		"before 1970":          {testBindA, Cursor{Created: time.Unix(-1, 0), Kind: "listing", ID: "x"}},
		"after 2262":           {testBindA, Cursor{Created: time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC), Kind: "listing", ID: "x"}},
		"empty network":        {Binding{Viewer: "v"}, testPos},
		"empty viewer":         {Binding{Network: "n"}, testPos},
		"network too long":     {Binding{Network: strings.Repeat("n", maxBindingLen+1), Viewer: "v"}, testPos},
		"viewer too long":      {Binding{Network: "n", Viewer: strings.Repeat("v", maxBindingLen+1)}, testPos},
		"revision too long":    {Binding{Network: "n", Viewer: "v", Filter: Filter{PreferenceRevision: strings.Repeat("r", maxBindingLen+1)}}, testPos},
		"filter private kind":  {Binding{Network: "n", Viewer: "v", Filter: Filter{Kinds: []string{"listing", "transaction"}}}, testPos},
		"filter too many kind": {Binding{Network: "n", Viewer: "v", Filter: Filter{Kinds: make([]string, maxFilterKinds+1)}}, testPos},
	} {
		if tok, err := c.Seal(tc.b, tc.pos); err == nil {
			t.Fatalf("%s: sealed %q", name, tok)
		}
	}
	if _, err := sealWith("bad.id", testKey1, testBindA, testPos, testNow); err == nil {
		t.Fatal("sealed with an invalid key id")
	}
}

func TestFilterCanonical(t *testing.T) {
	_, all, _ := Filter{}.canonical()
	_, both, _ := Filter{Kinds: []string{"listing", "announcement"}}.canonical()
	_, listing, _ := Filter{Kinds: []string{"listing"}}.canonical()
	_, rev, _ := Filter{PreferenceRevision: "1"}.canonical()
	if all != both || all == listing || all == rev || listing == rev {
		t.Fatal("filter digests are not canonical")
	}
	kinds, _, _ := Filter{Kinds: []string{"listing", "announcement", "listing"}}.canonical()
	if strings.Join(kinds, ",") != "announcement,listing" {
		t.Fatalf("kinds = %v", kinds)
	}
}

func TestCursorUniformRefusal(t *testing.T) {
	// Every refusal reason yields the same error value and message.
	c, clk := testCodec(t)
	tok := mustSeal(t, c, testBindA, testPos)
	other := mustCodec(t, mustRing(t, "k9", map[string][]byte{"k9": testKey2}), 0, clk.Now)
	reasons := map[string]func() (Cursor, error){
		"garbage":       func() (Cursor, error) { return c.Open(testBindA, "garbage") },
		"tampered":      func() (Cursor, error) { return c.Open(testBindA, tok[:len(tok)-1]+"A") },
		"wrong network": func() (Cursor, error) { return c.Open(Binding{Network: "x", Viewer: "member-A"}, tok) },
		"wrong viewer":  func() (Cursor, error) { return c.Open(Binding{Network: "network-A", Viewer: "x"}, tok) },
		"unknown key":   func() (Cursor, error) { return other.Open(testBindA, tok) },
		"expired": func() (Cursor, error) {
			clk.Set(testNow.Add(2 * DefaultCursorTTL))
			defer clk.Set(testNow)
			return c.Open(testBindA, tok)
		},
	}
	for name, open := range reasons {
		got, err := open()
		wantInvalid(t, name, got, err)
		if got != (Cursor{}) {
			t.Fatalf("%s: refusal leaked a position %+v", name, got)
		}
	}
}

func TestCursorCodecConcurrentUse(t *testing.T) {
	c, _ := testCodec(t)
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			b := Binding{Network: "network-A", Viewer: "member-" + itoa64(int64(g))}
			for i := 0; i < 50; i++ {
				pos := Cursor{Created: time.Unix(int64(1000+i), 0), Kind: "listing", ID: "item-" + itoa64(int64(i))}
				tok, err := c.Seal(b, pos)
				if err != nil {
					errs <- err.Error()
					return
				}
				if got, err := c.Open(b, tok); err != nil || got.ID != pos.ID {
					errs <- "round trip failed"
					return
				}
				if _, err := c.Open(Binding{Network: "network-A", Viewer: "other"}, tok); err != ErrInvalidCursor {
					errs <- "cross-viewer cursor accepted"
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}
