package notifications

import (
	"bytes"
	"testing"
	"time"
)

func TestDedupeKeyRejectsInvalidUTF8(t *testing.T) {
	valid := [5]string{"network", "event", "member", "email", KindDispute}
	for i := range valid {
		invalid := valid
		invalid[i] += "\xff"
		if key, err := DedupeKey(invalid[0], invalid[1], invalid[2], invalid[3], invalid[4]); err == nil || key != "" {
			t.Errorf("invalid UTF-8 in field %d produced a dedupe key", i)
		}
	}
	a, err := DedupeKey("network", "event\ufffd", "member", "email", KindDispute)
	if err != nil {
		t.Fatal(err)
	}
	b, err := DedupeKey("network", "event\u00e9", "member", "email", KindDispute)
	if err != nil || a == b {
		t.Fatal("valid Unicode event identities must remain distinct", err)
	}
}

func TestActorAndEventRejectInvalidUTF8(t *testing.T) {
	for _, actor := range []Actor{
		{NetworkID: "n\xff", MemberID: "member"},
		{NetworkID: "network", MemberID: "m\xff"},
		{NetworkID: "network", MemberID: "member", SessionID: "s\xff"},
	} {
		if err := actor.validate(); err == nil {
			t.Error("actor accepted invalid UTF-8")
		}
	}
	for _, event := range []Event{
		{NetworkID: "n\xff", MemberID: "member", EventID: "event", Kind: KindDispute},
		{NetworkID: "network", MemberID: "m\xff", EventID: "event", Kind: KindDispute},
		{NetworkID: "network", MemberID: "member", EventID: "e\xff", Kind: KindDispute},
	} {
		if err := event.validate(); err == nil {
			t.Error("event accepted invalid UTF-8")
		}
	}
}

func TestSignedTokenDoesNotNormalizeInvalidIdentity(t *testing.T) {
	key := bytes.Repeat([]byte{0x6b}, 32)
	valid := tokenPayload{P: tokUnsubscribe, N: "network", M: "member", C: "email", T: "target"}
	for _, field := range []string{"purpose", "network", "member", "channel", "target"} {
		p := valid
		switch field {
		case "purpose":
			p.P += "\xff"
		case "network":
			p.N += "\xff"
		case "member":
			p.M += "\xff"
		case "channel":
			p.C += "\xff"
		case "target":
			p.T += "\xff"
		}
		if token := signToken(key, p); token != "" {
			t.Errorf("invalid UTF-8 in %s produced a signed token", field)
		}
	}
	valid.N = "network\ufffd"
	valid.M = "membre\u00e9"
	token := signToken(key, valid)
	decoded, err := verifyToken(key, token, tokUnsubscribe, time.Unix(1, 0))
	if err != nil || decoded != valid {
		t.Fatalf("valid Unicode identity changed: %+v %v", decoded, err)
	}
}
