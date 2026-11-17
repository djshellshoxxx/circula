//go:build sqlite_cgo_test

// Package sqlitefixture opens real SQLite files for tests using the cgo
// driver github.com/mattn/go-sqlite3. It is compiled only with the
// sqlite_cgo_test build tag, so the module's default build has no driver
// dependency. It exercises real SQL, WAL, locking and file durability but does
// NOT qualify the production driver (modernc.org/sqlite) used by the host.
package sqlitefixture

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// DSN returns a file DSN with WAL, a busy timeout and foreign keys.
func DSN(path string) string {
	return "file:" + path + "?_journal_mode=WAL&_busy_timeout=10000&_foreign_keys=on&_synchronous=FULL"
}

// Path returns a database path inside a fresh temp directory.
func Path(t testing.TB) string { return filepath.Join(t.TempDir(), "favorites.db") }

// Open opens (creating if needed) the database file at path.
func Open(t testing.TB, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Driver is the registered database/sql driver name.
const Driver = "sqlite3"

// DriverDSN exposes the DSN for command-line wiring.
func DriverDSN(path string) string { return DSN(path) }
