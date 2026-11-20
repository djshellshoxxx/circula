package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// OPS-023: valid migration SQL can still damage a ledger invariant. Inject a
// benign fixture change after the real SQL, while its transaction remains
// open. Refuse before committing the DDL, data or migration version.
func TestMigrationRejectsNewLedgerFindingsBeforeCommit(t *testing.T) {
	for _, tc := range []struct{ name, change, finding string }{
		{"cached_balances", "UPDATE accounts SET balance=balance+1 WHERE id='a1'; UPDATE accounts SET balance=balance-1 WHERE id='a2'", "cached balance"},
		{"reversal_total", "UPDATE transactions SET reversed_total=1 WHERE id='t1'", "reversed_total"},
		{"platform_audit", "INSERT INTO audit_heads(network_id,seq,hash) VALUES('',1,'migration-probe')", "recorded head"},
		{"verification_query", "ALTER TABLE accounts RENAME TO migration_probe_accounts", "verification after migration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := migrationIntegrityFixture(t, 21)
			st, faults := openFaultStore(t, path)
			before := readMigrationIntegrityState(t, st.db)
			migrations, err := loadMigrations()
			if err != nil {
				t.Fatal(err)
			}
			faults.afterStatement, faults.afterStatementSQL = migrations[21].sql, tc.change
			err = st.migrateTo(ctx, 0)
			if err == nil || !strings.Contains(err.Error(), tc.finding) {
				t.Errorf("migration did not refuse the new %s failure: %v", tc.name, err)
			}
			if faults.afterStatementRuns != 1 {
				t.Fatalf("migration fault ran %d times, want 1", faults.afterStatementRuns)
			}
			if after := readMigrationIntegrityState(t, st.db); after != before {
				t.Error("refused migration changed schema, migration metadata or ledger data")
			}
			if err := st.Close(); err != nil {
				t.Fatal(err)
			}
			assertMigrationIntegrityRetry(t, path, 21, before)
		})
	}
}

// OPS-023 compares findings. Passing a migration with an unchanged older
// inconsistency does not authorize writes: ordinary verification still
// reports that finding. A new failure must not hide behind an existing one.
func TestMigrationPreservesExistingFindingsWithoutAllowingNewOnes(t *testing.T) {
	for _, introduce := range []bool{false, true} {
		name := "unchanged"
		if introduce {
			name = "new_failure"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			path := migrationIntegrityFixture(t, 21)
			st, faults := openFaultStore(t, path)
			if _, err := st.db.ExecContext(ctx, "UPDATE accounts SET balance=balance+1 WHERE id='a1'"); err != nil {
				t.Fatal(err)
			}
			before := readMigrationIntegrityState(t, st.db)
			if introduce {
				migrations, err := loadMigrations()
				if err != nil {
					t.Fatal(err)
				}
				faults.afterStatement, faults.afterStatementSQL = migrations[21].sql, "UPDATE accounts SET balance=balance+1 WHERE id='a2'"
			}
			err := st.migrateTo(ctx, 0)
			if introduce {
				if err == nil || !strings.Contains(err.Error(), "introduced ledger finding") {
					t.Errorf("existing finding hid the newly introduced failure: %v", err)
				}
				if after := readMigrationIntegrityState(t, st.db); after != before {
					t.Error("failed comparison did not preserve the original database")
				}
			} else {
				if err != nil {
					t.Fatalf("unchanged legacy finding was treated as new: %v", err)
				}
				if after := readMigrationIntegrityState(t, st.db); after.data != before.data {
					t.Error("ordinary migration modified the pre-existing ledger state")
				}
			}
		})
	}
}

