package main

import (
	"io"
	"log/slog"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// UX-012: the hourly job sends the limit warning once and does not repeat it.
func TestPurgeAndWarnSendsLimitWarning(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "jobs", nil) // default limit 100.00
	a, aM := e.Member(admin, net, "ana@example.org")
	b, bM := e.Member(admin, net, "ben@example.org")
	e.Trade(net.ID, a, b, aM, bM, 8000) // ana uses 80.00 of 100.00

	j := &jobRunner{svc: e.Svc, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	for i := 0; i < 2; i++ { // second run: same ISO week, no second warning
		if err := j.purgeAndWarn(e.Ctx); err != nil {
			t.Fatal(err)
		}
	}
	inbox := tu.Must(e.Svc.ListNotifications(e.Ctx, a, net.ID, service.NotificationQuery{}))
	warnings := 0
	for _, n := range inbox {
		if n.Kind == service.NotifLimitWarning {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("limit warnings = %d, want 1", warnings)
	}
	if got := tu.Must(e.Svc.ListNotifications(e.Ctx, b, net.ID, service.NotificationQuery{})); len(got) != 1 || got[0].Kind != service.NotifTradeProposed {
		t.Errorf("creditor inbox = %+v", got)
	}
}
