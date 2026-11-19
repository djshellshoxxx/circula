package store

import (
	"fmt"
	"net/url"
	"testing"
)

// CommitFaultStats and OpenCommitFaultStore exist only in the augmented test
// package. The external service-level regression can reuse testutil's real
// fixture and raw-SQL snapshot without a store -> testutil -> store import cycle.
// No test hook or fault mode is compiled into production.
type CommitFaultStats struct {
	Opened, Closed                        int
	CommitAttempts, Commits               int
	BeforeCommitErrors, AfterCommitErrors int
	RollbackAttempts, RollbackErrors      int
}

// OpenCommitFaultStore opens an already migrated database through the existing
// driver-boundary connector. arm selects a one-shot error before COMMIT or,
// when afterCommit is true, only after the underlying COMMIT has succeeded.
func OpenCommitFaultStore(t *testing.T, path string) (
	st *SQLite, arm func(afterCommit bool, err error), stats func() CommitFaultStats,
) {
	t.Helper()
	st, faults := openFaultStore(t, path)
	// sql.OpenDB is lazy: set the production default pragmas before the first
	// connection opens. The regression checks these on the replacement too.
	q := url.Values{}
	for _, pragma := range []string{
		"trusted_schema(0)", fmt.Sprintf("busy_timeout(%d)", DefaultBusyTimeout.Milliseconds()),
		"journal_mode(WAL)", "synchronous(" + DefaultSynchronous + ")", "foreign_keys(1)", "secure_delete(1)",
	} {
		q.Add("_pragma", pragma)
	}
	faults.dsn += "?" + q.Encode()
	arm = func(afterCommit bool, err error) {
		faults.mu.Lock()
		defer faults.mu.Unlock()
		faults.statement, faults.statementErr, faults.commitAcknowledgmentErr = "COMMIT", nil, nil
		if afterCommit {
			faults.commitAcknowledgmentErr = err
		} else {
			faults.statementErr = err
		}
	}
	stats = func() CommitFaultStats {
		faults.mu.Lock()
		defer faults.mu.Unlock()
		return faults.stats
	}
	return st, arm, stats
}
