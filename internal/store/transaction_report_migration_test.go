package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / DAT-042: widening the report target constraint preserves every
// legacy report field, index and foreign key. A late DDL failure after the
// copy/drop/rename rolls back all of those steps, and a retry succeeds without
// changing the financial ledger or audit history.
func TestTransactionReportMigrationPreservesAndRetries(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "transaction-report-upgrade.db")
	legacy := tu.Must(store.OpenAtVersion(ctx, path, 22))
	t.Cleanup(func() { _ = legacy.Close() })
	ids := &tu.IDs{}
	cfg := service.Config{IDs: ids, Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}}
	e := &tu.Env{T: t, Ctx: ctx, Store: legacy, Svc: tu.Must(service.New(legacy, cfg))}
	admin := e.Person("administrator")
	net, _ := e.Network(admin, "reports", nil)
	alice, aliceM := e.Member(admin, net, "alice")
	bob, bobM := e.Member(admin, net, "bob")
	trade := e.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	listing := tu.Must(e.Svc.CreateListing(ctx, bob, net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Existing listing", IdempotencyKey: e.Key(),
	}))
	memberReport := tu.Must(e.Svc.CreateReport(ctx, alice, net.ID, service.ReportInput{
		TargetType: service.ReportMember, TargetID: bobM.ID, Category: "other", Note: "Preserve open note", IdempotencyKey: e.Key(),
	}))
	listingReport := tu.Must(e.Svc.CreateReport(ctx, alice, net.ID, service.ReportInput{
		TargetType: service.ReportListing, TargetID: listing.ID, Category: "other", Note: "Preserve handled note", IdempotencyKey: e.Key(),
	}))
	listingReport = tu.Must(e.Svc.HandleReport(ctx, admin, net.ID, service.HandleReportInput{
		ReportID: listingReport.ID, Note: "Preserve handling details", IdempotencyKey: e.Key(),
	}))
	wantReports := []*store.Report{memberReport, listingReport}
	before := tu.Snapshot(t, legacy.DB())
	shape := reportSchemaShape(t, legacy.DB())
	var oldTableSQL string
	if err := legacy.DB().QueryRow("SELECT sql FROM sqlite_master WHERE name='reports'").Scan(&oldTableSQL); err != nil {
		t.Fatal(err)
	}
	// An ordinary conflicting index name fails the final migration statement,
	// after the report rows have been copied and the old table replaced.
	if _, err := legacy.DB().Exec("DROP INDEX reports_rate; CREATE INDEX reports_rate ON networks(name)"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	failed, err := store.Open(ctx, path)
	if err == nil {
		_ = failed.Close()
		t.Fatal("the conflicting final index did not fail migration 23")
	}
	if !strings.Contains(err.Error(), "0023_transaction_reports") {
		t.Fatalf("failure was outside the intended migration: %v", err)
	}
	legacy = tu.Must(store.OpenAtVersion(ctx, path, 22))
	if changed := before.Changed(tu.Snapshot(t, legacy.DB())); len(changed) != 0 {
		t.Fatalf("failed report migration changed durable rows: %v", changed)
	}
	var version, temporaryTables int
	var tableSQL string
	if err := legacy.DB().QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := legacy.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='reports_next'").Scan(&temporaryTables); err != nil {
		t.Fatal(err)
	}
	if err := legacy.DB().QueryRow("SELECT sql FROM sqlite_master WHERE name='reports'").Scan(&tableSQL); err != nil {
		t.Fatal(err)
	}
	if version != 22 || temporaryTables != 0 || tableSQL != oldTableSQL {
		t.Fatalf("failed migration left partial schema: version=%d temporary=%d table preserved=%t", version, temporaryTables, tableSQL == oldTableSQL)
	}
	if _, err := legacy.DB().Exec("DROP INDEX reports_rate; CREATE INDEX reports_rate ON reports(network_id, reporter_id, created_at)"); err != nil {
		t.Fatal(err)
	}
	if got := reportSchemaShape(t, legacy.DB()); !reflect.DeepEqual(got, shape) {
		t.Fatalf("rollback changed original report constraints/indexes: got %+v want %+v", got, shape)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	delete(before.Tables, "schema_migrations")
	for pass := 0; pass < 2; pass++ {
		upgraded := tu.Must(store.Open(ctx, path))
		t.Cleanup(func() { _ = upgraded.Close() })
		after := tu.Snapshot(t, upgraded.DB())
		delete(after.Tables, "schema_migrations")
		if changed := before.Changed(after); len(changed) != 0 {
			t.Fatalf("report migration or restart rewrote data: %v", changed)
		}
		if got := reportSchemaShape(t, upgraded.DB()); !reflect.DeepEqual(got, shape) {
			t.Fatalf("report columns/indexes/foreign key changed: got %+v want %+v", got, shape)
		}
		if err := upgraded.DB().QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
			t.Fatal(err)
		}
		if version != store.SchemaVersion() {
			t.Fatalf("report migration retry stopped at version %d", version)
		}
		if err := upgraded.View(ctx, func(tx store.Tx) error {
			for _, want := range wantReports {
				got, err := tx.GetReport(net.ID, want.ID)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("report fields changed: got %+v want %+v", got, want)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if pass == 1 {
			duplicate := *memberReport
			duplicate.ID = ids.New()
			if err := upgraded.Update(ctx, func(tx store.Tx) error { return tx.InsertReport(duplicate) }); domain.CodeOf(err) != domain.CodeConflict {
				t.Fatalf("existing duplicate-open constraint was lost: %v", err)
			}
			invalidReporter := *memberReport
			invalidReporter.ID, invalidReporter.ReporterID = ids.New(), ids.New()
			if err := upgraded.Update(ctx, func(tx store.Tx) error { return tx.InsertReport(invalidReporter) }); err == nil {
				t.Fatal("existing reporter membership foreign key was lost")
			}
			e.Svc = tu.Must(service.New(upgraded, cfg))
			filed := tu.Must(e.Svc.CreateReport(ctx, alice, net.ID, service.ReportInput{
				TargetType: service.ReportTransaction, TargetID: trade.ID, Category: "other", IdempotencyKey: e.Key(),
			}))
			if filed.TargetID != trade.ID || filed.TargetType != service.ReportTransaction {
				t.Fatalf("upgraded store did not persist the new report type: %+v", filed)
			}
		}
		if err := upgraded.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// reportSchemaShape excludes only the intentionally widened table CHECK.
// Column defaults/nullability, the member FK and all named indexes stay exact.
func reportSchemaShape(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	out := map[string]string{}
	for name, query := range map[string]string{
		"columns": `SELECT group_concat(shape, '|') FROM (SELECT name || ':' || type || ':' || "notnull" || ':' ||
			coalesce(dflt_value,'NULL') || ':' || pk AS shape FROM pragma_table_info('reports') ORDER BY cid)`,
		"foreign_keys": `SELECT group_concat(shape, '|') FROM (SELECT "table" || ':' || "from" || ':' || "to" || ':' ||
			on_update || ':' || on_delete || ':' || "match" AS shape FROM pragma_foreign_key_list('reports') ORDER BY id,seq)`,
	} {
		var value string
		if err := db.QueryRow(query).Scan(&value); err != nil {
			t.Fatal(err)
		}
		out[name] = value
	}
	rows, err := db.Query("SELECT name,sql FROM sqlite_master WHERE type='index' AND tbl_name='reports' AND sql IS NOT NULL ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			t.Fatal(err)
		}
		out[name] = strings.Join(strings.Fields(definition), " ")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
