package service_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-F05: final person erasure removes both login-bearing throttle families,
// including all address buckets, without unlocking another person's account.
func TestPersonErasureClearsLoginAndRecoveryThrottleFamilies(t *testing.T) {
	w := newWorld(t, nil)
	const login = "élise%_@example.org"
	person, member := w.Member(w.admin, w.net, login)
	other, _ := w.Network(w.admin, "other", nil)
	token, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, other.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, person, token, "Other membership"))
	// Prefix neighbours and a LIKE-wildcard lookalike must keep their counters.
	logins := []string{login, login + ".other", "prefix-" + login, "éliseab@example.org"}
	keys := func(login string) []string {
		var out []string
		for _, prefix := range []string{login, "recover: " + login} {
			out = append(out, prefix, prefix+" ip:192.0.2.1", prefix+" ip:2001:db8::/64")
		}
		return out
	}
	for _, candidate := range logins {
		for _, ip := range []string{"192.0.2.1", "2001:db8::1234"} {
			ctx := service.WithClientIP(w.Ctx, ip)
			_, err := w.Svc.LoginStep(ctx, candidate, "incorrect passphrase")
			wantCode(t, err, domain.ErrUnauthenticated)
			wantCode(t, w.Svc.RecoverPassword(ctx, candidate, "AAAA-AAAA-AAAA-AAAA", "replacement passphrase"), domain.ErrUnauthenticated)
		}
	}
	type throttle struct {
		Failures int
		Until    time.Time
	}
	snapshot := func() map[string]throttle {
		t.Helper()
		out := map[string]throttle{}
		if err := w.Store.View(w.Ctx, func(tx store.Tx) error {
			for _, candidate := range logins {
				for _, key := range keys(candidate) {
					n, until, found, err := tx.GetThrottle(key)
					if err != nil {
						return err
					}
					if found {
						out[key] = throttle{n, until}
					}
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := snapshot()
	if len(before) != len(logins)*6 {
		t.Fatalf("fixture created %d throttle rows, want %d", len(before), len(logins)*6)
	}
	w.depart(w.admin, member)
	if report := tu.Must(w.Svc.AnonymiseMember(w.Ctx, w.admin, w.net.ID, member.ID, "member request")); report.PersonErased {
		t.Fatal("person erased while another membership remained live")
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("member-only erasure changed person-wide throttle state")
	}
	tu.Must(w.Svc.DepartMember(w.Ctx, person, other.ID, service.DepartInput{}))
	if report := tu.Must(w.Svc.EraseMyData(w.Ctx, person, other.ID, "")); !report.PersonErased {
		t.Fatal("last membership erasure did not erase the person")
	}
	for _, key := range keys(login) {
		delete(before, key)
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		for _, key := range keys(login) {
			if _, remains := after[key]; remains {
				t.Errorf("erased login remains in throttle key %q", key)
			}
		}
		for key, want := range before {
			if got, found := after[key]; !found || got != want {
				t.Errorf("unrelated throttle %q changed: %+v, exists=%v", key, got, found)
			}
		}
	}
}
