package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// world is a network with an administrator and two members, A and B.
type world struct {
	*tu.Env
	net    *domain.Network
	admin  domain.Actor
	adminM *domain.Member
	a, b   domain.Actor
	aM, bM *domain.Member
}

func newWorld(t *testing.T, mod func(*domain.Policy)) *world {
	t.Helper()
	e := tu.NewEnv(t)
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", mod)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	return w
}

func (w *world) propose(from domain.Actor, to *domain.Member, dir service.Direction, amount int64) (*domain.Transaction, error) {
	return w.Svc.ProposeTransaction(w.Ctx, from, w.net.ID, service.ProposeInput{CounterpartyID: to.ID,
		Direction: dir, Amount: amount, IdempotencyKey: w.Key()})
}

func (w *world) accept(by domain.Actor, id string) (*domain.Transaction, error) {
	return w.Svc.AcceptTransaction(w.Ctx, by, w.net.ID, service.TxRef{TxID: id, IdempotencyKey: w.Key()})
}

func (w *world) balance(a domain.Actor) *service.BalanceView {
	return tu.Must(w.Svc.GetBalance(w.Ctx, a, w.net.ID))
}

func wantCode(t *testing.T, err error, target *domain.Error) {
	t.Helper()
	if err == nil || !errors.Is(err, target) {
		t.Fatalf("want error code %q, got %v", target.Code, err)
	}
}

// DOM-006 worked illustration: 25.00 from payer to payee.
func TestTradeScenarioBalances(t *testing.T) {
	w := newWorld(t, nil)
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2500)
	if tx.Status != domain.TxCompleted || tx.CompletedAt == nil || tx.LedgerSeq != 1 {
		t.Fatalf("unexpected tx %+v", tx)
	}
	if got := w.balance(w.a).Balance; got != -2500 {
		t.Errorf("payer balance = %d, want -2500", got)
	}
	if got := w.balance(w.b).Balance; got != 2500 {
		t.Errorf("payee balance = %d, want 2500", got)
	}
	v := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, tx.ID))
	if len(v.Entries) != 2 || v.Entries[0].Amount != -2500 || v.Entries[1].Amount != 2500 {
		t.Errorf("entries = %+v", v.Entries)
	}
	if n := len(v.History); n != 2 || v.History[1].To != domain.TxCompleted {
		t.Errorf("history = %+v", v.History)
	}
	rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if !rep.OK {
		t.Errorf("ledger inconsistent: %v", rep.Findings)
	}
}

// DOM-011: a payer-initiated proposal needs the payee, a payee-initiated one needs the payer.
func TestConsentBothDirections(t *testing.T) {
	w := newWorld(t, nil)
	pay := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1000))
	if pay.PayerMemberID != w.aM.ID || pay.PayeeMemberID != w.bM.ID {
		t.Fatalf("payer-initiated parties wrong: %+v", pay)
	}
	_, err := w.accept(w.a, pay.ID) // proposer accepting own proposal
	wantCode(t, err, domain.ErrForbidden)
	if w.balance(w.a).Balance != 0 {
		t.Error("nothing may post without the counterparty's accept")
	}
	tu.Must(w.accept(w.b, pay.ID))

	req := tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 700)) // B asks A to pay B
	if req.PayerMemberID != w.aM.ID || req.PayeeMemberID != w.bM.ID {
		t.Fatalf("payee-initiated parties wrong: %+v", req)
	}
	_, err = w.accept(w.b, req.ID)
	wantCode(t, err, domain.ErrForbidden)
	tu.Must(w.accept(w.a, req.ID))
	if got := w.balance(w.a).Balance; got != -1700 {
		t.Errorf("A balance = %d want -1700", got)
	}
}

func TestProposalValidation(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.MaxTxAmount = 5000 })
	cases := []struct {
		name string
		in   service.ProposeInput
		want *domain.Error
	}{
		{"zero", service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 0}, domain.ErrValidation},
		{"negative", service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: -5}, domain.ErrValidation},
		{"self", service.ProposeInput{CounterpartyID: w.aM.ID, Direction: service.ProposerPays, Amount: 5}, domain.ErrValidation},
		{"direction", service.ProposeInput{CounterpartyID: w.bM.ID, Direction: "x", Amount: 5}, domain.ErrValidation},
		{"over-max", service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 5001}, domain.ErrAmountExceeds},
		{"unknown-counterparty", service.ProposeInput{CounterpartyID: "nope", Direction: service.ProposerPays, Amount: 5}, domain.ErrNotFound},
	}
	for _, c := range cases {
		c.in.IdempotencyKey = w.Key()
		_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, c.in)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %s", c.name, err, c.want.Code)
		}
	}
	// exactly at the max is accepted (edge), limit is 100.00 so 50.00 fits
	if _, err := w.propose(w.a, w.bM, service.ProposerPays, 5000); err != nil {
		t.Errorf("amount equal to max must be accepted: %v", err)
	}
	// note too long
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, service.ProposeInput{CounterpartyID: w.bM.ID,
		Direction: service.ProposerPays, Amount: 1, Note: string(long), IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
}

