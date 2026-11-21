package service_test

import (
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-003 / AUD4-F06: a sign-in with the old password that is still hashing
// when the password is changed, recovered or reset must not obtain a session.
// Argon2 runs outside the write units, so the change can commit in between;
// ObserveArgon2 places it there deterministically.
func TestLoginStraddlingPasswordChangeGetsNoSession(t *testing.T) {
	for name, change := range map[string]func(w *world) error{
		"self-service change": func(w *world) error {
			return w.Svc.ChangePassword(w.Ctx, w.a, pw, "a brand new password 99")
		},
		"administrator reset": func(w *world) error {
			_, err := w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, w.aM.ID, "account compromised")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			// The observer admits a second real operation during the first.
			w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 },
				func(c *service.Config) { c.PasswordWork = tu.Must(passwordwork.New(2)) })
			var fired atomic.Bool
			var changeErr error
			stop := service.ObserveArgon2(func(service.Argon2Params) {
				if fired.CompareAndSwap(false, true) { // during the old-password sign-in's hash check
					changeErr = change(w)
				}
			})
			tok, _, err := w.Svc.Login(w.Ctx, "ana@example.org", pw)
			stop()
			if changeErr != nil {
				t.Fatal(changeErr)
			}
			if err == nil {
				if a, aerr := w.Svc.Authenticate(w.Ctx, tok); aerr == nil {
					t.Fatalf("a session issued for the superseded password survives (must_change=%v)", a.MustChangePassword)
				}
				t.Fatal("login reported success for the superseded password")
			}
			if domain.CodeOf(err) != domain.CodeUnauthenticated {
				t.Fatalf("err = %v, want unauthenticated", err)
			}
		})
	}
}

// AUD4-F06: two password changes that both verified the same old password
// must not both apply; the later one would silently replace the first.
func TestChangePasswordReplacesOnlyTheVerifiedHash(t *testing.T) {
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 },
		func(c *service.Config) { c.PasswordWork = tu.Must(passwordwork.New(2)) })
	var fired atomic.Bool
	var first error
	stop := service.ObserveArgon2(func(service.Argon2Params) {
		if fired.CompareAndSwap(false, true) { // while the second change verifies the old password
			first = w.Svc.ChangePassword(w.Ctx, w.a, pw, "first new password 111")
		}
	})
	second := w.Svc.ChangePassword(w.Ctx, w.a, pw, "second new password 222")
	stop()
	if first != nil {
		t.Fatal(first)
	}
	if second == nil {
		t.Fatal("a change verified against a superseded password was applied")
	}
	if _, _, err := w.Svc.Login(w.Ctx, "ana@example.org", "first new password 111"); err != nil {
		t.Fatalf("the first change was lost: %v", err)
	}
}

// AUD4-F04: guesses of the current password through a stolen session are
// throttled durably, like the other re-authentication checks (A3-03).
func TestChangePasswordGuessesAreThrottled(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	limited := -1
	for i := 0; i < 30; i++ {
		err := w.Svc.ChangePassword(w.Ctx, w.a, "a wrong guess number one", "another long password 1")
		if domain.CodeOf(err) == domain.CodeRateLimited {
			limited = i
			break
		}
		if domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("guess %d: %v", i, err)
		}
	}
	if limited < 0 {
		t.Fatal("30 current-password guesses were all evaluated")
	}
	if err := w.Svc.ChangePassword(w.Ctx, w.a, pw, "another long password 1"); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Fatalf("the correct password was accepted while locked: %v", err)
	}
}
