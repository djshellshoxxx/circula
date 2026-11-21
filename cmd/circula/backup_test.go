package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
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
	if mf.Ledger.Status != "ok" || mf.Ledger.Networks != 1 || mf.Ledger.PlatformAudit != "ok" || mf.Integrity != "ok" {
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
	code := run([]string{"restore", "-allow-unsigned", "-database", db, path}, &out, &errOut)
	if code == exitOK || !strings.Contains(errOut.String(), "-overwrite") {
		t.Fatalf("restore over existing db without -overwrite: code %d, %s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"restore", "-allow-unsigned", "-database", db, "-overwrite", path}, &out, &errOut)
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

// OPS-017: backups taken within one second need distinct names. Both naming
// generations remain discoverable, and retention compares their full instants.
func TestBackupNamesPreserveSubsecondTimeAndLegacyRetention(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 11, 16, 8, 30, 0, 0, time.FixedZone("local", -8*60*60))
	first, second := base.Add(100*time.Nanosecond), base.Add(200*time.Nanosecond)
	for _, name := range []func(time.Time) string{backupFileName, encryptedBackupFileName} {
		if name(first) == name(second) {
			t.Fatalf("same-second backups collide: %s", name(first))
		}
		if name(first) != name(first.UTC()) {
			t.Fatal("backup name depends on the caller's time zone")
		}
	}
	legacyPlain := "circula-20261116T163000Z.db"
	legacyEncrypted := "circula-20261016T163000Z.db.enc"
	names := []string{legacyPlain, legacyEncrypted, backupFileName(first), encryptedBackupFileName(second)}
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("backup"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".manifest.json", []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := listBackups(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{names[3], names[2], names[0], names[1]}
	if len(entries) != len(want) {
		t.Fatalf("listed %d backups, want %d", len(entries), len(want))
	}
	for i, name := range want {
		if filepath.Base(entries[i].path) != name {
			t.Errorf("backup %d = %s, want %s", i, entries[i].path, name)
		}
	}
	if !entries[0].at.Equal(second) || !entries[1].at.Equal(first) {
		t.Fatal("listing discarded the subsecond timestamp")
	}
	removed, err := pruneBackups(dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed %d backups, want 2", len(removed))
	}
	for _, name := range []string{names[0], names[2]} {
		for _, suffix := range []string{"", ".manifest.json"} {
			if _, err := os.Stat(filepath.Join(dir, name+suffix)); !os.IsNotExist(err) {
				t.Errorf("older backup or manifest still present: %s%s, %v", name, suffix, err)
			}
		}
	}
	for _, name := range []string{names[1], names[3]} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("newest monthly backup was removed: %s, %v", name, err)
		}
	}
}

// A3-F10, OPS-018: a backup whose name is dated after now (written while the
// clock ran ahead, or a misplaced file) is kept with its manifest and anchor,
// but it takes neither a daily nor a monthly slot: real backups are removed
// only as the configured counts require. Scheduling ignores such a file in the
// same way (TestJobUntilNextBackupIgnoresFutureFiles).
func TestPruneKeepsFutureDatedBackupsOutOfCounts(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 30, 0, 123456789, time.UTC)
	ahead := now.AddDate(50, 0, 0)
	type backup struct {
		at   time.Time
		kept bool
	}
	date := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 3, 0, 0, 0, time.UTC) }
	// The default counts (30 daily, 12 monthly; DEC-059) over 40 daily
	// backups, a second September backup and one backup per month back to
	// November 2025, the 13th month.
	var defaults []backup
	for i := 0; i < 40; i++ {
		defaults = append(defaults, backup{now.Add(-time.Hour - time.Duration(i)*24*time.Hour), i < 30})
	}
	defaults = append(defaults, backup{date(2026, 9, 20), true}, backup{date(2026, 9, 5), false})
	for m := time.August; m >= time.January; m-- {
		defaults = append(defaults, backup{date(2026, m, 15), true})
	}
	defaults = append(defaults, backup{date(2025, 12, 15), true}, backup{date(2025, 11, 15), false})
	for _, tc := range []struct {
		name           string
		daily, monthly int
		future         []time.Time
		past           []backup
	}{
		{name: "50 years ahead, default counts", daily: 30, monthly: 12, future: []time.Time{ahead}, past: defaults},
		{name: "an hour ahead in the current month", daily: 1, monthly: 1, future: []time.Time{now.Add(time.Hour)},
			past: []backup{{now.Add(-time.Minute), true}, {now.Add(-24 * time.Hour), false}}},
		{name: "backup at now counts", daily: 1, future: []time.Time{ahead},
			past: []backup{{now, true}, {now.Add(-time.Hour), false}}},
		{name: "several future-dated files", daily: 1, monthly: 1, future: []time.Time{ahead, now.AddDate(0, 0, 1), now.Add(time.Nanosecond)},
			past: []backup{{now.Add(-time.Hour), true}, {now.Add(-48 * time.Hour), false}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			files := func(name string) []string { return []string{name, name + ".manifest.json", name + ".anchor.json"} }
			write := func(name string) []byte {
				content := []byte("backup " + name)
				for _, p := range files(name) {
					if err := os.WriteFile(filepath.Join(dir, p), content, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return content
			}
			future := map[string][]byte{}
			for i, at := range tc.future {
				name := backupFileName(at)
				if i%2 == 0 {
					name = encryptedBackupFileName(at)
				}
				future[name] = write(name)
			}
			var wantRemoved []string
			for _, b := range tc.past {
				write(backupFileName(b.at))
				if !b.kept {
					wantRemoved = append(wantRemoved, backupFileName(b.at))
				}
			}
			removed, err := pruneBackupsAt(dir, tc.daily, tc.monthly, now)
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(removed)
			slices.Sort(wantRemoved)
			if !slices.Equal(removed, wantRemoved) {
				t.Errorf("removed %v, want %v", removed, wantRemoved)
			}
			for name, content := range future {
				for _, p := range files(name) {
					if got, err := os.ReadFile(filepath.Join(dir, p)); err != nil || !bytes.Equal(got, content) {
						t.Errorf("future-dated %s changed: %q, %v", p, got, err)
					}
				}
			}
			for _, b := range tc.past {
				for _, p := range files(backupFileName(b.at)) {
					if _, err := os.Stat(filepath.Join(dir, p)); (err == nil) != b.kept {
						t.Errorf("%s: present=%v, want %v", p, err == nil, b.kept)
					}
				}
			}
		})
	}
}

func TestSnapshotRefusesExistingSubsecondBackup(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		dir := t.TempDir()
		at := time.Date(2026, 11, 16, 16, 30, 0, 123456789, time.UTC)
		name := backupFileName(at)
		keys := snapshotKeys{}
		if encrypted {
			name = encryptedBackupFileName(at)
			keys.Encrypt = bytes.Repeat([]byte{1}, 32)
		}
		path := filepath.Join(dir, name)
		original := []byte("existing backup must survive")
		if err := os.WriteFile(path, original, 0o600); err != nil {
			t.Fatal(err)
		}
		// A nil database also proves refusal occurs before snapshot work starts.
		if _, err := snapshotWith(context.Background(), nil, path, "test", at, nil, keys); err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("existing backup not refused: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, original) {
			t.Fatalf("existing backup changed: %q, %v", got, err)
		}
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
