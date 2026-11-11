package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-003 with break-glass (batch-2 review): an administrator role granted
// through break-glass passes admin checks, so it must shorten the session to
// the administrator lifetime like an assigned role does.
func TestBreakGlassAdminGetsAdministratorSessionLifetime(t *testing.T) {
	w := newWorldCfg(t, nil, func(c *service.Config) { c.AdminSessionIdle = time.Hour })
	tok, _, err := w.Svc.Login(w.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	w.openBG(4 * time.Hour)
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "admin locked out"}))
	w.Clock.Advance(59 * time.Minute)
	if _, err := w.Svc.Authenticate(w.Ctx, tok); err != nil {
		t.Fatalf("session ended before the administrator idle timeout: %v", err)
	}
	w.Clock.Advance(61 * time.Minute)
	if _, err := w.Svc.Authenticate(w.Ctx, tok); err == nil {
		t.Error("break-glass administrator session accepted after 61 minutes idle")
	}
}

// SEC-027 / SA2-13 (batch-2 review): an emergency sign-out also drops pending
// second-factor challenges, which would otherwise turn into a session within
// their 5-minute life; one person's revocation leaves other people's alone.
func TestRevokeSessionsDropsPendingSecondFactorChallenges(t *testing.T) {
	e := tu.NewEnv(t)
	ana, bob := e.Person("ana@example.org"), e.Person("bob@example.org")
	anaSecret, _ := enrol(t, e, ana)
	bobSecret, _ := enrol(t, e, bob)

	chAna, chBob := login(e, "ana@example.org").MFAChallenge, login(e, "bob@example.org").MFAChallenge
	if chAna == "" || chBob == "" {
		t.Fatal("no second-factor challenge issued")
	}
	if _, err := e.Svc.RevokeAllSessions(e.Ctx, sys, service.RevokeSessionsInput{Login: "ana@example.org", Reason: "lost phone"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, chAna, totpAt(t, anaSecret, now(e))); err == nil {
		t.Error("ana's pending challenge still completes after her sessions were revoked")
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, chBob, totpAt(t, bobSecret, now(e))); err != nil {
		t.Errorf("bob's challenge was dropped by a revocation of ana only: %v", err)
	}

	e.Clock.Advance(31 * time.Second)
	chBob = login(e, "bob@example.org").MFAChallenge
	if _, err := e.Svc.RevokeAllSessions(e.Ctx, sys, service.RevokeSessionsInput{Reason: "incident 7", WebKeyRotated: true}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, chBob, totpAt(t, bobSecret, now(e))); err == nil {
		t.Error("a pending challenge still completes after everyone was signed out")
	}
	var after string
	if err := e.Store.DB().QueryRow(`SELECT summary_after FROM audit_events WHERE action = ? AND reason = ?`,
		domain.ActSessionsRevoked, "incident 7").Scan(&after); err != nil {
		t.Fatalf("audit event: %v", err)
	}
	if !strings.Contains(after, `"web_key_rotated":true`) {
		t.Errorf("audit event does not record the key rotation: %s", after)
	}
}
