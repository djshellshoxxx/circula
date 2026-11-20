package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-F22: the operator login argument uses the same canonicalization as
// registration and sign-in, while membership and expiry still gate authority.
func TestBreakGlassGrantNormalizesLogin(t *testing.T) {
	w := newWorld(t, nil)
	w.openBG(time.Hour)
	_, err := w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{
		Network: "demo", Login: "  ANA@Example.ORG  ", Reason: "restore administrator access",
	})
	noErr(t, err)
	if _, err := w.Svc.ListInvitations(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("normalized login did not grant effective admin: %v", err)
	}
	// Grant is temporary: no permanent administrator role was assigned.
	stored := tu.Must(w.Svc.MyMemberships(w.Ctx, w.a))
	if len(stored) != 1 || stored[0].HasRole(domain.RoleAdmin) {
		t.Fatalf("grant changed permanent roles: %+v", stored)
	}
	w.Clock.Advance(time.Hour)
	_, err = w.Svc.ListInvitations(w.Ctx, w.a, w.net.ID)
	wantCode(t, err, domain.ErrForbidden)
}
