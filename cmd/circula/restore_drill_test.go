package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// OPS-021: scripts/restore-drill.sh restores a backup into a scratch directory,
// verifies the ledger and appends a record to the drill log.
func TestRestoreDrillScript(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	var out, errOut bytes.Buffer
	if code := run([]string{"backup", "-database", db, filepath.Join(dir, "drill-src.bak")}, &out, &errOut); code != exitOK {
		t.Fatalf("backup: %d %s", code, errOut.String())
	}
	files := []string{filepath.Join(dir, "drill-src.bak")}
	if len(files) == 0 {
		t.Fatalf("no backup file in %s (%s)", dir, out.String())
	}
	bin := filepath.Join(t.TempDir(), "circula")
	build := exec.Command("go", "build", "-o", bin, ".")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, b)
	}
	script, _ := filepath.Abs(filepath.Join("..", "..", "scripts", "restore-drill.sh"))
	logf := filepath.Join(t.TempDir(), "drills.jsonl")
	cmd := exec.Command("sh", script, files[0], logf)
	cmd.Env = append(os.Environ(), "CIRCULA_BIN="+bin)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("drill failed: %v\n%s", err, b)
	}
	rec, err := os.ReadFile(logf)
	if err != nil || !strings.Contains(string(rec), `"restore":"ok"`) || !strings.Contains(string(rec), `"ledger":"ok"`) {
		t.Fatalf("drill log = %q err=%v", rec, err)
	}
	// a corrupt backup fails the drill and is recorded as failed
	bad := filepath.Join(t.TempDir(), "bad.db")
	if err := os.WriteFile(bad, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("sh", script, bad, logf)
	cmd.Env = append(os.Environ(), "CIRCULA_BIN="+bin)
	if _, err := cmd.CombinedOutput(); err == nil {
		t.Fatal("corrupt backup passed the drill")
	}
	rec, _ = os.ReadFile(logf)
	if !strings.Contains(string(rec), `"restore":"failed"`) {
		t.Errorf("failed drill not recorded: %q", rec)
	}
}
