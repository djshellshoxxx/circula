package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) setPolicy(mod func(*domain.Policy), notice *int, noticeReason string) (*domain.Network, error) {
	n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID))
	pol := n.Policy
	mod(&pol)
	return w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "why", NoticeDays: notice,
		NoticeReason: noticeReason, IdempotencyKey: w.Key()})
}

func iptr(n int) *int { return &n }

func (w *world) notifKinds(a domain.Actor) []string {
	var kinds []string
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, a, w.net.ID, service.NotificationQuery{})) {
		kinds = append(kinds, n.Kind)
	}
	return kinds
}

func hasNotifKind(kinds []string, k string) bool {
	for _, x := range kinds {
		if x == k {
			return true
		}
	}
	return false
}

// GOV-003: an adverse change waits out its notice; members are told; it can be cancelled.
func TestPolicyNoticePeriod(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000; p.DisputeWindowDays = 30 })
	lower := func(p *domain.Policy) { p.DefaultCreditLimit = 1000 }

	// 13 days under the 14-day rule is refused; a shortened notice needs a reason.
	_, err := w.setPolicy(lower, iptr(13), "")
	wantCode(t, err, domain.ErrValidation)
	if n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.admin, w.net.ID)); n.Policy.DefaultCreditLimit != 5000 {
		t.Fatalf("refused change took effect: %d", n.Policy.DefaultCreditLimit)
	}

	// Default notice: scheduled, not in force, announced to members.
	n := tu.Must(w.setPolicy(lower, nil, ""))
	if n.Policy.DefaultCreditLimit != 5000 {
		t.Fatalf("adverse change took effect at once: %d", n.Policy.DefaultCreditLimit)
	}
	if !hasNotifKind(w.notifKinds(w.a), service.NotifPolicyNotice) {
		t.Errorf("member not told: %v", w.notifKinds(w.a))
	}
	log := tu.Must(w.Svc.PolicyLog(w.Ctx, w.a, w.net.ID))
	if len(log) != 2 || !log[0].Pending || log[0].NoticeDays != 14 || log[0].Changes["default_credit_limit"] != int64(1000) {
		t.Fatalf("log = %+v", log)
	}
	if !log[0].EffectiveAt.Equal(w.Clock.Now().AddDate(0, 0, 14)) {
		t.Errorf("effective at %v", log[0].EffectiveAt)
	}

	// Another change while one is scheduled is refused.
	_, err = w.setPolicy(func(p *domain.Policy) { p.DisputeWindowDays = 10 }, nil, "")
	wantCode(t, err, domain.ErrInvalidState)

	// It takes effect once the notice has run out.
	w.Clock.Advance(14*24*time.Hour + time.Minute)
	if n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)); n.Policy.DefaultCreditLimit != 1000 || n.PolicyVersion != 2 {
		t.Fatalf("after notice: limit %d version %d", n.Policy.DefaultCreditLimit, n.PolicyVersion)
	}
	if log := tu.Must(w.Svc.PolicyLog(w.Ctx, w.a, w.net.ID)); log[0].Pending {
		t.Error("still pending")
	}

	// A shortened notice with a reason shows in the change log; a non-adverse change is immediate.
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.DisputeWindowDays = 5 }, iptr(3), "fraud rule fix"))
	log = tu.Must(w.Svc.PolicyLog(w.Ctx, w.a, w.net.ID))
	if log[0].NoticeDays != 3 || log[0].NoticeReason != "fraud rule fix" || !log[0].Pending {
		t.Errorf("shortened notice = %+v", log[0])
	}
	w.Clock.Advance(4 * 24 * time.Hour)
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.LimitWarningDays = 9 }, nil, ""))
	if n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)); n.Policy.LimitWarningDays != 9 || n.Policy.DisputeWindowDays != 5 {
		t.Errorf("immediate change: %+v", n.Policy)
	}
}

// GOV-003: restoring the current values cancels a scheduled change.
func TestPolicyNoticeCancel(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 })
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit = 100 }, nil, ""))
	tu.Must(w.setPolicy(func(p *domain.Policy) {}, nil, "")) // same values as in force
	log := tu.Must(w.Svc.PolicyLog(w.Ctx, w.a, w.net.ID))
	if log[0].Pending || !log[1].Withdrawn {
		t.Fatalf("log = %+v", log)
	}
	w.Clock.Advance(30 * 24 * time.Hour)
	if n := tu.Must(w.Svc.GetNetwork(w.Ctx, w.a, w.net.ID)); n.Policy.DefaultCreditLimit != 5000 {
		t.Errorf("cancelled change took effect: %d", n.Policy.DefaultCreditLimit)
	}
}

