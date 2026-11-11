package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// dcWorld is a network with dual control on at 1000 and a second administrator.
type dcWorld struct {
	*world
	admin2  domain.Actor
	admin2M *domain.Member
}

func newDC(t *testing.T) *dcWorld {
	t.Helper()
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	d := &dcWorld{world: w}
	d.admin2, d.admin2M = w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, domain.SystemActor("fixture"), w.net.ID, d.admin2M.ID, domain.RoleAdmin))
	w.Clock.Advance(25 * time.Hour) // A3-01: a new administrator waits before deciding
	tu.Must(0, d.setThreshold(1000))
	return d
}

func (d *dcWorld) setThreshold(v int64) error {
	n := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = v
	_, err := d.Svc.UpdatePolicy(d.Ctx, d.admin, d.net.ID, service.PolicyChange{Policy: pol, Reason: "dual control", IdempotencyKey: d.Key()})
	return err
}

func (d *dcWorld) request(by domain.Actor, m *domain.Member, amt int64, dir service.AdjustDirection) (*service.AdjustResult, error) {
	return d.Svc.CreateAdjustmentRequest(d.Ctx, by, d.net.ID, service.AdjustInput{MemberID: m.ID, Amount: amt, Direction: dir,
		Reason: "correction", IdempotencyKey: d.Key()})
}

func (d *dcWorld) pending(amt int64) *domain.PendingAdjustment {
	return tu.Must(d.request(d.admin, d.aM, amt, service.AdjustCredit)).Pending
}

func (d *dcWorld) audits(action string) int {
	n := 0
	evs := tu.Must(d.Svc.ListAudit(d.Ctx, d.admin, d.net.ID, service.AuditQuery{Action: action, Limit: 500}))
	for range evs {
		n++
	}
	return n
}

func TestDualControlBelowAndAtThresholdPostsAtOnce(t *testing.T) {
	d := newDC(t)
	for _, amt := range []int64{500, 1000} { // at the threshold is not above it
		r := tu.Must(d.request(d.admin, d.aM, amt, service.AdjustCredit))
		if r.Tx == nil || r.Pending != nil || r.Tx.Status != domain.TxCompleted {
			t.Fatalf("amount %d: %+v", amt, r)
		}
	}
	if got := d.balance(d.a).Balance; got != 1500 {
		t.Errorf("balance = %d", got)
	}
}

func TestDualControlDisabledByDefault(t *testing.T) {
	w := newWorld(t, nil)
	r := tu.Must(w.Svc.CreateAdjustmentRequest(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 5_000_000,
		Direction: service.AdjustCredit, Reason: "big", IdempotencyKey: w.Key()}))
	if r.Tx == nil {
		t.Fatal("expected immediate posting when threshold is 0")
	}
}

