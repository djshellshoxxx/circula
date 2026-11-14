package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// corruptLive edits the live database file in place, as a faulty or hostile
// writer would: the append-only guard is dropped and an audit row rewritten.
func corruptLive(t *testing.T, db string) {
	t.Helper()
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	for _, s := range []string{`DROP TRIGGER audit_no_update`, `UPDATE audit_events SET reason = 'forged' WHERE seq = 1`} {
		if _, err := raw.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

// Backup command: usage, destination handling, refusal to overwrite, and the
// manifest it leaves behind.
func TestBackupCommandPaths(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	var out, errOut bytes.Buffer

	if code := run([]string{"backup", "-database", db}, &out, &errOut); code != exitUsage || !strings.Contains(errOut.String(), "usage: circula backup") {
		t.Fatalf("no destination: code %d, %s", code, errOut.String())
	}
	if code := run([]string{"backup", "-database", filepath.Join(dir, "missing.db"), dir}, &out, &errOut); code != exitFailure ||
		!strings.Contains(errOut.String(), "does not exist") {
		t.Fatalf("missing database: code %d, %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.db")); err == nil {
		t.Error("backup created a database file where none existed")
	}

	// Into a directory: the file name is generated from the time.
	into := filepath.Join(dir, "bk-dir")
	if err := os.Mkdir(into, 0o700); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"backup", "-database", db, into}, &out, &errOut); code != exitOK {
		t.Fatalf("backup into dir: code %d, %s", code, errOut.String())
	}
	for _, want := range []string{"backup: ", "manifest: ", "sha256: ", "schema: v", "ledger: ok"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stdout lacks %q: %s", want, out.String())
		}
	}
	files, _ := filepath.Glob(filepath.Join(into, "circula-*.db"))
	if len(files) != 1 {
		t.Fatalf("backup files in dir: %v", files)
	}
	m := mustManifest(t, files[0])
	if err := verifyBackupFile(files[0], m); err != nil {
		t.Errorf("manifest does not match backup: %v", err)
	}
	if m.Reason != "manual" || m.Ledger.Status != "ok" || m.Integrity != "ok" {
		t.Errorf("manifest = %+v", m)
	}

	// Explicit file name, then the same name again is refused.
	explicit := filepath.Join(dir, "explicit.db")
	out.Reset()
	errOut.Reset()
	if code := run([]string{"backup", "-database", db, explicit}, &out, &errOut); code != exitOK {
		t.Fatalf("backup to file: code %d, %s", code, errOut.String())
	}
	if _, err := os.Stat(explicit + ".manifest.json"); err != nil {
		t.Errorf("manifest missing: %v", err)
	}
	errOut.Reset()
	if code := run([]string{"backup", "-database", db, explicit}, &out, &errOut); code != exitFailure || !strings.Contains(errOut.String(), "already exists") {
		t.Errorf("overwrite: code %d, %s", code, errOut.String())
	}

	// A trailing separator names a directory that must already exist.
	errOut.Reset()
	if code := run([]string{"backup", "-database", db, filepath.Join(dir, "nope") + string(filepath.Separator)}, &out, &errOut); code != exitFailure ||
		!strings.Contains(errOut.String(), "destination directory") {
		t.Errorf("missing destination directory: code %d, %s", code, errOut.String())
	}
}

// BACKUP-KEY: with backup-key-file set the manifest is signed; the signature
// verifies under the key and fails under any other.
func TestBackupCommandSignsManifestWithKey(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	const key = "0123456789abcdef-backup-key"
	keyFile := writeSecret(t, key, 0o600)
	dest := filepath.Join(dir, "signed.db")
	var out, errOut bytes.Buffer
	if code := run([]string{"backup", "-database", db, "-backup-key-file", keyFile, dest}, &out, &errOut); code != exitOK {
		t.Fatalf("code %d, %s", code, errOut.String())
	}
	m := mustManifest(t, dest)
	if m.HMAC == "" {
		t.Fatal("manifest not signed with backup-key-file set")
	}
	if ok, err := checkManifestAuth(m, []byte(key)); !ok || err != nil {
		t.Errorf("signature under the configured key: %v %v", ok, err)
	}
	if _, err := checkManifestAuth(m, []byte("a-different-backup-key")); err == nil {
		t.Error("signature accepted under a different key")
	}
	// A manifest whose signed fields were edited no longer verifies.
	edited := *m
	edited.SchemaVersion++
	if ok, _ := checkManifestAuth(&edited, []byte(key)); ok {
		t.Error("edited manifest still authenticates")
	}
	// Without the key the manifest is unsigned.
	unkeyed := filepath.Join(dir, "unkeyed.db")
	if code := run([]string{"backup", "-database", db, unkeyed}, &out, &errOut); code != exitOK {
		t.Fatalf("unkeyed backup: %d %s", code, errOut.String())
	}
	if mustManifest(t, unkeyed).HMAC != "" {
		t.Error("manifest signed without a key")
	}
}

