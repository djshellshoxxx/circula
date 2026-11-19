package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-026: a signal evaluation that fails inside a committed proposal is
// logged by serve as a warning with the network id and the signal kind only;
// the proposal itself succeeds.
func TestSEC026SignalFailureIsLogged(t *testing.T) {
	var logs bytes.Buffer
	logger := newLogger(&Config{LogLevel: slog.LevelDebug, LogFormat: "json"}, &logs)
	var fs *testutil.FaultStore
	e := testutil.NewEnvWith(t, func(s store.Store) store.Store { fs = testutil.NewFaultStore(s); return fs })
	e.Svc.SetSignalFailureHandler(signalFailureLog(logger))
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "town", func(p *domain.Policy) {
		p.ActivitySignalsEnabled = true // Exercise the failure path with explicit activation.
		p.VelocitySignalProposals = 1
	})
	ana, _ := e.Member(admin, net, "ana@example.org")
	_, ben := e.Member(admin, net, "ben@example.org")
	propose := func() error {
		_, err := e.Svc.ProposeTransaction(context.Background(), ana, net.ID, service.ProposeInput{CounterpartyID: ben.ID,
			Direction: service.ProposerPays, Amount: 1, IdempotencyKey: e.Key()})
		return err
	}
	if err := propose(); err != nil {
		t.Fatal(err)
	}
	fs.Arm("InsertActivitySignal", 1)
	if err := propose(); err != nil {
		t.Fatalf("proposal failed with the signal: %v", err)
	}
	out := logs.String()
	for _, want := range []string{`"level":"WARN"`, "activity signal not recorded; the trade was not affected", `"kind":"velocity"`, `"network":"` + net.ID + `"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "ana@example.org") || strings.Contains(out, ben.ID) {
		t.Errorf("log names a member:\n%s", out)
	}
}
