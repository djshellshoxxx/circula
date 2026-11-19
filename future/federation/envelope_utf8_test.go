package federation

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSignRejectsMalformedUTF8(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	peer := Peer{"local", "remote", true, key.Public().(ed25519.PublicKey)}
	badStrings := []string{"\xff", "\x80", "\xc0\xaf", "\xed\xa0\x80", "\xe2\x82", "\xf4\x90\x80\x80"}
	for field, name := range []string{"id", "issuer", "audience", "kind", "listing"} {
		for _, bad := range badStrings {
			t.Run(fmt.Sprintf("%s/%x", name, bad), func(t *testing.T) {
				event := Event{1, "event", "remote", "local", "listing.publish", "listing", 100, 200}
				fields := []*string{&event.ID, &event.Issuer, &event.Audience, &event.Kind, &event.Listing}
				*fields[field] = "prefix-" + bad + "-suffix"
				env, err := Sign(event, key)
				if !errors.Is(err, ErrInvalid) || len(env.Payload) != 0 || len(env.Signature) != 0 {
					got, verifyErr := Verify(peer, env, time.Unix(100, 0))
					t.Fatalf("Sign accepted malformed UTF-8: error=%v; Verify error=%v, event=%+v", err, verifyErr, got)
				}
			})
		}
	}
}

func TestSignPreservesValidUnicodeEncoding(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	event := Event{1, "event-\ufffd", "réseau-北", "local-🧭", "listing.revoke", "cafe\u0301-<>&\u2028\u2029", 100, 200}
	want := "{\"version\":1,\"id\":\"event-�\",\"issuer\":\"réseau-北\",\"audience\":\"local-🧭\",\"kind\":\"listing.revoke\",\"listing\":\"café-\\u003c\\u003e\\u0026\\u2028\\u2029\",\"issued\":100,\"expires\":200}"
	env, err := Sign(event, key)
	if err != nil || string(env.Payload) != want {
		t.Fatalf("valid encoding changed: %q, error=%v", env.Payload, err)
	}
	peer := Peer{event.Audience, event.Issuer, true, key.Public().(ed25519.PublicKey)}
	got, err := Verify(peer, env, time.Unix(100, 0))
	if err != nil || got != event {
		t.Fatalf("valid strings did not round trip: %+v, error=%v", got, err)
	}
}

func TestVerifyRejectsMalformedUTF8Representations(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	peer := Peer{"local", "remote", true, key.Public().(ed25519.PublicKey)}
	event := Event{1, "event", "remote", "local", "listing.publish", "listing", 100, 200}
	valid, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, rawID string }{
		{"invalid_bytes", "prefix-\xff-suffix"},
		{"unpaired_surrogate", "prefix-\\ud800-suffix"},
		// This is the encoding json.Marshal produces for an invalid byte.
		// Verify already rejects it: re-encoding decoded U+FFFD uses the
		// literal rune instead. The Sign repair is not a verifier bypass fix.
		{"replacement_escape", "prefix-\\ufffd-suffix"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := []byte(strings.Replace(string(valid), "\"event\"", "\""+tc.rawID+"\"", 1))
			signed := append([]byte("circula.discovery.v1\x00"), payload...)
			env := Envelope{payload, ed25519.Sign(key, signed)}
			if _, err := Verify(peer, env, time.Unix(100, 0)); !errors.Is(err, ErrInvalid) {
				t.Fatalf("accepted noncanonical signed payload: error=%v", err)
			}
		})
	}
}

func FuzzSignPreservesEventStrings(f *testing.F) {
	for field := uint8(0); field < 5; field++ {
		for _, value := range []string{"", "event", "réseau-北", "cafe\u0301", "\ufffd", "\xff", "\x80", "\xc0\xaf", "\xed\xa0\x80", "\xe2\x82", "\xf4\x90\x80\x80"} {
			f.Add(field, value)
		}
	}
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	pub := key.Public().(ed25519.PublicKey)
	f.Fuzz(func(t *testing.T, field uint8, value string) {
		if len(value) > 256 {
			t.Skip()
		}
		event := Event{1, "event", "remote", "local", "listing.publish", "listing", 100, 200}
		fields := []*string{&event.ID, &event.Issuer, &event.Audience, &event.Kind, &event.Listing}
		*fields[int(field)%len(fields)] = value
		env, err := Sign(event, key)
		if !utf8.ValidString(value) {
			if !errors.Is(err, ErrInvalid) || len(env.Payload) != 0 || len(env.Signature) != 0 {
				t.Fatalf("signed malformed string: error=%v", err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		var decoded Event
		if err := json.Unmarshal(env.Payload, &decoded); err != nil || decoded != event {
			t.Fatalf("signed strings changed: %+v, error=%v", decoded, err)
		}
		canonical, err := json.Marshal(decoded)
		if err != nil || !bytes.Equal(canonical, env.Payload) {
			t.Fatal("valid strings produced noncanonical payload")
		}
		signed := append([]byte("circula.discovery.v1\x00"), env.Payload...)
		if !ed25519.Verify(pub, signed, env.Signature) {
			t.Fatal("signature does not authenticate exact payload with draft domain prefix")
		}
	})
}
