//go:build sqlite_cgo_test

package sqlitetest

import _ "github.com/mattn/go-sqlite3" // MIT; test-only driver, see THIRD_PARTY_NOTICES.md

func init() {
	available = true
	driverName = "sqlite3"
}
