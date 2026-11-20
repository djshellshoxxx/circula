package store_test

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// LED-4 / LED-6 / L9: distinguish a COMMIT rejected before execution from a
// successful COMMIT whose driver acknowledgment is lost. This models the
// application/driver error boundary, not a physical I/O or power-loss fault.
// The latter must preserve the committed unit despite its reported error and
// failed cleanup ROLLBACK; retries must find its durable idempotency record.
func TestCommitAcknowledgmentFailurePreservesPostingAndReplay(t *testing.T) {
	for _, afterCommit := range []bool{false, true} {
		name := "before_commit"
		if afterCommit {
			name = "after_successful_commit"
		}
		t.Run(name, func(t *testing.T) {
			fx := tu.NewFixture(t)
			propose := func(amount int64) *domain.Transaction {
				t.Helper()
				return tu.Must(fx.Svc.ProposeTransaction(fx.Ctx, fx.Alice, fx.Net.ID, service.ProposeInput{
					CounterpartyID: fx.BobM.ID, Direction: service.ProposerPays, Amount: amount, IdempotencyKey: fx.Key()}))
			}
			// An earlier committed trade must survive both failure outcomes.
			previous := propose(1500)
			tu.Must(fx.Svc.AcceptTransaction(fx.Ctx, fx.Bob, fx.Net.ID, service.TxRef{TxID: previous.ID, IdempotencyKey: fx.Key()}))
			pending := propose(300)
			ref := service.TxRef{TxID: pending.ID, IdempotencyKey: "commit-acknowledgment-loss"}
			var seq int
			var database, path string
			if err := fx.Store.DB().QueryRow("PRAGMA database_list").Scan(&seq, &database, &path); err != nil {
				t.Fatal(err)
			}
			if err := fx.Store.Close(); err != nil {
				t.Fatal(err)
			}
			st, arm, stats := store.OpenCommitFaultStore(t, path)
			cfg := service.Config{Clock: fx.Clock, Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}}
			svc := tu.Must(service.New(st, cfg))
			checkCommitFaultConnection(t, st.DB())
			before := tu.Snapshot(t, st.DB())
			original := readCommitFaultPosting(t, st.DB(), fx, pending.ID, ref.IdempotencyKey)
			committed := original
			committed.entries += 2
			committed.tradeEntries = 2
			committed.alice -= pending.Amount
			committed.bob += pending.Amount
			committed.status = string(domain.TxCompleted)
			committed.ledgerSeq++
			committed.tradeSeq = committed.ledgerSeq
			committed.audits++
			committed.statusEvents++
			committed.notifications++
			committed.idempotency = 1
			committed.result = pending.ID
			counts := stats()
			wantErr := errors.New("injected COMMIT acknowledgment failure")
			arm(afterCommit, wantErr)
			_, err := svc.AcceptTransaction(fx.Ctx, fx.Bob, fx.Net.ID, ref)
			if !errors.Is(err, wantErr) || !strings.HasPrefix(err.Error(), "store: commit:") || domain.IsRetryable(err) {
				t.Fatalf("original nonretryable COMMIT error not preserved: %v", err)
			}
			after := stats()
			wantCounts := counts
			wantCounts.CommitAttempts++
			wantCounts.RollbackAttempts++
			if afterCommit {
				wantCounts.Commits++
				wantCounts.AfterCommitErrors++
				wantCounts.RollbackErrors++ // real SQLite has no transaction left
				wantCounts.Closed++         // production rollbackOrDiscard discards it
			} else {
				wantCounts.BeforeCommitErrors++
			}
			if after != wantCounts {
				t.Fatalf("fault or cleanup boundary was not reached: got %+v, want %+v", after, wantCounts)
			}
			state := tu.Snapshot(t, st.DB())
			wantPosting := original
			if afterCommit {
				wantPosting = committed
			} else if changed := before.Changed(state); len(changed) != 0 {
				t.Fatalf("COMMIT rejected before execution changed state: %v", changed)
			}
			if got := readCommitFaultPosting(t, st.DB(), fx, pending.ID, ref.IdempotencyKey); got != wantPosting {
				t.Fatalf("error left an incomplete or incorrect financial unit: got %+v, want %+v", got, wantPosting)
			}
			// A real fresh BEGIN IMMEDIATE must work on the live pool, before a
			// reopen could conceal failed connection cleanup. This writes nothing.
			if err := st.Update(fx.Ctx, func(tx store.Tx) error {
				_, err := tx.GetNetwork(fx.Net.ID)
				return err
			}); err != nil {
				t.Fatalf("next live transaction failed: %v", err)
			}
			checkCommitFaultConnection(t, st.DB())
			wantOpened := counts.Opened
			if afterCommit {
				wantOpened++
			}
			if got := stats(); got.Opened != wantOpened || got.Closed != wantCounts.Closed ||
				got.AfterCommitErrors != wantCounts.AfterCommitErrors || got.BeforeCommitErrors != wantCounts.BeforeCommitErrors {
				t.Fatalf("connection replacement or one-shot injection changed: %+v", got)
			}
			if changed := state.Changed(tu.Snapshot(t, st.DB())); len(changed) != 0 {
				t.Fatalf("connection-health check changed durable state: %v", changed)
			}
			verify := func() {
				t.Helper()
				rep, err := svc.VerifyLedger(fx.Ctx, domain.SystemActor("commit-ack-test"), fx.Net.ID)
				if err != nil || rep == nil || !rep.OK {
					t.Fatalf("ledger or audit chain failed verification: %v %+v", err, rep)
				}
				var integrity string
				if err := st.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
					t.Fatalf("SQLite integrity: %q %v", integrity, err)
				}
			}
			verify()
			reopen := func(want tu.State) {
				t.Helper()
				if err := st.Close(); err != nil {
					t.Fatal(err)
				}
				st = tu.Must(store.Open(fx.Ctx, path))
				opened := st
				t.Cleanup(func() { _ = opened.Close() })
				svc = tu.Must(service.New(st, cfg))
				if changed := want.Changed(tu.Snapshot(t, st.DB())); len(changed) != 0 {
					t.Fatalf("reopen changed the complete stable-table snapshot: %v", changed)
				}
				verify()
			}
			// Reopen before retry: the error's actual outcome must already be
			// durable, rather than becoming correct only through a later command.
			reopen(state)
			retry := func() {
				t.Helper()
				result, err := svc.AcceptTransaction(fx.Ctx, fx.Bob, fx.Net.ID, ref)
				if err != nil || result == nil || result.ID != pending.ID || result.Status != domain.TxCompleted {
					t.Fatalf("same-key retry did not recover the original result: %v %+v", err, result)
				}
			}
			retry()
			if got := readCommitFaultPosting(t, st.DB(), fx, pending.ID, ref.IdempotencyKey); got != committed {
				t.Fatalf("retry did not yield exactly one complete posting: got %+v, want %+v", got, committed)
			}
			final := tu.Snapshot(t, st.DB())
			if afterCommit {
				if changed := state.Changed(final); len(changed) != 0 {
					t.Fatalf("retry after a committed error changed state: %v", changed)
				}
			}
			retry()
			if changed := final.Changed(tu.Snapshot(t, st.DB())); len(changed) != 0 {
				t.Fatalf("replay changed the committed unit: %v", changed)
			}
			reopen(final)
			retry()
			if changed := final.Changed(tu.Snapshot(t, st.DB())); len(changed) != 0 {
				t.Fatalf("replay after a second reopen changed state: %v", changed)
			}
			verify()
		})
	}
}

