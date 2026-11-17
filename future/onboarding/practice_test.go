package onboarding

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"math/rand"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func pid(s string) PracticeID { return PracticeID{s} }

func balances(w *World) map[string]int64 {
	out := map[string]int64{}
	for _, m := range w.Snapshot().Members {
		out[m.ID.String()] = m.Balance
	}
	return out
}

func TestPracticeLearningOutcomeFixtures(t *testing.T) {
	w := NewWorld()
	s := w.Snapshot()
	if !s.Practice || s.Notice != PracticeMarker {
		t.Fatal("snapshot not labelled practice")
	}
	// Fixture: a canned member has a legitimately negative balance within limit.
	cy, _ := s.MemberByID(pid("practice-cy"))
	if cy.Balance != -40 || cy.Limit != 100 || cy.Available() != 60 {
		t.Fatalf("negative-balance fixture: %+v", cy)
	}
	// Reserved credit fixture.
	pv, err := w.PreviewProposal(pid("practice-listing-1"))
	if err != nil || !pv.GoesNegative || pv.PayerBefore != 0 || pv.PayerAfter != -30 || pv.PayeeAfter != 70 || pv.CapacityNow != 100 || pv.CapacityPost != 70 {
		t.Fatalf("%+v %v", pv, err)
	}
	before := balances(w)
	id, err := w.Propose("k1", pid("practice-listing-1"))
	if err != nil {
		t.Fatal(err)
	}
	// Pending proposal: reservation only, balances unchanged ("not completed payment").
	if got := balances(w); !reflect.DeepEqual(got, before) {
		t.Fatalf("pending proposal moved balances: %v", got)
	}
	if l := w.Snapshot().Learner(); l.Reserved != 30 || l.Available() != 70 {
		t.Fatalf("%+v", l)
	}
	if w.Snapshot().Progress.IsCompleted(Trade) {
		t.Fatal("trade step completed by a mere proposal")
	}
	// Completed transfer fixture.
	if err := w.Respond("k2", id, true); err != nil {
		t.Fatal(err)
	}
	got := balances(w)
	if got["practice-ana"] != -30 || got["practice-bo"] != 70 {
		t.Fatalf("%v", got)
	}
	if l := w.Snapshot().Learner(); l.Reserved != 0 || !w.Snapshot().Progress.IsCompleted(Trade) {
		t.Fatalf("%+v", l)
	}
}

func TestPracticeIdempotentAndNoDoubleAccept(t *testing.T) {
	w := NewWorld()
	id, _ := w.Propose("p", pid("practice-listing-1"))
	if id2, err := w.Propose("p", pid("practice-listing-1")); err != nil || id2 != (PracticeID{}) {
		// replay returns the first error (nil) without creating anything new
		t.Fatalf("%v %v", id2, err)
	}
	if n := len(w.Snapshot().Proposals); n != 1 {
		t.Fatalf("replayed propose created %d proposals", n)
	}
	if err := w.Respond("a", id, true); err != nil {
		t.Fatal(err)
	}
	b := balances(w)
	if err := w.Respond("a", id, true); err != nil { // same key: replay
		t.Fatal(err)
	}
	if err := w.Respond("other", id, true); !errors.Is(err, ErrPracticeState) { // new key: refused
		t.Fatalf("second accept: %v", err)
	}
	if err := w.Respond("c", id, false); !errors.Is(err, ErrPracticeState) {
		t.Fatalf("decline after accept: %v", err)
	}
	if err := w.Cancel("d", id); !errors.Is(err, ErrPracticeState) {
		t.Fatalf("cancel after accept: %v", err)
	}
	if !reflect.DeepEqual(balances(w), b) {
		t.Fatal("balances moved twice")
	}
}

