package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func auditActions(w *world) []string {
	var out []string
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Limit: 200})) {
		out = append(out, e.Action)
	}
	return out
}

// GOV-023: a suspended network refuses new postings, keeps reads, notifies
// members, and resumption is audited.
func TestAdminSuspensionRefusesPostingsKeepsReads(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	open := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))

	if err := w.Svc.SuspendNetwork(w.Ctx, w.a, w.net.ID, "pause"); err == nil {
		t.Fatal("non-admin suspended the network")
	}
	if err := w.Svc.SuspendNetwork(w.Ctx, w.admin, w.net.ID, " "); err == nil {
		t.Fatal("suspension without a reason")
	}
	noErr(t, w.Svc.SuspendNetwork(w.Ctx, w.admin, w.net.ID, "security review"))
	wantCode(t, w.Svc.SuspendNetwork(w.Ctx, w.admin, w.net.ID, "again"), domain.ErrInvalidState)

	_, err := w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: open.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrInvalidState)

	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("balance read during suspension: %v", err)
	}
	if n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)); !n.Suspended() {
		t.Fatal("network not reported as suspended")
	}
	if !hasKind(w.inbox(w.a), service.NotifNetworkSuspended) {
		t.Fatalf("member not notified: %v", w.inbox(w.a))
	}

	noErr(t, w.Svc.ResumeNetwork(w.Ctx, w.admin, w.net.ID, "review done"))
	wantCode(t, w.Svc.ResumeNetwork(w.Ctx, w.admin, w.net.ID, "again"), domain.ErrInvalidState)
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 100); err != nil {
		t.Fatalf("after resume: %v", err)
	}
	acts := auditActions(w)
	if !hasKind(acts, domain.ActNetworkSuspended) || !hasKind(acts, domain.ActNetworkResumed) {
		t.Fatalf("audit lacks suspend/resume: %v", acts)
	}
	if !hasKind(w.inbox(w.a), service.NotifNetworkResumed) {
		t.Fatal("member not told trading resumed")
	}
}

// GOV-024: platform suspension needs a documented category, notifies the
// administrators, is visible in the network's audit, keeps reads, and only the
// operator can lift it.
func TestPlatformSuspension(t *testing.T) {
	w := newWorld(t, nil)
	in := service.PlatformSuspendInput{Network: "demo", Operator: "ops", Reason: "court order 12"}
	if err := w.Svc.PlatformSuspendNetwork(w.Ctx, sys, in); err == nil {
		t.Fatal("suspension without a category accepted")
	}
	in.Category = "members traded too much"
	if err := w.Svc.PlatformSuspendNetwork(w.Ctx, sys, in); err == nil {
		t.Fatal("trade-related category accepted")
	}
	in.Category = domain.SuspendLegalOrder
	if err := w.Svc.PlatformSuspendNetwork(w.Ctx, w.admin, in); err == nil {
		t.Fatal("a network administrator acted as operator")
	}
	noErr(t, w.Svc.PlatformSuspendNetwork(w.Ctx, sys, in))
	wantCode(t, w.Svc.PlatformSuspendNetwork(w.Ctx, sys, in), domain.ErrInvalidState)

	_, err := w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState)
	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("read during platform suspension: %v", err)
	}
	if !hasKind(w.inbox(w.admin), service.NotifNetworkSuspended) {
		t.Fatal("administrator not notified")
	}
	if !hasKind(auditActions(w), domain.ActNetworkPlatformSuspended) {
		t.Fatal("platform suspension not visible in the network's audit")
	}
	// the administrator cannot lift it
	wantCode(t, w.Svc.ResumeNetwork(w.Ctx, w.admin, w.net.ID, "no"), domain.ErrInvalidState)
	// an administrator suspension on top is independent
	noErr(t, w.Svc.SuspendNetwork(w.Ctx, w.admin, w.net.ID, "also pause"))
	noErr(t, w.Svc.PlatformResumeNetwork(w.Ctx, sys, "demo", "ops", "order lifted"))
	_, err = w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState) // admin suspension still holds
	noErr(t, w.Svc.ResumeNetwork(w.Ctx, w.admin, w.net.ID, "done"))
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 100); err != nil {
		t.Fatalf("after both lifted: %v", err)
	}
	if !hasKind(auditActions(w), domain.ActNetworkPlatformResumed) {
		t.Fatal("platform resume not audited")
	}
}

// FR-068 / GOV-025: closure lists blockers, previews balances, notifies members.
func TestClosureBlockersPreviewAndNotice(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	open := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))

	pv := tu.Must(w.Svc.PreviewClosure(w.Ctx, w.admin, w.net.ID))
	if len(pv.Blockers) != 1 || pv.NonZeroBalances != 2 {
		t.Fatalf("preview = %+v", pv)
	}
	if _, err := w.Svc.PreviewClosure(w.Ctx, w.a, w.net.ID); err == nil {
		t.Fatal("member previewed a closure")
	}
	wantCode(t, w.Svc.CloseNetwork(w.Ctx, w.admin, w.net.ID, "end"), domain.ErrBlocked)

	tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: open.ID, IdempotencyKey: w.Key()}))
	noErr(t, w.Svc.CloseNetwork(w.Ctx, w.admin, w.net.ID, "end"))
	if !hasKind(w.inbox(w.b), service.NotifNetworkClosed) {
		t.Fatal("member not notified of closure")
	}
	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("read after closure: %v", err)
	}
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrInvalidState)
}
