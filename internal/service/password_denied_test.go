package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Breached-password check: embedded list, case-insensitive, look-alike and
// suffix variants, and passwords derived from the login.
func TestPasswordDeniedCommonAndDerived(t *testing.T) {
	e := tu.NewEnv(t)
	reg := func(login, p string) error {
		_, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: login, Password: p})
		return err
	}
	for _, p := range []string{
		"password1234", "PassWord123456", "qwertyuiop12", "1q2w3e4r5t6y", "iloveyou2025!", "P@ssw0rd1234!",
		"letmein123456", "Welcome2025!!!", "football123456", "123456789012",
	} {
		wantCode(t, reg("deny@example.org", p), domain.ErrValidation)
	}
	// derived from the login, with case and look-alike changes
	for _, p := range []string{"zuzana-novak-2025", "ZUZANA.novak", "Zuz4na_Novak9"} {
		wantCode(t, reg("zuzana.novak@example.org", p), domain.ErrValidation)
	}
	// acceptable passphrases still pass, also with a short login
	for i, p := range []string{"correct horse battery", "a brand new passphrase", "tiny-ana-needs-a-long-one"} {
		if err := reg("ok"+string(rune('a'+i))+"@example.org", p); err != nil {
			t.Errorf("%q refused: %v", p, err)
		}
	}
}

// The check also guards a password change and recovery.
func TestPasswordDeniedOnChangeAndRecover(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("anastasia@example.org")
	wantCode(t, e.Svc.ChangePassword(e.Ctx, ana, pw, "Password123456"), domain.ErrValidation)
	wantCode(t, e.Svc.ChangePassword(e.Ctx, ana, pw, "my-anastasia-secret"), domain.ErrValidation)
	if err := e.Svc.ChangePassword(e.Ctx, ana, pw, "a different passphrase"); err != nil {
		t.Fatal(err)
	}
	codes := tu.Must(e.Svc.GenerateRecoveryCodes(e.Ctx, ana, "a different passphrase"))
	wantCode(t, e.Svc.RecoverPassword(e.Ctx, "anastasia@example.org", codes[0], "qwertyuiop123"), domain.ErrValidation)
	if err := e.Svc.RecoverPassword(e.Ctx, "anastasia@example.org", codes[0], newPW); err != nil {
		t.Fatalf("code consumed by a refused password: %v", err)
	}
}
