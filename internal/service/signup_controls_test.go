package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func wantField(t *testing.T, err error, field string) {
	t.Helper()
	wantCode(t, err, domain.ErrValidation)
	de, _ := err.(*domain.Error)
	if de == nil || len(de.Fields) != 1 || de.Fields[0].Field != field {
		t.Errorf("error %v: want one field error at %s", err, field)
	}
}

func countAction(t *testing.T, e *tu.Env, action string) int {
	t.Helper()
	var n int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func grantCount(t *testing.T, e *tu.Env, netID string) int {
	t.Helper()
	var n int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind = 'grant' AND network_id = ?`, netID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// SEC-032 acceptance 1: the default network configuration has grant zero.
func TestSignupGrantDefaultZero(t *testing.T) {
	if domain.DefaultPolicy().SignupGrant != 0 {
		t.Fatal("default policy has a sign-up grant")
	}
	e := tu.NewEnv(t)
	a := e.Person("founder@example.org")
	n, _ := tu.Must2(e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Plain", Slug: "plain", CreatorDisplayName: "F"}))
	if n.Policy.SignupGrant != 0 {
		t.Errorf("new network grant = %d", n.Policy.SignupGrant)
	}
}

// SEC-032 acceptance 2: enabling a grant in open join mode without the
// confirmation step is refused, at creation and on a policy change; the
// confirmation is an audited action.
func TestOpenModeGrantNeedsConfirmation(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("founder@example.org")
	pol := domain.DefaultPolicy()
	pol.JoinMode, pol.SignupGrant = domain.JoinOpen, 500
	_, _, err := e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Open", Slug: "open-net", Policy: &pol, CreatorDisplayName: "F"})
	wantField(t, err, "confirm_open_grant_risk")
	if countRows(t, e, "networks") != 0 {
		t.Fatal("refused creation left a network behind")
	}
	n, _ := tu.Must2(e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Open", Slug: "open-net", Policy: &pol,
		CreatorDisplayName: "F", ConfirmOpenGrantRisk: true}))
	var actor, reason string
	if err := e.Store.DB().QueryRow(`SELECT actor_type, reason FROM audit_events WHERE action = ? AND network_id = ?`,
		domain.ActOpenGrantRiskConfirmed, n.ID).Scan(&actor, &reason); err != nil {
		t.Fatalf("confirmation not audited: %v", err)
	}
	if actor != string(domain.ActorAdmin) || reason == "" {
		t.Errorf("confirmation audit: %s %q", actor, reason)
	}
	// open mode without a grant, or a grant without open mode, needs nothing
	pol.SignupGrant = 0
	tu.Must2(e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Open2", Slug: "open-two", Policy: &pol, CreatorDisplayName: "F"}))
	pol.JoinMode, pol.SignupGrant = domain.JoinApproval, 500
	tu.Must2(e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Appr", Slug: "approval-net", Policy: &pol, CreatorDisplayName: "F"}))

	// policy change on an invite network with a grant: switching to open mode
	w := newWorld(t, func(p *domain.Policy) { p.SignupGrant = 100 })
	zero := 0
	change := func(mod func(*domain.Policy), confirm bool) (*domain.Network, error) {
		p := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).Policy
		mod(&p)
		return w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: p, Reason: "test",
			NoticeDays: &zero, NoticeReason: "test network", ConfirmOpenGrantRisk: confirm, IdempotencyKey: w.Key()})
	}
	before := countAction(t, w.Env, domain.ActPolicyChanged)
	_, err = change(func(p *domain.Policy) { p.JoinMode = domain.JoinOpen }, false)
	wantField(t, err, "confirm_open_grant_risk")
	if countAction(t, w.Env, domain.ActPolicyChanged) != before || tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)).Policy.JoinMode != domain.JoinInvite {
		t.Fatal("refused change was applied")
	}
	n2 := tu.Must(change(func(p *domain.Policy) { p.JoinMode = domain.JoinOpen }, true))
	if n2.Policy.JoinMode != domain.JoinOpen || countAction(t, w.Env, domain.ActOpenGrantRiskConfirmed) != 1 {
		t.Fatalf("confirmed change: mode %s, confirmations %d", n2.Policy.JoinMode, countAction(t, w.Env, domain.ActOpenGrantRiskConfirmed))
	}
	// an unrelated change does not ask again; raising the grant does
	tu.Must(change(func(p *domain.Policy) { p.CurrencyName = "hours" }, false))
	_, err = change(func(p *domain.Policy) { p.SignupGrant = 200 }, false)
	wantField(t, err, "confirm_open_grant_risk")
	tu.Must(change(func(p *domain.Policy) { p.SignupGrant = 50 }, false)) // lowering is fine
	if countAction(t, w.Env, domain.ActOpenGrantRiskConfirmed) != 1 {
		t.Error("confirmation recorded for a change that did not need one")
	}
}

// SEC-032 acceptance 3: a grant above the per-network platform cap is refused;
// the default cap is the old policy bound, so nothing changes by default.
func TestSignupGrantCap(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.MaxSignupGrant = 1000 })
	a := e.Person("founder@example.org")
	pol := domain.DefaultPolicy()
	pol.JoinMode, pol.SignupGrant = domain.JoinApproval, 1001
	_, _, err := e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Capped", Slug: "capped", Policy: &pol, CreatorDisplayName: "F"})
	wantField(t, err, "signup_grant")
	pol.SignupGrant = 1000
	n, _ := tu.Must2(e.Svc.CreateNetwork(e.Ctx, a, service.CreateNetworkInput{Name: "Capped", Slug: "capped", Policy: &pol, CreatorDisplayName: "F"}))
	pol.SignupGrant = 1001
	_, err = e.Svc.UpdatePolicy(e.Ctx, a, n.ID, service.PolicyChange{Policy: pol, Reason: "more", IdempotencyKey: e.Key()})
	wantField(t, err, "signup_grant")
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, a, n.ID)); got.PolicyVersion != 1 || got.Policy.SignupGrant != 1000 {
		t.Errorf("refused change applied: v%d grant %d", got.PolicyVersion, got.Policy.SignupGrant)
	}

	// a negative cap (signup-grant-cap=0) forbids every grant
	off := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.MaxSignupGrant = -1 })
	b := off.Person("founder@example.org")
	pol.SignupGrant = 1
	_, _, err = off.Svc.CreateNetwork(off.Ctx, b, service.CreateNetworkInput{Name: "Off", Slug: "no-grants", Policy: &pol, CreatorDisplayName: "F"})
	wantField(t, err, "signup_grant")

	// default: the general policy bound still applies, nothing lower
	d := tu.NewEnv(t)
	c := d.Person("founder@example.org")
	pol.SignupGrant = domain.MaxPolicyAmount
	tu.Must2(d.Svc.CreateNetwork(d.Ctx, c, service.CreateNetworkInput{Name: "Max", Slug: "max-grant", Policy: &pol, CreatorDisplayName: "F"}))
}

// SEC-032 with DOM-027 / DEC-051: once the administrator confirmed the
// open-mode risk, every member who becomes active receives the grant, an open
// self-join included (the behaviour before the SEC-032 controls; whether open
// self-joins should be excluded even after confirmation is an owner question,
// reported as a spec delta).
func TestSignupGrantEligibility(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("founder@example.org")
	pol := domain.DefaultPolicy()
	pol.JoinMode, pol.SignupGrant = domain.JoinOpen, 700
	n, _ := tu.Must2(e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "Open", Slug: "open-net",
		Policy: &pol, CreatorDisplayName: "F", ConfirmOpenGrantRisk: true}))
	n = tu.Must(e.Svc.ActivateNetwork(e.Ctx, admin, n.ID))

	self := e.Person("self@example.org")
	m := tu.Must(e.Svc.JoinNetwork(e.Ctx, self, n.ID, "Self"))
	if m.Status != domain.MemberActive {
		t.Fatalf("open join status %s", m.Status)
	}
	if b := tu.Must(e.Svc.GetBalance(e.Ctx, self, n.ID)); b.Balance != 700 {
		t.Errorf("open self-join balance %d, want the grant 700", b.Balance)
	}
	if grantCount(t, e, n.ID) != 1 {
		t.Errorf("grants after the open self-join = %d, want 1", grantCount(t, e, n.ID))
	}
	invited, _ := e.Member(admin, n, "invited@example.org")
	if b := tu.Must(e.Svc.GetBalance(e.Ctx, invited, n.ID)); b.Balance != 700 {
		t.Errorf("invited member balance %d, want the grant 700", b.Balance)
	}
	if grantCount(t, e, n.ID) != 2 {
		t.Errorf("grants = %d, want 2", grantCount(t, e, n.ID))
	}
	if rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, n.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// SEC-032: the platform cap also bounds the grant when it is posted, so a
// network that stored a grant before the operator lowered signup-grant-cap
// pays at most the cap, and nothing once the cap is 0 (MaxSignupGrant < 0).
// Invitation redemption and an approved join request are both capped; the
// default cap leaves the stored grant as it is.
func TestSignupGrantCapAtPosting(t *testing.T) {
	var cfg service.Config
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { cfg = *c })
	admin := e.Person("founder@example.org")
	n, _ := e.Network(admin, "granting", func(p *domain.Policy) { p.JoinMode, p.SignupGrant = domain.JoinApproval, 500 })
	withCap := func(c int64) *service.Service {
		cc := cfg // same store, clock and id generator
		cc.MaxSignupGrant = c
		return tu.Must(service.New(e.Store, cc))
	}
	join := func(svc *service.Service, login string) domain.Actor {
		a := e.Person(login)
		tok, _ := tu.Must2(svc.CreateInvitation(e.Ctx, admin, n.ID, service.InviteInput{}))
		tu.Must(svc.RedeemInvitation(e.Ctx, a, tok, login))
		return a
	}
	approve := func(svc *service.Service, login string) domain.Actor {
		a := e.Person(login)
		m := tu.Must(svc.JoinNetwork(e.Ctx, a, n.ID, login))
		tu.Must(svc.DecideJoinRequest(e.Ctx, admin, n.ID, m.ID, true, ""))
		return a
	}
	balance := func(a domain.Actor) int64 { return tu.Must(e.Svc.GetBalance(e.Ctx, a, n.ID)).Balance }

	off := withCap(-1) // signup-grant-cap=0
	if b := balance(join(off, "off-invited@example.org")); b != 0 {
		t.Errorf("cap 0, invited: balance %d, want 0", b)
	}
	if b := balance(approve(off, "off-approved@example.org")); b != 0 {
		t.Errorf("cap 0, approved: balance %d, want 0", b)
	}
	if grantCount(t, e, n.ID) != 0 {
		t.Errorf("grants with cap 0 = %d, want 0", grantCount(t, e, n.ID))
	}
	low := withCap(200)
	if b := balance(join(low, "low-invited@example.org")); b != 200 {
		t.Errorf("cap 200, invited: balance %d, want 200", b)
	}
	if b := balance(approve(low, "low-approved@example.org")); b != 200 {
		t.Errorf("cap 200, approved: balance %d, want 200", b)
	}
	var audited int64
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ? AND network_id = ? AND summary_after LIKE '%"amount":200%'`,
		domain.ActGrant, n.ID).Scan(&audited); err != nil || audited != 2 {
		t.Errorf("capped grants audited with amount 200: %d (%v), want 2", audited, err)
	}
	if b := balance(join(e.Svc, "default@example.org")); b != 500 { // default cap: the stored grant
		t.Errorf("default cap: balance %d, want 500", b)
	}
	if grantCount(t, e, n.ID) != 3 {
		t.Errorf("grants = %d, want 3", grantCount(t, e, n.ID))
	}
	if rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, n.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// SEC-006: the per-network aggregate join limit counts invitation redemptions
// and join requests together, is off by default, leaves other networks and
// existing memberships alone and recovers after the window.
func TestNetworkJoinLimit(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.NetworkJoinLimit, c.NetworkJoinWindow = 2, time.Hour })
	admin := e.Person("founder@example.org")
	n, _ := e.Network(admin, "busy", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	other, _ := e.Network(admin, "quiet", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })

	p1, p2, p3 := e.Person("p1@example.org"), e.Person("p2@example.org"), e.Person("p3@example.org")
	tu.Must(e.Svc.JoinNetwork(e.Ctx, p1, n.ID, "P1")) // request (pending)
	tok, _ := tu.Must2(e.Svc.CreateInvitation(e.Ctx, admin, n.ID, service.InviteInput{}))
	tu.Must(e.Svc.RedeemInvitation(e.Ctx, p2, tok, "P2")) // redemption
	_, err := e.Svc.JoinNetwork(e.Ctx, p3, n.ID, "P3")
	wantCode(t, err, domain.ErrRateLimited)
	tok3, _ := tu.Must2(e.Svc.CreateInvitation(e.Ctx, admin, n.ID, service.InviteInput{}))
	_, err = e.Svc.RedeemInvitation(e.Ctx, p3, tok3, "P3")
	wantCode(t, err, domain.ErrRateLimited)
	// the refused redemption did not consume the invitation or create a member
	var members int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM members WHERE network_id = ? AND person_id = ?`, n.ID, p3.PersonID).Scan(&members); err != nil || members != 0 {
		t.Errorf("refused join created a membership (%d, %v)", members, err)
	}
	// an existing membership is returned unchanged, not refused
	if m, err := e.Svc.JoinNetwork(e.Ctx, p1, n.ID, "P1"); err != nil || m.Status != domain.MemberPending {
		t.Errorf("repeat join of an existing member: %v", err)
	}
	// another network is not affected
	tu.Must(e.Svc.JoinNetwork(e.Ctx, p3, other.ID, "P3"))
	// after the window the network accepts members again, with the same invitation
	e.Clock.Advance(61 * time.Minute)
	tu.Must(e.Svc.RedeemInvitation(e.Ctx, p3, tok3, "P3"))

	// off by default
	d := tu.NewEnv(t)
	da := d.Person("founder@example.org")
	dn, _ := d.Network(da, "free", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	for i := 0; i < 5; i++ {
		tu.Must(d.Svc.JoinNetwork(d.Ctx, d.Person("x"+string(rune('a'+i))+"@example.org"), dn.ID, "X"))
	}
}
