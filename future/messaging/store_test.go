package messaging

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()
var alice = Actor{"network", "alice"}
var bob = Actor{"network", "bob"}
var mod = Actor{"network", "moderator"}

type fixture struct {
	t    testing.TB
	path string
	s    *Store
	now  time.Time
}

func setup(t testing.TB) *fixture {
	t.Helper()
	f := &fixture{t: t, path: filepath.Join(t.TempDir(), "messages.sqlite"), now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	f.open()
	if e := f.s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := f.s.db.Exec(`CREATE TABLE memberships(network TEXT,member TEXT,active INTEGER,PRIMARY KEY(network,member)); INSERT INTO memberships VALUES('network','alice',1),('network','bob',1),('network','carol',1),('network','moderator',1),('other','alice',1)`); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.s.Close() })
	return f
}
func (f *fixture) open() {
	s, e := Open(f.path, func(ctx context.Context, tx *sql.Tx, a Actor, action, target string) error {
		var active bool
		if e := tx.QueryRowContext(ctx, `SELECT active FROM memberships WHERE network=? AND member=?`, a.Network, a.Member).Scan(&active); e != nil || !active {
			return ErrDenied
		}
		switch action {
		case "member", "conversation":
			return nil
		case "moderate", "retention":
			if a.Member == "moderator" {
				return nil
			}
		case "reference":
			if target == "listing-1" {
				return nil
			}
		case "retention-days":
			if target == "1" || target == "30" {
				return nil
			}
		}
		return ErrDenied
	})
	if e != nil {
		f.t.Fatal(e)
	}
	s.now = func() time.Time { return f.now }
	f.s = s
}
func (f *fixture) thread() Conversation {
	f.t.Helper()
	th, e := f.s.Create(ctx, alice, "bob", "listing-1", "create-01", 30)
	if e != nil {
		f.t.Fatal(e)
	}
	return th
}
func (f *fixture) send(id, body, key string) Message {
	f.t.Helper()
	m, e := f.s.Send(ctx, alice, id, body, key)
	if e != nil {
		f.t.Fatal(e)
	}
	return m
}

