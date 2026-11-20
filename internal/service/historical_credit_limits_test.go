package service_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestHistoricalCreditLimitRetainsExpiredAndReplacedOverride(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100 })
	limit := int64(9000)
	expires := w.Clock.Now().Add(time.Hour)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{
		MemberID: w.aM.ID, Limit: &limit, ExpiresAt: &expires, Reason: "temporary capacity"}))
	posted := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 4000)
	w.Clock.Advance(2 * time.Hour)
	lower := int64(10)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{
		MemberID: w.aM.ID, Limit: &lower, Reason: "replace expired capacity"}))
	w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 500)
	if current := w.balance(w.a); current.Balance != -3500 || current.CreditLimit != 10 {
		t.Fatalf("current position = %+v", current)
	}
	report := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !report.OK || report.HistoricalLimits == nil || report.HistoricalLimits.Status != "complete" || report.HistoricalLimits.CheckedDebits != 2 {
		t.Fatalf("present limit reinterpreted earlier valid debits: %+v", report)
	}
	var captured, until, effective int64
	if err := w.Store.DB().QueryRow(`SELECT w.limit_override,w.limit_override_until,w.effective_limit
 FROM posting_credit_checks w JOIN entries e ON e.id=w.entry_id WHERE e.tx_id=?`, posted.ID).Scan(&captured, &until, &effective); err != nil {
		t.Fatal(err)
	}
	if captured != 9000 || until != expires.UnixMilli() || effective != 9000 {
		t.Fatalf("posting inputs changed: override=%d until=%d effective=%d", captured, until, effective)
	}
}

func TestHistoricalCreditLimitBindsPolicyAcrossEqualClockChange(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 })
	posted := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 4000)
	policy := w.net.Policy
	policy.DefaultCreditLimit = 1000
	zero := 0
	changed := tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{
		Policy: policy, Reason: "prospective policy", NoticeDays: &zero,
		NoticeReason: "fixture immediate change", IdempotencyKey: w.Key()}))
	if changed.PolicyVersion <= w.net.PolicyVersion {
		t.Fatal("fixture did not append a new policy version")
	}
	var version int
	var checked int64
	if err := w.Store.DB().QueryRow(`SELECT w.policy_version,w.checked_at FROM posting_credit_checks w
 JOIN entries e ON e.id=w.entry_id WHERE e.tx_id=?`, posted.ID).Scan(&version, &checked); err != nil {
		t.Fatal(err)
	}
	if version != w.net.PolicyVersion || checked != w.Clock.Now().UnixMilli() {
		t.Fatalf("policy binding = %d at %d", version, checked)
	}
	if current := w.balance(w.a); current.Balance != -4000 || current.CreditLimit != 1000 {
		t.Fatalf("current position = %+v", current)
	}
	report := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !report.OK || report.HistoricalLimits == nil || report.HistoricalLimits.CheckedDebits != 1 {
		t.Fatalf("equal-clock policy change reinterpreted the debit: %+v", report)
	}
}

func TestHistoricalCreditWitnessFailureRollsBackAndReplays(t *testing.T) {
	w := newWorld(t, nil)
	proposal := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	in := service.TxRef{TxID: proposal.ID, IdempotencyKey: w.Key()}
	if _, err := w.Store.DB().Exec(`CREATE TRIGGER historical_credit_test_fault BEFORE INSERT ON posting_credit_checks
 BEGIN SELECT RAISE(ABORT,'deliberate witness write failure'); END`); err != nil {
		t.Fatal(err)
	}
	before := dbChecksum(t, w.Env)
	if _, err := w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, in); err == nil {
		t.Fatal("posting succeeded despite the witness write failure")
	}
	if got := dbChecksum(t, w.Env); got != before {
		t.Fatal("failed witness changed a balance, entry, sequence, status, audit or idempotency row")
	}
	if _, err := w.Store.DB().Exec(`DROP TRIGGER historical_credit_test_fault`); err != nil {
		t.Fatal(err)
	}
	first := tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, in))
	replay := tu.Must(w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, in))
	if first.ID != replay.ID || replay.Status != domain.TxCompleted {
		t.Fatal("retry or durable replay changed the posting")
	}
	var witnesses, entries int
	if err := w.Store.DB().QueryRow(`SELECT (SELECT COUNT(*) FROM posting_credit_checks),
 (SELECT COUNT(*) FROM entries)`).Scan(&witnesses, &entries); err != nil || witnesses != 1 || entries != 2 {
		t.Fatalf("witnesses=%d entries=%d err=%v", witnesses, entries, err)
	}
	report := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !report.OK || report.HistoricalLimits == nil || report.HistoricalLimits.CheckedDebits != 1 {
		t.Fatalf("retry did not produce one verified debit: %+v", report)
	}
}

