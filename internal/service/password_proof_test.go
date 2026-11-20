package service_test

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
	store "github.com/djshellshoxxx/circula/internal/repository"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A current-password proof must still name the stored hash when its protected
// write starts. ObserveArgon2 commits a real password change/reset after that
// hash has been read, while the old password is being verified outside a write.
func TestPasswordProtectedChangesRejectSupersededPassword(t *testing.T) {
	for _, change := range []struct {
		name string
		run  func(*world) (string, error)
	}{
		{"self-service change", func(w *world) (string, error) {
			const replacement = "a newly chosen password 99"
			return replacement, w.Svc.ChangePassword(w.Ctx, w.a, pw, replacement)
		}},
		{"administrator reset", func(w *world) (string, error) {
			return w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, w.aM.ID, "account compromised")
		}},
	} {
		for _, operation := range []string{"begin", "resume", "confirm", "disable TOTP", "disable recovery", "account recovery codes"} {
			t.Run(change.name+"/"+operation, func(t *testing.T) {
				// Both the observed proof and the legitimate password change
				// use ordinary tickets in this isolated two-operation scope.
				w := newWorldCfg(t, nil, func(c *service.Config) { c.PasswordWork = tu.Must(passwordwork.New(2)) })
				// The actor represents a request that authenticated before the
				// password change revoked its session.
				token, _, err := w.Svc.Login(w.Ctx, "ana@example.org", pw)
				if err != nil {
					t.Fatal(err)
				}
				w.a = tu.Must(w.Svc.Authenticate(w.Ctx, token))
				var run func(string) (returnedSecret bool, err error)
				switch operation {
				case "begin", "resume":
					if operation == "resume" {
						tu.Must(w.Svc.BeginMFAEnrolment(w.Ctx, w.a, pw))
					}
					run = func(password string) (bool, error) {
						en, err := w.Svc.BeginMFAEnrolment(w.Ctx, w.a, password)
						return en != nil, err
					}
				case "confirm":
					en := tu.Must(w.Svc.BeginMFAEnrolment(w.Ctx, w.a, pw))
					code := totpAt(t, en.Secret, now(w.Env))
					run = func(password string) (bool, error) {
						codes, err := w.Svc.ConfirmMFAEnrolment(w.Ctx, w.a, password, code)
						return len(codes) != 0, err
					}
				case "disable TOTP", "disable recovery":
					secret, codes := enrol(t, w.Env, w.a)
					code := totpAt(t, secret, now(w.Env))
					if operation == "disable recovery" {
						code = codes[0]
					}
					run = func(password string) (bool, error) {
						return false, w.Svc.DisableMFA(w.Ctx, w.a, password, code)
					}
				case "account recovery codes":
					tu.Must(w.Svc.GenerateRecoveryCodes(w.Ctx, w.a, pw))
					run = func(password string) (bool, error) {
						codes, err := w.Svc.GenerateRecoveryCodes(w.Ctx, w.a, password)
						return len(codes) != 0, err
					}
				}

				var fired atomic.Bool
				var replacement string
				var changeErr error
				var want passwordProtectedState
				stop := service.ObserveArgon2(func(service.Argon2Params) {
					if fired.CompareAndSwap(false, true) {
						replacement, changeErr = change.run(w)
						// Snapshot after the legitimate change, so its intended
						// revocations, notices and audit events are preserved.
						want = readPasswordProtectedState(t, w)
					}
				})
				defer stop()
				returnedSecret, err := run(pw)
				stop()
				if !fired.Load() || changeErr != nil {
					t.Fatalf("password change did not commit during verification: fired=%v err=%v", fired.Load(), changeErr)
				}
				if _, err := w.Svc.Authenticate(w.Ctx, token); err == nil {
					t.Fatal("password change did not revoke the original session")
				}
				wantCode := domain.CodeValidation
				if operation == "account recovery codes" {
					wantCode = domain.CodeUnauthenticated
				}
				if domain.CodeOf(err) != wantCode {
					t.Errorf("superseded password: got %v, want %s", err, wantCode)
				}
				if wantCode == domain.CodeValidation && err != nil {
					var de *domain.Error
					if !errors.As(err, &de) || len(de.Fields) != 1 || de.Fields[0].Field != "password" {
						t.Errorf("MFA refusal must retain the password field error: %v", err)
					}
				}
				if returnedSecret {
					t.Error("superseded password returned an enrolment secret or recovery codes")
				}
				if got := readPasswordProtectedState(t, w); !reflect.DeepEqual(got, want) {
					t.Error("superseded password changed the factor, recovery codes, password, audit events or security notices")
				}
				if t.Failed() {
					return
				}
				// A fresh password proof can use the very same TOTP/recovery
				// code: rejection must not burn the factor. A reset separately
				// requires a password change before normal adapter operations.
				if change.name == "self-service change" {
					if _, err := run(replacement); err != nil {
						t.Fatalf("fresh password with the same factor: %v", err)
					}
				}
			})
		}
	}
}

type passwordProtectedState struct {
	hash          string
	mfa           *domain.MFARecord
	mfaCodes      []string
	accountCodes  []string
	auditEvents   int
	securityNotes int
}

func readPasswordProtectedState(t *testing.T, w *world) passwordProtectedState {
	t.Helper()
	var st passwordProtectedState
	if err := w.Store.View(w.Ctx, func(tx store.Tx) error {
		_, hash, err := tx.GetPerson(w.a.PersonID)
		if err != nil {
			return err
		}
		st.hash = hash
		st.mfa, err = tx.GetMFA(w.a.PersonID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []struct {
		name string
		dst  *[]string
	}{{"recovery_codes", &st.mfaCodes}, {"account_recovery_codes", &st.accountCodes}} {
		rows, err := w.Store.DB().Query(`SELECT code_hash || ':' || COALESCE(used_at, '') FROM `+table.name+` WHERE person_id = ? ORDER BY code_hash`, w.a.PersonID)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				_ = rows.Close()
				t.Fatal(err)
			}
			*table.dst = append(*table.dst, code)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	st.auditEvents = count(t, w.Env, `SELECT COUNT(*) FROM audit_events WHERE target_id = ?`, w.a.PersonID)
	st.securityNotes = count(t, w.Env, `SELECT COUNT(*) FROM security_notices WHERE person_id = ?`, w.a.PersonID)
	return st
}
