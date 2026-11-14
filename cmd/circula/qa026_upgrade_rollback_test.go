package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// QA-026 (OPS-022 to OPS-024): `circula migrate` on a database one version
// behind takes a verified pre-upgrade backup, the head migration is then
// forced to fail, the database keeps its prior schema and data, and
// restoring the pre-upgrade backup gives back exactly the pre-upgrade state.
func TestQA026UpgradeWithForcedFailureRollsBack(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "circula.db")
	head := store.SchemaVersion()
	prev, err := store.OpenAtVersion(ctx, db, head-1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prev.DB().Exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('p1','before-upgrade','h',1)`); err != nil {
		t.Fatal(err)
	}
	if err := prev.Close(); err != nil {
		t.Fatal(err)
	}
	// Force the head migration to fail midway: an object it creates already exists.
	files, err := filepath.Glob("../../internal/store/migrations/*.sql")
	if err != nil || len(files) != head {
		t.Fatalf("migration files: %v (%d)", err, len(files))
	}
	sort.Strings(files)
	last, err := os.ReadFile(files[len(files)-1])
	if err != nil {
		t.Fatal(err)
	}
	creates := regexp.MustCompile(`(?i)CREATE\s+TRIGGER\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)`).FindAllStringSubmatch(string(last), -1)
	if len(creates) == 0 {
		t.Fatalf("head migration %s creates no trigger to block", filepath.Base(files[len(files)-1]))
	}
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	blocked := creates[len(creates)-1][1]
	if _, err := raw.Exec(fmt.Sprintf(`CREATE TABLE blocker_tbl (x INTEGER); CREATE TRIGGER %s AFTER INSERT ON blocker_tbl BEGIN SELECT 1; END`, blocked)); err != nil {
		t.Fatal(err)
	}
	before := schemaAndRows(t, raw)
	_ = raw.Close()

	var out, errOut bytes.Buffer
	code := run([]string{"migrate", "-database", db}, &out, &errOut)
	if code == 0 {
		t.Fatalf("migrate succeeded although the head migration must fail:\n%s", out.String())
	}
	m := regexp.MustCompile(`pre-migration backup: (\S+)`).FindStringSubmatch(out.String())
	if m == nil {
		t.Fatalf("no pre-upgrade backup reported:\nstdout: %s\nstderr: %s", out.String(), errOut.String())
	}
	backup := m[1]
	raw, err = openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	if got := schemaAndRows(t, raw); got != before {
		t.Fatalf("failed migration changed the database:\nbefore:\n%s\nafter:\n%s", before, got)
	}
	if v, err := store.AppliedVersion(ctx, raw); err != nil || v != head-1 {
		t.Fatalf("applied version %d (%v), want %d", v, err, head-1)
	}
	_ = raw.Close()
	// Rollback (OPS-024) is the previous binary restoring the pre-upgrade
	// backup; this binary refuses an older schema by design. Check what that
	// restore relies on: a verified manifest of the prior schema whose file
	// holds exactly the pre-upgrade state.
	mf, err := readManifest(backup + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if mf.SchemaVersion != head-1 || mf.Integrity != "ok" {
		t.Fatalf("pre-upgrade manifest: schema v%d integrity %q, want v%d ok", mf.SchemaVersion, mf.Integrity, head-1)
	}
	if err := verifyBackupFile(backup, mf); err != nil {
		t.Fatalf("pre-upgrade backup checksum: %v", err)
	}
	if res, err := store.IntegrityCheck(ctx, backup); err != nil || res != "ok" {
		t.Fatalf("pre-upgrade backup integrity: %q %v", res, err)
	}
	bk, err := openRaw(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bk.Close() }()
	if got := schemaAndRows(t, bk); got != before {
		t.Fatalf("pre-upgrade backup differs from the pre-upgrade state:\nbefore:\n%s\nbackup:\n%s", before, got)
	}
}

// schemaAndRows renders the schema objects and the persons rows of db.
func schemaAndRows(t *testing.T, db *sql.DB) string {
	t.Helper()
	var b strings.Builder
	for _, q := range []string{
		`SELECT type || ' ' || name || ' ' || COALESCE(sql, '') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`,
		`SELECT id || ' ' || login FROM persons ORDER BY id`,
	} {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatal(err)
			}
			b.WriteString(line + "\n")
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
	}
	return b.String()
}
