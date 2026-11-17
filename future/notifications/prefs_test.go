package notifications_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	n "github.com/djshellshoxxx/circula/future/notifications"
)

func TestDefaultsAreAllOff(t *testing.T) {
	e := newEnv(t)
	rows, err := e.svc.ListPreferences(context.Background(), actor(net1, "alice"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(n.Kinds)*len(n.Channels) {
		t.Fatalf("matrix size %d", len(rows))
	}
	for _, r := range rows {
		if r.Enabled || r.TZ != "UTC" || r.QuietStart != nil {
			t.Fatalf("default not off: %+v", r)
		}
	}
	// No preference => an event creates no outbox row on any channel.
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev1", MemberID: "alice", Kind: n.KindProposalReceived})
	if len(got) != 0 || e.count(`SELECT COUNT(*) FROM notification_delivery`) != 0 {
		t.Fatal("default-off violated")
	}
}

func TestSetPreferencesValidation(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	ctx := context.Background()
	bad := []n.PrefUpdate{
		{EventType: "nope", Channel: "email"},
		{EventType: n.KindDispute, Channel: "sms"},
		{EventType: n.KindDispute, Channel: "email", TZ: "Mars/Olympus"},
		{EventType: n.KindDispute, Channel: "email", TZ: "Local"},
		{EventType: n.KindDispute, Channel: "email", TZ: "../etc/passwd"},
		{EventType: n.KindDispute, Channel: "email", QuietStart: ptr(5)},
		{EventType: n.KindDispute, Channel: "email", QuietStart: ptr(-1), QuietEnd: ptr(5)},
		{EventType: n.KindDispute, Channel: "email", QuietStart: ptr(0), QuietEnd: ptr(1440)},
	}
	for i, u := range bad {
		err := e.svc.SetPreferences(ctx, a, []n.PrefUpdate{u})
		if !errors.Is(err, n.ErrInvalid) {
			t.Fatalf("case %d: want ErrInvalid, got %v", i, err)
		}
	}
	dup := []n.PrefUpdate{{EventType: n.KindDispute, Channel: "email"}, {EventType: n.KindDispute, Channel: "email"}}
	if err := e.svc.SetPreferences(ctx, a, dup); !errors.Is(err, n.ErrInvalid) {
		t.Fatal("duplicate rows accepted")
	}
	if err := e.svc.SetPreferences(ctx, a, nil); !errors.Is(err, n.ErrInvalid) {
		t.Fatal("empty accepted")
	}
	if e.count(`SELECT COUNT(*) FROM notification_preferences`) != 0 {
		t.Fatal("invalid update persisted")
	}
	// Equal quiet endpoints mean "no quiet hours" and are stored as NULL.
	if err := e.svc.SetPreferences(ctx, a, []n.PrefUpdate{{EventType: n.KindDispute, Channel: "email", Enabled: true, QuietStart: ptr(60), QuietEnd: ptr(60), TZ: "America/Los_Angeles"}}); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT COUNT(*) FROM notification_preferences WHERE quiet_start IS NULL AND quiet_end IS NULL`) != 1 {
		t.Fatal("equal endpoints not normalised")
	}
}

func TestPreferencesTenantAndMemberIsolation(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "email", n.KindDispute)
	for _, a := range []n.Actor{actor(net2, "alice"), actor(net1, "bob")} {
		rows, err := e.svc.ListPreferences(context.Background(), a)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.Enabled {
				t.Fatalf("%v sees alice's preference", a)
			}
		}
	}
}

func TestInactiveMemberCannotChangePreferences(t *testing.T) {
	e := newEnv(t)
	e.dir.SetActive(net1, "alice", false)
	err := e.svc.SetPreferences(context.Background(), actor(net1, "alice"), []n.PrefUpdate{{EventType: n.KindDispute, Channel: "email", Enabled: true}})
	if !errors.Is(err, n.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	if _, err := e.svc.AddPushSubscription(context.Background(), actor(net1, "alice"), n.PushSubscriptionInput{Endpoint: fcm}); !errors.Is(err, n.ErrForbidden) {
		t.Fatalf("push: %v", err)
	}
	if _, err := e.svc.AddEmailTarget(context.Background(), actor(net1, "alice"), "a@example.org"); !errors.Is(err, n.ErrForbidden) {
		t.Fatalf("email: %v", err)
	}
}

func TestPreferenceAuditTrail(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "email", n.KindDispute)
	e.optIn(a, "email", n.KindDispute) // unchanged: no second record
	if err := e.svc.SetPreferences(context.Background(), a, []n.PrefUpdate{{EventType: n.KindDispute, Channel: "email", Enabled: false}}); err != nil {
		t.Fatal(err)
	}
	recs, err := e.svc.ListAudit(context.Background(), a, 10)
	if err != nil || len(recs) != 2 {
		t.Fatalf("audit %v %v", recs, err)
	}
	if !strings.Contains(recs[0].Detail, "enabled=false") || !strings.Contains(recs[1].Detail, "enabled=true") {
		t.Fatalf("audit detail: %+v", recs)
	}
	e.amu.Lock()
	hook := len(e.audit)
	e.amu.Unlock()
	if hook != 2 {
		t.Fatalf("hook got %d records", hook)
	}
	// Other members never see it, and the table is append-only.
	other, _ := e.svc.ListAudit(context.Background(), actor(net1, "bob"), 10)
	if len(other) != 0 {
		t.Fatal("audit leaked across members")
	}
	if _, err := e.db.Exec(`UPDATE notification_audit SET detail='x'`); err == nil {
		t.Fatal("audit update allowed")
	}
	if _, err := e.db.Exec(`DELETE FROM notification_audit`); err == nil {
		t.Fatal("audit delete allowed")
	}
}

func TestDisablingCancelsReadyDeliveries(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute, n.KindProposalReceived)
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev1", MemberID: "alice", Kind: n.KindDispute})
	other := e.enqueue(n.Event{NetworkID: net1, EventID: "ev2", MemberID: "alice", Kind: n.KindProposalReceived})
	if err := e.svc.SetPreferences(context.Background(), a, []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push"}}); err != nil {
		t.Fatal(err)
	}
	if s, why := e.status(got[0].DeliveryID); s != "cancelled" || why != "opt_out" {
		t.Fatalf("dispute row: %s %s", s, why)
	}
	if s, _ := e.status(other[0].DeliveryID); s != "ready" {
		t.Fatalf("unrelated kind touched: %s", s)
	}
}
