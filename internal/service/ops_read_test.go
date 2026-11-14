package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// ARC-002: the operator CLI lists networks through the service, not by
// querying the database; only the system actor may.
func TestOperatorNetworks(t *testing.T) {
	e := tu.NewEnv(t)
	founder := e.Person("founder")
	a, _ := e.Network(founder, "alpha", nil)
	b, _ := e.Network(founder, "beta", nil)
	sys := domain.SystemActor("test")

	all, err := e.Svc.OperatorNetworks(e.Ctx, sys, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []service.NetworkRef{{ID: a.ID, Slug: "alpha"}, {ID: b.ID, Slug: "beta"}}
	if a.ID > b.ID {
		want[0], want[1] = want[1], want[0]
	}
	if len(all) != 2 || all[0] != want[0] || all[1] != want[1] {
		t.Fatalf("OperatorNetworks = %+v, want %+v (id order)", all, want)
	}
	for _, f := range []string{"beta", b.ID} {
		got, err := e.Svc.OperatorNetworks(e.Ctx, sys, f)
		if err != nil || len(got) != 1 || got[0].ID != b.ID {
			t.Errorf("filter %q = %+v, %v; want only beta", f, got, err)
		}
	}
	if got, err := e.Svc.OperatorNetworks(e.Ctx, sys, "nope"); err != nil || len(got) != 0 {
		t.Errorf("unknown filter = %+v, %v; want none", got, err)
	}
	if _, err := e.Svc.OperatorNetworks(e.Ctx, founder, ""); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("member actor: err = %v, want forbidden", err)
	}
}

// ARC-002 / OPS-011: the storage settings check runs through the service.
func TestCheckStorage(t *testing.T) {
	e := tu.NewEnv(t)
	if err := e.Svc.CheckStorage(e.Ctx); err != nil {
		t.Fatalf("CheckStorage on a WAL store: %v", err)
	}
	// Foreign keys off on every pooled connection: the check fails.
	db := e.Store.DB()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(e.Ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatal(err)
	}
	if err := e.Svc.CheckStorage(e.Ctx); err == nil {
		t.Error("CheckStorage passed with foreign keys off")
	}
	// A store that cannot be inspected is refused, not waved through.
	opaque := tu.NewEnvWith(t, func(s store.Store) store.Store { return tu.NewFaultStore(s) })
	if err := opaque.Svc.CheckStorage(opaque.Ctx); err == nil {
		t.Error("CheckStorage passed for a store whose settings cannot be read")
	}
}
