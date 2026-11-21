package service_test

import (
	"context"
	"errors"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type authenticationCounts struct {
	countObs
	mu                sync.Mutex
	failed, throttled map[string]int
}

func (o *authenticationCounts) AuthenticationFailed(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failed == nil {
		o.failed = make(map[string]int)
	}
	o.failed[reason]++
}

func (o *authenticationCounts) AuthenticationThrottled(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.throttled == nil {
		o.throttled = make(map[string]int)
	}
	o.throttled[reason]++
}

func (o *authenticationCounts) require(t *testing.T, failed, throttled map[string]int) {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	if !maps.Equal(o.failed, failed) || !maps.Equal(o.throttled, throttled) {
		t.Fatalf("credential observations: failed=%v throttled=%v; want failed=%v throttled=%v", o.failed, o.throttled, failed, throttled)
	}
}

// OPS-014: direct service calls are visible, the delegating Login wrapper
// counts once, and unknown/malformed credentials use the same fixed class.
func TestAuthenticationObserverPasswordAndThrottle(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	wrongPassword := e.Key()
	o := &authenticationCounts{}
	e.Svc.SetObserver(o)
	_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", wrongPassword)
	wantCode(t, err, domain.ErrUnauthenticated)
	for _, login := range []string{"unknown@example.org", "not a login"} {
		_, err := e.Svc.LoginStep(e.Ctx, login, wrongPassword)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	o.require(t, map[string]int{"password": 3}, nil)
	// The default allowance is five. The prior known-login call used one;
	// lock refusal remains distinct from a credential that was evaluated.
	for i := 1; i < 5; i++ {
		_, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", wrongPassword)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, err = e.Svc.LoginStep(e.Ctx, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	o.require(t, map[string]int{"password": 7}, map[string]int{"password": 1})
	e.Clock.Advance(15*time.Minute + time.Second)
	tu.Must(e.Svc.LoginStep(e.Ctx, "ana@example.org", pw))
	o.require(t, map[string]int{"password": 7}, map[string]int{"password": 1})
}

// The first password step can succeed with an MFA challenge. Invalid factor
// proofs and typed management-field refusals count; challenge-required does not.
func TestAuthenticationObserverMFAAndManagement(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	wrongPassword, wrongCode := e.Key(), e.Key()
	o := &authenticationCounts{}
	e.Svc.SetObserver(o)
	_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if domain.CodeOf(err) != domain.CodeMFARequired {
		t.Fatalf("password-only wrapper returned %v", err)
	}
	r := tu.Must(e.Svc.LoginStep(e.Ctx, "ana@example.org", pw))
	if r.MFAChallenge == "" {
		t.Fatal("first factor did not return a challenge")
	}
	o.require(t, nil, nil)
	for range 5 {
		_, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, wrongCode)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, _, err = e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, wrongCode)
	wantCode(t, err, domain.ErrRateLimited)
	o.require(t, map[string]int{"mfa": 5}, map[string]int{"mfa": 1})
	e.Clock.Advance(15*time.Minute + time.Second)
	_, err = e.Svc.BeginMFAEnrolment(e.Ctx, a, wrongPassword)
	wantCode(t, err, domain.ErrValidation)
	_, err = e.Svc.ConfirmMFAEnrolment(e.Ctx, a, wrongPassword, wrongCode)
	wantCode(t, err, domain.ErrValidation)
	wantCode(t, e.Svc.DisableMFA(e.Ctx, a, wrongPassword, wrongCode), domain.ErrValidation)
	for range 4 { // one disable plus four regeneration attempts exhaust the management budget
		_, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, wrongCode)
		wantCode(t, err, domain.ErrValidation)
	}
	_, err = e.Svc.RegenerateRecoveryCodes(e.Ctx, a, wrongCode)
	wantCode(t, err, domain.ErrRateLimited)
	o.require(t, map[string]int{"mfa": 5, "mfa_management": 7}, map[string]int{"mfa": 1, "mfa_management": 1})
	// The existing separate sign-in budget still admits a fresh valid factor.
	r = tu.Must(e.Svc.LoginStep(e.Ctx, "ana@example.org", pw))
	tu.Must2(e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, secret, now(e))))
	o.require(t, map[string]int{"mfa": 5, "mfa_management": 7}, map[string]int{"mfa": 1, "mfa_management": 1})
}

