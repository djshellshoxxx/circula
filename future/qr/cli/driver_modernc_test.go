//go:build sqlite_modernc_test

package cli_test

import _ "modernc.org/sqlite"

const (
	driverName = "sqlite"
	dsnOpts    = "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
)
