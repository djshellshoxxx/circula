package service_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func selfSessionLogin(t *testing.T, e *tu.Env, login string) (string, domain.Actor) {
	t.Helper()
	raw, _, err := e.Svc.Login(e.Ctx, login, pw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := e.Svc.Authenticate(e.Ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, a
}

func currentSessionID(t *testing.T, e *tu.Env, a domain.Actor) string {
	t.Helper()
	for _, row := range tu.Must(e.Svc.ListOwnSessions(e.Ctx, a, service.Page{})).Rows {
		if row.Current {
			return row.ID
		}
	}
	t.Fatal("current session missing")
	return ""
}

// The self-service selector is neither a bearer token nor its stored hash.
// Ownership is checked independently of knowing another session's selector.
func TestOwnSessionsArePrivateAndIndividuallyRevocable(t *testing.T) {
	e := tu.NewEnv(t)
	owner := e.Person("owner@example.org")
	e.Person("other@example.org")
	raw, a := selfSessionLogin(t, e, "owner@example.org")
	secondRaw, second := selfSessionLogin(t, e, "owner@example.org")
	foreignRaw, foreign := selfSessionLogin(t, e, "other@example.org")
	currentID := currentSessionID(t, e, a)
	secondID := currentSessionID(t, e, second)
	foreignID := currentSessionID(t, e, foreign)
	before := checksumExcept(t, e)
	rows := tu.Must(e.Svc.ListOwnSessions(e.Ctx, a, service.Page{})).Rows
	if len(rows) != 2 || checksumExcept(t, e) != before {
		t.Fatal("listing changed state or returned another owner's sessions")
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{raw, secondRaw, foreignRaw, a.SessionID, second.SessionID, foreign.SessionID, foreignID} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("session view exposed credentials or a foreign selector")
		}
	}
	for _, id := range []string{foreignID, a.SessionID, raw, "invalid"} {
		if _, err := e.Svc.GetOwnSession(e.Ctx, a, id); domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("foreign/invalid confirmation target: %v", err)
		}
		if err := e.Svc.RevokeOwnSession(e.Ctx, a, id); domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("foreign/invalid selector: %v", err)
		}
	}
	if _, err := e.Svc.ListOwnSessions(e.Ctx, owner, service.Page{}); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("non-session actor listed sessions: %v", err)
	}
	if _, err := e.Svc.ListOwnSessions(e.Ctx, domain.SystemActor("test"), service.Page{}); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("system actor listed sessions: %v", err)
	}
	spoof := a
	spoof.PersonID = foreign.PersonID
	if _, err := e.Svc.ListOwnSessions(e.Ctx, spoof, service.Page{}); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("session owner mismatch: %v", err)
	}
	readOnly := a
	readOnly.Scopes = []domain.Scope{domain.ScopeRead}
	if err := e.Svc.RevokeOwnSession(e.Ctx, readOnly, secondID); domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("read-only actor revoked a session: %v", err)
	}
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(a.PersonID, true) }); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.ListOwnSessions(e.Ctx, a, service.Page{}); err != nil {
		t.Fatalf("forced password change blocked an API read: %v", err)
	}
	if err := e.Svc.RevokeOwnSession(e.Ctx, a, secondID); domain.CodeOf(err) != domain.CodePasswordChangeRequired {
		t.Fatalf("stale actor bypassed required password change: %v", err)
	}
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(a.PersonID, false) }); err != nil {
		t.Fatal(err)
	}
	if err := e.Svc.RevokeOwnSession(e.Ctx, a, secondID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.Authenticate(e.Ctx, secondRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("ended session authenticated: %v", err)
	}
	for _, token := range []string{raw, foreignRaw} {
		if _, err := e.Svc.Authenticate(e.Ctx, token); err != nil {
			t.Fatalf("unselected session changed: %v", err)
		}
	}
	if err := e.Svc.RevokeOwnSession(e.Ctx, a, secondID); err != nil {
		t.Fatalf("repeat revocation: %v", err)
	}
	var auditCount int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action=?`, domain.ActSessionRevoked).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("revocation must audit once: count=%d err=%v", auditCount, err)
	}
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		events, err := tx.ListAudit("", store.AuditFilter{Action: domain.ActSessionRevoked})
		if err != nil {
			return err
		}
		if len(events) != 1 || events[0].TargetType != "person" || events[0].TargetID != a.PersonID ||
			events[0].SummaryBefore != "" || events[0].SummaryAfter != `{"current":false}` {
			t.Fatalf("session audit is not minimal owner metadata: %+v", events)
		}
		data, err := json.Marshal(events)
		if err != nil {
			return err
		}
		for _, secret := range []string{raw, secondRaw, foreignRaw, a.SessionID, second.SessionID, foreign.SessionID, currentID, secondID, foreignID} {
			if strings.Contains(string(data), secret) {
				t.Fatal("session audit exposed a credential or session selector")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.Svc.RevokeOwnSession(e.Ctx, a, currentID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.ListOwnSessions(e.Ctx, a, service.Page{}); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("stale actor listed after own revocation: %v", err)
	}
	if _, err := e.Svc.GetOwnSession(e.Ctx, a, currentID); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("stale actor fetched a confirmation target: %v", err)
	}
	if err := e.Svc.RevokeOwnSession(e.Ctx, a, secondID); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("stale actor wrote after own revocation: %v", err)
	}
}

// Listing applies the same absolute, idle and current-role limits as
// Authenticate, including their exact expiry boundaries.
func TestOwnSessionsApplyExistingLifetimeRules(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) {
		c.SessionIdle, c.SessionMax = time.Hour, 24*time.Hour
		c.AdminSessionIdle, c.AdminSessionMax = 5*time.Minute, 30*time.Minute
	})
	owner := e.Person("owner@example.org")
	e.Network(owner, "session-town", nil)
	_, actor := selfSessionLogin(t, e, "owner@example.org")
	now := e.Clock.Now()
	fixtures := []struct {
		raw              string
		created, seen    time.Time
		expires          time.Time
		revoked, allowed bool
	}{
		{"still-active", now.Add(-2 * time.Minute), now.Add(-time.Minute), now.Add(time.Hour), false, true},
		{"idle-boundary", now.Add(-10 * time.Minute), now.Add(-5 * time.Minute), now.Add(time.Hour), false, false},
		{"absolute-boundary", now.Add(-30 * time.Minute), now, now.Add(time.Hour), false, false},
		{"stored-boundary", now, now, now, false, false},
		{"already-revoked", now, now, now.Add(time.Hour), true, false},
	}
	publicIDs := map[string]string{}
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		for _, f := range fixtures {
			sum := sha256.Sum256([]byte(f.raw))
			id := hex.EncodeToString(sum[:])
			sess := &domain.Session{IDHash: id, PersonID: owner.PersonID,
				CreatedAt: f.created, LastSeenAt: f.seen, ExpiresAt: f.expires}
			if err := tx.InsertSession(sess); err != nil {
				return err
			}
			publicIDs[f.raw] = sess.PublicID
			if f.revoked {
				if _, err := tx.RevokeOwnedSession(owner.PersonID, id, now); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rows := tu.Must(e.Svc.ListOwnSessions(e.Ctx, actor, service.Page{})).Rows
	if len(rows) != 2 {
		t.Fatalf("dead sessions leaked into active list: %+v", rows)
	}
	for _, row := range rows {
		want := now.Add(4 * time.Minute)
		if row.Current {
			want = now.Add(5 * time.Minute)
		}
		if !row.ExpiresAt.Equal(want) {
			t.Errorf("role-specific expiry %s, want %s", row.ExpiresAt, want)
		}
	}
	for _, f := range fixtures {
		_, targetErr := e.Svc.GetOwnSession(e.Ctx, actor, publicIDs[f.raw])
		if (targetErr == nil) != f.allowed || targetErr != nil && domain.CodeOf(targetErr) != domain.CodeNotFound {
			t.Errorf("confirmation lifetime disagreement for %s: %v", f.raw, targetErr)
		}
		_, err := e.Svc.Authenticate(e.Ctx, f.raw)
		if (err == nil) != f.allowed {
			t.Errorf("list/auth lifetime disagreement for %s: %v", f.raw, err)
		}
	}
	// The API's idempotent revoke still accepts owned retained dead rows;
	// only the HTML confirmation read requires its target to remain active.
	for _, id := range []string{publicIDs["stored-boundary"], publicIDs["already-revoked"]} {
		for repeat := 0; repeat < 2; repeat++ {
			if err := e.Svc.RevokeOwnSession(e.Ctx, actor, id); err != nil {
				t.Fatalf("retained dead-row revoke: %v", err)
			}
		}
	}
}

func TestOwnSessionRevocationRollsBackWithAuditFailure(t *testing.T) {
	var faults *tu.FaultStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		faults = tu.NewFaultStore(st)
		return faults
	})
	e.Person("owner@example.org")
	_, actor := selfSessionLogin(t, e, "owner@example.org")
	raw, other := selfSessionLogin(t, e, "owner@example.org")
	id := currentSessionID(t, e, other)
	faults.Arm("InsertAudit", 1)
	if err := e.Svc.RevokeOwnSession(e.Ctx, actor, id); err == nil {
		t.Fatal("revocation committed without its audit")
	}
	faults.Disarm()
	if _, err := e.Svc.Authenticate(e.Ctx, raw); err != nil {
		t.Fatalf("failed revocation changed session: %v", err)
	}
}
