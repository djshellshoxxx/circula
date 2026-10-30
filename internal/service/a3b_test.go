package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// dcSybil: dual control on (threshold 1000) in a network whose only admin is the founder.
func dcSybil(t *testing.T) *world {
	t.Helper()
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin)) // the policy needs two administrators
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dc", IdempotencyKey: w.Key()}))
	return w
}

func (w *world) bigRequest(by domain.Actor) *domain.PendingAdjustment {
	w.T.Helper()
	r := tu.Must(w.Svc.CreateAdjustmentRequest(w.Ctx, by, w.net.ID, service.AdjustInput{MemberID: w.bM.ID, Amount: 5_000_000,
		Direction: service.AdjustCredit, Reason: "x", IdempotencyKey: w.Key()}))
	if r.Pending == nil {
		w.T.Fatal("expected a pending request")
	}
	return r.Pending
}

// A3-01: a puppet administrator appointed by the proposer cannot approve, however long it waits.
func TestA3_01_SybilAdminCannotApprove(t *testing.T) {
	w := dcSybil(t)
	pup, pupM := w.Member(w.admin, w.net, "puppet@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, pupM.ID, domain.RoleAdmin))
	pa := w.bigRequest(w.admin)
	w.Clock.Advance(48 * time.Hour)
	_, err := w.Svc.ApproveAdjustment(w.Ctx, pup, w.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.RejectAdjustment(w.Ctx, pup, w.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	if got := w.balance(w.b).Balance; got != 0 {
		t.Fatalf("minted %d with one human", got)
	}
	// A puppet of a puppet is the same human.
	pup2, pup2M := w.Member(w.admin, w.net, "puppet2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, pup, w.net.ID, pup2M.ID, domain.RoleAdmin))
	w.Clock.Advance(48 * time.Hour)
	_, err = w.Svc.ApproveAdjustment(w.Ctx, pup2, w.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	if got := w.balance(w.b).Balance; got != 0 {
		t.Fatalf("minted %d via chained puppet", got)
	}
}

// A3-01: the honest direction works: an administrator appointed by the founder
// proposes and the founder (root of the grant chain) approves.
func TestA3_01_IndependentApproverSucceeds(t *testing.T) {
	w := dcSybil(t)
	adm2, adm2M := w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	pa := w.bigRequest(adm2)
	tx := tu.Must(w.Svc.ApproveAdjustment(w.Ctx, w.admin, w.net.ID, pa.ID))
	if tx.Status != domain.TxCompleted {
		t.Fatal(tx.Status)
	}
}

// A3-01: a newly granted administrator cannot decide for 24 hours (injected clock).
func TestA3_01_NewAdminWaits(t *testing.T) {
	w := dcSybil(t)
	adm2, adm2M := w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	pa := w.bigRequest(w.admin)
	for _, d := range []time.Duration{0, time.Hour, service.DefaultNewAdminApprovalDelay - time.Hour - time.Millisecond} {
		w.Clock.Advance(d)
		_, err := w.Svc.ApproveAdjustment(w.Ctx, adm2, w.net.ID, pa.ID)
		wantCode(t, err, domain.ErrForbidden)
		_, err = w.Svc.RejectAdjustment(w.Ctx, adm2, w.net.ID, pa.ID)
		wantCode(t, err, domain.ErrForbidden)
	}
	w.Clock.Advance(time.Millisecond) // exactly 24h after the grant
	tx := tu.Must(w.Svc.ApproveAdjustment(w.Ctx, adm2, w.net.ID, pa.ID))
	if tx.Status != domain.TxCompleted {
		t.Fatal(tx.Status)
	}
}

// A3-01: the delay is configurable and can be switched off.
func TestA3_01_DelayConfigurable(t *testing.T) {
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 }, func(c *service.Config) { c.NewAdminApprovalDelay = -1 })
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleAdmin))
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "dc", IdempotencyKey: w.Key()}))
	adm2, adm2M := w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	pa := w.bigRequest(w.admin)
	tu.Must(w.Svc.ApproveAdjustment(w.Ctx, adm2, w.net.ID, pa.ID))
}

