package service_test

import (
	"crypto/sha256"
	"errors"
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

// GOV-021 tests: recovery when no administrator is available.

var recoveryOp = domain.SystemActor("cli-admin-recovery")

// orphan produces the GOV-020 state the way TestAdminRequiredReadOnly does:
// the only administrator is suspended directly in the database.
func (w *world) orphan() {
	w.T.Helper()
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, w.adminM.ID); err != nil {
		w.T.Fatal(err)
	}
	if !tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		w.T.Fatal("network still has an active administrator")
	}
}

func (w *world) askRecovery(by domain.Actor) (*domain.AdminRecoveryRequest, error) {
	return w.Svc.RequestAdminRecovery(w.Ctx, by, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: w.Key()})
}

func (w *world) approveRecovery(id, operator string) (*service.AdminRecoveryOutcome, error) {
	return w.Svc.ApproveAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: id,
		Operator: operator, Reason: "ticket 7: identity confirmed against the member record"})
}

func (w *world) refuseRecovery(id, operator string) (*domain.AdminRecoveryRequest, error) {
	return w.Svc.RefuseAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: id,
		Operator: operator, Reason: "ticket 8: could not confirm the request"})
}

func (w *world) isAdmin(a domain.Actor) bool {
	_, err := w.Svc.AdminListMembers(w.Ctx, a, w.net.ID, nil, service.Page{})
	return err == nil
}

