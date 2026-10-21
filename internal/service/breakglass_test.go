package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

var sys = domain.SystemActor("test-operator")

func (w *world) openBG(ttl time.Duration) *domain.BreakGlassSession {
	w.T.Helper()
	return tu.Must(w.Svc.OpenBreakGlass(w.Ctx, sys, service.OpenBreakGlassInput{Network: "demo", Operator: "Olga", Reason: "ticket 42: repair", TTL: ttl}))
}

func (w *world) auditActions(netID string) []string {
	w.T.Helper()
	var out []string
	err := w.Store.View(w.Ctx, func(tx store.Tx) error {
		evs, err := tx.ListAudit(netID, store.AuditFilter{Limit: 500})
		for _, e := range evs {
			out = append(out, e.Action)
		}
		return err
	})
	if err != nil {
		w.T.Fatal(err)
	}
	return out
}

func countStr(list []string, s string) int {
	n := 0
	for _, x := range list {
		if x == s {
			n++
		}
	}
	return n
}

// SEC-013: the TTL is clamped to four hours and the database agrees.
func TestBreakGlassTTLClamp(t *testing.T) {
	w := newWorld(t, nil)
	cases := []struct {
		name string
		ttl  time.Duration
		want time.Duration
	}{
		{"default", 0, 4 * time.Hour},
		{"negative", -time.Hour, 4 * time.Hour},
		{"short kept", time.Hour, time.Hour},
		{"exactly max", 4 * time.Hour, 4 * time.Hour},
		{"over max clamped", 24 * time.Hour, 4 * time.Hour},
		{"one ms over", 4*time.Hour + time.Millisecond, 4 * time.Hour},
	}
	for _, c := range cases {
		b := w.openBG(c.ttl)
		if got := b.ExpiresAt.Sub(b.OpenedAt); got != c.want {
			t.Errorf("%s: ttl %v, want %v", c.name, got, c.want)
		}
	}
	// A direct insert past four hours is refused by the schema.
	_, err := w.Store.DB().Exec(`INSERT INTO break_glass_sessions(id,network_id,operator_label,reason,opened_at,expires_at)
		VALUES ('x',?, 'o','r',0,14400001)`, w.net.ID)
	if err == nil {
		t.Error("schema accepted a session longer than four hours")
	}
	// Closed sessions cannot be reopened or edited.
	b := w.openBG(time.Hour)
	if _, err := w.Store.DB().Exec(`UPDATE break_glass_sessions SET expires_at = expires_at + 1000 WHERE id = ?`, b.ID); err == nil {
		t.Error("expiry was editable")
	}
	if _, err := w.Store.DB().Exec(`DELETE FROM break_glass_sessions WHERE id = ?`, b.ID); err == nil {
		t.Error("session was deletable")
	}
}

func TestBreakGlassInputAndAuthority(t *testing.T) {
	w := newWorld(t, nil)
	_, err := w.Svc.OpenBreakGlass(w.Ctx, sys, service.OpenBreakGlassInput{Network: "demo", Operator: "Olga", Reason: " "})
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.OpenBreakGlass(w.Ctx, sys, service.OpenBreakGlassInput{Network: "demo", Operator: "", Reason: "x"})
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.OpenBreakGlass(w.Ctx, sys, service.OpenBreakGlassInput{Network: "nope", Operator: "Olga", Reason: "x"})
	wantCode(t, err, domain.ErrNotFound)
	// Only the operator CLI (System actor) may use the operator methods: not even a network administrator.
	for _, a := range []domain.Actor{w.admin, w.a} {
		_, err = w.Svc.OpenBreakGlass(w.Ctx, a, service.OpenBreakGlassInput{Network: "demo", Operator: "Olga", Reason: "x"})
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.Svc.GrantBreakGlassAdmin(w.Ctx, a, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "x"})
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.Svc.CloseBreakGlass(w.Ctx, a, "id", "")
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.Svc.ExpireBreakGlass(w.Ctx, a)
		wantCode(t, err, domain.ErrForbidden)
	}
}

