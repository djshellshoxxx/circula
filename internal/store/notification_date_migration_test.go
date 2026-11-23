package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Existing notices have no reconstructable event instant. New context survives a restart
// and follows the same erasure boundary as its body.
func TestNotificationDateMigrationAndErasure(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "notices.db")
	legacy, err := store.OpenAtVersion(ctx, path, 25)
	if err != nil {
		t.Fatal(err)
	}
	seedV1Ledger(t, legacy.DB())
	if _, err := legacy.DB().Exec(`INSERT INTO notifications(id,network_id,member_id,kind,title,body,created_at)
		VALUES ('old','n1','m2','policy.notice','Rules','Recorded 1 Mar 2026.', 123)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	date := domain.NotificationDate{Event: domain.NoticeClosureScheduled, At: time.UnixMilli(456).UTC(), ClosureReason: "retired", ReadOnlyDays: 90}
	n := store.Notification{ID: "new", NetworkID: "n1", MemberID: "m1", Kind: "network.closure_scheduled", Title: "Closure", Body: "stored fallback", CreatedAt: date.At, EventDate: &date}
	if err := db.Update(ctx, func(tx store.Tx) error {
		created, err := tx.InsertNotification(n)
		if err == nil && !created {
			t.Error("dated notification missing")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	read := func(member string) *store.Notification {
		t.Helper()
		var result *store.Notification
		if err := db.View(ctx, func(tx store.Tx) error {
			rows, err := tx.ListNotifications("n1", member, store.NotificationFilter{Limit: 10})
			if err == nil && len(rows) == 1 {
				result = rows[0]
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if result == nil {
			t.Fatal("expected one notification for", member)
		}
		return result
	}
	if old := read("m2"); old.EventDate != nil || old.Body != "Recorded 1 Mar 2026." || old.CreatedAt.UnixMilli() != 123 {
		t.Fatalf("old row changed: %+v", old)
	}
	if current := read("m1"); current.EventDate == nil || *current.EventDate != date || current.Body != n.Body {
		t.Fatalf("new row changed: %+v", current)
	}
	if err := db.Update(ctx, func(tx store.Tx) error {
		redacter := tx.(interface {
			RedactNotificationNames(string, string, string) (int64, error)
		})
		_, err := redacter.RedactNotificationNames("n1", "m1", "")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if current := read("m1"); current.Body != "" || current.EventDate != nil {
		t.Fatalf("erased context remains: %+v", current)
	}
	if old := read("m2"); old.Body != "Recorded 1 Mar 2026." {
		t.Fatal("erasure affected another member")
	}
}
