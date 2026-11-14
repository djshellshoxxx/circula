package api

import (
	"context"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// BR-04: an administrator's network-wide export (ledger, members, audit,
// full) holds the erased member's data, so erasure drops every such artefact
// of that network. Other people's own-data exports and exports of other
// networks stay.
func TestBR04_ErasureDropsNetworkWideExports(t *testing.T) {
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.OnMemberErased = OnMemberErased })
	ctx := context.Background()
	admin := env.Person("admin")
	net, _ := env.Network(admin, "acme", nil)
	other, _ := env.Network(admin, "other", nil)
	alice, _ := env.Member(admin, net, "alice")
	bob, _ := env.Member(admin, net, "bob")
	h := NewHandlerWithOptions(env.Svc, Options{}).(*server)

	put := func(id, netID, person, scope string) {
		t.Helper()
		if !h.jobs.putExport(&exportEntry{net: netID, person: person, at: time.Now(),
			dto: exportDTO{ID: id, Scope: scope}, file: []byte(id)}) {
			t.Fatalf("put %s", id)
		}
	}
	for _, sc := range []string{"ledger", "members", "audit", "full"} {
		put("net-"+sc, net.ID, admin.PersonID, sc)
	}
	put("other-full", other.ID, admin.PersonID, "full")
	put("bob-mine", net.ID, bob.PersonID, "mine")

	if _, err := env.Svc.DepartMember(ctx, alice, net.ID, service.DepartInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Svc.EraseMyData(ctx, alice, net.ID, ""); err != nil {
		t.Fatal(err)
	}
	for _, sc := range []string{"ledger", "members", "audit", "full"} {
		if h.jobs.getExport(net.ID, admin.PersonID, "net-"+sc) != nil {
			t.Errorf("%s export of the erased member's network survives", sc)
		}
	}
	if h.jobs.getExport(other.ID, admin.PersonID, "other-full") == nil {
		t.Error("export of another network was dropped")
	}
	if h.jobs.getExport(net.ID, bob.PersonID, "bob-mine") == nil {
		t.Error("another member's own-data export was dropped")
	}
	if want := int64(len("other-full") + len("bob-mine")); h.jobs.bytes != want {
		t.Errorf("job store byte count = %d, want %d", h.jobs.bytes, want)
	}
}