// Expiry at the four hour boundary: live one millisecond before, over at the instant.
func TestBreakGlassExpiryBoundary(t *testing.T) {
	w := newWorld(t, nil)
	wantCode(t, w.Svc.RequireBreakGlass(w.Ctx, w.net.ID), domain.ErrForbidden) // none yet
	b := w.openBG(0)
	if err := w.Svc.RequireBreakGlass(w.Ctx, w.net.ID); err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(4*time.Hour - time.Millisecond)
	if err := w.Svc.RequireBreakGlass(w.Ctx, w.net.ID); err != nil {
		t.Fatalf("session over one millisecond early: %v", err)
	}
	w.Clock.Advance(time.Millisecond) // now == expires_at
	wantCode(t, w.Svc.RequireBreakGlass(w.Ctx, w.net.ID), domain.ErrForbidden)
	if !b.ExpiresAt.Equal(w.Clock.Now()) {
		t.Fatalf("clock %v != expiry %v", w.Clock.Now(), b.ExpiresAt)
	}
}

// Expired sessions are closed lazily by the next operator call and by the job, each audited once.
func TestBreakGlassAutoClose(t *testing.T) {
	w := newWorld(t, nil)
	b1 := w.openBG(time.Hour)
	w.Clock.Advance(2 * time.Hour)
	// Listing reports it expired before anything closed it.
	hist := tu.Must(w.Svc.ListBreakGlass(w.Ctx, w.admin, w.net.ID, service.Page{}))
	if len(hist) != 1 || hist[0].Status(w.Clock.Now()) != "expired" || hist[0].ClosedAt != nil {
		t.Fatalf("history before sweep: %+v", hist)
	}
	n := tu.Must(w.Svc.ExpireBreakGlass(w.Ctx, sys))
	if n != 1 {
		t.Fatalf("job closed %d sessions", n)
	}
	if n := tu.Must(w.Svc.ExpireBreakGlass(w.Ctx, sys)); n != 0 {
		t.Fatalf("job closed %d on the second run", n)
	}
	got := tu.Must(w.Svc.ListBreakGlassOperator(w.Ctx, sys, "demo", service.Page{}))
	if got[0].ID != b1.ID || got[0].ClosedAt == nil || got[0].ClosedReason != domain.BreakGlassClosedExpired || !got[0].ClosedAt.Equal(b1.ExpiresAt) {
		t.Fatalf("closed row: %+v", got[0])
	}
	// Lazy path: a second session expires and the next Open closes it.
	w.openBG(time.Hour)
	w.Clock.Advance(2 * time.Hour)
	w.openBG(time.Hour)
	acts := w.auditActions(w.net.ID)
	if countStr(acts, domain.ActBreakGlassExpired) != 2 || countStr(acts, domain.ActBreakGlassOpened) != 3 {
		t.Errorf("audit actions: %v", acts)
	}
}

// SEC-013 b / SEC-012: platform AND network scope events exist; every one verifies.
func TestBreakGlassAuditRows(t *testing.T) {
	w := newWorld(t, nil)
	b := w.openBG(time.Hour)
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "admin locked out"}))
	tu.Must(w.Svc.CloseBreakGlass(w.Ctx, sys, b.ID, "done"))
	netActs, platActs := w.auditActions(w.net.ID), w.auditActions("")
	for _, a := range []string{domain.ActBreakGlassOpened, domain.ActBreakGlassRoleGranted, domain.ActBreakGlassClosed} {
		if countStr(netActs, a) != 1 {
			t.Errorf("network scope lacks %s: %v", a, netActs)
		}
		if countStr(platActs, a) != 1 {
			t.Errorf("platform scope lacks %s: %v", a, platActs)
		}
	}
	// The network's administrators can read them through the audit log, and the chain verifies.
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActBreakGlassOpened}))
	if len(evs) != 1 || evs[0].ActorType != domain.ActorOperator || evs[0].Reason != "ticket 42: repair" {
		t.Fatalf("admin-visible event: %+v", evs)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("verify: %v", rep.Findings)
	}
	// Closing twice is refused.
	_, err := w.Svc.CloseBreakGlass(w.Ctx, sys, b.ID, "again")
	wantCode(t, err, domain.ErrInvalidState)
	// The granted member sees the grant in their own audit view (a rights change notice).
	mine := tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{}))
	found := false
	for _, e := range mine {
		found = found || e.Action == domain.ActBreakGlassRoleGranted
	}
	if !found {
		t.Error("granted member cannot see the grant in their audit view")
	}
}

