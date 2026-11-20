package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// fullStageStore observes the staged posting immediately before its final
// idempotency write. It injects no error: SQLite's pager must return SQLITE_FULL.
type fullStageStore struct {
	store.Store
	t       *testing.T
	before  ledgerState
	audit   int64
	reached bool
}

func (s *fullStageStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&fullStageTx{Tx: tx, GovRepo: tx.(store.GovRepo), probe: s})
	})
}

type fullStageTx struct {
	store.Tx
	store.GovRepo
	probe *fullStageStore
}

const fullPostingKey = "l9-real-sqlite-full"

func (tx *fullStageTx) PutIdem(r store.IdemRecord) error {
	if r.Key == fullPostingKey {
		p := tx.probe
		t := p.t
		t.Helper()
		id := strings.TrimPrefix(r.ResultRef, "t:") // Immediate adjustments use a tagged result reference.
		posted := tu.Must(tx.GetTransaction(r.NetworkID, id))
		entries := tu.Must(tx.ListEntries(r.NetworkID, id))
		payer := tu.Must(tx.GetAccount(r.NetworkID, posted.PayerAccountID))
		payee := tu.Must(tx.GetAccount(r.NetworkID, posted.PayeeAccountID))
		seq, _, err := tx.LastAudit(r.NetworkID)
		if err != nil {
			t.Fatal(err)
		}
		if posted.Status != domain.TxCompleted || len(entries) != 2 || entries[0].Amount+entries[1].Amount != 0 ||
			payer.Balance != p.before.Balances[payer.ID]-posted.Amount || payee.Balance != p.before.Balances[payee.ID]+posted.Amount || seq <= p.audit {
			t.Fatalf("storage failure was not reached after a complete staged posting: tx=%+v entries=%v payer=%d payee=%d audit=%d", posted, entries, payer.Balance, payee.Balance, seq)
		}
		p.reached = true
	}
	return tx.Tx.PutIdem(r)
}

