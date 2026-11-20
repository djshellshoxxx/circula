package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Inert admission reservations exercise every password entry point without
// running a load test. Refusal must precede the first repository operation,
// so it cannot spend a durable guessing allowance or alter credentials.
func TestPasswordAdmissionPrecedesCredentialReadsAndEffects(t *testing.T) {
	limit := tu.Must(passwordwork.New(1))
	e, ops := newOpEnv(t, func(c *service.Config) { c.PasswordWork = limit })
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "admission", nil)
	ana, member := e.Member(admin, net, "ana@example.org")
	codes := tu.Must(e.Svc.GenerateRecoveryCodes(e.Ctx, ana, pw))
	before := dbChecksum(t, e)
	ops.take()
	argon := observeArgon(t)
	observations := &authenticationCounts{}
	e.Svc.SetObserver(observations)
	held := tu.Must(limit.Acquire(e.Ctx))
	defer held.Release()
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"register", func() error {
			_, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: "new@example.org", Password: pw})
			return err
		}},
		{"operator register", func() error {
			_, err := e.Svc.RegisterPersonAsOperator(e.Ctx, domain.SystemActor("test"), service.RegisterInput{Login: "new@example.org", Password: pw})
			return err
		}},
		{"known login", func() error { _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", pw); return err }},
		{"unknown login", func() error { _, err := e.Svc.LoginStep(e.Ctx, "unknown@example.org", pw); return err }},
		{"malformed login", func() error { _, err := e.Svc.LoginStep(e.Ctx, "not a login", pw); return err }},
		{"password change", func() error { return e.Svc.ChangePassword(e.Ctx, ana, pw, newPW) }},
		{"account recovery codes", func() error { _, err := e.Svc.GenerateRecoveryCodes(e.Ctx, ana, pw); return err }},
		{"account recovery", func() error { return e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW) }},
		{"MFA begin", func() error { _, err := e.Svc.BeginMFAEnrolment(e.Ctx, ana, pw); return err }},
		{"MFA confirm", func() error { _, err := e.Svc.ConfirmMFAEnrolment(e.Ctx, ana, pw, "123456"); return err }},
		{"MFA disable", func() error { return e.Svc.DisableMFA(e.Ctx, ana, pw, "123456") }},
		{"administrator reset", func() error {
			_, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, member.ID, "verified in person")
			return err
		}},
		{"operator reset", func() error {
			_, err := e.Svc.OperatorResetPassword(e.Ctx, domain.SystemActor("test"), "ana@example.org", "verified in person")
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, domain.ErrUnavailable) {
				t.Fatalf("busy password admission: %v", err)
			}
			if got := ops.take(); len(got) != 0 {
				t.Fatalf("repository operations before admission: %v", got)
			}
			if got := argon.take(); len(got) != 0 {
				t.Fatalf("hashes before admission: %v", got)
			}
		})
	}
	observations.require(t, nil, nil)
	if after := dbChecksum(t, e); after != before {
		t.Fatal("busy password work changed durable state")
	}
	held.Release()
	// The same code is still usable: overload never consumes recovery material.
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatalf("recovery after capacity returned: %v", err)
	}
}

func TestPasswordAdmissionSharesDefaultAcrossServicesAndStartup(t *testing.T) {
	first := tu.Must(service.New(nil, service.Config{Argon: testArgon}))
	second := tu.Must(service.New(nil, service.Config{Argon: testArgon}))
	var held []*passwordwork.Ticket
	for range passwordwork.DefaultLimit {
		work := tu.Must(passwordwork.Process().Acquire(t.Context()))
		defer work.Release()
		held = append(held, work)
	}
	for _, svc := range []*service.Service{first, second} {
		if _, err := svc.LoginStep(t.Context(), "ana@example.org", pw); !errors.Is(err, domain.ErrUnavailable) {
			t.Fatalf("service did not use the shared process budget: %v", err)
		}
	}
	if svc, err := service.New(nil, service.Config{Argon: testArgon}); svc != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("startup dummy bypassed admission: %v, %v", svc, err)
	}
	held[0].Release()
	if _, err := service.New(nil, service.Config{Argon: testArgon}); err != nil {
		t.Fatalf("startup after capacity returned: %v", err)
	}
}