// SEC-011: not posted until a second approver records approval; both identities audited.
func TestDualControlApproveFlow(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	if pa.Status != domain.PendingOpen || pa.Proposer != d.adminM.ID || pa.Target != d.aM.ID {
		t.Fatalf("pending = %+v", pa)
	}
	if want := d.Clock.Now().Add(7 * 24 * time.Hour); !pa.ExpiresAt.Equal(want) {
		t.Errorf("expiry = %v, want %v", pa.ExpiresAt, want)
	}
	if got := d.balance(d.a).Balance; got != 0 {
		t.Fatalf("balance moved before approval: %d", got)
	}
	if d.audits(domain.ActAdjustmentPending) != 1 || d.audits(domain.ActAdjustment) != 0 {
		t.Fatal("audit after request")
	}
	tx := tu.Must(d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID))
	if tx.Kind != domain.KindAdjustment || tx.Status != domain.TxCompleted || tx.Amount != 5000 {
		t.Fatalf("tx = %+v", tx)
	}
	if es := d.entries(tx.ID); len(es) != 2 || es[0].Amount+es[1].Amount != 0 {
		t.Errorf("entries = %+v", es)
	}
	if got := d.balance(d.a).Balance; got != 5000 {
		t.Errorf("balance = %d", got)
	}
	got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
	if got.Status != domain.PendingApproved || got.DecidedBy != d.admin2M.ID || got.TxID != tx.ID || got.DecidedAt == nil {
		t.Errorf("after approve = %+v", got)
	}
	if d.audits(domain.ActAdjustmentApproved) != 1 || d.audits(domain.ActAdjustment) != 1 {
		t.Error("approval audit events missing")
	}
	// Both identities appear in the approval event.
	evs := tu.Must(d.Svc.ListAudit(d.Ctx, d.admin, d.net.ID, service.AuditQuery{Action: domain.ActAdjustmentApproved, Limit: 10}))
	if len(evs) != 1 || evs[0].ActorID != d.admin2M.ID || !strings.Contains(evs[0].SummaryAfter, d.adminM.ID) {
		t.Errorf("approval event = %+v", evs)
	}
	// Replay by the same approver is idempotent: same transaction, nothing more posted.
	again := tu.Must(d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID))
	if again.ID != tx.ID || d.balance(d.a).Balance != 5000 {
		t.Error("replay posted twice")
	}
	if rep := tu.Must(d.Svc.VerifyLedger(d.Ctx, d.admin, d.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

func TestDualControlWrapperFailsSafe(t *testing.T) {
	d := newDC(t)
	_, err := d.adjust(d.admin, d.aM, 5000, service.AdjustCredit, "big")
	wantCode(t, err, domain.ErrApprovalRequired)
	if d.balance(d.a).Balance != 0 {
		t.Error("wrapper posted")
	}
	if rows := tu.Must(d.Svc.ListPendingAdjustments(d.Ctx, d.admin, d.net.ID, nil, service.Page{})); len(rows) != 0 {
		t.Error("wrapper left a pending row")
	}
}

func TestDualControlProposerCannotApprove(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = d.Svc.RejectAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	if d.balance(d.a).Balance != 0 {
		t.Error("self approval posted")
	}
}

func TestDualControlTargetAdminCannotApprove(t *testing.T) {
	d := newDC(t)
	// Request targets admin2; the only other administrator is the proposer, so
	// admin2 (the target) must be refused (DOM-037).
	pa := tu.Must(d.request(d.admin, d.admin2M, 5000, service.AdjustCredit)).Pending
	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = d.Svc.RejectAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	// Self-targeting cannot even be proposed.
	_, err = d.request(d.admin, d.adminM, 5000, service.AdjustCredit)
	wantCode(t, err, domain.ErrForbidden)
}

func TestDualControlNonAdminAndCrossNetwork(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.b, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = d.Svc.CancelAdjustment(d.Ctx, d.b, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrForbidden)
	_, err = d.Svc.ListPendingAdjustments(d.Ctx, d.b, d.net.ID, nil, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	// An administrator of another network sees not-found, and cannot use this network's id there.
	otherAdmin := d.Person("other@example.org")
	other, _ := d.Network(otherAdmin, "other", nil)
	_, err = d.Svc.ApproveAdjustment(d.Ctx, otherAdmin, other.ID, pa.ID)
	wantCode(t, err, domain.ErrNotFound)
	_, err = d.Svc.ApproveAdjustment(d.Ctx, otherAdmin, d.net.ID, pa.ID)
	wantCode(t, err, domain.ErrNotFound)
	_, err = d.Svc.GetPendingAdjustment(d.Ctx, otherAdmin, other.ID, pa.ID)
	wantCode(t, err, domain.ErrNotFound)
	if d.balance(d.a).Balance != 0 {
		t.Error("posted")
	}
}

func TestDualControlRejectAndCancel(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	r := tu.Must(d.Svc.RejectAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID))
	if r.Status != domain.PendingRejected || r.DecidedBy != d.admin2M.ID {
		t.Fatalf("reject = %+v", r)
	}
	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID) // final
	wantCode(t, err, domain.ErrInvalidState)
	tu.Must(d.Svc.RejectAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)) // replay by same decider is fine
	if d.audits(domain.ActAdjustmentRejected) != 1 {
		t.Error("reject audited more than once")
	}

	pb := d.pending(5000)
	_, err = d.Svc.CancelAdjustment(d.Ctx, d.admin2, d.net.ID, pb.ID) // only the proposer cancels
	wantCode(t, err, domain.ErrForbidden)
	c := tu.Must(d.Svc.CancelAdjustment(d.Ctx, d.admin, d.net.ID, pb.ID))
	if c.Status != domain.PendingCancelled {
		t.Fatalf("cancel = %+v", c)
	}
	_, err = d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pb.ID)
	wantCode(t, err, domain.ErrInvalidState)
	if d.balance(d.a).Balance != 0 || d.audits(domain.ActAdjustmentCancelled) != 1 {
		t.Error("cancel leaked")
	}
}

