package service_test

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// checksumExcept hashes the content of every table except skip, in rowid
// order: the ledger (accounts, entries, transactions, status events) and all
// other state.
func checksumExcept(t *testing.T, e *tu.Env, skip ...string) string {
	t.Helper()
	db := e.Store.DB()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	_ = rows.Close()
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
	h := sha256.New()
	for _, tb := range tables {
		if skipped[tb] {
			continue
		}
		r, err := db.Query("SELECT * FROM " + tb + " ORDER BY rowid")
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := r.Columns()
		fmt.Fprintf(h, "[%s]", tb)
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(h, "%v\n", vals)
		}
		_ = r.Close()
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ledgerChecksum hashes only the ledger tables.
func ledgerChecksum(t *testing.T, e *tu.Env) string {
	t.Helper()
	h := sha256.New()
	for _, q := range []string{
		`SELECT id, network_id, member_id, is_system, balance FROM accounts ORDER BY id`,
		`SELECT * FROM entries ORDER BY rowid`,
		`SELECT * FROM transactions ORDER BY rowid`,
		`SELECT * FROM transaction_status_events ORDER BY rowid`,
	} {
		r, err := e.Store.DB().Query(q)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := r.Columns()
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(h, "%v\n", vals)
		}
		_ = r.Close()
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// SEC-027 / OPS-010: the operator's revocation ends every old session (or one
// person's), records one audited platform event and leaves the ledger and all
// other data unchanged; API tokens stay valid.
func TestRevokeAllSessions(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1200)
	w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 300)
	login := func(l string) string {
		tok, _, err := w.Svc.Login(w.Ctx, l, pw)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ana1, ana2, ben := login("ana@example.org"), login("ana@example.org"), login("ben@example.org")
	admin := login("admin@example.org")
	apiTok, _ := tu.Must2(w.Svc.CreateAPIToken(w.Ctx, w.a, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead}}))
	tu.Must(0, w.Svc.Logout(w.Ctx, admin)) // already revoked: not counted again

	sys := domain.SystemActor("test")
	_, err := w.Svc.RevokeAllSessions(w.Ctx, w.admin, service.RevokeSessionsInput{Reason: "not an operator"})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.RevokeAllSessions(w.Ctx, sys, service.RevokeSessionsInput{Reason: " "})
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.RevokeAllSessions(w.Ctx, sys, service.RevokeSessionsInput{Login: "ghost@example.org", Reason: "suspected leak"})
	wantCode(t, err, domain.ErrNotFound)

	ledger := ledgerChecksum(t, w.Env)
	auditBefore := countRows(t, w.Env, "audit_events")
	// Everything but the sessions and the audit log is compared around each
	// call (Authenticate itself touches tokens and activity rows).
	others := func() string {
		return checksumExcept(t, w.Env, "sessions", "audit_events", "audit_heads", "audit_subjects")
	}
	rest := others()

	// one person
	if n := tu.Must(w.Svc.RevokeAllSessions(w.Ctx, sys, service.RevokeSessionsInput{Login: " BEN@example.org", Reason: "lost phone"})); n != 1 {
		t.Errorf("revoked %d sessions of ben, want 1", n)
	}
	if others() != rest {
		t.Error("revoking one person's sessions changed data other than sessions and the audit log")
	}
	if _, err := w.Svc.Authenticate(w.Ctx, ben); err == nil {
		t.Error("ben's old session accepted after revocation")
	}
	for _, tok := range []string{ana1, ana2} {
		if _, err := w.Svc.Authenticate(w.Ctx, tok); err != nil {
			t.Errorf("another person's session was revoked: %v", err)
		}
	}

	// everyone
	rest = others()
	if n := tu.Must(w.Svc.RevokeAllSessions(w.Ctx, sys, service.RevokeSessionsInput{Reason: "web key rotation"})); n != 2 {
		t.Errorf("revoked %d sessions, want 2 (ana's two; ben's and the admin's were already revoked)", n)
	}
	if others() != rest {
		t.Error("revoking every session changed data other than sessions and the audit log")
	}
	for _, tok := range []string{ana1, ana2, ben, admin} {
		if _, err := w.Svc.Authenticate(w.Ctx, tok); err == nil {
			t.Error("old session accepted after revoking every session")
		}
	}
	if _, err := w.Svc.AuthenticateToken(w.Ctx, apiTok); err != nil {
		t.Errorf("API token revoked too: %v", err)
	}
	if got := ledgerChecksum(t, w.Env); got != ledger {
		t.Error("ledger checksum changed")
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
	if got := countRows(t, w.Env, "audit_events"); got != auditBefore+2 {
		t.Errorf("audit events %d -> %d, want +2", auditBefore, got)
	}
	var typ, reason, after, netID string
	if err := w.Store.DB().QueryRow(`SELECT actor_type, reason, summary_after, network_id FROM audit_events
		WHERE action = ? ORDER BY seq DESC LIMIT 1`, domain.ActSessionsRevoked).Scan(&typ, &reason, &after, &netID); err != nil {
		t.Fatal(err)
	}
	if typ != string(domain.ActorOperator) || reason != "web key rotation" || after != `{"revoked":2,"scope":"all"}` || netID != "" {
		t.Errorf("audit event: %s %q %s net=%q", typ, reason, after, netID)
	}
	// signing in again works and yields a fresh, valid session
	if _, err := w.Svc.Authenticate(w.Ctx, login("ana@example.org")); err != nil {
		t.Errorf("new session after revocation: %v", err)
	}
}