func TestHistoricalCreditCoverageSavedRunRoundTrip(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	if _, err := w.Store.DB().Exec(`INSERT INTO consistency_runs(id,network_id,started_at,finished_at,ok,findings)
 VALUES('older-run',?,1,2,1,'')`, w.net.ID); err != nil {
		t.Fatal(err)
	}
	tu.Must(0, w.Store.View(w.Ctx, func(tx store.Tx) error {
		run, err := tx.LastConsistencyRun(w.net.ID)
		if err == nil && (run == nil || run.HistoricalLimits != nil) {
			t.Fatalf("older run invented coverage: %+v", run)
		}
		return err
	}))
	report := tu.Must(w.Svc.VerifyAndRecordLedger(w.Ctx, w.admin, w.net.ID))
	if !report.OK || report.RunID == "" || report.HistoricalLimits == nil || report.HistoricalLimits.Status != "complete" || report.HistoricalLimits.CheckedDebits != 1 {
		t.Fatalf("explicit report lost coverage: %+v", report)
	}
	tu.Must(0, w.Store.View(w.Ctx, func(tx store.Tx) error {
		run, err := tx.LastConsistencyRun(w.net.ID)
		if err == nil && (run == nil || run.ID != report.RunID || !reflect.DeepEqual(run.HistoricalLimits, report.HistoricalLimits)) {
			t.Fatalf("saved report lost its coverage: %+v", run)
		}
		return err
	}))
	var summary string
	if err := w.Store.DB().QueryRow(`SELECT summary_after FROM audit_events WHERE network_id=? AND action=? ORDER BY seq DESC LIMIT 1`,
		w.net.ID, domain.ActVerify).Scan(&summary); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, `"historical_limit_coverage":"complete"`) || !strings.Contains(summary, `"checked_debits":1`) {
		t.Fatalf("explicit verification audit lost coverage: %s", summary)
	}
}

func TestHistoricalCreditCoverageIncludesAllPostingKinds(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.SignupGrant = 10 })
	trade := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	tu.Must(w.adjust(w.admin, w.aM, 50, service.AdjustDebit, "debit fixture"))
	tu.Must(w.adjust(w.admin, w.aM, 25, service.AdjustCredit, "credit fixture"))
	tu.Must(w.reverse(w.admin, trade.ID, 0, "reversal fixture"))
	rows, err := w.Store.DB().Query(`SELECT t.kind,COUNT(*) FROM posting_credit_checks w
 JOIN entries e ON e.id=w.entry_id JOIN transactions t ON t.id=e.tx_id GROUP BY t.kind`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for rows.Next() {
		var kind string
		var count int
		if err := rows.Scan(&kind, &count); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		got[kind] = count
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"grant": 2, "trade": 1, "adjustment": 2, "reversal": 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("witness counts by posting kind = %v, want %v", got, want)
	}
	report := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !report.OK || report.HistoricalLimits == nil || report.HistoricalLimits.Status != "complete" ||
		report.HistoricalLimits.CheckedDebits != 3 || report.HistoricalLimits.SystemDebits != 3 {
		t.Fatalf("posting kinds or system exemption lost: %+v", report)
	}
}
