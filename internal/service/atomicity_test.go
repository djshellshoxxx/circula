package service_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func countRows(t *testing.T, e *tu.Env, table string) int {
	t.Helper()
	var n int
	if err := e.Store.DB().QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// LED-4 / LED-6 / DOM-015 / DOM-035: a failure at any step of the posting unit,
// including between the state change and the audit write, leaves no entries, no
// balance change, no status change and no audit event.
func TestAcceptFaultInjectionLeavesNoTrace(t *testing.T) {
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "fault", nil)
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	p := tu.Must(e.Svc.ProposeTransaction(e.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: bM.ID,
		Direction: service.ProposerPays, Amount: 1000, IdempotencyKey: e.Key()}))

	steps := []struct {
		op  string
		nth int
	}{{"InsertEntry", 1}, {"InsertEntry", 2}, {"AddBalance", 1}, {"AddBalance", 2}, {"UpdateTransaction", 1},
		{"InsertStatusEvent", 1}, {"InsertAudit", 1}, {"PutIdem", 1}}
	for _, st := range steps {
		t.Run(fmt.Sprintf("%s#%d", st.op, st.nth), func(t *testing.T) {
			auditBefore, idemBefore := countRows(t, e, "audit_events"), countRows(t, e, "idempotency_keys")
			fs.Arm(st.op, st.nth)
			_, err := e.Svc.AcceptTransaction(e.Ctx, b, net.ID, service.TxRef{TxID: p.ID, IdempotencyKey: "acc"})
			fs.Disarm()
			if !errors.Is(err, tu.ErrInjected) {
				t.Fatalf("want injected fault, got %v", err)
			}
			if got := countRows(t, e, "entries"); got != 0 {
				t.Errorf("%d entries leaked", got)
			}
			if got := countRows(t, e, "audit_events"); got != auditBefore {
				t.Errorf("audit events %d -> %d", auditBefore, got)
			}
			if got := countRows(t, e, "idempotency_keys"); got != idemBefore {
				t.Errorf("idempotency record leaked")
			}
			v := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, p.ID))
			if v.Tx.Status != domain.TxProposed || v.Tx.LedgerSeq != 0 {
				t.Errorf("status = %s seq=%d", v.Tx.Status, v.Tx.LedgerSeq)
			}
			if bal := tu.Must(e.Svc.GetBalance(e.Ctx, a, net.ID)); bal.Balance != 0 {
				t.Errorf("balance moved: %d", bal.Balance)
			}
			if rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, net.ID)); !rep.OK {
				t.Errorf("ledger: %v", rep.Findings)
			}
		})
	}
	fs.Disarm()
	// With no fault the very same request, same key, now succeeds exactly once.
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, b, net.ID, service.TxRef{TxID: p.ID, IdempotencyKey: "acc"}))
	if got := tu.Must(e.Svc.GetBalance(e.Ctx, a, net.ID)).Balance; got != -1000 {
		t.Errorf("balance %d", got)
	}
	_ = aM
}

// FR-080: a sweep over every state-changing command finds an audit event for
// each, and an audit failure rolls back the change itself.
func TestEveryCommandWritesAuditEvent(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	c, cM := w.joinPending("carol@example.org")
	last := func() int { return countRows(t, w.Env, "audit_events") }

	steps := []struct {
		name   string
		action string
		run    func() error
	}{
		{"approve", domain.ActMemberApproved, func() error { _, err := w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, cM.ID, true, ""); return err }},
		{"suspend", domain.ActMemberSuspended, func() error { return w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, cM.ID, "r") }},
		{"restore", domain.ActMemberRestored, func() error { return w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, cM.ID, "r") }},
		{"limit", domain.ActLimitChanged, func() error {
			l := int64(5)
			return w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: cM.ID, Limit: &l, Reason: "r"})
		}},
		{"role", domain.ActRoleGranted, func() error { return w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, cM.ID, domain.RoleReviewer) }},
		{"role-revoke", domain.ActRoleRevoked, func() error { return w.Svc.RevokeRole(w.Ctx, w.admin, w.net.ID, cM.ID, domain.RoleReviewer) }},
		{"flag", domain.ActDefaultFlagSet, func() error { return w.Svc.SetDefaultFlag(w.Ctx, w.admin, w.net.ID, cM.ID, true, "r") }},
		{"propose", domain.ActTxProposed, func() error {
			_, err := w.propose(w.a, cM, service.ProposerPays, 5)
			return err
		}},
		{"adjust", domain.ActAdjustment, func() error {
			_, err := w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: cM.ID, Amount: 5, Direction: service.AdjustCredit, Reason: "r", IdempotencyKey: w.Key()})
			return err
		}},
		{"listing", domain.ActListingCreated, func() error {
			_, err := w.Svc.CreateListing(w.Ctx, c, w.net.ID, service.ListingInput{Type: domain.ListingRequest, Title: "t", IdempotencyKey: w.Key()})
			return err
		}},
		{"invite", domain.ActInviteCreated, func() error {
			_, _, err := w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{})
			return err
		}},
		{"policy", domain.ActPolicyChanged, func() error {
			p := w.net.Policy
			p.ProposalExpiryDays = 3
			_, err := w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: p, Reason: "r", IdempotencyKey: w.Key()})
			return err
		}},
		{"depart", domain.ActMemberDeparted, func() error { _, err := w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{}); return err }},
	}
	for _, s := range steps {
		before := last()
		if err := s.run(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		var n int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, s.action).Scan(&n); err != nil || n == 0 {
			t.Errorf("%s: no %q audit event (err=%v)", s.name, s.action, err)
		}
		if last() <= before {
			t.Errorf("%s: audit count did not grow", s.name)
		}
	}
}

func (w *world) joinPending(login string) (domain.Actor, *domain.Member) {
	a := w.Person(login)
	m := tu.Must(w.Svc.JoinNetwork(w.Ctx, a, w.net.ID, login))
	if m.Status != domain.MemberPending {
		w.T.Fatalf("status %s", m.Status)
	}
	return a, m
}

// An audit write failure undoes a suspend (state change and audit are one unit).
func TestAuditFailureRollsBackStateChange(t *testing.T) {
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "fault2", nil)
	_, m := e.Member(admin, net, "m@example.org")
	fs.Arm("InsertAudit", 1)
	err := e.Svc.SuspendMember(e.Ctx, admin, net.ID, m.ID, "x")
	if !errors.Is(err, tu.ErrInjected) {
		t.Fatalf("got %v", err)
	}
	fs.Disarm()
	list := tu.Must(e.Svc.AdminListMembers(e.Ctx, admin, net.ID, []domain.MemberStatus{domain.MemberActive}, service.Page{}))
	found := false
	for _, x := range list {
		found = found || x.ID == m.ID
	}
	if !found {
		t.Error("member was suspended although the audit write failed")
	}
}
