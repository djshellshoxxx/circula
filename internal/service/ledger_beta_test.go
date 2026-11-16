package service_test

// Beta qualification tests for the ledger (docs/plans/circula-completion-matrix.md,
// scenarios L3-L6, L9, L12, L15). They close gaps found in the 2026-11-11 coverage
// map: refusals were checked by error code only, fault injection covered Accept
// only, and restart persistence rested on a single CLI scenario.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// ledgerState is everything a refused or failed command must leave untouched.
type ledgerState struct {
	Balances     map[string]int64
	Entries      int
	EntrySum     int64
	Transactions int
	StatusEvents int
	MaxSeq       int64
	Idempotency  int
}

func snapLedger(t *testing.T, e *tu.Env, netID string) ledgerState {
	t.Helper()
	db := e.Store.DB()
	s := ledgerState{Balances: map[string]int64{}}
	rows, err := db.Query(`SELECT id, balance FROM accounts WHERE network_id = ?`, netID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		var b int64
		if err := rows.Scan(&id, &b); err != nil {
			t.Fatal(err)
		}
		s.Balances[id] = b
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	q := func(dst any, sql string) {
		t.Helper()
		if err := db.QueryRow(sql, netID).Scan(dst); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	q(&s.Entries, `SELECT COUNT(*) FROM entries WHERE network_id = ?`)
	q(&s.EntrySum, `SELECT COALESCE(SUM(amount), 0) FROM entries WHERE network_id = ?`)
	q(&s.Transactions, `SELECT COUNT(*) FROM transactions WHERE network_id = ?`)
	q(&s.StatusEvents, `SELECT COUNT(*) FROM transaction_status_events WHERE network_id = ?`)
	q(&s.MaxSeq, `SELECT COALESCE(MAX(ledger_seq), 0) FROM entries WHERE network_id = ?`)
	if err := db.QueryRow(`SELECT COUNT(*) FROM idempotency_keys`).Scan(&s.Idempotency); err != nil {
		t.Fatal(err)
	}
	return s
}

func requireSumZero(t *testing.T, s ledgerState) {
	t.Helper()
	var sum int64
	for _, b := range s.Balances {
		sum += b
	}
	if sum != 0 || s.EntrySum != 0 {
		t.Fatalf("mutual-credit invariant broken: sum(balances)=%d sum(entries)=%d", sum, s.EntrySum)
	}
}

func requireVerified(t *testing.T, e *tu.Env, admin domain.Actor, netID string) {
	t.Helper()
	rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, netID))
	if !rep.OK {
		t.Fatalf("VerifyLedger: %v", rep.Findings)
	}
}

// L3-L6, L13: every refused transfer leaves balances, entries, transactions,
// status events and idempotency records exactly as they were, not just the
// right error code.
func TestRefusedTransfersLeaveLedgerUnchanged(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "refuse", func(p *domain.Policy) { p.DefaultCreditLimit = 10000 })
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	_, cM := e.Member(admin, net, "c@example.org")
	e.Trade(net.ID, a, b, aM, bM, 2500) // a non-trivial starting ledger
	tu.Must(0, e.Svc.SuspendMember(e.Ctx, admin, net.ID, cM.ID, "beta qualification"))

	before := snapLedger(t, e, net.ID)
	requireSumZero(t, before)

	cases := []struct {
		name string
		in   service.ProposeInput
		want error
	}{
		{"zero", service.ProposeInput{CounterpartyID: bM.ID, Direction: service.ProposerPays, Amount: 0}, domain.ErrValidation},
		{"negative", service.ProposeInput{CounterpartyID: bM.ID, Direction: service.ProposerPays, Amount: -1}, domain.ErrValidation},
		{"min-int64", service.ProposeInput{CounterpartyID: bM.ID, Direction: service.ProposerPays, Amount: math.MinInt64}, domain.ErrValidation},
		{"max-int64", service.ProposeInput{CounterpartyID: bM.ID, Direction: service.ProposerPays, Amount: math.MaxInt64}, nil},
		{"self", service.ProposeInput{CounterpartyID: aM.ID, Direction: service.ProposerPays, Amount: 100}, domain.ErrValidation},
		{"unknown-counterparty", service.ProposeInput{CounterpartyID: "00000000-0000-7000-8000-000000000000", Direction: service.ProposerPays, Amount: 100}, domain.ErrNotFound},
		{"suspended-counterparty", service.ProposeInput{CounterpartyID: cM.ID, Direction: service.ProposerPays, Amount: 100}, domain.ErrCounterpartyInact},
		{"suspended-counterparty-request", service.ProposeInput{CounterpartyID: cM.ID, Direction: service.ProposerIsPaid, Amount: 100}, domain.ErrCounterpartyInact},
		{"bad-direction", service.ProposeInput{CounterpartyID: bM.ID, Direction: "sideways", Amount: 100}, domain.ErrValidation},
		{"over-limit-by-one", service.ProposeInput{CounterpartyID: bM.ID, Direction: service.ProposerPays, Amount: 10000 - 2500 + 1}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.in.IdempotencyKey = e.Key()
			tx, err := e.Svc.ProposeTransaction(e.Ctx, a, net.ID, c.in)
			if err == nil {
				t.Fatalf("accepted: %+v", tx)
			}
			if c.want != nil && !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			var de *domain.Error
			if !errors.As(err, &de) {
				t.Fatalf("not a domain error: %v", err)
			}
			if after := snapLedger(t, e, net.ID); !reflect.DeepEqual(before, after) {
				t.Fatalf("ledger changed by a refused transfer:\nbefore %+v\nafter  %+v", before, after)
			}
		})
	}

	// A suspended member cannot accept, and that refusal is also clean.
	p := tu.Must(e.Svc.ProposeTransaction(e.Ctx, b, net.ID, service.ProposeInput{CounterpartyID: aM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: e.Key()}))
	before = snapLedger(t, e, net.ID)
	tu.Must(0, e.Svc.SuspendMember(e.Ctx, admin, net.ID, aM.ID, "beta qualification"))
	if _, err := e.Svc.AcceptTransaction(e.Ctx, a, net.ID, service.TxRef{TxID: p.ID, IdempotencyKey: e.Key()}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("suspended payee accepted: %v", err)
	}
	if after := snapLedger(t, e, net.ID); !reflect.DeepEqual(before.Balances, after.Balances) || before.Entries != after.Entries {
		t.Fatalf("ledger changed by a suspended accept:\nbefore %+v\nafter  %+v", before, after)
	}
	requireSumZero(t, snapLedger(t, e, net.ID))
	requireVerified(t, e, admin, net.ID)
}

