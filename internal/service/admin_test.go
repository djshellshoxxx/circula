package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) adjust(by domain.Actor, m *domain.Member, amt int64, dir service.AdjustDirection, reason string) (*domain.Transaction, error) {
	return w.Svc.CreateAdjustment(w.Ctx, by, w.net.ID, service.AdjustInput{MemberID: m.ID, Amount: amt, Direction: dir, Reason: reason, IdempotencyKey: w.Key()})
}

// DOM-022 / FR-062.
func TestAdjustments(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1000 })
	adj := tu.Must(w.adjust(w.admin, w.aM, 400, service.AdjustCredit, "community bonus"))
	if adj.Kind != domain.KindAdjustment || adj.Status != domain.TxCompleted || adj.Reason == "" {
		t.Fatalf("adjustment = %+v", adj)
	}
	if es := w.entries(adj.ID); len(es) != 2 || es[0].Amount+es[1].Amount != 0 {
		t.Errorf("entries = %+v", es)
	}
	if w.balance(w.a).Balance != 400 {
		t.Errorf("balance = %d", w.balance(w.a).Balance)
	}
	tu.Must(w.adjust(w.admin, w.aM, 1400, service.AdjustDebit, "correction")) // 400 - 1400 = -1000 = limit: allowed
	if w.balance(w.a).Balance != -1000 {
		t.Errorf("balance = %d", w.balance(w.a).Balance)
	}
	_, err := w.adjust(w.admin, w.aM, 1, service.AdjustDebit, "one too many") // LED-5
	wantCode(t, err, domain.ErrInsufficientCredit)
	_, err = w.adjust(w.admin, w.aM, 5, service.AdjustCredit, "")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.adjust(w.admin, w.aM, 0, service.AdjustCredit, "zero")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.adjust(w.admin, w.aM, -5, service.AdjustCredit, "negative")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.adjust(w.admin, w.adminM, 5, service.AdjustCredit, "self-dealing") // DOM-037
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.adjust(w.a, w.bM, 5, service.AdjustCredit, "members cannot adjust")
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: "ghost", Amount: 1, Direction: service.AdjustCredit, Reason: "x", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrNotFound)
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

