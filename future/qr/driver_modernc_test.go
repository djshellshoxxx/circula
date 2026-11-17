//go:build sqlite_modernc_test

package qr_test

// Test-only pure-Go driver: modernc.org/sqlite, the driver Circula's core uses.
import _ "modernc.org/sqlite"

const (
	driverName = "sqlite"
	haveDriver = true
	dsnOpts    = "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=foreign_keys(1)"
)