// L9, L11: a fault at any write of a reversal or an adjustment rolls the whole
// unit back. The original transaction stays completed and unreversed, and the
// same idempotency key then succeeds exactly once.
func TestReversalAndAdjustmentFaultInjectionLeavesNoTrace(t *testing.T) {
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "fault2", nil)
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	orig := e.Trade(net.ID, a, b, aM, bM, 1500)

	steps := []struct {
		op  string
		nth int
	}{{"InsertEntry", 1}, {"InsertEntry", 2}, {"AddBalance", 1}, {"AddBalance", 2}, {"UpdateTransaction", 1},
		{"InsertStatusEvent", 1}, {"InsertAudit", 1}, {"PutIdem", 1}}

	ops := []struct {
		name string
		run  func(key string) error
		done func(t *testing.T)
	}{
		{"reversal", func(key string) error {
			_, err := e.Svc.ReverseTransaction(e.Ctx, admin, net.ID, service.ReverseInput{TxID: orig.ID, Reason: "beta fault drill", IdempotencyKey: key})
			return err
		}, func(t *testing.T) {
			v := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, orig.ID))
			if v.Tx.Status != domain.TxReversed {
				t.Fatalf("original status %s after successful reversal", v.Tx.Status)
			}
		}},
		{"adjustment", func(key string) error {
			_, err := e.Svc.CreateAdjustment(e.Ctx, admin, net.ID, service.AdjustInput{MemberID: aM.ID, Amount: 700,
				Direction: service.AdjustCredit, Reason: "beta fault drill", IdempotencyKey: key})
			return err
		}, func(t *testing.T) {}},
	}
	for _, op := range ops {
		t.Run(op.name, func(t *testing.T) {
			key := "beta-" + op.name
			for _, st := range steps {
				t.Run(fmt.Sprintf("%s#%d", st.op, st.nth), func(t *testing.T) {
					before := snapLedger(t, e, net.ID)
					auditBefore := countRows(t, e, "audit_events")
					origBefore := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, orig.ID)).Tx.Status
					fs.Arm(st.op, st.nth)
					err := op.run(key)
					fs.Disarm()
					if !errors.Is(err, tu.ErrInjected) {
						t.Fatalf("want injected fault, got %v", err)
					}
					if after := snapLedger(t, e, net.ID); !reflect.DeepEqual(before, after) {
						t.Fatalf("partial commit:\nbefore %+v\nafter  %+v", before, after)
					}
					if got := countRows(t, e, "audit_events"); got != auditBefore {
						t.Fatalf("audit events %d -> %d", auditBefore, got)
					}
					v := tu.Must(e.Svc.GetTransaction(e.Ctx, a, net.ID, orig.ID))
					if v.Tx.Status != origBefore {
						t.Fatalf("original status %s after failed unit", v.Tx.Status)
					}
					requireVerified(t, e, admin, net.ID)
				})
			}
			before := snapLedger(t, e, net.ID)
			if err := op.run(key); err != nil {
				t.Fatalf("clean retry: %v", err)
			}
			if err := op.run(key); err != nil { // replay: same key, same body
				t.Fatalf("replay: %v", err)
			}
			after := snapLedger(t, e, net.ID)
			if after.Entries != before.Entries+2 || after.MaxSeq != before.MaxSeq+1 {
				t.Fatalf("retry+replay posted %d entries (seq %d -> %d), want exactly one posting",
					after.Entries-before.Entries, before.MaxSeq, after.MaxSeq)
			}
			requireSumZero(t, after)
			op.done(t)
			requireVerified(t, e, admin, net.ID)
		})
	}
}

