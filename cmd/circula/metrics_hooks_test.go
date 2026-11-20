package main

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

var _ service.Observer = (*metrics)(nil)
var _ service.PostingObserver = (*metrics)(nil)

// OPS-014 / ARC-022: fixed failure classes expose outcomes without accepting
// a caller's free text as a metric label.
func TestMetricsPostingAttemptsAndBoundedFailureReasons(t *testing.T) {
	m := newMetrics(time.Now())
	m.PostingAttempted(3)
	m.Posted(1)
	m.PostingFailed(1, "credit_limit")
	m.PostingFailed(1, "private-context-canary")
	m.PostingAttempted(-1)
	m.PostingFailed(-1, "credit_limit")
	var b strings.Builder
	m.write(&b, dynamicMetrics{now: time.Now()})
	for _, want := range []string{"circula_ledger_posting_attempts_total 3", "circula_ledger_postings_total 1",
		`circula_ledger_posting_failures_total{reason="credit_limit"} 1`,
		`circula_ledger_posting_failures_total{reason="internal"} 1`,
		`circula_ledger_posting_failures_total{reason="timeout"} 0`} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing metric %q", want)
		}
	}
	if strings.Contains(b.String(), "private-context-canary") {
		t.Fatal("free text escaped into a metric label")
	}
}

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
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "metrics", func(p *domain.Policy) { p.DefaultCreditLimit = 2500 })
	payer, pm := e.Member(admin, net, "payer@example.org")
	payee, qm := e.Member(admin, net, "payee@example.org")
	m := newMetrics(time.Now())
	e.Svc.SetObserver(m)
	if m.postings != 0 || m.replays != 0 {
		t.Fatal("counters not zero at start")
	}
	p := tu.Must(e.Svc.ProposeTransaction(e.Ctx, payer, net.ID, service.ProposeInput{
		CounterpartyID: qm.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: e.Key()}))
	in := service.TxRef{TxID: p.ID, IdempotencyKey: e.Key()}
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, payee, net.ID, in))
	tu.Must(e.Svc.AcceptTransaction(e.Ctx, payee, net.ID, in))
	request := tu.Must(e.Svc.ProposeTransaction(e.Ctx, payee, net.ID, service.ProposeInput{
		CounterpartyID: pm.ID, Direction: service.ProposerIsPaid, Amount: 2400, IdempotencyKey: e.Key()}))
	lower := int64(2000)
	tu.Must(0, e.Svc.SetCreditLimit(e.Ctx, admin, net.ID, service.SetLimitInput{MemberID: pm.ID, Limit: &lower, Reason: "capacity changed before acceptance"}))
	if _, err := e.Svc.AcceptTransaction(e.Ctx, payer, net.ID, service.TxRef{TxID: request.ID, IdempotencyKey: e.Key()}); domain.CodeOf(err) != domain.CodeInsufficientCredit {
		t.Fatalf("expected credit refusal, got %v", err)
	}
	var b strings.Builder
	m.write(&b, dynamicMetrics{now: time.Now()})
	for _, want := range []string{"circula_ledger_posting_attempts_total 2", "circula_ledger_postings_total 1",
		"circula_idempotent_replays_total 1", `circula_ledger_posting_failures_total{reason="credit_limit"} 1`} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("service event missing from metrics: %q", want)
		}
	}
}

// OPS-014: status-class latency separates successful and refused requests,
// while the pre-existing route-only series remains available to dashboards.
func TestMetricsLatencyByStatusClass(t *testing.T) {
	m := newMetrics(time.Now())
	m.observe("/api/", 200, 100*time.Millisecond)
	m.observe("/api/", 201, 200*time.Millisecond)
	m.observe("/api/", 403, 50*time.Millisecond)
	var b strings.Builder
	m.write(&b, dynamicMetrics{now: time.Now()})
	for _, want := range []string{
		`circula_http_request_duration_seconds_count{route="/api/"} 3`,
		`circula_http_request_duration_by_status_seconds_count{route="/api/",class="2xx"} 2`,
		`circula_http_request_duration_by_status_seconds_count{route="/api/",class="4xx"} 1`,
		`circula_http_request_duration_by_status_seconds_sum{route="/api/",class="4xx"} 0.05`,
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing latency metric %q", want)
		}
	}
}
