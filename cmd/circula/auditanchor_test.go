package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// dropAuditTriggers plays an attacker with write access to the database file.
func dropAuditTriggers(t *testing.T, db string) {
	t.Helper()
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	rows, err := raw.Query(`SELECT name FROM sqlite_master WHERE type='trigger' AND tbl_name IN ('audit_events','audit_heads')`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		names = append(names, n)
	}
	_ = rows.Close()
	for _, n := range names {
		if _, err := raw.Exec(`DROP TRIGGER "` + n + `"`); err != nil {
			t.Fatal(err)
		}
	}
}

func runAnchor(t *testing.T, db string, args ...string) (int, string) {
	t.Helper()
	t.Setenv(envName("database"), db)
	var out, errb bytes.Buffer
	code := cmdAuditAnchor(context.Background(), args, &out, &errb)
	return code, out.String() + errb.String()
}

func TestAuditAnchorWriteVerifyAndTamper(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	anchor := filepath.Join(t.TempDir(), "anchor.json")

	if code, out := runAnchor(t, db, "write", anchor); code != exitOK {
		t.Fatalf("write = %d %s", code, out)
	}
	if code, out := runAnchor(t, db, "verify", anchor); code != exitOK || !strings.Contains(out, "ok") {
		t.Fatalf("verify clean = %d %s", code, out)
	}

	// The chain legitimately grows: an older anchor still verifies.
	ctx := context.Background()
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: "second01", Password: "correct horse battery"})
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	if _, _, err := svc.CreateNetwork(ctx, actor, service.CreateNetworkInput{Name: "Two", Slug: "two-net", CreatorDisplayName: "Second"}); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	if code, out := runAnchor(t, db, "verify", anchor); code != exitOK {
		t.Fatalf("verify after growth = %d %s", code, out)
	}

	// Rewrite of the anchored head (chain recomputed by an attacker).
	dropAuditTriggers(t, db)
	raw, _ := openRaw(db)
	if _, err := raw.Exec(`UPDATE audit_events SET hash = 'ff' || substr(hash, 3) WHERE network_id = (SELECT id FROM networks WHERE slug='test-net') AND seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = (SELECT id FROM networks WHERE slug='test-net'))`); err != nil {
		t.Fatal(err)
	}
	if code, out := runAnchor(t, db, "verify", anchor); code != exitLedgerFailed || !strings.Contains(out, "rewritten") {
		t.Fatalf("rewrite not detected = %d %s", code, out)
	}

	// Tail truncation.
	if _, err := raw.Exec(`DELETE FROM audit_events WHERE network_id = (SELECT id FROM networks WHERE slug='test-net') AND seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = (SELECT id FROM networks WHERE slug='test-net'))`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	if code, out := runAnchor(t, db, "verify", anchor); code != exitLedgerFailed || !strings.Contains(out, "missing") {
		t.Fatalf("truncation not detected = %d %s", code, out)
	}
}

func TestAuditAnchorMAC(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	keyFile := writeSecret(t, "0123456789abcdef-anchor-key\n", 0o600)
	t.Setenv(envName("audit-anchor-key-file"), keyFile)
	anchor := filepath.Join(t.TempDir(), "a.json")
	if code, out := runAnchor(t, db, "write", anchor); code != exitOK {
		t.Fatalf("write = %d %s", code, out)
	}
	if b, _ := os.ReadFile(anchor); !strings.Contains(string(b), `"mac"`) {
		t.Fatal("anchor has no MAC")
	}
	if code, out := runAnchor(t, db, "verify", anchor); code != exitOK {
		t.Fatalf("verify = %d %s", code, out)
	}
	// Edit the file: MAC no longer verifies.
	b, _ := os.ReadFile(anchor)
	if err := os.WriteFile(anchor, bytes.Replace(b, []byte(`"seq": `), []byte(`"seq": 9`), 1), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runAnchor(t, db, "verify", anchor); code != exitLedgerFailed || !strings.Contains(out, "MAC") {
		t.Fatalf("tampered anchor accepted = %d %s", code, out)
	}
	// Anchor without a MAC while a key is configured is refused.
	t.Setenv(envName("audit-anchor-key-file"), "")
	plain := filepath.Join(t.TempDir(), "p.json")
	if code, _ := runAnchor(t, db, "write", plain); code != exitOK {
		t.Fatal("plain write")
	}
	t.Setenv(envName("audit-anchor-key-file"), keyFile)
	if code, out := runAnchor(t, db, "verify", plain); code != exitLedgerFailed {
		t.Fatalf("unsigned anchor accepted under key = %d %s", code, out)
	}
}

func TestBackupWritesAnchorSidecar(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	cfg := testConfigFor(db)
	cfg.AuditAnchorWithBackups = true
	st, err := store.Open(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	j := &jobRunner{cfg: cfg, st: st, logger: discardLogger(), metrics: newMetrics(time.Now())}
	if err := j.backupNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	m, _ := filepath.Glob(filepath.Join(cfg.BackupDir, "*.anchor.json"))
	if len(m) != 1 {
		t.Fatalf("anchor sidecars = %v", m)
	}
}