func checkCommitFaultConnection(t *testing.T, db *sql.DB) {
	t.Helper()
	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("fault connector is not using WAL: %q %v", journal, err)
	}
	for _, p := range []struct {
		name string
		want int
	}{{"foreign_keys", 1}, {"trusted_schema", 0}, {"secure_delete", 1}, {"synchronous", 1}, {"query_only", 0}} {
		var got int
		if err := db.QueryRow("PRAGMA " + p.name).Scan(&got); err != nil || got != p.want {
			t.Fatalf("connection pragma %s: got %d, want %d: %v", p.name, got, p.want, err)
		}
	}
}

type commitFaultPosting struct {
	entries, tradeEntries, audits, statusEvents, notifications, idempotency int
	alice, bob, entrySum, ledgerSeq, tradeSeq                               int64
	status, result                                                          string
}

func readCommitFaultPosting(t *testing.T, db *sql.DB, fx *tu.Fixture, trade, key string) commitFaultPosting {
	t.Helper()
	var s commitFaultPosting
	for _, q := range []struct {
		dst  any
		sql  string
		args []any
	}{
		{&s.entries, `SELECT COUNT(*) FROM entries WHERE network_id = ?`, []any{fx.Net.ID}},
		{&s.tradeEntries, `SELECT COUNT(*) FROM entries WHERE tx_id = ?`, []any{trade}},
		{&s.entrySum, `SELECT COALESCE(SUM(amount),0) FROM entries WHERE tx_id = ?`, []any{trade}},
		{&s.alice, `SELECT balance FROM accounts WHERE member_id = ?`, []any{fx.AliceM.ID}},
		{&s.bob, `SELECT balance FROM accounts WHERE member_id = ?`, []any{fx.BobM.ID}},
		{&s.ledgerSeq, `SELECT ledger_seq FROM networks WHERE id = ?`, []any{fx.Net.ID}},
		{&s.tradeSeq, `SELECT COALESCE(ledger_seq,0) FROM transactions WHERE id = ?`, []any{trade}},
		{&s.status, `SELECT status FROM transactions WHERE id = ?`, []any{trade}},
		{&s.audits, `SELECT COUNT(*) FROM audit_events WHERE network_id = ?`, []any{fx.Net.ID}},
		{&s.statusEvents, `SELECT COUNT(*) FROM transaction_status_events WHERE tx_id = ?`, []any{trade}},
		{&s.notifications, `SELECT COUNT(*) FROM notifications WHERE network_id = ?`, []any{fx.Net.ID}},
		{&s.idempotency, `SELECT COUNT(*) FROM idempotency_keys WHERE network_id = ? AND actor_id = ? AND operation = 'tx.accept' AND key = ?`, []any{fx.Net.ID, fx.Bob.PersonID, key}},
		{&s.result, `SELECT COALESCE(MAX(result_ref),'') FROM idempotency_keys WHERE network_id = ? AND actor_id = ? AND operation = 'tx.accept' AND key = ?`, []any{fx.Net.ID, fx.Bob.PersonID, key}},
	} {
		if err := db.QueryRow(q.sql, q.args...).Scan(q.dst); err != nil {
			t.Fatalf("%s: %v", q.sql, err)
		}
	}
	return s
}
