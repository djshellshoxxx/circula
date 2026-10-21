package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func (w *world) inbox(a domain.Actor) []string {
	var kinds []string
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, a, w.net.ID, service.NotificationQuery{})) {
		kinds = append(kinds, n.Kind)
	}
	return kinds
}

func hasKind(kinds []string, k string) bool {
	for _, x := range kinds {
		if x == k {
			return true
		}
	}
	return false
}

// FR-038: trade events notify the right member, with mark-read and unread count.
func TestNotificationsTradeFlow(t *testing.T) {
	w := newWorld(t, nil)
	p := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 500))
	if got := w.inbox(w.b); len(got) != 1 || got[0] != service.NotifTradeProposed {
		t.Fatalf("b inbox = %v", got)
	}
	if got := w.inbox(w.a); len(got) != 0 {
		t.Fatalf("proposer notified of own proposal: %v", got)
	}
	if n := tu.Must(w.Svc.CountUnreadNotifications(w.Ctx, w.b, w.net.ID)); n != 1 {
		t.Errorf("unread = %d", n)
	}
	tu.Must(w.accept(w.b, p.ID))
	if got := w.inbox(w.a); len(got) != 1 || got[0] != service.NotifTradeAccepted {
		t.Fatalf("a inbox = %v", got)
	}
	p2 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: p2.ID, IdempotencyKey: w.Key()}))
	if !hasKind(w.inbox(w.a), service.NotifTradeDeclined) {
		t.Error("decline not notified")
	}
	p3 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	tu.Must(w.Svc.CancelTransaction(w.Ctx, w.a, w.net.ID, service.CancelInput{TxID: p3.ID, IdempotencyKey: w.Key()}))
	if !hasKind(w.inbox(w.b), service.NotifTradeCancelled) {
		t.Error("cancel not notified")
	}
}

func TestNotificationsMarkRead(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 500))
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 600))
	all := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{}))
	if len(all) != 2 || all[0].ReadAt != nil {
		t.Fatalf("inbox = %+v", all)
	}
	// someone else's ids and junk are ignored
	if n := tu.Must(w.Svc.MarkNotificationsRead(w.Ctx, w.a, w.net.ID, []string{all[0].ID, "nope"})); n != 0 {
		t.Errorf("a marked b's notification: %d", n)
	}
	if n := tu.Must(w.Svc.CountUnreadNotifications(w.Ctx, w.b, w.net.ID)); n != 2 {
		t.Errorf("unread = %d", n)
	}
	if n := tu.Must(w.Svc.MarkNotificationsRead(w.Ctx, w.b, w.net.ID, []string{all[0].ID})); n != 1 {
		t.Errorf("marked %d", n)
	}
	unread := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{UnreadOnly: true}))
	if len(unread) != 1 || unread[0].ID != all[1].ID {
		t.Errorf("unread = %+v", unread)
	}
	// repeat is a no-op; empty list marks all
	if n := tu.Must(w.Svc.MarkNotificationsRead(w.Ctx, w.b, w.net.ID, []string{all[0].ID})); n != 0 {
		t.Errorf("repeat marked %d", n)
	}
	if n := tu.Must(w.Svc.MarkNotificationsRead(w.Ctx, w.b, w.net.ID, nil)); n != 1 {
		t.Errorf("mark all marked %d", n)
	}
	if n := tu.Must(w.Svc.CountUnreadNotifications(w.Ctx, w.b, w.net.ID)); n != 0 {
		t.Errorf("unread = %d", n)
	}
	// paging
	page := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{Page: service.Page{Limit: 1}}))
	next := tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{Page: service.Page{Limit: 1, AfterID: page[0].ID}}))
	if len(page) != 1 || len(next) != 1 || next[0].ID == page[0].ID {
		t.Errorf("paging: %+v %+v", page, next)
	}
	// read-only token cannot mark; outsiders and pending members have no inbox
	ro := w.b
	ro.Scopes = []domain.Scope{domain.ScopeRead}
	_, err := w.Svc.MarkNotificationsRead(w.Ctx, ro, w.net.ID, nil)
	wantCode(t, err, domain.ErrForbidden)
	stranger := w.Person("stranger@example.org")
	_, err = w.Svc.ListNotifications(w.Ctx, stranger, w.net.ID, service.NotificationQuery{})
	wantCode(t, err, domain.ErrNotFound)
	pend, _ := w.joinPending("pending@example.org")
	_, err = w.Svc.ListNotifications(w.Ctx, pend, w.net.ID, service.NotificationQuery{})
	wantCode(t, err, domain.ErrForbidden)
	ids := make([]string, 201)
	_, err = w.Svc.MarkNotificationsRead(w.Ctx, w.b, w.net.ID, ids)
	wantCode(t, err, domain.ErrValidation)
}

