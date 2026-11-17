package onboarding

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

// PracticeMarker is shown on every practice page, error and export.
const PracticeMarker = "Practice — synthetic credits, no real trade"

// This file is the synthetic practice engine. It is deliberately self-contained:
// it imports no database, filesystem or network package, accepts no handle to
// any real service, and its ID/amount types are unexported-field structs that
// cannot be passed to anything that posts a transaction (this module has no
// such API). A source-level test enforces the import restriction.

// Practice limits keep a hostile or buggy client from growing a world.
const (
	maxPracticeListings  = 40
	maxPracticeProposals = 40
	maxPracticeEvents    = 100
	maxPracticeKeys      = 256
	MaxPracticeAmount    = 10_000
	MaxTitleRunes        = 60
	practicePrefix       = "practice-"
)

// PracticeID identifies a synthetic member, listing or proposal. The prefix
// makes a practice ID unmistakable and impossible to confuse with real IDs.
type PracticeID struct{ s string }

func (id PracticeID) String() string { return id.s }

// IsPractice reports whether id is a valid practice identifier.
func (id PracticeID) IsPractice() bool {
	return strings.HasPrefix(id.s, practicePrefix) && len(id.s) <= 40
}

// ParsePracticeID validates client-supplied text as a practice ID.
func ParsePracticeID(s string) (PracticeID, error) {
	id := PracticeID{s}
	if !id.IsPractice() {
		return PracticeID{}, ErrPracticeID
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return PracticeID{}, ErrPracticeID
		}
	}
	return id, nil
}

// Practice errors.
var (
	ErrPracticeID       = errors.New("practice: not a practice identifier")
	ErrPracticeNotFound = errors.New("practice: no such item")
	ErrPracticeState    = errors.New("practice: not allowed in the current state")
	ErrPracticeAmount   = errors.New("practice: invalid amount")
	ErrPracticeLimit    = errors.New("practice: limit reached")
	ErrPracticeText     = errors.New("practice: invalid text")
	ErrPracticeCapacity = errors.New("practice: this exceeds the credit limit")
)

// PracticeMember is a canned synthetic member.
type PracticeMember struct {
	ID       PracticeID
	Name     string
	Balance  int64 // negative is normal and allowed down to -Limit
	Limit    int64
	Reserved int64 // credit held by pending proposals this member would pay
	Learner  bool
}

// Available is the credit the member can still commit.
func (m PracticeMember) Available() int64 { return m.Balance + m.Limit - m.Reserved }

// ListingKind is offer or request.
type ListingKind string

const (
	KindOffer   ListingKind = "offer"
	KindRequest ListingKind = "request"
)

// PracticeListing is a synthetic offer or request.
type PracticeListing struct {
	ID     PracticeID
	Owner  PracticeID
	Kind   ListingKind
	Title  string
	Amount int64
}

// ProposalState is the synthetic proposal lifecycle.
type ProposalState string

const (
	ProposalPending   ProposalState = "pending"
	ProposalAccepted  ProposalState = "accepted"
	ProposalDeclined  ProposalState = "declined"
	ProposalCancelled ProposalState = "cancelled"
)

// PracticeProposal is a synthetic exchange. Only Accepted moves balances.
type PracticeProposal struct {
	ID      PracticeID
	Listing PracticeID
	Payer   PracticeID
	Payee   PracticeID
	Amount  int64
	State   ProposalState
}

// World is one learner's isolated synthetic network. Safe for concurrent use.
type World struct {
	mu        sync.Mutex
	members   []*PracticeMember
	listings  []PracticeListing
	proposals []*PracticeProposal
	progress  Progress // practice-only checklist state; never persisted or exported to the real checklist
	partner   PracticeID
	events    []string
	keys      map[string]error
	keyOrder  []string
	seq       int
}

// LearnerID is the canned learner member.
var learnerID = PracticeID{"practice-ana"}