// SEC-011: an unapproved request past expiry is closed with an audit event and posts nothing.
func TestDualControlExpiry(t *testing.T) {
	d := newDC(t)
	pa := d.pending(5000)
	d.Clock.Advance(7*24*time.Hour + time.Second)
	_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID) // lazy close
	wantCode(t, err, domain.ErrInvalidState)
	got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
	if got.Status != domain.PendingExpired || d.audits(domain.ActAdjustmentExpired) != 1 {
		t.Fatalf("lazy expiry: %+v", got)
	}
	if d.balance(d.a).Balance != 0 {
		t.Error("expired request posted")
	}
	// Job path: another request, expired by the job, once.
	pb := tu.Must(d.request(d.admin, d.bM, 5000, service.AdjustDebit)).Pending
	d.Clock.Advance(8 * 24 * time.Hour)
	if _, err := d.Svc.ExpirePendingAdjustments(d.Ctx, d.admin, 10); err == nil {
		t.Error("non-system actor ran the expiry job")
	}
	sys := domain.SystemActor("job-expiry")
	if n := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, sys, 10)); n != 1 {
		t.Errorf("expired %d, want 1", n)
	}
	if n := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, sys, 10)); n != 0 {
		t.Errorf("second run expired %d", n)
	}
	if got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pb.ID)); got.Status != domain.PendingExpired {
		t.Errorf("job expiry: %+v", got)
	}
	_, err = d.Svc.CancelAdjustment(d.Ctx, d.admin, d.net.ID, pb.ID)
	wantCode(t, err, domain.ErrInvalidState)
	if d.audits(domain.ActAdjustmentExpired) != 2 {
		t.Error("expiry events")
	}
}

func TestDualControlConfigurableTTL(t *testing.T) {
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 }, func(c *service.Config) { c.PendingAdjustmentTTL = time.Hour })
	d := &dcWorld{world: w}
	d.admin2, d.admin2M = w.Member(w.admin, w.net, "admin2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, domain.SystemActor("fixture"), w.net.ID, d.admin2M.ID, domain.RoleAdmin))
	w.Clock.Advance(25 * time.Hour) // A3-01: a new administrator waits before deciding
	tu.Must(0, d.setThreshold(1000))
	pa := d.pending(5000)
	if !pa.ExpiresAt.Equal(w.Clock.Now().Add(time.Hour)) {
		t.Errorf("expires %v", pa.ExpiresAt)
	}
}

func TestDualControlRevalidatesAtApproval(t *testing.T) {
	t.Run("target departed", func(t *testing.T) {
		d := newDC(t)
		pa := d.pending(5000)
		tu.Must(d.Svc.DepartMember(d.Ctx, d.a, d.net.ID, service.DepartInput{}))
		_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
		wantCode(t, err, domain.ErrInvalidState)
		if got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID)); got.Status != domain.PendingOpen {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("limit breached by debit", func(t *testing.T) {
		d := newDC(t)
		pa := tu.Must(d.request(d.admin, d.aM, 500000, service.AdjustDebit)).Pending // limit is 100000
		_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
		wantCode(t, err, domain.ErrInsufficientCredit)
		if d.balance(d.a).Balance != 0 {
			t.Error("posted")
		}
	})
	t.Run("proposer lost admin role", func(t *testing.T) {
		d := newDC(t)
		pa := d.pending(5000)
		tu.Must(0, d.Svc.RevokeRole(d.Ctx, d.admin2, d.net.ID, d.adminM.ID, domain.RoleAdmin))
		// admin2 is now the only administrator and the proposer is not one.
		_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
		wantCode(t, err, domain.ErrInvalidState)
	})
	t.Run("approver lost admin role", func(t *testing.T) {
		d := newDC(t)
		pa := d.pending(5000)
		tu.Must(0, d.Svc.RevokeRole(d.Ctx, d.admin, d.net.ID, d.admin2M.ID, domain.RoleAdmin))
		_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
		wantCode(t, err, domain.ErrForbidden)
	})
	t.Run("network closed", func(t *testing.T) {
		d := newDC(t)
		pa := d.pending(5000)
		tu.Must(d.Svc.ScheduleClosure(d.Ctx, d.admin, d.net.ID, service.ClosureInput{Reason: "done"}))
		d.Clock.Advance(31 * 24 * time.Hour)
		_, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin2, d.net.ID, pa.ID)
		if err == nil {
			t.Error("approved on a closed network")
		}
	})
}

