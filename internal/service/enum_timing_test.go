package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// BR-05 / SEC-005: sign-in and account recovery must not tell an existing
// account from an unknown one by the work they do. These tests compare the
// work deterministically (Argon2id evaluations with their parameters, and the
// store operations in order) instead of measuring wall-clock time.
// TestLoginTimingStatistical measures time too, behind an environment variable.

// argonLog records the Argon2id evaluations seen through service.ObserveArgon2.
type argonLog struct {
	mu    sync.Mutex
	calls []service.Argon2Params
}

func observeArgon(t *testing.T) *argonLog {
	t.Helper()
	l := &argonLog{}
	t.Cleanup(service.ObserveArgon2(func(p service.Argon2Params) {
		l.mu.Lock()
		l.calls = append(l.calls, p)
		l.mu.Unlock()
	}))
	return l
}

func (l *argonLog) take() []service.Argon2Params {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.calls
	l.calls = nil
	return out
}

// opStore records the units of work and the account-lookup, throttle and
// recovery-code statements a call makes, in order.
type opStore struct {
	store.Store
	mu  sync.Mutex
	ops []string
}

func (s *opStore) rec(op string) { s.mu.Lock(); s.ops = append(s.ops, op); s.mu.Unlock() }

func (s *opStore) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.ops
	s.ops = nil
	return out
}

func (s *opStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	s.rec("update")
	return s.Store.Update(ctx, func(tx store.Tx) error { return fn(&opTx{Tx: tx, s: s}) })
}

func (s *opStore) View(ctx context.Context, fn func(store.Tx) error) error {
	s.rec("view")
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(&opTx{Tx: tx, s: s}) })
}

type opTx struct {
	store.Tx
	s *opStore
}

func (t *opTx) GetPersonByLogin(login string) (*domain.Person, string, error) {
	p, h, err := t.Tx.GetPersonByLogin(login)
	t.s.rec(fmt.Sprintf("GetPersonByLogin err=%v", domain.CodeOf(err)))
	return p, h, err
}

func (t *opTx) GetThrottle(key string) (int, time.Time, bool, error) {
	t.s.rec("GetThrottle")
	return t.Tx.GetThrottle(key)
}

func (t *opTx) PutThrottle(key string, n int, until time.Time) error {
	t.s.rec("PutThrottle")
	return t.Tx.PutThrottle(key, n, until)
}

func (t *opTx) HasUnusedAccountRecoveryCode(personID, hash string) (bool, error) {
	t.s.rec("HasUnusedAccountRecoveryCode")
	return t.Tx.HasUnusedAccountRecoveryCode(personID, hash)
}

func newOpEnv(t *testing.T, mod func(*service.Config)) (*tu.Env, *opStore) {
	t.Helper()
	var ops *opStore
	e := tu.NewEnvConfig(t, func(s store.Store) store.Store { ops = &opStore{Store: s}; return ops }, mod)
	return e, ops
}

// testArgon is the cheap parameter set tu.NewEnv configures.
var testArgon = service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}

// Every refused sign-in costs exactly one Argon2id evaluation with the
// configured parameters and runs the same store statements, whether the
// login is unknown, malformed, has a wrong password, or belongs to an account
// whose stored hash cannot be evaluated (an erased person's marker, a damaged
// row). Before BR-05 the last case did no hashing at all.
func TestLoginRefusalsDoEqualWork(t *testing.T) {
	e, ops := newOpEnv(t, nil)
	e.Person("ana@example.org")
	bea := e.Person("bea@example.org")
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		return tx.SetPasswordHash(bea.PersonID, "!erased") // the marker AnonymiseMember stores
	}); err != nil {
		t.Fatal(err)
	}
	argon := observeArgon(t)
	ops.take()

	type probe struct{ name, login string }
	probes := []probe{
		{"unknown login", "nobody@example.org"},
		{"wrong password", "ana@example.org"},
		{"unusable stored hash", "bea@example.org"},
		{"malformed login", "a b"},
	}
	var wantOps []string
	for _, p := range probes {
		_, _, err := e.Svc.Login(e.Ctx, p.login, "wrong password!!")
		if !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("%s: got %v, want unauthenticated", p.name, err)
		}
		if got := argon.take(); !slices.Equal(got, []service.Argon2Params{testArgon}) {
			t.Errorf("%s: Argon2id evaluations %+v, want exactly one with %+v", p.name, got, testArgon)
		}
		got := ops.take()
		if p.name == "malformed login" {
			continue // refused before the store; no account can have such a login
		}
		// Only the lookup result may differ, and it never leaves the service.
		for i := range got {
			if got[i] == "GetPersonByLogin err=" {
				got[i] = "GetPersonByLogin err=not-found"
			}
		}
		if wantOps == nil {
			wantOps = got
		} else if !slices.Equal(got, wantOps) {
			t.Errorf("%s: store operations %v, want %v", p.name, got, wantOps)
		}
	}
	if !slices.Contains(wantOps, "GetPersonByLogin err=not-found") {
		t.Fatalf("store operations were not recorded: %v", wantOps)
	}
	// Control: the observer sees a real verification too.
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Fatal(err)
	}
	if got := argon.take(); !slices.Equal(got, []service.Argon2Params{testArgon}) {
		t.Errorf("successful sign-in: Argon2id evaluations %+v", got)
	}
}