// Exit code 3: the operation completes, but the ledger is faulty. Backup still
// writes its copy and says so; verify-ledger names the failing network.
func TestLedgerFailureExitCodeThree(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	corruptLive(t, db)

	var out, errOut bytes.Buffer
	dest := filepath.Join(dir, "faulty.db")
	if code := run([]string{"backup", "-database", db, dest}, &out, &errOut); code != exitLedgerFailed {
		t.Fatalf("backup of faulty ledger: code %d, want %d (%s)", code, exitLedgerFailed, errOut.String())
	}
	if !strings.Contains(errOut.String(), "WARNING") || !strings.Contains(out.String(), "ledger: failed") {
		t.Errorf("backup did not report the fault: out %q err %q", out.String(), errOut.String())
	}
	if m := mustManifest(t, dest); m.Ledger.Status != "failed" || m.Ledger.Failing != 1 {
		t.Errorf("manifest ledger = %+v", m.Ledger)
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"verify-ledger", "-database", db}, &out, &errOut); code != exitLedgerFailed {
		t.Fatalf("verify-ledger: code %d, want %d (%s)", code, exitLedgerFailed, errOut.String())
	}
	if !strings.Contains(out.String(), "test-net: FAILED") {
		t.Errorf("verify-ledger output = %q", out.String())
	}
	// Filtering to a network that does not exist is a usage-style failure, not a ledger fault.
	out.Reset()
	errOut.Reset()
	if code := run([]string{"verify-ledger", "-database", db, "-network", "no-such-net"}, &out, &errOut); code != exitFailure ||
		!strings.Contains(errOut.String(), "no network matches") {
		t.Errorf("unknown network: code %d, %s", code, errOut.String())
	}
}

// CMD-01 / OPS-016: verification leaves the database file byte-identical, both
// for a clean ledger and for one that fails.
func TestVerifyLedgerLeavesDatabaseChecksumUnchanged(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	var out, errOut bytes.Buffer
	before := sha(t, db)
	if code := run([]string{"verify-ledger", "-database", db}, &out, &errOut); code != exitOK {
		t.Fatalf("clean verify: %d %s", code, errOut.String())
	}
	if after := sha(t, db); after != before {
		t.Error("verify-ledger changed the database file (clean ledger)")
	}

	corruptLive(t, db)
	before = sha(t, db)
	out.Reset()
	if code := run([]string{"verify-ledger", "-database", db}, &out, &errOut); code != exitLedgerFailed {
		t.Fatalf("faulty verify: %d", code)
	}
	if after := sha(t, db); after != before {
		t.Error("verify-ledger changed the database file (faulty ledger)")
	}
	// A backup taken from the same database also leaves the source alone.
	if _, _, err := backupTo(context.Background(), testConfigFor(db), filepath.Join(dir, "bk"), "manual", nil); err != nil {
		t.Fatal(err)
	}
	if after := sha(t, db); after != before {
		t.Error("backup changed the source database")
	}
}

// Admin create-operator: argument checks, password sources, and refusal of
// duplicates. The password is never taken from the command line.
func TestAdminCreateOperatorPaths(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	var out, errOut bytes.Buffer

	if code := run([]string{"admin"}, &out, &errOut); code != exitUsage || !strings.Contains(errOut.String(), "usage: circula admin") {
		t.Fatalf("admin without subcommand: %d %s", code, errOut.String())
	}
	if code := run([]string{"admin", "delete-everything"}, &out, &errOut); code != exitUsage {
		t.Errorf("unknown admin subcommand: %d", code)
	}
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db}, &out, &errOut); code != exitUsage ||
		!strings.Contains(errOut.String(), "-login is required") {
		t.Errorf("missing -login: %d %s", code, errOut.String())
	}
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops01", "-password", "hunter2hunter2"}, &out, &errOut); code != exitUsage {
		t.Errorf("password on the command line accepted: %d %s", code, errOut.String())
	}

	// Password from the environment.
	t.Setenv(operatorPasswordEnv, "correct horse battery")
	out.Reset()
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops01"}, &out, &errOut); code != exitOK ||
		!strings.Contains(out.String(), "created person") {
		t.Fatalf("create from env: %d %s", code, errOut.String())
	}
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops01"}, &out, &errOut); code != exitFailure {
		t.Errorf("duplicate login: %d %s", code, errOut.String())
	}
	t.Setenv(operatorPasswordEnv, "short")
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops02"}, &out, &errOut); code != exitFailure {
		t.Errorf("password below the minimum: %d %s", code, errOut.String())
	}

	// Password from stdin when the environment is empty; nothing in the store is plaintext.
	t.Setenv(operatorPasswordEnv, "")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("stdin operator pass\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	realStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = realStdin; _ = r.Close() })
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops03"}, &out, &errOut); code != exitOK {
		t.Fatalf("create from stdin: %d %s", code, errOut.String())
	}
	os.Stdin = realStdin
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	var hash string
	if err := raw.QueryRowContext(context.Background(), `SELECT password_hash FROM persons WHERE login = 'ops03'`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	if !strings.HasPrefix(hash, "$argon2id$") || strings.Contains(hash, "stdin operator pass") {
		t.Errorf("stored password form unexpected: %.20s", hash)
	}
	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Login(context.Background(), "ops03", "stdin operator pass"); err != nil {
		t.Errorf("operator cannot sign in with the stdin password: %v", err)
	}

	// Nothing supplied at all is a usage error, and nothing is created.
	empty, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = empty
	errOut.Reset()
	if code := run([]string{"admin", "create-operator", "-database", db, "-login", "ops04"}, &out, &errOut); code != exitUsage ||
		!strings.Contains(errOut.String(), "no password") {
		t.Errorf("no password: %d %s", code, errOut.String())
	}
	os.Stdin = realStdin
	_ = empty.Close()
}
