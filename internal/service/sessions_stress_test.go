package service_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

type sessionScanStore struct {
	store.Store
	calls atomic.Int64
	rows  atomic.Int64
}

type sessionScanTx struct {
	store.Tx
	parent *sessionScanStore
}

func (s *sessionScanStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(&sessionScanTx{Tx: tx, parent: s}) })
}

func (tx *sessionScanTx) ListSessions(owner string, position *store.SessionPosition, limit int) ([]*domain.Session, error) {
	tx.parent.calls.Add(1)
	rows, err := tx.Tx.ListSessions(owner, position, limit)
	tx.parent.rows.Add(int64(len(rows)))
	return rows, err
}

// API-036: measure the actual production SQLite/service boundary, including
// dead-history gaps that would defeat an SQL LIMIT applied after filtering.
func TestOwnSessionSQLiteStressHasConstantWorkAndAllocations(t *testing.T) {
	var scan *sessionScanStore
	e := tu.NewEnvConfig(t, func(st store.Store) store.Store {
		scan = &sessionScanStore{Store: st}
		return scan
	}, func(c *service.Config) {
		c.SessionIdle, c.SessionMax = 24*time.Hour, 24*time.Hour
		c.AdminSessionIdle, c.AdminSessionMax = 24*time.Hour, 24*time.Hour
		c.ReviewerSessionIdle, c.ReviewerSessionMax = 24*time.Hour, 24*time.Hour
	})
	e.Person("session-stress-owner")
	_, actor := selfSessionLogin(t, e, "session-stress-owner")
	e.Clock.Advance(time.Minute)
	now := e.Clock.Now().UnixMilli()
	insert := func(first, last int) {
		t.Helper()
		if _, err := e.Store.DB().Exec(`WITH RECURSIVE n(i) AS (VALUES(?) UNION ALL SELECT i+1 FROM n WHERE i < ?)
			INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at,revoked_at,public_id)
			SELECT printf('stress-private-%d',i),?,?,?,
			CASE WHEN i%2=0 THEN ? ELSE ? END,
			CASE WHEN i%2=1 THEN ? ELSE NULL END,printf('%064x',i) FROM n`,
			first, last, actor.PersonID, now, now, now, now+3600000, now); err != nil {
			t.Fatal(err)
		}
	}
	insert(1, 10000)
	var after string
	active, pages := 0, 0
	for ; pages < 7; pages++ {
		scan.calls.Store(0)
		scan.rows.Store(0)
		page, err := e.Svc.ListOwnSessions(e.Ctx, actor, service.Page{AfterID: after})
		if err != nil {
			t.Fatal(err)
		}
		if scan.calls.Load() > 20 || scan.rows.Load() > 2000 || len(page.Rows) > 25 {
			t.Fatalf("unbounded actual SQLite work: calls=%d rows=%d output=%d", scan.calls.Load(), scan.rows.Load(), len(page.Rows))
		}
		for _, row := range page.Rows {
			if !row.Current {
				t.Fatal("expired or revoked session escaped filtering")
			}
			active++
		}
		if page.Next == "" {
			pages++
			break
		}
		if page.Next == after {
			t.Fatal("empty filtered page did not make progress")
		}
		after = page.Next
	}
	if pages != 6 || active != 1 {
		t.Fatalf("current session lost behind retained history: pages=%d active=%d", pages, active)
	}
	allocations := func() float64 {
		return testing.AllocsPerRun(3, func() {
			scan.calls.Store(0)
			scan.rows.Store(0)
			page, err := e.Svc.ListOwnSessions(e.Ctx, actor, service.Page{})
			if err != nil || len(page.Rows) != 0 || page.Next == "" || scan.calls.Load() != 20 || scan.rows.Load() != 2000 {
				t.Fatalf("work-cap measurement changed semantics: %+v %v calls=%d rows=%d", page, err, scan.calls.Load(), scan.rows.Load())
			}
		})
	}
	small := allocations()
	insert(10001, 50000)
	large := allocations()
	// SQL and output work are identical. Allow driver/runtime bookkeeping
	// variance, while rejecting allocation growth proportional to history.
	if large > small*1.10+100 {
		t.Fatalf("allocations grew with retained history: 10001 rows=%.0f, 50001 rows=%.0f", small, large)
	}
	t.Logf("10001 and 50001 retained rows: 20 queries/2000 fetched rows per capped page; allocations/request %.0f and %.0f; complete 10001-row traversal used %d pages", small, large, pages)
}
