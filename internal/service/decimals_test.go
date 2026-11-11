package service_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// decimalsRefused reports whether err is the field-level refusal of a scale change.
func decimalsRefused(err error) bool {
	var de *domain.Error
	return errors.As(err, &de) && de.Code == domain.CodeValidation && len(de.Fields) == 1 && de.Fields[0].Field == "decimals"
}

// storedScales returns the scale of every stored policy version of a network, oldest first.
func storedScales(t *testing.T, e *tu.Env, netID string) []int {
	t.Helper()
	rows, err := e.Store.DB().Query(`SELECT policy_json FROM network_policies WHERE network_id = ? ORDER BY version`, netID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []int
	for rows.Next() {
		var pj string
		var p domain.Policy
		if err := rows.Scan(&pj); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(pj), &p); err != nil {
			t.Fatal(err)
		}
		out = append(out, p.Decimals)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// FR-005, GOV-004, DOM-031, QA-045: the scale is fixed from the moment the
// network is created. Every other scale in range is refused, at once or with
// notice, in setup and once active, before any proposal, posting, price or
// limit override exists; a refusal writes nothing and the initial limits keep
// their meaning. Updates that keep the scale still apply, at once or after
// their notice.
func TestDecimalsFixedFromCreation(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit, pol.MaxTxAmount = 12345, 67890
	n, _, err := e.Svc.CreateNetwork(e.Ctx, admin, service.CreateNetworkInput{Name: "Fixed", Slug: "fixed",
		Policy: &pol, CreatorDisplayName: "Founder"})
	if err != nil {
		t.Fatal(err)
	}
	update := func(p domain.Policy, scheduled bool) (*domain.Network, error) {
		var notice *int
		if scheduled {
			notice = iptr(14)
		}
		return e.Svc.UpdatePolicy(e.Ctx, admin, n.ID, service.PolicyChange{Policy: p, Reason: "rescale",
			NoticeDays: notice, IdempotencyKey: e.Key()})
	}
	for _, status := range []domain.NetworkStatus{domain.NetworkSetup, domain.NetworkActive} {
		if status == domain.NetworkActive {
			tu.Must(e.Svc.ActivateNetwork(e.Ctx, admin, n.ID))
		}
		before := dbChecksum(t, e)
		for dec := 0; dec <= domain.MaxDecimals; dec++ {
			if dec == pol.Decimals {
				continue
			}
			for _, scheduled := range []bool{false, true} {
				p := pol
				p.Decimals = dec
				if _, err := update(p, scheduled); !decimalsRefused(err) {
					t.Errorf("%s: decimals %d to %d (scheduled %v): %v", status, pol.Decimals, dec, scheduled, err)
				}
			}
		}
		if dbChecksum(t, e) != before {
			t.Errorf("%s: a refused scale change wrote to the database", status)
		}
		got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID))
		if got.Status != status || got.LedgerSeq != 0 || got.PolicyVersion != 1 || got.Policy != pol {
			t.Fatalf("%s: network %s seq %d v%d %+v", status, got.Status, got.LedgerSeq, got.PolicyVersion, got.Policy)
		}
	}

	p := pol
	p.MaxTxAmount = 70000
	if got := tu.Must(update(p, false)); got.PolicyVersion != 2 || got.Policy != p {
		t.Fatalf("same-scale update: v%d %+v", got.PolicyVersion, got.Policy)
	}
	p.DefaultCreditLimit = 100 // adverse, so it waits out the default notice
	tu.Must(update(p, false))
	e.Clock.Advance(15 * 24 * time.Hour)
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 3 || got.Policy != p {
		t.Errorf("scheduled same-scale update: v%d %+v", got.PolicyVersion, got.Policy)
	}
	if s := storedScales(t, e, n.ID); !slices.Equal(s, []int{2, 2, 2}) {
		t.Errorf("stored scales %v", s)
	}
}

