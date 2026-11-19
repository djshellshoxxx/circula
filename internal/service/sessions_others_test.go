package service_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func seedOtherSession(t *testing.T, e *tu.Env, owner domain.Actor, raw string, expired bool) *domain.Session {
	t.Helper()
	hash := sha256.Sum256([]byte(raw))
	now := e.Clock.Now()
	expires := now.Add(time.Hour)
	if expired {
		expires = now
	}
	sess := &domain.Session{IDHash: hex.EncodeToString(hash[:]), PersonID: owner.PersonID,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: expires}
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.InsertSession(sess) }); err != nil {
		t.Fatal(err)
	}
	return sess
}

func seedSessionChallenge(t *testing.T, e *tu.Env, owner domain.Actor, hash string) {
	t.Helper()
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.InsertMFAChallenge(&domain.MFAChallenge{IDHash: hash, PersonID: owner.PersonID,
			CreatedAt: e.Clock.Now(), ExpiresAt: e.Clock.Now().Add(5 * time.Minute)})
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeOtherOwnSessionsPreservesCurrentAndForeignCredentials(t *testing.T) {
	e := tu.NewEnv(t)
	owner, foreign := e.Person("owner@example.org"), e.Person("foreign@example.org")
	raw, actor := selfSessionLogin(t, e, "owner@example.org")
	foreignRaw, _ := selfSessionLogin(t, e, "foreign@example.org")
	apiToken, _ := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, actor, service.TokenInput{
		Name: "retained", Scopes: []domain.Scope{domain.ScopeRead},
	}))
	other := seedOtherSession(t, e, owner, "other-session-credential", false)
	expired := seedOtherSession(t, e, owner, "expired-session-credential", true)
	already := seedOtherSession(t, e, owner, "already-revoked-credential", false)
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.RevokeSession(already.IDHash, e.Clock.Now()) }); err != nil {
		t.Fatal(err)
	}
	seedSessionChallenge(t, e, owner, "owner-pending-challenge-hash")
	seedSessionChallenge(t, e, foreign, "foreign-pending-challenge-hash")
	unchanged := checksumExcept(t, e, "sessions", "mfa_challenges", "audit_events", "audit_heads", "audit_subjects")
	for i, want := range []int64{2, 0} {
		n, err := e.Svc.RevokeOtherOwnSessions(e.Ctx, actor)
		if err != nil || n != want {
			t.Fatalf("bulk request %d: revoked=%d err=%v want=%d", i, n, err, want)
		}
	}
	if after := checksumExcept(t, e, "sessions", "mfa_challenges", "audit_events", "audit_heads", "audit_subjects"); after != unchanged {
		t.Fatal("bulk revocation changed unrelated account, monetary or governance state")
	}
	if _, err := e.Svc.Authenticate(e.Ctx, "other-session-credential"); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("revoked credential authenticated: %v", err)
	}
	for _, credential := range []string{raw, foreignRaw} {
		if _, err := e.Svc.Authenticate(e.Ctx, credential); err != nil {
			t.Fatalf("retained credential stopped authenticating: %v", err)
		}
	}
	if _, err := e.Svc.AuthenticateToken(e.Ctx, apiToken); err != nil {
		t.Fatalf("bulk session revocation changed an API token: %v", err)
	}
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		for _, id := range []string{other.IDHash, expired.IDHash, already.IDHash} {
			sess, err := tx.GetSession(id)
			if err != nil || sess.RevokedAt == nil {
				t.Fatalf("other retained row not revoked: id=%s err=%v", id, err)
			}
		}
		if _, err := tx.GetMFAChallenge("owner-pending-challenge-hash"); domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("owner challenge survived sign-out: %v", err)
		}
		if _, err := tx.GetMFAChallenge("foreign-pending-challenge-hash"); err != nil {
			t.Fatalf("foreign challenge changed: %v", err)
		}
		events, err := tx.ListAudit("", store.AuditFilter{Action: "sessions.others_revoked"})
		if err != nil {
			return err
		}
		if len(events) != 2 {
			t.Fatalf("bulk requests require audit records, including a no-op: %d", len(events))
		}
		for _, event := range events {
			if event.TargetType != "person" || event.TargetID != actor.PersonID || event.SummaryBefore != "" || event.Reason != "" {
				t.Fatalf("bulk audit contains unrelated metadata: %+v", event)
			}
			if event.SummaryAfter != `{"current_retained":true,"revoked":2}` && event.SummaryAfter != `{"current_retained":true,"revoked":0}` {
				t.Fatalf("bulk audit is not minimal: %q", event.SummaryAfter)
			}
		}
		encoded, err := json.Marshal(events)
		if err != nil {
			return err
		}
		for _, secret := range []string{raw, foreignRaw, apiToken, actor.SessionID, other.IDHash, other.PublicID,
			"owner-pending-challenge-hash", "foreign-pending-challenge-hash"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatal("bulk audit exposed session or challenge material")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeOtherOwnSessionsRevalidatesAuthority(t *testing.T) {
	for _, mode := range []string{"no-session", "operator", "read-only", "network-scoped", "foreign", "missing", "revoked", "expired", "forced-password-change", "role-expired"} {
		t.Run(mode, func(t *testing.T) {
			e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.AdminSessionIdle = time.Minute })
			owner := e.Person("owner@example.org")
			foreign := e.Person("foreign@example.org")
			_, actor := selfSessionLogin(t, e, "owner@example.org")
			other := seedOtherSession(t, e, owner, "other-credential", false)
			seedSessionChallenge(t, e, owner, "pending-challenge-hash")
			switch mode {
			case "no-session":
				actor.SessionID = ""
			case "operator":
				actor = domain.SystemActor("operator")
			case "read-only":
				actor.Scopes = []domain.Scope{domain.ScopeRead}
			case "network-scoped":
				actor.NetworkScope = "restricted"
			case "foreign":
				actor.PersonID = foreign.PersonID
			case "missing":
				actor.SessionID = "missing"
			case "revoked":
				if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.RevokeSession(actor.SessionID, e.Clock.Now()) }); err != nil {
					t.Fatal(err)
				}
			case "expired":
				if _, err := e.Store.DB().Exec(`UPDATE sessions SET expires_at=? WHERE id=?`, e.Clock.Now().UnixMilli(), actor.SessionID); err != nil {
					t.Fatal(err)
				}
			case "forced-password-change":
				if err := e.Store.Update(e.Ctx, func(tx store.Tx) error { return tx.SetMustChangePassword(actor.PersonID, true) }); err != nil {
					t.Fatal(err)
				}
			case "role-expired":
				e.Network(owner, "strict-admin", nil)
				e.Clock.Advance(time.Minute)
			}
			before := checksumExcept(t, e)
			if n, err := e.Svc.RevokeOtherOwnSessions(e.Ctx, actor); err == nil || n != 0 {
				t.Fatalf("invalid authority revoked=%d err=%v", n, err)
			}
			if checksumExcept(t, e) != before {
				t.Fatal("rejected bulk revocation mutated database state")
			}
			if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
				sess, err := tx.GetSession(other.IDHash)
				if err != nil || sess.RevokedAt != nil {
					t.Fatalf("rejected bulk revoked target: %+v %v", sess, err)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRevokeOtherOwnSessionsRollsBackAuditAndChallengeFailure(t *testing.T) {
	for _, fail := range []string{"audit", "challenge"} {
		t.Run(fail, func(t *testing.T) {
			var faults *tu.FaultStore
			e := tu.NewEnvWith(t, func(st store.Store) store.Store { faults = tu.NewFaultStore(st); return faults })
			owner := e.Person("owner@example.org")
			_, actor := selfSessionLogin(t, e, "owner@example.org")
			seedOtherSession(t, e, owner, "other-credential", false)
			seedSessionChallenge(t, e, owner, "pending-challenge-hash")
			if fail == "audit" {
				faults.Arm("InsertAudit", 1)
			} else if _, err := e.Store.DB().Exec(`CREATE TRIGGER fail_challenge_delete BEFORE DELETE ON mfa_challenges
				BEGIN SELECT RAISE(ABORT,'test challenge failure'); END`); err != nil {
				t.Fatal(err)
			}
			before := checksumExcept(t, e)
			if n, err := e.Svc.RevokeOtherOwnSessions(e.Ctx, actor); err == nil || n != 0 {
				t.Fatalf("failed bulk revocation returned success: n=%d err=%v", n, err)
			}
			faults.Disarm()
			if checksumExcept(t, e) != before {
				t.Fatal("bulk failure left a partial session, challenge or audit write")
			}
			if _, err := e.Svc.Authenticate(e.Ctx, "other-credential"); err != nil {
				t.Fatalf("rollback lost other session: %v", err)
			}
		})
	}
}

func TestRevokeOtherOwnSessionsConcurrentWithIndividualRevocation(t *testing.T) {
	e := tu.NewEnv(t)
	owner := e.Person("owner@example.org")
	raw, actor := selfSessionLogin(t, e, "owner@example.org")
	const others = 16
	var target string
	for i := 0; i < others; i++ {
		target = seedOtherSession(t, e, owner, fmt.Sprintf("other-credential-%d", i), false).PublicID
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	counts := make(chan int64, 8)
	errs := make(chan error, 9)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			n, err := e.Svc.RevokeOtherOwnSessions(e.Ctx, actor)
			counts <- n
			errs <- err
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		errs <- e.Svc.RevokeOwnSession(e.Ctx, actor, target)
	}()
	close(start)
	wg.Wait()
	close(counts)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent revocation: %v", err)
		}
	}
	var total int64
	for n := range counts {
		total += n
	}
	var individual int64
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action=?`, domain.ActSessionRevoked).Scan(&individual); err != nil {
		t.Fatal(err)
	}
	if total+individual != others {
		t.Fatalf("concurrent revocation lost or double-counted rows: bulk=%d individual=%d", total, individual)
	}
	for i := 0; i < others; i++ {
		if _, err := e.Svc.Authenticate(e.Ctx, fmt.Sprintf("other-credential-%d", i)); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("other session %d survived concurrent revocation: %v", i, err)
		}
	}
	if _, err := e.Svc.Authenticate(e.Ctx, raw); err != nil {
		t.Fatalf("concurrent revocation ended current session: %v", err)
	}
}

func TestRevokeOtherOwnSessionsDropsPendingMFALogins(t *testing.T) {
	e := tu.NewEnv(t)
	owner, foreign := e.Person("owner@example.org"), e.Person("foreign@example.org")
	ownerSecret, _ := enrol(t, e, owner)
	foreignSecret, _ := enrol(t, e, foreign)
	currentChallenge := login(e, "owner@example.org").MFAChallenge
	raw, _, err := e.Svc.CompleteMFA(e.Ctx, currentChallenge, totpAt(t, ownerSecret, now(e)))
	if err != nil {
		t.Fatal(err)
	}
	actor := tu.Must(e.Svc.Authenticate(e.Ctx, raw))
	e.Clock.Advance(31 * time.Second)
	ownerChallenge := login(e, "owner@example.org").MFAChallenge
	foreignChallenge := login(e, "foreign@example.org").MFAChallenge
	if ownerChallenge == "" || foreignChallenge == "" {
		t.Fatal("MFA login fixtures did not issue pending challenges")
	}
	// There are no other established owner sessions. Pending login authority
	// still has to be invalidated, and the successful request still audited.
	if n, err := e.Svc.RevokeOtherOwnSessions(e.Ctx, actor); err != nil || n != 0 {
		t.Fatalf("challenge-only bulk revocation: n=%d err=%v", n, err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, ownerChallenge, totpAt(t, ownerSecret, now(e))); err == nil {
		t.Fatal("pending owner MFA login created a session after bulk sign-out")
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, foreignChallenge, totpAt(t, foreignSecret, now(e))); err != nil {
		t.Fatalf("bulk sign-out changed a foreign pending MFA login: %v", err)
	}
	if _, err := e.Svc.Authenticate(e.Ctx, raw); err != nil {
		t.Fatalf("bulk sign-out ended current MFA-authenticated session: %v", err)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action='sessions.others_revoked'`); n != 1 {
		t.Fatalf("challenge-only sign-out was not audited: %d", n)
	}
}

