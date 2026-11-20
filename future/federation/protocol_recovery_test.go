package federation

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func recoveryDiscovery() Discovery {
	return Discovery{
		Version: 2, ID: "event", Issuer: "remote", Audience: "local",
		Kind: "listing.publish", Listing: "listing", Revision: 1,
		Issued: 1800000000, Expires: 1800003600,
		Content: &SharedListing{"Title", "Description", "category", "area"},
	}
}

func recoveryStringField(event *Discovery, kid *string, field uint8, value string) {
	switch field % 10 {
	case 0:
		event.ID = value
	case 1:
		event.Issuer = value
	case 2:
		event.Audience = value
	case 3:
		event.Kind = value
	case 4:
		event.Listing = value
	case 5:
		event.Content.Title = value
	case 6:
		event.Content.Description = value
	case 7:
		event.Content.Category = value
	case 8:
		event.Content.Area = value
	case 9:
		*kid = value
	}
}

func TestRecoveredV2RejectsMalformedStringsBeforeSigning(t *testing.T) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize))
	fields := []string{"id", "issuer", "audience", "kind", "listing", "title", "description", "category", "area", "kid"}
	for field, name := range fields {
		for i, malformed := range []string{"\xff", "\x80", "\xc0\xaf", "\xed\xa0\x80", "\xe2\x82", "\xf4\x90\x80\x80"} {
			t.Run(fmt.Sprintf("%s/%d", name, i), func(t *testing.T) {
				event, kid := recoveryDiscovery(), "key"
				recoveryStringField(&event, &kid, uint8(field), "value"+malformed)
				wire, err := SignDiscovery(event, kid, key)
				if err == nil || wire != "" {
					t.Fatalf("malformed v2 string accepted: wire=%q err=%v", wire, err)
				}
			})
		}
	}
}

func TestRecoveredV2PreservesValidPublicUnicode(t *testing.T) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize))
	event := recoveryDiscovery()
	event.Content = &SharedListing{
		Title: "literal \uFFFD cafe\u0301 🧭", Description: "<>&\u2028\u2029\n\t",
		Category: "réseau", Area: "北",
	}
	wire, err := SignDiscovery(event, "key", key)
	if err != nil {
		t.Fatal(err)
	}
	checkRecoveredV2Fidelity(t, event, "key", key.Public().(ed25519.PublicKey), wire)
	// V2 identifiers deliberately remain ASCII, even though public text and
	// the separate legacy v1 API can represent valid non-ASCII strings.
	event.ID = "literal-\uFFFD"
	if wire, err := SignDiscovery(event, "key", key); err == nil || wire != "" {
		t.Fatal("v2 identifier grammar was broadened", wire, err)
	}
}

func checkRecoveredV2Fidelity(t *testing.T, event Discovery, kid string, key ed25519.PublicKey, wire string) {
	t.Helper()
	parts := strings.Split(wire, ".")
	if len(parts) != 3 || len(wire) > MaxWire {
		t.Fatal("invalid compact wire bounds")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(event)
	if err != nil || !bytes.Equal(payload, want) {
		t.Fatal("signed payload did not preserve exact event JSON", err)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(key, []byte(parts[0]+"."+parts[1]), signature) {
		t.Fatal("compact signature does not cover the exact protected header/payload", err)
	}
	selector, err := WireKeyID(wire)
	if err != nil || selector != kid {
		t.Fatal("key selector changed", selector, err)
	}
	got, err := VerifyDiscovery(wire, event.Audience, event.Issuer, kid, key, time.Unix(event.Issued, 0))
	if err != nil || !reflect.DeepEqual(got, event) {
		t.Fatalf("valid v2 event changed: got=%+v want=%+v err=%v", got, event, err)
	}
}

func FuzzRecoveredV2StringFidelity(f *testing.F) {
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x31}, ed25519.SeedSize))
	public := key.Public().(ed25519.PublicKey)
	for field := uint8(0); field < 10; field++ {
		for _, value := range []string{"event", "listing.publish", "literal-\uFFFD", "cafe\u0301", "<>&\u2028", "\xff", "\xed\xa0\x80", ""} {
			f.Add(field, value)
		}
	}
	f.Fuzz(func(t *testing.T, field uint8, value string) {
		if len(value) > 2048 {
			return
		}
		event, kid := recoveryDiscovery(), "key"
		recoveryStringField(&event, &kid, field, value)
		wire, err := SignDiscovery(event, kid, key)
		if err != nil {
			if wire != "" {
				t.Fatal("failed v2 signing returned a wire payload")
			}
			return
		}
		if !utf8.ValidString(value) {
			t.Fatal("malformed v2 field signed successfully")
		}
		checkRecoveredV2Fidelity(t, event, kid, public, wire)
	})
}
