package store_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / DAT-042: the compatible rebuild preserves the original report
// fields, indexes and membership FK, adds no inferred relation, and rolls back
// late DDL failure before retry/reopen. Non-null context is network-bound and
// immutable; old context-free named-column inserts remain valid.
func TestReportDisputeContextMigrationPreservesAndRetries(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "report-context-upgrade.db")
	legacy := tu.Must(store.OpenAtVersion(ctx, path, 24))
	t.Cleanup(func() { _ = legacy.Close() })
	ids := &tu.IDs{}
	cfg := service.Config{IDs: ids, Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}}
	e := &tu.Env{T: t, Ctx: ctx, Store: legacy, Svc: tu.Must(service.New(legacy, cfg))}
	admin := e.Person("context-admin")
	net, adminM := e.Network(admin, "report-context", nil)
	alice, aliceM := e.Member(admin, net, "context-alice")
	bob, bobM := e.Member(admin, net, "context-bob")
	listing := tu.Must(e.Svc.CreateListing(ctx, bob, net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Retained report listing", IdempotencyKey: e.Key(),
	}))
	trade := e.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	first := tu.Must(e.Svc.OpenDispute(ctx, alice, net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Retained first case", IdempotencyKey: e.Key(),
	}))
	tu.Must(e.Svc.WithdrawDispute(ctx, alice, net.ID, first.ID))
	second := tu.Must(e.Svc.OpenDispute(ctx, bob, net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Retained second case", IdempotencyKey: e.Key(),
	}))
	farNet, _ := e.Network(admin, "far-report-context", nil)
	farA, farAM := e.Member(admin, farNet, "far-context-a")
	farB, farBM := e.Member(admin, farNet, "far-context-b")
	farTrade := e.Trade(farNet.ID, farA, farB, farAM, farBM, 100)
	farCase := tu.Must(e.Svc.OpenDispute(ctx, farA, farNet.ID, service.OpenDisputeInput{
		TxID: farTrade.ID, Category: "other", Statement: "Other network case", IdempotencyKey: e.Key(),
	}))
	created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	memberReport := &store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: aliceM.ID,
		TargetType: service.ReportMember, TargetID: bobM.ID, Category: "other", Note: "Original member note",
		Status: "open", CreatedAt: created}
	listingReport := &store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: aliceM.ID,
		TargetType: service.ReportListing, TargetID: listing.ID, Category: "scam", Note: "Original listing note",
		Status: "handled", CreatedAt: created, HandledBy: adminM.ID, HandledAt: &created, HandlingNote: "Original handling note"}
	transactionReport := &store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: bobM.ID,
		TargetType: service.ReportTransaction, TargetID: trade.ID, Category: "other", Note: "Original transaction note",
		Status: "open", CreatedAt: created}
	oldReports := []*store.Report{memberReport, listingReport, transactionReport}
	for _, report := range oldReports {
		tu.Must(0, insertHistoricalReport(legacy.DB(), *report))
	}
	before := tu.Snapshot(t, legacy.DB())
	shape := reportSchemaShape(t, legacy.DB())
	var originalSQL string
	tu.Must(0, legacy.DB().QueryRow("SELECT sql FROM sqlite_master WHERE name='reports'").Scan(&originalSQL))
	// This fails after the copied table replaces the original and all three
	// original indexes have been recreated.
	_, err := legacy.DB().Exec("CREATE INDEX reports_dispute ON networks(name)")
	tu.Must(0, err)
	tu.Must(0, legacy.Close())
	failed, err := store.Open(ctx, path)
	if err == nil {
		_ = failed.Close()
		t.Fatal("late context index collision did not refuse migration 25")
	}
	if !strings.Contains(err.Error(), "0025_report_dispute_context") {
		t.Fatalf("failure was outside the intended migration: %v", err)
	}
	legacy = tu.Must(store.OpenAtVersion(ctx, path, 24))
	if changed := before.Changed(tu.Snapshot(t, legacy.DB())); len(changed) != 0 {
		t.Fatalf("failed context migration changed durable data: %v", changed)
	}
	var version, temporary, contextColumns int
	var rolledBackSQL string
	tu.Must(0, legacy.DB().QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version))
	tu.Must(0, legacy.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='reports_next'").Scan(&temporary))
	tu.Must(0, legacy.DB().QueryRow("SELECT COUNT(*) FROM pragma_table_info('reports') WHERE name='dispute_id'").Scan(&contextColumns))
	tu.Must(0, legacy.DB().QueryRow("SELECT sql FROM sqlite_master WHERE name='reports'").Scan(&rolledBackSQL))
	if version != 24 || temporary != 0 || contextColumns != 0 || rolledBackSQL != originalSQL || !reflect.DeepEqual(reportSchemaShape(t, legacy.DB()), shape) {
		t.Fatal("failed context migration left a partial schema")
	}
	_, err = legacy.DB().Exec("DROP INDEX reports_dispute")
	tu.Must(0, err)
	tu.Must(0, legacy.Close())
	delete(before.Tables, "schema_migrations")
	delete(before.Tables, "reports") // all original fields and row count are compared separately
	for pass := 0; pass < 2; pass++ {
		upgraded := tu.Must(store.Open(ctx, path))
		t.Cleanup(func() { _ = upgraded.Close() })
		after := tu.Snapshot(t, upgraded.DB())
		delete(after.Tables, "schema_migrations")
		delete(after.Tables, "reports")
		if changed := before.Changed(after); len(changed) != 0 {
			t.Fatalf("context migration/reopen changed unrelated data: %v", changed)
		}
		current := reportSchemaShape(t, upgraded.DB())
		if current["columns"] != shape["columns"]+"|dispute_id:TEXT:0:NULL:0" {
			t.Fatal("the original twelve columns changed during the rebuild")
		}
		for key, value := range shape {
			if key != "columns" && key != "foreign_keys" && current[key] != value {
				t.Fatalf("original report index %s changed", key)
			}
		}
		for _, oldFK := range strings.Split(shape["foreign_keys"], "|") {
			if !strings.Contains("|"+current["foreign_keys"]+"|", "|"+oldFK+"|") {
				t.Fatal("the original reporter membership FK changed")
			}
		}
		var linked, reportCount int
		tu.Must(0, upgraded.DB().QueryRow("SELECT COUNT(*) FROM reports WHERE dispute_id IS NOT NULL").Scan(&linked))
		tu.Must(0, upgraded.DB().QueryRow("SELECT COUNT(*) FROM reports").Scan(&reportCount))
		if linked != 0 || reportCount != len(oldReports) {
			t.Fatal("migration inferred case relations, duplicated or lost reports")
		}
		for _, want := range oldReports {
			if got := readHistoricalReport(t, upgraded.DB(), net.ID, want.ID); !reflect.DeepEqual(got, want) {
				t.Fatalf("original report values changed: got %+v want %+v", got, want)
			}
		}
		if pass == 1 {
			e.Store, e.Svc = upgraded, tu.Must(service.New(upgraded, cfg))
			tu.Must(e.Svc.HandleReport(ctx, admin, net.ID, service.HandleReportInput{ReportID: memberReport.ID, IdempotencyKey: e.Key()}))
			filed := tu.Must(e.Svc.CreateReport(ctx, alice, net.ID, service.ReportInput{
				TargetType: service.ReportMember, TargetID: bobM.ID, DisputeID: first.ID,
				Category: "other", Note: "New explicit context", IdempotencyKey: e.Key(),
			}))
			if filed.DisputeID != first.ID {
				t.Fatal("current writer/reader did not persist case context")
			}
			for _, caseID := range []string{ids.New(), farCase.ID} {
				candidate := *filed
				candidate.ID, candidate.DisputeID, candidate.Status = ids.New(), caseID, "handled"
				if err := upgraded.Update(ctx, func(tx store.Tx) error { return tx.InsertReport(candidate) }); err == nil || !strings.Contains(err.Error(), "FOREIGN KEY") {
					t.Fatalf("missing or cross-network context was not refused by the FK: %v", err)
				}
			}
			for _, change := range []struct {
				id string
				to any
			}{{filed.ID, second.ID}, {filed.ID, nil}, {listingReport.ID, first.ID}} {
				if _, err := upgraded.DB().Exec("UPDATE reports SET dispute_id=? WHERE id=?", change.to, change.id); err == nil || !strings.Contains(err.Error(), "report dispute context is immutable") {
					t.Fatalf("context was redirected, removed or attached after creation: %v", err)
				}
			}
			contextFree := *listingReport
			contextFree.ID = ids.New()
			tu.Must(0, insertHistoricalReport(upgraded.DB(), contextFree))
			var contextIsNull bool
			tu.Must(0, upgraded.DB().QueryRow("SELECT dispute_id IS NULL FROM reports WHERE id=?", contextFree.ID).Scan(&contextIsNull))
			if !contextIsNull {
				t.Fatal("legacy named-column insert acquired a case relation")
			}
		}
		tu.Must(0, upgraded.Close())
	}
}