// A3-01: granting the admin role notifies every other active administrator, not the grantee.
func TestA3_01_AdminGrantNotifiesOtherAdmins(t *testing.T) {
	w := newWorld(t, nil)
	adm2, adm2M := w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin)) // notifies the founder
	adm3, adm3M := w.Member(w.admin, w.net, "admin3@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm3M.ID, domain.RoleAdmin)) // notifies founder and admin2
	count := func(a domain.Actor) int {
		n := 0
		for _, x := range tu.Must(w.Svc.ListSecurityNotices(w.Ctx, a, false)) {
			if x.Kind == domain.NoticeAdminGranted {
				n++
			}
		}
		return n
	}
	if got := count(w.admin); got != 2 {
		t.Errorf("founder notices = %d, want 2", got)
	}
	if got := count(adm2); got != 1 {
		t.Errorf("admin2 notices = %d, want 1", got)
	}
	if got := count(adm3); got != 0 {
		t.Errorf("grantee was notified of their own appointment: %d", got)
	}
	// A reviewer grant is not an administrator appointment.
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleReviewer))
	if got := count(w.admin); got != 2 {
		t.Errorf("reviewer grant produced an admin notice: %d", got)
	}
}

// A3-02: an administrator who is admin only through break-glass cannot make
// lasting changes: close the network, remove/erase/suspend members, flag defaults.
func TestA3_02_BreakGlassAdminCannotMakeLastingChanges(t *testing.T) {
	w := newWorld(t, nil)
	carol, carolM := w.Member(w.admin, w.net, "carol@example.org")
	_ = carol
	tu.Must(w.Svc.DepartMember(w.Ctx, w.b, w.net.ID, service.DepartInput{})) // erasure target
	w.openBG(time.Hour)
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, sys, service.GrantBreakGlassInput{Network: "demo", Login: "ana@example.org", Reason: "support"}))
	// Control: ana really is an administrator for reads.
	tu.Must(w.Svc.AdminListMembers(w.Ctx, w.a, w.net.ID, nil, service.Page{}))

	_, err := w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{MemberID: carolM.ID, Reason: "bg"})
	wantCode(t, err, domain.ErrForbidden)
	wantCode(t, w.Svc.SuspendMember(w.Ctx, w.a, w.net.ID, carolM.ID, "bg"), domain.ErrForbidden)
	wantCode(t, w.Svc.SetDefaultFlag(w.Ctx, w.a, w.net.ID, carolM.ID, true, "bg"), domain.ErrForbidden)
	_, err = w.Svc.AnonymiseMember(w.Ctx, w.a, w.net.ID, w.bM.ID, "bg")
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.ScheduleClosure(w.Ctx, w.a, w.net.ID, service.ClosureInput{Reason: "bg"})
	wantCode(t, err, domain.ErrForbidden)
	wantCode(t, w.Svc.CancelClosure(w.Ctx, w.a, w.net.ID, "bg"), domain.ErrForbidden)
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
	if n.Status != domain.NetworkActive {
		t.Fatalf("network %s", n.Status)
	}
	// Control: a permanent administrator still can.
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, carolM.ID, "ok"))
}

