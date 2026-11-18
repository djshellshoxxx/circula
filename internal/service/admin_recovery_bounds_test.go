package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// seedOpenRecoveries adds n members of the network, each with an open,
// unexpired recovery request whose id sorts after every id the service
// generates, so they fill the newest pages of any id-descending listing. Rows
// are seeded directly to avoid n password hashes and n x members notices.
func seedOpenRecoveries(t *testing.T, w *world, n int, prefix string) []string {
	t.Helper()
	now := w.Clock.Now()
	ids := make([]string, 0, n)
	err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
		for i := 0; i < n; i++ {
			pid := fmt.Sprintf("00000000-0000-4000-8%s-%012x", prefix, 2*i)
			mid := fmt.Sprintf("00000000-0000-4000-8%s-%012x", prefix, 2*i+1)
			if err := tx.InsertPerson(&domain.Person{ID: pid, Login: fmt.Sprintf("seed%s-%d@example.org", prefix, i), CreatedAt: now}, "unused-fixture"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: mid, NetworkID: w.net.ID, PersonID: pid, DisplayName: "Seed",
				Status: domain.MemberActive, JoinMethod: "invitation", JoinedAt: now}); err != nil {
				return err
			}
			id := fmt.Sprintf("018f0000-0000-7000-9%s-%012x", prefix, i)
			if err := tx.InsertAdminRecovery(&domain.AdminRecoveryRequest{ID: id, NetworkID: w.net.ID, RequesterID: mid,
				Status: domain.AdminRecoveryRequested, RequestedAt: now, ExpiresAt: now.Add(domain.AdminRecoveryTTL)}); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// GOV-021: an old expired request must not block the member's new request,
// and an old open one must still be found, however many newer open requests
// the network has. The rules look the member's own open request up through
// the one-open-request index instead of scanning the newest 500 rows.
func TestGOV021OpenRequestLookupBeyondFirstPage(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	old := tu.Must(w.askRecovery(w.a))
	tu.Must(w.askRecovery(w.b))
	w.Clock.Advance(domain.AdminRecoveryTTL + time.Minute)
	tu.Must(w.askRecovery(w.b)) // b's own expired request is closed and replaced
	seeded := seedOpenRecoveries(t, w, 501, "000")

	// a's expired request is older than the 501 seeded ones: closed, not blocking.
	again, err := w.askRecovery(w.a)
	if err != nil {
		t.Fatalf("expired request beyond the first page blocked a new one: %v", err)
	}
	var status string
	if err := w.Store.DB().QueryRow(`SELECT status FROM admin_recovery_requests WHERE id = ?`, old.ID).Scan(&status); err != nil || status != "expired" {
		t.Fatalf("old request status %q, %v", status, err)
	}
	// b's open request is older than the 501 seeded ones: still found.
	_, err = w.askRecovery(w.b)
	if domain.CodeOf(err) != domain.CodeConflict || !strings.Contains(err.Error(), "already have an open") {
		t.Fatalf("second open request beyond the first page: %v", err)
	}
	// Applying one request supersedes every other open request, all pages of them.
	tu.Must(w.approveRecovery(again.ID, "Olga"))
	tu.Must(w.approveRecovery(again.ID, "Piet"))
	var open int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM admin_recovery_requests WHERE network_id = ? AND status IN ('requested','approved')`,
		w.net.ID).Scan(&open); err != nil || open != 0 {
		t.Fatalf("open requests after the appointment: %d, %v", open, err)
	}
	var superseded int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM admin_recovery_requests WHERE status = 'superseded'`).Scan(&superseded); err != nil ||
		superseded != len(seeded)+1 {
		t.Fatalf("superseded = %d, %v; want %d", superseded, err, len(seeded)+1)
	}
}

