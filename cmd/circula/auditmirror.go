package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Audit mirror (RB-012). With audit-mirror-file set, every audit event is
// appended as one JSON line after its transaction commits. Limits, by design:
//   - Best effort. A crash between commit and write, or a full disk, loses
//     lines; failures are logged and counted, never fail the request.
//   - Only chain metadata is mirrored (ids, seq, time, actor, action, target,
//     request id, prev_hash, hash). Summaries, reasons and subject ids stay in
//     the database so erasure of personal data is not defeated by the mirror.
//   - The service opens the file append-only but cannot make it immutable. The
//     protection comes from where it points: a path the service account may
//     append to but not rewrite (chattr +a, a syslog or log-shipper pipe, an
//     object-locked mount).
type auditMirror struct {
	mu     sync.Mutex
	f      *os.File
	logger *slog.Logger
	failed atomic.Int64
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
	return &auditMirror{f: f, logger: logger}, nil
}

// write appends one event. It is the store's post-commit sink.
func (m *auditMirror) write(e *domain.AuditEvent) {
	b, err := json.Marshal(mirrorLine{
		NetworkID: e.NetworkID, Seq: e.Seq, ID: e.ID, OccurredAt: e.OccurredAt.UTC().Format(time.RFC3339Nano),
		ActorType: string(e.ActorType), ActorID: e.ActorID, Action: e.Action, TargetType: e.TargetType,
		TargetID: e.TargetID, RequestID: e.RequestID, PrevHash: e.PrevHash, Hash: e.Hash,
	})
	if err == nil {
		m.mu.Lock()
		_, err = m.f.Write(append(b, '\n'))
		m.mu.Unlock()
	}
	if err != nil {
		if m.failed.Add(1) == 1 {
			m.logger.Error("audit mirror write failed; mirror may be incomplete", "error", err)
		}
	}
}

func (m *auditMirror) close() error { return m.f.Close() }
