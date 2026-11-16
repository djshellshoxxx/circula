package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// QA-026 (OPS-022 to OPS-024): an instance is upgraded through a migration
// that fails, and is rolled back. The failure is not simulated: the seeded
// data holds a completed trade with no network policy in force at its
// completion, for which migration 0016 cannot derive a dispute deadline and
// stops for an owner mapping. Expected: the pre-upgrade backup is taken and
// verified; the failed migration leaves the schema and data exactly as the
// last good migration left them; rolling back with the pre-upgrade backup
// restores the pre-upgrade state.

const qa26Base = int64(1_700_000_000_000)

// qa26Seed builds a database at schema v with one network, two members, a
// policy and two completed trades, one completed before the policy was in
// force (the 0016 blocker), with balances that match the entries.
func qa26Seed(t *testing.T, path string, v int) {
	t.Helper()
	st, err := store.OpenAtVersion(context.Background(), path, v)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	exec := func(q string, a ...any) {
		t.Helper()
		if _, err := st.DB().Exec(q, a...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	day := int64(86_400_000)
	exec(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('n','Up','up','active',2,?)`, qa26Base)
	exec(`INSERT INTO accounts(id,network_id,is_system,balance) VALUES ('sys','n',1,0)`)
	for _, m := range []string{"a", "b"} {
		exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES (?,?,'x',?)`, "p"+m, m+"@up.test", qa26Base)
		exec(`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES (?,'n',?,?,'active','invitation',?)`,
			"m"+m, "p"+m, "Member "+m, qa26Base)
		exec(`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES (?,'n',?,0,0)`, "acc"+m, "m"+m)
	}
	exec(`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES ('n',1,'{"dispute_window_days":30}',?)`, qa26Base+10*day)
	for i, tr := range []struct {
		id        string
		completed int64
	}{{"t-early", qa26Base + day}, {"t-late", qa26Base + 20*day}} { // t-early predates every policy
		seq := i + 1
		exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,created_by,proposed_at)
			VALUES (?,'n','trade','proposed','acca','accb',100,'ma',?)`, tr.id, tr.completed-1000)
		exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?,'n',?,'acca',-100,?,?),(?,'n',?,'accb',100,?,?)`,
			tr.id+"-d", tr.id, seq, tr.completed, tr.id+"-c", tr.id, seq, tr.completed)
		exec(`UPDATE transactions SET status='completed', completed_at=?, ledger_seq=? WHERE id=?`, tr.completed, seq, tr.id)
		exec(`UPDATE accounts SET balance = balance - 100 WHERE id = 'acca'`)
		exec(`UPDATE accounts SET balance = balance + 100 WHERE id = 'accb'`)
	}
}

