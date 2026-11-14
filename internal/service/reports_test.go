package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) report(by domain.Actor, typ, id, cat, note string) (string, error) {
	r, err := w.Svc.CreateReport(w.Ctx, by, w.net.ID, service.ReportInput{TargetType: typ, TargetID: id, Category: cat, Note: note, IdempotencyKey: w.Key()})
	if err != nil {
		return "", err
	}
	return r.ID, nil
}

func (w *world) listing(by domain.Actor, title string) string {
	return tu.Must(w.Svc.CreateListing(w.Ctx, by, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: title, IdempotencyKey: w.Key()})).ID
}

// FR-024: report a listing and a member; queue is administrator-only; nothing about the target changes.
func TestReports(t *testing.T) {
	w := newWorld(t, nil)
	l := w.listing(w.b, "Bike repair")
	id := tu.Must(w.report(w.a, "listing", l, "spam", "  looks fake  "))
	mid := tu.Must(w.report(w.a, "member", w.bM.ID, "harassment", ""))

	// nothing changed on the target
	if got := tu.Must(w.Svc.GetListing(w.Ctx, w.b, w.net.ID, l)); got.Status != domain.ListingActive {
		t.Errorf("listing status = %s", got.Status)
	}
	if got := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.bM.ID)); got.Status != domain.MemberActive {
		t.Errorf("member status = %s", got.Status)
	}

	q := tu.Must(w.Svc.ListReports(w.Ctx, w.admin, w.net.ID, "open", service.Page{}))
	if len(q) != 2 || q[0].ID != mid || q[1].ID != id {
		t.Fatalf("queue = %+v", q)
	}
	if q[1].ReporterName != w.aM.DisplayName || q[1].TargetLabel != "Bike repair" || q[1].Note != "looks fake" {
		t.Errorf("view = %+v", q[1])
	}
	if q[0].TargetLabel != w.bM.DisplayName {
		t.Errorf("member label = %q", q[0].TargetLabel)
	}

	// the queue and the reporter's identity are for administrators only
	for _, who := range []domain.Actor{w.a, w.b} {
		_, err := w.Svc.ListReports(w.Ctx, who, w.net.ID, "", service.Page{})
		wantCode(t, err, domain.ErrForbidden)
	}
	rev, _ := w.reviewerAs("rev@example.org")
	_, err := w.Svc.ListReports(w.Ctx, rev, w.net.ID, "", service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.ListReports(w.Ctx, w.admin, w.net.ID, "weird", service.Page{})
	wantCode(t, err, domain.ErrValidation)

	// the reported member's audit view does not mention the report; the reporter's does
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{})) {
		if strings.HasPrefix(e.Action, "report.") {
			t.Errorf("reported member sees %s", e.Action)
		}
	}
	seen := false
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{})) {
		seen = seen || e.Action == domain.ActReportCreated
	}
	if !seen {
		t.Error("reporter does not see own report event")
	}
	if n := len(tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActReportCreated}))); n != 2 {
		t.Errorf("admin audit events = %d", n)
	}
}