// GOV-021: the expiry job closes every expired request, across batches.
func TestGOV021ExpiryJobCoversEveryBatch(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	seeded := seedOpenRecoveries(t, w, 150, "001")
	w.Clock.Advance(domain.AdminRecoveryTTL)
	if n := tu.Must(w.Svc.ExpireAdminRecovery(w.Ctx, sysJob)); n != len(seeded) {
		t.Fatalf("job expired %d, want %d", n, len(seeded))
	}
	var open int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM admin_recovery_requests WHERE status IN ('requested','approved')`).Scan(&open); err != nil || open != 0 {
		t.Fatalf("open after the job: %d, %v", open, err)
	}
}

// GOV-021: a network that reaches its closing date cannot receive an
// administrator through recovery, even with an unexpired, once-approved
// request. The request stays open; the refusal names the network state.
func TestGOV021ClosingNetworkRefusesAppointment(t *testing.T) {
	w := newWorld(t, nil)
	c := tu.Must(w.schedule("end", 0)) // closing date in 30 days
	w.Clock.Advance(c.ClosingAt.Sub(w.Clock.Now()) - 24*time.Hour)
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	w.Clock.Advance(48 * time.Hour) // past the closing date, well inside the 7-day request window
	if !w.Clock.Now().Before(req.ExpiresAt) {
		t.Fatal("fixture: the request expired; the lifecycle guard would not be reached")
	}
	_, err := w.approveRecovery(req.ID, "Piet")
	if domain.CodeOf(err) != domain.CodeInvalidState || !strings.Contains(err.Error(), "network is closing") {
		t.Fatalf("second approval on a closing network: %v", err)
	}
	if w.adminGrants(w.aM.ID) != 0 {
		t.Fatal("a closing network received an administrator")
	}
	got := tu.Must(w.Svc.ListAdminRecoveryRequests(w.Ctx, w.b, w.net.ID, service.Page{}))
	if len(got) != 1 || got[0].Status != domain.AdminRecoveryApproved {
		t.Fatalf("request after the refused approval: %+v", got)
	}
	_, err = w.askRecovery(w.b)
	if domain.CodeOf(err) != domain.CodeInvalidState || !strings.Contains(err.Error(), "network is closing") {
		t.Fatalf("new request on a closing network: %v", err)
	}
}

// GOV-021 / SEC-013 a, b: the operator's listing reads member data, so it
// needs a label and a documented reason, shows only the fields needed to
// decide, pages by keyset, and is audited at platform scope and in each
// network whose requests it showed, where the network can read it.
func TestGOV021OperatorListIsAudited(t *testing.T) {
	w := newWorld(t, nil)
	w.orphan()
	req := tu.Must(w.askRecovery(w.a))
	founder2 := w.Person("founder2@example.org")
	other, otherFounder := w.Network(founder2, "other", nil)
	cora, coraM := w.Member(founder2, other, "cora@example.org")
	quiet := w.Person("quiet@example.org")
	quietNet, _ := w.Network(quiet, "quiet", nil)
	if _, err := w.Store.DB().Exec(`UPDATE members SET status='suspended' WHERE id = ?`, otherFounder.ID); err != nil {
		t.Fatal(err)
	}
	coraReq := tu.Must(w.Svc.RequestAdminRecovery(w.Ctx, cora, other.ID, service.AdminRecoveryInput{IdempotencyKey: w.Key()}))
	listed := func(netID string) int {
		t.Helper()
		var n int
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = ?`, netID,
			domain.ActAdminRecoveryListed).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	list := func(in service.AdminRecoveryListInput) ([]service.AdminRecoveryOperatorView, error) {
		return w.Svc.ListAdminRecoveryOperator(w.Ctx, recoveryOp, in)
	}

	_, err := list(service.AdminRecoveryListInput{Operator: "Olga"})
	wantCode(t, err, domain.ErrValidation) // no reason
	_, err = list(service.AdminRecoveryListInput{Reason: "ticket 7"})
	wantCode(t, err, domain.ErrValidation) // no operator label
	_, err = w.Svc.ListAdminRecoveryOperator(w.Ctx, w.b, service.AdminRecoveryListInput{Operator: "Olga", Reason: "ticket 7"})
	wantCode(t, err, domain.ErrForbidden)
	if listed("") != 0 || listed(w.net.ID) != 0 {
		t.Fatal("a refused listing was audited")
	}

	in := service.AdminRecoveryListInput{OpenOnly: true, Operator: "Olga", Reason: "ticket 7: review open requests"}
	got := tu.Must(list(in))
	if len(got) != 2 || got[0].ID != coraReq.ID || got[1].ID != req.ID {
		t.Fatalf("listing = %+v", got)
	}
	v := got[1]
	if v.NetworkSlug != w.net.Slug || v.Status != domain.AdminRecoveryRequested || !v.AdminRequired || !v.Eligible ||
		v.RequesterID != w.aM.ID || v.RequesterName != "ana@example.org" || v.FirstApprovedAt != nil ||
		!v.MemberSince.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) || !v.ExpiresAt.Equal(req.ExpiresAt) {
		t.Fatalf("operator view = %+v", v)
	}
	if listed("") != 1 || listed(w.net.ID) != 1 || listed(other.ID) != 1 || listed(quietNet.ID) != 0 {
		t.Fatalf("list audit rows: platform %d, demo %d, other %d, quiet %d", listed(""), listed(w.net.ID), listed(other.ID), listed(quietNet.ID))
	}
	var subjects string
	if err := w.Store.DB().QueryRow(`SELECT group_concat(s.member_id) FROM audit_subjects s JOIN audit_events e ON e.id = s.audit_id
		WHERE e.network_id = ? AND e.action = ?`, other.ID, domain.ActAdminRecoveryListed).Scan(&subjects); err != nil || subjects != coraM.ID {
		t.Fatalf("subjects of the other network's list event = %q, %v", subjects, err)
	}

	// Keyset paging, also audited per page.
	page1 := tu.Must(list(service.AdminRecoveryListInput{Operator: "Olga", Reason: "ticket 7", Page: service.Page{Limit: 1}}))
	page2 := tu.Must(list(service.AdminRecoveryListInput{Operator: "Olga", Reason: "ticket 7", Page: service.Page{AfterID: page1[0].ID, Limit: 1}}))
	if len(page1) != 1 || len(page2) != 1 || page1[0].ID != coraReq.ID || page2[0].ID != req.ID {
		t.Fatalf("pages = %+v / %+v", page1, page2)
	}
	if one := tu.Must(list(service.AdminRecoveryListInput{Network: other.Slug, Operator: "Olga", Reason: "ticket 8"})); len(one) != 1 || listed(w.net.ID) != 2 {
		t.Fatalf("network filter = %+v; demo list events %d", one, listed(w.net.ID))
	}

	// The network reads the access: members on the recovery page (without the
	// reason or label), the requester in their own audit view, the
	// administrator in the full log with reason and label.
	seen := false
	for _, e := range tu.Must(w.Svc.ListAdminRecoveryEvents(w.Ctx, w.b, w.net.ID, service.AuditQuery{})) {
		if e.Action == domain.ActAdminRecoveryListed {
			seen = true
			if e.Reason != "" || strings.Contains(e.SummaryAfter, "Olga") || e.ActorType != domain.ActorOperator {
				t.Errorf("member view of the list event = %+v", e)
			}
		}
	}
	if !seen {
		t.Fatal("members cannot see the operator's access")
	}
	own := false
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{Limit: 200})) {
		own = own || e.Action == domain.ActAdminRecoveryListed
	}
	if !own {
		t.Fatal("the requester does not see the access to their request")
	}
	tu.Must(w.approveRecovery(req.ID, "Olga"))
	tu.Must(w.approveRecovery(req.ID, "Piet"))
	full := false
	for _, e := range tu.Must(w.Svc.ListAudit(w.Ctx, w.a, w.net.ID, service.AuditQuery{Action: domain.ActAdminRecoveryListed, Limit: 50})) {
		full = full || (e.Reason == "ticket 7: review open requests" && strings.Contains(e.SummaryAfter, `"operator":"Olga"`))
	}
	if !full {
		t.Fatal("the network's administrator does not see the list event with its reason and label")
	}
}