func (w *world) adminGrants(memberID string) int {
	w.T.Helper()
	var n int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM role_assignments WHERE network_id = ? AND member_id = ? AND role = 'admin'`,
		w.net.ID, memberID).Scan(&n); err != nil {
		w.T.Fatal(err)
	}
	return n
}

// ledgerDigest hashes every ledger table: accounts (balances and credit-limit
// overrides), entries, transactions and their status events, plus the
// network's posting counter.
func ledgerDigest(t *testing.T, w *world) string {
	t.Helper()
	h := sha256.New()
	for _, tb := range []string{"accounts", "entries", "transactions", "transaction_status_events", "pending_adjustments"} {
		r, err := w.Store.DB().Query("SELECT * FROM " + tb + " ORDER BY rowid")
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
	var seq int64
	if err := w.Store.DB().QueryRow(`SELECT ledger_seq FROM networks WHERE id = ?`, w.net.ID).Scan(&seq); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(h, "seq=%d", seq)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GOV-021 acceptance 1: an appointment without a second approval is refused.
// One operator records the first approval, which changes no right; the same
// operator (under any spelling of the label) cannot give the second; only a
// different operator's approval applies the request.
func TestGOV021AppointmentNeedsSecondApproval(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	if req.Status != domain.AdminRecoveryRequested || req.RequesterID != w.aM.ID {
		t.Fatalf("request = %+v", req)
	}
	first := tu.Must(w.approveRecovery(req.ID, "Olga"))
	if first.Applied || first.Request.Status != domain.AdminRecoveryApproved || first.Request.FirstOperator != "Olga" {
		t.Fatalf("first approval = %+v", first.Request)
	}
	if w.isAdmin(w.a) || w.adminGrants(w.aM.ID) != 0 || !tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		t.Fatal("a single approval appointed an administrator")
	}
	for _, same := range []string{"Olga", "olga", "  OLGA ", "Olga "} {
		_, err := w.approveRecovery(req.ID, same)
		wantCode(t, err, domain.ErrForbidden)
	}
	if w.isAdmin(w.a) || w.adminGrants(w.aM.ID) != 0 {
		t.Fatal("the first operator completed the appointment alone")
	}
	// No other operator path appoints without the request: an unknown request,
	// a member calling the operator method, and a direct jump in the database.
	_, err := w.approveRecovery("018f0000-0000-7000-8000-00000000ffff", "Piet")
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.ApproveAdminRecovery(w.Ctx, w.a, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: req.ID, Operator: "Piet", Reason: "self"})
	wantCode(t, err, domain.ErrForbidden)
	if _, err := w.Store.DB().Exec(`UPDATE admin_recovery_requests SET status='applied', closed_by='Olga', closed_reason='x',
		closed_at=1, role_assignment_id=NULL WHERE id = ?`, req.ID); err == nil {
		t.Fatal("schema accepted an applied request without a second operator and an appointment")
	}

	second := tu.Must(w.approveRecovery(req.ID, "Piet"))
	if !second.Applied || second.Request.Status != domain.AdminRecoveryApplied || second.Request.ClosedBy != "Piet" ||
		second.Request.RoleAssignment == "" {
		t.Fatalf("second approval = %+v", second.Request)
	}
	if !w.isAdmin(w.a) || w.adminGrants(w.aM.ID) != 1 || tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		t.Fatal("the dual-approved request did not appoint the administrator")
	}
	var by string
	if err := w.Store.DB().QueryRow(`SELECT granted_by FROM role_assignments WHERE id = ?`, second.Request.RoleAssignment).Scan(&by); err != nil || by != "" {
		t.Fatalf("appointment row granted_by = %q, %v; want the ordinary row with no granting member", by, err)
	}
	// Applied once: no further approval or refusal changes it.
	_, err = w.approveRecovery(req.ID, "Ravi")
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.refuseRecovery(req.ID, "Ravi")
	wantCode(t, err, domain.ErrInvalidState)
	if w.adminGrants(w.aM.ID) != 1 {
		t.Fatal("request applied twice")
	}
}

// GOV-021 acceptance 2: recovery actions produce audit events visible to the
// network: at network scope (administrators' log, members' recovery view) and
// at platform scope, for request, both approvals, refusal and expiry.
func TestGOV021RecoveryAuditVisibleToNetwork(t *testing.T) {
	w := newWorld(t, nil)
	c, cM := w.Member(w.admin, w.net, "cleo@example.org")
	w.orphan()
	refused := tu.Must(w.askRecovery(w.b))
	tu.Must(w.refuseRecovery(refused.ID, "Olga"))
	stale := tu.Must(w.askRecovery(c))
	w.Clock.Advance(domain.AdminRecoveryTTL)
	if n := tu.Must(w.Svc.ExpireAdminRecovery(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("expired %d", n)
	}
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.approveRecovery(req.ID, "Piet"))

	want := []string{domain.ActAdminRecoveryRequested, domain.ActAdminRecoveryRefused, domain.ActAdminRecoveryRequested,
		domain.ActAdminRecoveryExpired, domain.ActAdminRecoveryRequested, domain.ActAdminRecoveryApproved, domain.ActAdminRecoveryApplied}
	// A plain member who did not take part reads the whole sequence.
	for _, reader := range []domain.Actor{c, w.b, w.a} {
		evs := tu.Must(w.Svc.ListAdminRecoveryEvents(w.Ctx, reader, w.net.ID, service.AuditQuery{}))
		var got []string
		for _, e := range evs {
			got = append(got, e.Action)
			if e.Hash != "" || e.PrevHash != "" || len(e.SubjectIDs) != 0 {
				t.Errorf("member view of %s carries hash or subject data", e.Action)
			}
			if e.ActorType == domain.ActorOperator && (e.Reason != "" || strings.Contains(e.SummaryAfter, "Olga")) {
				t.Errorf("member view of %s carries the operator's reason or label: %+v", e.Action, e)
			}
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("events seen by a member = %v, want %v", got, want)
		}
	}
	// The appointed administrator sees the full events with the reasons.
	full := map[string]*domain.AuditEvent{}
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{Limit: 500})) {
		full[e.Action+"/"+e.TargetID] = e
	}
	ap := full[domain.ActAdminRecoveryApplied+"/"+req.ID]
	if ap == nil || ap.ActorType != domain.ActorOperator || ap.Reason == "" || !strings.Contains(ap.SummaryAfter, `"first_operator":"Olga"`) ||
		!strings.Contains(ap.SummaryAfter, `"operator":"Piet"`) {
		t.Fatalf("applied event in the administrators' log = %+v", ap)
	}
	if g := full[domain.ActRoleGranted+"/"+w.aM.ID]; g == nil || g.ActorType != domain.ActorOperator {
		t.Fatalf("role.granted of the appointment = %+v", g)
	}
	if e := full[domain.ActAdminRecoveryExpired+"/"+stale.ID]; e == nil || e.ActorType != domain.ActorSystem {
		t.Fatalf("expiry event = %+v", e)
	}
	// Operator steps are also written at platform scope.
	for _, act := range []string{domain.ActAdminRecoveryApproved, domain.ActAdminRecoveryApplied, domain.ActAdminRecoveryRefused, domain.ActAdminRecoveryExpired} {
		var n int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = '' AND action = ?`, act).Scan(&n); err != nil || n != 1 {
			t.Errorf("platform-scope %s events = %d, %v", act, n, err)
		}
	}
	// The requester sees their own request in their personal audit view.
	mine := tu.Must(w.Svc.ListAudit(w.Ctx, c, w.net.ID, service.AuditQuery{Limit: 100}))
	found := false
	for _, e := range mine {
		found = found || (e.Action == domain.ActAdminRecoveryRequested && e.TargetID == stale.ID)
	}
	if !found {
		t.Error("requester does not see their request in their own audit view")
	}
	// The chain still verifies (DAT-030).
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.a, w.net.ID)); !rep.OK {
		t.Fatalf("ledger or audit chain broken: %v", rep.Findings)
	}
	_ = cM
}