// Concurrent double-approve: exactly one posting.
func TestDualControlConcurrentApprove(t *testing.T) {
	d := newDC(t)
	third, thirdM := d.Member(d.admin, d.net, "admin3@example.org")
	tu.Must(0, d.Svc.AssignRole(d.Ctx, domain.SystemActor("fixture"), d.net.ID, thirdM.ID, domain.RoleAdmin))
	d.Clock.Advance(25 * time.Hour) // A3-01
	pa := d.pending(5000)
	var wg sync.WaitGroup
	var mu sync.Mutex
	okN := 0
	for _, who := range []domain.Actor{d.admin2, third, d.admin2, third} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := d.Svc.ApproveAdjustment(context.Background(), who, d.net.ID, pa.ID); err == nil {
				mu.Lock()
				okN++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	// The winner and its own replay may both succeed; the other administrator must not.
	if got := d.balance(d.a).Balance; got != 5000 {
		t.Fatalf("balance = %d, want exactly one posting of 5000", got)
	}
	if d.audits(domain.ActAdjustment) != 1 || d.audits(domain.ActAdjustmentApproved) != 1 {
		t.Error("more than one posting audited")
	}
	if okN < 1 {
		t.Error("nobody succeeded")
	}
	if rep := tu.Must(d.Svc.VerifyLedger(d.Ctx, d.admin, d.net.ID)); !rep.OK {
		t.Error(rep.Findings)
	}
}

func TestDualControlIdempotentRequest(t *testing.T) {
	d := newDC(t)
	in := service.AdjustInput{MemberID: d.aM.ID, Amount: 5000, Direction: service.AdjustCredit, Reason: "x", IdempotencyKey: "same-key"}
	r1 := tu.Must(d.Svc.CreateAdjustmentRequest(d.Ctx, d.admin, d.net.ID, in))
	r2 := tu.Must(d.Svc.CreateAdjustmentRequest(d.Ctx, d.admin, d.net.ID, in))
	if r1.Pending == nil || r2.Pending == nil || r1.Pending.ID != r2.Pending.ID {
		t.Fatal("retry created a second request")
	}
	in.Amount = 6000
	_, err := d.Svc.CreateAdjustmentRequest(d.Ctx, d.admin, d.net.ID, in)
	wantCode(t, err, domain.ErrKeyReused)
}

func TestDualControlPolicyValidation(t *testing.T) {
	w := newWorld(t, nil) // single administrator
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	pol.DualControlThreshold = 1000
	_, err := w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "try", IdempotencyKey: w.Key()})
	var de *domain.Error
	if !errors.As(err, &de) || de.Code != domain.CodeValidation || len(de.Fields) != 1 || de.Fields[0].Field != "dual_control_threshold" {
		t.Fatalf("single admin enabling: %v", err)
	}
	pol.DualControlThreshold = -1
	_, err = w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "try", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	// With two administrators it can be tightened; only the operator disables it (SA2-01). Each change is audited.
	d := newDC(t)
	tu.Must(0, d.setThreshold(500))
	tu.Must(0, d.Svc.OperatorSetDualControl(d.Ctx, domain.SystemActor("t"), d.net.Slug, 0, "wind down"))
	if d.audits(domain.ActPolicyChanged) < 3 {
		t.Error("policy changes not audited")
	}
	got := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
	if got.Policy.DualControlThreshold != 0 {
		t.Errorf("threshold = %d", got.Policy.DualControlThreshold)
	}
}

// With dual control on but no second administrator left, requests are refused
// rather than parked where nobody can decide them.
func TestDualControlLoneAdminRefused(t *testing.T) {
	d := newDC(t)
	tu.Must(0, d.Svc.RevokeRole(d.Ctx, d.admin, d.net.ID, d.admin2M.ID, domain.RoleAdmin))
	_, err := d.request(d.admin, d.aM, 5000, service.AdjustCredit)
	wantCode(t, err, domain.ErrInvalidState)
	r := tu.Must(d.request(d.admin, d.aM, 900, service.AdjustCredit)) // below threshold still works
	if r.Tx == nil {
		t.Error("below-threshold adjustment should post")
	}
}

func TestDualControlListFilters(t *testing.T) {
	d := newDC(t)
	p1, p2 := d.pending(2000), d.pending(3000)
	tu.Must(d.Svc.RejectAdjustment(d.Ctx, d.admin2, d.net.ID, p1.ID))
	open := tu.Must(d.Svc.ListPendingAdjustments(d.Ctx, d.admin2, d.net.ID, []domain.PendingStatus{domain.PendingOpen}, service.Page{Limit: 10}))
	if len(open) != 1 || open[0].ID != p2.ID {
		t.Errorf("open = %+v", open)
	}
	all := tu.Must(d.Svc.ListPendingAdjustments(d.Ctx, d.admin2, d.net.ID, nil, service.Page{Limit: 10}))
	if len(all) != 2 {
		t.Errorf("all = %d", len(all))
	}
}
