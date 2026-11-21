package service_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-025 e: the checkpoint has a durable byte representation. A normal
// download still reports when it was generated; verification pins ClosedAt.
func TestClosureCheckpointPersistsAfterClockAdvanceAndRestart(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("finished", 30))
	w.Clock.Advance(31 * 24 * time.Hour)
	tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
	c := w.netNow().Closure
	if c.FinalExportVersion != 1 || c.FinalSHA256 == "" {
		t.Fatalf("checkpoint = %+v", c)
	}
	man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatJSON, &service.MemSink{}))
	if !man.GeneratedAt.Equal(c.ClosedAt) || service.ManifestChecksum(man) != c.FinalSHA256 {
		t.Fatal("version 1 checkpoint does not match the ledger export at ClosedAt")
	}
	before := snapLedger(t, w.Env, w.net.ID)
	w.Clock.Advance(2*24*time.Hour + 3*time.Second)
	if chk := tu.Must(w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID)); !chk.Match || chk.Version != 1 {
		t.Fatalf("check after elapsed time = %+v", chk)
	}
	current := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatJSON, &service.MemSink{}))
	if !current.GeneratedAt.Equal(w.Clock.Now()) || service.ManifestChecksum(current) == c.FinalSHA256 {
		t.Fatal("ordinary download lost its current generation timestamp")
	}

	var seq int
	var name, path string
	noErr(t, w.Store.DB().QueryRow(`PRAGMA database_list`).Scan(&seq, &name, &path))
	noErr(t, w.Store.Close())
	st := tu.Must(store.Open(w.Ctx, path))
	t.Cleanup(func() { _ = st.Close() })
	w.Store = st
	w.Svc = tu.Must(service.New(st, service.Config{Clock: w.Clock}))
	w.Clock.Advance(time.Hour)
	chk := tu.Must(w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID))
	if !chk.Match || chk.Stored != c.FinalSHA256 || chk.Version != 1 {
		t.Fatalf("check after reopening database and service = %+v", chk)
	}
	if got := w.netNow().Closure; !reflect.DeepEqual(got, c) {
		t.Fatalf("verification or restart changed the checkpoint: got %+v want %+v", got, c)
	}
	if after := snapLedger(t, w.Env, w.net.ID); !reflect.DeepEqual(after, before) {
		t.Fatalf("verification or restart changed financial history: got %+v want %+v", after, before)
	}
	requireVerified(t, w.Env, w.admin, w.net.ID)
}

// GOV-025 e: versioning removes only time dependence. Changing a covered
// nonfinancial account field in this local repository fixture still differs.
func TestClosureCheckpointDetectsChangedContent(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	tu.Must(w.schedule("finished", 30))
	w.Clock.Advance(31 * 24 * time.Hour)
	tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
	c := w.netNow().Closure
	before := snapLedger(t, w.Env, w.net.ID)
	noErr(t, w.Store.Update(w.Ctx, func(tx store.Tx) error {
		a, err := tx.GetAccountByMember(w.net.ID, w.aM.ID)
		if err != nil {
			return err
		}
		limit := int64(12000)
		return tx.SetLimitOverride(w.net.ID, a.ID, &limit, nil)
	}))
	w.Clock.Advance(time.Hour)
	chk := tu.Must(w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID))
	if chk.Match || chk.Stored != c.FinalSHA256 || chk.Recomputed == "" || chk.Version != 1 || chk.Erased {
		t.Fatalf("changed-content check = %+v", chk)
	}
	if got := w.netNow().Closure; !reflect.DeepEqual(got, c) {
		t.Fatalf("mismatch rewrote the stored checkpoint: got %+v want %+v", got, c)
	}
	if after := snapLedger(t, w.Env, w.net.ID); !reflect.DeepEqual(after, before) {
		t.Fatal("metadata fixture or verification changed balances or financial history")
	}
}

