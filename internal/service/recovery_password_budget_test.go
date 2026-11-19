package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-008 / AUD4-F04: a successful password reset invalidates the old
// current-password proof budget. Otherwise the forced change can remain
// locked by guesses made before recovery revoked the compromised session.
func TestPasswordRecoveryRestoresCurrentPasswordAttemptBudget(t *testing.T) {
	for _, method := range []string{"administrator", "operator", "recovery code"} {
		t.Run(method, func(t *testing.T) {
			w := newWorld(t, nil)
			codes := tu.Must(w.Svc.GenerateRecoveryCodes(w.Ctx, w.a, pw))
			locked := false
			for range 30 {
				err := w.Svc.ChangePassword(w.Ctx, w.a, "wrong current password", "a different long password 5")
				if domain.CodeOf(err) == domain.CodeRateLimited {
					locked = true
					break
				}
				if domain.CodeOf(err) != domain.CodeUnauthenticated {
					t.Fatalf("could not establish a current-password lock: %v", err)
				}
			}
			if !locked {
				t.Fatal("current-password guesses were not throttled")
			}
			// A refused reset must not clear another person's lock.
			if _, err := w.Svc.AdminResetPassword(w.Ctx, w.b, w.net.ID, w.aM.ID, "not an administrator"); err == nil {
				t.Fatal("non-administrator reset succeeded")
			}
			if err := w.Svc.ChangePassword(w.Ctx, w.a, pw, "a different long password 5"); domain.CodeOf(err) != domain.CodeRateLimited {
				t.Fatalf("refused reset cleared the lock: %v", err)
			}

			var replacement string
			switch method {
			case "administrator":
				replacement = tu.Must(w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, w.aM.ID, "account recovery"))
			case "operator":
				replacement = tu.Must(w.Svc.OperatorResetPassword(w.Ctx, domain.SystemActor("recovery-test"), "ana@example.org", "account recovery"))
			case "recovery code":
				replacement = "a recovered account password 29"
				noErr(t, w.Svc.RecoverPassword(w.Ctx, "ana@example.org", codes[0], replacement))
			}
			token, _, err := w.Svc.Login(w.Ctx, "ana@example.org", replacement)
			if err != nil {
				t.Fatalf("cannot sign in with the replacement password: %v", err)
			}
			actor := tu.Must(w.Svc.Authenticate(w.Ctx, token))
			if method != "recovery code" && !actor.MustChangePassword {
				t.Fatal("reset did not require a password change")
			}
			if err := w.Svc.ChangePassword(w.Ctx, actor, replacement, "a final chosen password 84"); err != nil {
				t.Fatalf("successful recovery retained the old password-proof lock: %v", err)
			}
			if _, err := w.Svc.Authenticate(w.Ctx, token); domain.CodeOf(err) != domain.CodeUnauthenticated {
				t.Fatalf("password change did not revoke the replacement session: %v", err)
			}
		})
	}
}
