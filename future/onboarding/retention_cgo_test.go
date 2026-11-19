//go:build sqlite_cgo_test

package onboarding

import "testing"

// Use the existing real-SQLite factory and pinned test driver without changing
// production imports, dependencies or the schema.
func TestSQLPurgeStaleChronology(t *testing.T) {
	purgeCutoffContract(t, openSQL)
}