// SEC-003 / DEC-005: role-specific idle and absolute timeouts apply to a person
// who holds the administrator or reviewer role in any network; members keep
// the member timeouts. Injected clock.
func TestRoleSpecificSessionTimeouts(t *testing.T) {
	w := newWorldCfg(t, nil, func(c *service.Config) {
		c.AdminSessionIdle, c.AdminSessionMax = time.Hour, 12*time.Hour
		c.ReviewerSessionIdle, c.ReviewerSessionMax = 2*time.Hour, 24*time.Hour
	})
	login := func(l string) string {
		tok, _, err := w.Svc.Login(w.Ctx, l, pw)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ok := func(tok string) bool { _, err := w.Svc.Authenticate(w.Ctx, tok); return err == nil }

	// idle: the administrator is signed out after 1h idle, a member is not
	adm, ana := login("admin@example.org"), login("ana@example.org")
	w.Clock.Advance(59 * time.Minute)
	if !ok(adm) || !ok(ana) {
		t.Fatal("sessions ended before the idle timeout")
	}
	w.Clock.Advance(61 * time.Minute)
	if ok(adm) {
		t.Error("administrator session accepted after 61 minutes idle")
	}
	if !ok(ana) {
		t.Error("member session ended by the administrator timeout")
	}

	// absolute: activity keeps an administrator session alive only up to 12h
	adm = login("admin@example.org")
	ana = login("ana@example.org")
	for i := 0; i < 14; i++ { // 14 x 50 min = 11h40m
		w.Clock.Advance(50 * time.Minute)
		if !ok(adm) || !ok(ana) {
			t.Fatalf("step %d: active session ended early", i)
		}
	}
	w.Clock.Advance(25 * time.Minute) // 12h05m after sign-in
	if ok(adm) {
		t.Error("administrator session accepted beyond the 12h absolute lifetime")
	}
	if !ok(ana) {
		t.Error("member session ended by the administrator absolute lifetime")
	}

	// a role granted after sign-in shortens the running session at once
	ben := login("ben@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleReviewer))
	w.Clock.Advance(119 * time.Minute)
	if !ok(ben) {
		t.Error("reviewer session ended before 2h idle")
	}
	w.Clock.Advance(121 * time.Minute)
	if ok(ben) {
		t.Error("reviewer session accepted after 2h idle")
	}
	// after the role is removed the member timeouts apply again
	tu.Must(0, w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleReviewer))
	ben = login("ben@example.org")
	w.Clock.Advance(3 * time.Hour)
	if !ok(ben) {
		t.Error("former reviewer still gets the reviewer timeout")
	}
}

// SEC-003: without role-specific settings (the default) an administrator has
// the member timeouts, as before; a role value longer than the member value
// never lengthens a session.
func TestRoleSessionTimeoutsDefaultUnchanged(t *testing.T) {
	w := newWorldCfg(t, nil, func(c *service.Config) {
		c.SessionIdle, c.SessionMax = 2*time.Hour, 4*time.Hour
		c.AdminSessionIdle, c.AdminSessionMax = 10*time.Hour, 100*time.Hour // longer: ignored
	})
	adm, _, err := w.Svc.Login(w.Ctx, "admin@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(119 * time.Minute)
	if _, err := w.Svc.Authenticate(w.Ctx, adm); err != nil {
		t.Errorf("admin session ended early: %v", err)
	}
	w.Clock.Advance(121 * time.Minute)
	if _, err := w.Svc.Authenticate(w.Ctx, adm); err == nil {
		t.Error("a longer administrator idle timeout lengthened the session")
	}

	d := newWorld(t, nil) // all defaults: 30 days idle for everyone
	tok, _, err := d.Svc.Login(d.Ctx, "admin@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	d.Clock.Advance(29 * 24 * time.Hour)
	if _, err := d.Svc.Authenticate(d.Ctx, tok); err != nil {
		t.Errorf("default administrator session ended before the member idle timeout: %v", err)
	}
}
