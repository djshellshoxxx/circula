package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestDualControlUnprovenAncestryPreservesState(t *testing.T) {
	for _, operationalFailure := range []bool{false, true} {
		name := "missing history"
		if operationalFailure {
			name = "lookup failure"
		}
		t.Run(name, func(t *testing.T) {
			var wrapped *grantHistoryStore
			e := tu.NewEnvWith(t, func(s store.Store) store.Store {
				wrapped = &grantHistoryStore{Store: s}
				return wrapped
			})
			admin := e.Person("admin@example.org")
			net, _ := e.Network(admin, "grant-history", nil)
			approver, approverRow := e.Member(admin, net, "approver@example.org")
			_, target := e.Member(admin, net, "member@example.org")
			tu.Must(0, e.Svc.AssignRole(e.Ctx, domain.SystemActor("fixture"), net.ID, approverRow.ID, domain.RoleAdmin))
			e.Clock.Advance(service.DefaultNewAdminApprovalDelay)
			policy := net.Policy
			policy.DualControlThreshold = 1000
			tu.Must(e.Svc.UpdatePolicy(e.Ctx, admin, net.ID, service.PolicyChange{
				Policy: policy, Reason: "Require independent approvals", IdempotencyKey: e.Key()}))
			pending := tu.Must(e.Svc.CreateAdjustmentRequest(e.Ctx, admin, net.ID, service.AdjustInput{
				MemberID: target.ID, Amount: 2000, Direction: service.AdjustCredit,
				Reason: "Correction awaiting review", IdempotencyKey: e.Key()})).Pending
			if pending == nil {
				t.Fatal("fixture did not create an open adjustment request")
			}
			wrapped.memberID = approverRow.ID
			var want error = domain.ErrForbidden
			if operationalFailure {
				wrapped.failure = errors.New("grant history unavailable")
				want = wrapped.failure
			}
			tables := []string{"pending_adjustments", "transactions", "entries", "accounts", "networks",
				"audit_events", "audit_heads", "notifications", "security_notices"}
			before := authorityTableRows(t, e, tables...)
			decidable, err := e.Svc.DecidableAdjustments(e.Ctx, approver, net.ID, []string{pending.ID})
			if operationalFailure {
				if !errors.Is(err, want) {
					t.Fatalf("decision hint replaced an operational failure: %v", err)
				}
			} else if err != nil {
				t.Fatalf("decision hint did not suppress unproven authority: %v", err)
			}
			if decidable[pending.ID] {
				t.Fatal("decision hint offered a request without established independence")
			}
			posted, err := e.Svc.ApproveAdjustment(e.Ctx, approver, net.ID, pending.ID)
			if !errors.Is(err, want) || posted != nil {
				t.Fatalf("approval must refuse unproven authority: %v", err)
			}
			rejected, err := e.Svc.RejectAdjustment(e.Ctx, approver, net.ID, pending.ID)
			if !errors.Is(err, want) || rejected != nil {
				t.Fatalf("rejection must refuse unproven authority: %v", err)
			}
			after := authorityTableRows(t, e, tables...)
			for _, table := range tables {
				if !reflect.DeepEqual(after[table], before[table]) {
					t.Errorf("refused decision changed %s", table)
				}
			}
		})
	}
}

// This fixture changes only the repository's historical-proof response for
// one ordinary administrator. Membership and request setup use supported APIs.
type grantHistoryStore struct {
	store.Store
	memberID string
	failure  error
}

func (s *grantHistoryStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&grantHistoryTx{Tx: tx, GovRepo: tx.(store.GovRepo), control: s})
	})
}

func (s *grantHistoryStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&grantHistoryTx{Tx: tx, GovRepo: tx.(store.GovRepo), control: s})
	})
}

type grantHistoryTx struct {
	store.Tx
	store.GovRepo
	control *grantHistoryStore
}

func (t *grantHistoryTx) LatestAdminGrant(netID, memberID string) (string, time.Time, bool, error) {
	if memberID == t.control.memberID {
		return "", time.Time{}, false, t.control.failure
	}
	return t.Tx.LatestAdminGrant(netID, memberID)
}
