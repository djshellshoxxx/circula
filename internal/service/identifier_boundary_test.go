package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Observe the repository boundary, including filters and batch mutations. The
// assertions below permit the established authentication/network queries but
// forbid passing a malformed target identifier to persistence (SEC-024/A2-16).
type identifierStore struct {
	store.Store
	mu   sync.Mutex
	seen []string
}

func (s *identifierStore) record(ids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, ids...)
}
func (s *identifierStore) count(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, seen := range s.seen {
		if seen == id {
			n++
		}
	}
	return n
}
func (s *identifierStore) wrap(tx store.Tx) store.Tx {
	return &identifierTx{Tx: tx, GovRepo: tx.(store.GovRepo), MemberPager: tx.(store.MemberPager),
		AnnouncementRepo: tx.(store.AnnouncementRepo), PrefsRepo: tx.(store.PrefsRepo), OverviewRepo: tx.(store.OverviewRepo), s: s}
}
func (s *identifierStore) View(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.View(ctx, func(tx store.Tx) error { return fn(s.wrap(tx)) })
}
func (s *identifierStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	return s.Store.Update(ctx, func(tx store.Tx) error { return fn(s.wrap(tx)) })
}

type identifierTx struct {
	store.Tx
	store.GovRepo
	store.MemberPager
	store.AnnouncementRepo
	store.PrefsRepo
	store.OverviewRepo
	s *identifierStore
}

func (tx *identifierTx) GetMember(netID, id string) (*domain.Member, error) {
	tx.s.record(id)
	return tx.Tx.GetMember(netID, id)
}
func (tx *identifierTx) GetListing(netID, id string) (*domain.Listing, error) {
	tx.s.record(id)
	return tx.Tx.GetListing(netID, id)
}
func (tx *identifierTx) GetTransaction(netID, id string) (*domain.Transaction, error) {
	tx.s.record(id)
	return tx.Tx.GetTransaction(netID, id)
}
func (tx *identifierTx) GetDispute(netID, id string) (*domain.Dispute, error) {
	tx.s.record(id)
	return tx.Tx.GetDispute(netID, id)
}
func (tx *identifierTx) GetReport(netID, id string) (*store.Report, error) {
	tx.s.record(id)
	return tx.Tx.GetReport(netID, id)
}
func (tx *identifierTx) GetChallenge(netID, id string) (*store.Challenge, error) {
	tx.s.record(id)
	return tx.Tx.GetChallenge(netID, id)
}
func (tx *identifierTx) GetPendingAdjustment(netID, id string) (*domain.PendingAdjustment, error) {
	tx.s.record(id)
	return tx.Tx.GetPendingAdjustment(netID, id)
}
func (tx *identifierTx) GetAnnouncement(netID, id string) (*domain.Announcement, error) {
	tx.s.record(id)
	return tx.AnnouncementRepo.GetAnnouncement(netID, id)
}
func (tx *identifierTx) GetMemberByPerson(netID, id string) (*domain.Member, error) {
	tx.s.record(netID)
	return tx.Tx.GetMemberByPerson(netID, id)
}
func (tx *identifierTx) RevokeInvitation(netID, id string, at time.Time) (bool, error) {
	tx.s.record(id)
	return tx.Tx.RevokeInvitation(netID, id, at)
}
func (tx *identifierTx) RevokeToken(personID, id string, at time.Time) (bool, error) {
	tx.s.record(id)
	return tx.Tx.RevokeToken(personID, id, at)
}
func (tx *identifierTx) RetractAnnouncement(netID, id, by string, at time.Time) (bool, error) {
	tx.s.record(id)
	return tx.AnnouncementRepo.RetractAnnouncement(netID, id, by, at)
}
func (tx *identifierTx) MarkNotificationsRead(netID, memberID string, ids []string, at time.Time) (int64, error) {
	tx.s.record(ids...)
	return tx.Tx.MarkNotificationsRead(netID, memberID, ids, at)
}
func (tx *identifierTx) ListMembers(netID string, f store.MemberFilter) ([]*domain.Member, error) {
	tx.s.record(f.AfterID)
	return tx.Tx.ListMembers(netID, f)
}
func (tx *identifierTx) PageMembers(netID string, f store.MemberPageFilter) ([]*domain.Member, error) {
	tx.s.record(f.AfterID, f.BeforeID)
	return tx.MemberPager.PageMembers(netID, f)
}
func (tx *identifierTx) ListTransactions(netID string, f store.TxFilter) ([]*domain.Transaction, error) {
	tx.s.record(f.AfterID, f.OriginalTxID)
	return tx.Tx.ListTransactions(netID, f)
}
func (tx *identifierTx) ListDisputes(netID string, f store.DisputeFilter) ([]*domain.Dispute, error) {
	tx.s.record(f.AfterID, f.TxID)
	return tx.Tx.ListDisputes(netID, f)
}
func (tx *identifierTx) ListListings(netID string, f store.ListingFilter) ([]*domain.Listing, error) {
	tx.s.record(f.AfterID, f.MemberID)
	return tx.Tx.ListListings(netID, f)
}
func (tx *identifierTx) ListNotifications(netID, memberID string, f store.NotificationFilter) ([]*store.Notification, error) {
	tx.s.record(f.AfterID)
	return tx.Tx.ListNotifications(netID, memberID, f)
}
func (tx *identifierTx) ListReports(netID string, f store.ReportFilter) ([]*store.Report, error) {
	tx.s.record(f.AfterID)
	return tx.Tx.ListReports(netID, f)
}
func (tx *identifierTx) ListAnnouncements(netID string, f store.AnnouncementFilter) ([]*domain.Announcement, error) {
	tx.s.record(f.AfterID)
	return tx.AnnouncementRepo.ListAnnouncements(netID, f)
}
func (tx *identifierTx) ListChallengesPage(netID, memberID, status, afterID string, limit int) ([]store.Challenge, error) {
	tx.s.record(afterID)
	return tx.Tx.ListChallengesPage(netID, memberID, status, afterID, limit)
}
func (tx *identifierTx) ListBreakGlass(netID, afterID string, limit int) ([]*domain.BreakGlassSession, error) {
	tx.s.record(afterID)
	return tx.Tx.ListBreakGlass(netID, afterID, limit)
}
func (tx *identifierTx) ListInactiveMembers(netID string, cutoff time.Time, afterID string, limit int) ([]*domain.Member, error) {
	tx.s.record(afterID)
	return tx.Tx.ListInactiveMembers(netID, cutoff, afterID, limit)
}
func (tx *identifierTx) ListPendingAdjustments(netID string, statuses []domain.PendingStatus, afterID string, limit int) ([]*domain.PendingAdjustment, error) {
	tx.s.record(afterID)
	return tx.Tx.ListPendingAdjustments(netID, statuses, afterID, limit)
}

