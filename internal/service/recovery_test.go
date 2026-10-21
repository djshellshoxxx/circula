package service_test

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const newPW = "a brand new passphrase"

func codesFor(t *testing.T, e *tu.Env, a domain.Actor) []string {
	t.Helper()
	return tu.Must(e.Svc.GenerateRecoveryCodes(e.Ctx, a, pw))
}

func auditCount(t *testing.T, e *tu.Env, action string) int {
	t.Helper()
	var n int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// SEC-008: codes are generated with the password, shown once, hashed at rest.
func TestRecoveryCodesGenerate(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	if _, err := e.Svc.GenerateRecoveryCodes(e.Ctx, ana, "wrong password!!"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("wrong password must be refused, got %v", err)
	}
	codes := codesFor(t, e, ana)
	if len(codes) != service.RecoveryCodeCount {
		t.Fatalf("got %d codes", len(codes))
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if len(c) != 19 || seen[c] {
			t.Fatalf("bad or duplicate code %q", c)
		}
		seen[c] = true
		var n int
		_ = e.Store.DB().QueryRow(`SELECT COUNT(*) FROM account_recovery_codes WHERE code_hash LIKE '%' || ? || '%'`,
			strings.ReplaceAll(c, "-", "")).Scan(&n)
		if n != 0 {
			t.Fatal("plaintext code stored")
		}
	}
	if n := tu.Must(e.Svc.RecoveryCodesRemaining(e.Ctx, ana)); n != service.RecoveryCodeCount {
		t.Fatalf("remaining %d", n)
	}
	// audit never carries a code
	rows, _ := e.Store.DB().Query(`SELECT summary_after, summary_before, reason FROM audit_events`)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var a, b, r string
		_ = rows.Scan(&a, &b, &r)
		for _, c := range codes {
			if strings.Contains(a+b+r, c) {
				t.Fatal("code in audit")
			}
		}
	}
	// token scoped to one network cannot mint codes
	scoped := ana
	scoped.NetworkScope = "x"
	if _, err := e.Svc.GenerateRecoveryCodes(e.Ctx, scoped, pw); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("scoped: %v", err)
	}
	// regenerate invalidates the old batch
	fresh := codesFor(t, e, ana)
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("old code must fail, got %v", err)
	}
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", fresh[0], newPW); err != nil {
		t.Fatalf("new code: %v", err)
	}
}

// Replay: a used code never works again; lower case and missing dashes are accepted.
func TestRecoverReplay(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	codes := codesFor(t, e, ana)
	sloppy := strings.ToLower(strings.ReplaceAll(codes[3], "-", " "))
	if err := e.Svc.RecoverPassword(e.Ctx, " Ana@Example.org ", sloppy, newPW); err != nil {
		t.Fatal(err)
	}
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[3], "yet another passphrase"); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("replay must fail, got %v", err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", newPW); err != nil {
		t.Fatalf("login with the recovered password: %v", err)
	}
	if auditCount(t, e, domain.ActPersonRecovered) != 1 {
		t.Fatal("expected one person.recovered event")
	}
	// remaining codes were wiped on recovery
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[4], "yet another passphrase"); err == nil {
		t.Fatal("other codes must be invalid after a recovery")
	}
}

// Wrong codes are throttled per login; the lock also refuses the right code.
func TestRecoverThrottle(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	codes := codesFor(t, e, ana)
	for i := 0; i < 5; i++ {
		if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", "AAAA-AAAA-AAAA-AAAA", newPW); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("locked login must be rate limited even for a right code, got %v", err)
	}
	// the normal login of ana is not locked by recovery guesses
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatalf("login must be unaffected: %v", err)
	}
	e.Clock.Advance(16 * time.Minute)
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatalf("after the lock: %v", err)
	}
}

