package federation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	jose "github.com/go-jose/go-jose/v4"
	"testing"
)

func signedRaw(t *testing.T, key ed25519.PrivateKey, payload []byte, options *jose.SignerOptions) string {
	t.Helper()
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: key}, options)
	if e != nil {
		t.Fatal(e)
	}
	signed, e := signer.Sign(payload)
	if e != nil {
		t.Fatal(e)
	}
	wire, e := signed.CompactSerialize()
	if e != nil {
		t.Fatal(e)
	}
	return wire
}
func TestSignedButMalformedPayloadAndHeaders(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	event := fixtureEvent("event", "listing.publish", 1)
	raw, _ := json.Marshal(event)
	options := (&jose.SignerOptions{}).WithType(ProtocolType).WithHeader("kid", "key")
	for _, payload := range [][]byte{append([]byte(`{"private_balance":1,`), raw[1:]...), append([]byte(`{"version":2,`), raw[1:]...), []byte(`{"version":2}`), append([]byte(" "), raw...), []byte{255}} {
		wire := signedRaw(t, key, payload, options)
		if _, e := VerifyDiscovery(wire, "local", "remote", "key", pub, fixtureNow); e == nil {
			t.Fatal("signed invalid schema accepted")
		}
	}
	for _, o := range []*jose.SignerOptions{(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "key"), (&jose.SignerOptions{}).WithType(ProtocolType).WithHeader("kid", "key").WithHeader("jku", "https://example.com/key")} {
		wire := signedRaw(t, key, raw, o)
		if _, e := VerifyDiscovery(wire, "local", "remote", "key", pub, fixtureNow); e == nil {
			t.Fatal("header confusion")
		}
	}
}
