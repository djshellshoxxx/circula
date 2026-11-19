//go:build linux

package service_test

import (
	"bytes"
	"context"
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

const checkpointFaultEnv = "CIRCULA_L9_CHECKPOINT_FIXTURE"

// L9 / LED-4 / LED-6: failure while checkpointing an already committed WAL
// must preserve the complete posting, including its idempotency result.
// This tests an actual OS write error, not a returned store-interface fault.
func TestSQLiteCheckpointWriteFailurePreservesCommittedTrade(t *testing.T) {
	if raw := os.Getenv(checkpointFaultEnv); raw != "" {
		checkpointWriteFailureChild(t, raw)
		return
	}
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1500)
	pending := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 300))
	f := walFaultFixture{Network: w.net.ID, Proposal: pending.ID, Actor: w.b, Now: w.Clock.Now()}
	var seq int
	var name string
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA database_list").Scan(&seq, &name, &f.Path))
	before := snapLedger(t, w.Env, w.net.ID)
	tu.Must(0, w.Store.Close())
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSQLiteCheckpointWriteFailurePreservesCommittedTrade$", "-test.count=1")
	cmd.Env = append(os.Environ(), checkpointFaultEnv+"="+string(tu.Must(json.Marshal(f))))
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "L9 checkpoint write failed and limit restored") {
		t.Fatalf("checkpoint failure child: %v\n%s", err, out)
	}
	var committed tu.State
	tu.Must(0, json.Unmarshal(tu.Must(os.ReadFile(f.Path+".committed.json")), &committed))
	// The failed checkpoint must leave the committed WAL available for recovery.
	if !bytes.Equal(tu.Must(os.ReadFile(f.Path+".committed.wal")), tu.Must(os.ReadFile(f.Path+"-wal"))) {
		t.Fatal("failed checkpoint changed the committed WAL")
	}
	w.Store = tu.Must(store.Open(w.Ctx, f.Path))
	t.Cleanup(func() { _ = w.Store.Close() })
	w.Store.DB().SetMaxOpenConns(1)
	w.Store.DB().SetMaxIdleConns(1)
	w.Svc = tu.Must(service.New(w.Store, service.Config{Clock: w.Clock, Argon: qa010Argon}))
	if changes := committed.Changed(tu.Snapshot(t, w.Store.DB())); len(changes) != 0 {
		t.Fatalf("restart lost committed state after failed checkpoint: %v", changes)
	}
	requireVerified(t, w.Env, w.admin, f.Network)
	observer := &countObs{}
	w.Svc.SetObserver(observer)
	sinkCalls := 0
	w.Store.SetAuditSink(func(*domain.AuditEvent) { sinkCalls++ })
	replay := tu.Must(w.Svc.AcceptTransaction(w.Ctx, f.Actor, f.Network, service.TxRef{TxID: f.Proposal, IdempotencyKey: walFaultKey}))
	after := snapLedger(t, w.Env, f.Network)
	if replay.ID != f.Proposal || len(committed.Changed(tu.Snapshot(t, w.Store.DB()))) != 0 ||
		after.Entries != before.Entries+2 || after.MaxSeq != before.MaxSeq+1 ||
		sinkCalls != 0 || observer.posted.Load() != 0 || observer.replayed.Load() != 1 {
		t.Fatal("replay after checkpoint failure did not preserve exactly one committed posting")
	}
	var busy, frames, checkpointed int
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &frames, &checkpointed))
	if busy != 0 || frames != 0 || checkpointed != 0 || tu.Must(os.Stat(f.Path+"-wal")).Size() != 0 {
		t.Fatal("checkpoint retry did not finish and truncate the WAL")
	}
	tu.Must(0, w.Store.Close())
	w.Store = tu.Must(store.Open(w.Ctx, f.Path))
	w.Svc = tu.Must(service.New(w.Store, service.Config{Clock: w.Clock, Argon: qa010Argon}))
	if changes := committed.Changed(tu.Snapshot(t, w.Store.DB())); len(changes) != 0 {
		t.Fatalf("reopen after checkpoint retry changed committed state: %v", changes)
	}
	var integrity string
	tu.Must(0, w.Store.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity))
	if integrity != "ok" {
		t.Fatalf("recovered database integrity: %s", integrity)
	}
	requireVerified(t, w.Env, w.admin, f.Network)
}

