//go:build unix

package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// SA-06: database and backups are owner-only whatever the umask, and a wider
// pre-existing directory or database file is tightened.
func TestFilesAreOwnerOnly(t *testing.T) {
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	if err := os.Chmod(db, 0o644); err != nil {
		t.Fatal(err)
	}
	if tightened, err := secureDatabase(db); err != nil || !tightened || mode(t, db) != 0o600 {
		t.Fatalf("secureDatabase: %v %v %v", tightened, err, mode(t, db))
	}
	fresh := filepath.Join(dir, "new.db")
	if _, err := secureDatabase(fresh); err != nil || mode(t, fresh) != 0o600 {
		t.Fatalf("new database: %v %v", err, mode(t, fresh))
	}

	bk := filepath.Join(dir, "backups")
	if err := os.Mkdir(bk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bk, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := testConfigFor(db)
	_, path, err := backupTo(ctx, cfg, bk, "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	if m := mode(t, bk); m != 0o700 {
		t.Errorf("backup dir mode %o", m)
	}
	for _, p := range []string{path, path + ".manifest.json"} {
		if m := mode(t, p); m != 0o600 {
			t.Errorf("%s mode %o", filepath.Base(p), m)
		}
	}
	// Manual backup to an arbitrary destination.
	dest := filepath.Join(dir, "manual.db")
	raw, _ := openRaw(db)
	defer func() { _ = raw.Close() }()
	if _, err := snapshot(ctx, raw, dest, "manual", testNow(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if m := mode(t, dest); m != 0o600 {
		t.Errorf("manual backup mode %o", m)
	}
}
