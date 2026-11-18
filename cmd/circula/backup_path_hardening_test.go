package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

// AUD4-D01: a backup directory whose name contains SQLite URI metacharacters
// must still have its integrity, schema and ledger checks run on the snapshot
// itself, not on a different file named by the truncated URI.
func TestBackupDirectoryWithURIMetacharacters(t *testing.T) {
	for _, name := range []string{"bk#x", "bk?x", "bk%41x"} {
		if runtime.GOOS == "windows" && strings.Contains(name, "?") {
			continue // not a valid Windows file name
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := testConfigFor(seedDatabase(t, dir))
			cfg.BackupDir = filepath.Join(dir, name)
			mf, path, err := backupTo(context.Background(), cfg, cfg.BackupDir, "scheduled", nil)
			if err != nil {
				t.Fatalf("backup: %v", err)
			}
			if filepath.Dir(path) != cfg.BackupDir {
				t.Fatalf("backup written to %s, want inside %s", path, cfg.BackupDir)
			}
			if mf.Integrity != "ok" || mf.SchemaVersion != store.SchemaVersion() || mf.Ledger.Status != "ok" {
				t.Fatalf("checks did not inspect the snapshot: integrity=%q schema=%d ledger=%+v", mf.Integrity, mf.SchemaVersion, mf.Ledger)
			}
			if _, err := os.Lstat(filepath.Join(dir, "bk")); err == nil {
				t.Fatal("a stray database was created at the truncated URI path")
			}
		})
	}
}

// AUD4-D02: zeroFile must not write through a name that no longer refers
// to the inspected file.
func TestZeroFileRefusesReplacedName(t *testing.T) {
	dir := t.TempDir()
	inspected, victim := filepath.Join(dir, "inspected"), filepath.Join(dir, "victim")
	for _, p := range []string{inspected, victim} {
		if err := os.WriteFile(p, []byte("keep this data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.Lstat(inspected)
	if err != nil {
		t.Fatal(err)
	}
	// The name now refers to another file (as after a swap).
	zeroFile(victim, want)
	if b, _ := os.ReadFile(victim); string(b) != "keep this data" {
		t.Fatalf("replaced file was overwritten: %q", b)
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(dir, "link")
		if err := os.Symlink(victim, link); err != nil {
			t.Fatal(err)
		}
		vi, _ := os.Stat(victim)
		zeroFile(link, vi) // even with the target's own identity, the symlink is not followed
		if b, _ := os.ReadFile(victim); string(b) != "keep this data" {
			t.Fatalf("symlink target was overwritten: %q", b)
		}
	}
	zeroFile(inspected, want)
	if b, _ := os.ReadFile(inspected); !bytes.Equal(b, make([]byte, len("keep this data"))) {
		t.Fatalf("the inspected file was not overwritten: %q", b)
	}
}

// AUD4-D02: a file with another hard link is unlinked but never zeroed, so
// cleanup cannot destroy data reachable through a name outside the work area.
func TestSecureRemoveLeavesHardLinkedDataIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("link counts are not read on Windows")
	}
	dir := t.TempDir()
	victim, planted := filepath.Join(dir, "victim"), filepath.Join(dir, "work-entry")
	if err := os.WriteFile(victim, []byte("keep this data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(victim, planted); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	secureRemove(planted)
	if _, err := os.Lstat(planted); !os.IsNotExist(err) {
		t.Fatalf("planted name not removed: %v", err)
	}
	if b, _ := os.ReadFile(victim); string(b) != "keep this data" {
		t.Fatalf("hard-linked data was overwritten: %q", b)
	}
}

// AUD4-D02: a directory replaced by a symlink is unlinked, never descended.
func TestSecureRemoveAllDoesNotFollowDirectorySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(outside, "keep")
	if err := os.WriteFile(keep, []byte("keep this data"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, tempDirPrefix+"swapped")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	secureRemoveAll(link)
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("symlink not removed: %v", err)
	}
	if b, err := os.ReadFile(keep); err != nil || string(b) != "keep this data" {
		t.Fatalf("data behind the symlink was touched: %q %v", b, err)
	}
}

// AUD4-D02: leftover work directories are swept only from a directory no
// other user can write, and staging refuses a shared non-sticky directory.
func TestBackupStagingRequiresPrivateOrStickyParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	parent := t.TempDir()
	orphan := filepath.Join(parent, tempDirPrefix+"orphan")
	if err := os.Mkdir(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(orphan, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o770); err != nil { // group-writable, not sticky
		t.Fatal(err)
	}
	sweepStaleTemp(parent)
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("orphan swept from a shared directory: %v", err)
	}
	if _, _, err := newBackupWork(parent); err == nil {
		t.Fatal("private work created in a shared non-sticky directory")
	}
	if err := os.Chmod(parent, os.ModeSticky|0o770); err != nil { // sticky: entries cannot be renamed by others
		t.Fatal(err)
	}
	work, cleanup, err := newBackupWork(parent)
	if err != nil {
		t.Fatalf("sticky shared directory refused: %v", err)
	}
	cleanup()
	if _, err := os.Stat(work); !os.IsNotExist(err) {
		t.Fatalf("work not cleaned: %v", err)
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	sweepStaleTemp(parent)
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan not swept from a private directory: %v", err)
	}
}

// AUD4-D02 (review G03): if the leased work directory is renamed and its old
// name replaced with a symlink, cleanup must not follow the name into the
// symlink target.
func TestLeasedWorkCleanupRefusesSubstitutedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	parent := t.TempDir()
	unrelated := filepath.Join(t.TempDir(), "unrelated")
	if err := os.Mkdir(unrelated, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(unrelated, "keep")
	if err := os.WriteFile(keep, []byte("keep this data"), 0o600); err != nil {
		t.Fatal(err)
	}
	work, cleanup, err := newBackupWork(parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(work, work+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unrelated, work); err != nil {
		t.Fatal(err)
	}
	cleanup()
	if b, err := os.ReadFile(keep); err != nil || string(b) != "keep this data" {
		t.Fatalf("cleanup followed the substituted name: %q %v", b, err)
	}
}

// AUD4-D02 (review G03): the owner of the destination directory can rename
// any entry in it, sticky or not, so it must be this user or root.
func TestBackupStagingRefusesForeignOwnedParent(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() != 0 {
		t.Skip("needs root on Unix to create a directory owned by another user")
	}
	parent := t.TempDir()
	if err := os.Chown(parent, 65534, 65534); err != nil {
		t.Skipf("chown unavailable: %v", err)
	}
	if err := os.Chmod(parent, os.ModeSticky|0o777); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newBackupWork(parent); err == nil {
		t.Fatal("private work created in a sticky directory owned by another user")
	}
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newBackupWork(parent); err == nil {
		t.Fatal("private work created in a directory owned by another user")
	}
}
