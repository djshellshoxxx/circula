package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const (
	auditMirrorQueueSize = 256
	auditMirrorMaxRecord = 16 << 10
	auditMirrorDrainTime = 2 * time.Second
)

type mirrorDropReason int

const (
	mirrorDropFull mirrorDropReason = iota
	mirrorDropOversize
	mirrorDropEncoding
	mirrorDropClosed
	mirrorDropShutdown
	mirrorDropReasons
)

var mirrorDropNames = [mirrorDropReasons]string{"queue_full", "oversize", "encoding", "closed", "shutdown"}

// Audit mirror (RB-012). Committed metadata is offered to a bounded FIFO; one
// worker appends admitted records as JSON lines. Limits, by design:
//   - Best effort. Crashes, write errors, queue overflow, oversized records
//     and an expired shutdown drain can lose lines. Counters expose these
//     outcomes; the worker logs failures and coalesced drops off the store path.
//   - Only chain metadata is mirrored (ids, seq, time, actor, action, target,
//     request id, prev_hash, hash). Summaries, reasons and subject ids stay in
//     the database so erasure of personal data is not defeated by the mirror.
//   - The service opens the file append-only but cannot make it immutable. The
//     protection comes from where it points: a path the service account may
//     append to but not rewrite (chattr +a, a syslog or log-shipper pipe, an
//     object-locked mount).
type auditMirror struct {
	// mu protects admission versus closing the queue. It is never held while
	// writing a file, closing one, or logging; the store callback does no IO.
	mu       sync.Mutex
	closed   bool
	queue    chan []byte
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	f        io.WriteCloser
	logger   *slog.Logger
	closeErr error // published by closing done

	written atomic.Int64
	failed  atomic.Int64
	pending atomic.Int64 // queued plus the current delivery attempt
	dropped [mirrorDropReasons]atomic.Int64

	// Reporting state belongs to the single delivery worker.
	reported   [mirrorDropReasons]int64
	reportedAt time.Time
}

type auditMirrorStats struct {
	Written, Failed, Pending int64
	Dropped                  [mirrorDropReasons]int64
}

func (m *auditMirror) stats() auditMirrorStats {
	s := auditMirrorStats{Written: m.written.Load(), Failed: m.failed.Load(), Pending: m.pending.Load()}
	for reason := range mirrorDropReasons {
		s.Dropped[reason] = m.dropped[reason].Load()
	}
	return s
}

