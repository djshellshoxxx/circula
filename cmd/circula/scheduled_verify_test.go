package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type scheduledVerifyNoWrites struct {
	store.Store
	writes *int
}

func (s scheduledVerifyNoWrites) Update(context.Context, func(store.Tx) error) error {
	*s.writes++
	return errors.New("scheduled verification attempted a database write")
}

// A3-F15 / OPS-016 / DEC-031: hourly verification must not enter the write
// gate, record consistency runs, append audits or change the database/WAL.
// Freeze/thaw decisions remain in memory and observable through metrics/logs.
func TestScheduledVerifyIsReadOnlyAndStillFreezesAndThaws(t *testing.T) {
	ctx := context.Background()
	db := seedDatabase(t, t.TempDir())
	st := tu.Must(store.Open(ctx, db))
	defer func() { _ = st.Close() }()
	writes := 0
	svc := tu.Must(newService(scheduledVerifyNoWrites{st, &writes}, testConfigFor(db)))
	var logs bytes.Buffer
	j := jobRunner{svc: svc, metrics: newMetrics(time.Now()), logger: slog.New(slog.NewTextHandler(&logs, nil))}
	var networkID string
	if err := st.DB().QueryRow(`SELECT id FROM networks`).Scan(&networkID); err != nil {
		t.Fatal(err)
	}
	count := func(q string) int {
		t.Helper()
		var n int
		if err := st.DB().QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	runs0 := count(`SELECT count(*) FROM consistency_runs`)
	audit0 := count(`SELECT count(*) FROM audit_events`)
	snapshot := func() map[string][]byte {
		t.Helper()
		out := map[string][]byte{}
		for _, suffix := range []string{"", "-wal"} {
			data, err := os.ReadFile(db + suffix)
			if err == nil {
				out[suffix] = data
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
		return out
	}
	verify := func(failing bool) {
		t.Helper()
		before := snapshot()
		if err := j.verify(ctx); err != nil {
			t.Fatal(err)
		}
		after := snapshot()
		if len(before) != len(after) {
			t.Fatal("scheduled verification changed database/WAL files")
		}
		for suffix, b := range before {
			if !bytes.Equal(b, after[suffix]) {
				t.Fatalf("scheduled verification changed database%s", suffix)
			}
		}
		if writes != 0 {
			t.Fatalf("scheduled verification entered %d write units", writes)
		}
		if got := svc.IsFrozen(networkID); got != failing {
			t.Fatalf("frozen=%v want %v", got, failing)
		}
		wantOK, wantFailing := 1, 0
		if failing {
			wantOK, wantFailing = 0, 1
		}
		if j.metrics.ledgerOK != wantOK || j.metrics.ledgerFailing != wantFailing || j.metrics.ledgerCheckedAt == 0 {
			t.Fatalf("verification metrics missing outcome: ok=%d failing=%d at=%f", j.metrics.ledgerOK, j.metrics.ledgerFailing, j.metrics.ledgerCheckedAt)
		}
		if count(`SELECT count(*) FROM consistency_runs`) != runs0 || count(`SELECT count(*) FROM audit_events`) != audit0 {
			t.Fatal("scheduled verification persisted a result")
		}
	}
	verify(false)
	verify(false)
	// Deliberate corruption is external to the read-only job under test.
	if _, err := st.DB().Exec(`UPDATE accounts SET balance=balance+7 WHERE rowid=(SELECT min(rowid) FROM accounts)`); err != nil {
		t.Fatal(err)
	}
	verify(true)
	verify(true)
	if !strings.Contains(logs.String(), "network frozen") {
		t.Fatal("failure missing from operational log")
	}
	if _, err := st.DB().Exec(`UPDATE accounts SET balance=balance-7 WHERE rowid=(SELECT min(rowid) FROM accounts)`); err != nil {
		t.Fatal(err)
	}
	verify(false)
	if !strings.Contains(logs.String(), "network unfrozen") {
		t.Fatal("recovery missing from operational log")
	}
	// Explicit checks keep their documented persisted result and audit event.
	explicit := tu.Must(newService(st, testConfigFor(db)))
	rep := tu.Must(explicit.VerifyAndRecordLedger(ctx, domain.SystemActor("explicit-check"), networkID))
	if !rep.OK || rep.RunID == "" || count(`SELECT count(*) FROM consistency_runs`) != runs0+1 || count(`SELECT count(*) FROM audit_events WHERE action='ledger.verified'`) != 1 {
		t.Fatal("explicit check no longer records its result")
	}
}
