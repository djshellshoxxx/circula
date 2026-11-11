package store_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// RB-003: text deleted from a non-ledger table does not remain in the
// database file or its WAL.
func TestSecureDeleteLeavesNoCanary(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "c.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	canary := "canary-login-7f3a9c1e2b-do-not-keep"
	if _, err := s.DB().ExecContext(ctx, `INSERT INTO login_throttle(login,failures,locked_until) VALUES (?,1,0)`, canary); err != nil {
		t.Fatal(err)
	}
	checkpoint := func() {
		if _, err := s.DB().ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			t.Fatal(err)
		}
	}
	has := func() bool {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if w, err := os.ReadFile(path + "-wal"); err == nil {
			b = append(b, w...)
		}
		return bytes.Contains(b, []byte(canary))
	}
	checkpoint()
	if !has() {
		t.Fatal("test is blind: canary not found in the file before deletion")
	}
	if _, err := s.DB().ExecContext(ctx, `DELETE FROM login_throttle WHERE login = ?`, canary); err != nil {
		t.Fatal(err)
	}
	checkpoint()
	if has() {
		t.Fatal("deleted text remains in the database file (secure_delete is off)")
	}
}

// RB-012: the audit sink sees each committed event, in order, and nothing
// from a rolled-back unit.
func TestAuditSinkAfterCommitOnly(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)
	var got []int64
	s.SetAuditSink(func(e *domain.AuditEvent) { got = append(got, e.Seq) })
	ev := func(seq int64) *domain.AuditEvent {
		return &domain.AuditEvent{ID: "e" + string(rune('0'+seq)), Seq: seq, OccurredAt: time.Unix(1, 0), ActorType: domain.ActorSystem,
			Action: "x.y", Hash: "h", PrevHash: "p"}
	}
	err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertAudit(ev(1)); err != nil {
			return err
		}
		return context.Canceled // roll back
	})
	if err == nil || len(got) != 0 {
		t.Fatalf("rolled back unit reached the sink: %v %v", err, got)
	}
	if err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertAudit(ev(1)); err != nil {
			return err
		}
		return tx.InsertAudit(ev(2))
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("sink saw %v", got)
	}
}
