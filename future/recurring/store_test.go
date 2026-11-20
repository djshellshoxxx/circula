package recurring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T, path string) (Service, Actor) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { db.Close() })
	now := at("2026-11-16T12:00:00Z")
	s := Service{DB: db, Enabled: true, Now: func() time.Time { return now }}
	s.Auth = func(ctx context.Context, tx *sql.Tx, a Actor, target string, amount *int64, action string) error {
		for _, m := range []string{a.Member, target} {
			if m == "" {
				continue
			}
			var active bool
			if err := tx.QueryRowContext(ctx, `SELECT active FROM fixture_members WHERE network=? AND member=?`, a.Network, m).Scan(&active); err != nil || !active {
				return ErrDenied
			}
		}
		if action == "draft" && amount != nil && *amount > 100 {
			return ErrDenied
		}
		return nil
	}
	ctx := context.Background()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS fixture_members(network TEXT,member TEXT,active INTEGER,PRIMARY KEY(network,member))`); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"n", "other"} {
		for _, m := range []string{"a", "b"} {
			if _, err = db.Exec(`INSERT OR IGNORE INTO fixture_members VALUES(?,?,1)`, n, m); err != nil {
				t.Fatal(err)
			}
		}
	}
	return s, Actor{"n", "a"}
}
func record() Record {
	amount := int64(10)
	return Record{Target: "b", Note: "<script>private</script>", Amount: &amount, Calendar: Calendar{Frequency: "daily", Zone: "UTC", Hour: 10}, Active: true}
}
func create(t *testing.T, s Service, a Actor, key string) Record {
	t.Helper()
	r, err := s.Save(context.Background(), a, record(), 0, key)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestCRUDIdempotencyAuthAndDraft(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := create(t, s, a, "create")
	retry := create(t, s, a, "create")
	if retry.ID != r.ID {
		t.Fatal("unstable create")
	}
	changed := record()
	changed.Note = "different"
	if _, err := s.Save(ctx, a, changed, 0, "create"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	rows, _, err := s.List(ctx, Actor{"other", "a"}, "", 10)
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	if _, err = s.Change(ctx, Actor{"n", "b"}, r.ID, "pause", r.Version, "key"); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
	first, err := s.CreateDraft(ctx, a, r.ID, "draft-1")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreateDraft(ctx, a, r.ID, "draft-1")
	if err != nil || again.Intent != first.Intent {
		t.Fatal(again, err)
	}
	next, err := s.CreateDraft(ctx, a, r.ID, "draft-2")
	if err != nil || next.Intent == first.Intent {
		t.Fatal(next, err)
	}
	paused, err := s.Change(ctx, a, r.ID, "pause", r.Version, "pause")
	if err != nil || paused.Active {
		t.Fatal(paused, err)
	}
	if _, err = s.Change(ctx, a, r.ID, "resume", r.Version, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	resumed, err := s.Change(ctx, a, r.ID, "resume", paused.Version, "resume")
	if err != nil || !resumed.Active {
		t.Fatal(resumed, err)
	}
	cancelled, err := s.Change(ctx, a, r.ID, "cancel", resumed.Version, "cancel")
	if err != nil || !cancelled.Cancelled || cancelled.Active {
		t.Fatal(cancelled, err)
	}
	if _, err = s.CreateDraft(ctx, a, r.ID, "cancelled"); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
	if _, err = s.Change(ctx, a, r.ID, "resume", cancelled.Version, "uncancel"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network='n' AND member='a'`); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.List(ctx, a, "", 10); !errors.Is(err, ErrDenied) {
		t.Fatal("stale membership", err)
	}
}
func TestFiringsAtomicRestartConcurrencyAndSkip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, a := fixture(t, path)
	ctx := context.Background()
	r := create(t, s, a, "create")
	s.Remind = func(context.Context, *sql.Tx, Reminder) error { return errors.New("inbox failed") }
	if _, err := s.RunDue(ctx, "n", "", 10); err == nil {
		t.Fatal("callback failure")
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM future_template_firings`).Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	s.Remind = nil
	other, _ := fixture(t, path)
	var wg sync.WaitGroup
	results := make(chan Scan, 2)
	errs := make(chan error, 2)
	for _, service := range []Service{s, other} {
		wg.Add(1)
		go func(service Service) {
			defer wg.Done()
			scan, err := service.RunDue(ctx, "n", "", 10)
			results <- scan
			errs <- err
		}(service)
	}
	wg.Wait()
	close(results)
	close(errs)
	total := 0
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for scan := range results {
		total += scan.Fired
	}
	if total != 1 {
		t.Fatal("duplicate firing", total)
	}
	reopened, _ := fixture(t, path)
	scan, err := reopened.RunDue(ctx, "n", "", 10)
	if err != nil || scan.Fired != 0 {
		t.Fatal(scan, err)
	}
	inbox, err := reopened.Inbox(ctx, a, 10)
	if err != nil || len(inbox) != 1 || inbox[0].Template != r.ID {
		t.Fatal(inbox, err)
	}
	foreign, err := reopened.Inbox(ctx, Actor{"other", "a"}, 10)
	if err != nil || len(foreign) != 0 {
		t.Fatal(foreign, err)
	}
	now := at("2026-11-17T09:00:00Z")
	reopened.Now = func() time.Time { return now }
	r, err = reopened.Change(ctx, a, r.ID, "skip", r.Version, "skip")
	if err != nil {
		t.Fatal(err)
	}
	now = at("2026-11-17T12:00:00Z")
	scan, err = reopened.RunDue(ctx, "n", "", 10)
	if err != nil || scan.Fired != 0 {
		t.Fatal("skip resurrected", scan, err)
	}
	now = at("2026-11-18T12:00:00Z")
	scan, err = reopened.RunDue(ctx, "n", "", 10)
	if err != nil || scan.Fired != 1 {
		t.Fatal(scan, err)
	}
	// Editing during today's window cannot bypass the stable window key.
	edit := record()
	edit.ID = r.ID
	edit.Calendar.Hour = 11
	updated, err := reopened.Save(ctx, a, edit, r.Version, "edit")
	if err != nil {
		t.Fatal(err)
	}
	scan, err = reopened.RunDue(ctx, "n", "", 10)
	if err != nil || scan.Fired != 0 {
		t.Fatal("edit duplicate", scan, err)
	}
	_ = updated
}
func TestQuotaBoundedScanAndSuspension(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		create(t, s, a, fmt.Sprintf("create%d", i))
	}
	if _, err := s.Save(ctx, a, record(), 0, "over"); err == nil {
		t.Fatal("quota")
	}
	cursor := ""
	fired := 0
	for i := 0; i < 25; i++ {
		scan, err := s.RunDue(ctx, "n", cursor, 3)
		if err != nil {
			t.Fatal(err)
		}
		fired += scan.Fired
		cursor = scan.Next
		if scan.Complete {
			break
		}
		if i == 24 {
			t.Fatal("scan never ends")
		}
	}
	if fired != 20 {
		t.Fatal("starved", fired)
	}
	if _, err := s.DB.Exec(`UPDATE fixture_members SET active=0 WHERE network='n' AND member='b'`); err != nil {
		t.Fatal(err)
	}
	scan, err := s.RunDue(ctx, "n", "", 100)
	if err != nil || scan.Suspended != 20 || scan.Fired != 0 {
		t.Fatal(scan, err)
	}
}
func TestInputAndDisabled(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	s.Enabled = false
	if _, err := s.Save(context.Background(), a, record(), 0, "key"); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
	s.Enabled = true
	bad := record()
	bad.Note = string([]byte{255})
	if _, err := s.Save(context.Background(), a, bad, 0, "key"); err == nil {
		t.Fatal("utf8")
	}
	bad = record()
	amount := int64(101)
	bad.Amount = &amount
	r, err := s.Save(context.Background(), a, bad, 0, "large")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateDraft(context.Background(), a, r.ID, "draft"); !errors.Is(err, ErrDenied) {
		t.Fatal("capacity", err)
	}
}

func TestBackupRestoreRetentionAndMigrationRollback(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "source.db")
	s, a := fixture(t, path)
	r := create(t, s, a, "create")
	if _, err := s.RunDue(ctx, "n", "", 20); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Change(ctx, a, r.ID, "cancel", r.Version, "cancel"); err != nil {
		t.Fatal(err)
	}
	now := at("2027-05-16T12:00:00Z")
	s.Now = func() time.Time { return now }
	if err := s.Cleanup(ctx, "n", now.Add(-91*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.List(ctx, a, "", 20)
	if err != nil || len(rows) != 1 || rows[0].Note != "" || rows[0].Amount != nil {
		t.Fatal(rows, err)
	}
	if _, err = s.Save(ctx, a, record(), 0, "create"); !errors.Is(err, ErrConflict) {
		t.Fatal("retired retry replay", err)
	}
	var firings int
	if err = s.DB.QueryRow(`SELECT COUNT(*) FROM future_template_firings`).Scan(&firings); err != nil || firings != 1 {
		t.Fatal("tombstones lost", firings, err)
	}
	if err = s.DB.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	restore := filepath.Join(dir, "restore.db")
	if err = os.WriteFile(restore, raw, 0600); err != nil {
		t.Fatal(err)
	}
	restored, _ := fixture(t, restore)
	if err = restored.DB.QueryRow(`PRAGMA integrity_check`).Scan(new(string)); err != nil {
		t.Fatal(err)
	}
	if err = restored.DB.QueryRow(`SELECT COUNT(*) FROM future_template_firings`).Scan(&firings); err != nil || firings != 1 {
		t.Fatal(firings, err)
	}
	badDB, err := sql.Open("sqlite", filepath.Join(dir, "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer badDB.Close()
	if _, err = badDB.Exec(`CREATE TABLE future_template_owner(x)`); err != nil {
		t.Fatal(err)
	}
	bad := Service{DB: badDB}
	if err = bad.Migrate(ctx); err == nil {
		t.Fatal("migration conflict accepted")
	}
	var count int
	if err = badDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='future_recurring_lock'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("migration leaked partial schema", count, err)
	}
}

func TestTimezoneEditGuardAndDeparture(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	ctx := context.Background()
	r := create(t, s, a, "create")
	scan, err := s.RunDue(ctx, "n", "", 100)
	if err != nil || scan.Fired != 1 {
		t.Fatal(scan, err)
	}
	edit := record()
	edit.ID = r.ID
	edit.Calendar.Zone = "Pacific/Kiritimati"
	edit.Calendar.Hour = 1
	if _, err = s.Save(ctx, a, edit, r.Version, "zone-edit"); err != nil {
		t.Fatal(err)
	}
	scan, err = s.RunDue(ctx, "n", "", 100)
	if err != nil || scan.Fired != 0 {
		t.Fatal("timezone changed logical window to duplicate", scan, err)
	}
	if err = s.Depart(ctx, "n", "b"); err != nil {
		t.Fatal(err)
	}
	rows, _, err := s.List(ctx, a, "", 100)
	if err != nil || len(rows) != 1 || !rows[0].Cancelled || rows[0].Note != "" || rows[0].Amount != nil {
		t.Fatal(rows, err)
	}
	if _, err = s.Change(ctx, a, r.ID, "resume", rows[0].Version, "resume-after-departure"); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestDowntimeBackfillIsOneCurrentWindow(t *testing.T) {
	s, a := fixture(t, filepath.Join(t.TempDir(), "state.db"))
	create(t, s, a, "create")
	now := at("2031-11-16T12:00:00Z")
	s.Now = func() time.Time { return now }
	scan, err := s.RunDue(context.Background(), "n", "", 100)
	if err != nil || scan.Fired != 1 {
		t.Fatal(scan, err)
	}
	scan, err = s.RunDue(context.Background(), "n", "", 100)
	if err != nil || scan.Fired != 0 {
		t.Fatal(scan, err)
	}
	var count int
	if err = s.DB.QueryRow(`SELECT COUNT(*) FROM future_template_firings`).Scan(&count); err != nil || count != 1 {
		t.Fatal("unbounded catchup", count, err)
	}
}
