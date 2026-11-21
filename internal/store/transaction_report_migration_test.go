package store_test

import (
	"context"
	"database/sql"
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
	e := &tu.Env{T: t, Ctx: ctx, Store: legacy, Svc: tu.Must(service.New(reportMigrationPriorCreditStore(t, legacy), cfg))}
	admin := e.Person("administrator")
	net, adminM := e.Network(admin, "reports", nil)
	alice, aliceM := e.Member(admin, net, "alice")
	bob, bobM := e.Member(admin, net, "bob")
	trade := e.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	listing := tu.Must(e.Svc.CreateListing(ctx, bob, net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Existing listing", IdempotencyKey: e.Key(),
	}))
	created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	memberReport := &store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: aliceM.ID,
		TargetType: service.ReportMember, TargetID: bobM.ID, Category: "other", Note: "Preserve open note",
		Status: "open", CreatedAt: created}
	listingReport := &store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: aliceM.ID,
		TargetType: service.ReportListing, TargetID: listing.ID, Category: "other", Note: "Preserve handled note",
		Status: "handled", CreatedAt: created, HandledBy: adminM.ID, HandledAt: &created, HandlingNote: "Preserve handling details"}
	tu.Must(0, insertHistoricalReport(legacy.DB(), *memberReport))
	tu.Must(0, insertHistoricalReport(legacy.DB(), *listingReport))
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
	failed, err := store.OpenAtVersion(ctx, path, 23)
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
		upgraded := tu.Must(store.OpenAtVersion(ctx, path, 23))
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
		if version != 23 {
			t.Fatalf("report migration retry stopped at version %d", version)
		}
		for _, want := range wantReports {
			if got := readHistoricalReport(t, upgraded.DB(), net.ID, want.ID); !reflect.DeepEqual(got, want) {
				t.Fatalf("report fields changed: got %+v want %+v", got, want)
			}
		}
		if pass == 1 {
			duplicate := *memberReport
			duplicate.ID = ids.New()
			if err := insertHistoricalReport(upgraded.DB(), duplicate); err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				t.Fatalf("existing duplicate-open constraint was lost: %v", err)
			}
			invalidReporter := *memberReport
			invalidReporter.ID, invalidReporter.ReporterID = ids.New(), ids.New()
			if err := insertHistoricalReport(upgraded.DB(), invalidReporter); err == nil {
				t.Fatal("existing reporter membership foreign key was lost")
			}
			// Exercise the widened schema directly. Current service/store report
			// readers include fields added by later migrations and are qualified
			// separately on the current schema.
			transactionReport := store.Report{ID: ids.New(), NetworkID: net.ID, ReporterID: aliceM.ID,
				TargetType: service.ReportTransaction, TargetID: trade.ID, Category: "other", Status: "open", CreatedAt: created}
			tu.Must(0, insertHistoricalReport(upgraded.DB(), transactionReport))
			filed := readHistoricalReport(t, upgraded.DB(), net.ID, transactionReport.ID)
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

// These fixture helpers use exactly the report columns present in schemas
// 0008..0024, so an upgrade test does not use future readers to seed old data.
func insertHistoricalReport(db *sql.DB, r store.Report) error {
	var handledBy, handledAt any
	if r.HandledBy != "" {
		handledBy = r.HandledBy
	}
	if r.HandledAt != nil {
		handledAt = r.HandledAt.UnixMilli()
	}
	_, err := db.Exec(`INSERT INTO reports(id,network_id,reporter_id,target_type,target_id,category,note,status,
		created_at,handled_by,handled_at,handling_note) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.NetworkID, r.ReporterID, r.TargetType, r.TargetID, r.Category, r.Note, r.Status,
		r.CreatedAt.UnixMilli(), handledBy, handledAt, r.HandlingNote)
	return err
}

func readHistoricalReport(t *testing.T, db *sql.DB, netID, id string) *store.Report {
	t.Helper()
	var r store.Report
	var created int64
	var handled sql.NullInt64
	if err := db.QueryRow(`SELECT id,network_id,reporter_id,target_type,target_id,category,note,status,
		created_at,COALESCE(handled_by,''),handled_at,handling_note FROM reports WHERE network_id=? AND id=?`, netID, id).
		Scan(&r.ID, &r.NetworkID, &r.ReporterID, &r.TargetType, &r.TargetID, &r.Category, &r.Note,
			&r.Status, &created, &r.HandledBy, &handled, &r.HandlingNote); err != nil {
		t.Fatal(err)
	}
	r.CreatedAt = time.UnixMilli(created).UTC()
	if handled.Valid {
		at := time.UnixMilli(handled.Int64).UTC()
		r.HandledAt = &at
	}
	return &r
}

// Seed the v22 fixture with the prior release's posting behavior. Every real
// posting, audit, balance and idempotency operation still reaches SQLite; the
// single witness write introduced by v24 is absent from that historical
// release. This adapter is test-only and refuses to wrap an adopted schema.
func reportMigrationPriorCreditStore(t *testing.T, st *store.SQLite) store.Store {
	t.Helper()
	version := tu.Must(store.AppliedVersion(context.Background(), st.DB()))
	if version >= 24 {
		t.Fatal("prior-credit fixture adapter cannot wrap an adopted schema")
	}
	return reportPriorCreditStore{Store: st}
}

type reportPriorCreditStore struct{ store.Store }

func (s reportPriorCreditStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(reportPriorCreditTx{Tx: tx, GovRepo: tx.(store.GovRepo)})
	})
}

type reportPriorCreditTx struct {
	store.Tx
	store.GovRepo
}

func (reportPriorCreditTx) InsertPostingCreditCheck(store.PostingCreditCheck) error { return nil }
