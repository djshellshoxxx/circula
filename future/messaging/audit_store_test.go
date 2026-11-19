package messaging

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAuditExpiredContentCannotAcquireNewHold(t *testing.T) {
	f := setup(t)
	first := f.thread()
	f.send(first.ID, "expired unheld", "audit-send-01")
	held, e := f.s.Create(ctx, alice, "bob", "", "audit-create-held", 30)
	if e != nil {
		t.Fatal(e)
	}
	f.send(held.ID, "previously held", "audit-send-02")
	if e = f.s.Moderate(ctx, mod, held.ID, "hold", 0, ""); e != nil {
		t.Fatal(e)
	}
	f.now = f.now.Add(30 * 24 * time.Hour)
	if e = f.s.Moderate(ctx, mod, first.ID, "hold", 0, ""); !errors.Is(e, ErrDenied) {
		t.Fatal("expired unheld content resurrected", e)
	}
	if _, e = f.s.HeldMessages(ctx, mod, first.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	if e = f.s.Moderate(ctx, mod, held.ID, "release", 0, ""); e != nil {
		t.Fatal(e)
	}
	if e = f.s.Moderate(ctx, mod, held.ID, "hold", 0, ""); !errors.Is(e, ErrDenied) {
		t.Fatal("released expired evidence resurrected", e)
	}
	if n, e := f.s.Purge(ctx, mod); e != nil || n != 2 {
		t.Fatal(n, e)
	}
}

func TestAuditCreateQuotaSurvivesClosedChurn(t *testing.T) {
	f := setup(t)
	thread := f.thread()
	if e := f.s.CloseThread(ctx, alice, thread.ID); e != nil {
		t.Fatal(e)
	}
	tx, e := f.s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	for i := 0; i < 59; i++ {
		if _, e = tx.Exec(`INSERT INTO message_threads(network,id,a,b,created,activity,retention,closed) VALUES('network',?,'alice','bob',?,?,30,1)`, fmt.Sprintf("quota-%04d", i), f.now.UnixNano(), f.now.UnixNano()); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if retry, e := f.s.Create(ctx, alice, "bob", "listing-1", "create-01", 30); e != nil || retry.ID != thread.ID {
		t.Fatal(retry, e)
	}
	if _, e = f.s.Create(ctx, alice, "bob", "", "new-create-01", 30); !errors.Is(e, ErrQuota) {
		t.Fatal("closed thread churn bypass", e)
	}
	f.now = f.now.Add(time.Minute)
	if _, e = f.s.Create(ctx, alice, "bob", "", "new-create-01", 30); e != nil {
		t.Fatal("exact quota boundary", e)
	}
}

func TestAuditDailySendQuota(t *testing.T) {
	f := setup(t)
	th := f.thread()
	tx, e := f.s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	for i := 0; i < 1000; i++ {
		if _, e = tx.Exec(`INSERT INTO message_entries(network,thread,sequence,id,sender,body,created) VALUES('network',?,?,?,'alice','prior',?)`, th.ID, i+1, fmt.Sprintf("old-%04d", i), f.now.Add(-2*time.Minute).UnixNano()); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.Send(ctx, alice, th.ID, "new", "daily-quota-01"); !errors.Is(e, ErrQuota) {
		t.Fatal(e)
	}
}

func TestAuditBlockScopeAndQuota(t *testing.T) {
	f := setup(t)
	if e := f.s.Block(ctx, alice, "unknown", true); !errors.Is(e, ErrDenied) {
		t.Fatal("arbitrary block target", e)
	}
	tx, e := f.s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	for i := 0; i < 1000; i++ {
		if _, e = tx.Exec(`INSERT INTO message_blocks(network,actor,target) VALUES('network','alice',?)`, fmt.Sprintf("departed-%04d", i)); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = f.s.Block(ctx, alice, "bob", true); !errors.Is(e, ErrQuota) {
		t.Fatal(e)
	}
	if e = f.s.Block(ctx, alice, "departed-0000", false); e != nil {
		t.Fatal("unblock departed must remain possible", e)
	}
	if e = f.s.Block(ctx, alice, "bob", true); e != nil {
		t.Fatal(e)
	}
	if e = f.s.Block(ctx, alice, "bob", true); e != nil {
		t.Fatal("idempotent block at quota", e)
	}
}

func TestAuditCurrentPairPolicyOnMutationAndReplay(t *testing.T) {
	f := setup(t)
	th := f.thread()
	message := f.send(th.ID, "before policy change", "audit-pair-send")
	original := f.s.authorize
	f.s.authorize = func(ctx context.Context, tx *sql.Tx, a Actor, action, target string) error {
		if action == "conversation" && a.Member == "alice" && target == "bob" {
			return ErrDenied
		}
		return original(ctx, tx, a, action, target)
	}
	if _, e := f.s.Create(ctx, alice, "bob", "listing-1", "create-01", 30); !errors.Is(e, ErrDenied) {
		t.Fatal("creation retry bypasses changed pair policy", e)
	}
	if _, e := f.s.Send(ctx, alice, th.ID, "before policy change", "audit-pair-send"); !errors.Is(e, ErrDenied) {
		t.Fatal("send retry bypasses changed pair policy", e)
	}
	if _, e := f.s.Send(ctx, alice, th.ID, "after policy change", "audit-pair-new"); !errors.Is(e, ErrDenied) {
		t.Fatal(e)
	}
	messages, e := f.s.Messages(ctx, alice, th.ID, 0, 100)
	if e != nil || len(messages) != 1 || messages[0].ID != message.ID {
		t.Fatal("historical participant read should remain", messages, e)
	}
}
