package service_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-001 (two networks): random operation sequences, including deliberate
// cross-network misuse, keep LED-1/2/3/5 true in both networks after every step,
// never leak an internal error, and keep VerifyLedger green.
func TestPropTwoNetworkLedger(t *testing.T) {
	seeds, steps := 4, 150
	if raceOn {
		seeds, steps = 2, 50
	}
	for seed := int64(1); seed <= int64(seeds); seed++ {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) { runTwoNet(t, seed+1000, steps) })
	}
}

func runTwoNet(t *testing.T, seed int64, steps int) {
	rng := rand.New(rand.NewSource(seed))
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 4000; p.ProposalExpiryDays = 2 })
	admin2 := w.Person("admin2@example.org")
	net2, _ := w.Network(admin2, "other", func(p *domain.Policy) { p.DefaultCreditLimit = 2500; p.ProposalExpiryDays = 2 })

	type party struct {
		actor domain.Actor
		m     *domain.Member
		net   *domain.Network
		admin domain.Actor
	}
	ps := []party{{w.a, w.aM, w.net, w.admin}, {w.b, w.bM, w.net, w.admin}}
	for i := 0; i < 3; i++ {
		a, m := w.Member(w.admin, w.net, fmt.Sprintf("a%d@example.org", i))
		ps = append(ps, party{a, m, w.net, w.admin})
	}
	for i := 0; i < 4; i++ {
		a, m := w.Member(admin2, net2, fmt.Sprintf("b%d@example.org", i))
		ps = append(ps, party{a, m, net2, admin2})
	}
	nets := []*domain.Network{w.net, net2}
	limits := map[string]int64{w.net.ID: 4000, net2.ID: 2500}
	fault := func(err error, what string) {
		if err != nil && domain.CodeOf(err) == domain.CodeInternal {
			t.Fatalf("%s: internal error: %v", what, err)
		}
	}
	pick := func() party { return ps[rng.Intn(len(ps))] }
	sameNet := func(p party) party {
		for {
			q := pick()
			if q.net.ID == p.net.ID {
				return q
			}
		}
	}
	actorOf := map[string]domain.Actor{}
	for _, p := range ps {
		actorOf[p.m.ID] = p.actor
	}
	for step := 0; step < steps; step++ {
		before := balances(t, w, limits)
		p := pick()
		what := fmt.Sprintf("seed%d/step%d", seed, step)
		switch rng.Intn(9) {
		case 0, 1, 2:
			q := sameNet(p)
			dir := service.ProposerPays
			if rng.Intn(2) == 0 {
				dir = service.ProposerIsPaid
			}
			_, err := w.Svc.ProposeTransaction(w.Ctx, p.actor, p.net.ID, service.ProposeInput{CounterpartyID: q.m.ID, Direction: dir, Amount: 1 + rng.Int63n(2500), IdempotencyKey: w.Key()})
			fault(err, what)
		case 3, 4:
			for _, n := range nets {
				l, err := w.Svc.AdminListTransactions(w.Ctx, map[string]domain.Actor{w.net.ID: w.admin, net2.ID: admin2}[n.ID], n.ID, "", []domain.TxStatus{domain.TxProposed}, service.Page{Limit: 50})
				fault(err, what)
				if len(l) == 0 {
					continue
				}
				tx := l[rng.Intn(len(l))]
				cp := tx.PayeeMemberID
				if tx.CreatedBy == tx.PayeeMemberID {
					cp = tx.PayerMemberID
				}
				_, err = w.Svc.AcceptTransaction(w.Ctx, actorOf[cp], n.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: w.Key()})
				fault(err, what)
				break
			}
		case 5: // cross-network misuse must be refused, never post
			q := pick()
			if q.net.ID != p.net.ID {
				_, err := w.Svc.ProposeTransaction(w.Ctx, p.actor, p.net.ID, service.ProposeInput{CounterpartyID: q.m.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.Key()})
				if err == nil {
					t.Fatalf("%s: proposal across networks accepted", what)
				}
				fault(err, what)
				_, err = w.Svc.ProposeTransaction(w.Ctx, p.actor, q.net.ID, service.ProposeInput{CounterpartyID: q.m.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.Key()})
				if err == nil {
					t.Fatalf("%s: non-member proposed in a foreign network", what)
				}
				fault(err, what)
			}
		case 6:
			w.Clock.Advance(time.Duration(rng.Int63n(int64(60 * time.Hour))))
			_, err := w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("sweep"), 100)
			fault(err, what)
		case 7:
			l, err := w.Svc.AdminListTransactions(w.Ctx, p.admin, p.net.ID, "", []domain.TxStatus{domain.TxCompleted}, service.Page{Limit: 50})
			fault(err, what)
			if len(l) > 0 {
				tx := l[rng.Intn(len(l))]
				if tx.Kind != domain.KindReversal {
					_, err = w.Svc.ReverseTransaction(w.Ctx, p.admin, p.net.ID, service.ReverseInput{TxID: tx.ID, Amount: 1 + rng.Int63n(tx.Amount), Reason: "prop", IdempotencyKey: w.Key()})
					fault(err, what)
				}
			}
		case 8:
			q := sameNet(p)
			dir := service.AdjustCredit
			if rng.Intn(2) == 0 {
				dir = service.AdjustDebit
			}
			_, err := w.Svc.CreateAdjustment(w.Ctx, p.admin, p.net.ID, service.AdjustInput{MemberID: q.m.ID, Amount: 1 + rng.Int63n(3000), Direction: dir, Reason: "prop", IdempotencyKey: w.Key()})
			fault(err, what)
		}
		checkInvariants(t, w, what)
		// LED-5: a balance that went down never ends below its network's limit
		for id, a := range balances(t, w, limits) {
			if b, ok := before[id]; ok && a[0] < b[0] && a[0] < -a[1] {
				t.Fatalf("%s: LED-5 breached on %s: %d -> %d, limit %d", what, id, b[0], a[0], a[1])
			}
		}
		// per network: sum of balances is zero; trades never push a non-system account below its limit
		for _, n := range nets {
			var sum int64
			if err := w.Store.DB().QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts WHERE network_id = ?`, n.ID).Scan(&sum); err != nil || sum != 0 {
				t.Fatalf("%s: network %s sum %d (%v)", what, n.Slug, sum, err)
			}
		}
		if step%25 == 24 {
			for _, n := range nets {
				rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, map[string]domain.Actor{w.net.ID: w.admin, net2.ID: admin2}[n.ID], n.ID))
				if !rep.OK {
					t.Fatalf("%s: verify %s: %v", what, n.Slug, rep.Findings)
				}
			}
		}
	}
}

// balances reads every non-system account with raw SQL, with its network's default limit.
func balances(t *testing.T, w *world, limits map[string]int64) map[string][2]int64 {
	t.Helper()
	rows, err := w.Store.DB().Query(`SELECT id, network_id, balance FROM accounts WHERE is_system = 0`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string][2]int64{}
	for rows.Next() {
		var id, net string
		var bal int64
		if err := rows.Scan(&id, &net, &bal); err != nil {
			t.Fatal(err)
		}
		out[id] = [2]int64{bal, limits[net]}
	}
	return out
}
