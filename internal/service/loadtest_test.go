package service_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// TestLoadConcurrentTrades is a throughput and latency probe, not a pass/fail
// performance gate. It is skipped unless CIRCULA_LOAD=1. Tunables (env):
//
//	CIRCULA_LOAD_WORKERS  concurrent goroutines          (default 16)
//	CIRCULA_LOAD_MEMBERS  members in the network         (default 50)
//	CIRCULA_LOAD_TRADES   trades attempted per worker     (default 250)
//
// Each trade is a propose by the payer followed by an accept by the payee, each
// its own committed transaction, on a real SQLite file (WAL). Pass/fail is only
// correctness: no unexpected errors, ledger verification clean, member balances
// sum to zero, and the number of completed trades matches the ledger.
//
//	CIRCULA_LOAD=1 go test ./internal/service -run LoadConcurrentTrades -v -count=1
func TestLoadConcurrentTrades(t *testing.T) {
	if os.Getenv("CIRCULA_LOAD") != "1" {
		t.Skip("set CIRCULA_LOAD=1 to run the load test")
	}
	workers := envInt("CIRCULA_LOAD_WORKERS", 16)
	members := envInt("CIRCULA_LOAD_MEMBERS", 50)
	trades := envInt("CIRCULA_LOAD_TRADES", 250)

	env := tu.NewEnv(t)
	admin := env.Person("admin")
	net, _ := env.Network(admin, "load", nil)
	type mem struct {
		a domain.Actor
		m *domain.Member
	}
	ms := make([]mem, members)
	for i := range ms {
		a, m := env.Member(admin, net, fmt.Sprintf("member%03d", i))
		ms[i] = mem{a, m}
	}

	type sample struct{ propose, accept time.Duration }
	var (
		mu           sync.Mutex
		samples      []sample
		completed    atomic.Int64
		refused      atomic.Int64 // insufficient credit: a business outcome, not a fault
		unavailable  atomic.Int64 // 503-class errors seen (each is retried)
		retried      atomic.Int64 // retries performed
		gaveUp       atomic.Int64 // operations that exhausted retries
		unexpected   atomic.Int64
		unexpectedMu sync.Mutex
		firstUnexp   error
	)
	// attempt runs fn, retrying only retryable (unavailable) errors with the
	// same idempotency key, as a client would.
	attempt := func(fn func() error) error {
		const maxRetries = 20
		for try := 0; ; try++ {
			err := fn()
			if err == nil || !domain.IsRetryable(err) {
				return err
			}
			unavailable.Add(1)
			if try >= maxRetries {
				gaveUp.Add(1)
				return err
			}
			retried.Add(1)
			time.Sleep(time.Duration(try+1) * time.Millisecond)
		}
	}
	noteUnexpected := func(err error) {
		unexpected.Add(1)
		unexpectedMu.Lock()
		if firstUnexp == nil {
			firstUnexp = err
		}
		unexpectedMu.Unlock()
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(w)+1, 42))
			local := make([]sample, 0, trades)
			for i := 0; i < trades; i++ {
				pi := rng.IntN(members)
				qi := (pi + 1 + rng.IntN(members-1)) % members
				payer, payee := ms[pi], ms[qi]
				amt := int64(1 + rng.IntN(300))
				pk, ak := fmt.Sprintf("w%d-t%d-p", w, i), fmt.Sprintf("w%d-t%d-a", w, i)

				var tx *domain.Transaction
				t0 := time.Now()
				err := attempt(func() (err error) {
					tx, err = env.Svc.ProposeTransaction(ctx, payer.a, net.ID, service.ProposeInput{
						CounterpartyID: payee.m.ID, Direction: service.ProposerPays, Amount: amt, IdempotencyKey: pk})
					return err
				})
				dp := time.Since(t0)
				if errors.Is(err, domain.ErrInsufficientCredit) {
					refused.Add(1)
					continue
				}
				if err != nil {
					noteUnexpected(fmt.Errorf("propose: %w", err))
					continue
				}
				t1 := time.Now()
				err = attempt(func() error {
					_, err := env.Svc.AcceptTransaction(ctx, payee.a, net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: ak})
					return err
				})
				da := time.Since(t1)
				if errors.Is(err, domain.ErrInsufficientCredit) {
					refused.Add(1)
					continue
				}
				if err != nil {
					noteUnexpected(fmt.Errorf("accept: %w", err))
					continue
				}
				completed.Add(1)
				local = append(local, sample{dp, da})
			}
			mu.Lock()
			samples = append(samples, local...)
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	if n := unexpected.Load(); n != 0 {
		t.Errorf("%d unexpected errors, first: %v", n, firstUnexp)
	}

	// Ledger invariants after the run.
	var sum int64
	for _, x := range ms {
		b := tu.Must(env.Svc.GetBalance(ctx, x.a, net.ID))
		if b.Balance < -b.CreditLimit {
			t.Errorf("member %s balance %d below -limit %d", x.m.ID, b.Balance, b.CreditLimit)
		}
		sum += b.Balance
	}
	if sum != 0 {
		t.Errorf("member balances sum to %d, want 0", sum)
	}
	rep := tu.Must(env.Svc.VerifyLedger(ctx, admin, net.ID))
	if !rep.OK {
		t.Errorf("VerifyLedger findings: %v", rep.Findings)
	}
	if len(samples) != int(completed.Load()) {
		t.Errorf("samples %d != completed %d", len(samples), completed.Load())
	}
	pct := func(v []time.Duration, p float64) time.Duration {
		if len(v) == 0 {
			return 0
		}
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
		return v[min(len(v)-1, int(float64(len(v))*p))]
	}
	var props, accs, both []time.Duration
	for _, s := range samples {
		props = append(props, s.propose)
		accs = append(accs, s.accept)
		both = append(both, s.propose+s.accept)
	}
	ops := 2 * completed.Load()
	t.Logf("CONFIG workers=%d members=%d trades_per_worker=%d attempted=%d", workers, members, trades, workers*trades)
	t.Logf("RESULT elapsed=%.2fs completed_trades=%d refused_insufficient_credit=%d committed_write_ops=%d", elapsed.Seconds(), completed.Load(), refused.Load(), ops)
	t.Logf("RESULT throughput trades/s=%.1f write_ops/s=%.1f", float64(completed.Load())/elapsed.Seconds(), float64(ops)/elapsed.Seconds())
	t.Logf("RESULT unavailable_errors=%d retries=%d gave_up=%d unexpected_errors=%d", unavailable.Load(), retried.Load(), gaveUp.Load(), unexpected.Load())
	for _, row := range []struct {
		name string
		v    []time.Duration
	}{{"propose", props}, {"accept", accs}, {"trade(propose+accept)", both}} {
		t.Logf("RESULT latency %-22s p50=%v p95=%v p99=%v max=%v", row.name, pct(row.v, 0.50), pct(row.v, 0.95), pct(row.v, 0.99), pct(row.v, 1))
	}
	t.Logf("RESULT ledger_ok=%v sum_of_member_balances=%d", rep.OK, sum)
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