// DOM-027 grants: posted from the system account on activation, two entries each.
func TestSignupGrant(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.SignupGrant = 1500; p.JoinMode = domain.JoinApproval })
	// newWorld joined a and b by invitation: each got a grant. The founder did not.
	if got := w.balance(w.a).Balance; got != 1500 {
		t.Errorf("grant balance = %d", got)
	}
	if got := w.balance(w.admin).Balance; got != 0 {
		t.Errorf("founder balance = %d", got)
	}
	// a pending member has no grant until approved
	_, pM := w.joinPending("p@example.org")
	var n int
	_ = w.Store.DB().QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind='grant'`).Scan(&n)
	if n != 2 {
		t.Fatalf("grants before approval = %d, want 2", n)
	}
	tu.Must(w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, true, ""))
	_ = w.Store.DB().QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind='grant'`).Scan(&n)
	if n != 3 {
		t.Errorf("grants after approval = %d, want 3", n)
	}
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM entries e JOIN transactions t ON t.id = e.tx_id WHERE t.kind = 'grant'`).Scan(&n); err != nil || n != 6 {
		t.Errorf("grant entries = %d (want 2 per grant)", n)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
	// disabled: no grant
	w2 := newWorld(t, nil)
	if w2.balance(w2.a).Balance != 0 || countRows(t, w2.Env, "entries") != 0 {
		t.Error("no grant expected when disabled")
	}
}

// DOM-024 / FR-014 / FR-015.
func TestDepartureRetain(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	open := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 50)) // pending proposal involving A
	got := tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
	if got.Status != domain.MemberDeparted || got.DepartedAt == nil {
		t.Fatalf("member = %+v", got)
	}
	if w.txStatus(open.ID).Status != domain.TxCancelled {
		t.Errorf("pending proposal status = %s", w.txStatus(open.ID).Status)
	}
	// balance retained on the closed account; history and export remain readable
	bv := w.balance(w.a)
	if bv.Balance != -1000 || bv.AccountState != domain.AccountClosed {
		t.Errorf("balance view = %+v", bv)
	}
	if h := tu.Must(w.Svc.ListTransactions(w.Ctx, w.a, w.net.ID, nil, service.Page{})); len(h) == 0 {
		t.Error("departed member lost history")
	}
	sink := &service.MemSink{}
	tu.Must(w.Svc.Export(w.Ctx, w.a, w.net.ID, service.ExportMine, service.FormatCSV, sink))
	// no new activity
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.propose(w.b, w.aM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrCounterpartyInact)
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

func TestDepartureSettleToZero(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DepartureRule = domain.DepartSettleToZero })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	before := countRows(t, w.Env, "entries")
	tu.Must(w.Svc.DepartMember(w.Ctx, w.b, w.net.ID, service.DepartInput{}))
	if got := w.balance(w.b).Balance; got != 0 {
		t.Errorf("settled balance = %d", got)
	}
	if got := countRows(t, w.Env, "entries") - before; got != 2 {
		t.Errorf("settlement produced %d entries, want exactly 2", got)
	}
	var n int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind='adjustment' AND reason='departure settlement'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("settlement adjustments = %d", n)
	}
	// negative balance settles upward
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
	if got := w.balance(w.a).Balance; got != 0 {
		t.Errorf("A settled balance = %d", got)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

func TestDepartureBlockersAndAdminOffboarding(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	tu.Must(w.open(w.a, tx.ID))
	_, err := w.Svc.DepartMember(w.Ctx, w.b, w.net.ID, service.DepartInput{})
	wantCode(t, err, domain.ErrBlocked)
	var de *domain.Error
	if !asDomainErr(err, &de) || len(de.Blockers) == 0 {
		t.Errorf("blocker list missing: %v", err)
	}
	// last administrator cannot leave or be offboarded
	_, err = w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{})
	wantCode(t, err, domain.ErrBlocked)
	// administrative offboarding needs admin + reason
	c, cM := w.Member2(w.admin, "carol@example.org")
	_, err = w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{MemberID: w.bM.ID, Reason: "x"})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: cM.ID})
	wantCode(t, err, domain.ErrValidation)
	got := tu.Must(w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: cM.ID, Reason: "left town"}))
	if got.Status != domain.MemberDeparted {
		t.Error("offboarding failed")
	}
	_, err = w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{}) // already departed
	wantCode(t, err, domain.ErrInvalidState)
}

func TestDepartedAdminLosesRole(t *testing.T) {
	w := newWorld(t, nil)
	adm2, m2 := w.Member2(w.admin, "adm2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin))
	tu.Must(w.Svc.DepartMember(w.Ctx, adm2, w.net.ID, service.DepartInput{}))
	got := tu.Must(w.Svc.AdminListMembers(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberDeparted}, service.Page{}))
	if len(got) != 1 || len(got[0].Roles) != 0 {
		t.Errorf("departed member roles = %+v", got)
	}
	if err := w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, m2.ID, domain.RoleAdmin); err == nil {
		t.Error("departed member must not receive roles")
	}
}

// DOM-031 / DOM-030 / FR-003.
func TestPolicyChanges(t *testing.T) {
	w := newWorld(t, nil)
	pol := w.net.Policy
	pol.Decimals = 3
	pc := service.PolicyChange{Policy: pol, Reason: "finer units", IdempotencyKey: w.Key()}
	n := tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, pc))
	if n.Policy.Decimals != 3 || n.PolicyVersion != 2 {
		t.Fatalf("policy = %+v v%d", n.Policy, n.PolicyVersion)
	}
	// prospective: an in-flight proposal keeps its expiry; the dispute window applies to later decisions
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	exp := *p.ExpiresAt
	pol.ProposalExpiryDays = 30
	pol.Decimals = 3
	tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "slower", IdempotencyKey: w.Key()}))
	if got := w.txStatus(p.ID); !got.ExpiresAt.Equal(exp) {
		t.Error("policy change altered an existing proposal's expiry")
	}
	tu.Must(w.accept(w.b, p.ID))
	// decimals frozen after the first posting
	pol.Decimals = 2
	_, err := w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "back", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	// bounds + reason + admin only
	pol.Decimals = 3
	pol.DisputeWindowDays = 0
	_, err = w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "x", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	pol.DisputeWindowDays = 10
	_, err = w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.UpdatePolicy(w.Ctx, w.a, w.net.ID, service.PolicyChange{Policy: pol, Reason: "x", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden)
	// the audit event records old and new values
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActPolicyChanged}))
	if len(evs) != 2 || evs[0].SummaryBefore == "" || evs[0].SummaryAfter == "" || evs[0].Reason == "" {
		t.Errorf("policy audit events = %+v", evs)
	}
}

// FR-004: a network in setup accepts no members or trades; FR-068 closure.
func TestNetworkLifecycle(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit = 1000
	n, m, err := e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "N", Slug: "life", Policy: &pol, CreatorDisplayName: "Founder"})
	if err != nil || n.Status != domain.NetworkSetup || m.Status != domain.MemberActive {
		t.Fatalf("create: %+v %+v %v", n, m, err)
	}
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "N", Slug: "life", CreatorDisplayName: "F"}); err == nil {
		t.Error("duplicate slug must be refused")
	}
	for _, bad := range []string{"A", "ab", "has space", "UPPER", "-lead-", ""} {
		if _, _, err := e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "N", Slug: bad, CreatorDisplayName: "F"}); err == nil {
			t.Errorf("slug %q accepted", bad)
		}
	}
	var sys int
	_ = e.Store.DB().QueryRow(`SELECT COUNT(*) FROM accounts WHERE network_id=? AND is_system=1`, n.ID).Scan(&sys)
	if sys != 1 || countRows(t, e, "entries") != 0 {
		t.Errorf("system accounts=%d entries=%d", sys, countRows(t, e, "entries"))
	}
	tok, _ := tu.Must2(e.Svc.CreateInvitation(e.Ctx, admin, n.ID, service.InviteInput{}))
	_, err = e.Svc.RedeemInvitation(e.Ctx, e.Person("p@example.org"), tok, "P")
	wantCode(t, err, domain.ErrInvalidState)
	n = tu.Must(e.Svc.ActivateNetwork(e.Ctx, admin, n.ID))
	if n.Status != domain.NetworkActive {
		t.Fatal("not active")
	}
	_, err = e.Svc.ActivateNetwork(e.Ctx, admin, n.ID)
	wantCode(t, err, domain.ErrInvalidState)

	pa := e.Person("pa@example.org")
	pm := tu.Must(e.Svc.RedeemInvitation(e.Ctx, pa, tok, "P"))
	prop := tu.Must(e.Svc.ProposeTransaction(e.Ctx, admin, n.ID, service.ProposeInput{CounterpartyID: pm.ID, Direction: service.ProposerIsPaid, Amount: 5, IdempotencyKey: "k"}))
	// GOV-025: closing is announced with notice; a proposal does not block it, it is cancelled at the date.
	_, err = e.Svc.ScheduleClosure(e.Ctx, admin, n.ID, service.ClosureInput{Reason: "winding up", NoticeDays: 29})
	wantCode(t, err, domain.ErrValidation)
	tu.Must(e.Svc.ScheduleClosure(e.Ctx, admin, n.ID, service.ClosureInput{Reason: "winding up"}))
	e.Clock.Advance(31 * 24 * time.Hour)
	_, err = e.Svc.ProposeTransaction(e.Ctx, admin, n.ID, service.ProposeInput{CounterpartyID: pm.ID, Direction: service.ProposerIsPaid, Amount: 5, IdempotencyKey: "k3"})
	wantCode(t, err, domain.ErrInvalidState)
	tu.Must(e.Svc.FinaliseClosures(e.Ctx, domain.SystemActor("job")))
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.Status != domain.NetworkClosed {
		t.Fatalf("status = %s", got.Status)
	}
	_ = prop
	// history and exports stay available on a closed network (FR-068, FR-094)
	tu.Must(e.Svc.ListTransactions(e.Ctx, pa, n.ID, nil, service.Page{}))
	tu.Must(e.Svc.Export(e.Ctx, admin, n.ID, service.ExportFull, service.FormatJSON, &service.MemSink{}))
}

func asDomainErr(err error, out **domain.Error) bool {
	e, ok := err.(*domain.Error)
	if ok {
		*out = e
	}
	return ok
}
