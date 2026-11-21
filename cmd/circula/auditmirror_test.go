package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type mirrorTestWriter struct {
	write func([]byte) (int, error)
	close func() error
}

func (w mirrorTestWriter) Write(p []byte) (int, error) { return w.write(p) }
func (w mirrorTestWriter) Close() error {
	if w.close != nil {
		return w.close()
	}
	return nil
}

func mirrorEvent(seq int64) *domain.AuditEvent {
	return &domain.AuditEvent{ID: fmt.Sprintf("event-%d", seq), NetworkID: "network-fixture", Seq: seq,
		OccurredAt: time.Unix(seq, 0), ActorType: domain.ActorSystem, Action: "fixture.committed",
		TargetType: "network", TargetID: "network-fixture", RequestID: "fixture-request", PrevHash: "previous", Hash: "current",
		SummaryBefore: "PRIVATE-BEFORE", SummaryAfter: "PRIVATE-AFTER", Reason: "PRIVATE-REASON", SubjectIDs: []string{"PRIVATE-SUBJECT"}}
}

func awaitMirrorSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func finishMirror(t *testing.T, m *auditMirror) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.close(ctx); err != nil {
		t.Fatalf("drain audit mirror: %v", err)
	}
}

func mirrorMetrics(m *auditMirror) string {
	metrics := newMetrics(time.Now())
	metrics.auditMirror = m
	var out strings.Builder
	metrics.write(&out, dynamicMetrics{now: time.Now()})
	return out.String()
}

// OPS-02 / RB-012: the post-commit callback acknowledges ordinary service
// transactions and releases the store's writer gate while delivery is paused.
func TestAuditMirrorDoesNotHoldTransactionAcknowledgement(t *testing.T) {
	e := testutil.NewEnv(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var started, released sync.Once
	m := newAuditMirror(mirrorTestWriter{write: func(p []byte) (int, error) {
		started.Do(func() { close(entered) })
		<-release
		return len(p), nil
	}}, discardLogger())
	e.Store.SetAuditSink(m.write)
	t.Cleanup(func() {
		e.Store.SetAuditSink(nil)
		released.Do(func() { close(release) })
		finishMirror(t, m)
	})
	register := func(login string) <-chan error {
		result := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(e.Ctx, 3*time.Second)
			defer cancel()
			_, err := e.Svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: "correct horse battery"})
			result <- err
		}()
		return result
	}
	first := register("mirror-first@example.org")
	awaitMirrorSignal(t, entered, "the inert destination's first delivery")
	for i, done := range []<-chan error{first, register("mirror-second@example.org")} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("ordinary transaction %d: %v", i+1, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("ordinary transaction %d waited for mirror delivery", i+1)
		}
	}
	var people, audits int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM persons`).Scan(&people); err != nil {
		t.Fatal(err)
	}
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if people != 2 || audits != 2 {
		t.Fatalf("acknowledgements did not correspond to committed rows: people=%d audits=%d", people, audits)
	}
	if state := m.stats(); state.Pending != 2 || state.Written != 0 {
		t.Fatalf("paused delivery state = %+v", state)
	}
	released.Do(func() { close(release) })
	finishMirror(t, m)
	if state := m.stats(); state.Written != 2 || state.Pending != 0 || state.Failed != 0 {
		t.Fatalf("resumed delivery state = %+v", state)
	}
}

// The single worker preserves FIFO order after a pause. The finite queue
// drops new offers when full, and the counters remain readable while blocked.
func TestAuditMirrorBoundedQueuePreservesOrder(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var started, released sync.Once
	var calls atomic.Int64
	var buf bytes.Buffer
	m := newAuditMirror(mirrorTestWriter{write: func(p []byte) (int, error) {
		calls.Add(1)
		started.Do(func() { close(entered) })
		<-release
		return buf.Write(p)
	}}, discardLogger())
	t.Cleanup(func() { released.Do(func() { close(release) }); finishMirror(t, m) })
	first := mirrorEvent(1)
	m.write(first)
	awaitMirrorSignal(t, entered, "the first delivery")
	first.Action = "changed.after.admission" // the queued bytes must own their metadata
	for i := int64(2); i <= auditMirrorQueueSize+2; i++ {
		m.write(mirrorEvent(i))
	}
	state := m.stats()
	if state.Pending != auditMirrorQueueSize+1 || state.Dropped[mirrorDropFull] != 1 || calls.Load() != 1 {
		t.Fatalf("queue or worker bound = %+v, destination calls=%d", state, calls.Load())
	}
	metrics := mirrorMetrics(m)
	for _, want := range []string{
		"circula_audit_mirror_enabled 1", "circula_audit_mirror_pending_records 257",
		`circula_audit_mirror_dropped_total{reason="queue_full"} 1`,
	} {
		if !strings.Contains(metrics, want) {
			t.Errorf("blocked mirror metrics missing %q", want)
		}
	}
	released.Do(func() { close(release) })
	finishMirror(t, m)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != auditMirrorQueueSize+1 || strings.Contains(buf.String(), "PRIVATE-") {
		t.Fatalf("unexpected line count or private body data: lines=%d", len(lines))
	}
	for i, raw := range lines {
		var line mirrorLine
		if err := json.Unmarshal([]byte(raw), &line); err != nil || line.Seq != int64(i+1) || line.Action != "fixture.committed" {
			t.Fatalf("FIFO line %d = %+v, err=%v", i+1, line, err)
		}
	}
	m.write(mirrorEvent(999))
	if state = m.stats(); state.Pending != 0 || state.Written != auditMirrorQueueSize+1 || state.Dropped[mirrorDropClosed] != 1 {
		t.Fatalf("drained/closed state = %+v", state)
	}
}

func TestAuditMirrorRejectsRawAndEncodedOversize(t *testing.T) {
	var buf bytes.Buffer
	m := newAuditMirror(mirrorTestWriter{write: buf.Write}, discardLogger())
	t.Cleanup(func() { finishMirror(t, m) })
	raw := mirrorEvent(1)
	raw.RequestID = strings.Repeat("a", auditMirrorMaxRecord+1)
	m.write(raw)
	escaped := mirrorEvent(2)
	escaped.RequestID = strings.Repeat(`"`, auditMirrorMaxRecord/2)
	m.write(escaped)
	m.write(mirrorEvent(3))
	finishMirror(t, m)
	state := m.stats()
	if state.Dropped[mirrorDropOversize] != 2 || state.Written != 1 || state.Pending != 0 {
		t.Fatalf("oversize accounting = %+v", state)
	}
	var line mirrorLine
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil || line.Seq != 3 {
		t.Fatalf("oversize rejection changed the following record: %+v, %v", line, err)
	}
}

