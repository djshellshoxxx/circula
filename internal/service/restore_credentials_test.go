package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type restoreAuditFaultStore struct {
	store.Store
	fail *atomic.Bool
}

func (s restoreAuditFaultStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error { return fn(restoreAuditFaultTx{tx, s.fail}) })
}

type restoreAuditFaultTx struct {
	store.Tx
	fail *atomic.Bool
}

func (t restoreAuditFaultTx) InsertAudit(e *domain.AuditEvent) error {
	if t.fail.Load() && e.Action == service.ActRestoreCredentialsInvalidated {
		return errors.New("injected restore audit failure")
	}
	return t.Tx.InsertAudit(e)
}

// A3-F09: bearer invalidation and its audit event form one write unit; all
// credentials remain unchanged on any error. No network role can invoke it.
func TestRestoreCredentialInvalidationAtomicAndOperatorOnly(t *testing.T) {
	var fail atomic.Bool
	e := tu.NewEnvWith(t, func(st store.Store) store.Store { return restoreAuditFaultStore{st, &fail} })
	actor := e.Person("restore-member")
	net, _ := e.Network(actor, "restore", nil)
	tu.Must2(e.Svc.Login(e.Ctx, "restore-member", "correct horse battery"))
	tu.Must2(e.Svc.CreateAPIToken(e.Ctx, actor, service.TokenInput{Name: "fixture", Scopes: []domain.Scope{domain.ScopeRead}}))
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.InsertMFAChallenge(&domain.MFAChallenge{IDHash: "synthetic-challenge", PersonID: actor.PersonID, CreatedAt: e.Clock.Now(), ExpiresAt: e.Clock.Now().Add(time.Hour)})
	}); err != nil {
		t.Fatal(err)
	}
	// Even a future-dated unclosed session must be closed if the clock rolls back.
	e.Clock.Advance(24 * time.Hour)
	bg := tu.Must(e.Svc.OpenBreakGlass(e.Ctx, domain.SystemActor("fixture"), service.OpenBreakGlassInput{Network: net.ID, Operator: "fixture", Reason: "restore test"}))
	e.Clock.Advance(-24 * time.Hour)
	live := func() [5]int {
		t.Helper()
		var x [5]int
		err := e.Store.DB().QueryRow(`SELECT
   (SELECT count(*) FROM sessions WHERE revoked_at IS NULL),
   (SELECT count(*) FROM api_tokens WHERE revoked_at IS NULL),
   (SELECT count(*) FROM mfa_challenges),
   (SELECT count(*) FROM break_glass_sessions WHERE closed_at IS NULL),
   (SELECT count(*) FROM audit_events WHERE action=?)`, service.ActRestoreCredentialsInvalidated).Scan(&x[0], &x[1], &x[2], &x[3], &x[4])
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	before := live()
	if before != [5]int{1, 1, 1, 1, 0} {
		t.Fatalf("bad fixture %v", before)
	}
	if _, err := e.Svc.InvalidateRestoredCredentials(e.Ctx, actor); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member invalidation: %v", err)
	}
	if live() != before {
		t.Fatal("member mutated credentials")
	}
	fail.Store(true)
	if counts, err := e.Svc.InvalidateRestoredCredentials(e.Ctx, domain.SystemActor("restore")); err == nil || counts != (service.RestoredCredentialCounts{}) {
		t.Fatalf("audit failure did not fail closed: %+v %v", counts, err)
	}
	if live() != before {
		t.Fatal("failed invalidation did not roll back all credentials")
	}
	fail.Store(false)
	got := tu.Must(e.Svc.InvalidateRestoredCredentials(e.Ctx, domain.SystemActor("restore")))
	if got != (service.RestoredCredentialCounts{Sessions: 1, Tokens: 1, MFAChallenges: 1, BreakGlassSessions: 1}) {
		t.Fatalf("counts %+v", got)
	}
	if live() != [5]int{0, 0, 0, 0, 1} {
		t.Fatalf("after invalidation %v", live())
	}
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		b, err := tx.GetBreakGlass(bg.ID)
		if err == nil && (b.ClosedAt == nil || !b.ClosedAt.Equal(e.Clock.Now())) {
			t.Errorf("future-dated authority survived: %+v", b)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
