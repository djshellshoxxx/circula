package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// API-027: a service/web/operator transaction can change a policy after the
// API has read its ETag. Conditional updates must check inside their own write
// transaction, including changes to the pending version and its activation.
func TestPolicyConditionalWriteRejectsInterveningChange(t *testing.T) {
	for _, change := range []string{"immediate", "schedule", "cancel", "activate", "wildcard"} {
		t.Run(change, func(t *testing.T) {
			var wrapped *policyRaceStore
			env := testutil.NewEnvWith(t, func(s store.Store) store.Store {
				wrapped = &policyRaceStore{Store: s}
				return wrapped
			})
			w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{Now: env.Clock.Now})}
			t.Cleanup(func() { api.CloseHandler(w.h) })
			w.admin = env.Person("admin")
			w.net, w.adminM = env.Network(w.admin, "policy-race", nil)
			w.alice, w.aliceM = env.Member(w.admin, w.net, "alice")
			w.adminT = w.token(w.admin, domain.ScopeRead, domain.ScopeWrite)
			schedule := func() {
				pol := w.net.Policy
				pol.CurrencyName = "scheduled credits"
				days := 14
				testutil.Must(env.Svc.UpdatePolicy(env.Ctx, w.admin, w.net.ID, service.PolicyChange{
					Policy: pol, Reason: "scheduled by another administrator", NoticeDays: &days, IdempotencyKey: env.Key(),
				}))
			}
			if change == "cancel" || change == "activate" {
				schedule()
			}
			const path = "/api/v1/admin/policy"
			current := w.do(req{method: "GET", path: path, token: w.adminT})
			wantStatus(t, current, 200)
			etag := current.Header().Get("ETag")
			body := m{"reason": "conditional edit"}
			if change == "immediate" || change == "wildcard" {
				body["default_credit_limit_minor"] = "20000"
			}
			if change == "wildcard" {
				etag = "*"
			}
			var afterChange policyRowCounts
			intervened := false
			wrapped.afterPolicyRead = func() {
				switch change {
				case "immediate", "wildcard":
					pol := w.net.Policy
					pol.DefaultCreditLimit = 11000
					testutil.Must(env.Svc.UpdatePolicy(env.Ctx, w.admin, w.net.ID, service.PolicyChange{
						Policy: pol, Reason: "intervening effective change", IdempotencyKey: env.Key(),
					}))
				case "schedule":
					// Without an atomic check, the stale request's effective values
					// would silently cancel this newly announced policy.
					schedule()
				case "cancel":
					testutil.Must(env.Svc.UpdatePolicy(env.Ctx, w.admin, w.net.ID, service.PolicyChange{
						Policy: w.net.Policy, Reason: "cancel scheduled change", IdempotencyKey: env.Key(),
					}))
				case "activate":
					env.Clock.Advance(14 * 24 * time.Hour)
				}
				afterChange = policyCounts(t, env)
				intervened = true
			}
			result := w.do(req{method: "PATCH", path: path, token: w.adminT, body: body,
				hdr: map[string]string{"If-Match": etag, "Idempotency-Key": w.key()}})
			if !intervened {
				t.Fatal("test did not intervene between precondition read and write")
			}
			if change == "wildcard" {
				wantStatus(t, result, 200)
				got := testutil.Must(env.Svc.GetNetwork(env.Ctx, w.admin, w.net.ID))
				if got.PolicyVersion != 3 || got.Policy.DefaultCreditLimit != 20000 {
					t.Fatalf("wildcard update no longer unconditional: %+v", got)
				}
				return
			}
			wantProblem(t, result, 412, "precondition-failed")
			if got := policyCounts(t, env); got != afterChange {
				t.Fatalf("refused update wrote policy/audit/notifications/idempotency: before=%+v after=%+v", afterChange, got)
			}
			if change == "schedule" {
				log := testutil.Must(env.Svc.PolicyLog(env.Ctx, w.admin, w.net.ID))
				if len(log) != 2 || !log[0].Pending || log[0].Withdrawn {
					t.Fatalf("stale update cancelled the new pending policy: %+v", log)
				}
			}
		})
	}
}

type policyRowCounts struct{ policies, audit, notifications, idempotency int }

func policyCounts(t *testing.T, env *testutil.Env) policyRowCounts {
	t.Helper()
	var c policyRowCounts
	err := env.Store.DB().QueryRow(`SELECT
		(SELECT COUNT(*) FROM network_policies), (SELECT COUNT(*) FROM audit_events),
		(SELECT COUNT(*) FROM notifications), (SELECT COUNT(*) FROM idempotency_keys)`).
		Scan(&c.policies, &c.audit, &c.notifications, &c.idempotency)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type policyRaceStore struct {
	store.Store
	afterPolicyRead func()
}

func (s *policyRaceStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if s.afterPolicyRead == nil {
		return s.Store.View(ctx, fn)
	}
	read := false
	err := s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&policyReadTx{Tx: tx, GovRepo: tx.(store.GovRepo), read: &read})
	})
	if err == nil && read && s.afterPolicyRead != nil {
		f := s.afterPolicyRead
		s.afterPolicyRead = nil
		f()
	}
	return err
}

type policyReadTx struct {
	store.Tx
	store.GovRepo
	read *bool
}

func (t *policyReadTx) ListPolicyVersions(net string) ([]store.PolicyVersionRow, error) {
	*t.read = true
	return t.GovRepo.ListPolicyVersions(net)
}