func TestMigrationBaselineReadFailurePreventsDDL(t *testing.T) {
	ctx := context.Background()
	path := migrationIntegrityFixture(t, 21)
	st, faults := openFaultStore(t, path)
	before := readMigrationIntegrityState(t, st.db)
	want := errors.New("injected migration baseline read failure")
	faults.readPrefix, faults.readErr = "SELECT id FROM networks ORDER BY id", want
	if err := st.migrateTo(ctx, 0); !errors.Is(err, want) {
		t.Errorf("incomplete baseline was accepted: %v", err)
	}
	if faults.readErr != nil {
		t.Error("migration did not read the verification baseline")
	}
	if after := readMigrationIntegrityState(t, st.db); after != before {
		t.Error("migration changed state after a baseline read failure")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationIntegrityRetry(t, path, 21, before)
}

// Schema 0001 has a hash chain but no audit_heads table. Only that version
// skips recorded-head comparison; migration 0002 must enable it immediately.
func TestMigrationChecksNewAuditHeadsFromOriginalSchema(t *testing.T) {
	ctx := context.Background()
	path := migrationIntegrityFixture(t, 1)
	st, faults := openFaultStore(t, path)
	before := readMigrationIntegrityState(t, st.db)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	faults.afterStatement, faults.afterStatementSQL = migrations[1].sql,
		"INSERT INTO audit_heads(network_id,seq,hash) VALUES('n1',1,'migration-probe')"
	if err := st.migrateTo(ctx, 0); err == nil || !strings.Contains(err.Error(), "recorded head") {
		t.Errorf("newly supported audit head was not checked: %v", err)
	}
	if faults.afterStatementRuns != 1 {
		t.Fatalf("migration fault ran %d times, want 1", faults.afterStatementRuns)
	}
	if after := readMigrationIntegrityState(t, st.db); after != before {
		t.Error("failed original-schema upgrade changed persisted state")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationIntegrityRetry(t, path, 1, before)
}

type migrationIntegrityState struct{ schema, metadata, data, auditHeads string }

func readMigrationIntegrityState(t *testing.T, db *sql.DB) migrationIntegrityState {
	t.Helper()
	var state migrationIntegrityState
	for _, query := range []struct {
		sql string
		out *string
	}{
		{`SELECT group_concat(record, char(10)) FROM (
 SELECT type || ':' || name || ':' || COALESCE(sql,'') AS record FROM sqlite_schema ORDER BY type,name)`, &state.schema},
		{`SELECT group_concat(record, char(10)) FROM (
 SELECT version || ':' || name || ':' || applied_at AS record FROM schema_migrations ORDER BY version)`, &state.metadata},
		{`SELECT group_concat(record, char(10)) FROM (
 SELECT 'account:' || id || ':' || balance AS record FROM accounts
 UNION ALL SELECT 'entry:' || id || ':' || tx_id || ':' || account_id || ':' || amount || ':' || ledger_seq FROM entries
 UNION ALL SELECT 'transaction:' || id || ':' || status || ':' || amount || ':' || reversed_total FROM transactions
 UNION ALL SELECT 'audit:' || id || ':' || seq || ':' || hash FROM audit_events
 ORDER BY record)`, &state.data},
	} {
		if err := db.QueryRow(query.sql).Scan(query.out); err != nil {
			t.Fatal(err)
		}
	}
	var hasAuditHeads bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='table' AND name='audit_heads')`).Scan(&hasAuditHeads); err != nil {
		t.Fatal(err)
	}
	if hasAuditHeads {
		if err := db.QueryRow(`SELECT COALESCE(group_concat(record, char(10)), '') FROM (
 SELECT network_id || ':' || seq || ':' || hash AS record FROM audit_heads ORDER BY network_id)`).Scan(&state.auditHeads); err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func assertMigrationIntegrityRetry(t *testing.T, path string, previousVersion int, before migrationIntegrityState) {
	t.Helper()
	ctx := context.Background()
	legacy, err := OpenAtVersion(ctx, path, previousVersion)
	if err != nil {
		t.Fatalf("reopen refused migration: %v", err)
	}
	if after := readMigrationIntegrityState(t, legacy.db); after != before {
		t.Error("rollback did not survive reopening the previous schema")
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("retry original migration: %v", err)
	}
	t.Cleanup(func() { _ = upgraded.Close() })
	if version, err := AppliedVersion(ctx, upgraded.db); err != nil || version != SchemaVersion() {
		t.Fatalf("retry schema=%d err=%v, want %d", version, err, SchemaVersion())
	}
	if after := readMigrationIntegrityState(t, upgraded.db); after.data != before.data || after.auditHeads != before.auditHeads {
		t.Error("successful retry changed the original ledger records")
	}
}

// Build a consistent posting with fields present since schema 0001. Later
// fixtures provide their required immutable deadline without changing triggers.
func migrationIntegrityFixture(t *testing.T, version int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "migration-integrity.db")
	st, err := OpenAtVersion(context.Background(), path, version)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('p1','p1','unused',1),('p2','p2','unused',1)`,
		`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('n1','N','n','active',1,1)`,
		`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES ('n1',1,'{"dispute_window_days":30}',1)`,
		`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES ('m1','n1','p1','A','active','invite',1),('m2','n1','p2','B','active','invite',1)`,
		`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES ('a1','n1','m1',0,-100),('a2','n1','m2',0,100)`,
		`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at,created_by)
 VALUES ('t1','n1','trade','proposed','a1','a2',100,1,'m1')`,
		`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES ('e1','n1','t1','a1',-100,1,2),('e2','n1','t1','a2',100,1,2)`,
	} {
		if _, err := st.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	complete := `UPDATE transactions SET status='completed',completed_at=2,ledger_seq=1,accepted_by='m2'`
	if version >= 16 {
		complete += `,dispute_deadline_at=2592000002,dispute_policy_version=1`
	}
	if _, err := st.db.Exec(complete + ` WHERE id='t1'`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