func TestRevokeOtherOwnSessionsRacingCurrentRevocation(t *testing.T) {
	e := tu.NewEnv(t)
	owner := e.Person("owner@example.org")
	_, actor := selfSessionLogin(t, e, "owner@example.org")
	currentID := currentSessionID(t, e, actor)
	other := seedOtherSession(t, e, owner, "other-racing-credential", false)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var count int64
	var bulkErr, currentErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		count, bulkErr = e.Svc.RevokeOtherOwnSessions(e.Ctx, actor)
	}()
	go func() {
		defer wg.Done()
		<-start
		currentErr = e.Svc.RevokeOwnSession(e.Ctx, actor, currentID)
	}()
	close(start)
	wg.Wait()
	if currentErr != nil {
		t.Fatalf("current revocation failed: %v", currentErr)
	}
	if bulkErr != nil && domain.CodeOf(bulkErr) != domain.CodeUnauthenticated {
		t.Fatalf("unexpected concurrent bulk error: %v", bulkErr)
	}
	if bulkErr == nil && count != 1 || bulkErr != nil && count != 0 {
		t.Fatalf("bulk result disagrees with commit order: count=%d err=%v", count, bulkErr)
	}
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		current, err := tx.GetSession(actor.SessionID)
		if err != nil || current.RevokedAt == nil {
			t.Fatalf("current session survived: %+v %v", current, err)
		}
		target, err := tx.GetSession(other.IDHash)
		if err != nil || (target.RevokedAt != nil) != (bulkErr == nil) {
			t.Fatalf("revoked actor mutated other session: %+v %v", target, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
