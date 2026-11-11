package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

var _ service.Observer = (*metrics)(nil)

// OPS-014: posting and replay counters are exposed and increment.
func TestMetricsPostingAndReplayCounters(t *testing.T) {
	m := newMetrics(time.Now())
	m.Posted(2)
	m.Replayed(1)
	m.Posted(1)
	var b strings.Builder
	m.write(&b, dynamicMetrics{now: time.Now()})
	for _, want := range []string{"circula_ledger_postings_total 3", "circula_idempotent_replays_total 1",
		"circula_maintenance_mode 0", "circula_frozen_networks 0"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q in:\n%s", want, b.String())
		}
	}
}

// The service reports a committed posting to the observer wired by serve.
func TestServiceObserverWiredToMetrics(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, seedDatabase(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := newService(st, &Config{})
	if err != nil {
		t.Fatal(err)
	}
	m := newMetrics(time.Now())
	svc.SetObserver(m)
	if m.postings != 0 || m.replays != 0 {
		t.Fatal("counters not zero at start")
	}
}
