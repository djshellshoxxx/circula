package store_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// API-036 / DAT-042: a failure after the session migration's ALTER and
// backfill must roll back the entire expansion, then allow a clean retry.
func TestSessionMigrationFailureRollsBackAndRetries(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "session-upgrade.db")
	legacy, err := store.OpenAtVersion(ctx, path, 19)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('owner','owner','hash',0);
		INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at)
		VALUES ('unchanged-credential-hash','owner',100,200,300,250);
		CREATE INDEX sessions_owner_page ON persons(login)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	failed, err := store.Open(ctx, path)
	if err == nil {
		_ = failed.Close()
		t.Fatal("conflicting index did not fail the session expansion")
	}
	if !strings.Contains(err.Error(), "0020_session_public_id") {
		t.Fatalf("failure was outside the intended session migration: %v", err)
	}
	legacy, err = store.OpenAtVersion(ctx, path, 19)
	if err != nil {
		t.Fatal(err)
	}
	var version, publicColumns, assignedIndexes, count int
	if err := legacy.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := legacy.DB().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name='public_id'`).Scan(&publicColumns); err != nil {
		t.Fatal(err)
	}
	if err := legacy.DB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='sessions_public_id'`).Scan(&assignedIndexes); err != nil {
		t.Fatal(err)
	}
	if err := legacy.DB().QueryRow(`SELECT COUNT(*) FROM sessions WHERE id='unchanged-credential-hash'
		AND person_id='owner' AND created_at=100 AND last_seen_at=200 AND expires_at=300 AND revoked_at=250`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if version != 19 || publicColumns != 0 || assignedIndexes != 0 || count != 1 {
		t.Fatalf("failed migration changed persisted state: version=%d columns=%d indexes=%d preserved=%d", version, publicColumns, assignedIndexes, count)
	}
	if _, err := legacy.DB().Exec(`DROP INDEX sessions_owner_page`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = upgraded.Close() }()
	if err := upgraded.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.DB().QueryRow(`SELECT COUNT(*) FROM sessions WHERE id='unchanged-credential-hash'
		AND person_id='owner' AND created_at=100 AND last_seen_at=200 AND expires_at=300 AND revoked_at=250
		AND length(public_id)=64`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if version != store.SchemaVersion() || count != 1 {
		t.Fatalf("retry failed to preserve credentials and finish migration: version=%d preserved=%d", version, count)
	}
}
