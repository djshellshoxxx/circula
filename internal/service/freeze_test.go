package service_test

import (
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// CMD-07 / OPS-033: a frozen network refuses writes with network-frozen,
// still serves reads and verification, and does not affect other networks.
func TestFrozenNetworkRefusesWritesOnly(t *testing.T) {
	w := newWorld(t, nil)
	net2, _ := w.Network(w.admin, "other", nil)
	c, _ := w.Member(w.admin, net2, "cy@example.org")
	_, dM := w.Member(w.admin, net2, "di@example.org")

	w.Svc.FreezeNetwork(w.net.ID, "test")
	if !w.Svc.IsFrozen(w.net.ID) || w.Svc.IsFrozen(net2.ID) {
		t.Fatal("freeze state wrong")
	}
	if got := w.Svc.FrozenNetworks(); len(got) != 1 || got[0] != w.net.ID {
		t.Fatalf("FrozenNetworks = %v", got)
	}

	_, err := w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrNetworkFrozen)
	if _, err := w.Svc.GetBalance(w.Ctx, w.a, w.net.ID); err != nil {
		t.Fatalf("read on frozen network: %v", err)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("verify on frozen network: %+v", rep)
	}
	// the verification run may still be recorded
	if _, err := w.Svc.VerifyAndRecordLedger(w.Ctx, w.admin, w.net.ID); err != nil {
		t.Fatalf("record run on frozen network: %v", err)
	}
	// member creation (invitation redeem / join) is refused too
	if _, _, err := w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}); err == nil {
		t.Fatal("invitation created on frozen network")
	}

	// another network keeps working
	if _, err := w.Svc.ProposeTransaction(w.Ctx, c, net2.ID, service.ProposeInput{CounterpartyID: dM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.Key()}); err != nil {
		t.Fatalf("unfrozen network refused: %v", err)
	}

	w.Svc.UnfreezeNetwork(w.net.ID)
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 100); err != nil {
		t.Fatalf("after unfreeze: %v", err)
	}
}

type countObs struct{ posted, replayed atomic.Int64 }

func (c *countObs) Posted(n int)   { c.posted.Add(int64(n)) }
func (c *countObs) Replayed(n int) { c.replayed.Add(int64(n)) }

// Metrics hooks: postings count once per committed posting; a replay counts
// the retry and not a posting; a rolled-back unit counts nothing.
func TestObserverCountsPostingsAndReplays(t *testing.T) {
	w := newWorld(t, nil)
	o := &countObs{}
	w.Svc.SetObserver(o)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	if o.posted.Load() != 1 {
		t.Fatalf("posted = %d, want 1", o.posted.Load())
	}
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: "same"}
	tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in))
	if o.replayed.Load() != 1 {
		t.Fatalf("replayed = %d, want 1", o.replayed.Load())
	}
	// refused posting (credit limit) must not count
	before := o.posted.Load()
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 1<<40); err == nil {
		t.Fatal("expected refusal")
	}
	if o.posted.Load() != before {
		t.Fatalf("refused posting counted: %d -> %d", before, o.posted.Load())
	}
}
