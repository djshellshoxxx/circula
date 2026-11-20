package service_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func reportContextTrade(t *testing.T, w *world) (*domain.Transaction, string) {
	t.Helper()
	listingID := w.listing(w.b, "Listing for the disputed trade")
	trade := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{
		CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 100, ListingID: listingID, IdempotencyKey: w.Key(),
	}))
	return tu.Must(w.accept(w.b, trade.ID)), listingID
}

// SEC-025 / PER-003: explicit member/listing context belongs to the selected
// case, while a transaction report retains its derived relation to every case
// on that transaction. Both relations are filtered before keyset pagination.
func TestReportDisputeContextExactCaseAndProjection(t *testing.T) {
	w := newWorld(t, nil)
	reviewer, _ := w.reviewerAs("reviewer@example.org")
	trade, listingID := reportContextTrade(t, w)
	older := tu.Must(w.open(w.a, trade.ID))
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, older.ID))
	newer := tu.Must(w.open(w.b, trade.ID))
	file := func(typ, id, caseID string) *service.Report {
		t.Helper()
		return tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, service.ReportInput{
			TargetType: typ, TargetID: id, DisputeID: caseID, Category: "other", Note: "Private case report", IdempotencyKey: w.Key(),
		}))
	}
	legacy := file(service.ReportMember, w.bM.ID, "")
	tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: legacy.ID, IdempotencyKey: w.Key()}))
	listingReport := file(service.ReportListing, listingID, older.ID)
	memberReport := file(service.ReportMember, w.bM.ID, newer.ID)
	transactionReport := file(service.ReportTransaction, trade.ID, "")
	for _, tc := range []struct {
		caseID string
		ids    []string
	}{{older.ID, []string{transactionReport.ID, listingReport.ID}}, {newer.ID, []string{transactionReport.ID, memberReport.ID}}} {
		after := ""
		for _, want := range tc.ids {
			page := tu.Must(w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, tc.caseID, service.Page{AfterID: after, Limit: 1}))
			if len(page) != 1 || page[0] != (service.DisputeReport{ID: want, Category: "other", Status: "open"}) {
				t.Fatalf("case %s page after %s = %+v, want only %s", tc.caseID, after, page, want)
			}
			after = page[0].ID
		}
		if tail := tu.Must(w.Svc.ListDisputeReports(w.Ctx, reviewer, w.net.ID, tc.caseID, service.Page{AfterID: after, Limit: 1})); len(tail) != 0 {
			t.Fatal("projection inferred a context-free or other-case relation")
		}
	}
	context := tu.Must(w.Svc.GetReportDisputeContext(w.Ctx, w.a, w.net.ID, newer.ID))
	if context.MemberID != w.bM.ID || context.ListingID != listingID || context.DisputeID != newer.ID {
		t.Fatalf("wrong case report targets: %+v", context)
	}
	otherParty := tu.Must(w.Svc.GetReportDisputeContext(w.Ctx, w.b, w.net.ID, newer.ID))
	if otherParty.MemberID != w.aM.ID || otherParty.ListingID != "" {
		t.Fatal("the listing owner received their own listing as a report target")
	}
	for _, report := range []*service.Report{listingReport, memberReport} {
		done := tu.Must(w.Svc.HandleReport(w.Ctx, w.admin, w.net.ID, service.HandleReportInput{ReportID: report.ID, IdempotencyKey: w.Key()}))
		if done.DisputeID != report.DisputeID || done.Status != "handled" {
			t.Fatal("handling changed or lost immutable case context")
		}
	}
	// A subject's later roles do not grant the case projection or full report.
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleReviewer))
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleAdmin))
	before := tu.Snapshot(t, w.Store.DB())
	_, err := w.Svc.ListDisputeReports(w.Ctx, w.b, w.net.ID, newer.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	if queue := tu.Must(w.Svc.ListReports(w.Ctx, w.b, w.net.ID, "", service.Page{})); len(queue) != 0 {
		t.Fatal("subject administrator received reports with case context")
	}
	if events := tu.Must(w.Svc.ListAudit(w.Ctx, w.b, w.net.ID, service.AuditQuery{Action: domain.ActReportCreated})); len(events) != 0 {
		t.Fatal("subject audit read disclosed case reports")
	}
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("projection/privacy reads changed durable state: %v", changed)
	}
}

