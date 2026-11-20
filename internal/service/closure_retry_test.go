package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type closureExportFaultStore struct {
	store.Store
	fail *atomic.Bool
}

func (s closureExportFaultStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(closureExportFaultTx{tx, s.fail}) })
}

type closureExportFaultTx struct {
	store.Tx
	fail *atomic.Bool
}

func (t closureExportFaultTx) StreamTable(tbl store.ExportTable, net, member string, fn func([]string, []any) error) error {
	if t.fail.Load() {
		return errors.New("injected final export read failure")
	}
	return t.Tx.StreamTable(tbl, net, member, fn)
}

// A1-07 / GOV-025: final export failure must remain retryable after the
// closed status has committed. Erasure cannot run before the checkpoint.
func TestClosureFinalExportRetriesAfterFailure(t *testing.T) {
	var fail atomic.Bool
	e := tu.NewEnvWith(t, func(st store.Store) store.Store { return closureExportFaultStore{st, &fail} })
	admin := e.Person("admin")
	net, _ := e.Network(admin, "retry", nil)
	tu.Must(e.Svc.ScheduleClosure(e.Ctx, admin, net.ID, service.ClosureInput{Reason: "closing"}))
	e.Clock.Advance(31 * 24 * time.Hour)
	fail.Store(true)
	if n, err := e.Svc.FinaliseClosures(e.Ctx, sysJob); err == nil || n != 0 {
		t.Fatalf("first finalise: %d, %v", n, err)
	}
	c := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)).Closure
	if c == nil || !c.Done() || c.FinalSHA256 != "" {
		t.Fatalf("failed export state: %+v", c)
	}
	e.Clock.Advance(time.Duration(c.ReadOnlyDays+1) * 24 * time.Hour)
	if n := tu.Must(e.Svc.EraseClosedNetworks(e.Ctx, sysJob)); n != 0 {
		t.Fatal("network erased before final export was recovered")
	}
	fail.Store(false)
	if n := tu.Must(e.Svc.FinaliseClosures(e.Ctx, sysJob)); n != 1 {
		t.Fatalf("retry finalised %d, want 1", n)
	}
	if n := tu.Must(e.Svc.FinaliseClosures(e.Ctx, sysJob)); n != 0 {
		t.Fatalf("second retry finalised %d, want 0", n)
	}
	var exports, closures int
	if err := e.Store.DB().QueryRow(`SELECT count(*) FROM audit_events WHERE network_id=? AND action=?`, net.ID, service.ActClosureExported).Scan(&exports); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.DB().QueryRow(`SELECT count(*) FROM audit_events WHERE network_id=? AND action='network.closed'`, net.ID).Scan(&closures); err != nil {
		t.Fatal(err)
	}
	if exports != 1 || closures != 1 {
		t.Fatalf("duplicate or missing audit: exports=%d closures=%d", exports, closures)
	}
	var checksum string
	if err := e.Store.DB().QueryRow(`SELECT final_sha256 FROM network_closures WHERE network_id=?`, net.ID).Scan(&checksum); err != nil || len(checksum) != 64 {
		t.Fatalf("recovered checksum %q: %v", checksum, err)
	}
}
