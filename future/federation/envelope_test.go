package federation

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestSignedDiscoveryBoundaries(t *testing.T) {
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Unix(100, 0)
	event := Event{1, "event", "remote", "local", "listing.publish", "listing", 100, 200}
	env, e := Sign(event, priv)
	if e != nil {
		t.Fatal(e)
	}
	p := Peer{"local", "remote", true, pub}
	got, e := Verify(p, env, now)
	if e != nil || got != event {
		t.Fatal(got, e)
	}
	p.Enabled = false
	if _, e = Verify(p, env, now); e == nil {
		t.Fatal("disabled")
	}
	p.Enabled = true
	p.Local = "other"
	if _, e = Verify(p, env, now); e == nil {
		t.Fatal("audience")
	}
	p.Local = "local"
	if _, e = Verify(p, env, time.Unix(200, 0)); e == nil {
		t.Fatal("expiry")
	}
	event.Kind = "ledger.post"
	env, _ = Sign(event, priv)
	if _, e = Verify(p, env, now); e == nil {
		t.Fatal("settlement")
	}
	env.Payload[0] = 'x'
	if _, e = Verify(p, env, now); e == nil {
		t.Fatal("tamper")
	}
}
