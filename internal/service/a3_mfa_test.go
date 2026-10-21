package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-03: a stolen session must not be able to bind an attacker's
// authenticator; enrolment needs the current password.
func TestA3_03_EnrolmentNeedsPassword(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, "not the password"); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("begin with wrong password: %v", err)
	}
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, ""); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("begin with empty password: %v", err)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM person_mfa`); n != 0 {
		t.Fatalf("a refused begin stored a secret (%d)", n)
	}
	en := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, a, pw))
	code := totpAt(t, en.Secret, now(e))
	if _, err := e.Svc.ConfirmMFAEnrolment(e.Ctx, a, "wrong password!!", code); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("confirm with wrong password: %v", err)
	}
	if st := tu.Must(e.Svc.MFAStatus(e.Ctx, a)); st.Enabled {
		t.Fatal("MFA enabled without the password")
	}
	if _, err := e.Svc.ConfirmMFAEnrolment(e.Ctx, a, pw, code); err != nil {
		t.Fatalf("confirm with password: %v", err)
	}
}

// A3-03: password guesses through enrolment are throttled, and the operator
// unlock clears that throttle.
func TestA3_03_EnrolmentPasswordGuessesThrottled(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	for i := 0; i < 5; i++ {
		_, _ = e.Svc.BeginMFAEnrolment(e.Ctx, a, "guess number "+string(rune('a'+i)))
	}
	_, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, pw)
	wantCode(t, err, domain.ErrRateLimited) // even the right password is refused while locked
	if _, err := e.Svc.OperatorUnlockLogin(e.Ctx, domain.SystemActor("t"), "ana@example.org"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, pw); err != nil {
		t.Fatalf("after unlock: %v", err)
	}
}

// A3-03: enabling MFA ends every other session but keeps the caller's own.
func TestA3_03_EnableRevokesOtherSessions(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	mine, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	stolen, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	a := tu.Must(e.Svc.Authenticate(e.Ctx, mine))
	en := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, a, pw))
	tu.Must(e.Svc.ConfirmMFAEnrolment(e.Ctx, a, pw, totpAt(t, en.Secret, now(e))))
	if _, err := e.Svc.Authenticate(e.Ctx, stolen); err == nil {
		t.Fatal("the other session survived enabling MFA")
	}
	if _, err := e.Svc.Authenticate(e.Ctx, mine); err != nil {
		t.Fatalf("the enrolling session was revoked: %v", err)
	}
}

// A3-03: operator reset-mfa removes the factor, codes, challenges and sessions,
// is audited as the operator, notifies the person and leaves password login working.
func TestA3_03_OperatorResetMFA(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	enrol(t, e, a)
	login(e, "ana@example.org") // leaves a pending challenge behind
	if n := count(t, e, `SELECT COUNT(*) FROM mfa_challenges WHERE person_id = ?`, a.PersonID); n == 0 {
		t.Fatal("expected a pending challenge")
	}
	sys := domain.SystemActor("cli-reset-mfa")
	if err := e.Svc.OperatorResetMFA(e.Ctx, a, "ana@example.org", "lost phone"); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("non-system actor: %v", err)
	}
	if err := e.Svc.OperatorResetMFA(e.Ctx, sys, "ana@example.org", "x"); domain.CodeOf(err) != domain.CodeValidation {
		t.Fatalf("reason required: %v", err)
	}
	if err := e.Svc.OperatorResetMFA(e.Ctx, sys, "ghost@example.org", "lost phone"); domain.CodeOf(err) != domain.CodeNotFound {
		t.Fatalf("unknown login: %v", err)
	}
	if err := e.Svc.OperatorResetMFA(e.Ctx, sys, "Ana@Example.org", "ticket 9: lost phone"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"person_mfa", "recovery_codes", "mfa_challenges"} {
		if n := count(t, e, `SELECT COUNT(*) FROM `+q+` WHERE person_id = ?`, a.PersonID); n != 0 {
			t.Errorf("%s rows left: %d", q, n)
		}
	}
	if n := count(t, e, `SELECT COUNT(*) FROM sessions WHERE revoked_at IS NULL AND person_id = ?`, a.PersonID); n != 0 {
		t.Errorf("%d live sessions", n)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM security_notices WHERE kind = ? AND person_id = ?`, domain.NoticeMFADisabled, a.PersonID); n != 1 {
		t.Errorf("notices: %d", n)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ? AND reason = ?`, domain.ActMFAReset, "ticket 9: lost phone"); n != 1 {
		t.Errorf("audit events: %d", n)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("password login after reset: %v", err)
	}
	if err := e.Svc.OperatorResetMFA(e.Ctx, sys, "ana@example.org", "again"); domain.CodeOf(err) != domain.CodeInvalidState {
		t.Fatalf("second reset: %v", err)
	}
}

// A3-04: RecoverPassword must not run Argon2 for a locked login or a wrong
// code, so guessing cannot be used to burn CPU.
func TestA3_04_RecoverSkipsHashingBeforeCodeCheck(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) {
		c.Argon = service.Argon2Params{MemoryKiB: 64 * 1024, Time: 3, Threads: 1, KeyLen: 32, SaltLen: 16}
	})
	a := e.Person("ana@example.org")
	codes := codesFor(t, e, a)
	start := time.Now()
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatal(err)
	}
	oneHash := time.Since(start)
	start = time.Now()
	for i := 0; i < 4; i++ { // wrong code, login not yet locked
		_ = e.Svc.RecoverPassword(e.Ctx, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW)
	}
	_ = e.Svc.RecoverPassword(e.Ctx, "nobody@example.org", "AAAA-AAAA-AAAA-AAAA", newPW)
	_ = e.Svc.RecoverPassword(e.Ctx, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW) // 5th: locks
	wantCode(t, e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW), domain.ErrRateLimited)
	if d := time.Since(start); d > 3*oneHash {
		t.Fatalf("7 refused recoveries took %v, one Argon2 hash takes %v: hashing runs before the code check", d, oneHash)
	}
	// the right code still works once the lock is over
	e.Clock.Advance(16 * time.Minute)
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", newPW); err != nil {
		t.Fatalf("new password: %v", err)
	}
}
