package service_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestSetupSecretStateLifecycle(t *testing.T) {
	e := tu.NewEnv(t)
	p := e.Person("founder@example.org")
	used, err := e.Svc.SetupSecretUsed(e.Ctx)
	if err != nil || used {
		t.Fatalf("fresh: used=%v err=%v", used, err)
	}
	if err := e.Svc.MarkSetupSecretUsed(e.Ctx, domain.Actor{}); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("anonymous mark: %v", err)
	}
	if err := e.Svc.MarkSetupSecretUsed(e.Ctx, p); err != nil {
		t.Fatal(err)
	}
	if used, _ := e.Svc.SetupSecretUsed(e.Ctx); !used {
		t.Fatal("not used after mark")
	}
	if err := e.Svc.MarkSetupSecretUsed(e.Ctx, p); domain.CodeOf(err) != domain.CodeConflict {
		t.Errorf("second mark: %v", err)
	}
	// Only the operator (system actor) may reset.
	if _, err := e.Svc.ResetSetupSecret(e.Ctx, p); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("member reset: %v", err)
	}
	was, err := e.Svc.ResetSetupSecret(e.Ctx, domain.SystemActor("cli-setup-reset"))
	if err != nil || !was {
		t.Fatalf("reset: was=%v err=%v", was, err)
	}
	if used, _ := e.Svc.SetupSecretUsed(e.Ctx); used {
		t.Fatal("still used after reset")
	}
	if was, err := e.Svc.ResetSetupSecret(e.Ctx, domain.SystemActor("x")); err != nil || was {
		t.Errorf("reset of an unused secret: was=%v err=%v", was, err)
	}
	if err := e.Svc.MarkSetupSecretUsed(e.Ctx, p); err != nil {
		t.Errorf("mark after reset: %v", err)
	}
	// Both transitions are audited at platform scope.
	var nUsed, nReset int
	err = e.Store.View(e.Ctx, func(tx store.Tx) error {
		evs, err := tx.ListAudit("", store.AuditFilter{Limit: 500})
		for _, ev := range evs {
			switch ev.Action {
			case domain.ActSetupSecretUsed:
				nUsed++
			case domain.ActSetupSecretReset:
				nReset++
				if ev.ActorType != domain.ActorSystem {
					t.Errorf("reset actor type = %s", ev.ActorType)
				}
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if nUsed != 2 || nReset != 1 {
		t.Errorf("audit events used=%d reset=%d", nUsed, nReset)
	}
}

func TestSetupSecretMarkRace(t *testing.T) {
	e := tu.NewEnv(t)
	p := e.Person("founder@example.org")
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e.Svc.MarkSetupSecretUsed(e.Ctx, p) == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("%d concurrent marks succeeded, want 1", ok.Load())
	}
}

// SA2-16: creating the network and spending the setup secret are one
// transaction; a second creation with the secret is refused and leaves nothing.
func TestCreateNetworkConsumesSetupSecretAtomically(t *testing.T) {
	e := tu.NewEnv(t)
	p := e.Person("founder@example.org")
	in := service.CreateNetworkInput{Name: "One", Slug: "one-net", CreatorDisplayName: "F", ConsumeSetupSecret: true}
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, p, in); err != nil {
		t.Fatal(err)
	}
	if used, _ := e.Svc.SetupSecretUsed(e.Ctx); !used {
		t.Fatal("secret not spent by CreateNetwork")
	}
	in.Slug, in.Name = "two-net", "Two"
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, p, in); domain.CodeOf(err) != domain.CodeConflict {
		t.Fatalf("second creation: %v", err)
	}
	if n := countRows(t, e, "networks"); n != 1 {
		t.Errorf("networks = %d, want 1", n)
	}
	// a failed creation (slug taken) does not spend the secret
	if _, err := e.Svc.ResetSetupSecret(e.Ctx, domain.SystemActor("t")); err != nil {
		t.Fatal(err)
	}
	in.Slug = "one-net"
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, p, in); err == nil {
		t.Fatal("duplicate slug accepted")
	}
	if used, _ := e.Svc.SetupSecretUsed(e.Ctx); used {
		t.Error("failed creation spent the secret")
	}
	// without the flag the secret state is untouched
	in.Slug, in.ConsumeSetupSecret = "three-net", false
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, p, in); err != nil {
		t.Fatal(err)
	}
	if used, _ := e.Svc.SetupSecretUsed(e.Ctx); used {
		t.Error("flag-less creation spent the secret")
	}
}
