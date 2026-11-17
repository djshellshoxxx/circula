// Package sqlitetest opens real SQLite databases for tests and tooling.
//
// The module imports no SQLite driver in normal builds. With the build tag
// sqlite_cgo_test it registers github.com/mattn/go-sqlite3 (cgo) so tests
// exercise real SQL. This does NOT qualify the production driver
// (modernc.org/sqlite in Circula core); see VALIDATION.md.
package sqlitetest

import (
	"database/sql"
	"net/url"
	"path/filepath"
	"testing"
)

var (
	available  bool
	driverName string
)

// Available reports whether a driver was compiled in.
func Available() bool { return available }

// DriverName is the registered database/sql driver name ("" if none).
func DriverName() string { return driverName }

// DSN builds a connection string with WAL, a busy timeout and full sync.
func DSN(path string) string {
	q := url.Values{}
	q.Set("_busy_timeout", "10000")
	q.Set("_journal_mode", "WAL")
	q.Set("_synchronous", "FULL")
	q.Set("_foreign_keys", "on")
	return "file:" + path + "?" + q.Encode()
}

// OpenFile opens (creating if needed) a database at path.
func OpenFile(path string) (*sql.DB, error) {
	return sql.Open(driverName, DSN(path))
}

// Open returns a database in a temp dir (or at path if non-empty) and closes
// it at cleanup. Without a compiled driver the test is skipped.
func Open(tb testing.TB, path string) *sql.DB {
	tb.Helper()
	if !available {
		tb.Skip("no SQLite driver compiled in: run with -tags=sqlite_cgo_test")
	}
	if path == "" {
		path = filepath.Join(tb.TempDir(), "discovery.db")
	}
	db, err := OpenFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	return db
}
