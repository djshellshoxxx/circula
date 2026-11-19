package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestDualControlProposerUsesCurrentBreakGlassGrant(t *testing.T) {
	for _, state := range []string{"live", "expired", "closed", "permanent", "other network", "suspended"} {
		t.Run(state, func(t *testing.T) {
			d := newDC(t)
			bg := d.openBG(time.Hour)
			tu.Must(d.Svc.GrantBreakGlassAdmin(d.Ctx, sys, service.GrantBreakGlassInput{
				Network: d.net.ID, Login: "ana@example.org", Reason: "Temporary recovery access"}))
			pa := tu.Must(d.request(d.a, d.bM, 5000, service.AdjustCredit)).Pending
			if pa == nil {
				t.Fatal("above-threshold request did not require independent approval")
			}
			switch state {
			case "expired":
				d.Clock.Advance(time.Hour) // Exact expiry, without running the expiry job.
			case "closed", "permanent", "other network":
				if state == "permanent" {
					tu.Must(0, d.Svc.AssignRole(d.Ctx, sys, d.net.ID, d.aM.ID, domain.RoleAdmin))
				}
				tu.Must(d.Svc.CloseBreakGlass(d.Ctx, sys, bg.ID, "Recovery session completed"))
				if state == "other network" {
					other, _ := d.Network(d.admin, "other", nil)
					token, _ := tu.Must2(d.Svc.CreateInvitation(d.Ctx, d.admin, other.ID, service.InviteInput{}))
					tu.Must(d.Svc.RedeemInvitation(d.Ctx, d.a, token, "Ana"))
					tu.Must(d.Svc.OpenBreakGlass(d.Ctx, sys, service.OpenBreakGlassInput{
						Network: other.ID, Operator: "Test operator", Reason: "Other network recovery", TTL: time.Hour}))
					tu.Must(d.Svc.GrantBreakGlassAdmin(d.Ctx, sys, service.GrantBreakGlassInput{
						Network: other.ID, Login: "ana@example.org", Reason: "Only this other network"}))
				}
			case "suspended":
				tu.Must(0, d.Svc.SuspendMember(d.Ctx, d.admin, d.net.ID, d.aM.ID, "Suspended during review"))
			}
			beforeAudit := d.audits(domain.ActAdjustmentApproved)
			posted, err := d.Svc.ApproveAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID)
			if state == "live" || state == "permanent" {
				if err != nil || posted == nil || posted.Amount != 5000 {
					t.Fatalf("currently authorized proposer refused: transaction=%+v error=%v", posted, err)
				}
				again := tu.Must(d.Svc.ApproveAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
				if again.ID != posted.ID || d.balance(d.b).Balance != 5000 || len(d.entries(posted.ID)) != 2 {
					t.Fatal("retry did not preserve one balanced posting")
				}
				if got := d.audits(domain.ActAdjustmentApproved); got != beforeAudit+1 {
					t.Fatalf("approval recorded %d audit events; want one", got-beforeAudit)
				}
			} else {
				wantCode(t, err, domain.ErrInvalidState)
				if posted != nil || d.balance(d.b).Balance != 0 || d.audits(domain.ActAdjustmentApproved) != beforeAudit {
					t.Fatal("refused approval changed the ledger or appended an approval")
				}
				stored := tu.Must(d.Svc.GetPendingAdjustment(d.Ctx, d.admin, d.net.ID, pa.ID))
				if stored.Status != domain.PendingOpen || stored.TxID != "" {
					t.Fatalf("refused approval changed pending state: %+v", stored)
				}
			}
		})
	}
}

// Database failures are operational errors, not evidence that a proposer lost
// their authority. Neither membership nor temporary-role lookup errors may be
// swallowed or permit a posting.
func TestDualControlProposerLookupErrorsPropagate(t *testing.T) {
	for _, lookup := range []string{"member", "roles"} {
		t.Run(lookup, func(t *testing.T) {
			var wrapped *proposerLookupStore
			env := tu.NewEnvWith(t, func(s store.Store) store.Store {
				wrapped = &proposerLookupStore{Store: s}
				return wrapped
			})
			admin := env.Person("admin@example.org")
			net, _ := env.Network(admin, "lookup-errors", nil)
			proposer, pm := env.Member(admin, net, "proposer@example.org")
			target, tm := env.Member(admin, net, "target@example.org")
			_, backup := env.Member(admin, net, "backup@example.org")
			tu.Must(0, env.Svc.AssignRole(env.Ctx, sys, net.ID, backup.ID, domain.RoleAdmin))
			policy := net.Policy
			policy.DualControlThreshold = 1000
			tu.Must(env.Svc.UpdatePolicy(env.Ctx, admin, net.ID, service.PolicyChange{
				Policy: policy, Reason: "Independent approvals", IdempotencyKey: env.Key()}))
			tu.Must(env.Svc.OpenBreakGlass(env.Ctx, sys, service.OpenBreakGlassInput{
				Network: net.ID, Operator: "Test operator", Reason: "Recovery test", TTL: time.Hour}))
			tu.Must(env.Svc.GrantBreakGlassAdmin(env.Ctx, sys, service.GrantBreakGlassInput{
				Network: net.ID, Login: "proposer@example.org", Reason: "Temporary recovery access"}))
			pa := tu.Must(env.Svc.CreateAdjustmentRequest(env.Ctx, proposer, net.ID, service.AdjustInput{
				MemberID: tm.ID, Amount: 5000, Direction: service.AdjustCredit, Reason: "Correction", IdempotencyKey: env.Key()})).Pending
			failure := errors.New("injected proposer lookup failure")
			wrapped.memberID, wrapped.lookup, wrapped.failure = pm.ID, lookup, failure
			_, err := env.Svc.ApproveAdjustment(env.Ctx, admin, net.ID, pa.ID)
			if !errors.Is(err, failure) {
				t.Fatalf("%s lookup error was replaced: %v", lookup, err)
			}
			wrapped.failure = nil
			if b := tu.Must(env.Svc.GetBalance(env.Ctx, target, net.ID)); b.Balance != 0 {
				t.Fatalf("failed authorization posted credit: %d", b.Balance)
			}
			stored := tu.Must(env.Svc.GetPendingAdjustment(env.Ctx, admin, net.ID, pa.ID))
			if stored.Status != domain.PendingOpen || stored.TxID != "" {
				t.Fatal("failed lookup consumed the request")
			}
		})
	}
}

type proposerLookupStore struct {
	store.Store
	memberID, lookup string
	failure          error
}

func (s *proposerLookupStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&proposerLookupTx{Tx: tx, GovRepo: tx.(store.GovRepo), control: s})
	})
}

type proposerLookupTx struct {
	store.Tx
	store.GovRepo
	control *proposerLookupStore
}

func (t *proposerLookupTx) GetMember(netID, memberID string) (*domain.Member, error) {
	if t.control.failure != nil && t.control.lookup == "member" && memberID == t.control.memberID {
		return nil, t.control.failure
	}
	return t.Tx.GetMember(netID, memberID)
}

func (t *proposerLookupTx) LiveBreakGlassRoles(netID, memberID string, now time.Time) ([]domain.Role, error) {
	if t.control.failure != nil && t.control.lookup == "roles" && memberID == t.control.memberID {
		return nil, t.control.failure
	}
	return t.Tx.LiveBreakGlassRoles(netID, memberID, now)
}