// GOV-021 acceptance 3: no recovery action changes a balance or an entry. The
// ledger (accounts with balances and limit overrides, entries, transactions,
// status events, posting counter) is byte-identical across a request, a
// refusal, an expiry, a first and a second approval and a supersession.
func TestGOV021RecoveryChangesNoBalanceOrEntry(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 700)
	w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 200)
	limit := int64(5000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.bM.ID, Limit: &limit, Reason: "fixture"}))
	w.orphan()
	before := ledgerDigest(t, w)
	balA, balB := w.balance(w.a).Balance, w.balance(w.b).Balance

	r1 := tu.Must(w.askRecovery(w.b))
	tu.Must(w.refuseRecovery(r1.ID, "Olga"))
	r2 := tu.Must(w.askRecovery(w.b))
	w.Clock.Advance(domain.AdminRecoveryTTL + time.Minute)
	_, err := w.approveRecovery(r2.ID, "Olga") // lazily expired by the attempt
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("approval of an expired request: %v", err)
	}
	other := tu.Must(w.askRecovery(w.b))
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.approveRecovery(req.ID, "Piet"))
	if got := tu.Must(w.Svc.ListAdminRecoveryRequests(w.Ctx, w.b, w.net.ID, service.Page{})); got[1].ID != other.ID || got[1].Status != domain.AdminRecoverySuperseded {
		t.Fatalf("other open request not superseded: %+v", got)
	}

	if after := ledgerDigest(t, w); after != before {
		t.Fatal("a recovery action changed the ledger")
	}
	if w.balance(w.a).Balance != balA || w.balance(w.b).Balance != balB {
		t.Fatal("a recovery action changed a balance")
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, sysJob, w.net.ID)); !rep.OK {
		t.Fatalf("ledger verification after recovery: %v", rep.Findings)
	}
	// The recovery table cannot reach the ledger: its schema names no ledger table.
	var ddl string
	if err := w.Store.DB().QueryRow(`SELECT group_concat(sql, ' ') FROM sqlite_master WHERE tbl_name = 'admin_recovery_requests'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	for _, tb := range []string{"entries", "accounts", "transactions", "balance"} {
		if strings.Contains(ddl, tb) {
			t.Errorf("recovery schema references %s", tb)
		}
	}
}

// GOV-021: who may ask. Only a recorded, active member of a network that has
// no active administrator; not a suspended, departed, pending or foreign
// person; not while an administrator is available (so an available, possibly
// malicious administrator cannot use this path to appoint someone).
func TestGOV021RequestEligibility(t *testing.T) {
	w := newWorld(t, nil)
	pending := w.Person("pending@example.org")
	if err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
		return tx.InsertMember(&domain.Member{ID: "018f0000-0000-7000-8000-0000000fffff", NetworkID: w.net.ID, PersonID: pending.PersonID,
			DisplayName: "Pending", Status: domain.MemberPending, JoinMethod: "approval", JoinedAt: w.Clock.Now()})
	}); err != nil {
		t.Fatal(err)
	}
	outsider := w.Person("outsider@example.org")
	_, err := w.askRecovery(w.a)
	wantCode(t, err, domain.ErrInvalidState) // an administrator is available
	_, err = w.askRecovery(w.admin)
	wantCode(t, err, domain.ErrInvalidState) // the administrator cannot use the recovery path either

	c, cM := w.Member(w.admin, w.net, "cleo@example.org")
	d, dM := w.Member(w.admin, w.net, "dora@example.org")
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, cM.ID, "fixture"))
	tu.Must(w.Svc.DepartMember(w.Ctx, d, w.net.ID, service.DepartInput{Reason: "moving"}))
	w.orphan()

	_, err = w.askRecovery(c)
	wantCode(t, err, domain.ErrForbidden) // suspended
	_, err = w.askRecovery(d)
	wantCode(t, err, domain.ErrForbidden) // departed
	_, err = w.askRecovery(pending)
	wantCode(t, err, domain.ErrForbidden) // pending applicant
	_, err = w.askRecovery(outsider)
	wantCode(t, err, domain.ErrNotFound) // not a member: indistinguishable from no network
	_, err = w.Svc.RequestAdminRecovery(w.Ctx, recoveryOp, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden) // the operator cannot start a request
	ro := w.a
	ro.Scopes = []domain.Scope{domain.ScopeRead}
	_, err = w.askRecovery(ro)
	wantCode(t, err, domain.ErrForbidden)
	_ = dM

	// A token scoped to another network reaches nothing here (IDOR).
	foreignAdmin := w.Person("foreign@example.org")
	other, _ := w.Network(foreignAdmin, "other", nil)
	scoped := w.a
	scoped.NetworkScope = other.ID
	_, err = w.askRecovery(scoped)
	wantCode(t, err, domain.ErrNotFound)

	// Idempotent: a retry returns the same request; a second request is refused.
	key := w.Key()
	r1 := tu.Must(w.Svc.RequestAdminRecovery(w.Ctx, w.a, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: key}))
	r2 := tu.Must(w.Svc.RequestAdminRecovery(w.Ctx, w.a, w.net.ID, service.AdminRecoveryInput{IdempotencyKey: key}))
	if r1.ID != r2.ID {
		t.Fatal("retry created a second request")
	}
	_, err = w.askRecovery(w.a)
	wantCode(t, err, domain.ErrConflict)
	_, err = w.Svc.RequestAdminRecovery(w.Ctx, w.b, w.net.ID, service.AdminRecoveryInput{})
	wantCode(t, err, domain.ErrKeyRequired)
	if !tu.Must(w.Svc.CanRequestAdminRecovery(w.Ctx, w.b, w.net.ID)) || tu.Must(w.Svc.CanRequestAdminRecovery(w.Ctx, w.a, w.net.ID)) ||
		tu.Must(w.Svc.CanRequestAdminRecovery(w.Ctx, c, w.net.ID)) {
		t.Fatal("CanRequestAdminRecovery disagrees with the request rules")
	}

	// IDOR on the operator side: a request id under another network's name.
	_, err = w.Svc.ApproveAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: other.Slug, RequestID: r1.ID,
		Operator: "Olga", Reason: "wrong network"})
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.RefuseAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: other.Slug, RequestID: r1.ID,
		Operator: "Olga", Reason: "wrong network"})
	wantCode(t, err, domain.ErrNotFound)
	if got := tu.Must(w.Svc.ListAdminRecoveryRequests(w.Ctx, foreignAdmin, other.ID, service.Page{})); len(got) != 0 {
		t.Fatalf("another network's members see %d requests", len(got))
	}
	_, err = w.Svc.ListAdminRecoveryRequests(w.Ctx, outsider, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)

	// Operator inputs: label and documented reason are mandatory.
	_, err = w.Svc.ApproveAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: r1.ID, Operator: "Olga"})
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.ApproveAdminRecovery(w.Ctx, recoveryOp, service.AdminRecoveryDecision{Network: w.net.Slug, RequestID: r1.ID, Reason: "x"})
	wantCode(t, err, domain.ErrValidation)
}

// GOV-021: requester eligibility is checked again at each approval; MFA is
// required of the requester when administrators must have it.
func TestGOV021RechecksAtEachApproval(t *testing.T) {
	t.Run("network regained an administrator", func(t *testing.T) {
		w := newWorld(t, nil)
		w.orphan()
		req := tu.Must(w.askRecovery(w.a))
		tu.Must(w.approveRecovery(req.ID, "Olga"))
		if _, err := w.Store.DB().Exec(`UPDATE members SET status='active' WHERE id = ?`, w.adminM.ID); err != nil {
			t.Fatal(err)
		}
		_, err := w.approveRecovery(req.ID, "Piet")
		wantCode(t, err, domain.ErrInvalidState)
		if w.isAdmin(w.a) || w.adminGrants(w.aM.ID) != 0 {
			t.Fatal("applied while an administrator was available")
		}
		tu.Must(w.refuseRecovery(req.ID, "Piet"))
	})
	t.Run("requester suspended before the first approval", func(t *testing.T) {
		w := newWorld(t, nil)
		w.orphan()
		req := tu.Must(w.askRecovery(w.a))
		if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, w.aM.ID); err != nil {
			t.Fatal(err)
		}
		_, err := w.approveRecovery(req.ID, "Olga")
		wantCode(t, err, domain.ErrInvalidState)
	})
	t.Run("requester departed before the second approval", func(t *testing.T) {
		w := newWorld(t, nil)
		w.orphan()
		req := tu.Must(w.askRecovery(w.a))
		tu.Must(w.approveRecovery(req.ID, "Olga"))
		if _, err := w.Store.DB().Exec(`UPDATE members SET status='departed', departed_at = 1 WHERE id = ?`, w.aM.ID); err != nil {
			t.Fatal(err)
		}
		_, err := w.approveRecovery(req.ID, "Piet")
		wantCode(t, err, domain.ErrInvalidState)
		if w.adminGrants(w.aM.ID) != 0 {
			t.Fatal("departed requester appointed")
		}
	})
	t.Run("network closing", func(t *testing.T) {
		w := newWorld(t, nil)
		tu.Must(w.schedule("end", 0))
		w.orphan()
		req := tu.Must(w.askRecovery(w.a))
		tu.Must(w.approveRecovery(req.ID, "Olga"))
		w.Clock.Advance(31 * 24 * time.Hour)
		_, err := w.approveRecovery(req.ID, "Piet")
		wantCode(t, err, domain.ErrInvalidState)
	})
	t.Run("MFA required for administrators", func(t *testing.T) {
		e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
		w := &world{Env: e}
		w.admin = e.Person("admin@example.org")
		enrol(t, e, w.admin)
		w.net, w.adminM = e.Network(w.admin, "demo", nil)
		w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
		w.orphan()
		_, err := w.askRecovery(w.a)
		if domain.CodeOf(err) != domain.CodeForbidden || !strings.Contains(err.Error(), "mfa-enrolment-required") {
			t.Fatalf("request without a second factor: %v", err)
		}
		_, codes := enrol(t, e, w.a)
		req := tu.Must(w.askRecovery(w.a))
		tu.Must(w.approveRecovery(req.ID, "Olga"))
		tu.Must(0, e.Svc.DisableMFA(e.Ctx, w.a, pw, codes[0]))
		_, err = w.approveRecovery(req.ID, "Piet")
		wantCode(t, err, domain.ErrInvalidState)
		if w.adminGrants(w.aM.ID) != 0 {
			t.Fatal("requester without a second factor appointed")
		}
	})
}

// GOV-021: a stale request expires (bounded, injected clock); the attempt or
// the job closes it, audits it and tells the members; nothing is appointed.
func TestGOV021RequestExpiry(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	if !req.ExpiresAt.Equal(req.RequestedAt.Add(domain.AdminRecoveryTTL)) {
		t.Fatalf("expiry = %v", req.ExpiresAt)
	}
	w.Clock.Advance(domain.AdminRecoveryTTL - time.Millisecond)
	tu.Must(w.approveRecovery(req.ID, "Olga")) // still open one millisecond before
	w.Clock.Advance(time.Millisecond)
	_, err := w.approveRecovery(req.ID, "Piet")
	wantCode(t, err, domain.ErrInvalidState)
	got := tu.Must(w.Svc.ListAdminRecoveryRequests(w.Ctx, w.b, w.net.ID, service.Page{}))
	if len(got) != 1 || got[0].Status != domain.AdminRecoveryExpired || got[0].ClosedAt == nil || !got[0].ClosedAt.Equal(req.ExpiresAt) {
		t.Fatalf("after expiry: %+v", got)
	}
	var stored string
	if err := w.Store.DB().QueryRow(`SELECT status FROM admin_recovery_requests WHERE id = ?`, req.ID).Scan(&stored); err != nil || stored != "expired" {
		t.Fatalf("expiry was not committed by the refused attempt: %q %v", stored, err)
	}
	if w.adminGrants(w.aM.ID) != 0 {
		t.Fatal("expired request appointed")
	}
	if !hasKind(w.inbox(w.b), service.NotifAdminRecoveryExpired) {
		t.Fatal("members not told of the expiry")
	}
	// The job closes the rest; an expired request does not block a new one.
	again := tu.Must(w.askRecovery(w.a))
	w.Clock.Advance(domain.AdminRecoveryTTL)
	if n := tu.Must(w.Svc.ExpireAdminRecovery(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("job expired %d", n)
	}
	if n := tu.Must(w.Svc.ExpireAdminRecovery(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("second job run expired %d", n)
	}
	if _, err := w.Svc.ExpireAdminRecovery(w.Ctx, w.a); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member ran the expiry job: %v", err)
	}
	_, err = w.refuseRecovery(again.ID, "Olga")
	wantCode(t, err, domain.ErrInvalidState)
}

// GOV-021: concurrent second approvals by different operators produce exactly
// one appointment; every other caller is refused and nothing is doubled.
func TestGOV021ConcurrentSecondApprovals(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	applied, refused := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out, err := w.approveRecovery(req.ID, fmt.Sprintf("Operator %d", i))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && out.Applied:
				applied++
			case errors.Is(err, domain.ErrInvalidState) || errors.Is(err, domain.ErrUnavailable):
				refused++
			default:
				t.Errorf("unexpected result %v, %v", out, err)
			}
		}(i)
	}
	wg.Wait()
	if applied != 1 || refused != n-1 {
		t.Fatalf("applied %d, refused %d", applied, refused)
	}
	if w.adminGrants(w.aM.ID) != 1 {
		t.Fatalf("role assignments = %d", w.adminGrants(w.aM.ID))
	}
	var events int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = ?`, w.net.ID,
		domain.ActAdminRecoveryApplied).Scan(&events); err != nil || events != 1 {
		t.Fatalf("applied events = %d, %v", events, err)
	}
}

