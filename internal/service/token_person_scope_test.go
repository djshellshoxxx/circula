package service_test

import (
	"reflect"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-02: a credential delegated to one network cannot enumerate or revoke
// person-wide credentials, or change preferences shared by other networks.
func TestNetworkScopedTokenCannotManagePersonResources(t *testing.T) {
	w := newWorld(t, nil)
	other, _ := w.Network(w.admin, "other", nil)
	token := func(name, netID string) (string, *domain.APIToken) {
		return tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.admin, service.TokenInput{
			Name: name, NetworkID: netID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite},
		}))
	}
	plain, local := token("delegated integration", w.net.ID)
	_, foreign := token("private other-network integration", other.ID)
	globalPlain, global := token("all networks", "")
	scoped := tu.Must(w.Svc.AuthenticateToken(w.Ctx, plain))
	full := tu.Must(w.Svc.AuthenticateToken(w.Ctx, globalPlain))
	tu.Must(0, w.Svc.SetTimeZone(w.Ctx, full, "Europe/Berlin"))
	before := tu.Must(w.Svc.ListAPITokens(w.Ctx, full))

	t.Run("list", func(t *testing.T) {
		rows, err := w.Svc.ListAPITokens(w.Ctx, scoped)
		wantCode(t, err, domain.ErrForbidden)
		if len(rows) != 0 {
			t.Fatalf("scoped token exposed token metadata: %+v", rows)
		}
	})
	t.Run("revoke", func(t *testing.T) {
		for _, id := range []string{foreign.ID, global.ID, local.ID, "missing-token"} {
			t.Run(id, func(t *testing.T) {
				wantCode(t, w.Svc.RevokeAPIToken(w.Ctx, scoped, id), domain.ErrForbidden)
			})
		}
	})
	t.Run("time zone write", func(t *testing.T) {
		wantCode(t, w.Svc.SetTimeZone(w.Ctx, scoped, "Asia/Tokyo"), domain.ErrForbidden)
	})
	if after := tu.Must(w.Svc.ListAPITokens(w.Ctx, full)); !reflect.DeepEqual(before, after) {
		t.Error("refused scoped commands changed token state")
	}
	// Reading the caller's display preference is still needed by GET /me and
	// page rendering; it does not reveal any other network's identifiers.
	if zone := tu.Must(w.Svc.GetTimeZone(w.Ctx, scoped)); zone != "Europe/Berlin" {
		t.Errorf("scoped read returned time zone %q, want unchanged Europe/Berlin", zone)
	}
	// An unrestricted credential retains person-wide management rights.
	tu.Must(0, w.Svc.SetTimeZone(w.Ctx, full, "Asia/Tokyo"))
	if err := w.Svc.RevokeAPIToken(w.Ctx, full, foreign.ID); err != nil {
		t.Fatalf("unrestricted credential could not revoke the other-network token: %v", err)
	}
	rows := tu.Must(w.Svc.ListAPITokens(w.Ctx, full))
	if len(rows) != 3 {
		t.Fatalf("unrestricted token listing has %d tokens, want 3", len(rows))
	}
	for _, row := range rows {
		if row.ID == foreign.ID && row.RevokedAt == nil {
			t.Fatal("unrestricted credential did not revoke the other-network token")
		}
	}
}