func TestAuditMirrorWriteFailuresAreObservable(t *testing.T) {
	var attempts []int64
	var logs bytes.Buffer
	m := newAuditMirror(mirrorTestWriter{write: func(p []byte) (int, error) {
		var line mirrorLine
		if err := json.Unmarshal(p, &line); err != nil {
			return 0, err
		}
		attempts = append(attempts, line.Seq)
		switch line.Seq {
		case 1:
			return 0, errors.New("controlled destination failure")
		case 2:
			return len(p) - 1, nil // an ordinary io.Writer short-write failure
		default:
			return len(p), nil
		}
	}}, slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { finishMirror(t, m) })
	for i := int64(1); i <= 3; i++ {
		m.write(mirrorEvent(i))
	}
	finishMirror(t, m)
	if state := m.stats(); state.Failed != 2 || state.Written != 1 || state.Pending != 0 {
		t.Fatalf("delivery failure accounting = %+v", state)
	}
	if fmt.Sprint(attempts) != "[1 2 3]" || strings.Count(logs.String(), "audit mirror write failed") != 1 {
		t.Fatalf("failure order/reporting: attempts=%v logs=%s", attempts, logs.String())
	}
	for _, want := range []string{"circula_audit_mirror_write_failures_total 2", "circula_audit_mirror_written_total 1"} {
		if !strings.Contains(mirrorMetrics(m), want) {
			t.Errorf("missing failure metric %q", want)
		}
	}
}

// An expired drain drops only records that have not started. The blocked
// write is explicitly unconfirmed; it can still finish after close returns.
func TestAuditMirrorShutdownBoundsBlockedWrite(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	closed := make(chan struct{})
	var started, released sync.Once
	m := newAuditMirror(mirrorTestWriter{
		write: func(p []byte) (int, error) { started.Do(func() { close(entered) }); <-release; return len(p), nil },
		close: func() error { close(closed); return nil },
	}, discardLogger())
	t.Cleanup(func() { released.Do(func() { close(release) }); finishMirror(t, m) })
	m.write(mirrorEvent(1))
	awaitMirrorSignal(t, entered, "the blocked write")
	m.write(mirrorEvent(2))
	m.write(mirrorEvent(3))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- m.close(ctx) }()
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expired drain = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown waited for the blocked destination")
	}
	if state := m.stats(); state.Dropped[mirrorDropShutdown] != 2 || state.Pending != 1 || state.Written != 0 || state.Failed != 0 {
		t.Fatalf("timeout misclassified the in-flight write: %+v", state)
	}
	select {
	case <-closed:
		t.Fatal("shutdown caller closed the destination while its write was blocked")
	default:
	}
	released.Do(func() { close(release) })
	awaitMirrorSignal(t, m.done, "the existing delivery worker to exit")
	if state := m.stats(); state.Written != 1 || state.Pending != 0 || state.Dropped[mirrorDropShutdown] != 2 {
		t.Fatalf("late completion was lost or queued records were delivered: %+v", state)
	}
}

func TestAuditMirrorShutdownBoundsBlockedClose(t *testing.T) {
	closing, release := make(chan struct{}), make(chan struct{})
	var released sync.Once
	m := newAuditMirror(mirrorTestWriter{
		write: io.Discard.Write,
		close: func() error { close(closing); <-release; return nil },
	}, discardLogger())
	t.Cleanup(func() { released.Do(func() { close(release) }); finishMirror(t, m) })
	m.write(mirrorEvent(1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- m.close(ctx) }()
	awaitMirrorSignal(t, closing, "the destination's close")
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("interrupted close = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown waited for the blocked Close")
	}
	if state := m.stats(); state.Pending != 0 || state.Written != 1 || state.Dropped[mirrorDropShutdown] != 0 {
		t.Fatalf("blocked Close misclassified completed delivery: %+v", state)
	}
	released.Do(func() { close(release) })
	awaitMirrorSignal(t, m.done, "the existing worker to finish Close")
}

func TestAuditMirrorConcurrentAdmissionAndClose(t *testing.T) {
	m := newAuditMirror(mirrorTestWriter{write: io.Discard.Write}, discardLogger())
	t.Cleanup(func() { finishMirror(t, m) })
	var producers sync.WaitGroup
	const records = 64
	for i := int64(1); i <= records; i++ {
		producers.Go(func() { m.write(mirrorEvent(i)) })
	}
	finishMirror(t, m)
	producers.Wait()
	state := m.stats()
	counted := state.Written + state.Failed
	for _, n := range state.Dropped {
		counted += n
	}
	if state.Pending != 0 || counted != records {
		t.Fatalf("admission/close lost an outcome: counted=%d state=%+v", counted, state)
	}
}