func TestPracticeCreditLimitBoundary(t *testing.T) {
	w := NewWorld()
	// Learner limit 100. Offers of 100 fit exactly; 101 does not.
	if _, err := w.CreateListing("", KindOffer, "x", 1); err != nil {
		t.Fatal(err)
	}
	// Learner buys sofa help? (Dee request: learner is the payee) — build offers from Bo via fixtures.
	w.mu.Lock()
	w.listings = append(w.listings,
		PracticeListing{pid("practice-listing-100"), pid("practice-bo"), KindOffer, "exact", 100},
		PracticeListing{pid("practice-listing-101"), pid("practice-bo"), KindOffer, "over", 101})
	w.mu.Unlock()
	if _, err := w.Propose("", pid("practice-listing-101")); !errors.Is(err, ErrPracticeCapacity) {
		t.Fatalf("limit+1: %v", err)
	}
	if _, err := w.Propose("", pid("practice-listing-100")); err != nil {
		t.Fatalf("exact limit: %v", err)
	}
	// Reservation consumes capacity: nothing more can be proposed.
	if _, err := w.Propose("", pid("practice-listing-1")); !errors.Is(err, ErrPracticeCapacity) {
		t.Fatalf("reserved credit ignored: %v", err)
	}
	// Cancel releases it.
	var pending PracticeID
	for _, p := range w.Snapshot().Proposals {
		pending = p.ID
	}
	if err := w.Cancel("", pending); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Propose("", pid("practice-listing-1")); err != nil {
		t.Fatalf("after cancel: %v", err)
	}
}

func TestPracticeValidation(t *testing.T) {
	w := NewWorld()
	bad := []struct {
		title  string
		amount int64
		kind   ListingKind
	}{
		{"", 5, KindOffer}, {"   ", 5, KindOffer}, {strings.Repeat("é", 61), 5, KindOffer},
		{"line\nbreak", 5, KindOffer}, {"ctl\x00", 5, KindOffer}, {"ok", 0, KindOffer}, {"ok", -5, KindOffer},
		{"ok", MaxPracticeAmount + 1, KindOffer}, {"ok", 5, "gift"}, {"bad\xff", 5, KindOffer},
	}
	for i, b := range bad {
		if _, err := w.CreateListing("", b.kind, b.title, b.amount); err == nil {
			t.Errorf("case %d accepted: %+v", i, b)
		}
	}
	for _, s := range []string{"", "ana", "practice-", "practice-A", "practice-../x", "practice-a b", strings.Repeat("practice-a", 10), "real-1"} {
		if id, err := ParsePracticeID(s); err == nil && s != "practice-" {
			t.Errorf("accepted %q -> %v", s, id)
		}
	}
	if _, err := ParsePracticeID("practice-bo"); err != nil {
		t.Fatal(err)
	}
	if err := w.ChoosePartner("", learnerID); !errors.Is(err, ErrPracticeState) {
		t.Fatalf("self as partner: %v", err)
	}
	if err := w.ChoosePartner("", pid("practice-nobody")); !errors.Is(err, ErrPracticeNotFound) {
		t.Fatal(err)
	}
	if _, err := w.Propose("", pid("practice-listing-99")); !errors.Is(err, ErrPracticeNotFound) {
		t.Fatal(err)
	}
	// A listing the learner owns cannot be proposed against (no self-trade).
	id, _ := w.CreateListing("", KindOffer, "mine", 5)
	if _, err := w.Propose("", id); !errors.Is(err, ErrPracticeState) {
		t.Fatalf("self trade: %v", err)
	}
	if err := w.AckTerms(strings.Repeat("k", 65)); !errors.Is(err, ErrPracticeText) {
		t.Fatalf("long idempotency key: %v", err)
	}
}

func TestPracticeLimitsBounded(t *testing.T) {
	w := NewWorld()
	var last error
	for i := 0; i < 200; i++ {
		_, last = w.CreateListing("", KindOffer, fmt.Sprintf("t%d", i), 1)
	}
	if !errors.Is(last, ErrPracticeLimit) || len(w.Snapshot().Listings) > maxPracticeListings {
		t.Fatalf("%v %d", last, len(w.Snapshot().Listings))
	}
	for i := 0; i < 500; i++ {
		w.AckTerms(fmt.Sprintf("k%d", i))
	}
	if len(w.keys) > maxPracticeKeys || len(w.Snapshot().Events) > maxPracticeEvents {
		t.Fatal("unbounded growth")
	}
}