func TestNotificationsDisputeFlow(t *testing.T) {
	w := newWorld(t, nil)
	rev, _ := w.reviewerAs("rev@example.org")
	d := tu.Must(w.open(w.a, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000).ID))
	if !hasKind(w.inbox(w.b), service.NotifDisputeOpened) {
		t.Error("other party not told of the dispute")
	}
	if hasKind(w.inbox(w.a), service.NotifDisputeOpened) {
		t.Error("opener told of own dispute")
	}
	aw := tu.Must(w.Svc.ListDisputesAwaitingMyResponse(w.Ctx, w.b, w.net.ID))
	if len(aw) != 1 || aw[0].ID != d.ID {
		t.Fatalf("awaiting = %+v", aw)
	}
	if aw := tu.Must(w.Svc.ListDisputesAwaitingMyResponse(w.Ctx, w.a, w.net.ID)); len(aw) != 0 {
		t.Errorf("opener awaiting = %+v", aw)
	}
	tu.Must(w.Svc.StartDisputeReview(w.Ctx, rev, w.net.ID, d.ID))
	if !hasKind(w.inbox(w.a), service.NotifDisputeReview) || !hasKind(w.inbox(w.b), service.NotifDisputeReview) {
		t.Error("review start not notified to both parties")
	}
	tu.Must(0, w.respond(w.b, d.ID, "answer"))
	if aw := tu.Must(w.Svc.ListDisputesAwaitingMyResponse(w.Ctx, w.b, w.net.ID)); len(aw) != 0 {
		t.Errorf("answered dispute still awaiting: %+v", aw)
	}
	tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "no action"))
	if !hasKind(w.inbox(w.a), service.NotifDisputeDecided) || !hasKind(w.inbox(w.b), service.NotifDisputeDecided) {
		t.Error("decision not notified to both parties")
	}
	if hasKind(w.inbox(rev), service.NotifDisputeDecided) {
		t.Error("the reviewer is notified of their own decision")
	}
	d2 := tu.Must(w.open(w.b, w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 10).ID))
	tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.b, w.net.ID, d2.ID))
	if !hasKind(w.inbox(w.a), service.NotifDisputeWithdraw) {
		t.Error("withdrawal not notified")
	}
}

func TestNotificationsRollBackWithFailedCommand(t *testing.T) {
	w := newWorld(t, nil)
	// An over-limit proposal is refused and must leave no notification behind.
	_, err := w.propose(w.a, w.bM, service.ProposerPays, 999_999)
	if err == nil {
		t.Fatal("expected refusal")
	}
	if got := w.inbox(w.b); len(got) != 0 {
		t.Errorf("notification survived a failed command: %v", got)
	}
}

// UX-012: a member at or above 80 percent of the limit gets one warning per week.
func TestRunLimitWarnings(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 10000 })
	c, cM := w.Member2(w.admin, "carol@example.org")
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 8000) // a: -8000 = 80%
	w.Trade(w.net.ID, c, w.b, cM, w.bM, 7900)     // c: -7900 = 79%: below

	n := tu.Must(w.Svc.RunLimitWarnings(w.Ctx))
	if n != 1 {
		t.Fatalf("warnings = %d, want 1", n)
	}
	if !hasKind(w.inbox(w.a), service.NotifLimitWarning) {
		t.Error("a not warned at 80%")
	}
	if hasKind(w.inbox(c), service.NotifLimitWarning) || hasKind(w.inbox(w.b), service.NotifLimitWarning) {
		t.Error("warned below threshold or in credit")
	}
	if n := tu.Must(w.Svc.RunLimitWarnings(w.Ctx)); n != 0 {
		t.Errorf("same week warned again: %d", n)
	}
	w.Clock.Advance(8 * 24 * time.Hour)
	if n := tu.Must(w.Svc.RunLimitWarnings(w.Ctx)); n != 1 {
		t.Errorf("next week: %d, want 1", n)
	}
	// repaying below the threshold stops warnings
	w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 5000)
	w.Clock.Advance(8 * 24 * time.Hour)
	if n := tu.Must(w.Svc.RunLimitWarnings(w.Ctx)); n != 0 {
		t.Errorf("after repayment: %d", n)
	}
}

func TestLimitWarningHelper(t *testing.T) {
	cases := []struct {
		bal, lim int64
		warn     bool
	}{{-8000, 10000, true}, {-7999, 10000, false}, {-10000, 10000, true}, {500, 10000, false}, {-100, 0, false}, {0, 10000, false}}
	for _, c := range cases {
		if _, got := service.LimitWarning(&service.BalanceView{Balance: c.bal, CreditLimit: c.lim}, 80); got != c.warn {
			t.Errorf("balance %d limit %d: warn=%v want %v", c.bal, c.lim, got, c.warn)
		}
	}
	if _, got := service.LimitWarning(nil, 80); got {
		t.Error("nil view warns")
	}
}
