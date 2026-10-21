package main

import (
	"context"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

// ARC-015: the hourly job records every run in consistency_runs but writes the
// ledger.verified audit event only on the first run, a failure or a state change.
func TestScheduledVerifyRecordsWithoutFlooding(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := newService(st, testConfigFor(db))
	if err != nil {
		t.Fatal(err)
	}
	count := func(q string) int {
		var n int
		if err := st.DB().QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	runs0 := count(`SELECT COUNT(*) FROM consistency_runs`)
	ev0 := count(`SELECT COUNT(*) FROM audit_events WHERE action = 'ledger.verified'`)
	for i := 0; i < 4; i++ {
		if _, err := verifyNetworks(ctx, svc, st.DB(), "job-verify", "", true); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(`SELECT COUNT(*) FROM consistency_runs`); n-runs0 != 4 {
		t.Errorf("runs recorded = %d, want 4", n)
	}
	if n := count(`SELECT COUNT(*) FROM audit_events WHERE action = 'ledger.verified'`); n-ev0 > 1 {
		t.Errorf("ledger.verified events = %d, want at most 1 (first run only)", n)
	}
}