// DOM-012 acceptance: available 30.00; propose 20.00 ok; then 15.00 refused (10.00 left);
// declining the first restores availability.
func TestReservationAndRelease(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 3000 })
	first := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 2000))
	bv := w.balance(w.a)
	if bv.ReservedOutgoing != 2000 || bv.Available != 1000 {
		t.Fatalf("after first proposal: reserved=%d available=%d", bv.ReservedOutgoing, bv.Available)
	}
	if w.balance(w.b).AwaitingFromOthers != 2000 || w.balance(w.b).Available != 3000 {
		t.Errorf("incoming pending must not increase available: %+v", w.balance(w.b))
	}
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 1500)
	wantCode(t, err, domain.ErrInsufficientCredit)
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: first.ID, IdempotencyKey: w.Key()}))
	if got := w.balance(w.a).Available; got != 3000 {
		t.Errorf("available after decline = %d want 3000", got)
	}
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1500))
}

// DOM-008 table: available credit across balance/limit/reservation combinations.
func TestAvailableCreditTable(t *testing.T) {
	cases := []struct {
		name            string
		limit, tradeOut int64 // A pays B tradeOut first (posted)
		reserveOut      int64
		wantAvail       int64
	}{
		{"fresh", 5000, 0, 0, 5000},
		{"negative-within-limit", 5000, 2000, 0, 3000},
		{"at-limit", 5000, 5000, 0, 0},
		{"with-reservation", 5000, 1000, 1500, 2500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = c.limit })
			if c.tradeOut > 0 {
				w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, c.tradeOut)
			}
			if c.reserveOut > 0 {
				tu.Must(w.propose(w.a, w.bM, service.ProposerPays, c.reserveOut))
			}
			if got := w.balance(w.a).Available; got != c.wantAvail {
				t.Errorf("available = %d want %d", got, c.wantAvail)
			}
		})
	}
	// positive balance spends above the limit-free amount
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 1000 })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	if got := w.balance(w.b).Available; got != 2000 {
		t.Errorf("positive balance available = %d want 2000", got)
	}
}

func TestDeclineCancelExitRules(t *testing.T) {
	w := newWorld(t, nil)
	p1 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	// proposer cannot decline their own; counterparty cannot "withdraw"
	_, err := w.Svc.DeclineTransaction(w.Ctx, w.a, w.net.ID, service.TxRef{TxID: p1.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.CancelTransaction(w.Ctx, w.b, w.net.ID, service.CancelInput{TxID: p1.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrForbidden)
	got := tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: p1.ID, IdempotencyKey: w.Key()}))
	if got.Status != domain.TxCancelled {
		t.Errorf("status = %s", got.Status)
	}
	// terminal: cannot accept, decline or cancel again
	_, err = w.accept(w.b, p1.ID)
	wantCode(t, err, domain.ErrInvalidState)
	_, err = w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: p1.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrInvalidState)

	// administrator (not a party) cancels with a reason; reason is mandatory
	p2 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	_, err = w.Svc.CancelTransaction(w.Ctx, w.admin, w.net.ID, service.CancelInput{TxID: p2.ID, IdempotencyKey: w.Key()})
	wantCode(t, err, domain.ErrValidation)
	c := tu.Must(w.Svc.CancelTransaction(w.Ctx, w.admin, w.net.ID, service.CancelInput{TxID: p2.ID, Reason: "spam", IdempotencyKey: w.Key()}))
	if c.Status != domain.TxCancelled {
		t.Errorf("admin cancel status = %s", c.Status)
	}
}

// DOM-013: expired proposals cannot be accepted; reservation released; sweep is idempotent.
func TestExpiry(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ProposalExpiryDays = 2; p.DefaultCreditLimit = 1000 })
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1000))
	if w.balance(w.a).Available != 0 {
		t.Fatal("reservation expected")
	}
	w.Clock.Advance(2*24*time.Hour - time.Second)
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("job"), 100)); n != 0 {
		t.Errorf("nothing should expire yet, got %d", n)
	}
	w.Clock.Advance(2 * time.Second)
	if w.balance(w.a).Available != 1000 {
		t.Error("reservation must be released at expiry even before the sweep runs")
	}
	_, err := w.accept(w.b, p.ID)
	wantCode(t, err, domain.ErrInvalidState)
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("job"), 100)); n != 1 {
		t.Errorf("sweep expired %d want 1", n)
	}
	if n := tu.Must(w.Svc.ExpireProposals(w.Ctx, domain.SystemActor("job"), 100)); n != 0 {
		t.Errorf("re-running the sweep changed %d rows", n)
	}
	v := tu.Must(w.Svc.GetTransaction(w.Ctx, w.a, w.net.ID, p.ID))
	if v.Tx.Status != domain.TxExpired {
		t.Errorf("status = %s", v.Tx.Status)
	}
	_, err = w.Svc.ExpireProposals(w.Ctx, w.a, 10)
	wantCode(t, err, domain.ErrForbidden)
}