// A3-03: a posting that would push EITHER side, the system account included,
// beyond +-2^62 is refused, and VerifyLedger still answers.
func TestA3_03_SystemAccountBounded(t *testing.T) {
	w := newWorld(t, nil)
	var sysAcc string
	if err := w.Store.DB().QueryRow(`SELECT id FROM accounts WHERE network_id = ? AND is_system = 1`, w.net.ID).Scan(&sysAcc); err != nil {
		t.Fatal(err)
	}
	const bound = int64(1) << 62
	// Credit adjustments make the system account the payer: it goes negative.
	if _, err := w.Store.DB().Exec(`UPDATE accounts SET balance = ? WHERE id = ?`, -(bound - 10), sysAcc); err != nil {
		t.Fatal(err)
	}
	_, err := w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 100,
		Direction: service.AdjustCredit, Reason: "r", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	var got int64
	if err := w.Store.DB().QueryRow(`SELECT balance FROM accounts WHERE id = ?`, sysAcc).Scan(&got); err != nil || got != -(bound-10) {
		t.Fatalf("balance changed by a refused posting: %d %v", got, err)
	}
	// Within the bound it still posts.
	tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 5,
		Direction: service.AdjustCredit, Reason: "r", IdempotencyKey: w.Key()}))
	// The tampered system balance is a finding, not a failure to verify.
	if _, err := w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID); err != nil {
		t.Fatalf("VerifyLedger failed: %v", err)
	}
}

// A3-01: DecidableAdjustments offers a decision exactly where the decision
// paths would accept one, so adapters do not show buttons that are refused.
func TestA3_01_DecidableMatchesDecisionRules(t *testing.T) {
	w := dcSybil(t) // ana (w.a) was made administrator by the founder just now
	adm2, adm2M := w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	sys, sysM := w.Member(w.admin, w.net, "sys@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, domain.SystemActor("fixture"), w.net.ID, sysM.ID, domain.RoleAdmin))
	byFounder := w.bigRequest(w.admin)
	byAdm2 := w.bigRequest(adm2)
	onAna := tu.Must(w.Svc.CreateAdjustmentRequest(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 5_000_000,
		Direction: service.AdjustCredit, Reason: "x", IdempotencyKey: w.Key()})).Pending
	rows := []*domain.PendingAdjustment{byFounder, byAdm2, onAna}
	ids := []string{byFounder.ID, byAdm2.ID, onAna.ID, "no-such-request"}

	check := func(when string, who domain.Actor, want ...*domain.PendingAdjustment) {
		t.Helper()
		got := tu.Must(w.Svc.DecidableAdjustments(w.Ctx, who, w.net.ID, ids))
		wantSet := map[string]bool{}
		for _, pa := range want {
			wantSet[pa.ID] = true
		}
		for _, pa := range rows {
			if got[pa.ID] != wantSet[pa.ID] {
				t.Errorf("%s: %s on %s request %s: got %v", when, who.PersonID, pa.Proposer, pa.ID, got[pa.ID])
			}
		}
		if len(got) != len(want) {
			t.Errorf("%s: offered %d decisions, want %d", when, len(got), len(want))
		}
	}
	// Within 24 h of their grants only the founder (root) may decide, and never
	// their own request.
	check("t0", w.admin, byAdm2)
	check("t0", adm2)
	check("t0", w.a)
	check("t0", sys)
	w.Clock.Advance(service.DefaultNewAdminApprovalDelay)
	// After the wait: founder-appointed administrators still never decide the
	// founder's requests; ana is the target of onAna; the System-granted
	// administrator has no granter in the chain.
	check("24h", w.admin, byAdm2)
	check("24h", adm2)
	check("24h", w.a, byAdm2)
	check("24h", sys, byFounder, byAdm2, onAna)
	if _, err := w.Svc.DecidableAdjustments(w.Ctx, w.b, w.net.ID, ids); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("plain member: %v", err)
	}

	// Every offered decision is accepted; a refused one stays refused.
	_, err := w.Svc.ApproveAdjustment(w.Ctx, adm2, w.net.ID, byFounder.ID)
	wantCode(t, err, domain.ErrForbidden)
	tu.Must(w.Svc.ApproveAdjustment(w.Ctx, w.a, w.net.ID, byAdm2.ID))
	tu.Must(w.Svc.ApproveAdjustment(w.Ctx, sys, w.net.ID, byFounder.ID))
	tu.Must(w.Svc.RejectAdjustment(w.Ctx, sys, w.net.ID, onAna.ID))
	check("decided", sys) // nothing open is left to offer
}