// NewWorld returns a freshly seeded world. It takes no arguments on purpose.
func NewWorld() *World {
	w := &World{}
	w.seed()
	return w
}

func (w *World) seed() {
	w.members = []*PracticeMember{
		{ID: learnerID, Name: "Ana (you)", Balance: 0, Limit: 100, Learner: true},
		{ID: PracticeID{"practice-bo"}, Name: "Bo", Balance: 40, Limit: 100},
		{ID: PracticeID{"practice-cy"}, Name: "Cy", Balance: -40, Limit: 100},
		{ID: PracticeID{"practice-dee"}, Name: "Dee", Balance: 0, Limit: 100},
	}
	w.listings = []PracticeListing{
		{PracticeID{"practice-listing-1"}, PracticeID{"practice-bo"}, KindOffer, "Bicycle tune-up (synthetic)", 30},
		{PracticeID{"practice-listing-2"}, PracticeID{"practice-cy"}, KindOffer, "Homemade bread (synthetic)", 15},
		{PracticeID{"practice-listing-3"}, PracticeID{"practice-dee"}, KindRequest, "Help moving a sofa (synthetic)", 25},
	}
	w.proposals = nil
	w.progress = Progress{Version: CurriculumVersion}
	w.partner = PracticeID{}
	w.events = []string{"Practice network created with synthetic members. Nothing here is real."}
	w.keys = map[string]error{}
	w.keyOrder = nil
	w.seq = 3
}

// Reset discards all practice state and re-seeds. Nothing is carried anywhere.
func (w *World) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.seed()
}

