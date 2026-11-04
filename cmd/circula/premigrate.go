package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// preMigrationSafeguard is the OPS-018 / OPS-023 gate in front of every
// migration of an existing database. It (1) checks that the live database
// passes PRAGMA integrity_check, (2) takes an online backup, and (3) re-reads
// that backup from disk and verifies it against its manifest: checksum and
// size, manifest signature when backup-key is configured, integrity_check "ok"
// and the schema version it must hold. Any failure returns an error and the
// caller must not migrate. A database with nothing recorded yet has no data
// to protect and passes without a backup.
func preMigrationSafeguard(ctx context.Context, cfg *Config, state schemaState) (*Manifest, string, error) {
	if !state.Exists || state.Applied == 0 {
		return nil, "", nil
	}
	res, err := integrityCheck(ctx, cfg.Database)
	if err != nil {
		return nil, "", fmt.Errorf("pre-migration integrity check could not run; migrations not applied: %w", err)
	}
	if res != "ok" {
		return nil, "", fmt.Errorf("database failed integrity_check before migration (%s); migrations not applied. Restore a verified backup (OPS-020)", res)
	}
	mf, path, err := backupTo(ctx, cfg, cfg.BackupDir, "pre-migration", nil)
	if err != nil {
		return nil, "", fmt.Errorf("pre-migration backup failed; migrations not applied: %w", err)
	}
	if err := verifyPreMigrationBackup(ctx, cfg, path, state); err != nil {
		return nil, "", fmt.Errorf("pre-migration backup %s did not verify; migrations not applied: %w", path, err)
	}
	return mf, path, nil
}

// verifyPreMigrationBackup checks a backup file against what is on disk, not
// against what the writer believes it wrote. An encrypted backup (OPS-019) is
// decrypted into a private work directory, matched against the plaintext
// checksum and size in its manifest and integrity-checked there; the
// plaintext copy is overwritten and removed before returning, as in restore.
// Opening the ciphertext itself as SQLite can never succeed (A3-F01).
func verifyPreMigrationBackup(ctx context.Context, cfg *Config, path string, state schemaState) error {
	m, err := readManifest(path + ".manifest.json")
	if err != nil {
		return err
	}
	if err := verifyBackupFile(path, m); err != nil {
		return err
	}
	if _, err := checkManifestAuth(m, cfg.BackupKey); err != nil {
		return err
	}
	if m.Integrity != "ok" {
		return fmt.Errorf("manifest records integrity_check %q", m.Integrity)
	}
	if m.SchemaVersion != state.Applied {
		return fmt.Errorf("backup holds schema v%d, expected v%d", m.SchemaVersion, state.Applied)
	}
	check := path
	if m.Encrypted {
		work, err := os.MkdirTemp(filepath.Dir(path), tempDirPrefix)
		if err != nil {
			return fmt.Errorf("private work directory: %w", err)
		}
		defer secureRemoveAll(work)
		plain := filepath.Join(work, "verify.db")
		sum, size, err := decryptFile(path, plain, cfg.BackupEncryptionKey)
		if err != nil {
			return fmt.Errorf("decrypt backup: %w", err)
		}
		if sum != m.PlaintextSHA256 || size != m.PlaintextBytes {
			return errors.New("decrypted backup does not match the plaintext checksum in the manifest")
		}
		check = plain
	}
	res, err := integrityCheck(ctx, check)
	if err != nil {
		return err
	}
	if res != "ok" {
		return fmt.Errorf("backup failed integrity_check: %s", res)
	}
	return nil
}