// GOV-004, DOM-031: whatever is stored under the scale (a proposal in any
// status, a posting, a listing price, a credit limit), a scale change stays
// refused, at once or with notice, and the refusal changes nothing.
func TestDecimalsLockedWhateverIsStored(t *testing.T) {
	proposal := func(w *world) *domain.Transaction { return tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 250)) }
	trade := func(w *world) *domain.Transaction { return w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 250) }
	cases := []struct {
		name  string
		store func(w *world)
	}{
		{"proposed", func(w *world) { proposal(w) }},
		{"declined", func(w *world) {
			tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: proposal(w).ID, IdempotencyKey: w.Key()}))
		}},
		{"cancelled", func(w *world) {
			tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: proposal(w).ID, IdempotencyKey: w.Key()}))
		}},
		{"expired", func(w *world) {
			proposal(w)
			w.Clock.Advance(time.Duration(w.net.Policy.ProposalExpiryDays)*24*time.Hour + time.Minute)
			if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("test"), 10)); n != 1 {
				w.T.Fatalf("expired %d proposals", n)
			}
		}},
		{"completed", func(w *world) { trade(w) }},
		{"disputed", func(w *world) { tu.Must(w.open(w.a, trade(w).ID)) }},
		{"reversed", func(w *world) { tu.Must(w.reverse(w.admin, trade(w).ID, 250, "entered twice")) }},
		{"listing price", func(w *world) {
			price := int64(250)
			tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bread",
				Price: &price, IdempotencyKey: w.Key()}))
		}},
		{"credit limit", func(w *world) {
			limit := int64(250)
			tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &limit, Reason: "trusted"}))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t, nil)
			c.store(w)
			before := dbChecksum(t, w.Env)
			for _, notice := range []*int{nil, iptr(14)} {
				if _, err := w.setPolicy(func(p *domain.Policy) { p.Decimals = 3 }, notice, ""); !decimalsRefused(err) {
					t.Errorf("scheduled %v: %v", notice != nil, err)
				}
			}
			if dbChecksum(t, w.Env) != before {
				t.Error("a refused scale change wrote to the database")
			}
			w.Clock.Advance(15 * 24 * time.Hour)
			if n := w.netNow(); n.Policy.Decimals != 2 || n.PolicyVersion != 1 {
				t.Errorf("scale %d, policy v%d", n.Policy.Decimals, n.PolicyVersion)
			}
		})
	}
}

// A1-02 (was TestA1PoC_02): a scale change scheduled with notice while
// nothing is posted used to pass the "after the first posting" guard and take
// effect once postings existed, reinterpreting every stored amount. It is now
// refused when it is requested, so the scale in force after the notice period
// is still the creation scale and the posted amount keeps its meaning.
func TestScheduledDecimalChangeCannotReinterpretPostings(t *testing.T) {
	w := newWorld(t, nil) // decimals 2, nothing posted
	if n := w.netNow(); n.LedgerSeq != 0 || n.Policy.Decimals != 2 {
		t.Fatalf("precondition: seq %d decimals %d", n.LedgerSeq, n.Policy.Decimals)
	}
	if _, err := w.setPolicy(func(p *domain.Policy) { p.Decimals = 0 }, iptr(14), ""); !decimalsRefused(err) {
		t.Fatalf("scale change with notice before the first posting: %v", err)
	}
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500) // 25.00 credits at decimals 2
	if _, err := w.setPolicy(func(p *domain.Policy) { p.Decimals = 4 }, nil, ""); !decimalsRefused(err) {
		t.Fatalf("immediate scale change after the first posting: %v", err)
	}
	w.Clock.Advance(15 * 24 * time.Hour)
	after := w.netNow()
	if after.Policy.Decimals != 2 || after.PolicyVersion != 1 {
		t.Fatalf("scale %d (policy v%d) after %d posting(s): persisted amounts reinterpreted",
			after.Policy.Decimals, after.PolicyVersion, after.LedgerSeq)
	}
	if got := domain.FormatAmount(w.balance(w.b).Balance, after.Policy.Decimals); got != "25.00" {
		t.Fatalf("payee balance reads %s, want 25.00", got)
	}
	if s := storedScales(t, w.Env, w.net.ID); !slices.Equal(s, []int{2}) {
		t.Errorf("stored scales %v", s)
	}
}

// FR-005, GOV-004, DOM-031: the scale and the initial monetary settings are
// written in the creation unit. A failure at its last write leaves no network,
// policy, account, member or role behind; the retry stores them together.
func TestNetworkCreationStoresScaleAtomically(t *testing.T) {
	var fs *tu.FaultStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = tu.NewFaultStore(s); return fs })
	admin := e.Person("admin@example.org")
	pol := domain.DefaultPolicy()
	pol.Decimals, pol.DefaultCreditLimit, pol.MaxTxAmount = 3, 5000, 900000
	in := service.CreateNetworkInput{Name: "Atomic", Slug: "atomic", Policy: &pol, CreatorDisplayName: "Founder"}
	before := dbChecksum(t, e)
	fs.Arm("InsertAudit", 1) // network.created is the unit's last write
	if _, _, err := e.Svc.CreateNetwork(e.Ctx, admin, in); !errors.Is(err, tu.ErrInjected) {
		t.Fatalf("creation with a failing audit write: %v", err)
	}
	fs.Disarm()
	if dbChecksum(t, e) != before {
		t.Fatal("a failed creation left rows behind")
	}
	n, _, err := e.Svc.CreateNetwork(e.Ctx, admin, in)
	if err != nil {
		t.Fatal(err)
	}
	if got := tu.Must(e.Svc.GetNetwork(e.Ctx, admin, n.ID)); got.PolicyVersion != 1 || got.Policy != pol {
		t.Errorf("created v%d %+v", got.PolicyVersion, got.Policy)
	}
}

