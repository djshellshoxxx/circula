package store_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
	_ "modernc.org/sqlite"
)

// QA-027: a database left at every earlier released schema version, holding a
// small ledger, migrates to head with data intact and integrity checks clean;
// a second open changes nothing.
func TestUpgradeFromEverySchemaVersion(t *testing.T) {
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations not found: %v", err)
	}
	sort.Strings(files)
	head := store.SchemaVersion()
	if len(files) != head {
		t.Fatalf("%d files, SchemaVersion %d", len(files), head)
	}
	for k := 1; k < head; k++ {
		t.Run(strings.TrimSuffix(filepath.Base(files[k-1]), ".sql"), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "old.db")
			raw, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := raw.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL)`); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < k; i++ {
				b, err := os.ReadFile(files[i])
				if err != nil {
					t.Fatal(err)
				}
				name := strings.SplitN(strings.TrimSuffix(filepath.Base(files[i]), ".sql"), "_", 2)[1]
				if _, err := raw.Exec(string(b)); err != nil {
					t.Fatalf("apply %s: %v", files[i], err)
				}
				if _, err := raw.Exec(`INSERT INTO schema_migrations VALUES (?,?,0)`, i+1, name); err != nil {
					t.Fatal(err)
				}
			}
			seedV1Ledger(t, raw)
			if err := raw.Close(); err != nil {
				t.Fatal(err)
			}

			for pass := 0; pass < 2; pass++ { // second pass: migrating again changes nothing
				s, err := store.Open(context.Background(), path)
				if err != nil {
					t.Fatalf("pass %d: upgrade from v%d: %v", pass, k, err)
				}
				db := s.DB()
				var v int
				var ic string
				if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil || v != head {
					t.Fatalf("version %d err %v want %d", v, err, head)
				}
				if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&ic); err != nil || ic != "ok" {
					t.Fatalf("integrity_check: %q %v", ic, err)
				}
				rows, err := db.Query(`PRAGMA foreign_key_check`)
				if err != nil {
					t.Fatal(err)
				}
				if rows.Next() {
					t.Fatal("foreign_key_check reports violations after upgrade")
				}
				_ = rows.Close()
				var sumBal, sumEnt, nEnt int64
				if err := db.QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts`).Scan(&sumBal); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM entries`).Scan(&sumEnt, &nEnt); err != nil {
					t.Fatal(err)
				}
				if sumBal != 0 || sumEnt != 0 || nEnt != 2 {
					t.Fatalf("ledger damaged: balances %d entries %d (%d rows)", sumBal, sumEnt, nEnt)
				}
				var bal int64
				if err := db.QueryRow(`SELECT balance FROM accounts WHERE id = 'a2'`).Scan(&bal); err != nil || bal != 2500 {
					t.Fatalf("payee balance %d err %v", bal, err)
				}
				var n int
				if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil || n != head {
					t.Fatalf("%d migration rows, want %d", n, head)
				}
				_ = s.Close()
			}
		})
	}
}

// seedV1Ledger inserts one completed 25.00 trade using only schema-1 columns.
func seedV1Ledger(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO persons(id,login,password_hash,created_at) VALUES ('p1','p1','h',1),('p2','p2','h',1)`,
		`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('n1','N','n','active',1,1)`,
		`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES ('m1','n1','p1','A','active','invite',1),('m2','n1','p2','B','active','invite',1)`,
		`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES ('a1','n1','m1',0,-2500),('a2','n1','m2',0,2500)`,
		`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at,completed_at,ledger_seq,created_by,accepted_by)
		 VALUES ('t1','n1','trade','proposed','a1','a2',2500,1,NULL,NULL,'m1',NULL)`,
		`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES ('e1','n1','t1','a1',-2500,1,2),('e2','n1','t1','a2',2500,1,2)`,
		`UPDATE transactions SET status='completed', completed_at=2, ledger_seq=1, accepted_by='m2' WHERE id='t1'`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
}
