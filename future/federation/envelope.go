// Package federation authenticates discovery-only envelopes. No HTTP transport,
// peer discovery, balance data, settlement or ledger authority is supplied.
package federation

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"
)

var ErrInvalid = errors.New("federation event rejected")

type Event struct {
	Version  int    `json:"version"`
	ID       string `json:"id"`
	Issuer   string `json:"issuer"`
	Audience string `json:"audience"`
	Kind     string `json:"kind"`
	Listing  string `json:"listing"`
	Issued   int64  `json:"issued"`
	Expires  int64  `json:"expires"`
}
type Envelope struct {
	Payload   []byte
	Signature []byte
}
type Peer struct {
	Local, Remote string
	Enabled       bool
	Key           ed25519.PublicKey
}

// Draft wire format: signed exact JSON bytes with a domain-separation prefix.
// Keys must come from bilateral configuration, never from the envelope.
func message(b []byte) []byte { return append([]byte("circula.discovery.v1\x00"), b...) }
func Sign(e Event, key ed25519.PrivateKey) (Envelope, error) {
	if len(key) != ed25519.PrivateKeySize {
		return Envelope{}, ErrInvalid
	}
	// encoding/json replaces malformed UTF-8. Refuse it before signing so
	// success always preserves the caller's strings in the signed payload.
	for _, value := range []string{e.ID, e.Issuer, e.Audience, e.Kind, e.Listing} {
		if !utf8.ValidString(value) {
			return Envelope{}, ErrInvalid
		}
	}
	b, err := json.Marshal(e)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{b, ed25519.Sign(key, message(b))}, nil
}

// Verify deliberately does not consume a replay ID. After verification, adapters
// MUST atomically persist peer+event ID and apply publication/revocation. A seen
// callback here would introduce a check/use race and false exactly-once claims.
func Verify(p Peer, env Envelope, now time.Time) (Event, error) {
	if !p.Enabled || p.Local == "" || p.Remote == "" || p.Local == p.Remote || len(p.Key) != ed25519.PublicKeySize || len(env.Payload) > 4096 || len(env.Signature) != ed25519.SignatureSize || !ed25519.Verify(p.Key, message(env.Payload), env.Signature) {
		return Event{}, ErrInvalid
	}
	var e Event
	if json.Unmarshal(env.Payload, &e) != nil {
		return Event{}, ErrInvalid
	}
	// Require this draft format's exact encoding: rejects duplicate/unknown fields,
	// alternate number spellings and ambiguous signed representations.
	canonical, err := json.Marshal(e)
	if err != nil || string(canonical) != string(env.Payload) {
		return Event{}, ErrInvalid
	}
	if e.Version != 1 || e.ID == "" || len(e.ID) > 128 || e.Listing == "" || len(e.Listing) > 128 || e.Issuer != p.Remote || e.Audience != p.Local || (e.Kind != "listing.publish" && e.Kind != "listing.revoke") {
		return Event{}, ErrInvalid
	}
	issued, expires := time.Unix(e.Issued, 0), time.Unix(e.Expires, 0)
	if issued.After(now.Add(5*time.Minute)) || !expires.After(now) || !expires.After(issued) || expires.Sub(issued) > 24*time.Hour {
		return Event{}, ErrInvalid
	}
	return e, nil
}
