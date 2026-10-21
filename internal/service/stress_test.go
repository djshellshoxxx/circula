package service_test

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// stressScale shrinks workloads under the race detector (pure-Go sqlite is slow there).
func stressScale(n int) int {
	if raceOn {
		return n / 3
	}
	return n
}

// noServerFault fails the test for errors that mean the service misbehaved:
// internal errors, non-domain errors, and busy/unavailable surfacing from
// in-process contention (QA-011: in-process writers queue, they do not fail).
func noServerFault(t *testing.T, what string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	switch c := domain.CodeOf(err); c {
	case domain.CodeInternal, domain.CodeUnavailable:
		t.Errorf("%s: server fault %q: %v", what, c, err)
	}
}

// sumInvariants checks LED-1/LED-3 database-wide with queries independent of the service.
func sumInvariants(t *testing.T, w *world, label string) {
	t.Helper()
	checkInvariants(t, w, label)
	var sum int64
	if err := w.Store.DB().QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts`).Scan(&sum); err != nil || sum != 0 {
		t.Fatalf("%s: global balance sum = %d (%v)", label, sum, err)
	}
}

// QA-007/QA-008: 100 disjoint pairs trade at once; all post, none deadlock, sum is zero.
func TestStressDisjointTradesNoDeadlock(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100000 })
	pairs := stressScale(100)
	type pair struct {
		pa, pb domain.Actor
		mb     *domain.Member
	}
	ps := make([]pair, pairs)
	for i := range ps {
		a, _ := w.Member(w.admin, w.net, fmt.Sprintf("p%da@example.org", i))
		b, bm := w.Member(w.admin, w.net, fmt.Sprintf("p%db@example.org", i))
		ps[i] = pair{a, b, bm}
	}
	done := make(chan struct{})
	var ok atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, p := range ps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			amt := int64(100 + i)
			tx, err := w.Svc.ProposeTransaction(w.Ctx, p.pa, w.net.ID, service.ProposeInput{CounterpartyID: p.mb.ID, Direction: service.ProposerPays, Amount: amt, IdempotencyKey: fmt.Sprintf("pr-%d", i)})
			noServerFault(t, "propose", err)
			if err != nil {
				return
			}
			_, err = w.Svc.AcceptTransaction(w.Ctx, p.pb, w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: fmt.Sprintf("ac-%d", i)})
			noServerFault(t, "accept", err)
			if err == nil {
				ok.Add(1)
			}
		}()
	}
	go func() { wg.Wait(); close(done) }()
	close(start)
	select {
	case <-done:
	case <-time.After(120 * time.Second):
		t.Fatal("deadlock or starvation: disjoint trades did not finish in 120s")
	}
	if int(ok.Load()) != pairs {
		t.Fatalf("%d of %d trades posted", ok.Load(), pairs)
	}
	sumInvariants(t, w, "disjoint")
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("verify: %v", rep.Findings)
	}
}

// QA-007: many racing partial reversals of one trade never exceed the original (DOM-019).
func TestStressParallelReversalsRespectCap(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1000000 })
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10000)
	const n, each = 20, int64(700) // 20*700 = 14000 > 10000, so only 14 fit
	var ok atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: tx.ID, Amount: each, Reason: "race", IdempotencyKey: fmt.Sprintf("rv-%d", i)})
			noServerFault(t, "reverse", err)
			if err == nil {
				ok.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	got := tu.Must(w.Svc.GetTransaction(w.Ctx, w.admin, w.net.ID, tx.ID)).Tx
	if got.ReversedTotal > 10000 || got.ReversedTotal != int64(ok.Load())*each {
		t.Fatalf("reversed_total=%d with %d successes of %d", got.ReversedTotal, ok.Load(), each)
	}
	if ok.Load() != 14 {
		t.Errorf("%d reversals fit, want 14", ok.Load())
	}
	if bal := w.balance(w.a).Balance; bal != -(10000 - got.ReversedTotal) {
		t.Errorf("payer balance %d inconsistent with reversed %d", bal, got.ReversedTotal)
	}
	sumInvariants(t, w, "reversals")
}

// A dispute decided twice at once posts at most one reversal.
func TestStressParallelDisputeResolve(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1000000 })
	rev, _ := w.reviewer()
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 5000)
	d := tu.Must(w.open(w.a, tx.ID))
	var ok atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := w.Svc.ResolveDispute(w.Ctx, rev, w.net.ID, service.ResolveInput{DisputeID: d.ID, Outcome: domain.OutcomeUpheld, Rationale: "r", IdempotencyKey: fmt.Sprintf("res-%d", i)})
			noServerFault(t, "resolve", err)
			if err == nil {
				ok.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if ok.Load() != 1 {
		t.Errorf("%d resolutions succeeded, want 1", ok.Load())
	}
	var n int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM transactions WHERE kind = 'reversal'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("%d reversals posted (%v), want 1", n, err)
	}
	if w.balance(w.a).Balance != 0 {
		t.Errorf("payer balance %d, want 0 after one full reversal", w.balance(w.a).Balance)
	}
	sumInvariants(t, w, "dispute")
}

// SEC-010/011: many pending adjustments approved concurrently by two administrators
// each post exactly once; the proposer can never approve its own.
func TestStressConcurrentApprovals(t *testing.T) {
	d := newDC(t)
	third, thirdM := d.Member(d.admin, d.net, "admin3@example.org")
	tu.Must(0, d.Svc.AssignRole(d.Ctx, d.admin, d.net.ID, thirdM.ID, domain.RoleAdmin))
	const n = 12
	pend := make([]*domain.PendingAdjustment, n)
	for i := range pend {
		pend[i] = d.pending(2000)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	var selfOK atomic.Int32
	for _, p := range pend {
		for _, who := range []domain.Actor{d.admin2, third, d.admin2, third, d.admin} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, err := d.Svc.ApproveAdjustment(d.Ctx, who, d.net.ID, p.ID)
				noServerFault(t, "approve", err)
				if err == nil && who.PersonID == d.admin.PersonID {
					selfOK.Add(1)
				}
			}()
		}
	}
	close(start)
	wg.Wait()
	if selfOK.Load() != 0 {
		t.Errorf("proposer approved its own request %d times", selfOK.Load())
	}
	if got := d.balance(d.a).Balance; got != n*2000 {
		t.Fatalf("balance %d, want %d (each request posted exactly once)", got, n*2000)
	}
	if c := d.audits(domain.ActAdjustment); c != n {
		t.Errorf("%d adjustment audit events, want %d", c, n)
	}
	sumInvariants(t, d.world, "approvals")
}

// Racing registrations of one login: exactly one wins, the rest see a conflict-class error, no faults.
func TestStressConcurrentRegisterAndRedeem(t *testing.T) {
	e := tu.NewEnv(t)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: "same@example.org", Password: "correct horse battery"})
			noServerFault(t, "register", err)
		}()
	}
	wg.Wait()
	// registration answers uniformly (no account enumeration), so every call may
	// report success; what matters is that exactly one row exists.
	var rows int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM persons WHERE login = 'same@example.org'`).Scan(&rows); err != nil {
		t.Fatal("persons count query:", err)
	}
	if rows != 1 {
		t.Errorf("%d person rows for one login", rows)
	}

	// single-use invitation redeemed by many people at once
	admin := e.Person("adm@example.org")
	net, _ := e.Network(admin, "inv", nil)
	tok, _ := tu.Must2(e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{}))
	actors := make([]domain.Actor, 8)
	for i := range actors {
		actors[i] = e.Person(fmt.Sprintf("r%d@example.org", i))
	}
	var joined atomic.Int32
	start := make(chan struct{})
	for i := range actors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := e.Svc.RedeemInvitation(e.Ctx, actors[i], tok, fmt.Sprintf("R%d", i))
			noServerFault(t, "redeem", err)
			if err == nil {
				joined.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if joined.Load() != 1 {
		t.Errorf("%d people redeemed a single-use invitation", joined.Load())
	}
}

