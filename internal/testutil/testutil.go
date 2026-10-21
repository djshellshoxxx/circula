// Package testutil builds deterministic service environments for tests.
package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Clock is a settable fake clock.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

// Now implements domain.Clock.
func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

// Advance moves the clock forward.
func (c *Clock) Advance(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// IDs generates sortable, deterministic UUID-shaped ids.
type IDs struct{ n atomic.Int64 }

// New implements domain.IDGen.
func (g *IDs) New() string {
	return fmt.Sprintf("018f0000-0000-7000-8000-%012x", g.n.Add(1))
}

// Env bundles a service over a fresh SQLite file.
type Env struct {
	T     *testing.T
	Ctx   context.Context
	Store *store.SQLite
	Svc   *service.Service
	Clock *Clock
	keys  atomic.Int64
}

// NewEnv creates an Env; the store is closed at test end. A custom store
// wrapper may be installed with NewEnvWith.
func NewEnv(t *testing.T) *Env { return NewEnvWith(t, nil) }

// NewEnvWith lets the caller wrap the store (fault injection).
func NewEnvWith(t *testing.T, wrap func(store.Store) store.Store) *Env {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "circula.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := &Clock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	var s store.Store = st
	if wrap != nil {
		s = wrap(st)
	}
	svc, err := service.New(s, service.Config{Clock: clk, IDs: &IDs{},
		Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	return &Env{T: t, Ctx: ctx, Store: st, Svc: svc, Clock: clk}
}

// Key returns a fresh idempotency key.
func (e *Env) Key() string { return fmt.Sprintf("key-%d", e.keys.Add(1)) }

// Must returns v or panics with err. A panic fails the test with a stack
// trace at the failing setup call, which keeps call sites to one line.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("testutil.Must: unexpected error: %v", err))
	}
	return v
}

// Person registers a person and returns a full-scope actor.
func (e *Env) Person(login string) domain.Actor {
	e.T.Helper()
	p := Must(e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: login, Password: "correct horse battery"}))
	return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
}

// Network creates and activates a network founded by admin. mod may adjust the policy.
func (e *Env) Network(admin domain.Actor, slug string, mod func(*domain.Policy)) (*domain.Network, *domain.Member) {
	e.T.Helper()
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit = 10000 // 100.00
	if mod != nil {
		mod(&pol)
	}
	n, m := e.createNetwork(admin, slug, &pol)
	n = Must(e.Svc.ActivateNetwork(e.Ctx, admin, n.ID))
	return n, m
}

func (e *Env) createNetwork(admin domain.Actor, slug string, pol *domain.Policy) (*domain.Network, *domain.Member) {
	n, m, err := e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "Net " + slug, Slug: slug,
		Policy: pol, CreatorDisplayName: "Founder"})
	if err != nil {
		e.T.Fatal(err)
	}
	return n, m
}

// Member creates a person and joins them to net by invitation.
func (e *Env) Member(admin domain.Actor, net *domain.Network, login string) (domain.Actor, *domain.Member) {
	e.T.Helper()
	a := e.Person(login)
	tok, _ := Must2(e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{}))
	m := Must(e.Svc.RedeemInvitation(e.Ctx, a, tok, login))
	return a, m
}

// Must2 unwraps (a, b, err).
func Must2[A, B any](a A, b B, err error) (A, B) {
	if err != nil {
		panic(fmt.Sprintf("testutil.Must2: unexpected error: %v", err))
	}
	return a, b
}

// Trade proposes (payer pays payee) and accepts, returning the completed tx.
func (e *Env) Trade(netID string, payer, payee domain.Actor, payerM, payeeM *domain.Member, amount int64) *domain.Transaction {
	e.T.Helper()
	t := Must(e.Svc.ProposeTransaction(e.Ctx, payer, netID, service.ProposeInput{CounterpartyID: payeeM.ID,
		Direction: service.ProposerPays, Amount: amount, IdempotencyKey: e.Key()}))
	return Must(e.Svc.AcceptTransaction(e.Ctx, payee, netID, service.TxRef{TxID: t.ID, IdempotencyKey: e.Key()}))
}
