package service_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A1-05: old proposals in a frozen network must not consume every sweep's
// bounded batch and permanently starve other networks of expiry processing.
func TestExpiryFrozenNetworkDoesNotStarveActiveProposals(t *testing.T) {
	policy := func(p *domain.Policy) { p.ProposalExpiryDays = 2 }
	w := newWorld(t, policy)
	var frozen []*domain.Transaction
	for range 3 {
		frozen = append(frozen, tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100)))
	}
	w.Svc.FreezeNetwork(w.net.ID, "ledger verification failed")

	w.Clock.Advance(time.Hour)
	active, _ := w.Network(w.admin, "active", policy)
	c, _ := w.Member(w.admin, active, "cy@example.org")
	d, dM := w.Member(w.admin, active, "di@example.org")
	propose := func() *domain.Transaction {
		return tu.Must(w.Svc.ProposeTransaction(w.Ctx, c, active.ID, service.ProposeInput{
			CounterpartyID: dM.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.Key(),
		}))
	}
	var due []*domain.Transaction
	for range 3 {
		due = append(due, propose())
	}
	// These three share the exact expiry timestamp, exercising the ID tie-break.
	w.Clock.Advance(2 * 24 * time.Hour)
	future := propose()

	snapshotFrozen := func() string {
		t.Helper()
		var snapshot []any
		err := w.Store.View(w.Ctx, func(tx store.Tx) error {
			for _, p := range frozen {
				snapshot = append(snapshot, tu.Must(tx.GetTransaction(w.net.ID, p.ID)),
					tu.Must(tx.ListStatusEvents(w.net.ID, p.ID)), tu.Must(tx.ListEntries(w.net.ID, p.ID)))
			}
			for _, m := range []*domain.Member{w.aM, w.bM} {
				snapshot = append(snapshot, tu.Must(tx.GetAccountByMember(w.net.ID, m.ID)),
					tu.Must(tx.ListNotifications(w.net.ID, m.ID, store.NotificationFilter{Limit: 500})))
			}
			snapshot = append(snapshot, tu.Must(tx.ListAudit(w.net.ID, store.AuditFilter{Limit: 500})))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(tu.Must(json.Marshal(snapshot)))
	}
	before := snapshotFrozen()
	sweep := func(want int) {
		t.Helper()
		if got := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("expiry-fairness"), 2)); got != want {
			t.Fatalf("sweep expired %d proposals, want %d", got, want)
		}
	}
	sweep(2)
	sweep(1)
	sweep(0)
	if after := snapshotFrozen(); after != before {
		t.Fatal("expiry sweep mutated frozen transactions, accounts, ledger, audit, or inbox")
	}
	for _, p := range due {
		view := tu.Must(w.Svc.GetTransaction(w.Ctx, c, active.ID, p.ID))
		if view.Tx.Status != domain.TxExpired || len(view.History) != 2 || len(view.Entries) != 0 {
			t.Fatalf("active expiry was missed or posted money: %+v", view)
		}
	}
	for _, actor := range []domain.Actor{c, d} {
		n := 0
		for _, notice := range tu.Must(w.Svc.ListNotifications(w.Ctx, actor, active.ID, service.NotificationQuery{})) {
			if notice.Kind == service.NotifTradeExpired {
				n++
			}
		}
		if n != len(due) {
			t.Fatalf("member %s has %d expiry notices, want %d", actor.PersonID, n, len(due))
		}
	}
	if view := tu.Must(w.Svc.GetTransaction(w.Ctx, c, active.ID, future.ID)); view.Tx.Status != domain.TxProposed {
		t.Fatalf("future proposal expired early: %s", view.Tx.Status)
	}
	// Skipped proposals must remain eligible after thawing; no persistent scan
	// cursor may strand them behind the already processed active proposals.
	w.Svc.UnfreezeNetwork(w.net.ID)
	sweep(2)
	sweep(1)
	sweep(0)
	for _, p := range frozen {
		if view := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, p.ID)); view.Tx.Status != domain.TxExpired {
			t.Fatalf("thawed proposal %s was missed", p.ID)
		}
	}
	w.Clock.Advance(2 * 24 * time.Hour)
	sweep(1)
	sweep(0)
	if view := tu.Must(w.Svc.GetTransaction(w.Ctx, c, active.ID, future.ID)); view.Tx.Status != domain.TxExpired {
		t.Fatal("later expiry was missed")
	}
}

func TestExpiryManyFrozenNetworks(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ProposalExpiryDays = 2 })
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	// Freeze state is in memory and has no network-count limit. An exclusion
	// query using one SQL placeholder per ID would exceed SQLite's 32766 limit.
	for i := range 33000 {
		w.Svc.FreezeNetwork(fmt.Sprintf("frozen-%d", i), "verification failed")
	}
	w.Clock.Advance(2 * 24 * time.Hour)
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("many-frozen"), 1)); n != 1 {
		t.Fatalf("sweep expired %d proposals, want 1", n)
	}
	if view := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, p.ID)); view.Tx.Status != domain.TxExpired {
		t.Fatal("active proposal was missed with many frozen networks")
	}
}