func checkpointWriteFailureChild(t *testing.T, raw string) {
	var f walFaultFixture
	tu.Must(0, json.Unmarshal([]byte(raw), &f))
	ctx := context.Background()
	st := tu.Must(store.Open(ctx, f.Path))
	db := st.DB()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	tu.Must(db.Exec("PRAGMA wal_autocheckpoint=0"))
	// Force database growth in the posting transaction so the checkpoint
	// must write beyond the pre-posting file size. All data stays in the WAL.
	tu.Must(db.Exec("CREATE TABLE l9_checkpoint_payload (payload BLOB)"))
	tu.Must(db.Exec(fmt.Sprintf(`CREATE TRIGGER l9_checkpoint_growth AFTER INSERT ON idempotency_keys
		WHEN NEW.key = '%s' BEGIN
		INSERT INTO l9_checkpoint_payload VALUES (zeroblob(1048576)); END`, walFaultKey)))
	var busy, frames, checkpointed int
	tu.Must(0, db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &frames, &checkpointed))
	if busy != 0 || frames != 0 || checkpointed != 0 {
		t.Fatal("fixture checkpoint did not truncate")
	}
	baselineSize := tu.Must(os.Stat(f.Path)).Size()
	e := &tu.Env{T: t, Ctx: ctx, Store: st}
	e.Svc = tu.Must(service.New(st, service.Config{Clock: walFaultClock{f.Now}, Argon: qa010Argon}))
	observer := &countObs{}
	e.Svc.SetObserver(observer)
	sinkCalls := 0
	st.SetAuditSink(func(*domain.AuditEvent) { sinkCalls++ })
	posted := tu.Must(e.Svc.AcceptTransaction(ctx, f.Actor, f.Network, service.TxRef{TxID: f.Proposal, IdempotencyKey: walFaultKey}))
	if !posted.IsPosted() || sinkCalls != 1 || observer.posted.Load() != 1 {
		t.Fatal("trade must commit and publish once before the checkpoint fault")
	}
	committed := tu.Snapshot(t, db)
	wal := tu.Must(os.ReadFile(f.Path + "-wal"))
	if len(wal) <= 1048576 || tu.Must(os.Stat(f.Path)).Size() != baselineSize {
		t.Fatal("fixture did not retain database growth in the WAL")
	}
	tu.Must(0, os.WriteFile(f.Path+".committed.json", tu.Must(json.Marshal(committed)), 0600))
	tu.Must(0, os.WriteFile(f.Path+".committed.wal", wal, 0600))
	var previous syscall.Rlimit
	tu.Must(0, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &previous))
	limit := previous
	limit.Cur = uint64(baselineSize)
	ignored := signal.Ignored(syscall.SIGXFSZ)
	signal.Ignore(syscall.SIGXFSZ)
	// Keep restoration local to the faulting call, including on panic/failure.
	var fault, restoreErr error
	func() {
		defer func() {
			restoreErr = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &previous)
			if !ignored {
				signal.Reset(syscall.SIGXFSZ)
			}
		}()
		tu.Must(0, syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit))
		fault = db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &frames, &checkpointed)
	}()
	var restored syscall.Rlimit
	tu.Must(0, syscall.Getrlimit(syscall.RLIMIT_FSIZE, &restored))
	var sqliteErr *sqlite.Error
	if restoreErr != nil || restored != previous || !errors.As(fault, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_IOERR_WRITE {
		t.Fatalf("want checkpoint SQLITE_IOERR_WRITE and restored limit: restore=%v fault=%T: %v", restoreErr, fault, fault)
	}
	if changes := committed.Changed(tu.Snapshot(t, db)); len(changes) != 0 {
		t.Fatalf("failed checkpoint changed live committed state: %v", changes)
	}
	if !bytes.Equal(wal, tu.Must(os.ReadFile(f.Path+"-wal"))) || sinkCalls != 1 || observer.posted.Load() != 1 || observer.replayed.Load() != 0 {
		t.Fatal("checkpoint failure changed WAL or republished posting effects")
	}
	requireVerified(t, e, domain.SystemActor("l9-checkpoint-recovery"), f.Network)
	fmt.Println("L9 checkpoint write failed and limit restored")
	// Preserve the exact failed checkpoint files for restart in the parent.
	os.Exit(0)
}
