package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

func TestWebKeyCreatedPersistedAndPrivate(t *testing.T) {
	dir := t.TempDir()
	k1, err := loadOrCreateWebKey(dir)
	if err != nil || len(k1) != webKeyLen {
		t.Fatalf("create: %v len %d", err, len(k1))
	}
	k2, err := loadOrCreateWebKey(dir)
	if err != nil || !bytes.Equal(k1, k2) {
		t.Fatalf("second start: %v equal=%v", err, bytes.Equal(k1, k2))
	}
	path := filepath.Join(dir, webKeyFile)
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
			t.Errorf("key mode %o", fi.Mode().Perm())
		}
	}
	for _, n := range listDir(t, dir) {
		if n != webKeyFile {
			t.Errorf("stray file %s", n)
		}
	}
}

func TestWebKeyStateDirCreatedPrivate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "deep")
	if _, err := loadOrCreateWebKey(dir); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o700 {
			t.Errorf("state dir mode %o", fi.Mode().Perm())
		}
	}
}

func TestWebKeyRefusesOpenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions")
	}
	dir := t.TempDir()
	if _, err := loadOrCreateWebKey(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, webKeyFile)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadOrCreateWebKey(dir); err == nil || !strings.Contains(err.Error(), "readable by group or others") {
		t.Fatalf("open perms: %v", err)
	}
}

func TestWebKeyRefusesCorruptOrWrongSize(t *testing.T) {
	for name, content := range map[string]string{
		"empty":     "",
		"garbage":   "not base64 !!!\n",
		"too short": "AAAA\n",
		"too long":  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\n",
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, webKeyFile)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadOrCreateWebKey(dir); err == nil || !strings.Contains(err.Error(), "corrupt") {
			t.Errorf("%s: %v", name, err)
		}
		// Refused, not silently replaced.
		if b, _ := os.ReadFile(path); string(b) != content {
			t.Errorf("%s: corrupt key file was overwritten", name)
		}
	}
}

func TestWebKeyRefusesNonRegularFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "elsewhere")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, webKeyFile)); err != nil {
		t.Skip("symlinks unavailable")
	}
	if _, err := loadOrCreateWebKey(dir); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("symlinked key file: %v", err)
	}
}

func TestWebKeyConcurrentFirstStart(t *testing.T) {
	dir := t.TempDir()
	const n = 16
	keys := make([][]byte, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); keys[i], errs[i] = loadOrCreateWebKey(dir) }()
	}
	wg.Wait()
	for i := range keys {
		if errs[i] != nil {
			t.Fatalf("racer %d: %v", i, errs[i])
		}
		if !bytes.Equal(keys[i], keys[0]) {
			t.Fatal("racing starters ended up with different keys")
		}
	}
	if names := listDir(t, dir); len(names) != 1 {
		t.Errorf("leftover files: %v", names)
	}
}

// A restart (a new handler set from the same config) keeps the signing key,
// so a CSRF token minted before it still validates after it (SEC-027).
func TestNewHandlersKeySurvivesRestart(t *testing.T) {
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
	if _, _, err := newHandlers(svc, cfg, discardLogger()); err != nil {
		t.Fatal(err)
	}
	k1, _ := loadOrCreateWebKey(cfg.StateDir)
	if _, _, err := newHandlers(svc, cfg, discardLogger()); err != nil {
		t.Fatal(err)
	}
	k2, _ := loadOrCreateWebKey(cfg.StateDir)
	if !bytes.Equal(k1, k2) {
		t.Error("key changed across restart")
	}
	// A corrupt key stops the service from starting.
	if err := os.WriteFile(filepath.Join(cfg.StateDir, webKeyFile), []byte("junk\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newHandlers(svc, cfg, discardLogger()); err == nil {
		t.Error("corrupt key accepted by newHandlers")
	}
}

func TestSetupResetCommand(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	ctx := context.Background()
	var out, errOut bytes.Buffer

	// Unused: nothing to do.
	if code := run([]string{"setup-reset", "-database", db}, &out, &errOut); code != exitOK || !strings.Contains(out.String(), "nothing to do") {
		t.Fatalf("unused reset: %d %s %s", code, out.String(), errOut.String())
	}

	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := newService(st, testConfigFor(db))
	if err != nil {
		t.Fatal(err)
	}
	_, p, err := svc.Login(ctx, "founder01", "correct horse battery")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.MarkSetupSecretUsed(ctx, domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeWrite}}); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	out.Reset()
	if code := run([]string{"setup-reset", "-database", db}, &out, &errOut); code != exitOK || !strings.Contains(out.String(), "valid again") {
		t.Fatalf("reset: %d %s %s", code, out.String(), errOut.String())
	}
	st, err = store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, _ = newService(st, testConfigFor(db))
	if used, err := svc.SetupSecretUsed(ctx); err != nil || used {
		t.Fatalf("after reset used=%v err=%v", used, err)
	}
	// The reset is in the audit log and the chain still verifies.
	checks, err := verifyAllNetworks(ctx, svc, st.DB(), "t", "")
	if err != nil || !printChecks(&out, checks) {
		t.Fatalf("verify after reset: %v", err)
	}
}