type mirrorLine struct {
	NetworkID  string `json:"network_id"`
	Seq        int64  `json:"seq"`
	ID         string `json:"id"`
	OccurredAt string `json:"occurred_at"`
	ActorType  string `json:"actor_type"`
	ActorID    string `json:"actor_id"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	RequestID  string `json:"request_id"`
	PrevHash   string `json:"prev_hash"`
	Hash       string `json:"hash"`
}

func openAuditMirror(path string, logger *slog.Logger) (*auditMirror, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // operator-supplied path
	if err != nil {
		return nil, err
	}
	return newAuditMirror(f, logger), nil
}

func newAuditMirror(f io.WriteCloser, logger *slog.Logger) *auditMirror {
	m := &auditMirror{f: f, logger: logger, queue: make(chan []byte, auditMirrorQueueSize),
		stop: make(chan struct{}), done: make(chan struct{})}
	go m.deliver()
	return m
}

// write is the store's post-commit sink. It copies only bounded metadata into
// an encoded record and makes a nonblocking queue offer; it never waits for
// the destination, the logger, or a metrics scrape.
func (m *auditMirror) write(e *domain.AuditEvent) {
	line := mirrorLine{
		NetworkID: e.NetworkID, Seq: e.Seq, ID: e.ID, OccurredAt: e.OccurredAt.UTC().Format(time.RFC3339Nano),
		ActorType: string(e.ActorType), ActorID: e.ActorID, Action: e.Action, TargetType: e.TargetType,
		TargetID: e.TargetID, RequestID: e.RequestID, PrevHash: e.PrevHash, Hash: e.Hash,
	}
	// Reject oversized source strings before JSON escaping can allocate a
	// large record. The subsequent exact cap includes escaping and framing.
	if !line.withinLimit() {
		m.dropped[mirrorDropOversize].Add(1)
		return
	}
	b, err := json.Marshal(line)
	if err != nil {
		m.dropped[mirrorDropEncoding].Add(1)
		return
	}
	if len(b) >= auditMirrorMaxRecord { // reserve one byte for the newline
		m.dropped[mirrorDropOversize].Add(1)
		return
	}
	// Keep exactly the framed record, without retaining spare encoder capacity.
	record := make([]byte, len(b)+1)
	copy(record, b)
	record[len(b)] = '\n'
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		m.dropped[mirrorDropClosed].Add(1)
		return
	}
	m.pending.Add(1)
	select {
	case m.queue <- record:
	default:
		m.pending.Add(-1)
		m.dropped[mirrorDropFull].Add(1)
	}
}

func (l mirrorLine) withinLimit() bool {
	n := 0
	for _, field := range [...]string{l.NetworkID, l.ID, l.OccurredAt, l.ActorType, l.ActorID,
		l.Action, l.TargetType, l.TargetID, l.RequestID, l.PrevHash, l.Hash} {
		if len(field) > auditMirrorMaxRecord-n {
			return false
		}
		n += len(field)
	}
	return true
}

func (m *auditMirror) deliver() {
	defer close(m.done)
	// The worker owns Close as well as Write: either can stall for an unusual
	// destination, and neither is allowed to hold the shutdown caller.
	defer func() { m.closeErr = m.f.Close() }()
	defer m.reportDrops(true)
	for b := range m.queue {
		select {
		case <-m.stop:
			m.dropped[mirrorDropShutdown].Add(1)
			m.pending.Add(-1)
			m.discardQueued()
			return
		default:
		}
		n, err := m.f.Write(b)
		if err == nil && n != len(b) {
			err = io.ErrShortWrite
		}
		firstFailure := false
		if err != nil {
			firstFailure = m.failed.Add(1) == 1
		} else {
			m.written.Add(1)
		}
		m.pending.Add(-1)
		if firstFailure {
			m.logger.Error("audit mirror write failed; mirror may be incomplete", "error", err)
		}
		m.reportDrops(false)
	}
}

func (m *auditMirror) reportDrops(force bool) {
	current := m.stats().Dropped
	if current == m.reported || (!force && !m.reportedAt.IsZero() && time.Since(m.reportedAt) < time.Minute) {
		return
	}
	attrs := make([]any, 0, int(mirrorDropReasons)*2)
	for reason, count := range current {
		attrs = append(attrs, mirrorDropNames[reason], count)
	}
	m.logger.Warn("audit mirror records dropped; mirror may be incomplete", attrs...)
	m.reported, m.reportedAt = current, time.Now()
}

func (m *auditMirror) discardQueued() {
	for {
		select {
		case _, ok := <-m.queue:
			if !ok {
				return
			}
			m.dropped[mirrorDropShutdown].Add(1)
			m.pending.Add(-1)
		default:
			return
		}
	}
}

// close stops admission and lets the single worker drain until ctx expires.
// On expiry, queued records are discarded and counted; a delivery already in
// progress remains unconfirmed and can finish later. No extra goroutine is
// created to interrupt a blocked Write or Close.
func (m *auditMirror) close(ctx context.Context) error {
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		close(m.queue)
	}
	m.mu.Unlock()
	select {
	case <-m.done:
		return m.closeErr
	default:
	}
	select {
	case <-m.done:
		return m.closeErr
	case <-ctx.Done():
		m.stopOnce.Do(func() { close(m.stop) })
		m.discardQueued()
		return ctx.Err()
	}
}
