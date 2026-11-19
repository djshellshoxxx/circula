package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	sqlite "modernc.org/sqlite"
)

// storageFaultConnector keeps the real SQLite engine and transaction handling.
// It fails one driver operation before forwarding it, so tests can exercise
// recovery from a failed ROLLBACK and a late migration-metadata read error
// without pretending to qualify physical disk or fsync failures.
type storageFaultConnector struct {
	driver *sqlite.Driver
	dsn    string
	mu     sync.Mutex
	// rollback fails once; migrationRead fails after the first metadata row.
	rollback, migrationRead error
	rollbackFailures        int
	statement               string
	statementErr            error
}

func (c *storageFaultConnector) Driver() driver.Driver { return c.driver }

func (c *storageFaultConnector) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.driver.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &storageFaultConn{Conn: conn, faults: c}, nil
}

type storageFaultConn struct {
	driver.Conn
	faults *storageFaultConnector
}

func (c *storageFaultConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.faults.mu.Lock()
	var statementErr error
	if query == c.faults.statement {
		statementErr = c.faults.statementErr
		c.faults.statementErr = nil
	}
	c.faults.mu.Unlock()
	if statementErr != nil {
		return nil, statementErr
	}
	if strings.EqualFold(strings.TrimSpace(query), "ROLLBACK") {
		c.faults.mu.Lock()
		err := c.faults.rollback
		c.faults.rollback = nil
		if err != nil {
			c.faults.rollbackFailures++
		}
		c.faults.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

func (c *storageFaultConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	rows, err := c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	if strings.Join(strings.Fields(query), " ") == "SELECT version, name FROM schema_migrations" {
		c.faults.mu.Lock()
		fault := c.faults.migrationRead
		c.faults.migrationRead = nil
		c.faults.mu.Unlock()
		if fault == nil {
			return rows, nil
		}
		// The metadata read fault is deferred until Rows.Next, after one row.
		return &storageFaultRows{Rows: rows, err: fault}, nil
	}
	return rows, nil
}

type storageFaultRows struct {
	driver.Rows
	err  error
	read bool
}

func (r *storageFaultRows) Next(values []driver.Value) error {
	if r.read {
		return r.err
	}
	r.read = true
	return r.Rows.Next(values)
}

func openFaultStore(t *testing.T, path string) (*SQLite, *storageFaultConnector) {
	t.Helper()
	faults := &storageFaultConnector{driver: &sqlite.Driver{}, dsn: fileURI(path)}
	db := sql.OpenDB(faults)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return &SQLite{db: db, gate: make(chan struct{}, 1)}, faults
}

// A callback can fail or panic after staging writes. If ROLLBACK itself then
// fails, the live transaction must not be returned to the pool. Closing the
// bad connection lets SQLite discard its uncommitted state, and the next
// request must succeed independently on a clean connection.
func TestFailedRollbackDiscardsConnection(t *testing.T) {
	for _, callback := range []string{"error", "panic"} {
		t.Run(callback, func(t *testing.T) {
			ctx := context.Background()
			st, faults := openFaultStore(t, filepath.Join(t.TempDir(), "rollback.db"))
			if _, err := st.db.ExecContext(ctx, "CREATE TABLE recovery_probe (id TEXT PRIMARY KEY)"); err != nil {
				t.Fatal(err)
			}
			original := errors.New("callback refused the operation")
			faults.rollback = errors.New("injected rollback driver failure")
			operation := func(tx Tx) error {
				if _, err := tx.(*txImpl).exec("INSERT INTO recovery_probe VALUES ('uncommitted')"); err != nil {
					return err
				}
				if callback == "panic" {
					panic(original)
				}
				return original
			}
			if callback == "panic" {
				func() {
					defer func() {
						if got := recover(); got != original {
							t.Errorf("original panic not preserved: %v", got)
						}
					}()
					_ = st.Update(ctx, operation)
				}()
			} else if err := st.Update(ctx, operation); !errors.Is(err, original) {
				t.Fatalf("original callback error not preserved: %v", err)
			}
			if faults.rollbackFailures != 1 {
				t.Fatalf("ROLLBACK failure not reached: %d", faults.rollbackFailures)
			}
			if err := st.Update(ctx, func(tx Tx) error {
				_, err := tx.(*txImpl).exec("INSERT INTO recovery_probe VALUES ('committed')")
				return err
			}); err != nil {
				t.Fatalf("request after failed ROLLBACK could not start a clean transaction: %v", err)
			}
			var count int
			var id string
			if err := st.db.QueryRowContext(ctx, "SELECT COUNT(*), MIN(id) FROM recovery_probe").Scan(&count, &id); err != nil {
				t.Fatal(err)
			}
			if count != 1 || id != "committed" {
				t.Fatalf("uncommitted state escaped: count=%d id=%q", count, id)
			}
		})
	}
}

// A late error while reading schema_migrations must abort before any pending
// migration is attempted. Rows.Close may succeed after Rows.Next failed;
// checking only Close incorrectly treats that incomplete read as successful.
func TestMigrationMetadataReadFailureStopsBeforeDDL(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metadata.db")
	legacy, err := OpenAtVersion(ctx, path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	st, faults := openFaultStore(t, path)
	want := errors.New("injected migration metadata read failure")
	faults.migrationRead = want
	if err := st.migrateTo(ctx, 0); !errors.Is(err, want) {
		t.Errorf("metadata read error not propagated: %v", err)
	}
	var version int
	if err := st.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("migration proceeded after incomplete metadata read: schema=%d, want 1", version)
	}
	// The next read is healthy, so the original database can still upgrade.
	if err := st.migrateTo(ctx, 0); err != nil {
		t.Fatalf("migration did not recover after read fault: %v", err)
	}
	if err := st.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != SchemaVersion() {
		t.Fatalf("recovered migration schema=%d, want %d", version, SchemaVersion())
	}
}

// Both recording a migration and its final COMMIT can fail after the DDL has
// run. The recorded version and entire schema must remain unchanged, including
// when the cleanup command also fails, and a fresh attempt must still work.
func TestMigrationWriteFailureRollsBackAndRecovers(t *testing.T) {
	for _, statement := range []struct{ name, query string }{
		{"metadata", `INSERT INTO schema_migrations(version,name,applied_at) VALUES (?,?,?)`},
		{"commit", "COMMIT"},
	} {
		for _, rollback := range []string{"succeeds", "fails"} {
			t.Run(statement.name+"/rollback_"+rollback, func(t *testing.T) {
				ctx := context.Background()
				path := filepath.Join(t.TempDir(), "migration.db")
				legacy, err := OpenAtVersion(ctx, path, 1)
				if err != nil {
					t.Fatal(err)
				}
				if err := legacy.Close(); err != nil {
					t.Fatal(err)
				}
				st, faults := openFaultStore(t, path)
				schema := func() string {
					t.Helper()
					var result string
					if err := st.db.QueryRowContext(ctx, `SELECT group_concat(type || ':' || name || ':' || COALESCE(sql,''), char(10))
						FROM (SELECT * FROM sqlite_schema ORDER BY type, name)`).Scan(&result); err != nil {
						t.Fatal(err)
					}
					return result
				}
				before := schema()
				want := errors.New("injected migration " + statement.name + " failure")
				faults.statement, faults.statementErr = statement.query, want
				if rollback == "fails" {
					faults.rollback = errors.New("injected rollback driver failure")
				}
				if err := st.migrateTo(ctx, 0); !errors.Is(err, want) {
					t.Errorf("original migration error not propagated: %v", err)
				}
				if rollback == "fails" && faults.rollbackFailures != 1 {
					t.Errorf("rollback fault not reached: %d", faults.rollbackFailures)
				}
				if after := schema(); after != before {
					t.Error("failed migration left partial DDL")
				}
				var version int
				if err := st.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
					t.Fatal(err)
				}
				if version != 1 {
					t.Errorf("failed migration advanced recorded schema to %d", version)
				}
				if err := st.migrateTo(ctx, 0); err != nil {
					t.Fatalf("migration retry after driver failure: %v", err)
				}
				if err := st.db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version); err != nil {
					t.Fatal(err)
				}
				if version != SchemaVersion() {
					t.Fatalf("retry schema=%d, want %d", version, SchemaVersion())
				}
			})
		}
	}
}
