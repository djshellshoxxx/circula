//go:build !windows

package messaging

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func privateFixtureDirectory(t testing.TB) string {
	t.Helper()
	return t.TempDir()
}

func assertPrivateBackup(t testing.TB, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		t.Fatalf("backup is not private: %v, %v", info, err)
	}
}

func TestPrivateStoragePreservesExistingUnixBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsafe.sqlite")
	content := []byte("existing caller data")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path, func(context.Context, *sql.Tx, Actor, string, string) error { return nil }); err == nil {
		s.Close()
		t.Fatal("nonprivate existing file accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(content) {
		t.Fatalf("rejected data changed: %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatalf("rejected permissions changed: %v, %v", info, err)
	}
	if err := CreatePrivateDirectory(dir); err == nil {
		t.Fatal("existing directory silently accepted")
	}
	private := filepath.Join(dir, "new")
	if err := CreatePrivateDirectory(private); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(private)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		t.Fatalf("new directory is not private: %v, %v", info, err)
	}
}