// GOV-021: every current member is told at each step, beyond the first page of
// members (cb25b41); pending, departed and foreign members are not.
func TestGOV021NoticesReachAllMembers(t *testing.T) {
	w := newWorld(t, nil)
	c, cM := w.Member(w.admin, w.net, "cleo@example.org")
	d, _ := w.Member(w.admin, w.net, "dora@example.org")
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, cM.ID, "fixture"))
	tu.Must(w.Svc.DepartMember(w.Ctx, d, w.net.ID, service.DepartInput{Reason: "moving"}))
	foreignAdmin := w.Person("foreign@example.org")
	other, _ := w.Network(foreignAdmin, "other", nil)
	const fillers = 501
	err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
		for i := 0; i < fillers; i++ {
			pid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i)
			mid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i+1)
			if err := tx.InsertPerson(&domain.Person{ID: pid, Login: fmt.Sprintf("filler%d@example.org", i), CreatedAt: w.Clock.Now()}, "unused-fixture"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: mid, NetworkID: w.net.ID, PersonID: pid, DisplayName: "Filler",
				Status: domain.MemberActive, JoinMethod: "invitation", JoinedAt: w.Clock.Now()}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.approveRecovery(req.ID, "Piet"))
	// Current members: fillers, the suspended founder, a, b and suspended c.
	current := fillers + 4
	for _, kind := range []string{service.NotifAdminRecoveryRequested, service.NotifAdminRecoveryApproved, service.NotifAdminRecoveryApplied} {
		var n, distinct int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*), COUNT(DISTINCT member_id) FROM notifications WHERE network_id = ? AND kind = ?`,
			w.net.ID, kind).Scan(&n, &distinct); err != nil {
			t.Fatal(err)
		}
		if n != current || distinct != current {
			t.Errorf("%s notices = %d (%d distinct), want %d", kind, n, distinct, current)
		}
		var foreign int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM notifications WHERE network_id = ? AND kind = ?`, other.ID, kind).Scan(&foreign); err != nil || foreign != 0 {
			t.Errorf("foreign network got %d %s notices", foreign, kind)
		}
	}
	if !hasKind(w.inbox(c), service.NotifAdminRecoveryApplied) || hasKind(w.inbox(d), service.NotifAdminRecoveryRequested) {
		t.Fatal("suspended member not told, or departed member told")
	}
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{})) {
		if strings.HasPrefix(n.Kind, "network.admin_recovery") && (strings.Contains(n.Body, "Olga") || strings.Contains(n.Body, "ana") ||
			strings.Contains(n.Body, "ticket") || n.Link != "/admin-recovery") {
			t.Errorf("notice %s is not the neutral catalogue text: %+v", n.Kind, n)
		}
	}
}