func TestReportValidationAndDuplicates(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	own := w.listing(w.a, "mine")
	l := w.listing(w.b, "theirs")

	_, err := w.report(w.a, "listing", own, "spam", "")
	wantCode(t, err, domain.ErrValidation) // own listing
	_, err = w.report(w.a, "member", w.aM.ID, "spam", "")
	wantCode(t, err, domain.ErrValidation) // self
	_, err = w.report(w.a, "listing", l, "bogus", "")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.report(w.a, "listing", l, "spam", strings.Repeat("n", 501))
	wantCode(t, err, domain.ErrValidation)
	_, err = w.report(w.a, "thing", l, "spam", "")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.report(w.a, "listing", "missing", "spam", "")
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.report(w.a, "member", "missing", "spam", "")
	wantCode(t, err, domain.ErrNotFound)
	pend, pendM := w.joinPending("pending@example.org")
	_, err = w.report(w.a, "member", pendM.ID, "spam", "")
	wantCode(t, err, domain.ErrNotFound) // pending members are not visible
	_, err = w.report(pend, "listing", l, "spam", "")
	wantCode(t, err, domain.ErrForbidden)

	id := tu.Must(w.report(w.a, "listing", l, "spam", ""))
	_, err = w.report(w.a, "listing", l, "scam", "")
	wantCode(t, err, domain.ErrConflict) // duplicate while open
	if _, err := w.report(w.b, "member", w.aM.ID, "spam", ""); err != nil {
		t.Errorf("another reporter / target refused: %v", err)
	}
	// replayed idempotent create returns the same report
	in := service.ReportInput{TargetType: "member", TargetID: w.bM.ID, Category: "other", IdempotencyKey: "same"}
	r1 := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, in))
	r2 := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, in))
	if r1.ID != r2.ID {
		t.Error("replay created a second report")
	}
	// once handled, the same member may report again
	tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, IdempotencyKey: w.Key()}))
	if _, err := w.report(w.a, "listing", l, "scam", ""); err != nil {
		t.Errorf("re-report after handling refused: %v", err)
	}
	// a suspended reporter cannot add reports
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "review"))
	_, err = w.report(w.a, "member", w.bM.ID, "spam", "")
	wantCode(t, err, domain.ErrForbidden)
}

func TestReportRateLimit(t *testing.T) {
	w := newWorld(t, nil)
	var ls []string
	for i := 0; i < service.MaxReportsPerDay+1; i++ {
		ls = append(ls, w.listing(w.b, fmt.Sprintf("l%d", i)))
	}
	for i := 0; i < service.MaxReportsPerDay; i++ {
		tu.Must(w.report(w.a, "listing", ls[i], "spam", ""))
	}
	_, err := w.report(w.a, "listing", ls[service.MaxReportsPerDay], "spam", "")
	wantCode(t, err, domain.ErrRateLimited)
	// another member is unaffected, and the day rolls over
	tu.Must(w.report(w.b, "member", w.aM.ID, "spam", ""))
	w.Clock.Advance(25 * time.Hour)
	tu.Must(w.report(w.a, "listing", ls[service.MaxReportsPerDay], "spam", ""))
}

func TestHandleReport(t *testing.T) {
	w := newWorld(t, nil)
	l := w.listing(w.b, "thing")
	id := tu.Must(w.report(w.a, "listing", l, "scam", ""))

	_, err := w.Svc.HandleReport(w.Ctx, w.a, w.net.ID, service.HandleReportInput{ReportID: id, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: "missing", IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, Note: strings.Repeat("n", 501), IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)

	r := tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, Note: "warned the member", IdempotencyKey: "h1"}))
	if r.Status != "handled" || r.HandledBy != w.adminM.ID || r.HandledAt == nil || r.HandlingNote != "warned the member" {
		t.Fatalf("report = %+v", r)
	}
	again := tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, Note: "warned the member", IdempotencyKey: "h1"}))
	if again.ID != r.ID {
		t.Error("replay differs")
	}
	_, err = w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: id, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrInvalidState)

	if open := tu.Must(w.Svc.ListReports(w.Ctx, w.admin, w.net.ID, "open", service.Page{})); len(open) != 0 {
		t.Errorf("open queue = %+v", open)
	}
	if done := tu.Must(w.Svc.ListReports(w.Ctx, w.admin, w.net.ID, "handled", service.Page{})); len(done) != 1 {
		t.Errorf("handled = %+v", done)
	}
	if !hasKind(w.inbox(w.a), service.NotifReportHandled) {
		t.Error("reporter not told of the outcome")
	}
	if hasKind(w.inbox(w.b), service.NotifReportHandled) {
		t.Error("reported member told")
	}
	evs := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActReportHandled}))
	if len(evs) != 1 || evs[0].Reason != "warned the member" {
		t.Errorf("audit = %+v", evs)
	}
}
