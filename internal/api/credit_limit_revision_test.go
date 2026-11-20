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

// A2-07: equal effective amounts do not mean equal override configuration.
func TestCreditLimitETagTracksOverrideAndExpiry(t *testing.T) {
	w := newWorld(t)
	path := "/api/v1/admin/accounts/" + w.aliceM.ID
	put := "/api/v1/admin/accounts/" + w.aliceM.ID + "/credit-limit"
	current := w.do(req{method: "GET", path: path, token: w.adminT})
	wantStatus(t, current, 200)
	old := current.Header().Get("ETag")
	limit := int64(10000)
	expires := w.env.Clock.Now().Add(time.Hour)
	for _, expiry := range []*time.Time{nil, &expires} {
		testutil.Must(0, w.env.Svc.SetCreditLimit(w.env.Ctx, w.admin, w.net.ID, service.SetLimitInput{
			MemberID: w.aliceM.ID, Limit: &limit, ExpiresAt: expiry, Reason: "equivalent amount, different override",
		}))
		current = w.do(req{method: "GET", path: path, token: w.adminT})
		wantStatus(t, current, 200)
		if current.Header().Get("ETag") == old {
			t.Fatal("override configuration did not change ETag")
		}
		wantProblem(t, w.do(req{method: "PUT", path: put, token: w.adminT,
			body: m{"credit_limit_minor": nil, "reason": "stale override removal"}, hdr: map[string]string{"If-Match": old}}), 412, "precondition-failed")
		old = current.Header().Get("ETag")
	}
	w.env.Clock.Advance(time.Hour)
	current = w.do(req{method: "GET", path: path, token: w.adminT})
	wantStatus(t, current, 200)
	if current.Header().Get("ETag") == old {
		t.Fatal("expiry boundary did not change ETag when default equals override")
	}
	wantProblem(t, w.do(req{method: "PUT", path: put, token: w.adminT,
		body: m{"credit_limit_minor": "12000", "reason": "stale expired limit"}, hdr: map[string]string{"If-Match": old}}), 412, "precondition-failed")
}

// Inject a real service write between the API's peek and conditional write.
// This is deterministic and uses separate store transactions: process-local
// API mutexes cannot serialize a web/CLI/service caller (API-027).
func TestCreditLimitConditionalWriteRejectsInterveningServiceEdit(t *testing.T) {
	for _, wildcard := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit", true: "wildcard"}[wildcard], func(t *testing.T) {
			var wrapped *creditLimitRaceStore
			env := testutil.NewEnvWith(t, func(s store.Store) store.Store {
				wrapped = &creditLimitRaceStore{Store: s}
				return wrapped
			})
			w := &world{t: t, env: env, h: api.NewHandlerWithOptions(env.Svc, api.Options{Now: env.Clock.Now})}
			t.Cleanup(func() { api.CloseHandler(w.h) })
			w.admin = env.Person("admin")
			w.net, w.adminM = env.Network(w.admin, "race", nil)
			w.alice, w.aliceM = env.Member(w.admin, w.net, "alice")
			w.adminT = w.token(w.admin, domain.ScopeRead, domain.ScopeWrite)
			path := "/api/v1/admin/accounts/" + w.aliceM.ID
			current := w.do(req{method: "GET", path: path, token: w.adminT})
			wantStatus(t, current, 200)
			etag := current.Header().Get("ETag")
			if wildcard {
				etag = "*"
			}
			limit := int64(9000)
			intervened := false
			wrapped.afterBalanceRead = func() {
				testutil.Must(0, env.Svc.SetCreditLimit(env.Ctx, w.admin, w.net.ID,
					service.SetLimitInput{MemberID: w.aliceM.ID, Limit: &limit, Reason: "another administrator"}))
				intervened = true
			}
			result := w.do(req{method: "PUT", path: path + "/credit-limit", token: w.adminT,
				body: m{"credit_limit_minor": "7000", "reason": "conditional edit"}, hdr: map[string]string{"If-Match": etag}})
			if !intervened {
				t.Fatal("test did not place a service write between read and write")
			}
			want := int64(9000)
			if wildcard {
				wantStatus(t, result, 200)
				want = 7000
			} else {
				wantProblem(t, result, 412, "precondition-failed")
			}
			balance := testutil.Must(env.Svc.GetBalance(env.Ctx, w.alice, w.net.ID))
			if balance.CreditLimit != want {
				t.Fatalf("limit = %d, want %d", balance.CreditLimit, want)
			}
		})
	}
}

type creditLimitRaceStore struct {
	store.Store
	afterBalanceRead func()
}

func (s *creditLimitRaceStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if s.afterBalanceRead == nil {
		return s.Store.View(ctx, fn)
	}
	read := false
	err := s.Store.View(ctx, func(tx store.Tx) error {
		return fn(&creditLimitReadTx{Tx: tx, GovRepo: tx.(store.GovRepo), read: &read})
	})
	if err == nil && read && s.afterBalanceRead != nil {
		f := s.afterBalanceRead
		s.afterBalanceRead = nil
		f()
	}
	return err
}

type creditLimitReadTx struct {
	store.Tx
	store.GovRepo
	read *bool
}

func (t *creditLimitReadTx) GetAccountByMember(net, member string) (*domain.Account, error) {
	*t.read = true
	return t.Tx.GetAccountByMember(net, member)
}
