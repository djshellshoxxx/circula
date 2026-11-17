//go:build sqlite_cgo_test

package notifications_test

// Test-only driver (mattn/go-sqlite3, MIT, pinned in go.mod). It exercises real
// SQLite but does not qualify the production driver Circula uses.
import _ "github.com/mattn/go-sqlite3"

const (
	driverName = "sqlite3"
	haveDriver = true
	// BEGIN IMMEDIATE is issued by the module itself.
	dsnOpts = "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=FULL&_foreign_keys=on"
)