// Self-service recovery refuses an unknown login, a known login with a wrong
// code, a known login without codes and a malformed code with the same store
// statements and without any Argon2id work (A3-04). Only an admitted, valid
// code hashes the new password.
func TestRecoveryRefusalsDoEqualWork(t *testing.T) {
	e, ops := newOpEnv(t, nil)
	ana := e.Person("ana@example.org")
	codes := codesFor(t, e, ana)
	e.Person("cy@example.org") // has no recovery codes
	argon := observeArgon(t)
	ops.take()

	probes := []struct{ name, login, code string }{
		{"unknown login", "nobody@example.org", "ABCD-EFGH-IJKL-MNOP"},
		{"wrong code", "ana@example.org", "ABCD-EFGH-IJKL-MNOP"},
		{"no codes", "cy@example.org", "ABCD-EFGH-IJKL-MNOP"},
		{"malformed code, unknown login", "nobody2@example.org", "not a code"},
		{"malformed code, known login", "cy@example.org", "not a code"},
	}
	var wantOps []string
	for _, p := range probes {
		err := e.Svc.RecoverPassword(e.Ctx, p.login, p.code, newPW)
		if !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("%s: got %v", p.name, err)
		}
		if got := argon.take(); len(got) != 0 {
			t.Errorf("%s: %d Argon2id evaluations, want none", p.name, len(got))
		}
		got := ops.take()
		for i := range got {
			if got[i] == "GetPersonByLogin err=" {
				got[i] = "GetPersonByLogin err=not-found"
			}
		}
		if wantOps == nil {
			wantOps = got
		} else if !slices.Equal(got, wantOps) {
			t.Errorf("%s: store operations %v, want %v", p.name, got, wantOps)
		}
	}
	if !slices.Contains(wantOps, "HasUnusedAccountRecoveryCode") {
		t.Fatalf("store operations were not recorded: %v", wantOps)
	}
	if err := e.Svc.RecoverPassword(e.Ctx, "ana@example.org", codes[0], newPW); err != nil {
		t.Fatal(err)
	}
	if got := argon.take(); !slices.Equal(got, []service.Argon2Params{testArgon}) {
		t.Errorf("valid code: Argon2id evaluations %+v, want one", got)
	}
}

// TestLoginTimingStatistical measures the median time of refused sign-ins for
// an existing and an unknown login with the default Argon2id parameters. It
// is slow and depends on the machine, so it runs only with
// CIRCULA_TIMING_TEST=1. The allowed difference, in percent of the slower
// median, is CIRCULA_TIMING_MAX_PCT (default 10). The release threshold is
// pending owner decision DEC-023 (QA-040); 10 is a provisional value.
func TestLoginTimingStatistical(t *testing.T) {
	if os.Getenv("CIRCULA_TIMING_TEST") != "1" {
		t.Skip("set CIRCULA_TIMING_TEST=1 to measure sign-in timing (BR-05)")
	}
	maxPct := 10.0
	if v := os.Getenv("CIRCULA_TIMING_MAX_PCT"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("CIRCULA_TIMING_MAX_PCT: %v", err)
		}
		maxPct = f
	}
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.Argon = service.DefaultArgon2Params() })
	const rounds = 25
	logins := make([]string, rounds)
	for i := range logins {
		logins[i] = fmt.Sprintf("member%02d@example.org", i)
		e.Person(logins[i]) // a fresh login per round keeps both sides below the throttle
	}
	timeOne := func(login string) time.Duration {
		start := time.Now()
		_, _, err := e.Svc.Login(e.Ctx, login, "wrong password!!")
		d := time.Since(start)
		if !errors.Is(err, domain.ErrUnauthenticated) {
			t.Fatalf("%s: %v", login, err)
		}
		return d
	}
	var known, unknown []time.Duration
	for i := range rounds { // interleaved so drift affects both sides alike
		if i%2 == 0 {
			known = append(known, timeOne(logins[i]))
			unknown = append(unknown, timeOne(fmt.Sprintf("ghost%02d@example.org", i)))
		} else {
			unknown = append(unknown, timeOne(fmt.Sprintf("ghost%02d@example.org", i)))
			known = append(known, timeOne(logins[i]))
		}
	}
	median := func(ds []time.Duration) time.Duration {
		s := slices.Clone(ds)
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		return s[len(s)/2]
	}
	mk, mu := median(known), median(unknown)
	diff := float64(mk - mu)
	if diff < 0 {
		diff = -diff
	}
	pct := 100 * diff / float64(max(mk, mu))
	t.Logf("median refused sign-in: existing %v, unknown %v, difference %.1f%% (limit %.1f%%)", mk, mu, pct, maxPct)
	if pct > maxPct {
		t.Errorf("timing differs by %.1f%%, more than %.1f%%", pct, maxPct)
	}
}
