package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// SEC-021: pruning removes log files older than the retention and nothing else.
func TestPruneLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	mk := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	day := 24 * time.Hour
	mk("old.log", 91*day)
	mk("old.log.1", 200*day)
	mk("old.jsonl", 120*day)
	mk("edge.log", 90*day)
	mk("new.log", day)
	mk("notes.txt", 400*day)
	if err := os.Mkdir(filepath.Join(dir, "sub.log"), 0o700); err != nil {
		t.Fatal(err)
	}
	removed, err := pruneLogs(dir, 90*day, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 3 {
		t.Fatalf("removed %v", removed)
	}
	for _, keep := range []string{"edge.log", "new.log", "notes.txt", "sub.log"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("%s was removed", keep)
		}
	}
}
