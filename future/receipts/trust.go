package receipts

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"time"
)

type keyScope struct{ issuer, network, id string }

// LocalTrustStore is immutable after construction and holds public keys only.
// Loading a file is a trust decision made by the caller, not by a certificate.
type LocalTrustStore struct{ keys map[keyScope]TrustedKey }

func (s *LocalTrustStore) Lookup(ctx context.Context, issuer, network, id string) (TrustedKey, error) {
	if ctx.Err() != nil {
		return TrustedKey{}, ctx.Err()
	}
	if s == nil {
		return TrustedKey{}, ErrUnknownKey
	}
	key, ok := s.keys[keyScope{issuer, network, id}]
	if !ok {
		return TrustedKey{}, ErrUnknownKey
	}
	key.Public = append(ed25519.PublicKey(nil), key.Public...)
	return key, nil
}

type publicKeyRecord struct {
	Issuer    string   `json:"issuer"`
	Network   string   `json:"network"`
	ID        string   `json:"key_id"`
	State     KeyState `json:"state"`
	NotBefore string   `json:"not_before"`
	NotAfter  string   `json:"not_after"`
	RetiredAt string   `json:"retired_at,omitempty"`
	Public    string   `json:"public_key_base64url"`
}
type keySet struct {
	Schema string            `json:"schema"`
	Keys   []publicKeyRecord `json:"keys"`
}

// LoadTrustStore is a bounded offline config parser. Operator-authenticated file
// distribution is required; a downloaded certificate must never choose this input.
func LoadTrustStore(r io.Reader) (*LocalTrustStore, error) {
	raw, e := io.ReadAll(io.LimitReader(r, 65537))
	if e != nil || len(raw) > 65536 {
		return nil, errors.New("invalid public key set")
	}
	canonical, e := canonicalJSON(raw, 65536)
	if e != nil {
		return nil, errors.New("invalid public key set")
	}
	var set keySet
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&set); e != nil || set.Schema != "circula-trusted-keys/v1" || len(set.Keys) == 0 || len(set.Keys) > 64 {
		return nil, errors.New("invalid public key set")
	}
	reencoded, _ := json.Marshal(set)
	recanonical, e := canonicalJSON(reencoded, 65536)
	if e != nil || !bytes.Equal(canonical, recanonical) {
		return nil, errors.New("invalid public key schema")
	}
	store := &LocalTrustStore{keys: map[keyScope]TrustedKey{}}
	parseTime := func(v string) (time.Time, error) {
		if len(v) > 40 {
			return time.Time{}, errors.New("invalid key time")
		}
		t, e := time.Parse(time.RFC3339Nano, v)
		if e != nil || t.UTC().Format(time.RFC3339Nano) != v {
			return time.Time{}, errors.New("invalid key time")
		}
		return t, nil
	}
	for _, record := range set.Keys {
		before, e := parseTime(record.NotBefore)
		if e != nil {
			return nil, e
		}
		after, e := parseTime(record.NotAfter)
		if e != nil {
			return nil, e
		}
		var retirement time.Time
		if record.RetiredAt != "" {
			retirement, e = parseTime(record.RetiredAt)
			if e != nil {
				return nil, e
			}
		}
		metadata := KeyMetadata{record.Issuer, record.Network, record.ID, record.State, before, after, retirement}
		if !metadataValid(metadata) || (metadata.State == KeyActive && !retirement.IsZero()) || (metadata.State == KeyRetired && retirement.IsZero()) {
			return nil, errors.New("invalid key metadata")
		}
		public, e := strictBase64(record.Public, ed25519.PublicKeySize)
		if e != nil || len(public) != ed25519.PublicKeySize {
			return nil, errors.New("invalid public key")
		}
		scope := keyScope{metadata.Issuer, metadata.Network, metadata.ID}
		if _, exists := store.keys[scope]; exists {
			return nil, errors.New("duplicate public key scope")
		}
		store.keys[scope] = TrustedKey{metadata, ed25519.PublicKey(public)}
	}
	return store, nil
}
