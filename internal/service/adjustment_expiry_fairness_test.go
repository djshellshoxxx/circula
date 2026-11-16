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

// A1-10: expiry must leave frozen requests and their audit chain untouched,
// without letting their older deadlines monopolise the bounded batch.
func TestAdjustmentExpirySkipsFrozenNetworksWithoutStarvation(t *testing.T) {
	d := newDC(t)
	var frozen []*domain.PendingAdjustment
	for range 3 {
		frozen = append(frozen, d.pending(5000))
	}
	d.Svc.FreezeNetwork(d.net.ID, "ledger verification failed")
	d.Clock.Advance(time.Hour)
	active, _ := d.Network(d.admin, "active", func(p *domain.Policy) { p.DualControlThreshold = 1000 })
	_, secondAdmin := d.Member(d.admin, active, "other-admin@example.org")
	tu.Must(0, d.Svc.AssignRole(d.Ctx, domain.SystemActor("fixture"), active.ID, secondAdmin.ID, domain.RoleAdmin))
	target, targetM := d.Member(d.admin, active, "other-member@example.org")
	request := func() *domain.PendingAdjustment {
		return tu.Must(d.Svc.CreateAdjustmentRequest(d.Ctx, d.admin, active.ID, service.AdjustInput{
			MemberID: targetM.ID, Amount: 5000, Direction: service.AdjustCredit,
			Reason: "correction", IdempotencyKey: d.Key(),
		})).Pending
	}
	var due []*domain.PendingAdjustment
	for range 3 {
		due = append(due, request())
	}
	d.Clock.Advance(7 * 24 * time.Hour)
	future := request()

	snapshotFrozen := func() string {
		t.Helper()
		var snapshot []any
		err := d.Store.View(d.Ctx, func(tx store.Tx) error {
			snapshot = append(snapshot,
				tu.Must(tx.ListPendingAdjustments(d.net.ID, nil, "", 500)),
				tu.Must(tx.ListAudit(d.net.ID, store.AuditFilter{Limit: 500})),
				tu.Must(tx.GetAccountByMember(d.net.ID, d.aM.ID)),
				tu.Must(tx.LedgerSeq(d.net.ID)))
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
		if got := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, domain.SystemActor("expiry"), 2)); got != want {
			t.Fatalf("sweep expired %d requests, want %d", got, want)
		}
	}
	for _, count := range []int{2, 1, 0} {
		sweep(count)
		if after := snapshotFrozen(); after != before {
			t.Fatal("expiry changed frozen requests, audit, account, or ledger")
		}
	}
	for _, p := range due {
		got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, active.ID, p.ID))
		if got.Status != domain.PendingExpired || got.TxID != "" {
			t.Fatalf("active request was missed or posted money: %+v", got)
		}
	}
	if got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, active.ID, future.ID)); got.Status != domain.PendingOpen {
		t.Fatal("future request expired early")
	}
	if balance := tu.Must(d.Svc.GetBalance(d.Ctx, target, active.ID)); balance.Balance != 0 {
		t.Fatal("expiry posted money")
	}
	if events := tu.Must(d.Svc.ListAudit(d.Ctx, d.admin, active.ID, service.AuditQuery{Action: domain.ActAdjustmentExpired})); len(events) != 3 {
		t.Fatalf("active expiry audit count = %d, want 3", len(events))
	}
	d.Svc.UnfreezeNetwork(d.net.ID)
	sweep(2)
	sweep(1)
	sweep(0)
	for _, p := range frozen {
		if got := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, p.ID)); got.Status != domain.PendingExpired {
			t.Fatalf("thawed request %s was missed", p.ID)
		}
	}
	if d.audits(domain.ActAdjustmentExpired) != 3 {
		t.Fatal("thawed requests must have exactly one expiry audit each")
	}
	d.Clock.Advance(7 * 24 * time.Hour)
	sweep(1)
	sweep(0)
}

func TestAdjustmentExpiryManyFrozenNetworks(t *testing.T) {
	d := newDC(t)
	d.pending(5000)
	for i := range 33000 { // exceed SQLite's per-statement bound-variable limit
		d.Svc.FreezeNetwork(fmt.Sprintf("frozen-%d", i), "verification failed")
	}
	d.Clock.Advance(7 * 24 * time.Hour)
	if n := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, domain.SystemActor("expiry"), 1)); n != 1 {
		t.Fatalf("sweep expired %d requests, want 1", n)
	}
}

func TestAdjustmentExpiryBoundsOversizedBatch(t *testing.T) {
	d := newDC(t)
	for range 101 {
		d.pending(5000)
	}
	d.Clock.Advance(7 * 24 * time.Hour)
	if n := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, domain.SystemActor("expiry"), 501)); n != 100 {
		t.Fatalf("oversized batch expired %d requests, want default 100", n)
	}
	if n := tu.Must(d.Svc.ExpirePendingAdjustments(d.Ctx, domain.SystemActor("expiry"), 500)); n != 1 {
		t.Fatalf("next batch expired %d requests, want remaining 1", n)
	}
}
