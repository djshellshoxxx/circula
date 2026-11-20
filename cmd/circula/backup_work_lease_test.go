package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Deterministically represents an active backup whose staging directory has
// existed for two hours. No wall-clock sleep or production change is needed.
func TestAuditGActiveBackupSurvivesStaleSweep(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := raw.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	}()
	dir := t.TempDir()
	first := filepath.Join(dir, "first.db")
	second := filepath.Join(dir, "second.db")
	_, err = snapshotWith(context.Background(), raw, first, "test", time.Now(), func(ctx context.Context, path string) (LedgerSummary, error) {
		work := filepath.Dir(path)
		old := time.Now().Add(-2 * time.Hour)
		if err := os.Chtimes(work, old, old); err != nil {
			t.Fatal(err)
		}
		if _, err := snapshotWith(ctx, raw, second, "test", time.Now(), nil, snapshotKeys{}); err != nil {
			t.Fatalf("second backup: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("second backup deleted the first active backup's verification file: %v", err)
		}
		return LedgerSummary{Status: "ok"}, nil
	}, snapshotKeys{})
	if err != nil {
		t.Fatalf("first active backup failed: %v", err)
	}
}

// The test binary holds the same OS-backed lease as a production backup. A
// forced child exit intentionally leaves its directory behind as crash evidence.
func TestBackupWorkLeaseChild(t *testing.T) {
	parent := os.Getenv("CIRCULA_BACKUP_LEASE_TEST_ROOT")
	if parent == "" {
		t.Skip("subprocess helper")
	}
	work, cleanup, err := newBackupWork(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := os.WriteFile(filepath.Join(work, "snapshot.db"), []byte("active backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(work, old, old); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, work)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestBackupWorkLeaseProtectsOtherProcessAndReclaimsCrash(t *testing.T) {
	parent := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestBackupWorkLeaseChild$")
	cmd.Env = append(os.Environ(), "CIRCULA_BACKUP_LEASE_TEST_ROOT="+parent)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("child did not publish its work directory: %v", err)
	}
	work := strings.TrimSpace(line)
	if filepath.Dir(work) != parent || !strings.HasPrefix(filepath.Base(work), tempDirPrefix) {
		t.Fatalf("unexpected child work directory: %q", work)
	}
	sweepStaleTemp(parent)
	if got, err := os.ReadFile(filepath.Join(work, "snapshot.db")); err != nil || string(got) != "active backup" {
		t.Fatalf("sweep touched an active process's backup: %q, %v", got, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait() // forced exit releases the OS lock without invoking cleanup
	waited = true
	sweepStaleTemp(parent)
	if _, err := os.Stat(work); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("crashed process's old work directory was not reclaimed: %v", err)
	}
}

func TestBackupWorkSweepRejectsSymlinks(t *testing.T) {
	parent, outside := t.TempDir(), t.TempDir()
	victim := filepath.Join(outside, "private-data")
	if err := os.WriteFile(victim, []byte("must remain unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirLink := filepath.Join(parent, tempDirPrefix+"directory-link")
	if err := os.Symlink(outside, dirLink); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privileges unavailable: %v", err)
		}
		t.Fatal(err)
	}
	work := filepath.Join(parent, tempDirPrefix+"lease-link")
	if err := os.Mkdir(work, 0o700); err != nil {
		t.Fatal(err)
	}
	leaseLink := filepath.Join(work, backupWorkLease)
	if err := os.Symlink(victim, leaseLink); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	for _, path := range []string{work, outside} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	sweepStaleTemp(parent)
	if got, err := os.ReadFile(victim); err != nil || string(got) != "must remain unchanged" {
		t.Errorf("sweep followed a symlink: %q, %v", got, err)
	}
	for _, path := range []string{dirLink, leaseLink} {
		if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("sweep removed or replaced symlink %s: %v", path, err)
		}
	}
}