type identifierFixture struct {
	*tu.Env
	spy               *identifierStore
	admin, alice, bob domain.Actor
	net               *domain.Network
	aliceM, bobM      *domain.Member
}

func newIdentifierFixture(t *testing.T) *identifierFixture {
	var spy *identifierStore
	e := tu.NewEnvWith(t, func(st store.Store) store.Store { spy = &identifierStore{Store: st}; return spy })
	f := &identifierFixture{Env: e, spy: spy}
	f.admin = e.Person("admin")
	f.net, _ = e.Network(f.admin, "town", func(p *domain.Policy) { p.InactivityDays = 90 })
	f.alice, f.aliceM = e.Member(f.admin, f.net, "alice")
	f.bob, f.bobM = e.Member(f.admin, f.net, "bob")
	return f
}
func identifierError[T any](_ T, err error) error { return err }

// SEC-024 / A2-16: malformed references never reach a repository method, and
// authorization retains its precedence over target-format refusal.
func TestRecordIdentifiersRejectedBeforeRepositoryAccess(t *testing.T) {
	f := newIdentifierFixture(t)
	s, c, n := f.Svc, f.Ctx, f.net.ID
	const bad = "x' OR '1'='1"
	cases := []struct {
		name string
		call func(domain.Actor, string) error
		want domain.Code
	}{
		{"member", func(a domain.Actor, id string) error { return identifierError(s.GetMember(c, a, n, id)) }, domain.CodeNotFound},
		{"listing", func(a domain.Actor, id string) error { return identifierError(s.GetListing(c, a, n, id)) }, domain.CodeNotFound},
		{"transaction", func(a domain.Actor, id string) error { return identifierError(s.GetTransaction(c, a, n, id)) }, domain.CodeNotFound},
		{"transaction_dispute", func(a domain.Actor, id string) error { return identifierError(s.GetDisputeForTx(c, a, n, id)) }, domain.CodeNotFound},
		{"reversals", func(a domain.Actor, id string) error { return identifierError(s.ReversalsOf(c, a, n, id)) }, domain.CodeNotFound},
		{"dispute", func(a domain.Actor, id string) error { return identifierError(s.GetDispute(c, a, n, id)) }, domain.CodeNotFound},
		{"counterparty", func(a domain.Actor, id string) error {
			return identifierError(s.ProposeTransaction(c, a, n, service.ProposeInput{CounterpartyID: id, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"listing_reference", func(a domain.Actor, id string) error {
			return identifierError(s.ProposeTransaction(c, a, n, service.ProposeInput{CounterpartyID: f.bobM.ID, ListingID: id, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: f.Key()}))
		}, domain.CodeValidation},
		{"accept", func(a domain.Actor, id string) error {
			return identifierError(s.AcceptTransaction(c, a, n, service.TxRef{TxID: id, IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"decline", func(a domain.Actor, id string) error {
			return identifierError(s.DeclineTransaction(c, a, n, service.TxRef{TxID: id, IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"open_dispute", func(a domain.Actor, id string) error {
			return identifierError(s.OpenDispute(c, a, n, service.OpenDisputeInput{TxID: id, Category: "other", Statement: "test", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"review_dispute", func(a domain.Actor, id string) error { return identifierError(s.StartDisputeReview(c, a, n, id)) }, domain.CodeNotFound},
		{"withdraw_dispute", func(a domain.Actor, id string) error { return identifierError(s.WithdrawDispute(c, a, n, id)) }, domain.CodeNotFound},
		{"respond_dispute", func(a domain.Actor, id string) error {
			return identifierError(s.RespondToDispute(c, a, n, service.RespondInput{DisputeID: id, Response: "response", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"request_rereview", func(a domain.Actor, id string) error {
			return identifierError(s.RequestReReview(c, a, n, service.ReReviewRequestInput{DisputeID: id, Reason: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"decide_rereview", func(a domain.Actor, id string) error {
			return identifierError(s.DecideReReview(c, a, n, service.ReReviewDecideInput{DisputeID: id, Outcome: "confirmed", Rationale: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"resolve_dispute", func(a domain.Actor, id string) error {
			return identifierError(s.ResolveDispute(c, a, n, service.ResolveInput{DisputeID: id, Outcome: domain.OutcomeDismissed, Rationale: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"balance", func(a domain.Actor, id string) error { return identifierError(s.AdminGetMemberBalance(c, a, n, id)) }, domain.CodeNotFound},
		{"peek_balance", func(a domain.Actor, id string) error { return identifierError(s.PeekMemberBalance(c, a, n, id)) }, domain.CodeNotFound},
		{"suspend", func(a domain.Actor, id string) error { return s.SuspendMember(c, a, n, id, "reason") }, domain.CodeNotFound},
		{"approve_member", func(a domain.Actor, id string) error {
			return identifierError(s.DecideJoinRequest(c, a, n, id, true, "reason"))
		}, domain.CodeNotFound},
		{"revoke_invitation", func(a domain.Actor, id string) error { return s.RevokeInvitation(c, a, n, id) }, domain.CodeNotFound},
		{"revoke_token", func(a domain.Actor, id string) error { return s.RevokeAPIToken(c, a, id) }, domain.CodeNotFound},
		{"token_network", func(a domain.Actor, id string) error {
			_, _, err := s.CreateAPIToken(c, a, service.TokenInput{Name: "test", NetworkID: id, Scopes: []domain.Scope{domain.ScopeRead}})
			return err
		}, domain.CodeNotFound},
		{"report_member", func(a domain.Actor, id string) error {
			return identifierError(s.CreateReport(c, a, n, service.ReportInput{TargetType: service.ReportMember, TargetID: id, Category: "other", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"report_listing", func(a domain.Actor, id string) error {
			return identifierError(s.CreateReport(c, a, n, service.ReportInput{TargetType: service.ReportListing, TargetID: id, Category: "other", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"handle_report", func(a domain.Actor, id string) error {
			return identifierError(s.HandleReport(c, a, n, service.HandleReportInput{ReportID: id, Note: "handled", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"announcement_reference", func(a domain.Actor, id string) error {
			return identifierError(s.CreateAnnouncement(c, a, n, service.AnnouncementInput{Title: "Title", Body: "Body", SupersedesID: id, IdempotencyKey: f.Key()}))
		}, domain.CodeValidation},
		{"retract_announcement", func(a domain.Actor, id string) error { return s.DeleteAnnouncement(c, a, n, id) }, domain.CodeNotFound},
		{"challenge_subject", func(a domain.Actor, id string) error {
			return identifierError(s.OpenChallenge(c, a, n, service.ChallengeInput{Kind: service.ChallengeListingRemoval, SubjectID: id, Reason: "reason"}))
		}, domain.CodeNotFound},
		{"decide_challenge", func(a domain.Actor, id string) error {
			return identifierError(s.DecideChallenge(c, a, n, service.ChallengeDecision{ID: id, Outcome: "rejected", Rationale: "reason"}))
		}, domain.CodeNotFound},
		{"update_listing", func(a domain.Actor, id string) error {
			return identifierError(s.UpdateListing(c, a, n, service.ListingUpdate{ListingID: id}))
		}, domain.CodeNotFound},
		{"anonymise_member", func(a domain.Actor, id string) error {
			return identifierError(s.AnonymiseMember(c, a, n, id, "reason"))
		}, domain.CodeNotFound},
		{"notification_batch", func(a domain.Actor, id string) error {
			return identifierError(s.MarkNotificationsRead(c, a, n, []string{id}))
		}, domain.CodeValidation},
		{"member_after", func(a domain.Actor, id string) error {
			return identifierError(s.AdminMemberPage(c, a, n, nil, service.MemberPageQuery{AfterID: id}))
		}, domain.CodeValidation},
		{"member_before", func(a domain.Actor, id string) error {
			return identifierError(s.ListDirectoryPage(c, a, n, service.MemberPageQuery{BeforeID: id}))
		}, domain.CodeValidation},
		{"directory_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListDirectory(c, a, n, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"admin_member_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.AdminListMembers(c, a, n, nil, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"trade_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListTransactions(c, a, n, nil, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"admin_trade_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.AdminListTransactions(c, a, n, "", nil, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"pending_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListPendingForActor(c, a, n, service.PendingQuery{Page: service.Page{AfterID: id}}))
		}, domain.CodeValidation},
		{"dispute_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListDisputes(c, a, n, nil, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"find_dispute_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.FindDisputes(c, a, n, service.DisputeQuery{Page: service.Page{AfterID: id}}))
		}, domain.CodeValidation},
		{"find_dispute_transaction", func(a domain.Actor, id string) error {
			return identifierError(s.FindDisputes(c, a, n, service.DisputeQuery{TxID: id}))
		}, domain.CodeValidation},
		{"listing_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListListings(c, a, n, service.ListingQuery{Page: service.Page{AfterID: id}}))
		}, domain.CodeValidation},
		{"listing_member_filter", func(a domain.Actor, id string) error {
			return identifierError(s.ListListings(c, a, n, service.ListingQuery{MemberID: id}))
		}, domain.CodeValidation},
		{"notification_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListNotifications(c, a, n, service.NotificationQuery{Page: service.Page{AfterID: id}}))
		}, domain.CodeValidation},
		{"report_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListReports(c, a, n, "", service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"announcement_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListAnnouncements(c, a, n, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"challenge_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListChallengesPage(c, a, n, "", service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"breakglass_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListBreakGlass(c, a, n, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"inactivity_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListInactiveMembers(c, a, n, service.Page{AfterID: id}))
		}, domain.CodeValidation},

		{"adjustment_member", func(a domain.Actor, id string) error {
			return identifierError(s.CreateAdjustmentRequest(c, a, n, service.AdjustInput{MemberID: id, Amount: 1, Direction: service.AdjustCredit, Reason: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"adjustment_relation", func(a domain.Actor, id string) error {
			return identifierError(s.CreateAdjustmentRequest(c, a, n, service.AdjustInput{MemberID: f.bobM.ID, RelatedTxID: id, Amount: 1, Direction: service.AdjustCredit, Reason: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeValidation},
		{"pending_adjustment", func(a domain.Actor, id string) error { return identifierError(s.GetPendingAdjustment(c, a, n, id)) }, domain.CodeNotFound},
		{"approve_adjustment", func(a domain.Actor, id string) error { return identifierError(s.ApproveAdjustment(c, a, n, id)) }, domain.CodeNotFound},
		{"cancel_adjustment", func(a domain.Actor, id string) error { return identifierError(s.CancelAdjustment(c, a, n, id)) }, domain.CodeNotFound},
		{"adjustment_cursor", func(a domain.Actor, id string) error {
			return identifierError(s.ListPendingAdjustments(c, a, n, nil, service.Page{AfterID: id}))
		}, domain.CodeValidation},
		{"reverse", func(a domain.Actor, id string) error {
			return identifierError(s.ReverseTransaction(c, a, n, service.ReverseInput{TxID: id, Reason: "reason", IdempotencyKey: f.Key()}))
		}, domain.CodeNotFound},
		{"depart_member", func(a domain.Actor, id string) error {
			return identifierError(s.DepartMember(c, a, n, service.DepartInput{MemberID: id, ReasonCode: "confirmed_fraud", Reason: "reason"}))
		}, domain.CodeNotFound},
		{"depart_matter", func(a domain.Actor, id string) error {
			return identifierError(s.DepartMember(c, a, n, service.DepartInput{MemberID: f.bobM.ID, MatterTxID: id, ReasonCode: "confirmed_fraud", Reason: "reason"}))
		}, domain.CodeValidation},
	}
	before := tu.Snapshot(t, f.Store.DB())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(f.admin, bad)
			if domain.CodeOf(err) != tc.want {
				t.Fatalf("code=%q want=%q err=%v", domain.CodeOf(err), tc.want, err)
			}
			if got := f.spy.count(bad); got != 0 {
				t.Fatalf("malformed target reached repository %d times", got)
			}
			readOnly := f.admin
			readOnly.Scopes = []domain.Scope{domain.ScopeRead}
			// Every mutation must still reject the missing write scope before target validation.
			if !strings.Contains(tc.name, "cursor") && tc.want == domain.CodeNotFound && tc.name != "member" && tc.name != "listing" && tc.name != "transaction" && tc.name != "transaction_dispute" && tc.name != "reversals" && tc.name != "dispute" && tc.name != "balance" && tc.name != "peek_balance" {
				if got := domain.CodeOf(tc.call(readOnly, bad)); got != domain.CodeForbidden {
					t.Fatalf("scope refusal moved after ID validation: %s", got)
				}
			}
		})
	}
	if diff := before.Changed(tu.Snapshot(t, f.Store.DB())); len(diff) > 0 {
		t.Fatalf("refused identifiers changed persistent state: %v", diff)
	}
	// Role checks also keep their old precedence over malformed targets/cursors.
	for _, call := range []func() error{
		func() error { return s.SuspendMember(c, f.alice, n, bad, "reason") },
		func() error {
			return identifierError(s.AdminMemberPage(c, f.alice, n, nil, service.MemberPageQuery{AfterID: bad}))
		},
		func() error { return identifierError(s.StartDisputeReview(c, f.alice, n, bad)) },
		func() error {
			return identifierError(s.CreateAdjustmentRequest(c, f.alice, n, service.AdjustInput{MemberID: bad, Amount: 1, Direction: service.AdjustCredit, Reason: "reason", IdempotencyKey: f.Key()}))
		},
	} {
		if got := domain.CodeOf(call()); got != domain.CodeForbidden {
			t.Fatalf("role refusal moved after ID validation: %s", got)
		}
	}
}

// SEC-024 / SA2-18: an invalid body reference retains the existing listing_id
// field error; direct resource reads retain opaque not-found responses.
func TestListingReferenceIdentifierErrorsRemainFieldScoped(t *testing.T) {
	f := newIdentifierFixture(t)
	other, _ := f.Network(f.admin, "other", nil)
	foreign := tu.Must(f.Svc.CreateListing(f.Ctx, f.admin, other.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Other network", IdempotencyKey: f.Key()}))
	hidden := tu.Must(f.Svc.CreateListing(f.Ctx, f.bob, f.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Hidden listing", IdempotencyKey: f.Key()}))
	paused := domain.ListingPaused
	tu.Must(f.Svc.UpdateListing(f.Ctx, f.bob, f.net.ID, service.ListingUpdate{ListingID: hidden.ID, Status: &paused}))
	const bad = "x' OR '1'='1"
	for _, id := range []string{bad, "018f0000-0000-7000-8000-ffffffffffff", foreign.ID, hidden.ID} {
		_, err := f.Svc.ProposeTransaction(f.Ctx, f.alice, f.net.ID, service.ProposeInput{CounterpartyID: f.bobM.ID, ListingID: id, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: f.Key()})
		de, ok := err.(*domain.Error)
		if !ok || de.Code != domain.CodeValidation || len(de.Fields) != 1 || de.Fields[0] != (domain.FieldError{Field: "listing_id", Detail: "unknown listing"}) {
			t.Fatalf("listing reference %q: %+v", id, err)
		}
		if _, err := f.Svc.GetListing(f.Ctx, f.alice, f.net.ID, id); domain.CodeOf(err) != domain.CodeNotFound {
			t.Fatalf("direct listing read %q: %v", id, err)
		}
	}
	readOnly := f.alice
	readOnly.Scopes = []domain.Scope{domain.ScopeRead}
	_, err := f.Svc.ProposeTransaction(f.Ctx, readOnly, f.net.ID, service.ProposeInput{CounterpartyID: f.bobM.ID, ListingID: bad, Direction: service.ProposerPays, Amount: 1, IdempotencyKey: f.Key()})
	if domain.CodeOf(err) != domain.CodeForbidden {
		t.Fatalf("scope must precede listing validation: %v", err)
	}
	if f.spy.count(bad) != 0 {
		t.Fatal("malformed listing reference reached repository")
	}
}