// GOV-004, FR-005, QA-045: creations race with policy writes on the networks
// just created. No network or account is ever visible without its scale, every
// scale change is refused, and updates that keep the scale apply with the
// amounts they store unchanged.
func TestDecimalsLockUnderConcurrentWrites(t *testing.T) {
	e := tu.NewEnv(t)
	const nets, writers = 5, 6
	founders := make([]domain.Actor, nets)
	for i := range founders {
		founders[i] = e.Person(fmt.Sprintf("founder%d@example.org", i))
	}
	stop := make(chan struct{})
	var unscaled atomic.Int64
	var watch sync.WaitGroup
	watch.Add(1)
	go func() { // every committed network and account has its creation policy, hence its scale
		defer watch.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			var k int
			err := e.Store.DB().QueryRow(`SELECT
				(SELECT COUNT(*) FROM networks n WHERE NOT EXISTS
					(SELECT 1 FROM network_policies p WHERE p.network_id = n.id AND p.version = 1)) +
				(SELECT COUNT(*) FROM accounts a WHERE NOT EXISTS
					(SELECT 1 FROM network_policies p WHERE p.network_id = a.network_id AND p.version = 1))`).Scan(&k)
			if err == nil && k > 0 {
				unscaled.Add(1)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	ids := make([]string, nets)
	var wg sync.WaitGroup
	for i := range nets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pol := domain.DefaultPolicy()
			pol.Decimals = i % (domain.MaxDecimals + 1)
			n, _, err := e.Svc.CreateNetwork(e.Ctx, founders[i], service.CreateNetworkInput{Name: "Race",
				Slug: fmt.Sprintf("race-%d", i), Policy: &pol, CreatorDisplayName: "Founder"})
			if err != nil {
				t.Errorf("create %d: %v", i, err)
				return
			}
			ids[i] = n.ID
			var inner sync.WaitGroup
			for j := range writers {
				inner.Add(1)
				go func() {
					defer inner.Done()
					// Even writers keep the scale and change an amount (not adverse,
					// so it applies at once); odd writers try another scale, every
					// other one with notice.
					p := pol
					p.MaxTxAmount = int64(1000 + j)
					var notice *int
					if j%2 == 1 {
						p.Decimals = (pol.Decimals + 1) % (domain.MaxDecimals + 1)
						if j%4 == 3 {
							notice = iptr(14)
						}
					}
					_, err := e.Svc.UpdatePolicy(e.Ctx, founders[i], n.ID, service.PolicyChange{Policy: p, Reason: "race",
						NoticeDays: notice, IdempotencyKey: e.Key()})
					switch {
					case j%2 == 1 && !decimalsRefused(err):
						t.Errorf("network %d writer %d: scale change: %v", i, j, err)
					case j%2 == 0 && err != nil:
						t.Errorf("network %d writer %d: same-scale update: %v", i, j, err)
					}
				}()
			}
			inner.Wait()
		}()
	}
	wg.Wait()
	close(stop)
	watch.Wait()

	if k := unscaled.Load(); k > 0 {
		t.Errorf("%d observation(s) of a network or account without its scale", k)
	}
	for i, id := range ids {
		if id == "" {
			continue // creation failed and was reported
		}
		scale := i % (domain.MaxDecimals + 1)
		for v, s := range storedScales(t, e, id) {
			if s != scale {
				t.Errorf("network %d version %d stores scale %d, created with %d", i, v+1, s, scale)
			}
		}
		got := tu.Must(e.Svc.GetNetwork(e.Ctx, founders[i], id))
		if got.Policy.Decimals != scale || !slices.Contains([]int64{1000, 1002, 1004}, got.Policy.MaxTxAmount) {
			t.Errorf("network %d: scale %d, max_tx_amount %d", i, got.Policy.Decimals, got.Policy.MaxTxAmount)
		}
	}
}