func TestAuthenticationObserverRecoveryAndPasswordProof(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	wrongPassword, wrongCode := e.Key(), e.Key()
	o := &authenticationCounts{}
	e.Svc.SetObserver(o)
	// New-password policy validation is not a failed proof of the old password.
	wantCode(t, e.Svc.ChangePassword(e.Ctx, a, pw, "short"), domain.ErrValidation)
	wantCode(t, e.Svc.RecoverPassword(e.Ctx, "ana@example.org", wrongCode, "short"), domain.ErrValidation)
	o.require(t, nil, nil)
	_, err := e.Svc.GenerateRecoveryCodes(e.Ctx, a, wrongPassword)
	wantCode(t, err, domain.ErrUnauthenticated)
	wantCode(t, e.Svc.ChangePassword(e.Ctx, a, wrongPassword, pw), domain.ErrUnauthenticated)
	for range 3 {
		_, err := e.Svc.GenerateRecoveryCodes(e.Ctx, a, wrongPassword)
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	_, err = e.Svc.GenerateRecoveryCodes(e.Ctx, a, pw)
	wantCode(t, err, domain.ErrRateLimited)
	for range 5 {
		wantCode(t, e.Svc.RecoverPassword(e.Ctx, "ana@example.org", wrongCode, pw), domain.ErrUnauthenticated)
	}
	wantCode(t, e.Svc.RecoverPassword(e.Ctx, "ana@example.org", wrongCode, pw), domain.ErrRateLimited)
	o.require(t, map[string]int{"password_proof": 5, "recovery": 5}, map[string]int{"password_proof": 1, "recovery": 1})
	e.Clock.Advance(15*time.Minute + time.Second)
	codes := tu.Must(e.Svc.GenerateRecoveryCodes(e.Ctx, a, pw))
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], pw); err != nil {
		t.Fatal(err)
	}
	o.require(t, map[string]int{"password_proof": 5, "recovery": 5}, map[string]int{"password_proof": 1, "recovery": 1})
}

// Credential refusals are observed independently of adapter status. Store
// failures retain AUTH-02 semantics and never become authentication failures.
func TestAuthenticationObserverSessionTokenAndStoreFailures(t *testing.T) {
	var fs *failingStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		fs = &failingStore{Store: st}
		return fs
	})
	a := e.Person("ana@example.org")
	token, _ := tu.Must2(e.Svc.Login(e.Ctx, "ana@example.org", pw))
	apiToken, apiRecord := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, a, service.TokenInput{Name: "fixture", Scopes: []domain.Scope{domain.ScopeRead}}))
	o := &authenticationCounts{}
	e.Svc.SetObserver(o)
	for _, cause := range []error{errors.New("fixture storage failure"), domain.ErrUnavailable, context.Canceled, context.DeadlineExceeded} {
		fs.err = cause
		_, sessionErr := e.Svc.Authenticate(e.Ctx, token)
		_, tokenErr := e.Svc.AuthenticateToken(e.Ctx, apiToken)
		if !errors.Is(sessionErr, cause) || !errors.Is(tokenErr, cause) {
			t.Fatal("credential lookup changed a store error")
		}
	}
	fs.err = nil
	tu.Must(e.Svc.Authenticate(e.Ctx, token))
	tu.Must(e.Svc.AuthenticateToken(e.Ctx, apiToken))
	o.require(t, nil, nil)
	if err := e.Svc.Logout(e.Ctx, token); err != nil {
		t.Fatal(err)
	}
	if err := e.Svc.RevokeAPIToken(e.Ctx, a, apiRecord.ID); err != nil {
		t.Fatal(err)
	}
	_, err := e.Svc.Authenticate(e.Ctx, token)
	wantCode(t, err, domain.ErrUnauthenticated)
	_, err = e.Svc.AuthenticateToken(e.Ctx, apiToken)
	wantCode(t, err, domain.ErrUnauthenticated)
	o.require(t, map[string]int{"session": 1, "api_token": 1}, nil)
}
