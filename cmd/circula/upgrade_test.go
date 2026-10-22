package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// seedV1Database builds a database exactly as the first release wrote it:
// only migration 0001 applied, with a network, members, a completed trade and
// a hash-chained audit trail.
func seedV1Database(t *testing.T, dir string) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(dir, "circula.db")
	ddl, err := os.ReadFile(filepath.Join("..", "..", "internal", "store", "migrations", "0001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	exec := func(q string, a ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, a...); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`)
	exec(string(ddl))
	exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES (1,'init',1)`)

	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	ms := at.UnixMilli()
	pol, _ := json.Marshal(domain.DefaultPolicy())
	exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('p1','alice01','hash-a',?),('p2','bob00002','hash-b',?)`, ms, ms)
	exec(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('n1','Old Net','old-net','active',1,?)`, ms)
	exec(`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES ('n1',1,?,?)`, string(pol), ms)
	exec(`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES
		('m1','n1','p1','Alice','active','founder',?),('m2','n1','p2','Bob','active','invitation',?)`, ms, ms)
	exec(`INSERT INTO role_assignments(id,network_id,member_id,role,granted_at) VALUES ('r1','n1','m1','admin',?)`, ms)
	exec(`INSERT INTO accounts(id,network_id,member_id,balance) VALUES ('a1','n1','m1',-100),('a2','n1','m2',100)`)
	exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,created_by,proposed_at)
		VALUES ('t1','n1','trade','proposed','a1','a2',100,'m1',?)`, ms)
	exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES
		('e1','n1','t1','a1',-100,1,?),('e2','n1','t1','a2',100,1,?)`, ms, ms)
	exec(`UPDATE transactions SET status='completed', accepted_by='m2', completed_at=?, ledger_seq=1 WHERE id='t1'`, ms)
	exec(`INSERT INTO transaction_status_events(id,network_id,tx_id,from_status,to_status,actor_id,occurred_at) VALUES ('s1','n1','t1','proposed','completed','m2',?)`, ms)

	prev := ""
	for i, ev := range []domain.AuditEvent{
		{ID: "ae1", Action: domain.ActNetworkCreated, TargetType: "network", TargetID: "n1"},
		{ID: "ae2", Action: domain.ActTxCompleted, TargetType: "transaction", TargetID: "t1", SummaryAfter: `{"amount":100}`},
	} {
		ev.NetworkID, ev.Seq, ev.OccurredAt = "n1", int64(i+1), at.Add(time.Duration(i)*time.Second)
		ev.ActorType, ev.ActorID, ev.PrevHash = domain.ActorMember, "m1", prev
		ev.Hash = service.AuditHash(&ev)
		prev = ev.Hash
		exec(`INSERT INTO audit_events(id,network_id,seq,occurred_at,actor_type,actor_id,action,target_type,target_id,summary_after,prev_hash,hash)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, ev.ID, ev.NetworkID, ev.Seq, ev.OccurredAt.UnixMilli(), string(ev.ActorType), ev.ActorID,
			ev.Action, ev.TargetType, ev.TargetID, ev.SummaryAfter, ev.PrevHash, ev.Hash)
	}
	return path
}

// Upgrade path: a database created at migration 0001 is migrated to the latest
// schema by `circula migrate`, keeps every row, passes SQLite's integrity and
// foreign-key checks and passes ledger and audit-chain verification.
func TestUpgradeFromMigration0001(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := seedV1Database(t, dir)

	state, err := inspectSchema(ctx, path)
	if err != nil || state.Applied != 1 || state.Pending() != store.SchemaVersion()-1 {
		t.Fatalf("seed schema state = %+v, %v", state, err)
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"migrate", "-database", path}, &out, &errOut); code != exitOK {
		t.Fatalf("migrate: code %d\n%s\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "pre-migration backup") {
		t.Errorf("no pre-migration backup reported: %s", out.String())
	}

	raw, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	count := func(q string) int {
		t.Helper()
		var n int
		if err := raw.QueryRowContext(ctx, q).Scan(&n); err != nil {
			t.Fatalf("%v: %s", err, q)
		}
		return n
	}
	for q, want := range map[string]int{
		`SELECT MAX(version) FROM schema_migrations`:                                    store.SchemaVersion(),
		`SELECT COUNT(*) FROM schema_migrations`:                                        store.SchemaVersion(),
		`SELECT COUNT(*) FROM persons`:                                                  2,
		`SELECT COUNT(*) FROM members`:                                                  2,
		`SELECT COUNT(*) FROM entries`:                                                  2,
		`SELECT COUNT(*) FROM audit_events`:                                             2,
		`SELECT COUNT(*) FROM transactions WHERE status='completed' AND amount=100`:     1,
		`SELECT balance FROM accounts WHERE id='a2'`:                                    100,
		`SELECT COUNT(*) FROM persons WHERE login='alice01' AND password_hash='hash-a'`: 1,
		`SELECT must_change_password FROM persons WHERE id='p1'`:                        0,
		`SELECT COUNT(*) FROM audit_heads WHERE network_id='n1' AND seq=2`:              1, // backfilled by 0002
	} {
		if got := count(q); got != want {
			t.Errorf("%s = %d, want %d", q, got, want)
		}
	}
	if res, err := integrityCheck(ctx, path); err != nil || res != "ok" {
		t.Errorf("integrity_check = %q, %v", res, err)
	}
	rows, err := raw.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Error("foreign_key_check reports violations after upgrade")
	}
	_ = rows.Close()

	// Ledger and audit chains verify on the upgraded database.
	cfg := testConfigFor(path)
	checks, err := verifyFileChecks(ctx, path, cfg)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(checks) != 1 || !checks[0].OK {
		t.Fatalf("ledger verification after upgrade: %+v", checks)
	}

	// The append-only guarantees introduced at 0001 still hold.
	if _, err := raw.ExecContext(ctx, `UPDATE audit_events SET reason='x'`); err == nil {
		t.Error("audit_events became mutable")
	}
	// A second migrate is a no-op.
	out.Reset()
	if code := run([]string{"migrate", "-database", path}, &out, &errOut); code != exitOK || !strings.Contains(out.String(), "up to date") {
		t.Errorf("second migrate: %d %s", code, out.String())
	}
}