func TestPracticeResetAndGuidedFlowFiveActions(t *testing.T) {
	w := NewWorld()
	// Acceptance: reach a (practice) listing and propose within five guided actions.
	actions := 0
	step := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		actions++
	}
	step(w.AckTerms("1"))
	step(w.AckCredit("2"))
	_, err := w.CreateListing("3", KindOffer, "Tutoring", 20)
	step(err)
	step(w.ChoosePartner("4", pid("practice-bo")))
	_, err = w.Propose("5", pid("practice-listing-1"))
	step(err)
	if actions != 5 {
		t.Fatal(actions)
	}
	if s := w.Snapshot(); s.Progress.Completed != 0b01111 {
		t.Fatalf("%05b", s.Progress.Completed)
	}
	w.Reset()
	s := w.Snapshot()
	if s.Progress.Completed != 0 || len(s.Proposals) != 0 || s.Learner().Reserved != 0 || len(s.Listings) != 3 {
		t.Fatalf("reset incomplete: %+v", s)
	}
	// a replayed key from before reset must not suppress new work
	if err := w.AckTerms("1"); err != nil || !w.Snapshot().Progress.IsCompleted(Terms) {
		t.Fatal("stale idempotency key survived reset")
	}
}

// Randomised operation sequences must preserve the ledger-style invariants.
func TestPracticeInvariantRandomOps(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		r := rand.New(rand.NewSource(seed))
		w := NewWorld()
		for i := 0; i < 60; i++ {
			switch r.Intn(6) {
			case 0:
				w.CreateListing("", ListingKind([]string{"offer", "request"}[r.Intn(2)]), "item", int64(r.Intn(130)))
			case 1:
				w.Propose("", pid(fmt.Sprintf("practice-listing-%d", 1+r.Intn(8))))
			case 2, 3:
				for _, p := range w.Snapshot().Proposals {
					if r.Intn(2) == 0 {
						w.Respond("", p.ID, r.Intn(3) != 0)
					}
				}
			case 4:
				for _, p := range w.Snapshot().Proposals {
					if r.Intn(3) == 0 {
						w.Cancel("", p.ID)
					}
				}
			case 5:
				w.AckCredit("")
			}
			if err := func() (err error) {
				w.mu.Lock()
				defer w.mu.Unlock()
				return w.invariant()
			}(); err != nil {
				t.Fatalf("seed %d step %d: %v", seed, i, err)
			}
		}
	}
}

func TestPracticeConcurrentAccept(t *testing.T) {
	w := NewWorld()
	id, _ := w.Propose("", pid("practice-listing-1"))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if w.Respond(fmt.Sprintf("c%d", i), id, true) == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if ok != 1 || balances(w)["practice-bo"] != 70 {
		t.Fatalf("ok=%d %v", ok, balances(w))
	}
}

// Practice isolation is structural: the engine file must not be able to reach
// any database, file, network or other package of this repo.
func TestPracticeSourceHasNoProductionReach(t *testing.T) {
	forbidden := []string{"database/sql", "os", "net", "net/http", "io/ioutil", "os/exec", "syscall", "unsafe", "plugin", "github.com", "reflect"}
	for _, f := range []string{"practice.go"} {
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, im := range af.Imports {
			p := strings.Trim(im.Path.Value, `"`)
			for _, bad := range forbidden {
				if p == bad || strings.HasPrefix(p, bad+"/") {
					t.Errorf("%s imports %s", f, p)
				}
			}
		}
	}
	// No constructor takes a handle: NewWorld has no parameters.
	if reflect.TypeOf(NewWorld).NumIn() != 0 {
		t.Fatal("NewWorld must take no arguments")
	}
}
