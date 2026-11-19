package federation

import (
	"crypto/ed25519"
	"testing"
	"time"
)

var fixtureNow = time.Unix(1800000000, 0)

func FuzzDiscoveryWire(f *testing.F) {
	f.Add("bad")
	f.Add("a.b.c")
	f.Fuzz(func(t *testing.T, wire string) {
		if len(wire) > MaxWire+1 {
			return
		}
		WireKeyID(wire)
		VerifyDiscovery(wire, "local", "remote", "key", make(ed25519.PublicKey, 32), fixtureNow)
	})
}
