package service_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-008: 100 concurrent trades among disjoint member pairs in one network
// neither deadlock nor leave the network unbalanced. The timeout is the
// DEC-023 recommended default (60 s, generous: the single-writer gate
// serialises commits, so the run is expected to take well under a second).
func TestQA008DisjointConcurrentTrades(t *testing.T) {
	const pairs, amt = 100, int64(700)
	const timeout = 60 * time.Second
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	type pair struct {
		payer, payee domain.Actor
		payerM       *domain.Member
		payeeM       *domain.Member
	}
	ps := make([]pair, pairs)
	for i := range ps {
		a, am := w.Env.Member(w.admin, w.net, fmt.Sprintf("p%03d@example.org", i))
		b, bm := w.Env.Member(w.admin, w.net, fmt.Sprintf("q%03d@example.org", i))
		ps[i] = pair{a, b, am, bm}
	}
	done := make(chan struct{})
	var ok atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, p := range ps {
		wg.Add(1)
		go func(p pair) {
			defer wg.Done()
			<-start
			tx, err := w.propose(p.payer, p.payeeM, service.ProposerPays, amt)
			if err != nil {
				t.Errorf("propose: %v", err)
				return
			}
			if _, err := w.accept(p.payee, tx.ID); err != nil {
				t.Errorf("accept: %v", err)
				return
			}
			ok.Add(1)
		}(p)
	}
	go func() { wg.Wait(); close(done) }()
	begin := time.Now()
	close(start)
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("deadlock suspected: %d of %d trades finished in %s", ok.Load(), pairs, timeout)
	}
	t.Logf("%d disjoint trades in %s", ok.Load(), time.Since(begin))
	if ok.Load() != pairs {
		t.Fatalf("completed %d of %d", ok.Load(), pairs)
	}
	var sum int64
	for _, p := range ps {
		pb, qb := w.balance(p.payer).Balance, w.balance(p.payee).Balance
		if pb != -amt || qb != amt {
			t.Fatalf("balances %d/%d, want %d/%d", pb, qb, -amt, amt)
		}
		sum += pb + qb
	}
	if sum != 0 {
		t.Errorf("sum of balances = %d, want 0", sum)
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("ledger: %v", rep.Findings)
	}
}
