package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const bad = "wrong password!!"

// SA2-10: an attacker address locks only itself; the victim from another
// address still signs in, and the lock never exceeds 15 minutes.
func TestLoginLockIsPerAddress(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	atk := service.WithClientIP(e.Ctx, "203.0.113.9")
	vic := service.WithClientIP(e.Ctx, "198.51.100.7")
	for i := 0; i < 5; i++ {
		_, err := e.Svc.LoginStep(atk, "ana@example.org", bad)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, err := e.Svc.LoginStep(atk, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	// the victim is not locked by the attacker's address
	if _, err := e.Svc.LoginStep(vic, "ana@example.org", pw); err != nil {
		t.Fatalf("victim locked out by attacker address: %v", err)
	}
	// success from the victim address must not unlock the attacker's pair
	_, err = e.Svc.LoginStep(atk, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	// lock is capped at 15 minutes even after repeated failures
	e.Clock.Advance(16 * time.Minute)
	for i := 0; i < 3; i++ {
		_, err = e.Svc.LoginStep(atk, "ana@example.org", bad)
		wantCode(t, err, domain.ErrUnauthenticated)
		e.Clock.Advance(16 * time.Minute)
	}
	if _, err := e.Svc.LoginStep(atk, "ana@example.org", pw); err != nil {
		t.Fatalf("lock longer than the cap: %v", err)
	}
}

// IPv6 clients share a /64 bucket.
func TestLoginLockIPv6Prefix(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	for i := 0; i < 5; i++ {
		ctx := service.WithClientIP(e.Ctx, fmt.Sprintf("2001:db8:1:2::%x", i+1))
		_, err := e.Svc.LoginStep(ctx, "ana@example.org", bad)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, err := e.Svc.LoginStep(service.WithClientIP(e.Ctx, "2001:db8:1:2:ffff::1"), "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
}

// SA2-10: guessing spread over many addresses is still bounded by the
// login-wide cap, and the operator can clear it.
func TestLoginGlobalCapAndOperatorUnlock(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	for i := 0; i < 100; i++ {
		ctx := service.WithClientIP(e.Ctx, fmt.Sprintf("10.1.%d.%d", i/200, i%200+1))
		_, err := e.Svc.LoginStep(ctx, "ana@example.org", bad)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	fresh := service.WithClientIP(e.Ctx, "192.0.2.77")
	_, err := e.Svc.LoginStep(fresh, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)

	if _, err := e.Svc.OperatorUnlockLogin(e.Ctx, domain.Actor{PersonID: "x"}, "ana@example.org"); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("non-operator unlock: %v", err)
	}
	if _, err := e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("t"), "ghost@example.org"); domain.CodeOf(err) != domain.CodeNotFound {
		t.Errorf("unknown login: %v", err)
	}
	n, err := e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("t"), "Ana@example.org")
	if err != nil || n < 100 {
		t.Fatalf("unlock: n=%d err=%v", n, err)
	}
	if _, err := e.Svc.LoginStep(fresh, "ana@example.org", pw); err != nil {
		t.Fatalf("after unlock: %v", err)
	}
	var detail string
	if err := e.Store.DB().QueryRow(`SELECT summary_after FROM audit_events WHERE action = ?`, domain.ActLoginUnlocked).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(detail, "password") || strings.Contains(detail, "wrong") {
		t.Errorf("audit detail: %s", detail)
	}
}

// Idle counters decay after a day: old failures do not lock a login later.
func TestLoginCounterDecays(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	ctx := service.WithClientIP(e.Ctx, "203.0.113.9")
	for i := 0; i < 4; i++ {
		_, _ = e.Svc.LoginStep(ctx, "ana@example.org", bad)
	}
	e.Clock.Advance(25 * time.Hour)
	_, err := e.Svc.LoginStep(ctx, "ana@example.org", bad) // would be the 5th without decay
	wantCode(t, err, domain.ErrUnauthenticated)
	if _, err := e.Svc.LoginStep(ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("decayed counter still locked: %v", err)
	}
}

// SA2-10: recovery lockout is per address too, and unlock clears it.
func TestRecoverLockIsPerAddress(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	codes := codesFor(t, e, ana)
	atk := service.WithClientIP(e.Ctx, "203.0.113.9")
	for i := 0; i < 5; i++ {
		wantCode(t, e.Svc.RecoverPassword(atk, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW), domain.ErrUnauthenticated)
	}
	wantCode(t, e.Svc.RecoverPassword(atk, "ana@example.org", codes[0], newPW), domain.ErrRateLimited)
	vic := service.WithClientIP(e.Ctx, "198.51.100.7")
	if err := e.Svc.RecoverPassword(vic, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatalf("victim recovery locked by attacker: %v", err)
	}
	// operator unlock also clears recovery locks
	for i := 0; i < 5; i++ {
		_ = e.Svc.RecoverPassword(atk, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW)
	}
	tu.Must(e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("t"), "ana@example.org"))
	wantCode(t, e.Svc.RecoverPassword(atk, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW), domain.ErrUnauthenticated)
}

// SA2-13: a TOTP challenge issued before a reset or recovery cannot finish.
func TestResetsDeletePendingMFAChallenges(t *testing.T) {
	setup := func(t *testing.T) (*tu.Env, domain.Actor, string, string) {
		e := tu.NewEnv(t)
		admin := e.Person("founder@example.org")
		net, _ := e.Network(admin, "mfa-net", nil)
		bob, bobM := e.Member(admin, net, "bob@example.org")
		secret, _ := enrol(t, e, bob)
		_ = bobM
		return e, admin, secret, net.ID + "|" + bobM.ID
	}
	pending := func(e *tu.Env) string { return login(e, "bob@example.org").MFAChallenge }
	finish := func(t *testing.T, e *tu.Env, ch, secret string) error {
		_, _, err := e.Svc.CompleteMFA(e.Ctx, ch, totpAt(t, secret, now(e)))
		return err
	}
	t.Run("admin reset", func(t *testing.T) {
		e, admin, secret, ids := setup(t)
		ch := pending(e)
		p := strings.Split(ids, "|")
		tu.Must(e.Svc.AdminResetPassword(e.Ctx, admin, p[0], p[1], "lost password, verified"))
		if err := finish(t, e, ch, secret); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("challenge survived admin reset: %v", err)
		}
	})
	t.Run("operator reset", func(t *testing.T) {
		e, _, secret, _ := setup(t)
		ch := pending(e)
		tu.Must(e.Svc.OperatorResetPassword(e.Ctx, domain.SystemActor("t"), "bob@example.org", "ticket 1"))
		if err := finish(t, e, ch, secret); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("challenge survived operator reset: %v", err)
		}
	})
	t.Run("recovery", func(t *testing.T) {
		e := tu.NewEnv(t)
		bob := e.Person("bob@example.org")
		secret, _ := enrol(t, e, bob)
		codes := codesFor(t, e, bob)
		ch := pending(e)
		if err := e.Svc.RecoverPassword(e.Ctx, "bob@example.org", codes[0], newPW); err != nil {
			t.Fatal(err)
		}
		if err := finish(t, e, ch, secret); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("challenge survived recovery: %v", err)
		}
	})
}
