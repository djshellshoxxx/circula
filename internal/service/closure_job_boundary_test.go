package service_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// closureJobBoundaryStore exercises real lifecycle operations between the
// job's candidate snapshot and write unit. Hooks run outside SQLite units so
// the ordinary cancellation and scheduling transactions commit normally.
type closureJobBoundaryStore struct {
	store.Store
	afterView    atomic.Pointer[func()]
	beforeUpdate atomic.Pointer[func()]
}

func (s *closureJobBoundaryStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if err := s.Store.View(ctx, fn); err != nil {
		return err
	}
	if f := s.afterView.Swap(nil); f != nil {
		(*f)()
	}
	return nil
}

func (s *closureJobBoundaryStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	if f := s.beforeUpdate.Swap(nil); f != nil {
		(*f)()
	}
	return s.Store.Update(ctx, fn)
}

func closureJobBoundaryWorld(t *testing.T) (*world, *closureJobBoundaryStore) {
	t.Helper()
	var boundary *closureJobBoundaryStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store {
		boundary = &closureJobBoundaryStore{Store: st}
		return boundary
	})
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "closure-job", nil)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	return w, boundary
}

func assertNoClosureEffects(t *testing.T, w *world, proposalID string) {
	t.Helper()
	if got := w.txStatus(proposalID).Status; got != domain.TxProposed {
		t.Errorf("proposal status = %s, want proposed", got)
	}
	if c := w.netNow().Closure; c != nil && (c.Done() || c.FinalSHA256 != "") {
		t.Errorf("closure completed or checkpointed: %+v", c)
	}
	for _, action := range []string{domain.ActTxCancelled, domain.ActNetworkClosed, service.ActClosureExported} {
		if hasKind(auditActions(w), action) {
			t.Errorf("ineligible candidate emitted audit %s", action)
		}
	}
	for _, actor := range []domain.Actor{w.admin, w.a, w.b} {
		inbox := w.inbox(actor)
		for _, kind := range []string{service.NotifTradeCancelled, service.NotifNetworkClosed, service.NotifClosureBlocked} {
			if hasKind(inbox, kind) {
				t.Errorf("ineligible candidate emitted notification %s", kind)
			}
		}
	}
}

// GOV-025: an announcement cancelled before its deadline cannot be acted on
// by an old batch, even if a replacement is also due when that batch resumes.
func TestClosureFinaliserRechecksAnnouncement(t *testing.T) {
	for _, change := range []string{"cancelled", "replacement still in notice", "replacement now due"} {
		t.Run(change, func(t *testing.T) {
			w, boundary := closureJobBoundaryWorld(t)
			proposal := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
			tu.Must(w.schedule("original announcement", 30))
			w.Clock.Advance(29 * 24 * time.Hour)
			var replacement *domain.Closure
			changeAnnouncement := func() {
				noErr(t, w.Svc.CancelClosure(w.Ctx, w.admin, w.net.ID, "reconsidered before the deadline"))
				if change != "cancelled" {
					replacement = tu.Must(w.schedule("replacement announcement", 30))
				}
				if change == "replacement now due" {
					w.Clock.Advance(31 * 24 * time.Hour)
				} else {
					w.Clock.Advance(2 * 24 * time.Hour)
				}
			}
			boundary.afterView.Store(&changeAnnouncement)
			if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
				t.Fatalf("old candidate completed %d closures", n)
			}
			if boundary.afterView.Load() != nil {
				t.Fatal("job did not load its candidate snapshot")
			}
			assertNoClosureEffects(t, w, proposal.ID)
			if replacement == nil {
				if n := w.netNow(); n.Status != domain.NetworkActive || n.Closure != nil {
					t.Fatalf("cancelled announcement survived: %+v", n)
				}
				return
			}
			c := w.netNow().Closure
			if c == nil || c.Reason != replacement.Reason || !c.ClosingAt.Equal(replacement.ClosingAt) {
				t.Fatalf("replacement announcement changed: %+v", c)
			}
			if w.Clock.Now().Before(c.ClosingAt) {
				w.Clock.Advance(c.ClosingAt.Sub(w.Clock.Now()))
			}
			if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
				t.Fatalf("fresh due candidate completed %d closures", n)
			}
		})
	}
}

// GOV-025: use the time at the write unit, including an ordinary wall-clock
// correction, rather than treating the earlier batch check as authorization.
func TestClosureFinaliserChecksDeadlineAtWrite(t *testing.T) {
	w, boundary := closureJobBoundaryWorld(t)
	proposal := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	tu.Must(w.schedule("closing", 30))
	w.Clock.Advance(30*24*time.Hour + time.Second)
	correctClock := func() { w.Clock.Advance(-2 * time.Second) }
	boundary.beforeUpdate.Store(&correctClock)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("not-yet-due announcement completed %d closures", n)
	}
	assertNoClosureEffects(t, w, proposal.ID)
	w.Clock.Advance(2 * time.Second)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("due announcement completed %d closures", n)
	}
}

// CMD-07 / OPS-033: a consistency freeze set after candidate selection still
// prevents every closure effect when the write unit starts.
func TestClosureFinaliserRechecksFreeze(t *testing.T) {
	w, boundary := closureJobBoundaryWorld(t)
	proposal := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	tu.Must(w.schedule("closing", 30))
	w.Clock.Advance(31 * 24 * time.Hour)
	freeze := func() { w.Svc.FreezeNetwork(w.net.ID, "verification is pending") }
	boundary.beforeUpdate.Store(&freeze)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 0 {
		t.Fatalf("frozen network completed %d closures", n)
	}
	assertNoClosureEffects(t, w, proposal.ID)
	w.Svc.UnfreezeNetwork(w.net.ID)
	if n := tu.Must(w.Svc.FinaliseClosures(w.Ctx, sysJob)); n != 1 {
		t.Fatalf("unfrozen due announcement completed %d closures", n)
	}
}
