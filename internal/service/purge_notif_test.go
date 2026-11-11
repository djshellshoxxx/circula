package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Read notifications are purged after the retention period; unread ones stay.
func TestPurgeReadNotifications(t *testing.T) {
	w := newWorld(t, nil)
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100)) // notifies ben
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 100)) // notifies ana
	tu.Must(w.Svc.MarkNotificationsRead(w.Ctx, w.b, w.net.ID, nil))
	w.Clock.Advance((service.NotificationRetentionDays - 1) * 24 * time.Hour)
	if rep := tu.Must(w.Svc.Purge(w.Ctx)); rep.Notifications != 0 {
		t.Fatalf("purged too early: %+v", rep)
	}
	w.Clock.Advance(2 * 24 * time.Hour)
	rep := tu.Must(w.Svc.Purge(w.Ctx))
	if rep.Notifications == 0 {
		t.Fatalf("read notification not purged: %+v", rep)
	}
	if got := len(tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{}))); got != 0 {
		t.Errorf("ben still has %d notifications", got)
	}
	if got := len(tu.Must(w.Svc.ListNotifications(w.Ctx, w.a, w.net.ID, service.NotificationQuery{}))); got == 0 {
		t.Error("unread notification purged")
	}
}
