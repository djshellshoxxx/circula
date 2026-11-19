//go:build linux

package service_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	walFaultEnv = "CIRCULA_L9_WAL_WRITE_FIXTURE"
	walFaultKey = "l9-os-wal-write"
	walFaultOK  = "L9 WAL write failed and limit restored"
)

type walFaultFixture struct {
	Path, Network, Proposal string
	Actor                   domain.Actor
	Now                     time.Time
	PageSize                int64
}

type walFaultClock struct{ now time.Time }

func (c walFaultClock) Now() time.Time { return c.now }

// walFaultStore injects no returned error. After the service callback has
// staged every write, it lowers this child process's soft file-size limit.
// The production store's actual COMMIT must encounter the OS write failure.
type walFaultStore struct {
	store.Store
	t          *testing.T
	f          walFaultFixture
	before     ledgerState
	audit      int64
	armed      bool
	restoreErr error
}

func (s *walFaultStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	var previous syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &previous); err != nil {
		return err
	}
	ignored := signal.Ignored(syscall.SIGXFSZ)
	defer func() {
		if s.armed {
			// Restore before the service can publish metrics or the caller
			// performs any recovery work. The hard limit is never changed.
			s.restoreErr = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &previous)
		}
		if !ignored {
			signal.Reset(syscall.SIGXFSZ)
		}
	}()
	return s.Store.Update(ctx, func(tx store.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		posted := tu.Must(tx.GetTransaction(s.f.Network, s.f.Proposal))
		entries := tu.Must(tx.ListEntries(s.f.Network, posted.ID))
		history := tu.Must(tx.ListStatusEvents(s.f.Network, posted.ID))
		payer := tu.Must(tx.GetAccount(s.f.Network, posted.PayerAccountID))
		payee := tu.Must(tx.GetAccount(s.f.Network, posted.PayeeAccountID))
		idem := tu.Must(tx.GetIdem(s.f.Network, s.f.Actor.PersonID, "tx.accept", walFaultKey))
		seq, _, err := tx.LastAudit(s.f.Network)
		if err != nil {
			return err
		}
		if !posted.IsPosted() || posted.LedgerSeq != s.before.MaxSeq+1 ||
			len(entries) != 2 || entries[0].Amount+entries[1].Amount != 0 ||
			payer.Balance != s.before.Balances[payer.ID]-posted.Amount ||
			payee.Balance != s.before.Balances[payee.ID]+posted.Amount ||
			len(history) != 2 || history[1].To != domain.TxCompleted ||
			seq != s.audit+1 || idem == nil || idem.ResultRef != posted.ID {
			s.t.Fatal("WAL limit was not reached after a complete staged posting, history, audit and idempotency record")
		}
		if info := tu.Must(os.Stat(s.f.Path + "-wal")); info.Size() != 0 {
			s.t.Fatalf("posting spilled to WAL before COMMIT: %d bytes", info.Size())
		}
		// A WAL header, one complete non-commit frame, and half the second
		// frame. EFBIG occurs while writing page bytes during COMMIT, after
		// a genuine partial WAL append. No complete commit frame can fit.
		limit := previous
		limit.Cur = uint64(32 + 24 + s.f.PageSize + 24 + s.f.PageSize/2)
		signal.Ignore(syscall.SIGXFSZ) // make the failed write return EFBIG
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
			return fmt.Errorf("arm WAL file-size limit: %w", err)
		}
		s.armed = true
		return nil
	})
}

