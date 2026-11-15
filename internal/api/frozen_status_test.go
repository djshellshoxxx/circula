package api_test

import (
	"testing"
)

// The network resource says whether it is read-only; the reason is for administrators only.
func TestNetworkReadOnlyField(t *testing.T) {
	w := newWorld(t)
	type netResp struct {
		ReadOnly     bool    `json:"read_only"`
		FrozenSince  *string `json:"frozen_since"`
		FrozenReason string  `json:"frozen_reason"`
	}
	if got := decode[netResp](t, w.get(w.aliceT, "/network")); got.ReadOnly || got.FrozenSince != nil || got.FrozenReason != "" {
		t.Fatalf("unfrozen: %+v", got)
	}
	w.env.Svc.FreezeNetwork(w.net.ID, "ledger hash mismatch at seq 12")
	m := decode[netResp](t, w.get(w.aliceT, "/network"))
	if !m.ReadOnly || m.FrozenSince == nil || m.FrozenReason != "" {
		t.Fatalf("member view: %+v", m)
	}
	a := decode[netResp](t, w.get(w.adminT, "/network"))
	if !a.ReadOnly || a.FrozenSince == nil || a.FrozenReason != "ledger hash mismatch at seq 12" {
		t.Fatalf("admin view: %+v", a)
	}
	wantProblem(t, w.post(w.aliceT, "/listings", map[string]any{"kind": "offer", "title": "x"}, w.key()), 423, "network-frozen")
	w.env.Svc.UnfreezeNetwork(w.net.ID)
	if got := decode[netResp](t, w.get(w.aliceT, "/network")); got.ReadOnly {
		t.Fatalf("thawed: %+v", got)
	}
}
