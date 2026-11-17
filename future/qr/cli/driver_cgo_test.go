//go:build sqlite_cgo_test

package cli_test

import _ "github.com/mattn/go-sqlite3"

const (
	driverName = "sqlite3"
	dsnOpts    = "?_busy_timeout=5000&_journal_mode=WAL"
)
