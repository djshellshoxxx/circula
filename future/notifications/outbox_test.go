package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	n "github.com/djshellshoxxx/circula/future/notifications"
)

func TestEnqueueAtomicWithSourceEvent(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "push", n.KindProposalReceived)
	boom := errors.New("source event failed")
	err := e.st.Update(context.Background(), func(ctx context.Context, q n.DBTX) error {
		if _, err := e.svc.EnqueueTx(ctx, q, n.Event{NetworkID: net1, EventID: "ev1", MemberID: "alice", Kind: n.KindProposalReceived}); err != nil {
			return err
		}
		return boom // the source transaction rolls back after the enqueue
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if e.count(`SELECT COUNT(*) FROM notification_delivery`) != 0 {
		t.Fatal("outbox row survived a rolled-back source event")
	}
	// Committed together, the row exists exactly once however often it is repeated.
	for i := 0; i < 3; i++ {
		got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev1", MemberID: "alice", Kind: n.KindProposalReceived})
		if len(got) != 1 || got[0].Created != (i == 0) {
			t.Fatalf("iteration %d: %+v", i, got)
		}
	}
	if e.count(`SELECT COUNT(*) FROM notification_delivery`) != 1 {
		t.Fatal("duplicate outbox rows")
	}
}

func TestDedupeIsPerNetworkRecipientChannelKind(t *testing.T) {
	e := newEnv(t)
	for _, a := range []n.Actor{actor(net1, "alice"), actor(net2, "alice"), actor(net1, "bob")} {
		e.optIn(a, "push", n.KindProposalReceived, n.KindDispute)
		e.optIn(a, "email", n.KindProposalReceived)
	}
	// Same event id in two networks / two recipients / two kinds is independent.
	for _, nw := range []string{net1, net2} {
		e.enqueue(n.Event{NetworkID: nw, EventID: "same", MemberID: "alice", Kind: n.KindProposalReceived})
	}
	e.enqueue(n.Event{NetworkID: net1, EventID: "same", MemberID: "bob", Kind: n.KindProposalReceived})
	e.enqueue(n.Event{NetworkID: net1, EventID: "same", MemberID: "alice", Kind: n.KindDispute})
	// 3 recipients x 2 channels (received) + 1 (dispute push only)
	if got := e.count(`SELECT COUNT(*) FROM notification_delivery`); got != 3*2+1 {
		t.Fatalf("rows=%d", got)
	}
}

func TestEnqueueValidationAndInactiveRecipient(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "push", n.KindDispute)
	for _, ev := range []n.Event{
		{NetworkID: net1, EventID: "", MemberID: "alice", Kind: n.KindDispute},
		{NetworkID: "", EventID: "e", MemberID: "alice", Kind: n.KindDispute},
		{NetworkID: net1, EventID: "e", MemberID: "alice", Kind: "bogus"},
		{NetworkID: net1, EventID: "e", MemberID: "alice", Kind: n.KindEmailVerification},
		{NetworkID: net1, EventID: "e", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(-time.Minute)},
	} {
		err := e.st.Update(context.Background(), func(ctx context.Context, q n.DBTX) error {
			_, err := e.svc.EnqueueTx(ctx, q, ev)
			return err
		})
		if !errors.Is(err, n.ErrInvalid) {
			t.Fatalf("%+v: %v", ev, err)
		}
	}
	e.dir.SetActive(net1, "alice", false)
	if got := e.enqueue(n.Event{NetworkID: net1, EventID: "e2", MemberID: "alice", Kind: n.KindDispute}); len(got) != 0 {
		t.Fatal("inactive member got an outbox row")
	}
}

func TestExpiryIsCappedAndDefaulted(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "push", n.KindDispute)
	a := e.enqueue(n.Event{NetworkID: net1, EventID: "a", MemberID: "alice", Kind: n.KindDispute})
	b := e.enqueue(n.Event{NetworkID: net1, EventID: "b", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(1000 * time.Hour)})
	exp := func(id string) int64 {
		var x int64
		if err := e.db.QueryRow(`SELECT expiry FROM notification_delivery WHERE id=?`, id).Scan(&x); err != nil {
			t.Fatal(err)
		}
		return x
	}
	if exp(a[0].DeliveryID) != t0.Add(24*time.Hour).Unix() || exp(b[0].DeliveryID) != t0.Add(7*24*time.Hour).Unix() {
		t.Fatal("expiry default/cap wrong")
	}
}

func TestMigrationRollbackAndReapply(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "push", n.KindDispute)
	if err := e.st.Migrate(context.Background()); err != nil { // idempotent
		t.Fatal(err)
	}
	if err := e.st.Rollback(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	var tables int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'notification_%' AND name<>'notification_schema_migrations' OR name='push_subscriptions'`).Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("tables left after rollback: %d %v", tables, err)
	}
	if err := e.st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT COUNT(*) FROM notification_preferences`) != 0 {
		t.Fatal("data survived rollback")
	}
	if n.SchemaVersion() != 1 {
		t.Fatal("schema version")
	}
}
