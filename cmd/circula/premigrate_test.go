package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

// OPS-023: the pre-migration gate backs up and re-verifies the backup from disk.
func TestPreMigrationSafeguardVerifiesBackup(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	cfg := testConfigFor(db)
	state, err := inspectSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	mf, path, err := preMigrationSafeguard(ctx, cfg, state)
	if err != nil {
		t.Fatalf("healthy database refused: %v", err)
	}
	if mf.Reason != "pre-migration" || mf.Integrity != "ok" || mf.SchemaVersion != state.Applied {
		t.Fatalf("manifest = %+v", mf)
	}

	// negative: the backup is altered after it was written
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 0xff
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyPreMigrationBackup(ctx, cfg, path, state); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("tampered backup accepted: %v", err)
	}

	// negative: backup holds a different schema than the live database
	cfg2 := *cfg
	cfg2.BackupDir = t.TempDir() // backup names have one-second resolution
	_, path2, err := preMigrationSafeguard(ctx, &cfg2, state)
	if err != nil {
		t.Fatal(err)
	}
	wrong := state
	wrong.Applied++
	if err := verifyPreMigrationBackup(ctx, cfg, path2, wrong); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("schema mismatch accepted: %v", err)
	}
}

func TestPreMigrationSafeguardSkipsNewDatabase(t *testing.T) {
	cfg := testConfigFor(t.TempDir() + "/none.db")
	mf, path, err := preMigrationSafeguard(context.Background(), cfg, schemaState{Exists: false, Known: 2})
	if err != nil || mf != nil || path != "" {
		t.Fatalf("new database: %v %v %q", err, mf, path)
	}
}

func TestPreMigrationSafeguardRefusesCorruptDatabase(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	cfg := testConfigFor(db)
	state, err := inspectSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	// Damage pages after the header: the file still opens, integrity_check or
	// the snapshot must fail, and no backup may be left behind.
	f, err := os.OpenFile(db, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	junk := make([]byte, 4096)
	for i := range junk {
		junk[i] = 0xA5
	}
	if _, err := f.WriteAt(junk, 8192); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	removeSideFiles(db)
	if _, _, err := preMigrationSafeguard(ctx, cfg, state); err == nil {
		t.Fatal("corrupt database was allowed to proceed to migration")
	}
}
