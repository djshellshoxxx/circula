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
