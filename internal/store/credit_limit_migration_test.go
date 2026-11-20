package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/integrity"
)

func creditHistoryReport(t *testing.T, st *SQLite) *integrity.Report {
	t.Helper()
	var report *integrity.Report
	if err := st.View(context.Background(), func(tx Tx) error {
		var err error
		report, err = integrity.CheckLedger(tx, "n1", integrity.Options{})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return report
}

// A controlled fixture adds a balanced posting with optional decision inputs.
// It changes no old entry, policy or immutability guard.
func appendCreditHistoryFixture(t *testing.T, st *SQLite, id string, seq, at int64, witness *PostingCreditCheck) {
	t.Helper()
	err := st.Update(context.Background(), func(tx Tx) error {
		q := tx.(*txImpl)
		if _, err := q.exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at,created_by)
 VALUES (?,'n1','trade','proposed','a1','a2',1,1,'m1')`, id); err != nil {
			return err
		}
		if _, err := q.exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at)
 VALUES (?,'n1',?,'a1',-1,?,?),(?,'n1',?,'a2',1,?,?)`, id+"-d", id, seq, at, id+"-c", id, seq, at); err != nil {
			return err
		}
		if witness != nil {
			witness.EntryID, witness.NetworkID = id+"-d", "n1"
			if err := tx.InsertPostingCreditCheck(*witness); err != nil {
				return err
			}
		}
		if _, err := q.exec(`UPDATE accounts SET balance=balance-1 WHERE id='a1'; UPDATE accounts SET balance=balance+1 WHERE id='a2'`); err != nil {
			return err
		}
		if _, err := q.exec(`UPDATE transactions SET status='completed',completed_at=?,ledger_seq=?,accepted_by='m2',
 dispute_deadline_at=?,dispute_policy_version=1 WHERE id=?`, at, seq, at+2592000000, id); err != nil {
			return err
		}
		_, err := q.exec(`UPDATE networks SET ledger_seq=? WHERE id='n1'`, seq)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalCreditMigrationFreezesExactLegacyEntryMembership(t *testing.T) {
	ctx := context.Background()
	path := migrationIntegrityFixture(t, 23)
	st, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	report := creditHistoryReport(t, st)
	if len(report.Findings) != 0 || report.HistoricalLimits == nil || report.HistoricalLimits.Status != "partial" || report.HistoricalLimits.LegacyUnverifiedDebits != 1 {
		t.Fatalf("old inputs were invented or misclassified: %+v", report)
	}
	var legacy, populated int
	if err := st.db.QueryRow(`SELECT legacy_unverified,
 checked_at IS NOT NULL OR policy_version IS NOT NULL OR balance_before IS NOT NULL OR is_system IS NOT NULL
 OR limit_override IS NOT NULL OR limit_override_until IS NOT NULL OR effective_limit IS NOT NULL
 FROM posting_credit_checks WHERE entry_id='e1'`).Scan(&legacy, &populated); err != nil || legacy != 1 || populated != 0 {
		t.Fatalf("legacy marker = %d populated=%d err=%v", legacy, populated, err)
	}
	// A later identity sharing the old clock and sequence still needs evidence.
	appendCreditHistoryFixture(t, st, "later", 1, 2, nil)
	for _, query := range []string{
		`INSERT INTO posting_credit_checks(entry_id,network_id,legacy_unverified) VALUES('later-d','n1',1)`,
		`UPDATE posting_credit_checks SET checked_at=2 WHERE entry_id='e1'`,
		`DELETE FROM posting_credit_checks WHERE entry_id='e1'`,
		`INSERT OR REPLACE INTO posting_credit_checks(entry_id,network_id,legacy_unverified,checked_at,policy_version,balance_before,is_system,effective_limit)
 VALUES('e1','n1',0,2,1,0,0,100)`,
	} {
		if _, err := st.db.Exec(query); err == nil {
			t.Fatalf("immutable adoption boundary accepted %s", query)
		}
	}
	report = creditHistoryReport(t, st)
	if report.HistoricalLimits.LegacyUnverifiedDebits != 1 || report.HistoricalLimits.UnverifiableDebits != 1 ||
		!strings.Contains(strings.Join(report.Findings, "\n"), "later-d historical credit check: forward debit has no immutable witness") {
		t.Fatalf("new debit escaped the adoption boundary: %+v", report)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()
	if got := creditHistoryReport(t, reopened); !reflect.DeepEqual(got.HistoricalLimits, report.HistoricalLimits) || !reflect.DeepEqual(got.Findings, report.Findings) {
		t.Fatal("adoption boundary or missing-witness finding changed after reopen")
	}
}

func TestHistoricalCreditCheckDetectsSeededInvalidForwardDebit(t *testing.T) {
	st, err := Open(context.Background(), migrationIntegrityFixture(t, 23))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	appendCreditHistoryFixture(t, st, "later", 2, 2, &PostingCreditCheck{
		CheckedAt: time.UnixMilli(2), PolicyVersion: 1, BalanceBefore: -100, EffectiveLimit: 0,
	})
	report := creditHistoryReport(t, st)
	if report.HistoricalLimits.CheckedDebits != 1 || report.HistoricalLimits.LegacyUnverifiedDebits != 1 ||
		!strings.Contains(strings.Join(report.Findings, "\n"), "below its then-effective limit 0") {
		t.Fatalf("historical debit condition was not evaluated: %+v", report)
	}
}

func TestHistoricalCreditReadFailureCannotReturnPartialCoverage(t *testing.T) {
	path := migrationIntegrityFixture(t, 23)
	ready, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ready.Close(); err != nil {
		t.Fatal(err)
	}
	st, faults := openFaultStore(t, path)
	want := errors.New("injected late credit history read")
	faults.readPrefix, faults.readErr = "SELECT e.id,e.network_id,e.tx_id,e.account_id", want
	err = st.View(context.Background(), func(tx Tx) error {
		report, err := integrity.CheckLedger(tx, "n1", integrity.Options{})
		if report != nil {
			t.Error("incomplete history produced a verification report")
		}
		return err
	})
	if !errors.Is(err, want) || faults.readErr != nil {
		t.Fatalf("late history read was accepted: %v", err)
	}
	if got := creditHistoryReport(t, st); len(got.Findings) != 0 || got.HistoricalLimits.LegacyUnverifiedDebits != 1 {
		t.Fatalf("healthy retry failed: %+v", got)
	}
}

func TestHistoricalCreditMigrationRollbackPreservesOriginalSchema(t *testing.T) {
	path := migrationIntegrityFixture(t, 23)
	st, faults := openFaultStore(t, path)
	before := readMigrationIntegrityState(t, st.db)
	want := errors.New("injected credit migration metadata failure")
	faults.statement = `INSERT INTO schema_migrations(version,name,applied_at) VALUES (?,?,?)`
	faults.statementErr = want
	if err := st.migrateTo(context.Background(), 0); !errors.Is(err, want) {
		t.Fatalf("metadata fault was not propagated: %v", err)
	}
	if after := readMigrationIntegrityState(t, st.db); after != before {
		t.Fatal("failed credit migration changed original schema, metadata or ledger")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationIntegrityRetry(t, path, 23, before)
}
