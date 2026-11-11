package service_test

import (
	"testing"
	"time"

	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestFreezeStatusAndOverview(t *testing.T) {
	w := newWorld(t, nil)
	if fs := tu.Must(w.Svc.NetworkFreezeStatus(w.Ctx, w.a, w.net.ID)); fs.ReadOnly {
		t.Fatal("fresh network is read-only")
	}
	w.Svc.FreezeNetwork(w.net.ID, "why")
	since := w.Clock.Now()
	w.Clock.Advance(time.Hour)
	w.Svc.FreezeNetwork(w.net.ID, "why again") // keeps the first start time
	m := tu.Must(w.Svc.NetworkFreezeStatus(w.Ctx, w.a, w.net.ID))
	if !m.ReadOnly || m.Reason != "" || m.Since.Before(since.Add(-time.Second)) || m.Since.After(since.Add(time.Second)) {
		t.Fatalf("member: %+v", m)
	}
	a := tu.Must(w.Svc.NetworkFreezeStatus(w.Ctx, w.admin, w.net.ID))
	if a.Reason != "why again" {
		t.Fatalf("admin: %+v", a)
	}
	ov := tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.Frozen == nil || ov.Frozen.Reason != "why again" {
		t.Fatalf("overview: %+v", ov.Frozen)
	}
	w.Svc.UnfreezeNetwork(w.net.ID)
	if ov := tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID)); ov.Frozen != nil {
		t.Fatal("still frozen")
	}
}
