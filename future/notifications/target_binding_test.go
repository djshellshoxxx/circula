package notifications

import (
	"bytes"
	"errors"
	"testing"
)

func targetBindingKeyring(t *testing.T) *Keyring {
	t.Helper()
	k, err := NewKeyring("binding-test", map[string][]byte{"binding-test": bytes.Repeat([]byte{0x5a}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestTargetAADSeparatesAcceptedActorScopes(t *testing.T) {
	a := Actor{NetworkID: "network|extra", MemberID: "member"}
	b := Actor{NetworkID: "network", MemberID: "extra|member"}
	for _, actor := range []Actor{a, b} {
		if err := actor.validate(); err != nil {
			t.Fatalf("fixture must be an accepted actor: %v", err)
		}
	}
	k := targetBindingKeyring(t)
	for _, kind := range []string{"push", "email"} {
		t.Run(kind, func(t *testing.T) {
			aad := sealAAD(kind, a.NetworkID, a.MemberID, "target-id")
			foreignAAD := sealAAD(kind, b.NetworkID, b.MemberID, "target-id")
			if bytes.Equal(aad, foreignAAD) {
				t.Fatal("different accepted actor scopes share authenticated data")
			}
			blob, err := k.Seal([]byte("private target"), aad)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := k.Open(blob, foreignAAD); !errors.Is(err, ErrSealed) {
				t.Fatalf("foreign actor opened target: %v", err)
			}
			plain, _, err := k.Open(blob, aad)
			if err != nil || string(plain) != "private target" {
				t.Fatalf("original actor could not open target: %q %v", plain, err)
			}
		})
	}
}

func TestTargetAADPreservesLegacyUnambiguousTargets(t *testing.T) {
	k := targetBindingKeyring(t)
	legacy := []byte("ntf/v1|email|network|member|target-id")
	current := sealAAD("email", "network", "member", "target-id")
	if !bytes.Equal(legacy, current) {
		t.Fatal("unambiguous legacy binding changed")
	}
	blob, err := k.Seal([]byte("owner@example.test"), legacy)
	if err != nil {
		t.Fatal(err)
	}
	plain, _, err := k.Open(blob, current)
	if err != nil || string(plain) != "owner@example.test" {
		t.Fatalf("legacy target did not survive: %q %v", plain, err)
	}
}

func TestTargetAADRejectsLegacyAmbiguousTargets(t *testing.T) {
	k := targetBindingKeyring(t)
	legacy := []byte("ntf/v1|push|network|extra|member|target-id")
	blob, err := k.Seal([]byte("private target"), legacy)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []Actor{
		{NetworkID: "network|extra", MemberID: "member"},
		{NetworkID: "network", MemberID: "extra|member"},
	} {
		if _, _, err := k.Open(blob, sealAAD("push", actor.NetworkID, actor.MemberID, "target-id")); !errors.Is(err, ErrSealed) {
			t.Fatalf("ambiguous legacy binding was accepted for %+v: %v", actor, err)
		}
	}
}

func TestTargetAADAuthenticatesEveryComponent(t *testing.T) {
	k := targetBindingKeyring(t)
	parts := [4]string{"push", "network|extra", "member", "target-id"}
	aad := sealAAD(parts[0], parts[1], parts[2], parts[3])
	blob, err := k.Seal([]byte("private target"), aad)
	if err != nil {
		t.Fatal(err)
	}
	for i := range parts {
		changed := parts
		changed[i] += "|changed"
		if _, _, err := k.Open(blob, sealAAD(changed[0], changed[1], changed[2], changed[3])); !errors.Is(err, ErrSealed) {
			t.Fatalf("changed component %d opened target: %v", i, err)
		}
	}
	// Raw invalid UTF-8 bytes must not normalize into the same binding.
	if bytes.Equal(sealAAD("push", "network|\xff", "member", "id"), sealAAD("push", "network|\xfe", "member", "id")) {
		t.Fatal("different identifier bytes share authenticated data")
	}
}