// Legacy data lacks the export generation time. Neither a coincidentally
// equal digest nor an unknown future representation may be called verified.
func TestClosureCheckpointRefusesUnversionedAndUnsupportedHashes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		delay   time.Duration
	}{
		{name: "legacy with later generation time", version: 0, delay: time.Second},
		{name: "legacy with coincidentally equal digest", version: 0},
		{name: "unsupported", version: 99, delay: time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t, nil)
			tu.Must(w.schedule("finished", 30))
			w.Clock.Advance(31 * 24 * time.Hour)
			tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob))
			w.Clock.Advance(tc.delay)
			// Fixture produced exactly as the pre-versioning writer did. Its
			// generation time may accidentally equal ClosedAt or differ from it.
			legacy := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportLedger, service.FormatJSON, &service.MemSink{}))
			hash := service.ManifestChecksum(legacy)
			_, err := w.Store.DB().Exec(`UPDATE network_closures SET final_export_version=?,final_sha256=? WHERE network_id=?`, tc.version, hash, w.net.ID)
			noErr(t, err)
			before := w.netNow().Closure
			w.Clock.Advance(time.Hour)
			chk, err := w.Svc.VerifyClosureExport(w.Ctx, w.admin, w.net.ID)
			wantCode(t, err, domain.ErrInvalidState)
			if chk != nil || !strings.Contains(err.Error(), map[int]string{0: "legacy closure checkpoint", 99: "version 99 is not supported"}[tc.version]) {
				t.Fatalf("unverifiable checkpoint reported as a comparison: check=%+v err=%v", chk, err)
			}
			if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
				t.Fatalf("job replaced an existing checkpoint: %d", n)
			}
			if after := w.netNow().Closure; !reflect.DeepEqual(after, before) || after.FinalSHA256 != hash {
				t.Fatalf("legacy/unsupported checkpoint changed: got %+v want %+v", after, before)
			}
			if _, err := w.Svc.VerifyClosureExport(w.Ctx, w.a, w.net.ID); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("legacy compatibility path weakened administrator restriction: %v", err)
			}
		})
	}
}

type closureStreamBoundaryStore struct {
	store.Store
	afterRow atomic.Pointer[func()]
}

func (s *closureStreamBoundaryStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(closureStreamBoundaryTx{tx, s}) })
}

type closureStreamBoundaryTx struct {
	store.Tx
	boundary *closureStreamBoundaryStore
}

func (t closureStreamBoundaryTx) StreamTable(tbl store.ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	return t.Tx.StreamTable(tbl, netID, memberID, func(cols []string, vals []any) error {
		if err := fn(cols, vals); err != nil {
			return err
		}
		if vals != nil {
			if f := t.boundary.afterRow.Swap(nil); f != nil {
				(*f)()
			}
		}
		return nil
	})
}

// Cancellation during a bounded, ordinary ledger stream records no partial
// checkpoint; the already-closed network remains retryable. A cancelled
// verification returns no comparison and never replaces a stored checkpoint.
func TestClosureCheckpointCancellationIsRetryable(t *testing.T) {
	var boundary *closureStreamBoundaryStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		boundary = &closureStreamBoundaryStore{Store: st}
		return boundary
	})
	admin := e.Person("admin")
	net, _ := e.Network(admin, "cancel-stream", nil)
	tu.Must(e.Svc.ScheduleClosure(e.Ctx, admin, net.ID, service.ClosureInput{Reason: "finished"}))
	e.Clock.Advance(31 * 24 * time.Hour)
	ctx, cancel := context.WithCancel(e.Ctx)
	defer cancel()
	stop := func() { cancel() }
	boundary.afterRow.Store(&stop)
	if n, err := e.Svc.FinaliseClosures(ctx, sysJob); n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled checkpoint = %d, %v", n, err)
	}
	c := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)).Closure
	if c == nil || !c.Done() || c.FinalSHA256 != "" || c.FinalExportVersion != 0 {
		t.Fatalf("cancelled stream saved a checkpoint: %+v", c)
	}
	e.Clock.Advance(time.Hour)
	if n := tu.Must(e.Svc.FinaliseClosures(e.Ctx, sysJob)); n != 1 {
		t.Fatalf("retry completed %d checkpoints", n)
	}
	c = tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)).Closure
	verifyCtx, verifyCancel := context.WithCancel(e.Ctx)
	defer verifyCancel()
	stopVerify := func() { verifyCancel() }
	boundary.afterRow.Store(&stopVerify)
	if chk, err := e.Svc.VerifyClosureExport(verifyCtx, admin, net.ID); chk != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled verification = %+v, %v", chk, err)
	}
	e.Clock.Advance(time.Hour)
	if chk := tu.Must(e.Svc.VerifyClosureExport(e.Ctx, admin, net.ID)); !chk.Match || chk.Stored != c.FinalSHA256 || chk.Version != 1 {
		t.Fatalf("verification after cancellation = %+v", chk)
	}
	if after := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, net.ID)).Closure; !reflect.DeepEqual(after, c) {
		t.Fatal("verification changed the recovered checkpoint")
	}
}
