package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const pw = "correct horse battery"

// SEC-001: only a salted Argon2id hash with its parameters is stored.
func TestPasswordStorage(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	var hash string
	if err := e.Store.DB().QueryRow(`SELECT password_hash FROM persons`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == pw || strings.Contains(hash, pw) || !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("stored value %q", hash)
	}
	// parameters travel with the hash: verification uses them, not the current config
	e2 := tu.NewEnv(t)
	e2.Person("ana@example.org")
	var h2 string
	_ = e2.Store.DB().QueryRow(`SELECT password_hash FROM persons`).Scan(&h2)
	if h2 == hash {
		t.Error("salts must differ between hashes")
	}
	// canary: the password appears nowhere in the database file
	for _, table := range []string{"persons", "audit_events", "sessions"} {
		rows, err := e.Store.DB().Query(`SELECT * FROM ` + table)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := rows.Columns()
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			_ = rows.Scan(ptrs...)
			for _, v := range vals {
				if s, ok := v.(string); ok && strings.Contains(s, pw) {
					t.Errorf("plaintext password found in %s", table)
				}
			}
		}
		_ = rows.Close()
	}
}

// SEC-002: 12 minimum, 64+ accepted, no composition rules.
func TestPasswordPolicy(t *testing.T) {
	e := tu.NewEnv(t)
	reg := func(login, p string) error {
		_, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: login, Password: p})
		return err
	}
	if err := reg("a@example.org", strings.Repeat("a", 11)); err == nil {
		t.Error("11 characters accepted")
	}
	if err := reg("b@example.org", strings.Repeat("a", 12)); err != nil {
		t.Errorf("12 lowercase letters must be accepted (no composition rules): %v", err)
	}
	long := strings.Repeat("pässwörd-", 10)[:70]
	if err := reg("c@example.org", long); err != nil {
		t.Errorf("long password: %v", err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "c@example.org", long); err != nil {
		t.Errorf("long password does not verify: %v", err)
	}
	// SA-10: a taken login (case-insensitively) is indistinguishable from a new one...
	if err := reg("B@Example.org ", "another passphrase!!"); err != nil {
		t.Errorf("duplicate login must get the uniform success response: %v", err)
	}
	// ...and must not change the existing account.
	if _, _, err := e.Svc.Login(e.Ctx, "b@example.org", "another passphrase!!"); err == nil {
		t.Error("duplicate registration overwrote the existing password")
	}
	if _, _, err := e.Svc.Login(e.Ctx, "b@example.org", strings.Repeat("a", 12)); err != nil {
		t.Errorf("original password no longer works: %v", err)
	}
	// Operator tooling can still ask for the conflict.
	_, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: "b@example.org", Password: pw, RevealDuplicate: true})
	wantCode(t, err, domain.ErrConflict)
	for _, bad := range []string{"", "ab", "has space@x.org"} {
		if err := reg(bad, pw); err == nil {
			t.Errorf("login %q accepted", bad)
		}
	}
}