// L15: closing the database and opening it again, as a restart does, keeps
// balances, entries, transaction states and history, the ledger still verifies,
// and new postings continue the ledger sequence.
func TestReopenPreservesLedger(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "reopen", nil)
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	t1 := e.Trade(net.ID, a, b, aM, bM, 1200)
	e.Trade(net.ID, b, a, bM, aM, 300)
	tu.Must(e.Svc.ReverseTransaction(e.Ctx, admin, net.ID, service.ReverseInput{TxID: t1.ID, Amount: 200, Reason: "partial", IdempotencyKey: e.Key()}))
	pending := tu.Must(e.Svc.ProposeTransaction(e.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: bM.ID,
		Direction: service.ProposerPays, Amount: 50, IdempotencyKey: e.Key()}))
	before := snapLedger(t, e, net.ID)
	histBefore := tu.Must(e.Svc.ListTransactions(e.Ctx, a, net.ID, nil, service.Page{}))

	var seq int
	var name, path string
	if err := e.Store.DB().QueryRow(`PRAGMA database_list`).Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.Close(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc, err := service.New(st, service.Config{Clock: e.Clock,
		Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	e2 := &tu.Env{T: t, Ctx: ctx, Store: st, Svc: svc, Clock: e.Clock}

	if after := snapLedger(t, e2, net.ID); !reflect.DeepEqual(before, after) {
		t.Fatalf("ledger changed across reopen:\nbefore %+v\nafter  %+v", before, after)
	}
	histAfter := tu.Must(svc.ListTransactions(ctx, a, net.ID, nil, service.Page{}))
	if len(histAfter) != len(histBefore) {
		t.Fatalf("history length %d -> %d", len(histBefore), len(histAfter))
	}
	for i := range histBefore {
		x, y := histBefore[i], histAfter[i]
		if x.ID != y.ID || x.Status != y.Status || x.Amount != y.Amount || x.LedgerSeq != y.LedgerSeq {
			t.Fatalf("history row %d differs: %+v vs %+v", i, x, y)
		}
	}
	requireVerified(t, e2, admin, net.ID)

	// The proposal survives the restart and can still be accepted, once.
	tu.Must(svc.AcceptTransaction(ctx, b, net.ID, service.TxRef{TxID: pending.ID, IdempotencyKey: "after-restart"}))
	final := snapLedger(t, e2, net.ID)
	if final.MaxSeq != before.MaxSeq+1 {
		t.Fatalf("ledger sequence %d -> %d, want +1", before.MaxSeq, final.MaxSeq)
	}
	requireSumZero(t, final)
	requireVerified(t, e2, admin, net.ID)
}

// L10, L12: if the append-only triggers are bypassed (direct file edit) and an
// entry amount is altered, verification reports it. The cached balance no longer
// matches the entries, and the network total is no longer zero.
func TestVerifyDetectsTamperedEntry(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "tamper", nil)
	a, aM := e.Member(admin, net, "a@example.org")
	b, bM := e.Member(admin, net, "b@example.org")
	e.Trade(net.ID, a, b, aM, bM, 900)
	requireVerified(t, e, admin, net.ID)

	db := e.Store.DB()
	if _, err := db.Exec(`UPDATE entries SET amount = amount + 1 WHERE network_id = ? AND amount > 0`, net.ID); err == nil {
		t.Fatal("append-only trigger did not block an entry update")
	}
	if _, err := db.Exec(`DROP TRIGGER entries_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE entries SET amount = amount + 1 WHERE network_id = ? AND amount > 0`, net.ID); err != nil {
		t.Fatal(err)
	}
	rep := tu.Must(e.Svc.VerifyLedger(e.Ctx, admin, net.ID))
	if rep.OK || len(rep.Findings) == 0 {
		t.Fatalf("tampered entry not reported: %+v", rep)
	}
}
