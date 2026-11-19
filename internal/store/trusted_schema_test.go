package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// E-02: both normal access and raw backup/restore inspection must distrust
// schema-controlled SQL on every physical connection, including later pool
// growth. A one-time PRAGMA on the first connection is not sufficient.
func TestSQLiteDistrustsSchemaOnEveryConnection(t *testing.T) {
	for _, raw := range []bool{false, true} {
		name := "normal"
		if raw {
			name = "raw"
		}
		t.Run(name, func(t *testing.T) {
			db := openSchemaTestDB(t, raw)
			ctx := context.Background()
			// Keep every acquired connection checked out so the next acquisition
			// creates another physical connection rather than reusing one.
			for i := 0; i < db.Stats().MaxOpenConnections; i++ {
				conn, err := db.Conn(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = conn.Close() }()
				var trusted int
				if err := conn.QueryRowContext(ctx, "PRAGMA trusted_schema").Scan(&trusted); err != nil {
					t.Fatal(err)
				}
				if trusted != 0 {
					t.Errorf("connection %d: trusted_schema = %d, want 0", i, trusted)
				}
			}
		})
	}
}

// E-02: use the existing application-defined search function, without
// registering any test-only global functions. Top-level application queries
// remain usable, but a view from an untrusted database cannot invoke it.
func TestSQLiteRefusesSchemaFunctionCalls(t *testing.T) {
	for _, raw := range []bool{false, true} {
		name := "normal"
		if raw {
			name = "raw"
		}
		t.Run(name, func(t *testing.T) {
			db := openSchemaTestDB(t, raw)
			var folded string
			if err := db.QueryRow(`SELECT circula_fold('CAFÉ')`).Scan(&folded); err != nil || folded != "cafe" {
				t.Fatalf("direct application function: result=%q err=%v", folded, err)
			}
			if _, err := db.Exec(`CREATE VIEW untrusted_function_view AS SELECT circula_fold('CAFÉ') AS folded`); err != nil {
				t.Fatal(err)
			}
			err := db.QueryRow(`SELECT folded FROM untrusted_function_view`).Scan(&folded)
			if err == nil || !strings.Contains(err.Error(), "unsafe use of circula_fold") {
				t.Fatalf("schema function: result=%q err=%v, want unsafe-use refusal", folded, err)
			}
		})
	}
}

func openSchemaTestDB(t *testing.T, raw bool) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema.db")
	var db *sql.DB
	var err error
	if raw {
		db, err = store.OpenRaw(path)
	} else {
		var st *store.SQLite
		st, err = store.Open(context.Background(), path)
		if err == nil {
			db = st.DB()
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
