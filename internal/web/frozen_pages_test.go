package web

import (
	"strings"
	"testing"
)

// A frozen network shows an administrator the banner with the reason, and every member a
// read-only notice without it; reads keep working.
func TestFrozenNetworkVisibility(t *testing.T) {
	w := newWorld(t)
	w.env.Svc.FreezeNetwork(w.net.ID, "ledger hash mismatch at seq 12")
	ab := w.browser(t)
	ab.login("admin@example.org")
	p := ab.get("/n/town/admin")
	ab.mustContain(p, "This network is read-only", "Frozen since", "ledger hash mismatch at seq 12")

	mb := w.browser(t)
	mb.login("alice@example.org")
	for _, path := range []string{"/n/town", "/n/town/listings/new", "/n/town/trades/new"} {
		p := mb.get(path)
		if p.Status != 200 {
			t.Fatalf("%s: %d", path, p.Status)
		}
		mb.mustContain(p, "read-only while its ledger is being checked")
		if strings.Contains(p.Body, "hash mismatch") {
			t.Errorf("%s leaks the reason to a member", path)
		}
	}
	w.env.Svc.UnfreezeNetwork(w.net.ID)
	if p := ab.get("/n/town/admin"); strings.Contains(p.Body, "This network is read-only") {
		t.Error("banner remains after thaw")
	}
	if p := mb.get("/n/town"); strings.Contains(p.Body, "read-only while") {
		t.Error("notice remains after thaw")
	}
}
