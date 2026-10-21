package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// seedDatabase creates a migrated database with one network and returns its path.
func seedDatabase(t *testing.T, dir string) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(dir, "circula.db")
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: "founder01", Password: "correct horse battery"})
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	if _, _, err := svc.CreateNetwork(ctx, actor, service.CreateNetworkInput{Name: "Test", Slug: "test-net", CreatorDisplayName: "Founder"}); err != nil {
		t.Fatal(err)
	}
	return path
}

func testConfigFor(db string) *Config {
	return &Config{Database: db, BackupDir: filepath.Join(filepath.Dir(db), "backups"), BackupRetainDaily: 30, BackupRetainMonthly: 12}
}

func TestSnapshotWritesVerifiedManifestAndRestores(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	ctx := context.Background()

	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "snap.db")
	mf, err := snapshot(ctx, raw, dest, "manual", time.Now(), func(ctx context.Context, p string) (LedgerSummary, error) {
		return verifyFile(ctx, p, cfg)
	}, nil)
	_ = raw.Close()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if mf.Ledger.Status != "ok" || mf.Ledger.Networks != 1 || mf.Integrity != "ok" {
		t.Errorf("manifest = %+v", mf)
	}
	if mf.SchemaVersion != store.SchemaVersion() {
		t.Errorf("manifest schema = %d", mf.SchemaVersion)
	}
	m, err := readManifest(dest + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupFile(dest, m); err != nil {
		t.Fatalf("checksum: %v", err)
	}
	if _, err := os.Stat(dest + ".partial"); err == nil {
		t.Error("partial file left behind")
	}

	// Tampering with the backup must be detected before restore.
	f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte("tamper"))
	_ = f.Close()
	if err := verifyBackupFile(dest, m); err == nil {
		t.Error("tampered backup passed checksum")
	}
}

func TestRestoreRefusesOverwriteWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	ctx := context.Background()
	cfg := testConfigFor(db)
	mf, path, err := backupTo(ctx, cfg, dir, "manual", nil)
	if err != nil {
		t.Fatalf("backupTo: %v", err)
	}
	if mf.Ledger.Status != "ok" {
		t.Fatalf("ledger = %+v", mf.Ledger)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", db, path}, &out, &errOut)
	if code == exitOK || !strings.Contains(errOut.String(), "-overwrite") {
		t.Fatalf("restore over existing db without -overwrite: code %d, %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"restore", "-database", db, "-overwrite", path}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("restore with -overwrite: code %d, stderr %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "pre-restore backup") || !strings.Contains(out.String(), "data written after") {
		t.Errorf("restore output lacks pre-restore backup or data window: %s", out.String())
	}
	recs, _ := filepath.Glob(filepath.Join(dir, "backups", "restore-*.json"))
	if len(recs) != 1 {
		t.Errorf("restore records = %v", recs)
	}
	// OPS-020: the restore is also an event in the audit chain.
	rdb, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rdb.Close() }()
	var n int
	if err := rdb.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = 'database.restored'`).Scan(&n); err != nil || n < 1 {
		t.Errorf("database.restored events = %d err=%v", n, err)
	}
}

func TestPruneKeepsDailyAndNewestPerMonth(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	var times []time.Time
	// 40 daily backups going back into the past, then one per older month.
	for i := 0; i < 40; i++ {
		times = append(times, base.AddDate(0, 0, -i))
	}
	times = append(times, time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC), time.Date(2026, 3, 9, 3, 0, 0, 0, time.UTC))
	times = append(times, time.Date(2025, 12, 1, 3, 0, 0, 0, time.UTC))
	for _, ts := range times {
		name := filepath.Join(dir, backupFileName(ts))
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name+".manifest.json", []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := pruneBackups(dir, 30, 12)
	if err != nil {
		t.Fatal(err)
	}
	// Daily: 30 newest kept. Older: 10 newer daily ones dropped except one per month.
	// March 2026 keeps 2026-03-09 (newest in month), and December 2025 is kept as its own month.
	left, _ := listBackups(dir)
	names := map[string]bool{}
	for _, e := range left {
		names[filepath.Base(e.path)] = true
	}
	if !names[backupFileName(time.Date(2026, 3, 9, 3, 0, 0, 0, time.UTC))] {
		t.Error("newest March backup pruned")
	}
	if names[backupFileName(time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC))] {
		t.Error("older March backup kept")
	}
	if !names[backupFileName(time.Date(2025, 12, 1, 3, 0, 0, 0, time.UTC))] {
		t.Error("December 2025 backup pruned")
	}
	if len(removed) == 0 {
		t.Error("nothing removed")
	}
	if _, err := os.Stat(filepath.Join(dir, backupFileName(time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC))+".manifest.json")); err == nil {
		t.Error("manifest of pruned backup kept")
	}
}

func TestLockRefusesSecondHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "circula.db.lock")
	first, err := acquireLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Release() }()
	if _, err := acquireLock(path); err != ErrLocked {
		t.Fatalf("second lock err = %v, want ErrLocked", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := acquireLock(path)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	_ = second.Release()
}

func TestMigrateIsNoOpWhenCurrent(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	var out, errOut bytes.Buffer
	if code := run([]string{"migrate", "-database", db}, &out, &errOut); code != exitOK {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "nothing to do") {
		t.Errorf("output = %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(db), "backups")); err == nil {
		entries, _ := os.ReadDir(filepath.Join(filepath.Dir(db), "backups"))
		if len(entries) != 0 {
			t.Error("migrate with nothing pending took a backup")
		}
	}
}

func TestVerifyLedgerCleanDatabase(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	var out, errOut bytes.Buffer
	if code := run([]string{"verify-ledger", "-database", db}, &out, &errOut); code != exitOK {
		t.Fatalf("code %d: %s %s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "test-net: ok") {
		t.Errorf("output = %q", out.String())
	}
}