// Concurrent mixed workload across two networks. After the dust settles every
// ledger invariant holds, VerifyLedger passes per network, and no worker saw a
// server fault (internal error, SQLITE_BUSY surfaced as unavailable).
func TestStressMixedWorkloadTwoNetworks(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 20000; p.ProposalExpiryDays = 2 })
	admin2 := w.Person("admin-two@example.org")
	net2, _ := w.Network(admin2, "second", func(p *domain.Policy) { p.DefaultCreditLimit = 20000 })

	type party struct {
		actor domain.Actor
		m     *domain.Member
		net   string
	}
	var parties []party
	parties = append(parties, party{w.a, w.aM, w.net.ID}, party{w.b, w.bM, w.net.ID})
	for i := 0; i < 3; i++ {
		a, m := w.Member(w.admin, w.net, fmt.Sprintf("n1m%d@example.org", i))
		parties = append(parties, party{a, m, w.net.ID})
	}
	for i := 0; i < 4; i++ {
		a, m := w.Member(admin2, net2, fmt.Sprintf("n2m%d@example.org", i))
		parties = append(parties, party{a, m, net2.ID})
	}
	byNet := map[string][]party{}
	for _, p := range parties {
		byNet[p.net] = append(byNet[p.net], p)
	}
	adminOf := map[string]domain.Actor{w.net.ID: w.admin, net2.ID: admin2}
	actorOf := map[string]domain.Actor{}
	for _, p := range parties {
		actorOf[p.m.ID] = p.actor
	}

	workers, ops := 8, stressScale(120)
	var wg sync.WaitGroup
	var key atomic.Int64
	k := func() string { return fmt.Sprintf("mx-%d", key.Add(1)) }
	var posted atomic.Int32
	for wi := 0; wi < workers; wi++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(wi) + 100))
			for i := 0; i < ops; i++ {
				me := parties[rng.Intn(len(parties))]
				peers := byNet[me.net]
				other := peers[rng.Intn(len(peers))]
				what := fmt.Sprintf("w%d/op%d", wi, i)
				switch rng.Intn(7) {
				case 0, 1, 2:
					dir := service.ProposerPays
					if rng.Intn(2) == 0 {
						dir = service.ProposerIsPaid
					}
					_, err := w.Svc.ProposeTransaction(w.Ctx, me.actor, me.net, service.ProposeInput{CounterpartyID: other.m.ID, Direction: dir, Amount: 1 + rng.Int63n(9000), IdempotencyKey: k()})
					noServerFault(t, what+" propose", err)
				case 3, 4:
					l, err := w.Svc.AdminListTransactions(w.Ctx, adminOf[me.net], me.net, "", []domain.TxStatus{domain.TxProposed}, service.Page{Limit: 50})
					noServerFault(t, what+" list", err)
					if len(l) == 0 {
						continue
					}
					tx := l[rng.Intn(len(l))]
					cp := tx.PayeeMemberID
					if tx.CreatedBy == tx.PayeeMemberID {
						cp = tx.PayerMemberID
					}
					switch rng.Intn(4) {
					case 0:
						_, err = w.Svc.DeclineTransaction(w.Ctx, actorOf[cp], me.net, service.TxRef{TxID: tx.ID, IdempotencyKey: k()})
					case 1:
						_, err = w.Svc.CancelTransaction(w.Ctx, actorOf[tx.CreatedBy], me.net, service.CancelInput{TxID: tx.ID, IdempotencyKey: k()})
					default:
						_, err = w.Svc.AcceptTransaction(w.Ctx, actorOf[cp], me.net, service.TxRef{TxID: tx.ID, IdempotencyKey: k()})
						if err == nil {
							posted.Add(1)
						}
					}
					noServerFault(t, what+" decide", err)
				case 5:
					l, err := w.Svc.AdminListTransactions(w.Ctx, adminOf[me.net], me.net, "", []domain.TxStatus{domain.TxCompleted}, service.Page{Limit: 50})
					noServerFault(t, what+" list", err)
					if len(l) == 0 {
						continue
					}
					tx := l[rng.Intn(len(l))]
					if tx.Kind == domain.KindReversal {
						continue
					}
					_, err = w.Svc.ReverseTransaction(w.Ctx, adminOf[me.net], me.net, service.ReverseInput{TxID: tx.ID, Amount: 1 + rng.Int63n(tx.Amount), Reason: "stress", IdempotencyKey: k()})
					noServerFault(t, what+" reverse", err)
				case 6:
					dir := service.AdjustCredit
					if rng.Intn(2) == 0 {
						dir = service.AdjustDebit
					}
					_, err := w.Svc.CreateAdjustment(w.Ctx, adminOf[me.net], me.net, service.AdjustInput{MemberID: other.m.ID, Amount: 1 + rng.Int63n(3000), Direction: dir, Reason: "stress", IdempotencyKey: k()})
					noServerFault(t, what+" adjust", err)
				}
				if rng.Intn(10) == 0 { // concurrent reads and verification must not disturb writers
					_, err := w.Svc.VerifyLedger(w.Ctx, adminOf[me.net], me.net)
					noServerFault(t, what+" verify", err)
				}
			}
		}()
	}
	wg.Wait()
	if posted.Load() == 0 {
		t.Error("workload never posted a trade; generator is too weak")
	}
	sumInvariants(t, w, "mixed")
	for _, id := range []string{w.net.ID, net2.ID} {
		rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, adminOf[id], id))
		if !rep.OK {
			t.Errorf("verify %s: %v", id, rep.Findings)
		}
		var sum int64
		if err := w.Store.DB().QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts WHERE network_id = ?`, id).Scan(&sum); err != nil || sum != 0 {
			t.Errorf("network %s balance sum %d (%v)", id, sum, err)
		}
	}
	// no account breached its limit (LED-5), checked against the raw rows
	for id, a := range snapshot(t, w) {
		if !a.system && a.bal < -a.limit {
			// a limit may only have been reduced by admin; none is in this workload
			t.Errorf("account %s balance %d below limit %d", id, a.bal, a.limit)
		}
	}
	t.Logf("mixed workload: %d accepted trades", posted.Load())
}