// GOV-021 vs SEC-013: the recovery appointment is a permanent role assignment,
// not a break-glass grant, and ordinary break-glass grant-admin is unchanged:
// still temporary and still not counted as an administrator.
func TestGOV021SeparateFromBreakGlass(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	tu.Must(w.Svc.OpenBreakGlass(w.Ctx, recoveryOp, service.OpenBreakGlassInput{Network: w.net.Slug, Operator: "Olga", Reason: "ticket 9"}))
	tu.Must(w.Svc.GrantBreakGlassAdmin(w.Ctx, recoveryOp, service.GrantBreakGlassInput{Network: w.net.Slug, Login: "ben@example.org", Reason: "ticket 9"}))
	if !tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)).AdminRequired {
		t.Fatal("a break-glass grant ended the administrator-required state")
	}
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.approveRecovery(req.ID, "Piet"))
	var grants, sessions int
	if err := w.Store.DB().QueryRow(`SELECT (SELECT COUNT(*) FROM break_glass_grants), (SELECT COUNT(*) FROM break_glass_sessions)`).Scan(&grants, &sessions); err != nil {
		t.Fatal(err)
	}
	if grants != 1 || sessions != 1 {
		t.Fatalf("recovery created break-glass rows: grants %d sessions %d", grants, sessions)
	}
	w.Clock.Advance(domain.BreakGlassMaxTTL)
	if !w.isAdmin(w.a) {
		t.Fatal("the recovery appointment ended with the break-glass session")
	}
	if w.isAdmin(w.b) {
		t.Fatal("break-glass admin outlived its session")
	}
}

// GOV-021 / SEC-018: a request row holds no personal data of the requester.
// Once the requester leaves and erases their data, nothing in the recovery
// table names them, and their still-open request can no longer be approved.
func TestGOV021ErasedRequesterLeavesNoPersonalData(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	req := tu.Must(w.askRecovery(w.b))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.Svc.DepartMember(w.Ctx, w.b, w.net.ID, service.DepartInput{Reason: "moving"}))
	tu.Must(w.Svc.EraseMyData(w.Ctx, w.b, w.net.ID, ""))
	for _, needle := range []string{"ben@example.org", "ben"} {
		for _, hit := range dbMatches(t, w.Store.DB(), needle) {
			if strings.HasPrefix(hit, "admin_recovery_requests.") {
				t.Errorf("erased requester still named in %s", hit)
			}
		}
	}
	_, err := w.approveRecovery(req.ID, "Piet")
	wantCode(t, err, domain.ErrInvalidState)
	if w.adminGrants(w.bM.ID) != 0 {
		t.Fatal("erased requester appointed")
	}
	tu.Must(w.refuseRecovery(req.ID, "Piet"))
}