// Unknown and known logins answer identically, including throttling.
func TestRecoverUnknownLoginIndistinguishable(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	_ = codesFor(t, e, ana)
	check := func(login string) []string {
		var out []string
		for i := 0; i < 7; i++ {
			err := e.Svc.RecoverPassword(e.Ctx, login, "ABCD-EFGH-IJKL-MNOP", newPW)
			out = append(out, string(domain.CodeOf(err))+"|"+err.Error())
		}
		return out
	}
	known, unknown := check("ana@example.org"), check("nobody@example.org")
	for i := range known {
		if known[i] != unknown[i] {
			t.Fatalf("attempt %d differs: %q vs %q", i, known[i], unknown[i])
		}
	}
	if !strings.HasPrefix(unknown[6], "rate-limited") {
		t.Fatalf("unknown login must lock too: %q", unknown[6])
	}
	// malformed login and malformed code give the same refusal
	for _, c := range []string{"short", "ÄÖÜ!-0000-0000-0000", ""} {
		if err := e.Svc.RecoverPassword(e.Ctx, "someone@example.org", c, newPW); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("code %q: %v", c, err)
		}
	}
	if err := e.Svc.RecoverPassword(e.Ctx, "a b", "ABCD-EFGH-IJKL-MNOP", newPW); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("bad login: %v", err)
	}
}

// Recovery revokes every session and API token and signs nobody in.
func TestRecoverRevokesEverything(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	tok, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	api, _ := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, ana, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead}}))
	codes := codesFor(t, e, ana)
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.Authenticate(e.Ctx, tok); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("session must be revoked, got %v", err)
	}
	if _, err := e.Svc.AuthenticateToken(e.Ctx, api); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("api token must be revoked, got %v", err)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("old password must be dead")
	}
	var sessions int
	_ = e.Store.DB().QueryRow(`SELECT COUNT(*) FROM sessions WHERE revoked_at IS NULL`).Scan(&sessions)
	if sessions != 0 {
		t.Fatalf("%d live sessions after recovery", sessions)
	}
	// weak new password is refused before the code is touched
	c2 := codesFor2(t, e)
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", c2[0], "short"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("weak password: %v", err)
	}
	if n := tu.Must(e.Svc.RecoveryCodesRemaining(e.Ctx, ana)); n != service.RecoveryCodeCount {
		t.Fatalf("a refused password must not burn the code, remaining %d", n)
	}
}

func codesFor2(t *testing.T, e *tu.Env) []string {
	t.Helper()
	tok, _, err := e.Svc.Login(e.Ctx, "ana@example.org", newPW)
	if err != nil {
		t.Fatal(err)
	}
	a := tu.Must(e.Svc.Authenticate(e.Ctx, tok))
	return tu.Must(e.Svc.GenerateRecoveryCodes(e.Ctx, a, newPW))
}

// Race: the same code used in parallel (even for different new passwords) has exactly one winner.
func TestRecoverCodeRace(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	codes := codesFor(t, e, ana)
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err == nil {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d winners, want exactly 1", wins.Load())
	}
	if auditCount(t, e, domain.ActPersonRecovered) != 1 {
		t.Fatal("exactly one audit event expected")
	}
}

