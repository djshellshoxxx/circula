package feed

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCursorPreservesExistingV2Token(t *testing.T) {
	// This token uses the v2 bytes emitted before the completion work. The
	// fixed value also checks that sealing does not silently change the wire
	// format or the length-prefixed binding construction.
	const token = "k1.eyJ2IjoyLCJrIjoibGlzd" +
		"GluZyIsImkiOiJpdGVtLTEiL" +
		"CJ0IjoxMjM0MDAwMDAwMDM0L" +
		"CJpYXQiOjE3OTUwMDMyMDB9." +
		"Ng8kUk7RhdpzMCdLMvCmrqt4" +
		"pv2brxkFKFzICbA4jc8"
	c, _ := testCodec(t)
	got, err := c.Open(testBindA, token)
	wantPos(t, "existing v2 token", got, err, testPos)
	if minted := mustSeal(t, c, testBindA, testPos); minted != token {
		t.Fatalf("v2 wire format changed: %q", minted)
	}
}

func TestCursorRejectsAmbiguousAuthenticatedJSON(t *testing.T) {
	c, _ := testCodec(t)
	_, payload, _ := splitToken(t, mustSeal(t, c, testBindA, testPos))
	text := string(payload)
	cases := map[string]string{
		"duplicate ID":            strings.Replace(text, `"i":"item-1"`, `"i":"other","i":"item-1"`, 1),
		"duplicate version":       strings.Replace(text, `"v":2`, `"v":1,"v":2`, 1),
		"case aliased field":      strings.Replace(text, `"k":`, `"K":`, 1),
		"unknown field":           strings.TrimSuffix(text, "}") + `,"network":"untrusted"}`,
		"escaped field name":      strings.Replace(text, `"i":`, `"\u0069":`, 1),
		"invalid UTF-8":           strings.Replace(text, "item-1", "\xff", 1),
		"unpaired high surrogate": strings.Replace(text, "item-1", `\ud800`, 1),
		"unpaired low surrogate":  strings.Replace(text, "item-1", `\udc00`, 1),
		"trailing whitespace":     text + "\n",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			token := signedToken(t, testKey1, "k1", testBindA, []byte(input))
			got, err := c.Open(testBindA, token)
			if err != ErrInvalidCursor || got != (Cursor{}) {
				t.Fatalf("noncanonical authenticated JSON accepted: got %+v, %v", got, err)
			}
		})
	}
	// A literal replacement character is valid UTF-8 and must still round-trip.
	pos := testPos
	pos.ID = "replacement-\ufffd"
	got, err := c.Open(testBindA, mustSeal(t, c, testBindA, pos))
	wantPos(t, "literal replacement character", got, err, pos)
}

func TestKeyRingReturnedKeysCannotMutateRing(t *testing.T) {
	ring := mustRing(t, "k1", map[string][]byte{"k1": testKey1})
	c := mustCodec(t, ring, 0, func() time.Time { return testNow })
	token := mustSeal(t, c, testBindA, testPos)

	_, signing := ring.SigningKey()
	for i := range signing {
		signing[i] = 0
	}
	verification, ok := ring.VerificationKey("k1")
	if !ok || !bytes.Equal(verification, testKey1) {
		t.Fatal("mutating a signing-key copy changed the ring")
	}
	for i := range verification {
		verification[i] = 0
	}
	got, err := c.Open(testBindA, token)
	wantPos(t, "key-copy mutation", got, err, testPos)
	if fresh := mustSeal(t, c, testBindA, testPos); fresh != token {
		t.Fatal("mutating a verification-key copy changed signing")
	}
}

type nilUnsafeKeys struct{ key []byte }

func (k *nilUnsafeKeys) SigningKey() (string, []byte)          { return "k1", k.key }
func (k *nilUnsafeKeys) VerificationKey(string) ([]byte, bool) { return k.key, true }

func TestCursorTypedNilCustomKeySetFailsClosed(t *testing.T) {
	var keys *nilUnsafeKeys
	c, err := NewCursorCodec(keys, CodecOptions{Now: func() time.Time { return testNow }})
	if err != nil {
		return // Rejecting the invalid configuration at construction is safe.
	}
	if _, err := c.Seal(testBindA, testPos); err == nil {
		t.Fatal("typed-nil key set sealed a cursor")
	}
	good, _ := testCodec(t)
	got, err := c.Open(testBindA, mustSeal(t, good, testBindA, testPos))
	if err != ErrInvalidCursor || got != (Cursor{}) {
		t.Fatalf("typed-nil key set returned %+v, %v", got, err)
	}
}
