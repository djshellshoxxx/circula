package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-006, SEC-007 (A3-F16, was TestA3Audit_StolenSessionLocksOwnerOutOfMFA):
// whoever holds a stolen session, without the password or the second factor,
// must not be able to keep the owner from signing in (the step that lets the
// owner change the password and end that session). Bad DisableMFA and
// RegenerateRecoveryCodes calls used to count on the sign-in counter and,
// with one call per lock window, re-arm it forever.
func TestStolenSessionCannotLockOwnerOutOfMFA(t *testing.T) {
	e := tu.NewEnv(t)
	victim := e.Person("ana@example.org")
	secret, _ := enrol(t, e, victim)
	stolen := victim // the attacker replays the victim's session; no password, no code
	hammer := func() {
		for i := 0; i < 8; i++ {
			_ = e.Svc.DisableMFA(e.Ctx, stolen, "not the password", "000000")
			_, _ = e.Svc.RegenerateRecoveryCodes(e.Ctx, stolen, "000000")
		}
	}
	hammer()
	// the session-side guesses are throttled...
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, stolen, "000000"); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("session second-factor guesses not throttled: %v", err)
	}
	for round := 0; round < 3; round++ {
		// ...but the owner, with the password and the authenticator, signs in.
		r := login(e, "ana@example.org")
		if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, secret, now(e))); err != nil {
			t.Errorf("round %d: owner with the right password and code is refused: %v", round, err)
		}
		e.Clock.Advance(15*time.Minute + time.Second) // the session lock runs out...
		hammer()                                      // ...and is hammered again at once
		e.Clock.Advance(31 * time.Second)
	}
}

// SEC-006 (A3-F16, A3-02 pattern): an expired sign-in second-factor lock grants
// half the allowance again, so one failure timed at its expiry does not lock
// the owner out for another 15 minutes.
func TestMFASignInLockNotRenewableByOneFailure(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	r := login(e, "ana@example.org")
	for i := 0; i < 5; i++ {
		if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, "000000"); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	e.Clock.Advance(15*time.Minute + time.Second)
	r = login(e, "ana@example.org")
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, "000000"); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("one failure at lock expiry: %v", err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, secret, now(e))); err != nil {
		t.Fatalf("owner refused after one failure at lock expiry: %v", err)
	}
}

// SEC-007: session second-factor guesses keep their own bound. Five bad
// regenerations lock regeneration and disabling (even with the right code)
// without touching sign-in; the operator unlock clears that lock too.
func TestMFASessionThrottleBoundAndUnlock(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	for i := 0; i < 5; i++ {
		if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, "000000"); domain.CodeOf(err) != domain.CodeValidation {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	code := totpAt(t, secret, now(e))
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, code); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("regenerate while locked: %v", err)
	}
	if err := e.Svc.DisableMFA(e.Ctx, a, pw, code); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("disable while locked: %v", err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, code); err != nil {
		t.Fatalf("sign-in locked by session guesses: %v", err)
	}
	// The sign-in success forgets the session counter as well.
	e.Clock.Advance(31 * time.Second)
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, totpAt(t, secret, now(e))); err != nil {
		t.Fatalf("regenerate after a successful sign-in: %v", err)
	}
	// The operator unlock clears a session lock.
	for i := 0; i < 5; i++ {
		_, _ = e.Svc.RegenerateRecoveryCodes(e.Ctx, a, "000000")
	}
	if n := tu.Must(e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("op"), "ana@example.org")); n < 1 {
		t.Fatalf("unlock cleared %d rows", n)
	}
	e.Clock.Advance(31 * time.Second)
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, totpAt(t, secret, now(e))); err != nil {
		t.Fatalf("regenerate after operator unlock: %v", err)
	}
}