func TestDurableOrderingIdempotencyAndFreshAuthorization(t *testing.T) {
	f := setup(t)
	th := f.thread()
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := f.s.Send(ctx, alice, th.ID, "literal", fmt.Sprintf("send-%08d", i))
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	list, e := f.s.Messages(ctx, bob, th.ID, 0, 100)
	if e != nil || len(list) != 40 {
		t.Fatal(len(list), e)
	}
	for i, m := range list {
		if m.Sequence != int64(i+1) {
			t.Fatal(m)
		}
	}
	m, e := f.s.Send(ctx, alice, th.ID, "literal", "send-00000000")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.Send(ctx, alice, th.ID, "different", "send-00000000"); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
	if e = f.s.MarkRead(ctx, bob, th.ID, 30); e != nil {
		t.Fatal(e)
	}
	if e = f.s.MarkRead(ctx, bob, th.ID, 10); e != nil {
		t.Fatal(e)
	}
	if e = f.s.MarkRead(ctx, bob, th.ID, 41); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}
	f.s.Close()
	f.open()
	again, e := f.s.Send(ctx, alice, th.ID, "literal", "send-00000000")
	if e != nil || again.ID != m.ID {
		t.Fatal(again, e)
	}
	threads, e := f.s.Threads(ctx, bob, "", 100)
	if e != nil || len(threads) != 1 || threads[0].Unread != 10 {
		t.Fatal(threads, e)
	}
	for _, a := range []Actor{{"network", "carol"}, {"other", "alice"}, {"network", ""}} {
		if _, e = f.s.Messages(ctx, a, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
			t.Fatal(a, e)
		}
	}
	if _, e = f.s.db.Exec(`UPDATE memberships SET active=0 WHERE member='bob'`); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.Messages(ctx, bob, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e = f.s.Send(ctx, alice, th.ID, "removed target", "send-removed"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e = f.s.Create(ctx, alice, "bob", "", "create-removed", 30); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
}
func TestBlocksReportsHoldsAndRetention(t *testing.T) {
	f := setup(t)
	th := f.thread()
	f.send(th.ID, "report this", "message-01")
	if e := f.s.Block(ctx, bob, "alice", true); e != nil {
		t.Fatal(e)
	}
	if _, e := f.s.Send(ctx, alice, th.ID, "blocked", "message-02"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e := f.s.Create(ctx, alice, "bob", "", "create-02", 30); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e := f.s.Messages(ctx, bob, th.ID, 0, 100); e != nil {
		t.Fatal("blocks preserve history", e)
	}
	f.s.Close()
	f.open()
	if _, e := f.s.Send(ctx, bob, th.ID, "blocked too", "message-03"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if e := f.s.Block(ctx, bob, "alice", false); e != nil {
		t.Fatal(e)
	}
	if e := f.s.Report(ctx, bob, th.ID, 1, "spam"); e != nil {
		t.Fatal(e)
	}
	if e := f.s.Report(ctx, bob, th.ID, 1, "duplicate"); e != nil {
		t.Fatal(e)
	}
	if e := f.s.Report(ctx, Actor{"network", "carol"}, th.ID, 1, "spy"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if e := f.s.Report(ctx, bob, th.ID, 99, "missing"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if e := f.s.Moderate(ctx, bob, th.ID, "hold", 0, ""); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if e := f.s.Moderate(ctx, mod, th.ID, "hold", 0, ""); e != nil {
		t.Fatal(e)
	}
	f.now = f.now.Add(30 * 24 * time.Hour)
	if _, e := f.s.Messages(ctx, bob, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal("expiry boundary", e)
	}
	var export bytes.Buffer
	if e := f.s.Export(ctx, bob, th.ID, &export); !errors.Is(e, ErrDenied) || export.Len() != 0 {
		t.Fatal(export.String(), e)
	}
	if n, e := f.s.Purge(ctx, mod); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	reports, e := f.s.Reports(ctx, mod, th.ID)
	if e != nil || len(reports) != 1 || reports[0].Reason != "spam" {
		t.Fatal(reports, e)
	}
	if msgs, e := f.s.HeldMessages(ctx, mod, th.ID, 0, 100); e != nil || len(msgs) != 1 {
		t.Fatal(msgs, e)
	}
	if e = f.s.Moderate(ctx, mod, th.ID, "resolve", 1, "bob"); e != nil {
		t.Fatal(e)
	}
	if e = f.s.Moderate(ctx, mod, th.ID, "release", 0, ""); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.HeldMessages(ctx, mod, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if n, e := f.s.Purge(ctx, mod); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	for _, table := range []string{"message_entries", "message_reports", "message_idempotency"} {
		var count int
		if e = f.s.db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); e != nil || count != 0 {
			t.Fatal(table, count, e)
		}
	}
	f.s.Close()
	f.open()
	if _, e = f.s.Messages(ctx, alice, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
}
func TestOwnViewHideAndActivityDeadline(t *testing.T) {
	f := setup(t)
	th := f.thread()
	f.send(th.ID, "private <script>", "message-01")
	if e := f.s.Hide(ctx, bob, th.ID); e != nil {
		t.Fatal(e)
	}
	if msgs, e := f.s.Messages(ctx, bob, th.ID, 0, 100); e != nil || len(msgs) != 0 {
		t.Fatal(msgs, e)
	}
	if threads, e := f.s.Threads(ctx, bob, "", 100); e != nil || len(threads) != 0 {
		t.Fatal(threads, e)
	}
	if msgs, e := f.s.Messages(ctx, alice, th.ID, 0, 100); e != nil || len(msgs) != 1 {
		t.Fatal(msgs, e)
	}
	if e := f.s.Report(ctx, bob, th.ID, 1, "hidden"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	f.now = f.now.Add(29 * 24 * time.Hour)
	f.send(th.ID, "new", "message-02")
	f.now = f.now.Add(24 * time.Hour)
	if msgs, e := f.s.Messages(ctx, bob, th.ID, 0, 100); e != nil || len(msgs) != 1 || msgs[0].Body != "new" {
		t.Fatal(msgs, e)
	}
	f.now = f.now.Add(29 * 24 * time.Hour)
	if _, e := f.s.Messages(ctx, alice, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
}
func TestRollbackQuotaAndValidation(t *testing.T) {
	f := setup(t)
	th := f.thread()
	if _, e := NewStore(f.s.db, nil); !errors.Is(e, ErrInvalid) {
		t.Fatal(e)
	}
	if _, e := f.s.Create(ctx, alice, "bob", "foreign", "create-invalid", 30); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e := f.s.Create(ctx, alice, "bob", "", "create-invalid", 2); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	for _, body := range []string{"", strings.Repeat("x", 4097), "a\x00b", string([]byte{0xff})} {
		if _, e := f.s.Send(ctx, alice, th.ID, body, "message-invalid"); !errors.Is(e, ErrInvalid) {
			t.Fatal(e)
		}
	}
	if _, e := f.s.db.Exec(`CREATE TRIGGER fail_idem BEFORE INSERT ON message_idempotency WHEN NEW.operation='send' BEGIN SELECT RAISE(ABORT,'injected'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := f.s.Send(ctx, alice, th.ID, "rollback", "message-rollback"); e == nil {
		t.Fatal("expected injected error")
	}
	msgs, e := f.s.Messages(ctx, alice, th.ID, 0, 100)
	if e != nil || len(msgs) != 0 {
		t.Fatal(msgs, e)
	}
	if _, e = f.s.db.Exec(`DROP TRIGGER fail_idem`); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 60; i++ {
		f.send(th.ID, "quota", fmt.Sprintf("message-%03d", i))
	}
	if _, e = f.s.Send(ctx, alice, th.ID, "over", "message-over"); !errors.Is(e, ErrQuota) {
		t.Fatal(e)
	}
	if e = f.s.CloseThread(ctx, bob, th.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.Send(ctx, bob, th.ID, "closed", "message-closed"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if _, e = f.s.Messages(ctx, bob, th.ID, 0, 100); e != nil {
		t.Fatal(e)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = f.s.Threads(canceled, alice, "", 10); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func TestMultipleConnectionsSerialize(t *testing.T) {
	f := setup(t)
	th := f.thread()
	other, e := Open(f.path, f.s.authorize)
	if e != nil {
		t.Fatal(e)
	}
	other.now = f.s.now
	defer other.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := f.s
			if i%2 == 0 {
				s = other
			}
			_, e := s.Send(ctx, alice, th.ID, "concurrent", fmt.Sprintf("shared-%08d", i))
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	msgs, e := f.s.Messages(ctx, bob, th.ID, 0, 100)
	if e != nil || len(msgs) != 20 {
		t.Fatal(msgs, e)
	}
}

func TestReportPaginationAndHiddenRetry(t *testing.T) {
	f := setup(t)
	th := f.thread()
	f.send(th.ID, "first", "message-first")
	f.send(th.ID, "second", "message-second")
	for _, seq := range []int64{1, 2} {
		if e := f.s.Report(ctx, bob, th.ID, seq, "spam"); e != nil {
			t.Fatal(e)
		}
	}
	first, e := f.s.ReportsPage(ctx, mod, th.ID, 0, "", 1)
	if e != nil || len(first) != 1 || first[0].Sequence != 1 {
		t.Fatal(first, e)
	}
	second, e := f.s.ReportsPage(ctx, mod, th.ID, first[0].Sequence, first[0].Reporter, 1)
	if e != nil || len(second) != 1 || second[0].Sequence != 2 {
		t.Fatal(second, e)
	}
	if e = f.s.Hide(ctx, alice, th.ID); e != nil {
		t.Fatal(e)
	}
	again, e := f.s.Send(ctx, alice, th.ID, "first", "message-first")
	if e != nil || again.Body != "" {
		t.Fatal(again, e)
	}
}
