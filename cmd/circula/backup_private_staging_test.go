package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// NEW-01: a copy must never follow a planted symlink or truncate a file that
// appeared at the destination after its caller checked or removed the path.
func TestCopyFileSyncRefusesExistingDestination(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		name := "regular file"
		if symlink {
			name = "symlink"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			src, dst, victim := filepath.Join(dir, "source"), filepath.Join(dir, "copy"), filepath.Join(dir, "victim")
			if err := os.WriteFile(src, []byte("replacement data"), 0o600); err != nil {
				t.Fatal(err)
			}
			protected := dst
			if symlink {
				protected = victim
			}
			const original = "must not be overwritten"
			if err := os.WriteFile(protected, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			if symlink {
				if err := os.Symlink(victim, dst); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink privileges unavailable: %v", err)
					}
					t.Fatal(err)
				}
			}
			if err := copyFileSync(src, dst); !errors.Is(err, os.ErrExist) {
				t.Errorf("copy to an existing destination: %v, want file-exists refusal", err)
			}
			if got, err := os.ReadFile(protected); err != nil || string(got) != original {
				t.Errorf("existing data was modified: %q, err=%v", got, err)
			}
		})
	}
}

// Snapshot plaintext and the verification copy must stay in a private,
// unpredictable directory, even when the final backup is not encrypted.
// Legacy predictable temporary names are not owned by this invocation.
func TestSnapshotUsesPrivateStaging(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	cfg := testConfigFor(db)
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	for _, encrypted := range []bool{false, true} {
		for _, failVerify := range []bool{false, true} {
			name := "plaintext"
			if encrypted {
				name = "encrypted"
			}
			if failVerify {
				name += "/verification failure"
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				dest := filepath.Join(dir, "backup.db")
				legacy := []string{dest + ".partial", dest + ".partial.verify"}
				for _, path := range legacy {
					if err := os.WriteFile(path, []byte("unowned file"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				keys := snapshotKeys{Sign: []byte("backup signing key")}
				if encrypted {
					keys.Encrypt = make([]byte, 32)
				}
				injected := errors.New("verification failed")
				var work string
				mf, err := snapshotWith(context.Background(), raw, dest, "test", time.Now(), func(ctx context.Context, path string) (LedgerSummary, error) {
					work = filepath.Dir(path)
					if work == dir || filepath.Dir(work) != dir || !strings.HasPrefix(filepath.Base(work), tempDirPrefix) {
						t.Errorf("verification file outside private staging directory: %s", path)
					}
					for _, item := range []struct {
						path string
						mode os.FileMode
					}{{work, 0o700}, {path, 0o600}} {
						info, err := os.Stat(item.path)
						if err != nil {
							t.Fatal(err)
						}
						if runtime.GOOS != "windows" && info.Mode().Perm() != item.mode {
							t.Errorf("permissions on %s: %o, want %o", item.path, info.Mode().Perm(), item.mode)
						}
					}
					if failVerify {
						return LedgerSummary{}, injected
					}
					return verifyFile(ctx, path, cfg)
				}, keys)
				if failVerify {
					if !errors.Is(err, injected) {
						t.Fatalf("snapshot failure: %v, want verification error", err)
					}
					for _, path := range []string{dest, dest + ".manifest.json"} {
						if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
							t.Errorf("failed snapshot published %s: %v", path, err)
						}
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					if err := verifyBackupFile(dest, mf); err != nil {
						t.Fatal(err)
					}
					if authenticated, err := checkManifestAuth(mf, keys.Sign); !authenticated || err != nil {
						t.Fatalf("signed backup did not authenticate: %v", err)
					}
				}
				for _, path := range legacy {
					if got, err := os.ReadFile(path); err != nil || string(got) != "unowned file" {
						t.Errorf("snapshot changed an unowned temporary path %s: %q, %v", path, got, err)
					}
				}
				if work == "" {
					t.Fatal("verification callback never ran")
				}
				if work != dir {
					if _, err := os.Stat(work); !errors.Is(err, os.ErrNotExist) {
						t.Errorf("private staging directory was not removed: %v", err)
					}
				}
			})
		}
	}
}

func TestVerifyCopyLeavesPredictablePathUntouched(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	legacy := db + ".verify"
	if err := os.WriteFile(legacy, []byte("unowned file"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _, err := fileSHA256(db)
	if err != nil {
		t.Fatal(err)
	}
	checks, err := verifyCopy(context.Background(), db, testConfigFor(db))
	if err != nil || len(checks) != 1 || !checks[0].OK {
		t.Fatalf("verify copy: %+v, %v", checks, err)
	}
	if got, err := os.ReadFile(legacy); err != nil || string(got) != "unowned file" {
		t.Errorf("verification changed an unowned path: %q, %v", got, err)
	}
	if after, _, err := fileSHA256(db); err != nil || after != before {
		t.Errorf("verification changed source bytes: %v", err)
	}
	if work, err := filepath.Glob(filepath.Join(filepath.Dir(db), tempDirPrefix+"*")); err != nil || len(work) != 0 {
		t.Errorf("verification left private staging files: %v, %v", work, err)
	}
}

func TestPlainSnapshotSweepsOnlyStaleBackupWork(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	dir := t.TempDir()
	stale, fresh, unrelated := filepath.Join(dir, tempDirPrefix+"stale"), filepath.Join(dir, tempDirPrefix+"fresh"), filepath.Join(dir, "operator-files")
	for _, path := range []string{stale, fresh, unrelated} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "marker"), []byte("retained data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	for _, path := range []string{stale, unrelated} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := snapshot(context.Background(), raw, filepath.Join(dir, "backup.db"), "test", time.Now(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old backup work directory was not removed: %v", err)
	}
	for _, path := range []string{fresh, unrelated} {
		if got, err := os.ReadFile(filepath.Join(path, "marker")); err != nil || string(got) != "retained data" {
			t.Errorf("sweep changed protected directory %s: %q, %v", path, got, err)
		}
	}
}

func TestCopyFileSyncRemovesOnlyItsFailedCopy(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "copy")
	// Opening a directory succeeds, but reading it as a regular source fails.
	if err := copyFileSync(dir, dst); err == nil {
		t.Fatal("copy unexpectedly read a directory")
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("failed copy left its newly created destination: %v", err)
	}
}
