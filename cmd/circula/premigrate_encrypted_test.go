package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// noPlaintextLeft fails when dir holds a plaintext database copy or a private
// work directory after a backup or its verification (OPS-019, SEC-028).
func noPlaintextLeft(t *testing.T, dir string) {
	t.Helper()
	for _, n := range listDir(t, dir) {
		if strings.HasSuffix(n, backupSuffix) || strings.HasPrefix(n, tempDirPrefix) || strings.HasSuffix(n, "-wal") {
			t.Errorf("plaintext left in %s: %s (all: %v)", dir, n, listDir(t, dir))
		}
	}
}

// OPS-023, OPS-025, OPS-019 (A3-F01, was TestA3Audit_MigrateWithEncryptionKey):
// with backup-encryption-key configured, `circula migrate` takes an encrypted
// pre-migration backup, verifies it and migrates. It used to integrity-check
// the ciphertext as SQLite and refuse every migration.
func TestMigrateWithEncryptionKey(t *testing.T) {
	dir := t.TempDir()
	path := seedV1Database(t, dir)
	keyFile := writeKeyFile(t, dir, "bk.key", testKey(t))

	var out, errOut bytes.Buffer
	code := run([]string{"migrate", "-database", path, "-backup-encryption-key-file", keyFile}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("migrate with backup-encryption-key failed: code %d\nstdout: %s\nstderr: %s", code, out.String(), errOut.String())
	}
	state, err := inspectSchema(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Pending() != 0 {
		t.Fatalf("pending migrations after migrate: %d", state.Pending())
	}
	bk := filepath.Join(dir, "backups")
	var enc int
	for _, n := range listDir(t, bk) {
		if strings.HasSuffix(n, encryptedSuffix) {
			enc++
		}
	}
	if enc != 1 {
		t.Errorf("encrypted pre-migration backups: %d (%v)", enc, listDir(t, bk))
	}
	noPlaintextLeft(t, bk)
}

// OPS-023, OPS-019 (A3-F01, was TestA3Audit_PreMigrationSafeguardEncrypted):
// the gate accepts the encrypted backup it just wrote by decrypting it into a
// private directory, matching the manifest's plaintext checksum and running
// integrity_check there, and leaves no plaintext behind. A manifest whose
// plaintext checksum does not match, or a wrong key, is refused.
func TestPreMigrationSafeguardEncrypted(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	cfg := encConfig(db, testKey(t))
	cfg.BackupRequireEncryption = true
	state, err := inspectSchema(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	mf, path, err := preMigrationSafeguard(ctx, cfg, state)
	if err != nil {
		t.Fatalf("pre-migration safeguard refused an encrypted backup it just wrote: %v (file %s)", err, path)
	}
	if !mf.Encrypted || !strings.HasSuffix(path, encryptedSuffix) || mf.Integrity != "ok" || mf.SchemaVersion != state.Applied {
		t.Fatalf("manifest = %+v, file %s", mf, path)
	}
	noPlaintextLeft(t, cfg.BackupDir)

	wrongKey := *cfg
	wrongKey.BackupEncryptionKey = testKey(t)
	if err := verifyPreMigrationBackup(ctx, &wrongKey, path, state); err == nil || !strings.Contains(err.Error(), "decrypt") {
		t.Fatalf("backup verified under another key: %v", err)
	}
	noPlaintextLeft(t, cfg.BackupDir)

	m, err := readManifest(path + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	m.PlaintextSHA256 = strings.Repeat("0", 64)
	if err := writeManifest(path+".manifest.json", m); err != nil {
		t.Fatal(err)
	}
	if err := verifyPreMigrationBackup(ctx, cfg, path, state); err == nil || !strings.Contains(err.Error(), "plaintext checksum") {
		t.Fatalf("plaintext checksum mismatch accepted: %v", err)
	}
	noPlaintextLeft(t, cfg.BackupDir)
}
