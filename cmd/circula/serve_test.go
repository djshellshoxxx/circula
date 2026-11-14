package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
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
// is refused in explicit mode and backed up first in auto mode.
func TestStartupMigrationGate(t *testing.T) {
	cases := []struct {
		name         string
		state        schemaState
		mode         string
		backup, fail bool
	}{
		{"new database, explicit", schemaState{Exists: false, Applied: 0, Known: 16}, "explicit", false, false},
		{"new database, auto", schemaState{Exists: false, Applied: 0, Known: 16}, "auto", false, false},
		{"current, explicit", schemaState{Exists: true, Applied: 16, Known: 16}, "explicit", false, false},
		{"pending, explicit", schemaState{Exists: true, Applied: 15, Known: 16}, "explicit", false, true},
		{"pending, auto", schemaState{Exists: true, Applied: 15, Known: 16}, "auto", true, false},
		{"empty file, explicit", schemaState{Exists: true, Applied: 0, Known: 16}, "explicit", false, true},
	}
	for _, c := range cases {
		backup, err := startupMigrationGate(c.state, c.mode)
		if backup != c.backup || (err != nil) != c.fail {
			t.Errorf("%s: backup=%v err=%v, want backup=%v fail=%v", c.name, backup, err, c.backup, c.fail)
		}
	}
}
