package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// cliLedgerSum hashes the ledger tables of the database at db.
func cliLedgerSum(t *testing.T, st *store.SQLite) string {
	t.Helper()
	h := sha256.New()
	for _, q := range []string{`SELECT id, network_id, member_id, is_system, balance FROM accounts ORDER BY id`,
		`SELECT * FROM entries ORDER BY rowid`, `SELECT * FROM transactions ORDER BY rowid`} {
		r, err := st.DB().Query(q)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := r.Columns()
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(h, "%v\n", vals)
		}
		_ = r.Close()
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// SEC-027 / OPS-010 session key rotation: `revoke-sessions -rotate-web-key`
// ends every session (old cookies are refused afterwards), replaces the web
// signing key, audits the run and leaves the ledger unchanged.
func TestRevokeSessionsCommand(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	ctx := context.Background()
	oldKey, err := loadOrCreateWebKey(dir)
	if err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var tokens []string
	for i := 0; i < 2; i++ {
		tok, _, err := svc.Login(ctx, "founder01", "correct horse battery")
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, tok)
	}
	ledger := cliLedgerSum(t, st)
	_ = st.Close()

	var out, errOut bytes.Buffer
	for _, args := range [][]string{
		{"revoke-sessions", "-database", db}, // no reason
		{"revoke-sessions", "-database", db, "-reason", "x y z", "-login", "a", "-rotate-web-key"}, // global rotation for one person
	} {
		if code := run(args, &out, &errOut); code != exitUsage {
			t.Errorf("%v: code %d, want usage error", args, code)
		}
	}
	// An unquoted multi-word reason must not silently drop the flags after it
	// (the scope would widen to every person, or the rotation be skipped).
	for _, args := range [][]string{
		{"revoke-sessions", "-database", db, "-reason", "lost", "laptop", "-login", "founder01"},
		{"revoke-sessions", "-database", db, "-state-dir", dir, "-reason", "incident", "7", "-rotate-web-key"},
	} {
		errOut.Reset()
		if code := run(args, &out, &errOut); code != exitUsage || !strings.Contains(errOut.String(), "unexpected arguments") {
			t.Errorf("%v: code %d, stderr %q; want usage error naming the extra arguments", args, code, errOut.String())
		}
	}
	if k, err := readWebKey(filepath.Join(dir, webKeyFile)); err != nil || !bytes.Equal(k, oldKey) {
		t.Errorf("a refused run replaced the web key (err %v)", err)
	}
	if code := run([]string{"revoke-sessions", "-database", db, "-reason", "lost laptop", "-login", "ghost"}, &out, &errOut); code != exitFailure {
		t.Errorf("unknown login: code %d", code)
	}

	out.Reset()
	errOut.Reset()
	code := run([]string{"revoke-sessions", "-database", db, "-state-dir", dir, "-reason", "suspected key exposure", "-rotate-web-key"}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("revoke-sessions: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "revoked 2 session(s) of every person") || !strings.Contains(out.String(), "web signing key") {
		t.Errorf("output: %q", out.String())
	}
	newKey, err := readWebKey(filepath.Join(dir, webKeyFile))
	if err != nil {
		t.Fatalf("rotated key unreadable: %v", err)
	}
	if bytes.Equal(newKey, oldKey) {
		t.Error("web key not replaced")
	}
	if strings.Contains(out.String()+errOut.String(), base64.StdEncoding.EncodeToString(newKey)) {
		t.Error("key printed")
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(filepath.Join(dir, webKeyFile)); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("key file mode: %v %v", fi.Mode(), err)
		}
	}

	st, err = store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err = service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if _, err := svc.Authenticate(ctx, tok); err == nil {
			t.Error("old session accepted after revoke-sessions")
		}
	}
	if got := cliLedgerSum(t, st); got != ledger {
		t.Error("ledger checksum changed")
	}
	var typ, reason string
	if err := st.DB().QueryRow(`SELECT actor_type, reason FROM audit_events WHERE action = ?`, domain.ActSessionsRevoked).Scan(&typ, &reason); err != nil {
		t.Fatalf("audit event missing: %v", err)
	}
	if typ != string(domain.ActorOperator) || reason != "suspected key exposure" {
		t.Errorf("audit event %s %q", typ, reason)
	}
	// serve accepts the rotated key
	if k, err := loadOrCreateWebKey(dir); err != nil || !bytes.Equal(k, newKey) {
		t.Errorf("reload rotated key: %v", err)
	}
}