// L9 / LED-4 / LED-6: a real Linux EFBIG during SQLite's WAL COMMIT returns
// SQLITE_IOERR_WRITE. This is neither a fabricated service error nor a disk
// exhaustion, fsync, checkpoint, read-failure or power-loss qualification.
func TestSQLiteWALWriteFailureRollsBackAndRecovers(t *testing.T) {
	if raw := os.Getenv(walFaultEnv); raw != "" {
		walWriteFailureChild(t, raw)
		return
	}
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1500)
	pending := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 300))
	f := walFaultFixture{Network: w.net.ID, Proposal: pending.ID, Actor: w.b, Now: w.Clock.Now()}
	var seq int
	var name string
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA database_list").Scan(&seq, &name, &f.Path))
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA page_size").Scan(&f.PageSize))
	before := tu.Snapshot(t, w.Store.DB())
	ledgerBefore := snapLedger(t, w.Env, w.net.ID)
	tu.Must(0, w.Store.Close())
	databaseBefore := tu.Must(os.ReadFile(f.Path))

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSQLiteWALWriteFailureRollsBackAndRecovers$", "-test.count=1")
	cmd.Env = append(os.Environ(), walFaultEnv+"="+string(tu.Must(json.Marshal(f))))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), walFaultOK) {
		t.Fatalf("WAL failure child: %v\n%s", err, out)
	}
	if !bytes.Equal(databaseBefore, tu.Must(os.ReadFile(f.Path))) {
		t.Fatal("failed WAL commit changed the database file")
	}
	// Inspect the preserved failed WAL before opening the file or permitting
	// a successful retry to replace its incomplete tail.
	wal := tu.Must(os.ReadFile(f.Path + "-wal"))
	wantSize := int(32 + 24 + f.PageSize + 24 + f.PageSize/2)
	if len(wal) != wantSize {
		t.Fatalf("failed WAL size=%d, want partial second frame at %d", len(wal), wantSize)
	}
	if magic := binary.BigEndian.Uint32(wal[:4]); magic != 0x377f0682 && magic != 0x377f0683 {
		t.Fatalf("unexpected WAL header: %x", magic)
	}
	if int64(binary.BigEndian.Uint32(wal[8:12])) != f.PageSize || binary.BigEndian.Uint32(wal[36:40]) != 0 {
		t.Fatal("failed WAL must contain a complete non-commit frame before its partial tail")
	}

	// No complete commit frame was written, so this specific fault has a
	// definite original-state outcome. Do not accept partially committed rows.
	w.Store = tu.Must(store.Open(w.Ctx, f.Path))
	t.Cleanup(func() { _ = w.Store.Close() })
	w.Svc = tu.Must(service.New(w.Store, service.Config{Clock: w.Clock, Argon: qa010Argon}))
	if changes := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changes) != 0 {
		t.Fatalf("restart after WAL write failure left partial state: %v", changes)
	}
	requireVerified(t, w.Env, w.admin, w.net.ID)
	observer := &countObs{}
	w.Svc.SetObserver(observer)
	sinkCalls := 0
	w.Store.SetAuditSink(func(*domain.AuditEvent) { sinkCalls++ })
	call := func() (*domain.Transaction, error) {
		return w.Svc.AcceptTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: pending.ID, IdempotencyKey: walFaultKey})
	}
	result := tu.Must(call())
	committed := tu.Snapshot(t, w.Store.DB())
	if replay := tu.Must(call()); replay.ID != result.ID || len(committed.Changed(tu.Snapshot(t, w.Store.DB()))) != 0 {
		t.Fatal("retry/replay did not return exactly one posting")
	}
	after := snapLedger(t, w.Env, w.net.ID)
	if after.Entries != ledgerBefore.Entries+2 || after.MaxSeq != ledgerBefore.MaxSeq+1 ||
		sinkCalls != 1 || observer.posted.Load() != 1 || observer.replayed.Load() != 1 {
		t.Fatalf("retry counts: entries=%d seq=%d sink=%d posted=%d replayed=%d", after.Entries, after.MaxSeq, sinkCalls, observer.posted.Load(), observer.replayed.Load())
	}
	tu.Must(0, w.Store.Close())
	w.Store = tu.Must(store.Open(w.Ctx, f.Path))
	w.Svc = tu.Must(service.New(w.Store, service.Config{Clock: w.Clock, Argon: qa010Argon}))
	if changes := committed.Changed(tu.Snapshot(t, w.Store.DB())); len(changes) != 0 {
		t.Fatalf("reopen changed recovered posting: %v", changes)
	}
	var integrity string
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity))
	if integrity != "ok" {
		t.Fatalf("reopened database integrity: %s", integrity)
	}
	requireVerified(t, w.Env, w.admin, w.net.ID)
}

