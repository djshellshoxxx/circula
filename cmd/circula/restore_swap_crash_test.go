package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const restoreCrashExit = 87

// TestRestoreSwapCrashChild is invoked only by this file's subprocess harness.
func TestRestoreSwapCrashChild(t *testing.T) {
	dir := os.Getenv("CIRCULA_RESTORE_CRASH_DIR")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	stage := os.Getenv("CIRCULA_RESTORE_CRASH_STAGE")
	if stage == "prepare-wal" {
		path := seedDatabase(t, dir)
		st := tu.Must(store.Open(context.Background(), path))
		if _, err := st.DB().Exec(`PRAGMA wal_autocheckpoint=0`); err != nil {
			t.Fatal(err)
		}
		svc := tu.Must(service.New(st, service.Config{Argon: qa28Argon}))
		tu.Must(svc.RegisterPerson(context.Background(), service.RegisterInput{Login: "wal-only-person", Password: "correct horse battery"}))
		os.Exit(restoreCrashExit) // committed WAL, no deferred checkpoint/close
	}
	path := filepath.Join(dir, "circula.db")
	err := swapDatabaseSteps(context.Background(), path, filepath.Join(dir, "incoming.db"), true, func(ctx context.Context) error {
		st, err := openStore(ctx, path, testConfigFor(path))
		if err != nil {
			return err
		}
		return st.Close()
	}, func(point string) {
		if point == stage {
			os.Exit(restoreCrashExit)
		}
	})
	t.Fatalf("crash stage %q not reached: %v", stage, err)
}

func runRestoreCrashChild(t *testing.T, dir, stage string) {
	t.Helper()
	exe := tu.Must(os.Executable())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestRestoreSwapCrashChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), "CIRCULA_RESTORE_CRASH_DIR="+dir, "CIRCULA_RESTORE_CRASH_STAGE="+stage)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != restoreCrashExit {
		t.Fatalf("child %s: %v\n%s", stage, err, out)
	}
}

// RESTORE-01 / OPS-020: every interruptible rename transition retains a guard
// that prevents SQLite from creating an empty database or opening mixed WALs.
func TestRestoreSwapCrashRefusesStartupAndPreservesFiles(t *testing.T) {
	source := t.TempDir()
	runRestoreCrashChild(t, source, "prepare-wal")
	old := map[string][]byte{}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		old[suffix] = tu.Must(os.ReadFile(filepath.Join(source, "circula.db") + suffix))
		if len(old[suffix]) == 0 {
			t.Fatalf("empty fixture %q", suffix)
		}
	}
	incoming := tu.Must(os.ReadFile(seedDatabase(t, t.TempDir())))
	for _, stage := range []string{"prepared", "old-database", "old-wal", "old-shm", "installed", "verified", "cleaned"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "circula.db")
			for suffix, data := range old {
				if err := os.WriteFile(path+suffix, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "incoming.db"), incoming, 0o600); err != nil {
				t.Fatal(err)
			}
			runRestoreCrashChild(t, dir, stage)
			if _, err := os.Stat(path + restoreMarkerSuffix); err != nil {
				t.Fatal("crash lost restore marker", err)
			}
			preserved := func(data []byte, paths ...string) bool {
				for _, p := range paths {
					got, err := os.ReadFile(p)
					if err == nil && bytes.Equal(got, data) {
						return true
					}
				}
				return false
			}
			if stage != "cleaned" {
				for suffix, data := range old {
					if !preserved(data, path+suffix, path+".pre-restore-swap"+suffix) {
						t.Fatalf("old %q bytes lost at %s", suffix, stage)
					}
				}
			}
			if !preserved(incoming, filepath.Join(dir, "incoming.db"), path) {
				t.Fatal("verified candidate bytes lost")
			}
			before := restoreCrashFiles(t, dir)
			assertRestoreBlocked(t, path)
			after := restoreCrashFiles(t, dir)
			if len(before) != len(after) {
				t.Fatalf("startup changed retained files: before=%v after=%v", before, after)
			}
			for p, b := range before {
				if !bytes.Equal(after[p], b) {
					t.Fatalf("startup mutated %s", p)
				}
			}
		})
	}
}

func assertRestoreBlocked(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	cfg := testConfigFor(path)
	require := func(err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), "interrupted restore detected") {
			t.Fatalf("open did not refuse interrupted restore: %v", err)
		}
	}
	_, err := inspectSchema(ctx, path)
	require(err)
	st, err := openStore(ctx, path, cfg)
	if st != nil {
		_ = st.Close()
	}
	require(err)
	raw, err := openRaw(path)
	if raw != nil {
		_ = raw.Close()
	}
	require(err)
	cfg.Listen = "127.0.0.1:0"
	cfg.PublicBaseURL = "https://x.test"
	timed, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	require(runServe(timed, cfg, slog.New(slog.NewTextHandler(io.Discard, nil))))
	var out, stderr bytes.Buffer
	if code := run([]string{"migrate", "-database", path}, &out, &stderr); code != exitFailure || !strings.Contains(stderr.String(), "interrupted restore detected") {
		t.Fatalf("migrate allowed recovery state: %d %s %s", code, out.String(), stderr.String())
	}
}

func restoreCrashFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	entries := tu.Must(os.ReadDir(dir))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}
		out[entry.Name()] = tu.Must(os.ReadFile(filepath.Join(dir, entry.Name())))
	}
	return out
}

func TestRestoreRefusesLegacySavedFilesAndCorruptMarker(t *testing.T) {
	for _, suffix := range []string{restoreMarkerSuffix, ".pre-restore-swap", ".pre-restore-swap-wal", ".pre-restore-swap-shm", ".pre-restore-swap-journal"} {
		t.Run(suffix, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "missing.db")
			sentinel := []byte("preserve: unfinished prior restore")
			if err := os.WriteFile(path+suffix, sentinel, 0o600); err != nil {
				t.Fatal(err)
			}
			assertRestoreBlocked(t, path)
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("empty database created: %v", err)
			}
			if got := tu.Must(os.ReadFile(path + suffix)); !bytes.Equal(got, sentinel) {
				t.Fatal("recovery evidence changed")
			}
		})
	}
}

// A rollback failure keeps its durable guard and every surviving source;
// another restore must not overwrite an earlier saved database.
func TestRestoreSwapRefusesExistingRecoveryFiles(t *testing.T) {
	dir := t.TempDir()
	path := seedDatabase(t, dir)
	incoming := seedDatabase(t, t.TempDir())
	sentinel := []byte("previous recovery copy")
	if err := os.WriteFile(path+".pre-restore-swap", sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	before := tu.Must(os.ReadFile(path))
	candidate := tu.Must(os.ReadFile(incoming))
	if err := swapDatabase(context.Background(), path, incoming, true, func(context.Context) error { return nil }); err == nil {
		t.Fatal("overwrote unfinished restore")
	}
	if !bytes.Equal(tu.Must(os.ReadFile(path)), before) || !bytes.Equal(tu.Must(os.ReadFile(incoming)), candidate) || !bytes.Equal(tu.Must(os.ReadFile(path+".pre-restore-swap")), sentinel) {
		t.Fatal("refused restore changed evidence")
	}
}

func TestRestoreSwapValidationBypassIsScopedAndExpires(t *testing.T) {
	dir := t.TempDir()
	path := seedDatabase(t, dir)
	incoming := seedDatabase(t, t.TempDir())
	other := filepath.Join(dir, "unrelated.db")
	if err := os.WriteFile(other+restoreMarkerSuffix, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	var saved context.Context
	err := swapDatabase(context.Background(), path, incoming, true, func(ctx context.Context) error {
		saved = ctx
		state, err := inspectSchema(ctx, path)
		if err != nil || !state.Exists {
			return errors.New("own validation did not open candidate")
		}
		if _, err := inspectSchema(ctx, other); err == nil || !strings.Contains(err.Error(), "interrupted restore detected") {
			return errors.New("validation bypass escaped to another database")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+restoreMarkerSuffix, []byte("later interruption"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectSchema(saved, path); err == nil || !strings.Contains(err.Error(), "interrupted restore detected") {
		t.Fatalf("validation permit survived its callback: %v", err)
	}
}

func TestRestoreSwapRollbackFailureRetainsRecoveryGuard(t *testing.T) {
	dir := t.TempDir()
	path := seedDatabase(t, dir)
	incoming := seedDatabase(t, t.TempDir())
	before := tu.Must(os.ReadFile(path))
	err := swapDatabase(context.Background(), path, incoming, true, func(context.Context) error {
		// Simulate a filesystem obstruction while the new candidate is being
		// validated. A failed cleanup must not discard the old saved database.
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(path, "obstruction"), []byte("preserve"), 0o600); err != nil {
			return err
		}
		return errors.New("injected validation failure")
	})
	if err == nil || !strings.Contains(err.Error(), "interrupted restore detected") {
		t.Fatalf("rollback did not report recovery required: %v", err)
	}
	if !bytes.Equal(tu.Must(os.ReadFile(path+".pre-restore-swap")), before) {
		t.Fatal("failed rollback damaged saved database")
	}
	if _, err := os.Stat(path + restoreMarkerSuffix); err != nil {
		t.Fatal("failed rollback removed guard", err)
	}
	assertRestoreBlocked(t, path)
}
