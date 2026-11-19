package receipts

import (
	"context"
	"crypto/ed25519"
	jose "github.com/go-jose/go-jose/v4"
)

// Health contains low-cardinality operational state only. A future host can add
// it to its existing authorized diagnostics; no public endpoint is registered.
type Health struct {
	Enabled bool   `json:"enabled"`
	Ready   bool   `json:"ready"`
	Reason  string `json:"reason"`
}

func (i Issuer) Health(ctx context.Context, network string) Health {
	if !i.Enabled || !i.Receipts.Enabled {
		return Health{Reason: "disabled"}
	}
	unavailable := func(reason string) Health { return Health{Enabled: true, Reason: reason} }
	if ctx.Err() != nil {
		return unavailable("cancelled")
	}
	if i.NetworkEnabled == nil || i.Receipts.NetworkEnabled == nil || i.Keys == nil || i.Now == nil || i.Receipts.Reader == nil || !safe(i.IssuerID) || !safe(network) {
		return unavailable("configuration")
	}
	if !i.NetworkEnabled(ctx, network) || !i.Receipts.NetworkEnabled(ctx, network) {
		return unavailable("network_disabled")
	}
	key, e := i.Keys.Current(ctx, network)
	if ctx.Err() != nil {
		return unavailable("cancelled")
	}
	if e != nil || key.Handle == nil {
		return unavailable("key_unavailable")
	}
	m := key.Metadata
	if !metadataValid(m) || m.Network != network || m.Issuer != i.IssuerID || m.State != KeyActive || !m.RetiredAt.IsZero() || !inWindow(m, i.Now()) {
		return unavailable("key_ineligible")
	}
	public := key.Handle.Public()
	if public == nil {
		return unavailable("key_ineligible")
	}
	p, ok := public.Key.(ed25519.PublicKey)
	if !ok || len(p) != ed25519.PublicKeySize || (public.Algorithm != "" && public.Algorithm != string(jose.EdDSA)) || (public.KeyID != "" && public.KeyID != m.ID) {
		return unavailable("key_ineligible")
	}
	supported := false
	for _, alg := range key.Handle.Algs() {
		if alg == jose.EdDSA {
			supported = true
		}
	}
	if !supported {
		return unavailable("key_ineligible")
	}
	if ctx.Err() != nil {
		return unavailable("cancelled")
	}
	return Health{true, true, "ready"}
}