func TestReportDisputeContextValidatesPartyTargetAndNetwork(t *testing.T) {
	w := newWorld(t, nil)
	trade, listingID := reportContextTrade(t, w)
	dispute := tu.Must(w.open(w.a, trade.ID))
	other, otherM := w.Member2(w.admin, "other@example.org")
	otherTrade := w.Trade(w.net.ID, w.a, other, w.aM, otherM, 200)
	otherCase := tu.Must(w.open(w.a, otherTrade.ID))
	otherListing := w.listing(w.b, "Another listing from the same owner")
	farNet, _ := w.Network(w.admin, "far-context", nil)
	farA, farAM := w.Member(w.admin, farNet, "far-a@example.org")
	farB, farBM := w.Member(w.admin, farNet, "far-b@example.org")
	farTrade := w.Trade(farNet.ID, farA, farB, farAM, farBM, 100)
	farCase := tu.Must(w.Svc.OpenDispute(w.Ctx, farA, farNet.ID, service.OpenDisputeInput{
		TxID: farTrade.ID, Category: "other", Statement: "Another network case", IdempotencyKey: w.Key(),
	}))
	for _, tc := range []struct {
		name, typ, target, caseID string
		actor                     domain.Actor
		want                      *domain.Error
	}{
		{"nonparty administrator", service.ReportMember, w.bM.ID, dispute.ID, w.admin, domain.ErrNotFound},
		{"other target", service.ReportMember, otherM.ID, dispute.ID, w.a, domain.ErrValidation},
		{"self target", service.ReportMember, w.aM.ID, dispute.ID, w.a, domain.ErrValidation},
		{"same owner different listing", service.ReportListing, otherListing, dispute.ID, w.a, domain.ErrValidation},
		{"listing owner", service.ReportListing, listingID, dispute.ID, w.b, domain.ErrValidation},
		{"other party case", service.ReportMember, w.bM.ID, otherCase.ID, w.a, domain.ErrValidation},
		{"other network case", service.ReportMember, w.bM.ID, farCase.ID, w.a, domain.ErrNotFound},
		{"missing case", service.ReportMember, w.bM.ID, "018f0000-0000-7000-8000-999999999999", w.a, domain.ErrNotFound},
		{"malformed case", service.ReportMember, w.bM.ID, "ordinary missing reference", w.a, domain.ErrNotFound},
		{"transaction uses derived context", service.ReportTransaction, trade.ID, dispute.ID, w.a, domain.ErrValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tu.Snapshot(t, w.Store.DB())
			_, err := w.Svc.CreateReport(w.Ctx, tc.actor, w.net.ID, service.ReportInput{
				TargetType: tc.typ, TargetID: tc.target, DisputeID: tc.caseID, Category: "other", IdempotencyKey: w.Key(),
			})
			wantCode(t, err, tc.want)
			if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
				t.Fatalf("refused case relation changed durable state: %v", changed)
			}
		})
	}
	_, err := w.Svc.GetReportDisputeContext(w.Ctx, w.admin, w.net.ID, dispute.ID)
	wantCode(t, err, domain.ErrNotFound)
}

// DAT-025..027 / SEC-025: omitting context keeps the exact pre-extension
// fingerprint. Replaying either shape does not attach or redirect a report.
func TestReportDisputeContextDurableReplayCompatibility(t *testing.T) {
	w := newWorld(t, nil)
	trade, listingID := reportContextTrade(t, w)
	first := tu.Must(w.open(w.a, trade.ID))
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, first.ID))
	second := tu.Must(w.open(w.b, trade.ID))
	legacy := service.ReportInput{TargetType: service.ReportMember, TargetID: w.bM.ID,
		Category: "other", Note: "Context-free report", IdempotencyKey: w.Key()}
	filed := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, legacy))
	oldPayload := struct{ TargetType, TargetID, Category, Note string }{legacy.TargetType, legacy.TargetID, legacy.Category, legacy.Note}
	bytes := tu.Must(json.Marshal(oldPayload))
	digest := sha256.Sum256(bytes)
	if err := w.Store.View(w.Ctx, func(tx store.Tx) error {
		record, err := tx.GetIdem(w.net.ID, w.a.PersonID, "report.create", legacy.IdempotencyKey)
		if err != nil {
			return err
		}
		if record == nil || record.Fingerprint != hex.EncodeToString(digest[:]) {
			t.Fatal("empty optional context changed the historical request fingerprint")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := tu.Snapshot(t, w.Store.DB())
	if again := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, legacy)); !reflect.DeepEqual(again, filed) {
		t.Fatal("context-free durable replay changed its record")
	}
	legacy.DisputeID = first.ID
	_, err := w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, legacy)
	wantCode(t, err, domain.ErrKeyReused)
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("legacy replay/changed-context refusal changed state: %v", changed)
	}
	in := service.ReportInput{TargetType: service.ReportListing, TargetID: listingID, DisputeID: first.ID,
		Category: "other", Note: "Explicit case report", IdempotencyKey: w.Key()}
	contextReport := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, in))
	before = tu.Snapshot(t, w.Store.DB())
	if again := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, in)); !reflect.DeepEqual(again, contextReport) || again.DisputeID != first.ID {
		t.Fatal("explicit case context was lost on durable replay")
	}
	in.DisputeID = second.ID
	_, err = w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, in)
	wantCode(t, err, domain.ErrKeyReused)
	if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
		t.Fatalf("context replay/redirect refusal changed state: %v", changed)
	}
}

func TestReportDisputeContextDoesNotInheritCaseWindowOrListingVisibility(t *testing.T) {
	w := newWorld(t, nil)
	trade, listingID := reportContextTrade(t, w)
	dispute := tu.Must(w.open(w.a, trade.ID))
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, dispute.ID))
	archived := domain.ListingArchived
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.b, w.net.ID, service.ListingUpdate{ListingID: listingID, Status: &archived}))
	w.Clock.Advance(365 * 24 * time.Hour)
	context := tu.Must(w.Svc.GetReportDisputeContext(w.Ctx, w.a, w.net.ID, dispute.ID))
	if context.ListingID != listingID || context.ListingTitle != "" {
		t.Fatal("historical case lost its listing relation or disclosed a hidden listing title")
	}
	for _, target := range []struct{ typ, id string }{{service.ReportMember, w.bM.ID}, {service.ReportListing, listingID}} {
		r := tu.Must(w.Svc.CreateReport(w.Ctx, w.a, w.net.ID, service.ReportInput{
			TargetType: target.typ, TargetID: target.id, DisputeID: dispute.ID, Category: "other", IdempotencyKey: w.Key(),
		}))
		if r.DisputeID != dispute.ID {
			t.Fatal("historical case context was not persisted")
		}
	}
}