// Forced change: Authenticate flags the actor; every write refuses on service
// and API-token paths until ChangePassword; reads still work.
func TestForcedPasswordChange(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "reset-net", nil)
	bob, bobM := e.Member(admin, net, "bob@example.org")
	oldAPI, _ := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, bob, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}))
	tok, _, _ := e.Svc.Login(e.Ctx, "bob@example.org", pw)

	temp, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, bobM.ID, "lost password, verified in person")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.Authenticate(e.Ctx, tok); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("sessions must be revoked")
	}
	if _, err := e.Svc.AuthenticateToken(e.Ctx, oldAPI); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("tokens must be revoked")
	}
	if _, _, err := e.Svc.Login(e.Ctx, "bob@example.org", pw); err == nil {
		t.Fatal("old password must be dead")
	}
	tok, _, err = e.Svc.Login(e.Ctx, "bob@example.org", temp)
	if err != nil {
		t.Fatalf("temp password login: %v", err)
	}
	a, err := e.Svc.Authenticate(e.Ctx, tok)
	if err != nil || !a.MustChangePassword {
		t.Fatalf("actor must be flagged: %+v %v", a, err)
	}
	mustBlock := func(what string, err error) {
		t.Helper()
		if domain.CodeOf(err) != domain.CodePasswordChangeRequired {
			t.Errorf("%s: want password-change-required, got %v", what, err)
		}
	}
	_, _, err = e.Svc.CreateAPIToken(e.Ctx, a, service.TokenInput{Name: "x", Scopes: []domain.Scope{domain.ScopeRead}})
	mustBlock("CreateAPIToken", err)
	_, err = e.Svc.ProposeTransaction(e.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: bobM.ID,
		Direction: service.ProposerPays, Amount: 1, IdempotencyKey: e.Key()})
	mustBlock("ProposeTransaction", err)
	_, _, err = e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "N", Slug: "n-n"})
	mustBlock("CreateNetwork", err)
	_, err = e.Svc.GenerateRecoveryCodes(e.Ctx, a, temp)
	mustBlock("GenerateRecoveryCodes", err)
	if _, err := e.Svc.MyMemberships(e.Ctx, a); err != nil {
		t.Errorf("reads stay allowed: %v", err)
	}
	// a read-only actor cannot use the flag as a back door to change the password
	ro := a
	ro.Scopes = []domain.Scope{domain.ScopeRead}
	if err := e.Svc.ChangePassword(e.Ctx, ro, temp, newPW); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("read-only: %v", err)
	}
	if err := e.Svc.ChangePassword(e.Ctx, a, temp, newPW); err != nil {
		t.Fatalf("ChangePassword must work while flagged: %v", err)
	}
	tok, _, err = e.Svc.Login(e.Ctx, "bob@example.org", newPW)
	if err != nil {
		t.Fatal(err)
	}
	if a, _ = e.Svc.Authenticate(e.Ctx, tok); a.MustChangePassword {
		t.Fatal("flag must clear after ChangePassword")
	}
	if _, _, err := e.Svc.CreateAPIToken(e.Ctx, a, service.TokenInput{Name: "x", Scopes: []domain.Scope{domain.ScopeRead}}); err != nil {
		t.Fatalf("writes work again: %v", err)
	}
}

