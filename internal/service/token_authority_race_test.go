package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Authentication and mutation run in separate transactions at the HTTP edge.
// End/reset the session after authentication, then resume its pending token
// command deterministically. No scheduling delay or probabilistic race is used.
func TestTokenMutationsRejectSessionEndedAfterAuthentication(t *testing.T) {
	for _, end := range []string{"logout", "password-change", "administrator-reset", "operator-reset", "forced-change"} {
		t.Run(end, func(t *testing.T) {
			w := newWorld(t, nil)
			raw, actor := selfSessionLogin(t, w.Env, "ana@example.org")
			tokenValue, token := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, actor, service.TokenInput{
				Name: "existing integration", Scopes: []domain.Scope{domain.ScopeRead}}))
			if _, err := w.Svc.AuthenticateToken(w.Ctx, tokenValue); err != nil {
				t.Fatalf("unchanged authorized session issued an unusable token: %v", err)
			}
			want := domain.CodeUnauthenticated
			var err error
			switch end {
			case "logout":
				err = w.Svc.Logout(w.Ctx, raw)
			case "password-change":
				err = w.Svc.ChangePassword(w.Ctx, actor, pw, "a new long password 54321")
			case "administrator-reset":
				_, err = w.Svc.AdminResetPassword(w.Ctx, w.admin, w.net.ID, w.aM.ID, "account recovery")
			case "operator-reset":
				_, err = w.Svc.OperatorResetPassword(w.Ctx, domain.SystemActor("token-race-test"), "ana@example.org", "account recovery")
			case "forced-change":
				err = w.Store.Update(w.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(actor.PersonID, true) })
				want = domain.CodePasswordChangeRequired
			}
			if err != nil {
				t.Fatalf("intervening credential action: %v", err)
			}
			before := checksumExcept(t, w.Env)
			value, issued, err := w.Svc.CreateAPIToken(w.Ctx, actor, service.TokenInput{
				Name: "pending request", Scopes: []domain.Scope{domain.ScopeRead}})
			if domain.CodeOf(err) != want || value != "" || issued != nil {
				t.Errorf("stale session issued a token: value-present=%v token-present=%v err=%v", value != "", issued != nil, err)
			}
			if err := w.Svc.RevokeAPIToken(w.Ctx, actor, token.ID); domain.CodeOf(err) != want {
				t.Errorf("stale session revoked a token: %v", err)
			}
			if after := checksumExcept(t, w.Env); after != before {
				t.Fatal("refused token command changed tokens, notices or audit")
			}
		})
	}
}

func TestDirectTokenMutationsReadCurrentPasswordRestriction(t *testing.T) {
	w := newWorld(t, nil)
	// Trusted non-session service callers retain their existing behavior, but
	// their copied flag must not override the person's current stored restriction.
	_, token := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{
		Name: "existing integration", Scopes: []domain.Scope{domain.ScopeRead}}))
	if err := w.Store.Update(w.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(w.a.PersonID, true) }); err != nil {
		t.Fatal(err)
	}
	before := checksumExcept(t, w.Env)
	_, _, err := w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{
		Name: "pending integration", Scopes: []domain.Scope{domain.ScopeRead}})
	if domain.CodeOf(err) != domain.CodePasswordChangeRequired {
		t.Errorf("direct caller bypassed stored password restriction: %v", err)
	}
	if err := w.Svc.RevokeAPIToken(w.Ctx, w.a, token.ID); domain.CodeOf(err) != domain.CodePasswordChangeRequired {
		t.Errorf("direct revocation bypassed stored password restriction: %v", err)
	}
	if after := checksumExcept(t, w.Env); after != before {
		t.Fatal("restricted token mutations changed state")
	}
	if err := w.Store.Update(w.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(w.a.PersonID, false) }); err != nil {
		t.Fatal(err)
	}
	if err := w.Svc.RevokeAPIToken(w.Ctx, w.a, token.ID); err != nil {
		t.Fatalf("cleared restriction did not restore legitimate direct caller: %v", err)
	}
}
