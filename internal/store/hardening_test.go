package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// SVC-02: a write through View fails and persists nothing, and the read-only
// mode never leaks to pooled connections later used for writes.
func TestViewIsReadOnly(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ { // cycle the pool well past its size
		err := s.View(ctx, func(tx store.Tx) error {
			return tx.InsertPerson(&domain.Person{ID: "v", Login: "viewer", CreatedAt: now}, "h")
		})
		if err == nil || !strings.Contains(err.Error(), "readonly") {
			t.Fatalf("write in View: err = %v", err)
		}
	}
	var n int
	_ = s.View(ctx, func(tx store.Tx) error { _, _, _, e := tx.GetThrottle("x"); return e })
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM persons`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("persons = %d err=%v", n, err)
	}
	// writers still work on every pooled connection afterwards
	for i := 0; i < 30; i++ {
		id := string(rune('a' + i))
		if err := s.Update(ctx, func(tx store.Tx) error {
			return tx.InsertPerson(&domain.Person{ID: id, Login: "u" + id, CreatedAt: now}, "h")
		}); err != nil {
			t.Fatalf("Update after View: %v", err)
		}
	}
	// and concurrent Views do not poison concurrent Updates
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- s.View(ctx, func(tx store.Tx) error { _, _, err := tx.GetPerson("a"); return err })
		}()
		go func(i int) {
			defer wg.Done()
			errs <- s.Update(ctx, func(tx store.Tx) error {
				return tx.PutThrottle("t"+string(rune('a'+i)), 1, now)
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("mixed load: %v", err)
		}
	}
}

// CMD-05: lock contention from another process becomes the retryable
// "unavailable" domain error and leaves no partial state.
func TestBusyDatabaseIsRetryable(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "busy.db")
	s, err := store.OpenWith(ctx, path, store.Options{BusyTimeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	other, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(50)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })
	conn, err := other.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil { // a second writer holds the lock
		t.Fatal(err)
	}

	start := time.Now()
	err = s.Update(ctx, func(tx store.Tx) error {
		return tx.PutThrottle("x", 1, time.Now())
	})
	if domain.CodeOf(err) != domain.CodeUnavailable || !domain.IsRetryable(err) || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("Update under contention: %v (%s)", err, domain.CodeOf(err))
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("waited %v; busy timeout not honoured", d)
	}
	if strings.Contains(err.Error(), "SQLITE") || strings.Contains(err.Error(), "locked") {
		t.Errorf("driver text leaked into the message: %v", err)
	}
	var de *domain.Error
	if !errors.As(err, &de) || de.Cause == nil {
		t.Error("cause not kept for logs")
	}

	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	// after the lock is released the same call succeeds, and nothing was left behind
	var n int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM login_throttle`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("partial state: %d err=%v", n, err)
	}
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.PutThrottle("x", 1, time.Now()) }); err != nil {
		t.Fatalf("after release: %v", err)
	}
}

func TestPurgePrimitives(t *testing.T) {
	s := openStore(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertPerson(&domain.Person{ID: "p", Login: "p@example.org", CreatedAt: now}, "h"); err != nil {
			return err
		}
		for i, sess := range []domain.Session{
			{IDHash: "live", CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)},
			{IDHash: "expired", CreatedAt: now, LastSeenAt: now, ExpiresAt: now},
			{IDHash: "idle", CreatedAt: now, LastSeenAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(time.Hour)},
		} {
			sess.PersonID = "p"
			if err := tx.InsertSession(&sess); err != nil {
				return err
			}
			_ = i
		}
		if err := tx.PutThrottle("old", 3, now.Add(-2*time.Hour)); err != nil {
			return err
		}
		return tx.PutThrottle("locked", 5, now.Add(time.Hour))
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Update(ctx, func(tx store.Tx) error {
		n, err := tx.DeleteDeadSessions(now, now.Add(-24*time.Hour), now.Add(-24*time.Hour), 1)
		if err != nil || n != 1 { // batch limit honoured
			t.Errorf("limited delete: n=%d err=%v", n, err)
		}
		n, err = tx.DeleteDeadSessions(now, now.Add(-24*time.Hour), now.Add(-24*time.Hour), 100)
		if err != nil || n != 1 {
			t.Errorf("second delete: n=%d err=%v", n, err)
		}
		if _, err := tx.GetSession("live"); err != nil {
			t.Errorf("live session deleted: %v", err)
		}
		n, err = tx.DeleteStaleThrottles(now.Add(-time.Hour), 100)
		if err != nil || n != 1 {
			t.Errorf("throttle delete: n=%d err=%v", n, err)
		}
		if _, _, found, _ := tx.GetThrottle("locked"); !found {
			t.Error("active lock deleted")
		}
		return nil
	})
}

// SA-09: upgrading a database that already holds audit events backfills the
// chain heads from the current tail.
func TestMigration0002BackfillsAuditHeads(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "up.db")
	s, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := func(id, net string, seq int64) *domain.AuditEvent {
		return &domain.AuditEvent{ID: id, NetworkID: net, Seq: seq, OccurredAt: now, ActorType: domain.ActorSystem,
			Action: "x", TargetType: "t", TargetID: "t", Hash: "h" + id}
	}
	if err := s.Update(ctx, func(tx store.Tx) error {
		for _, e := range []*domain.AuditEvent{ev("1", "", 1), ev("2", "", 2), ev("3", "", 3), ev("4", "netA", 1), ev("5", "netA", 2)} {
			if err := tx.InsertAudit(e); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// roll the schema back to v1 as an old database would be
	for _, q := range []string{`DROP TABLE audit_heads`, `DROP INDEX audit_actor_action`, `DROP INDEX sessions_expiry`,
		`DROP INDEX sessions_seen`, `DROP INDEX idem_expiry`, `DROP INDEX throttle_locked`,
		`DELETE FROM schema_migrations WHERE version = 2`} {
		if _, err := s.DB().Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	_ = s.Close()
	s, err = store.Open(ctx, path)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	_ = s.View(ctx, func(tx store.Tx) error {
		for net, want := range map[string]struct {
			seq  int64
			hash string
		}{"": {3, "h3"}, "netA": {2, "h5"}} {
			seq, hash, found, err := tx.AuditHead(net)
			if err != nil || !found || seq != want.seq || hash != want.hash {
				t.Errorf("head(%q) = %d %s %v %v", net, seq, hash, found, err)
			}
		}
		return nil
	})
}