func TestPasswordAdmissionSpansPasswordProofAndReplacement(t *testing.T) {
	limit := tu.Must(passwordwork.New(1))
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.PasswordWork = limit })
	ana := e.Person("ana@example.org")
	argon := observeArgon(t)
	if err := e.Svc.ChangePassword(e.Ctx, ana, pw, newPW); err != nil {
		t.Fatalf("sequential proof and replacement should use one ticket: %v", err)
	}
	if got := argon.take(); len(got) != 2 || got[0] != testArgon || got[1] != testArgon {
		t.Fatalf("password change work = %v, want unchanged proof and hash parameters", got)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("old password survived: %v", err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", newPW); err != nil {
		t.Fatalf("successful change or failed proof retained admission: %v", err)
	}
	// A rejected proof also releases its operation slot for the next caller.
	if err := e.Svc.ChangePassword(e.Ctx, ana, pw, "another distinct replacement 42"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("old-password refusal: %v", err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", newPW); err != nil {
		t.Fatalf("rejected proof retained admission: %v", err)
	}
}

func TestPasswordAdmissionCanceledBeforeLookup(t *testing.T) {
	limit := tu.Must(passwordwork.New(1))
	e, ops := newOpEnv(t, func(c *service.Config) { c.PasswordWork = limit })
	e.Person("ana@example.org")
	ops.take()
	argon := observeArgon(t)
	ctx, cancel := context.WithCancel(e.Ctx)
	cancel()
	if result, err := e.Svc.LoginStep(ctx, "ana@example.org", pw); result != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled sign-in: %v, %v", result, err)
	}
	if got := ops.take(); len(got) != 0 {
		t.Fatalf("canceled admission touched the store: %v", got)
	}
	if got := argon.take(); len(got) != 0 {
		t.Fatalf("canceled admission hashed: %v", got)
	}
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("canceled admission leaked capacity: %v", err)
	}
}

func TestPasswordAdmissionCanceledReplacementPreservesCredentials(t *testing.T) {
	limit := tu.Must(passwordwork.New(1))
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.PasswordWork = limit })
	ana := e.Person("ana@example.org")
	session, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	readHash := func() string {
		var hash string
		if err := e.Store.DB().QueryRow(`SELECT password_hash FROM persons WHERE id = ?`, ana.PersonID).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		return hash
	}
	beforeHash, beforeAudit := readHash(), countRows(t, e, "audit_events")
	ctx, cancel := context.WithCancel(e.Ctx)
	defer cancel()
	calls := 0
	stop := service.ObserveArgon2(func(service.Argon2Params) {
		calls++
		if calls == 2 { // old proof completed; replacement Argon has not started
			cancel()
		}
	})
	defer stop()
	err = e.Svc.ChangePassword(ctx, ana, pw, newPW)
	stop()
	if !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("canceled replacement: callbacks=%d err=%v", calls, err)
	}
	if readHash() != beforeHash || countRows(t, e, "audit_events") != beforeAudit {
		t.Fatal("cancellation before replacement changed credentials or recorded a password change")
	}
	if _, err := e.Svc.Authenticate(e.Ctx, session); err != nil {
		t.Fatalf("canceled replacement revoked the existing session: %v", err)
	}
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("old password or admission did not survive canceled replacement: %v", err)
	}
}

func TestPasswordAdmissionCancellationRetainsWorkUntilReturn(t *testing.T) {
	limit := tu.Must(passwordwork.New(1))
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.PasswordWork = limit })
	e.Person("ana@example.org")
	started, resume := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(resume) })
	defer unblock()
	stop := service.ObserveArgon2(func(service.Argon2Params) { close(started); <-resume })
	defer stop()
	ctx, cancel := context.WithCancel(e.Ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		result, err := e.Svc.LoginStep(ctx, "ana@example.org", pw)
		if result != nil {
			t.Error("canceled sign-in returned credentials")
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("admitted sign-in did not reach the inert observer")
	}
	cancel()
	if _, err := e.Svc.LoginStep(e.Ctx, "another@example.org", pw); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("in-flight cancellation released capacity early: %v", err)
	}
	unblock()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled work: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled sign-in did not finish")
	}
	stop()
	if countRows(t, e, "sessions") != 0 || countRows(t, e, "mfa_challenges") != 0 {
		t.Fatal("canceled sign-in issued a session or second-factor challenge")
	}
	// An admitted attempt stays reserved, while the capacity refusal above
	// creates no second login key. Cancellation never grants extra guesses.
	var failures int
	if err := e.Store.DB().QueryRow(`SELECT failures FROM login_throttle WHERE login = 'ana@example.org'`).Scan(&failures); err != nil || failures != 1 {
		t.Fatalf("admitted canceled attempt: failures=%d err=%v", failures, err)
	}
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("finished canceled work retained capacity: %v", err)
	}
}