// GOV-009: terms are versioned, accepted at join, and gate trading.
func TestTermsGovernance(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 5000 })
	if v := tu.Must(w.Svc.CurrentTerms(w.Ctx, w.a, w.net.ID)); v.Current != nil || v.NeedsAcceptance() {
		t.Fatalf("no terms yet: %+v", v)
	}
	_, err := w.Svc.PublishTerms(w.Ctx, w.a, w.net.ID, "rules")
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.PublishTerms(w.Ctx, w.admin, w.net.ID, "  ")
	wantCode(t, err, domain.ErrValidation)
	t1 := tu.Must(w.Svc.PublishTerms(w.Ctx, w.admin, w.net.ID, "Be kind."))
	if t1.Version != 1 {
		t.Fatalf("version %d", t1.Version)
	}
	if !hasNotifKind(w.notifKinds(w.a), service.NotifTermsPublished) {
		t.Error("members not told of the terms")
	}

	// Reading is allowed; trading is not until accepted.
	if v := tu.Must(w.Svc.CurrentTerms(w.Ctx, w.a, w.net.ID)); !v.NeedsAcceptance() || v.Current.Body != "Be kind." {
		t.Fatalf("view = %+v", v)
	}
	tu.Must(w.Svc.GetBalance(w.Ctx, w.a, w.net.ID))
	_, err = w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrTermsRequired)

	wantCode(t, w.Svc.AcceptTerms(w.Ctx, w.a, w.net.ID, 7), domain.ErrValidation)
	noErr(t, w.Svc.AcceptTerms(w.Ctx, w.a, w.net.ID, 1))
	noErr(t, w.Svc.AcceptTerms(w.Ctx, w.a, w.net.ID, 1)) // idempotent
	tx := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	_, err = w.accept(w.b, tx.ID) // b has not accepted
	wantCode(t, err, domain.ErrTermsRequired)
	noErr(t, w.Svc.AcceptTerms(w.Ctx, w.b, w.net.ID, 1))
	tu.Must(w.accept(w.b, tx.ID))

	// A new version needs a new acceptance; unchanged text is refused.
	_, err = w.Svc.PublishTerms(w.Ctx, w.admin, w.net.ID, "Be kind.")
	wantCode(t, err, domain.ErrValidation)
	tu.Must(w.Svc.PublishTerms(w.Ctx, w.admin, w.net.ID, "Be kind and prompt."))
	_, err = w.propose(w.a, w.bM, service.ProposerPays, 100)
	wantCode(t, err, domain.ErrTermsRequired)
	if v := tu.Must(w.Svc.CurrentTerms(w.Ctx, w.a, w.net.ID)); v.Accepted != 1 || v.Current.Version != 2 {
		t.Errorf("view = %+v", v)
	}
	noErr(t, w.Svc.AcceptTerms(w.Ctx, w.a, w.net.ID, 2))
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))

	// Join: the displayed version must be accepted, and is recorded with the member.
	inv, _, err := w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got := tu.Must(w.Svc.InvitationTerms(w.Ctx, inv)); got == nil || got.Version != 2 {
		t.Fatalf("invitation terms = %+v", got)
	}
	c := w.Person("carl@example.org")
	_, err = w.Svc.RedeemInvitation(w.Ctx, c, inv, "Carl")
	wantCode(t, err, domain.ErrValidation)
	_, err = w.Svc.RedeemInvitationTerms(w.Ctx, c, inv, "Carl", 1)
	wantCode(t, err, domain.ErrValidation)
	cm := tu.Must(w.Svc.RedeemInvitationTerms(w.Ctx, c, inv, "Carl", 2))
	if v := tu.Must(w.Svc.CurrentTerms(w.Ctx, c, w.net.ID)); v.NeedsAcceptance() || v.Accepted != 2 {
		t.Errorf("joiner view = %+v (member %s)", v, cm.ID)
	}
	var n int
	_ = w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = 'terms.published'`).Scan(&n)
	if n != 2 {
		t.Errorf("terms.published audit events = %d", n)
	}
}

// FR-014 / GOV-010 / GOV-011: preview of leaving, and notices to the people affected.
func TestDeparturePreviewAndNotices(t *testing.T) {
	w := newWorld(t, nil)
	open := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 50))
	pv := tu.Must(w.Svc.PreviewDeparture(w.Ctx, w.a, w.net.ID))
	if pv.OpenProposals != 1 || len(pv.Blockers) != 0 || pv.Rule != domain.DepartRetain {
		t.Fatalf("preview = %+v", pv)
	}
	if pv := tu.Must(w.Svc.PreviewDeparture(w.Ctx, w.admin, w.net.ID)); len(pv.Blockers) != 1 || !strings.Contains(pv.Blockers[0], "administrator") {
		t.Errorf("admin preview = %+v", pv)
	}
	tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
	if w.txStatus(open.ID).Status != domain.TxCancelled || !hasNotifKind(w.notifKinds(w.b), service.NotifMemberLeft) {
		t.Errorf("counterparty not told: %v", w.notifKinds(w.b))
	}
	_, err := w.Svc.PreviewDeparture(w.Ctx, w.a, w.net.ID)
	wantCode(t, err, domain.ErrInvalidState)

	tu.Must(w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bM.ID, Reason: "abuse"}))
	if !hasNotifKind(w.notifKinds(w.b), service.NotifMemberRemoved) {
		t.Errorf("removed member not told: %v", w.notifKinds(w.b))
	}
}

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
