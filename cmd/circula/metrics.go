package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// metrics holds the in-process counters and gauges exposed at /metrics in the
// Prometheus text exposition format (OPS-013, OPS-014). Label values are
// fixed vocabularies (route templates, status classes, job and result names);
// they never carry member, network, balance or personal data.
type metrics struct {
	mu sync.Mutex

	started time.Time

	requests     map[[2]string]uint64 // route, status class
	durSum       map[string]float64   // route, seconds
	durCount     map[string]uint64    // route
	authFailures uint64               // 401 responses
	rateLimited  uint64               // 429 responses
	jobRuns      map[[2]string]uint64 // job, result
	migrations   map[string]uint64    // result
	postings     uint64               // ledger postings committed (service.Observer)
	replays      uint64               // idempotent retries answered from a stored record

	ledgerOK        int     // 1 pass, 0 fail, -1 not yet run
	ledgerFailing   int     // networks failing at the last run
	ledgerCheckedAt float64 // unix seconds of the last run
	backupOKAt      float64 // unix seconds of the last successful backup
}

func newMetrics(now time.Time) *metrics {
	return &metrics{
		started:    now,
		requests:   map[[2]string]uint64{},
		durSum:     map[string]float64{},
		durCount:   map[string]uint64{},
		jobRuns:    map[[2]string]uint64{},
		migrations: map[string]uint64{},
		ledgerOK:   -1,
	}
}

// observe records one finished request.
func (m *metrics) observe(route string, status int, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[[2]string{route, fmt.Sprintf("%dxx", status/100)}]++
	m.durSum[route] += d.Seconds()
	m.durCount[route]++
	switch status {
	case 401:
		m.authFailures++
	case 429:
		m.rateLimited++
	}
}

// Posted and Replayed implement service.Observer (OPS-014).
func (m *metrics) Posted(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.postings += uint64(n) //nolint:gosec // G115: the service passes a positive count
}

func (m *metrics) Replayed(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replays += uint64(n) //nolint:gosec // G115: the service passes a positive count
}

func (m *metrics) jobRan(job, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobRuns[[2]string{job, result}]++
}

func (m *metrics) migrationRan(result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.migrations[result]++
}

// ledgerResult records the outcome of a ledger verification run.
func (m *metrics) ledgerResult(ok bool, failing int, at time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ledgerOK = 0
	if ok {
		m.ledgerOK = 1
	}
	m.ledgerFailing = failing
	m.ledgerCheckedAt = float64(at.Unix())
}

func (m *metrics) backupSucceeded(at time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backupOKAt = float64(at.Unix())
}

// dynamicMetrics are read at scrape time.
type dynamicMetrics struct {
	version    string
	schemaCur  int
	schemaPend int
	dbBytes    int64
	walBytes   int64
	now        time.Time

	maintenance bool // maintenance mode active
	frozen      int  // networks frozen after a ledger failure
}

// write renders the exposition text.
func (m *metrics) write(w io.Writer, dyn dynamicMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	gauge := func(name, help string, v any) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s gauge\n%s %v\n", name, help, name, name, v)
	}
	counter := func(name, help string) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s counter\n", name, help, name)
	}

	fmt.Fprintf(&b, "# HELP circula_build_info Build information; always 1.\n# TYPE circula_build_info gauge\n")
	fmt.Fprintf(&b, "circula_build_info{version=%q} 1\n", dyn.version)
	gauge("circula_process_start_time_seconds", "Unix time the process started.", m.started.Unix())
	gauge("circula_uptime_seconds", "Seconds since the process started.", int64(dyn.now.Sub(m.started).Seconds()))
	gauge("circula_schema_version", "Schema version applied to the database.", dyn.schemaCur)
	gauge("circula_schema_pending_migrations", "Migrations this binary knows that the database does not have.", dyn.schemaPend)
	gauge("circula_database_size_bytes", "Size of the database file.", dyn.dbBytes)
	gauge("circula_database_wal_size_bytes", "Size of the write-ahead log file.", dyn.walBytes)

	counter("circula_http_requests_total", "HTTP requests by route template and status class.")
	for _, k := range sortedKeys2(m.requests) {
		fmt.Fprintf(&b, "circula_http_requests_total{route=%q,class=%q} %d\n", k[0], k[1], m.requests[k])
	}
	fmt.Fprintf(&b, "# HELP circula_http_request_duration_seconds Request latency by route template (sum and count, no quantiles).\n# TYPE circula_http_request_duration_seconds summary\n")
	for _, r := range sortedKeys1(m.durCount) {
		fmt.Fprintf(&b, "circula_http_request_duration_seconds_sum{route=%q} %g\n", r, m.durSum[r])
		fmt.Fprintf(&b, "circula_http_request_duration_seconds_count{route=%q} %d\n", r, m.durCount[r])
	}
	counter("circula_http_auth_failures_total", "Responses with status 401 (authentication failures seen at the edge).")
	fmt.Fprintf(&b, "circula_http_auth_failures_total %d\n", m.authFailures)
	counter("circula_http_rate_limited_total", "Responses with status 429 (throttling seen at the edge).")
	fmt.Fprintf(&b, "circula_http_rate_limited_total %d\n", m.rateLimited)

	counter("circula_ledger_postings_total", "Ledger postings committed since start.")
	fmt.Fprintf(&b, "circula_ledger_postings_total %d\n", m.postings)
	counter("circula_idempotent_replays_total", "Retries answered from a stored idempotency record since start.")
	fmt.Fprintf(&b, "circula_idempotent_replays_total %d\n", m.replays)
	gauge("circula_maintenance_mode", "1 while maintenance mode is active.", map[bool]int{true: 1, false: 0}[dyn.maintenance])
	gauge("circula_frozen_networks", "Networks refusing writes after a ledger verification failure.", dyn.frozen)

	counter("circula_job_runs_total", "Scheduled job runs by job and result.")
	for _, k := range sortedKeys2(m.jobRuns) {
		fmt.Fprintf(&b, "circula_job_runs_total{job=%q,result=%q} %d\n", k[0], k[1], m.jobRuns[k])
	}
	counter("circula_migrations_total", "Migration runs by result.")
	for _, r := range sortedKeys1(m.migrations) {
		fmt.Fprintf(&b, "circula_migrations_total{result=%q} %d\n", r, m.migrations[r])
	}

	gauge("circula_ledger_verification_ok", "Result of the last ledger verification: 1 pass, 0 fail, -1 not run.", m.ledgerOK)
	gauge("circula_ledger_verification_failing_networks", "Networks that failed the last ledger verification.", m.ledgerFailing)
	gauge("circula_ledger_verification_timestamp_seconds", "Unix time of the last ledger verification.", m.ledgerCheckedAt)
	gauge("circula_backup_last_success_timestamp_seconds", "Unix time of the last successful backup; 0 if none.", m.backupOKAt)

	_, _ = io.WriteString(w, b.String())
}

func sortedKeys1[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeys2[V any](m map[[2]string]V) [][2]string {
	out := make([][2]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}