func (w *World) member(id PracticeID) *PracticeMember {
	for _, m := range w.members {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (w *World) listing(id PracticeID) (PracticeListing, bool) {
	for _, l := range w.listings {
		if l.ID == id {
			return l, true
		}
	}
	return PracticeListing{}, false
}

func (w *World) proposal(id PracticeID) *PracticeProposal {
	for _, p := range w.proposals {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (w *World) logf(format string, a ...any) {
	if len(w.events) >= maxPracticeEvents {
		w.events = w.events[1:]
	}
	w.events = append(w.events, fmt.Sprintf(format, a...))
}

// once runs fn at most once per non-empty command key (idempotency). A replay
// returns the first call's error without repeating any effect.
func (w *World) once(key string, fn func() error) error {
	if key != "" {
		if len(key) > 64 {
			return ErrPracticeText
		}
		if err, ok := w.keys[key]; ok {
			return err
		}
	}
	err := fn()
	if key != "" {
		if len(w.keyOrder) >= maxPracticeKeys {
			delete(w.keys, w.keyOrder[0])
			w.keyOrder = w.keyOrder[1:]
		}
		w.keys[key] = err
		w.keyOrder = append(w.keyOrder, key)
	}
	if err == nil {
		if ierr := w.invariant(); ierr != nil {
			panic(ierr) // programming error: never reachable through the API
		}
	}
	return err
}

func (w *World) invariant() error {
	var sum int64
	for _, m := range w.members {
		sum += m.Balance
		if m.Reserved < 0 || m.Balance < -m.Limit || m.Reserved > m.Balance+m.Limit {
			return fmt.Errorf("practice invariant broken for %s", m.ID)
		}
	}
	if sum != 0 {
		return fmt.Errorf("practice balances do not sum to zero: %d", sum)
	}
	return nil
}

// AckTerms records that the learner read the synthetic terms explanation.
func (w *World) AckTerms(key string) error { return w.ack(key, Terms) }

// AckCredit records that the learner read the credit-limit explanation.
func (w *World) AckCredit(key string) error { return w.ack(key, Credit) }

func (w *World) ack(key string, s Step) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.once(key, func() error {
		p, err := w.progress.Complete(s)
		w.progress = p
		return err
	})
}

// CreateListing adds a synthetic listing owned by the learner.
func (w *World) CreateListing(key string, kind ListingKind, title string, amount int64) (PracticeID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var id PracticeID
	err := w.once(key, func() error {
		if kind != KindOffer && kind != KindRequest {
			return ErrPracticeText
		}
		title = strings.TrimSpace(title)
		if title == "" || utf8.RuneCountInString(title) > MaxTitleRunes || ValidateText(title, MaxTitleRunes) != nil || strings.Contains(title, "\n") {
			return ErrPracticeText
		}
		if amount < 1 || amount > MaxPracticeAmount {
			return ErrPracticeAmount
		}
		if len(w.listings) >= maxPracticeListings {
			return ErrPracticeLimit
		}
		w.seq++
		id = PracticeID{fmt.Sprintf("practice-listing-%d", w.seq)}
		w.listings = append(w.listings, PracticeListing{id, learnerID, kind, title, amount})
		p, _ := w.progress.Complete(Listing)
		w.progress = p
		w.logf("You added a synthetic %s: %q for %d credits.", kind, title, amount)
		return nil
	})
	if err != nil {
		return PracticeID{}, err
	}
	return id, nil
}

// ChoosePartner records the learner's chosen trading partner (a canned member
// other than the learner).
func (w *World) ChoosePartner(key string, id PracticeID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.once(key, func() error {
		m := w.member(id)
		if m == nil {
			return ErrPracticeNotFound
		}
		if m.Learner {
			return ErrPracticeState
		}
		w.partner = id
		p, _ := w.progress.Complete(Partner)
		w.progress = p
		w.logf("You chose %s as a trading partner.", m.Name)
		return nil
	})
}

// PreviewResult is the explanatory before/after view of a proposal.
type PreviewResult struct {
	Preview
	Amount       int64
	Limit        int64
	ReservedNow  int64
	CapacityNow  int64
	CapacityPost int64
	GoesNegative bool
}

// PreviewProposal computes the balance preview for the learner paying for a
// listing, without changing anything.
func (w *World) PreviewProposal(listing PracticeID) (PreviewResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.previewLocked(listing)
}

func (w *World) previewLocked(lid PracticeID) (PreviewResult, error) {
	l, ok := w.listing(lid)
	if !ok {
		return PreviewResult{}, ErrPracticeNotFound
	}
	payer, payee, err := w.parties(l)
	if err != nil {
		return PreviewResult{}, err
	}
	pv, err := PreviewTrade(payer.Balance, payee.Balance, payer.Limit, payer.Reserved, l.Amount)
	if err != nil {
		return PreviewResult{}, ErrPracticeCapacity
	}
	return PreviewResult{Preview: pv, Amount: l.Amount, Limit: payer.Limit, ReservedNow: payer.Reserved,
		CapacityNow: payer.Available(), CapacityPost: payer.Available() - l.Amount, GoesNegative: pv.PayerAfter < 0}, nil
}

// parties resolves who pays whom for a listing. The learner can only trade with
// listings owned by someone else.
func (w *World) parties(l PracticeListing) (payer, payee *PracticeMember, err error) {
	owner := w.member(l.Owner)
	learner := w.member(learnerID)
	if owner == nil || learner == nil || owner.Learner {
		return nil, nil, ErrPracticeState
	}
	// An offer is paid for by the learner; a request is fulfilled by the learner.
	if l.Kind == KindOffer {
		return learner, owner, nil
	}
	return owner, learner, nil
}

// Propose creates a pending proposal and reserves the payer's credit. It never
// moves balances: a proposal is not completed payment.
func (w *World) Propose(key string, listing PracticeID) (PracticeID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var id PracticeID
	err := w.once(key, func() error {
		pr, err := w.previewLocked(listing)
		if err != nil {
			return err
		}
		if len(w.proposals) >= maxPracticeProposals {
			return ErrPracticeLimit
		}
		l, _ := w.listing(listing)
		payer, payee, _ := w.parties(l)
		for _, p := range w.proposals {
			if p.Listing == listing && p.State == ProposalPending {
				return ErrPracticeState // one open proposal per listing
			}
		}
		w.seq++
		id = PracticeID{fmt.Sprintf("practice-proposal-%d", w.seq)}
		payer.Reserved += pr.Amount
		w.proposals = append(w.proposals, &PracticeProposal{id, listing, payer.ID, payee.ID, pr.Amount, ProposalPending})
		w.logf("Proposal: %s would pay %s %d credits. Pending — no balance has changed yet.", payer.Name, payee.Name, pr.Amount)
		return nil
	})
	if err != nil {
		return PracticeID{}, err
	}
	return id, nil
}

// Respond is the scripted partner event: accept or decline a pending proposal.
// Accepting is the only action that moves synthetic balances, atomically with
// releasing the reservation. Replays and finished proposals are refused.
func (w *World) Respond(key string, id PracticeID, accept bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.once(key, func() error {
		p := w.proposal(id)
		if p == nil {
			return ErrPracticeNotFound
		}
		if p.State != ProposalPending {
			return ErrPracticeState
		}
		payer, payee := w.member(p.Payer), w.member(p.Payee)
		payer.Reserved -= p.Amount
		if !accept {
			p.State = ProposalDeclined
			w.logf("Scripted reply: %s declined. Reservation released; balances unchanged.", payee.Name)
			return nil
		}
		if _, err := PreviewTrade(payer.Balance, payee.Balance, payer.Limit, payer.Reserved, p.Amount); err != nil {
			payer.Reserved += p.Amount // roll back the release; proposal stays pending
			return ErrPracticeCapacity
		}
		payer.Balance -= p.Amount
		payee.Balance += p.Amount
		p.State = ProposalAccepted
		w.logf("Scripted reply: %s accepted. %s paid %s %d synthetic credits.", payee.Name, payer.Name, payee.Name, p.Amount)
		if payer.Learner || payee.Learner {
			np, _ := w.progress.Complete(Trade)
			w.progress = np
		}
		return nil
	})
}

// Cancel withdraws a pending proposal and releases the reservation.
func (w *World) Cancel(key string, id PracticeID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.once(key, func() error {
		p := w.proposal(id)
		if p == nil {
			return ErrPracticeNotFound
		}
		if p.State != ProposalPending {
			return ErrPracticeState
		}
		w.member(p.Payer).Reserved -= p.Amount
		p.State = ProposalCancelled
		w.logf("You cancelled the proposal. Reservation released; balances unchanged.")
		return nil
	})
}

// Snapshot is an immutable copy of the world for rendering and export.
type Snapshot struct {
	Practice  bool
	Notice    string
	Members   []PracticeMember
	Listings  []PracticeListing
	Proposals []PracticeProposal
	Progress  Progress
	Partner   PracticeID
	Events    []string
}

// Snapshot copies the current state.
func (w *World) Snapshot() Snapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := Snapshot{Practice: true, Notice: PracticeMarker, Progress: w.progress, Partner: w.partner,
		Events: append([]string(nil), w.events...)}
	for _, m := range w.members {
		s.Members = append(s.Members, *m)
	}
	s.Listings = append(s.Listings, w.listings...)
	for _, p := range w.proposals {
		s.Proposals = append(s.Proposals, *p)
	}
	return s
}

// ListingByID looks a listing up in a snapshot.
func (s Snapshot) ListingByID(id PracticeID) (PracticeListing, bool) {
	for _, l := range s.Listings {
		if l.ID == id {
			return l, true
		}
	}
	return PracticeListing{}, false
}

// MemberByID looks a member up in a snapshot.
func (s Snapshot) MemberByID(id PracticeID) (PracticeMember, bool) {
	for _, m := range s.Members {
		if m.ID == id {
			return m, true
		}
	}
	return PracticeMember{}, false
}

// Learner returns the learner member.
func (s Snapshot) Learner() PracticeMember { m, _ := s.MemberByID(learnerID); return m }
