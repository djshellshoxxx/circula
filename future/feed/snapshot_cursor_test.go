package feed

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func testSnapshot() SnapshotCursor {
	return SnapshotCursor{Position: testPos, HighWater: 987654321, IssuedAt: testNow.Add(-17*time.Minute + 123*time.Nanosecond)}
}

func sameSnapshot(a, b SnapshotCursor) bool {
	return samePos(a.Position, b.Position) && a.HighWater == b.HighWater && a.IssuedAt.Equal(b.IssuedAt)
}

func mustSealSnapshot(t testing.TB, c *CursorCodec, b Binding, snapshot SnapshotCursor) string {
	t.Helper()
	token, err := c.SealSnapshot(b, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func wantInvalidSnapshot(t testing.TB, c *CursorCodec, b Binding, token string) {
	t.Helper()
	if got, err := c.OpenSnapshot(b, token); err != ErrInvalidCursor || got != (SnapshotCursor{}) {
		t.Fatalf("snapshot refusal = %+v, %v; want zero snapshot and ErrInvalidCursor", got, err)
	}
}

// authenticatedSnapshotToken permits testing authenticated malformed payloads.
// Deterministic nonces are only for reproducible synthetic test/fuzz fixtures;
// the production sealer always uses crypto/rand.
func authenticatedSnapshotToken(t testing.TB, key []byte, kid string, binding Binding, payload []byte) string {
	t.Helper()
	plaintext, err := padSnapshotPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	return authenticatedSnapshotPlaintext(t, key, kid, binding, plaintext)
}

func authenticatedSnapshotPlaintext(t testing.TB, key []byte, kid string, binding Binding, plaintext []byte) string {
	t.Helper()
	_, digest, err := binding.canonical()
	if err != nil {
		t.Fatal(err)
	}
	aead, err := snapshotAEAD(key, kid)
	if err != nil {
		t.Fatal(err)
	}
	aad := snapshotAAD(kid, binding, digest)
	seed := sha256.Sum256(append(append([]byte(nil), aad...), plaintext...))
	nonce := append([]byte(nil), seed[:aead.NonceSize()]...)
	sealed := aead.Seal(nonce, nonce, plaintext, aad)
	return "3." + kid + "." + b64.EncodeToString(sealed)
}

func snapshotFixturePayload(t testing.TB, snapshot SnapshotCursor) []byte {
	t.Helper()
	payload, err := json.Marshal(snapshotPayload{
		V: 3, K: snapshot.Position.Kind, I: snapshot.Position.ID,
		T: snapshot.Position.Created.UnixNano(), H: snapshot.HighWater,
		Iat: snapshot.IssuedAt.UnixNano(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestSnapshotCursorRoundTripAndConfidentiality(t *testing.T) {
	c, _ := testCodec(t)
	binding := Binding{Network: "private-network-77", Viewer: "private-viewer-66", Filter: Filter{PreferenceRevision: "private-preference-55"}}
	for _, snapshot := range []SnapshotCursor{
		testSnapshot(),
		{Position: Cursor{Created: time.Unix(1, 1), Kind: "announcement", ID: "hidden-source-42"}, HighWater: math.MaxInt64, IssuedAt: testNow.Add(-time.Nanosecond)},
		{Position: Cursor{Created: testPos.Created, Kind: "listing", ID: strings.Repeat("<", maxSourceIDLen)}, HighWater: 1, IssuedAt: testNow},
	} {
		token := mustSealSnapshot(t, c, binding, snapshot)
		if !strings.HasPrefix(token, "3.k1.") || !isTokenText(token) || len(token) > maxTokenLen || strings.Count(token, ".") != 2 {
			t.Fatalf("invalid snapshot wire shape: %q", token)
		}
		got, err := c.OpenSnapshot(binding, token)
		if err != nil || !sameSnapshot(got, snapshot) || got.IssuedAt.Location() != time.UTC {
			t.Fatalf("snapshot round trip = %+v, %v; want %+v", got, err, snapshot)
		}
		_, ciphertextText, _ := strings.Cut(strings.TrimPrefix(token, "3."), ".")
		ciphertext, err := b64.DecodeString(ciphertextText)
		if err != nil {
			t.Fatal(err)
		}
		for _, confidential := range []string{binding.Network, binding.Viewer, binding.Filter.PreferenceRevision, snapshot.Position.ID, `"iat"`, `"t":`, "987654321"} {
			if len(confidential) >= 6 && (strings.Contains(token, confidential) || bytes.Contains(ciphertext, []byte(confidential))) {
				t.Fatalf("snapshot exposes confidential field %q", confidential)
			}
		}
		if fresh := mustSealSnapshot(t, c, binding, snapshot); fresh == token {
			t.Fatal("snapshot sealer reused a nonce")
		}
	}
}

func TestSnapshotCursorCiphertextLengthIsIndependentOfPrivateState(t *testing.T) {
	c, _ := testCodec(t)
	small := testSnapshot()
	small.Position.Kind, small.Position.ID, small.HighWater = "listing", "a", 1
	large := testSnapshot()
	large.Position.Kind, large.Position.ID, large.HighWater = "announcement", strings.Repeat("\x01", maxSourceIDLen), math.MaxInt64
	first := mustSealSnapshot(t, c, testBindA, small)
	second := mustSealSnapshot(t, c, testBindA, large)
	if len(first) != len(second) {
		t.Fatalf("ciphertext length reveals private state: shortest %d, longest %d", len(first), len(second))
	}
}

func TestSnapshotCursorRejectsAuthenticatedMalformedPadding(t *testing.T) {
	c, _ := testCodec(t)
	payload := snapshotFixturePayload(t, testSnapshot())
	valid := make([]byte, 1024)
	binary.BigEndian.PutUint16(valid, uint16(len(payload)))
	copy(valid[2:], payload)
	control := authenticatedSnapshotPlaintext(t, testKey1, "k1", testBindA, valid)
	if got, err := c.OpenSnapshot(testBindA, control); err != nil || !sameSnapshot(got, testSnapshot()) {
		t.Fatalf("independent padded fixture refused: %+v %v", got, err)
	}
	changed := func(change func([]byte)) []byte { p := append([]byte(nil), valid...); change(p); return p }
	for name, plaintext := range map[string][]byte{
		"short plaintext":         valid[:len(valid)-1],
		"long plaintext":          append(append([]byte(nil), valid...), 0),
		"zero length":             changed(func(p []byte) { binary.BigEndian.PutUint16(p, 0) }),
		"overflow length":         changed(func(p []byte) { binary.BigEndian.PutUint16(p, 65535) }),
		"length includes padding": changed(func(p []byte) { binary.BigEndian.PutUint16(p, uint16(len(payload)+1)) }),
		"length drops JSON":       changed(func(p []byte) { binary.BigEndian.PutUint16(p, uint16(len(payload)-1)) }),
		"nonzero padding":         changed(func(p []byte) { p[len(p)-1] = 1 }),
	} {
		t.Run(name, func(t *testing.T) {
			wantInvalidSnapshot(t, c, testBindA, authenticatedSnapshotPlaintext(t, testKey1, "k1", testBindA, plaintext))
		})
	}
	for _, invalid := range [][]byte{nil, make([]byte, snapshotPlaintextSize)} {
		if _, err := padSnapshotPayload(invalid); err == nil {
			t.Fatal("invalid payload length padded")
		}
	}
}

func TestSnapshotCursorBindingAndProtocolSeparation(t *testing.T) {
	c, _ := testCodec(t)
	snapshot := testSnapshot()
	token := mustSealSnapshot(t, c, testBindA, snapshot)
	for name, binding := range map[string]Binding{
		"network":          {Network: "network-B", Viewer: testBindA.Viewer, Filter: testBindA.Filter},
		"viewer":           {Network: testBindA.Network, Viewer: "member-B", Filter: testBindA.Filter},
		"filter":           {Network: testBindA.Network, Viewer: testBindA.Viewer, Filter: Filter{Kinds: []string{"listing"}, PreferenceRevision: "rev-1"}},
		"revision":         {Network: testBindA.Network, Viewer: testBindA.Viewer, Filter: Filter{PreferenceRevision: "rev-2"}},
		"field boundaries": {Network: "network-A" + testBindA.Viewer[:1], Viewer: testBindA.Viewer[1:], Filter: testBindA.Filter},
	} {
		t.Run(name, func(t *testing.T) { wantInvalidSnapshot(t, c, binding, token) })
	}
	equivalent := testBindA
	equivalent.Filter.Kinds = []string{"listing", "announcement", "listing"}
	if got, err := c.OpenSnapshot(equivalent, token); err != nil || !sameSnapshot(got, snapshot) {
		t.Fatalf("equivalent filter refused: %+v %v", got, err)
	}
	wantInvalidSnapshot(t, c, testBindA, mustSeal(t, c, testBindA, snapshot.Position))
	if got, err := c.Open(testBindA, token); err != ErrInvalidCursor || got != (Cursor{}) {
		t.Fatalf("snapshot token accepted as a legacy cursor: %+v %v", got, err)
	}
}

func TestSnapshotCursorTamperingAndMalformedInputs(t *testing.T) {
	c, _ := testCodec(t)
	token := mustSealSnapshot(t, c, testBindA, testSnapshot())
	parts := strings.Split(token, ".")
	wire, err := b64.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	for i := range wire {
		bad := append([]byte(nil), wire...)
		bad[i] ^= 0x80
		wantInvalidSnapshot(t, c, testBindA, "3.k1."+b64.EncodeToString(bad))
	}
	for length := 0; length < len(token); length++ {
		wantInvalidSnapshot(t, c, testBindA, token[:length])
	}
	for name, malformed := range map[string]string{
		"unknown version":      "4." + parts[1] + "." + parts[2],
		"leading version zero": "03." + parts[1] + "." + parts[2],
		"padded":               token + "=",
		"newline":              token + "\n",
		"extra segment":        token + ".",
		"empty key":            "3.." + parts[2],
		"long key":             "3." + strings.Repeat("k", maxKeyIDLen+1) + "." + parts[2],
		"invalid length":       "3.k1.A",
		"oversized":            "3.k1." + strings.Repeat("A", maxTokenLen),
		"short nonce":          "3.k1." + b64.EncodeToString(wire[:11]),
		"missing tag":          "3.k1." + b64.EncodeToString(wire[:12]),
	} {
		t.Run(name, func(t *testing.T) { wantInvalidSnapshot(t, c, testBindA, malformed) })
	}
	if len(wire)%3 != 0 {
		wantInvalidSnapshot(t, c, testBindA, "3.k1."+nonCanonical(parts[2]))
	}
}

func TestSnapshotCursorExpiryIsNotRenewed(t *testing.T) {
	c, clk := testCodec(t)
	snapshot := testSnapshot()
	snapshot.IssuedAt = testNow.Add(321 * time.Nanosecond)
	clk.Set(snapshot.IssuedAt)
	token := mustSealSnapshot(t, c, testBindA, snapshot)
	for _, tc := range []struct {
		name  string
		at    time.Time
		valid bool
	}{
		{"issued", snapshot.IssuedAt, true},
		{"before expiry", snapshot.IssuedAt.Add(DefaultCursorTTL - time.Nanosecond), true},
		{"at expiry", snapshot.IssuedAt.Add(DefaultCursorTTL), false},
		{"after expiry", snapshot.IssuedAt.Add(DefaultCursorTTL + time.Nanosecond), false},
		{"clock allowance", snapshot.IssuedAt.Add(-maxIssueSkew), true},
		{"too far ahead", snapshot.IssuedAt.Add(-maxIssueSkew - time.Nanosecond), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk.Set(tc.at)
			got, err := c.OpenSnapshot(testBindA, token)
			if tc.valid {
				if err != nil || !sameSnapshot(got, snapshot) {
					t.Fatalf("valid snapshot refused: %+v %v", got, err)
				}
			} else {
				wantInvalidSnapshot(t, c, testBindA, token)
				if _, err := c.SealSnapshot(testBindA, snapshot); err == nil {
					t.Fatal("sealing renewed an expired or invalid traversal")
				}
			}
		})
	}
	clk.Set(snapshot.IssuedAt.Add(DefaultCursorTTL - time.Nanosecond))
	snapshot.Position.ID = "next-page"
	continued := mustSealSnapshot(t, c, testBindA, snapshot)
	clk.Set(snapshot.IssuedAt.Add(DefaultCursorTTL))
	wantInvalidSnapshot(t, c, testBindA, continued)
}

func TestSnapshotCursorKeyRotation(t *testing.T) {
	before, _ := testCodec(t)
	during := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k1": testKey1, "k2": testKey2}), 0, func() time.Time { return testNow })
	retired := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k2": testKey2}), 0, func() time.Time { return testNow })
	alias := mustCodec(t, mustRing(t, "k2", map[string][]byte{"k1": testKey1, "k2": testKey1}), 0, func() time.Time { return testNow })
	old := mustSealSnapshot(t, before, testBindA, testSnapshot())
	fresh := mustSealSnapshot(t, during, testBindA, testSnapshot())
	for _, token := range []string{old, fresh} {
		if got, err := during.OpenSnapshot(testBindA, token); err != nil || !sameSnapshot(got, testSnapshot()) {
			t.Fatalf("rotation overlap failed: %+v %v", got, err)
		}
	}
	wantInvalidSnapshot(t, retired, testBindA, old)
	wantInvalidSnapshot(t, before, testBindA, fresh)
	wantInvalidSnapshot(t, alias, testBindA, strings.Replace(old, "3.k1.", "3.k2.", 1))
	unknown := authenticatedSnapshotToken(t, placeholderKey[:], "unknown", testBindA, snapshotFixturePayload(t, testSnapshot()))
	wantInvalidSnapshot(t, before, testBindA, unknown)
}

func TestSnapshotCursorRejectsInvalidState(t *testing.T) {
	c, _ := testCodec(t)
	with := func(change func(*SnapshotCursor)) SnapshotCursor { s := testSnapshot(); change(&s); return s }
	for name, invalid := range map[string]SnapshotCursor{
		"zero highwater":       with(func(s *SnapshotCursor) { s.HighWater = 0 }),
		"negative highwater":   with(func(s *SnapshotCursor) { s.HighWater = -1 }),
		"zero issue time":      with(func(s *SnapshotCursor) { s.IssuedAt = time.Time{} }),
		"epoch issue time":     with(func(s *SnapshotCursor) { s.IssuedAt = time.Unix(0, 0) }),
		"pre-epoch issue time": with(func(s *SnapshotCursor) { s.IssuedAt = time.Unix(-1, 0) }),
		"issue time overflow":  with(func(s *SnapshotCursor) { s.IssuedAt = time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC) }),
		"missing position":     with(func(s *SnapshotCursor) { s.Position = Cursor{} }),
		"invalid position":     with(func(s *SnapshotCursor) { s.Position.ID = "\xff" }),
	} {
		t.Run(name, func(t *testing.T) {
			if token, err := c.SealSnapshot(testBindA, invalid); err == nil || token != "" {
				t.Fatalf("invalid snapshot sealed: %q %v", token, err)
			}
		})
	}
	good := mustSealSnapshot(t, c, testBindA, testSnapshot())
	var nilCustom *nilUnsafeKeys
	for _, bad := range []*CursorCodec{nil, {}, {keys: nilCustom, ttl: time.Hour, now: func() time.Time { return testNow }}, mustCodec(t, shortKeySet{}, 0, nil)} {
		if _, err := bad.SealSnapshot(testBindA, testSnapshot()); err == nil {
			t.Fatal("unconfigured codec sealed a snapshot")
		}
		wantInvalidSnapshot(t, bad, testBindA, good)
	}
}

func TestSnapshotCursorRejectsAuthenticatedMalformedPayload(t *testing.T) {
	c, _ := testCodec(t)
	payload := string(snapshotFixturePayload(t, testSnapshot()))
	for name, malformed := range map[string]string{
		"duplicate ID":        strings.Replace(payload, `"i":"item-1"`, `"i":"other","i":"item-1"`, 1),
		"unknown field":       strings.TrimSuffix(payload, "}") + `,"extra":true}`,
		"case alias":          strings.Replace(payload, `"h":`, `"H":`, 1),
		"invalid Unicode":     strings.Replace(payload, "item-1", `\ud800`, 1),
		"invalid UTF-8":       strings.Replace(payload, "item-1", "\xff", 1),
		"wrong version":       strings.Replace(payload, `"v":3`, `"v":2`, 1),
		"zero highwater":      strings.Replace(payload, `"h":987654321`, `"h":0`, 1),
		"negative highwater":  strings.Replace(payload, `"h":987654321`, `"h":-1`, 1),
		"overflow highwater":  strings.Replace(payload, `"h":987654321`, `"h":9223372036854775808`, 1),
		"future issue time":   strings.Replace(payload, itoa64(testSnapshot().IssuedAt.UnixNano()), "9223372036854775807", 1),
		"negative issue time": strings.Replace(payload, itoa64(testSnapshot().IssuedAt.UnixNano()), "-9223372036854775808", 1),
		"empty":               `{}`,
		"null":                `null`,
	} {
		t.Run(name, func(t *testing.T) {
			wantInvalidSnapshot(t, c, testBindA, authenticatedSnapshotToken(t, testKey1, "k1", testBindA, []byte(malformed)))
		})
	}
}
