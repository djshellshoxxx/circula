package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// BR-06: export-spool-dir defaults to <state-dir>/export-spool and is set by
// flag or CIRCULA_EXPORT_SPOOL_DIR.
func TestExportSpoolDirSetting(t *testing.T) {
	cfg, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("state-dir"): "/srv/circula/state"}))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/srv/circula/state", "export-spool"); cfg.ExportSpoolDir != filepath.Clean(want) {
		t.Errorf("default spool dir %q, want %q", cfg.ExportSpoolDir, want)
	}
	if envName("export-spool-dir") != "CIRCULA_EXPORT_SPOOL_DIR" {
		t.Errorf("env name %s", envName("export-spool-dir"))
	}
	cfg, _, err = resolveWith(t, nil, baseEnv(t, map[string]string{"CIRCULA_EXPORT_SPOOL_DIR": "/var/spool/circula"}))
	if err != nil || cfg.ExportSpoolDir != filepath.Clean("/var/spool/circula") {
		t.Errorf("env spool dir %q: %v", cfg.ExportSpoolDir, err)
	}
}

// BR-06: serve prepares the spool directory (private, stale archives
// removed) and refuses to start when it is unusable, instead of exports
// failing, or falling back to memory, later.
func TestExportSpoolPreparedOrStartRefused(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	cfg.StateDir = filepath.Join(dir, "state")
	cfg.CookieSecure = false
	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := newService(st, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Default location, created private.
	if _, _, err := newHandlers(svc, cfg, discardLogger()); err != nil {
		t.Fatal(err)
	}
	spool := filepath.Join(cfg.StateDir, "export-spool")
	fi, err := os.Stat(spool)
	if err != nil || !fi.IsDir() {
		t.Fatalf("spool dir not created: %v", err)
	}
	if !isWindows() && fi.Mode().Perm() != 0o700 {
		t.Errorf("spool dir mode %v, want 0700", fi.Mode().Perm())
	}
	// A stale archive is removed; other files are left alone.
	cfg.ExportSpoolDir = spool
	stale := filepath.Join(spool, "circula-export-123.zip")
	keep := filepath.Join(spool, "notes.txt")
	for _, p := range []string{stale, keep} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := newHandlers(svc, cfg, discardLogger()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale archive kept: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("unrelated file removed: %v", err)
	}
	// A path that is a file stops the start with a named error.
	cfg.ExportSpoolDir = keep
	_, _, err = newHandlers(svc, cfg, discardLogger())
	if err == nil || !strings.Contains(err.Error(), "export-spool-dir") {
		t.Errorf("file as spool dir: %v", err)
	}
	// So does a directory that cannot be written (not checkable as root).
	if !isWindows() && os.Geteuid() != 0 {
		ro := filepath.Join(dir, "ro")
		if err := os.Mkdir(ro, 0o500); err != nil {
			t.Fatal(err)
		}
		cfg.ExportSpoolDir = ro
		if _, _, err := newHandlers(svc, cfg, discardLogger()); err == nil || !strings.Contains(err.Error(), "not writable") {
			t.Errorf("read-only spool dir: %v", err)
		}
	}
}
