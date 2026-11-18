package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestServeRefusesPendingMigrationsInExplicitMode simulates a database whose
// recorded schema is behind this binary and checks that serve refuses to start
// without migrating (OPS-025).
func TestServeRefusesPendingMigrationsInExplicitMode(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = (SELECT MAX(version) FROM schema_migrations)`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()

	cfg := testConfigFor(db)
	cfg.Listen = "127.0.0.1:0"
	cfg.MigrateMode = "explicit"
	cfg.PublicBaseURL = "https://x.test"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	err = runServe(ctx, cfg, logger)
	if err == nil || !strings.Contains(err.Error(), "pending migration") {
		t.Fatalf("explicit mode with pending migrations: err = %v", err)
	}
	if n := countBackups(t, filepath.Join(dir, "backups")); n != 0 {
		t.Errorf("explicit refusal took %d backups; it must not migrate", n)
	}
}

func countBackups(t *testing.T, dir string) int {
	t.Helper()
	entries, err := listBackups(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// OPS-025: a brand-new database is created at the current schema even in
// explicit mode (nothing to protect); an existing one with pending migrations
// is refused in explicit mode and backed up first in auto mode. An existing
// file with no recorded schema is not new and is refused in explicit mode.
func TestStartupMigrationGate(t *testing.T) {
	k := binarySchemaVersion()
	cases := []struct {
		name         string
		state        schemaState
		mode         string
		backup, fail bool
	}{
		{"new database, explicit", schemaState{Exists: false, Applied: 0, Known: k}, "explicit", false, false},
		{"new database, auto", schemaState{Exists: false, Applied: 0, Known: k}, "auto", false, false},
		{"current, explicit", schemaState{Exists: true, Applied: k, Known: k}, "explicit", false, false},
		{"current, auto", schemaState{Exists: true, Applied: k, Known: k}, "auto", false, false},
		{"pending, explicit", schemaState{Exists: true, Applied: k - 1, Known: k}, "explicit", false, true},
		{"pending, auto", schemaState{Exists: true, Applied: k - 1, Known: k}, "auto", true, false},
		{"empty file, explicit", schemaState{Exists: true, Applied: 0, Known: k}, "explicit", false, true},
		{"empty file, auto", schemaState{Exists: true, Applied: 0, Known: k}, "auto", true, false},
		{"newer schema, auto", schemaState{Exists: true, Applied: k + 1, Known: k}, "auto", false, false},
	}
	for _, c := range cases {
		backup, err := startupMigrationGate(c.state, c.mode)
		if backup != c.backup || (err != nil) != c.fail {
			t.Errorf("%s: backup=%v err=%v, want backup=%v fail=%v", c.name, backup, err, c.backup, c.fail)
		}
		if c.fail && (err == nil || !strings.Contains(err.Error(), "pending migration")) {
			t.Errorf("%s: refusal must name the pending migrations: %v", c.name, err)
		}
	}
}

// OPS-025 through runServe: in the default explicit mode, a database path
// that does not exist yet is created at the current schema and served, with
// no pre-migration backup. Guards the call site of startupMigrationGate, not
// only the helper.
func TestServeCreatesNewDatabaseInExplicitMode(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "new.db")
	cfg := testConfigFor(db)
	cfg.MigrateMode = "explicit"
	cfg.BackupInterval = 0
	cfg.PublicBaseURL = "https://x.test"
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Listen = probe.Addr().String()
	_ = probe.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- runServe(ctx, cfg, discardLogger()) }()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(60 * time.Second)
	for ready := false; !ready; {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("serve never became ready on a new database in explicit mode")
		}
		select {
		case err := <-errc:
			t.Fatalf("explicit mode refused a new database: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
		if resp, err := client.Get("http://" + cfg.Listen + "/readyz"); err == nil {
			ready = resp.StatusCode == http.StatusOK
			_ = resp.Body.Close()
		}
	}
	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("runServe after shutdown: %v", err)
		}
	case <-time.After(45 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
	state, err := inspectSchema(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Applied != binarySchemaVersion() || state.Pending() != 0 {
		t.Fatalf("new database schema: exists=%v applied=%d, want current v%d", state.Exists, state.Applied, binarySchemaVersion())
	}
	if n := countBackups(t, cfg.BackupDir); n != 0 {
		t.Errorf("a new database took %d pre-migration backup(s); there is nothing to protect", n)
	}
}