// qa26Fingerprint hashes the schema (every object's SQL) and the content of
// every table in rowid order. schema_migrations contributes version and name
// only, since applied_at is the wall-clock time of the run.
func qa26Fingerprint(t *testing.T, path string) string {
	t.Helper()
	db, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	h := sha256.New()
	rows, err := db.Query(`SELECT type, name, COALESCE(sql,'') FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var typ, name, ddl string
		if err := rows.Scan(&typ, &name, &ddl); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%s|%s|%s\n", typ, name, ddl)
		if typ == "table" {
			tables = append(tables, name)
		}
	}
	_ = rows.Close()
	for _, tb := range tables {
		q := `SELECT * FROM "` + tb + `" ORDER BY rowid`
		if tb == "schema_migrations" {
			q = `SELECT version, name FROM schema_migrations ORDER BY version`
		}
		fmt.Fprintf(h, "[%s]", tb)
		qa26HashRows(t, db, q, h)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func qa26HashRows(t *testing.T, db *sql.DB, q string, h interface{ Write([]byte) (int, error) }) {
	t.Helper()
	r, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
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
}

// qa26Migrate runs `circula migrate`; backups go to an isolated backupDir.
func qa26Migrate(t *testing.T, path, backupDir string) (code int, out string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run([]string{"migrate", "-database", path, "-backup-dir", backupDir}, &o, &e)
	return code, o.String() + e.String()
}

func TestQA026UpgradeWithFailedMigrationAndRollback(t *testing.T) {
	ctx := context.Background()
	// The intentionally unmappable trade is a blocker for migration 0016,
	// regardless of how many later migrations this binary contains.
	const head = 16
	for _, from := range []int{head - 1, head - 2} {
		t.Run(fmt.Sprintf("from-v%d", from), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "circula.db")
			qa26Seed(t, path, from)
			pre := qa26Fingerprint(t, path)

			// The state the last good migration leaves: the same seed migrated
			// only up to the version before the failing one.
			ref := filepath.Join(t.TempDir(), "ref.db")
			qa26Seed(t, ref, from)
			refSt, err := store.OpenAtVersion(ctx, ref, head-1)
			if err != nil {
				t.Fatal(err)
			}
			_ = refSt.Close()
			lastGood := qa26Fingerprint(t, ref)

			code, out := qa26Migrate(t, path, filepath.Join(dir, "backups"))
			if code == exitOK {
				t.Fatalf("migration through the blocker succeeded:\n%s", out)
			}
			if !strings.Contains(out, "pre-migration backup:") || !strings.Contains(out, "database left at its previous version") ||
				!strings.Contains(out, "dispute_deadline_backfill_needs_owner_mapping") {
				t.Fatalf("migrate output lacks the backup, the failure or its cause:\n%s", out)
			}

			// The failed migration left the prior schema, and nothing of 0016.
			state, err := inspectSchema(ctx, path)
			if err != nil || state.Applied != head-1 {
				t.Fatalf("schema after failed migration = %+v (%v), want v%d", state, err, head-1)
			}
			if got := qa26Fingerprint(t, path); got != lastGood {
				t.Fatal("failed migration changed the database beyond the last good migration")
			}
			if from == head-1 && lastGood != pre {
				t.Fatal("a failure at the first pending migration must leave the database unchanged")
			}
			if from < head-1 && lastGood == pre {
				t.Fatal("fingerprint does not see the earlier migration that committed; the comparison proves nothing")
			}

			// The pre-upgrade backup was taken, verifies and holds the pre-upgrade state.
			backups, err := filepath.Glob(filepath.Join(dir, "backups", "*.db"))
			if err != nil || len(backups) != 1 {
				t.Fatalf("backups = %v (%v)", backups, err)
			}
			backup := backups[0]
			m, err := readManifest(backup + ".manifest.json")
			if err != nil {
				t.Fatal(err)
			}
			if m.Reason != "pre-migration" || m.SchemaVersion != from || m.Integrity != "ok" {
				t.Fatalf("pre-upgrade manifest = %+v", m)
			}
			if err := verifyBackupFile(backup, m); err != nil {
				t.Fatal(err)
			}
			if got := qa26Fingerprint(t, backup); got != pre {
				t.Fatal("pre-upgrade backup does not hold the pre-upgrade state")
			}

			// A second attempt fails the same way and changes nothing further.
			if code, out := qa26Migrate(t, path, t.TempDir()); code == exitOK || !strings.Contains(out, "needs_owner_mapping") {
				t.Fatalf("second migration attempt: code %d\n%s", code, out)
			}
			if got := qa26Fingerprint(t, path); got != lastGood {
				t.Fatal("second failed migration changed the database")
			}

			// Rollback (OPS-024) restores the pre-upgrade backup with the
			// previous binary. This binary refuses a backup older than its own
			// schema, and says so without touching the database.
			var o, e bytes.Buffer
			if code := run([]string{"restore", "-allow-unsigned", "-overwrite", "-database", path, "-backup-dir", t.TempDir(), backup}, &o, &e); code == exitOK ||
				!strings.Contains(e.String(), "matching binary") {
				t.Fatalf("restore of an older backup by this binary: code %d\n%s%s", code, o.String(), e.String())
			}
			if got := qa26Fingerprint(t, path); got != lastGood {
				t.Fatal("refused restore changed the database")
			}
			// The previous binary's restore is the same version-independent
			// sequence: verify the manifest checksum, copy, integrity-check,
			// swap with a way back, and check the result opens at the backup's
			// schema.
			if err := verifyBackupFile(backup, m); err != nil {
				t.Fatal(err)
			}
			partial := path + ".restore-partial"
			if err := copyFileSync(backup, partial); err != nil {
				t.Fatal(err)
			}
			if res, err := integrityCheck(ctx, partial); err != nil || res != "ok" {
				t.Fatalf("integrity of the rollback copy: %q %v", res, err)
			}
			if err := swapDatabase(ctx, path, partial, true, func(ctx context.Context) error {
				s, err := inspectSchema(ctx, path)
				if err == nil && s.Applied != m.SchemaVersion {
					err = fmt.Errorf("restored schema v%d, backup holds v%d", s.Applied, m.SchemaVersion)
				}
				return err
			}); err != nil {
				t.Fatalf("rollback swap: %v", err)
			}
			if got := qa26Fingerprint(t, path); got != pre {
				t.Fatal("rollback did not restore the pre-upgrade state")
			}
		})
	}
}