func TestLoginSessionLifecycle(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	tok, p, err := e.Svc.Login(e.Ctx, " ANA@example.org ", pw)
	if err != nil || p.Login != "ana@example.org" || len(tok) < 40 {
		t.Fatalf("login: %v %v %d", p, err, len(tok))
	}
	actor, err := e.Svc.Authenticate(e.Ctx, tok)
	if err != nil || actor.PersonID != p.ID || !actor.Can(domain.ScopeWrite) {
		t.Fatalf("auth: %+v %v", actor, err)
	}
	// distinct tokens per login; only the hash is stored
	tok2, _, _ := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if tok == tok2 {
		t.Error("session ids must be regenerated at login")
	}
	var id string
	_ = e.Store.DB().QueryRow(`SELECT id FROM sessions LIMIT 1`).Scan(&id)
	if id == tok || id == tok2 {
		t.Error("raw token stored")
	}
	// idle timeout (30 days by default) and activity extension
	e.Clock.Advance(29 * 24 * time.Hour)
	if _, err := e.Svc.Authenticate(e.Ctx, tok); err != nil {
		t.Errorf("session expired early: %v", err)
	}
	e.Clock.Advance(29 * 24 * time.Hour) // active 29 days ago -> still within idle window
	if _, err := e.Svc.Authenticate(e.Ctx, tok); err != nil {
		t.Errorf("activity should extend idle window: %v", err)
	}
	e.Clock.Advance(31 * 24 * time.Hour)
	if _, err := e.Svc.Authenticate(e.Ctx, tok); err == nil {
		t.Error("idle session accepted")
	}
	// absolute lifetime
	tok3, _, _ := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	for i := 0; i < 4; i++ {
		e.Clock.Advance(25 * 24 * time.Hour)
		if _, err := e.Svc.Authenticate(e.Ctx, tok3); err != nil && i < 3 {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if _, err := e.Svc.Authenticate(e.Ctx, tok3); err == nil {
		t.Error("session beyond the absolute lifetime accepted")
	}
	// logout invalidates
	tok4, _, _ := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	tu.Must(0, e.Svc.Logout(e.Ctx, tok4))
	if _, err := e.Svc.Authenticate(e.Ctx, tok4); err == nil {
		t.Error("session valid after logout")
	}
	if _, err := e.Svc.Authenticate(e.Ctx, "garbage"); err == nil {
		t.Error("garbage token accepted")
	}
}

// SEC-005: identical errors for unknown login and wrong password.
func TestLoginDoesNotRevealExistence(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	_, _, e1 := e.Svc.Login(e.Ctx, "ana@example.org", "wrong password!!")
	_, _, e2 := e.Svc.Login(e.Ctx, "nobody@example.org", "wrong password!!")
	if e1 == nil || e2 == nil || e1.Error() != e2.Error() || domain.CodeOf(e1) != domain.CodeUnauthenticated {
		t.Errorf("responses differ: %v / %v", e1, e2)
	}
}

// SEC-006: five failures, then throttled with growing delays; success resets.
func TestLoginThrottle(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	for i := 0; i < 5; i++ {
		_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", "wrong password!!")
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	// even the right password is refused during the lockout
	_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	e.Clock.Advance(14 * time.Minute)
	_, _, err = e.Svc.Login(e.Ctx, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	e.Clock.Advance(2 * time.Minute)
	if _, _, err = e.Svc.Login(e.Ctx, "ana@example.org", "wrong password!!"); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("after the delay a (wrong) attempt is evaluated again: %v", err)
	}
	// the sixth failure doubles the delay
	e.Clock.Advance(16 * time.Minute)
	_, _, err = e.Svc.Login(e.Ctx, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
	e.Clock.Advance(15 * time.Minute)
	if _, _, err = e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("success after delay: %v", err)
	}
	// success reset the counter: a new wrong attempt is not throttled
	_, _, err = e.Svc.Login(e.Ctx, "ana@example.org", "wrong password!!")
	wantCode(t, err, domain.ErrUnauthenticated)
	_, _, err = e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Errorf("counter not reset: %v", err)
	}
}

// SEC-003: password change revokes every session.
func TestChangePasswordRevokesSessions(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	tokA, _, _ := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	tokB, _, _ := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	actor := tu.Must(e.Svc.Authenticate(e.Ctx, tokA))
	if err := e.Svc.ChangePassword(e.Ctx, actor, "wrong old password", "a brand new passphrase"); err == nil {
		t.Error("wrong old password accepted")
	}
	if err := e.Svc.ChangePassword(e.Ctx, actor, pw, "short"); err == nil {
		t.Error("weak new password accepted")
	}
	tu.Must(0, e.Svc.ChangePassword(e.Ctx, actor, pw, "a brand new passphrase"))
	for _, tok := range []string{tokA, tokB} {
		if _, err := e.Svc.Authenticate(e.Ctx, tok); err == nil {
			t.Error("session survived a password change")
		}
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err == nil {
		t.Error("old password still works")
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", "a brand new passphrase"); err != nil {
		t.Error(err)
	}
}

func TestAPITokens(t *testing.T) {
	w := newWorld(t, nil)
	ro, tokRO := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "ro", Scopes: []domain.Scope{domain.ScopeRead}}))
	if !strings.HasPrefix(ro, "cir_") || tokRO.TokenHash == ro {
		t.Fatalf("token %q hash %q", ro, tokRO.TokenHash)
	}
	roActor := tu.Must(w.Svc.AuthenticateToken(w.Ctx, ro))
	tu.Must(w.Svc.GetBalance(w.Ctx, roActor, w.net.ID))
	_, err := w.propose(roActor, w.bM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrForbidden) // read-only token cannot write
	// a read-only token cannot mint a write token (no escalation), nor widen network scope
	_, _, err = w.Svc.CreateAPIToken(w.Ctx, roActor, service.TokenInput{Name: "x", Scopes: []domain.Scope{domain.ScopeRead}})
	wantCode(t, err, domain.ErrForbidden)
	_, _, err = w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "x", Scopes: []domain.Scope{"admin"}})
	wantCode(t, err, domain.ErrValidation)
	_, _, err = w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "x", Scopes: []domain.Scope{domain.ScopeRead}, NetworkID: "not-mine"})
	wantCode(t, err, domain.ErrNotFound)

	// expiry and revocation
	exp := w.Clock.Now().Add(time.Hour)
	short, st := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "short", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}, NetworkID: w.net.ID, ExpiresAt: &exp}))
	act := tu.Must(w.Svc.AuthenticateToken(w.Ctx, short))
	if act.NetworkScope != w.net.ID {
		t.Error("network scope lost")
	}
	tu.Must(w.propose(act, w.bM, service.ProposerPays, 1))
	w.Clock.Advance(2 * time.Hour)
	if _, err := w.Svc.AuthenticateToken(w.Ctx, short); err == nil {
		t.Error("expired token accepted")
	}
	tu.Must(0, w.Svc.RevokeAPIToken(w.Ctx, w.a, tokRO.ID))
	if _, err := w.Svc.AuthenticateToken(w.Ctx, ro); err == nil {
		t.Error("revoked token accepted")
	}
	if err := w.Svc.RevokeAPIToken(w.Ctx, w.b, st.ID); err == nil {
		t.Error("another person revoked my token")
	}
	if l := tu.Must(w.Svc.ListAPITokens(w.Ctx, w.a)); len(l) != 2 {
		t.Errorf("tokens listed = %d", len(l))
	}
	if _, err := w.Svc.AuthenticateToken(w.Ctx, "cir_unknown"); err == nil {
		t.Error("unknown token accepted")
	}
}