func walWriteFailureChild(t *testing.T, raw string) {
	var f walFaultFixture
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st := tu.Must(store.Open(ctx, f.Path))
	// The limit is process-wide: this child runs only this test. Pin the
	// production connection and finish setup before reducing any OS limit.
	db := st.DB()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	tu.Must(db.Exec("PRAGMA wal_autocheckpoint=0"))
	tu.Must(db.Exec("PRAGMA cache_spill=OFF"))
	var busy, logFrames, checkpointed int
	tu.Must(0, db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed))
	if busy != 0 || logFrames != 0 || checkpointed != 0 {
		t.Fatal("fixture WAL checkpoint did not truncate")
	}
	// Failed rollback cleanup may discard the production connection. Keep a
	// separate query-only connection open so closing that writer is not SQLite's
	// last-close checkpoint: the parent must still inspect the partial WAL.
	// This witness neither holds a transaction nor changes the production pool.
	witnessDB := tu.Must(store.OpenRaw(f.Path))
	defer func() { _ = witnessDB.Close() }()
	witness := tu.Must(witnessDB.Conn(ctx))
	defer func() { _ = witness.Close() }()
	tu.Must(witness.ExecContext(ctx, "PRAGMA query_only=ON"))
	var schemaObjects int
	tu.Must(0, witness.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_schema").Scan(&schemaObjects))
	before := tu.Snapshot(t, db)
	e := &tu.Env{T: t, Ctx: ctx, Store: st}
	probe := &walFaultStore{Store: st, t: t, f: f, before: snapLedger(t, e, f.Network)}
	tu.Must(0, db.QueryRow("SELECT MAX(seq) FROM audit_events WHERE network_id=?", f.Network).Scan(&probe.audit))
	e.Svc = tu.Must(service.New(probe, service.Config{Clock: walFaultClock{f.Now}, Argon: qa010Argon}))
	observer := &countObs{}
	e.Svc.SetObserver(observer)
	sinkCalls := 0
	st.SetAuditSink(func(*domain.AuditEvent) { sinkCalls++ })
	var previous, restored syscall.Rlimit
	tu.Must(0, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &previous))
	_, err := e.Svc.AcceptTransaction(ctx, f.Actor, f.Network, service.TxRef{TxID: f.Proposal, IdempotencyKey: walFaultKey})
	if probe.restoreErr != nil {
		t.Fatalf("restore file-size limit: %v", probe.restoreErr)
	}
	tu.Must(0, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &restored))
	var sqliteErr *sqlite.Error
	if !probe.armed || previous != restored || !strings.HasPrefix(fmt.Sprint(err), "store: commit:") ||
		!errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_IOERR_WRITE {
		t.Fatalf("want actual commit SQLITE_IOERR_WRITE (%d) with restored limit; armed=%v restored=%v err=%T: %v", sqlite3.SQLITE_IOERR_WRITE, probe.armed, previous == restored, err, err)
	}
	if changes := before.Changed(tu.Snapshot(t, db)); len(changes) != 0 {
		t.Fatalf("failed WAL COMMIT left partial live state: %v", changes)
	}
	if sinkCalls != 0 || observer.posted.Load() != 0 || observer.replayed.Load() != 0 {
		t.Fatalf("uncommitted effects escaped: sink=%d posted=%d replayed=%d", sinkCalls, observer.posted.Load(), observer.replayed.Load())
	}
	requireVerified(t, e, domain.SystemActor("l9-wal-recovery"), f.Network)
	fmt.Println(walFaultOK)
	// Intentionally avoid Close/checkpoint and test cleanup: the parent must
	// restart with the exact failed database/WAL, including its partial frame.
	os.Exit(0)
}
