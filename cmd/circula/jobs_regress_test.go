package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// fakeClock is a settable domain.Clock so job tests can move time forward.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// syncBuf is a log sink that tolerates reads while jobs write.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// jobFixture is a migrated database with one open network, two active
// members and a clock the test controls. The job runner is wired to the same
// service and store the server would use.
type jobFixture struct {
	ctx    context.Context
	dir    string
	db     string
	clk    *fakeClock
	st     *store.SQLite
	svc    *service.Service
	cfg    *Config
	runner *jobRunner
	logs   *syncBuf
	payer  domain.Actor
	payee  domain.Actor
	netID  string
	payeeM *domain.Member
}

func newJobFixture(t *testing.T) *jobFixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db := filepath.Join(dir, "circula.db")
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := &fakeClock{now: time.Now().UTC().Truncate(time.Second)}
	svc, err := service.New(st, service.Config{Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	pol := domain.DefaultPolicy()
	pol.JoinMode = domain.JoinOpen
	pol.DefaultCreditLimit = 10000

	founder := registerActor(t, svc, "founder01")
	net, _, err := svc.CreateNetwork(ctx, founder, service.CreateNetworkInput{Name: "Test", Slug: "test-net",
		CreatorDisplayName: "Founder", Policy: &pol})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ActivateNetwork(ctx, founder, net.ID); err != nil {
		t.Fatal(err)
	}
	payer := registerActor(t, svc, "payer001")
	payee := registerActor(t, svc, "payee001")
	if _, err := svc.JoinNetwork(ctx, payer, net.ID, "Payer"); err != nil {
		t.Fatal(err)
	}
	payeeM, err := svc.JoinNetwork(ctx, payee, net.ID, "Payee")
	if err != nil {
		t.Fatal(err)
	}
	logs := &syncBuf{}
	cfg := &Config{Database: db, BackupDir: filepath.Join(dir, "backups"), BackupInterval: 24 * time.Hour,
		BackupRetainDaily: 30, BackupRetainMonthly: 12}
	logger := slog.New(slog.NewTextHandler(logs, nil))
	runner := &jobRunner{cfg: cfg, svc: svc, st: st, logger: logger, metrics: newMetrics(clk.Now())}
	return &jobFixture{ctx: ctx, dir: dir, db: db, clk: clk, st: st, svc: svc, cfg: cfg, runner: runner,
		logs: logs, payer: payer, payee: payee, netID: net.ID, payeeM: payeeM}
}

func registerActor(t *testing.T, svc *service.Service, login string) domain.Actor {
	t.Helper()
	p, err := svc.RegisterPerson(context.Background(), service.RegisterInput{Login: login, Password: "correct horse battery"})
	if err != nil {
		t.Fatal(err)
	}
	return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
}

// propose opens a payer-to-payee proposal at the fixture's current time.
func (f *jobFixture) propose(t *testing.T, key string) string {
	t.Helper()
	tx, err := f.svc.ProposeTransaction(f.ctx, f.payer, f.netID, service.ProposeInput{CounterpartyID: f.payeeM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return tx.ID
}

func (f *jobFixture) txStatus(t *testing.T, id string) string {
	t.Helper()
	var s string
	if err := f.st.DB().QueryRowContext(f.ctx, `SELECT status FROM transactions WHERE id = ?`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *jobFixture) count(t *testing.T, path, query string) int {
	t.Helper()
	raw, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	var n int
	if err := raw.QueryRowContext(f.ctx, query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func jobRuns(m *metrics, job, result string) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobRuns[[2]string{job, result}]
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// The schedule is part of the operator contract (docs/ops): proposals expire
// within minutes, ledger verification and expired-row purges run hourly, and a
// failed backup is retried after 15 minutes rather than a day later.
func TestJobScheduleConstants(t *testing.T) {
	if expiryInterval != 5*time.Minute || verifyInterval != time.Hour || purgeInterval != time.Hour ||
		backupRetryInterval != 15*time.Minute {
		t.Fatalf("schedule changed: expiry %v verify %v purge %v retry %v",
			expiryInterval, verifyInterval, purgeInterval, backupRetryInterval)
	}
}

// Expiry moves elapsed proposals to expired, leaves live ones alone, and a
// second pass finds nothing left to do.
func TestJobExpiryMovesOnlyElapsedProposals(t *testing.T) {
	f := newJobFixture(t)
	old := f.propose(t, "old-1")
	f.clk.Advance(8 * 24 * time.Hour) // policy default is 7 days
	live := f.propose(t, "live-1")
	if err := f.runner.expire(f.ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.txStatus(t, old); got != "expired" {
		t.Errorf("elapsed proposal status %q, want expired", got)
	}
	if got := f.txStatus(t, live); got != "proposed" {
		t.Errorf("live proposal status %q, want proposed", got)
	}
	if !strings.Contains(f.logs.String(), "proposals expired") {
		t.Errorf("no expiry log line: %s", f.logs.String())
	}
	before := f.count(t, f.db, `SELECT COUNT(*) FROM transaction_status_events`)
	if err := f.runner.expire(f.ctx); err != nil {
		t.Fatal(err)
	}
	if after := f.count(t, f.db, `SELECT COUNT(*) FROM transaction_status_events`); after != before {
		t.Errorf("second expiry pass wrote %d status events", after-before)
	}
}

// Purge removes sessions that have been idle past the idle limit and keeps
// the one in use.
func TestJobPurgeRemovesIdleSessions(t *testing.T) {
	f := newJobFixture(t)
	if _, _, err := f.svc.Login(f.ctx, "founder01", "correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if n := f.count(t, f.db, `SELECT COUNT(*) FROM sessions`); n != 1 {
		t.Fatalf("sessions before purge = %d", n)
	}
	f.clk.Advance(30*24*time.Hour + time.Hour)
	token, _, err := f.svc.Login(f.ctx, "founder01", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.runner.purge(f.ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.count(t, f.db, `SELECT COUNT(*) FROM sessions`); n != 1 {
		t.Errorf("sessions after purge = %d, want only the fresh one", n)
	}
	if _, err := f.svc.Authenticate(f.ctx, token); err != nil {
		t.Errorf("fresh session purged: %v", err)
	}
}

// Verification records the ledger outcome for metrics. A ledger fault found by
// the scheduled run is reported as a failure, not as a job error that would
// stop the schedule.
func TestJobVerifyReportsLedgerFault(t *testing.T) {
	f := newJobFixture(t)
	if err := f.runner.verify(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.runner.metrics.mu.Lock()
	ok, failing := f.runner.metrics.ledgerOK, f.runner.metrics.ledgerFailing
	f.runner.metrics.mu.Unlock()
	if ok != 1 || failing != 0 {
		t.Fatalf("clean ledger: ok=%d failing=%d", ok, failing)
	}

	tamper := func(stmts ...string) {
		raw, err := openRaw(f.db)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = raw.Close() }()
		for _, s := range stmts {
			if _, err := raw.Exec(s); err != nil {
				t.Fatalf("%s: %v", s, err)
			}
		}
	}
	tamper(`DROP TRIGGER audit_no_update`, `UPDATE audit_events SET reason = 'forged' WHERE seq = 1`)
	if err := f.runner.verify(f.ctx); err != nil {
		t.Fatalf("verify returned error for a ledger fault: %v", err)
	}
	f.runner.metrics.mu.Lock()
	ok, failing = f.runner.metrics.ledgerOK, f.runner.metrics.ledgerFailing
	f.runner.metrics.mu.Unlock()
	if ok != 0 || failing != 1 {
		t.Fatalf("tampered ledger: ok=%d failing=%d", ok, failing)
	}
	if !strings.Contains(f.logs.String(), "ledger verification failed") {
		t.Errorf("no error log for ledger fault")
	}
}

// every runs the job at once and then on each interval, records each attempt,
// and stops when its context ends.
func TestJobEveryRepeatsUntilCancelled(t *testing.T) {
	f := newJobFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	var mu sync.Mutex
	calls := 0
	done := make(chan struct{})
	go func() {
		f.runner.every(ctx, "probe", 5*time.Millisecond, func(context.Context) error {
			mu.Lock()
			calls++
			mu.Unlock()
			return nil
		})
		close(done)
	}()
	waitFor(t, "three runs", func() bool { mu.Lock(); defer mu.Unlock(); return calls >= 3 })
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("every did not return after cancel")
	}
	if n := jobRuns(f.runner.metrics, "probe", "ok"); n < 3 {
		t.Errorf("ok runs recorded = %d", n)
	}
}

// A failing attempt is counted and logged; an attempt interrupted by shutdown
// is not counted as a failure.
func TestJobAttemptRecordsFailureNotShutdown(t *testing.T) {
	f := newJobFixture(t)
	f.runner.attempt(f.ctx, "broken", func(context.Context) error { return io.ErrUnexpectedEOF })
	if jobRuns(f.runner.metrics, "broken", "failed") != 1 || !strings.Contains(f.logs.String(), "scheduled job failed") {
		t.Fatalf("failure not recorded: %s", f.logs.String())
	}
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	f.runner.attempt(ctx, "stopping", func(context.Context) error { return io.ErrUnexpectedEOF })
	if jobRuns(f.runner.metrics, "stopping", "failed") != 0 {
		t.Error("shutdown interruption counted as a job failure")
	}
}

// backupNow writes a verified backup, records it, and prunes according to the
// retention settings: newest daily copies plus the newest copy of each kept
// month, and nothing else.
func TestJobScheduledBackupAndRetention(t *testing.T) {
	f := newJobFixture(t)
	if err := os.MkdirAll(f.cfg.BackupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := []time.Time{now.AddDate(0, 0, -1), now.AddDate(0, 0, -2), now.AddDate(0, 0, -3), now.AddDate(0, 0, -40), now.AddDate(0, 0, -400)}
	for _, ts := range old {
		name := filepath.Join(f.cfg.BackupDir, backupFileName(ts))
		if err := os.WriteFile(name, []byte("stale"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.cfg.BackupRetainDaily = 2
	f.cfg.BackupRetainMonthly = 1
	if err := f.runner.backupNow(f.ctx); err != nil {
		t.Fatalf("backupNow: %v", err)
	}
	if jobRuns(f.runner.metrics, "backup", "ok") != 1 {
		t.Error("backup job not recorded as ok")
	}
	f.runner.metrics.mu.Lock()
	stamped := f.runner.metrics.backupOKAt
	f.runner.metrics.mu.Unlock()
	if stamped == 0 {
		t.Error("last backup time not recorded")
	}
	left, err := listBackups(f.cfg.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 {
		var names []string
		for _, e := range left {
			names = append(names, filepath.Base(e.path))
		}
		t.Fatalf("%d backups kept, want 2 (newest daily): %v", len(left), names)
	}
	newest := left[0]
	if err := verifyBackupFile(newest.path, mustManifest(t, newest.path)); err != nil {
		t.Errorf("new backup does not match its manifest: %v", err)
	}
	if !strings.Contains(f.logs.String(), "expired backups removed") {
		t.Error("pruning not logged")
	}
	if _, err := os.Stat(filepath.Join(f.cfg.BackupDir, backupFileName(now.AddDate(0, 0, -400)))); err == nil {
		t.Error("400-day-old backup kept")
	}
	// The source database is not modified by taking a backup.
	if n := f.count(t, f.db, `SELECT COUNT(*) FROM audit_events WHERE request_id = 'backup-verify'`); n != 0 {
		t.Errorf("backup verification wrote %d audit rows into the source", n)
	}
}

func mustManifest(t *testing.T, path string) *Manifest {
	t.Helper()
	m, err := readManifest(path + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// untilNextBackup counts down from the newest backup, and is zero when there is
// no backup or the newest is already a full interval old.
func TestJobUntilNextBackup(t *testing.T) {
	f := newJobFixture(t)
	if d := f.runner.untilNextBackup(); d != 0 {
		t.Fatalf("no backups: wait %v, want 0 (back up now)", d)
	}
	if err := os.MkdirAll(f.cfg.BackupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	recent := time.Now().UTC().Add(-time.Hour)
	if err := os.WriteFile(filepath.Join(f.cfg.BackupDir, backupFileName(recent)), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d := f.runner.untilNextBackup(); d < 22*time.Hour || d > 23*time.Hour+time.Minute {
		t.Errorf("one hour old backup: wait %v, want about 23h", d)
	}
	f.cfg.BackupDir = filepath.Join(f.dir, "old-backups")
	if err := os.MkdirAll(f.cfg.BackupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().UTC().Add(-48 * time.Hour)
	if err := os.WriteFile(filepath.Join(f.cfg.BackupDir, backupFileName(stale)), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d := f.runner.untilNextBackup(); d != 0 {
		t.Errorf("overdue backup: wait %v, want 0", d)
	}
}

// The backup loop takes a backup straight away when none exists, then sleeps
// for the interval and returns when shut down.
func TestJobBackupLoopRunsOnceThenStops(t *testing.T) {
	f := newJobFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	done := make(chan struct{})
	go func() { f.runner.backups(ctx); close(done) }()
	waitFor(t, "first scheduled backup", func() bool { return jobRuns(f.runner.metrics, "backup", "ok") == 1 })
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("backup loop did not stop on cancel")
	}
	if n := jobRuns(f.runner.metrics, "backup", "ok"); n != 1 {
		t.Errorf("loop took %d backups in one interval", n)
	}
}

// With backup-interval 0 the scheduled backups are disabled and the loop returns.
func TestJobBackupsDisabledByZeroInterval(t *testing.T) {
	f := newJobFixture(t)
	f.cfg.BackupInterval = 0
	done := make(chan struct{})
	go func() { f.runner.backups(f.ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("disabled backup loop kept running")
	}
	if _, err := os.Stat(f.cfg.BackupDir); err == nil {
		t.Error("backup directory created while backups are disabled")
	}
}

// A backup that cannot be written is logged and counted, and the loop retries
// instead of waiting a whole interval. Shutdown ends the retry wait.
func TestJobBackupFailureIsRetriedAndStoppable(t *testing.T) {
	f := newJobFixture(t)
	blocker := filepath.Join(f.dir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.cfg.BackupDir = filepath.Join(blocker, "backups")
	ctx, cancel := context.WithCancel(f.ctx)
	done := make(chan struct{})
	go func() { f.runner.backups(ctx); close(done) }()
	waitFor(t, "failed backup", func() bool { return jobRuns(f.runner.metrics, "backup", "failed") == 1 })
	if !strings.Contains(f.logs.String(), "scheduled backup failed") {
		t.Errorf("failure not logged: %s", f.logs.String())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("retry wait ignored shutdown")
	}
}

// run starts the four scheduled jobs; the expiry, verification and purge jobs
// each run once at start-up, and everything stops on shutdown.
func TestJobRunStartsAllJobsAndStops(t *testing.T) {
	f := newJobFixture(t)
	f.cfg.BackupInterval = 0
	ctx, cancel := context.WithCancel(f.ctx)
	done := make(chan struct{})
	go func() { f.runner.run(ctx); close(done) }()
	waitFor(t, "start-up jobs", func() bool {
		return jobRuns(f.runner.metrics, "expiry", "ok") >= 1 &&
			jobRuns(f.runner.metrics, "verify", "ok") >= 1 &&
			jobRuns(f.runner.metrics, "purge", "ok") >= 1
	})
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("job runner did not stop on cancel")
	}
}

// Start-up with a recent backup and an hourly verification: the backup loop
// uses the newest backup's age, not the process start time.
func TestBackupCadenceFollowsNewestBackup(t *testing.T) {
	f := newJobFixture(t)
	if err := os.MkdirAll(f.cfg.BackupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC().Add(-23 * time.Hour)
	if err := os.WriteFile(filepath.Join(f.cfg.BackupDir, backupFileName(ts)), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { f.runner.backups(ctx); close(done) }()
	time.Sleep(50 * time.Millisecond)
	if n := jobRuns(f.runner.metrics, "backup", "ok") + jobRuns(f.runner.metrics, "backup", "failed"); n != 0 {
		t.Fatalf("backup started %d time(s) before the interval elapsed", n)
	}
	cancel()
	<-done
}

// serve starts, reports ready on /readyz, and returns nil when its context is
// cancelled; the database is usable afterwards.
func TestServeStartsAndStopsCleanly(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	cfg.BackupInterval = 0
	cfg.PublicBaseURL = "https://x.test"
	// Reserve a loopback port, then let serve bind it.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Listen = probe.Addr().String()
	_ = probe.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- runServe(ctx, cfg, discardLogger()) }()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(60 * time.Second)
	for ready := false; !ready; {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("serve never became ready")
		}
		select {
		case err := <-errc:
			t.Fatalf("serve exited before becoming ready: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
		if resp, err := client.Get("http://" + cfg.Listen + "/readyz"); err == nil {
			ready = resp.StatusCode == http.StatusOK
			_ = resp.Body.Close()
		}
	}
	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("runServe after shutdown: %v", err)
		}
	case <-time.After(45 * time.Second):
		t.Fatal("serve did not stop after cancel")
	}
	if _, err := openRaw(db); err != nil {
		t.Fatalf("database unusable after shutdown: %v", err)
	}
}

// OPS-018 / OPS-023: before a database is migrated, a verified backup is taken
// with reason pre-migration; a database with no recorded schema has nothing to
// protect and gets none.
func TestPreMigrationBackupIsTakenBeforeMigrating(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	logger := discardLogger()
	m := newMetrics(time.Now().UTC())
	if err := preMigrationBackup(ctx, cfg, logger, m, schemaState{Exists: true, Applied: 0, Known: 1}); err != nil {
		t.Fatalf("empty schema record: %v", err)
	}
	if n := countBackups(t, cfg.BackupDir); n != 0 {
		t.Fatalf("backup taken for a database with no schema record: %d", n)
	}
	state := schemaState{Exists: true, Applied: store.SchemaVersion(), Known: store.SchemaVersion()}
	if err := preMigrationBackup(ctx, cfg, logger, m, state); err != nil {
		t.Fatalf("pre-migration backup: %v", err)
	}
	entries, err := listBackups(cfg.BackupDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("backups = %v %v", entries, err)
	}
	if got := mustManifest(t, entries[0].path); got.Reason != "pre-migration" || got.Ledger.Status != "ok" {
		t.Errorf("manifest = %+v", got)
	}
	if m.backupOKAt == 0 {
		t.Error("pre-migration backup not recorded in metrics")
	}
}