// L9 / LED-4 / LED-6: a real SQLite pager allocation failure, after the
// ledger, balance cache, status/history, notification and audit writes, rolls
// the complete financial unit back. This is SQLITE_FULL from max_page_count,
// not an injected Go error and not an OS write/fsync (SQLITE_IOERR) test.
func TestSQLiteFullDuringPostingRollsBackAndRecovers(t *testing.T) {
	for _, operation := range []string{"accept", "reversal", "adjustment"} {
		t.Run(operation, func(t *testing.T) {
			w := newWorld(t, nil)
			original := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1500)
			pending := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 300))
			call := func() (*domain.Transaction, error) {
				switch operation {
				case "accept":
					return w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: pending.ID, IdempotencyKey: fullPostingKey})
				case "reversal":
					return w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{
						TxID: original.ID, Amount: 300, Reason: "SQLite storage qualification", IdempotencyKey: fullPostingKey})
				default:
					return w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{
						MemberID: w.aM.ID, Amount: 300, Direction: service.AdjustCredit, Reason: "SQLite storage qualification", IdempotencyKey: fullPostingKey})
				}
			}
			db := w.Store.DB()
			// max_page_count is connection-local; pin the pool so the posting
			// uses the connection on which the real engine limit was set.
			db.SetMaxOpenConns(1)
			db.SetMaxIdleConns(1)
			var pages, pageSize, originalLimit int64
			tu.Must(0, db.QueryRow("PRAGMA max_page_count").Scan(&originalLimit))
			tu.Must(0, db.QueryRow("PRAGMA page_size").Scan(&pageSize))
			tu.Must(db.Exec(`CREATE TABLE l9_full_payload (payload BLOB)`))
			tu.Must(0, db.QueryRow("PRAGMA page_count").Scan(&pages))
			// Give ordinary posting writes ample room, then force overflow
			// allocation in the final idempotency INSERT's AFTER trigger.
			limit := pages + 64
			tu.Must(db.Exec(fmt.Sprintf(`CREATE TRIGGER l9_full_after_idem AFTER INSERT ON idempotency_keys
				WHEN NEW.key = '%s' BEGIN
				  INSERT INTO l9_full_payload VALUES (zeroblob(%d));
				END`, fullPostingKey, (limit+16)*pageSize)))
			var applied int64

			before := tu.Snapshot(t, db)
			ledgerBefore := snapLedger(t, w.Env, w.net.ID)
			probe := &fullStageStore{Store: w.Store, t: t, before: ledgerBefore}
			tu.Must(0, db.QueryRow("SELECT MAX(seq) FROM audit_events WHERE network_id = ?", w.net.ID).Scan(&probe.audit))
			w.Svc = tu.Must(service.New(probe, service.Config{Clock: w.Clock, Argon: qa010Argon}))
			observer := &countObs{}
			w.Svc.SetObserver(observer)
			sinkCalls := 0
			w.Store.SetAuditSink(func(*domain.AuditEvent) { sinkCalls++ })
			for attempt := 0; attempt < 2; attempt++ {
				// SQLITE_FULL can auto-roll back. If explicit rollback then
				// fails, the store discards that connection conservatively.
				// Re-arm this connection-local test limit for every attempt.
				tu.Must(0, db.QueryRow(fmt.Sprintf("PRAGMA max_page_count=%d", limit)).Scan(&applied))
				if applied != limit {
					t.Fatalf("page cap = %d, want %d", applied, limit)
				}
				probe.reached = false
				_, err := call()
				var sqliteErr *sqlite.Error
				if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_FULL {
					t.Fatalf("want real driver SQLITE_FULL (%d), got %T: %v", sqlite3.SQLITE_FULL, err, err)
				}
				if !probe.reached {
					t.Fatal("page cap failed before the staged ledger/cache/status/audit probe")
				}
				if changes := before.Changed(tu.Snapshot(t, db)); len(changes) != 0 {
					t.Fatalf("SQLITE_FULL left partial state: %v", changes)
				}
				if sinkCalls != 0 || observer.posted.Load() != 0 || observer.replayed.Load() != 0 {
					t.Fatalf("uncommitted effects escaped: sink=%d posted=%d replayed=%d", sinkCalls, observer.posted.Load(), observer.replayed.Load())
				}
				requireVerified(t, w.Env, w.admin, w.net.ID)
			}

			// Lift the real limit and remove only the test trigger. The same
			// request and idempotency key must now post once.
			tu.Must(0, db.QueryRow(fmt.Sprintf("PRAGMA max_page_count=%d", originalLimit)).Scan(&applied))
			tu.Must(db.Exec("DROP TRIGGER l9_full_after_idem"))
			result := tu.Must(call())
			committed := tu.Snapshot(t, db)
			replay := tu.Must(call())
			if result.ID != replay.ID || len(committed.Changed(tu.Snapshot(t, db))) != 0 {
				t.Fatal("retry/replay did not return exactly one durable posting")
			}
			after := snapLedger(t, w.Env, w.net.ID)
			if after.Entries != ledgerBefore.Entries+2 || after.MaxSeq != ledgerBefore.MaxSeq+1 || sinkCalls != 1 || observer.posted.Load() != 1 || observer.replayed.Load() != 1 {
				t.Fatalf("retry counts: entries=%d seq=%d sink=%d posted=%d replayed=%d", after.Entries, after.MaxSeq, sinkCalls, observer.posted.Load(), observer.replayed.Load())
			}
			requireVerified(t, w.Env, w.admin, w.net.ID)

			var seq int
			var name, path string
			tu.Must(0, db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path))
			tu.Must(0, w.Store.Close())
			st := tu.Must(store.Open(w.Ctx, path))
			t.Cleanup(func() { _ = st.Close() })
			if changes := committed.Changed(tu.Snapshot(t, st.DB())); len(changes) != 0 {
				t.Fatalf("reopen changed the recovered posting: %v", changes)
			}
			var integrity string
			tu.Must(0, st.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity))
			if integrity != "ok" {
				t.Fatalf("reopened database integrity: %s", integrity)
			}
			w.Store, w.Svc = st, tu.Must(service.New(st, service.Config{Clock: w.Clock, Argon: qa010Argon}))
			requireVerified(t, w.Env, w.admin, w.net.ID)
		})
	}
}