// DOM-032: reduce limit below the current negative balance: balance unchanged,
// new debits refused, incoming credits still allowed.
func TestLimitReductionDoesNotPost(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 4000)
	before := w.balance(w.a).Balance
	zero := int64(1000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &zero, Reason: "risk"}))
	if got := w.balance(w.a); got.Balance != before || got.CreditLimit != 1000 || got.Available != -3000 {
		t.Errorf("after reduction: %+v", got)
	}
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrInsufficientCredit)
	// B pays A: incoming credit moves A toward zero and is allowed
	w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 500)
	if got := w.balance(w.a).Balance; got != before+500 {
		t.Errorf("balance = %d", got)
	}
	// override can be removed -> default applies again
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Reason: "restore"}))
	if w.balance(w.a).CreditLimit != 5000 {
		t.Error("default must apply after override removal")
	}
}

// A limit lowered between proposal and acceptance is caught by the hard check (DOM-014).
func TestAcceptRechecksLimit(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 })
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 4000))
	low := int64(1000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &low, Reason: "risk"}))
	_, err := w.accept(w.b, p.ID)
	wantCode(t, err, domain.ErrInsufficientCredit)
	v := tu.Must(w.Svc.GetTransaction(w.Ctx, w.b, w.net.ID, p.ID))
	if v.Tx.Status != domain.TxProposed || len(v.Entries) != 0 {
		t.Errorf("failed accept must leave the proposal untouched: %s, %d entries", v.Tx.Status, len(v.Entries))
	}
}

// Time-limited override (DOM-021 path): expires back to the default.
func TestLimitOverrideExpires(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 100 })
	big := int64(9000)
	exp := w.Clock.Now().Add(time.Hour)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &big, ExpiresAt: &exp, Reason: "temp"}))
	if w.balance(w.a).CreditLimit != 9000 {
		t.Fatal("override not applied")
	}
	w.Clock.Advance(2 * time.Hour)
	if w.balance(w.a).CreditLimit != 100 {
		t.Error("override must expire")
	}
	past := w.Clock.Now().Add(-time.Minute)
	err := w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &big, ExpiresAt: &past, Reason: "x"})
	wantCode(t, err, domain.ErrValidation)
}

// DOM-040: listings have no balance effect.
func TestListingsDoNotTouchLedger(t *testing.T) {
	w := newWorld(t, nil)
	before := w.balance(w.a)
	price := int64(500)
	l := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike repair", Price: &price, IdempotencyKey: w.Key()}))
	paused := domain.ListingPaused
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &paused}))
	after := w.balance(w.a)
	if before.Balance != after.Balance || before.LedgerSeq != after.LedgerSeq || after.ReservedOutgoing != 0 {
		t.Errorf("listing operations changed the ledger: %+v -> %+v", before, after)
	}
	// others see only active listings; a paused one is hidden from B but visible to the owner
	if got := tu.Must(w.Svc.ListListings(w.Ctx, w.b, w.net.ID, service.ListingQuery{})); len(got) != 0 {
		t.Errorf("paused listing visible to others: %d", len(got))
	}
	if got := tu.Must(w.Svc.ListListings(w.Ctx, w.a, w.net.ID, service.ListingQuery{MemberID: w.aM.ID})); len(got) != 1 {
		t.Errorf("owner should see own paused listing, got %d", len(got))
	}
	// trade may reference a listing
	active := domain.ListingActive
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &active}))
	tx := tu.Must(w.Svc.ProposeTransaction(w.Ctx, w.b, w.net.ID, service.ProposeInput{CounterpartyID: w.aM.ID,
		Direction: service.ProposerPays, Amount: 500, ListingID: l.ID, IdempotencyKey: w.Key()}))
	if tx.ListingID != l.ID {
		t.Error("listing reference lost")
	}
	// another member cannot edit it; it reads as not found
	_, err := w.Svc.UpdateListing(w.Ctx, w.b, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &paused})
	wantCode(t, err, domain.ErrNotFound)
}