// The temporary admin role needs a session, and ends with it.
func TestBreakGlassGrantAdminExpires(t *testing.T) {
	w := newWorld(t, nil)
	in := service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "admin locked out"}
	_, err := w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, in)
	wantCode(t, err, domain.ErrForbidden) // no session: break-glass is the only way
	w.openBG(time.Hour)
	_, err = w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden) // opening a session alone grants nothing
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, in))
	if _, err := w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{}); err != nil {
		t.Fatalf("granted admin refused: %v", err)
	}
	// Other members are unaffected.
	_, err = w.Svc.AdminListMembers(w.Ctx, w.b, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	// The grant is temporary: not in the permanent role table, not counted as an administrator.
	m := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if m.HasRole(domain.RoleAdmin) {
		t.Error("break-glass admin leaked into the permanent roles")
	}
	_, err = w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, in)
	wantCode(t, err, domain.ErrInvalidState) // already granted
	// One millisecond before expiry still admin; at expiry it is gone with no job run.
	w.Clock.Advance(time.Hour - time.Millisecond)
	if _, err := w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{}); err != nil {
		t.Fatalf("lost admin early: %v", err)
	}
	w.Clock.Advance(time.Millisecond)
	_, err = w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	// A later session needs its own grant: the old one does not revive.
	w.openBG(time.Hour)
	_, err = w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
}

func TestBreakGlassCloseEndsGrantAndGrantRules(t *testing.T) {
	w := newWorld(t, nil)
	b := w.openBG(time.Hour)
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "x"}))
	tu.Must(w.Svc.CloseBreakGlass(w.Ctx, sys, b.ID, ""))
	_, err := w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	w.openBG(time.Hour)
	bad := []service.GrantBreakGlassInput{
		{Network: "demo", Login: "admin@example.org", Reason: "x"},    // already a permanent admin
		{Network: "demo", Login: "stranger@example.org", Reason: "x"}, // no such person
		{Network: "demo", Login: "ana@example.org", Reason: ""},       // reason mandatory
	}
	for _, in := range bad {
		if _, err := w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, in); err == nil {
			t.Errorf("grant accepted: %+v", in)
		}
	}
	// Not an active member of the network.
	w.Person("outsider@example.org")
	_, err = w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "outsider@example.org", Reason: "x"})
	wantCode(t, err, domain.ErrNotFound)
	// A suspended member cannot receive the role.
	if err := w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "rules"); err != nil {
		t.Fatal(err)
	}
	_, err = w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ben@example.org", Reason: "x"})
	wantCode(t, err, domain.ErrInvalidState)
}

// SEC-015: sessions and grants never cross networks; only administrators of the network see its history.
func TestBreakGlassCrossTenant(t *testing.T) {
	w := newWorld(t, nil)
	boss := w.Person("boss@other.example")
	other, _ := w.Network(boss, "other", nil)
	w.openBG(time.Hour) // demo only
	if err := w.Svc.RequireBreakGlass(w.Ctx, other.ID); err == nil {
		t.Fatal("a session on demo opened access to other")
	}
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "x"}))
	// The grant makes ana admin in demo only: other network is invisible to her.
	_, err := w.Svc.AdminListMembers(w.Ctx, w.a, other.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.ListBreakGlass(w.Ctx, w.a, other.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	// ana, a member of both networks, is no admin in other through the demo grant.
	tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, boss, other.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, w.a, tok, "Ana"))
	_, err = w.Svc.AdminListMembers(w.Ctx, w.a, other.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	// Admins see their own network's history only; the other network's admin sees none of demo's.
	got := tu.Must(w.Svc.ListBreakGlass(w.Ctx, boss, other.ID, service.Page{}))
	if len(got) != 0 {
		t.Errorf("other network sees %d sessions", len(got))
	}
	_, err = w.Svc.ListBreakGlass(w.Ctx, boss, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	got = tu.Must(w.Svc.ListBreakGlass(w.Ctx, w.admin, w.net.ID, service.Page{}))
	if len(got) != 1 || got[0].NetworkID != w.net.ID {
		t.Errorf("demo admin history: %+v", got)
	}
	// Plain members and tokens scoped elsewhere are refused.
	_, err = w.Svc.ListBreakGlass(w.Ctx, w.b, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	scoped := w.admin
	scoped.NetworkScope = other.ID
	_, err = w.Svc.ListBreakGlass(w.Ctx, scoped, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	// Operator events of demo are not in other's audit log.
	if countStr(w.auditActions(other.ID), domain.ActBreakGlassOpened) != 0 {
		t.Error("break-glass event leaked into another network's audit log")
	}
}