// Admin reset guards: self, non-admin, cross-network, departed, reason, audit, high risk, rate limit.
func TestAdminResetGuards(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, adminM := e.Network(admin, "net-one", nil)
	bob, bobM := e.Member(admin, net, "bob@example.org")
	other := e.Person("other@example.org")
	net2, _ := e.Network(other, "net-two", nil)
	_, carolM := e.Member(other, net2, "carol@example.org")

	ctx := e.Ctx
	if _, err := e.Svc.AdminResetPassword(ctx, admin, net.ID, adminM.ID, "forgot my own"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("self reset: %v", err)
	}
	if _, err := e.Svc.AdminResetPassword(ctx, bob, net.ID, adminM.ID, "i am not admin"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("non-admin: %v", err)
	}
	if _, err := e.Svc.AdminResetPassword(ctx, admin, net.ID, bobM.ID, "  "); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("reason required: %v", err)
	}
	// cross-network: admin of net-one vs carol of net-two, in both id spellings
	for _, nid := range []string{net.ID, net2.ID} {
		if _, err := e.Svc.AdminResetPassword(ctx, admin, nid, carolM.ID, "not my network"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("cross-network via %s: %v", nid, err)
		}
	}
	if _, _, err := e.Svc.Login(ctx, "carol@example.org", pw); err != nil {
		t.Errorf("carol must be untouched: %v", err)
	}
	// second admin: allowed, flagged high risk, never self through the same person
	if err := e.Svc.AssignRole(ctx, admin, net.ID, bobM.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.AdminResetPassword(ctx, bob, net.ID, bobM.ID, "self again"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("self as admin: %v", err)
	}
	temp, err := e.Svc.AdminResetPassword(ctx, admin, net.ID, bobM.ID, "bob lost his phone and codes")
	if err != nil {
		t.Fatal(err)
	}
	var after, reason, actor string
	if err := e.Store.DB().QueryRow(`SELECT summary_after, reason, actor_id FROM audit_events WHERE action = ?`,
		domain.ActMemberPasswordReset).Scan(&after, &reason, &actor); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after, `"high_risk":true`) || reason != "bob lost his phone and codes" || actor != adminM.ID {
		t.Errorf("audit: after=%s reason=%q actor=%q", after, reason, actor)
	}
	if strings.Contains(after+reason, temp) {
		t.Error("temp password in audit")
	}
	// plain member: high_risk false
	dan, danM := e.Member(admin, net, "dan@example.org")
	_ = dan
	if _, err := e.Svc.AdminResetPassword(ctx, admin, net.ID, danM.ID, "routine"); err != nil {
		t.Fatal(err)
	}
	var risky int
	_ = e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ? AND summary_after LIKE '%"high_risk":false%'`,
		domain.ActMemberPasswordReset).Scan(&risky)
	if risky != 1 {
		t.Errorf("expected one low-risk event, got %d", risky)
	}
	// departed members cannot be reset
	if _, err := e.Svc.DepartMember(ctx, admin, net.ID, service.DepartInput{MemberID: danM.ID, Reason: "left"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.AdminResetPassword(ctx, admin, net.ID, danM.ID, "after leaving"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("departed: %v", err)
	}
}

func TestAdminResetRateLimit(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "net-rl", nil)
	var ids []string
	for i := 0; i < 11; i++ {
		_, m := e.Member(admin, net, "m"+string(rune('a'+i))+"@example.org")
		ids = append(ids, m.ID)
	}
	for i := 0; i < 10; i++ {
		if _, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, ids[i], "routine reset"); err != nil {
			t.Fatalf("reset %d: %v", i, err)
		}
	}
	if _, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, ids[10], "one too many"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("11th: %v", err)
	}
	e.Clock.Advance(61 * time.Minute)
	if _, err := e.Svc.AdminResetPassword(e.Ctx, admin, net.ID, ids[10], "next hour"); err != nil {
		t.Fatalf("after the window: %v", err)
	}
}

// Operator (CLI) reset: system actor only, audited with reason, forced change.
func TestOperatorReset(t *testing.T) {
	e := tu.NewEnv(t)
	ana := e.Person("ana@example.org")
	if _, err := e.Svc.OperatorResetPassword(e.Ctx, ana, "ana@example.org", "not an operator"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-system: %v", err)
	}
	sys := domain.SystemActor("cli-test")
	if _, err := e.Svc.OperatorResetPassword(e.Ctx, sys, "ana@example.org", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("reason: %v", err)
	}
	if _, err := e.Svc.OperatorResetPassword(e.Ctx, sys, "ghost@example.org", "no such person"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	_ = codesFor(t, e, ana)
	temp, err := e.Svc.OperatorResetPassword(e.Ctx, sys, "ANA@example.org", "ticket 42: lost password and codes")
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := e.Svc.Login(e.Ctx, "ana@example.org", temp)
	if err != nil {
		t.Fatal(err)
	}
	if a := tu.Must(e.Svc.Authenticate(e.Ctx, tok)); !a.MustChangePassword {
		t.Fatal("must be flagged")
	}
	if n := tu.Must(e.Svc.RecoveryCodesRemaining(e.Ctx, ana)); n != 0 {
		t.Fatalf("recovery codes must be revoked, %d left", n)
	}
	var typ, reason string
	if err := e.Store.DB().QueryRow(`SELECT actor_type, reason FROM audit_events WHERE action = ?`, domain.ActPersonPasswordReset).Scan(&typ, &reason); err != nil {
		t.Fatal(err)
	}
	if typ != string(domain.ActorOperator) || !strings.Contains(reason, "ticket 42") {
		t.Fatalf("audit: %s %q", typ, reason)
	}
}
