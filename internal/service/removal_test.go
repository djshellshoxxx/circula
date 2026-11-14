package service_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// remove is an administrator removal of m (GOV-011).
func (w *world) remove(by domain.Actor, m *domain.Member, code, detail, matter string) (*domain.Member, error) {
	return w.Svc.DepartMember(w.Ctx, by, w.net.ID, service.DepartInput{MemberID: m.ID, ReasonCode: code, Reason: detail, MatterTxID: matter})
}

// departures returns the member.departed audit events about memberID.
func (w *world) departures(memberID string) []*domain.AuditEvent {
	w.T.Helper()
	var out []*domain.AuditEvent
	err := w.Store.View(w.Ctx, func(tx store.Tx) error {
		evs, err := tx.ListAudit(w.net.ID, store.AuditFilter{Action: domain.ActMemberDeparted, TargetID: memberID, Limit: 500})
		out = evs
		return err
	})
	if err != nil {
		w.T.Fatal(err)
	}
	return out
}

func (w *world) memberStatus(m *domain.Member) domain.MemberStatus {
	w.T.Helper()
	return tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, m.ID)).Status
}

func fieldRefused(err error, field string) bool {
	var de *domain.Error
	return errors.As(err, &de) && de.Code == domain.CodeValidation && len(de.Fields) == 1 && de.Fields[0].Field == field
}

// GOV-011: the documented list is exactly the one in the spec.
func TestRemovalReasonsAreTheDocumentedList(t *testing.T) {
	want := []string{"rule_breach", "confirmed_fraud", "dispute_non_response"}
	if !slices.Equal(domain.RemovalReasons, want) {
		t.Fatalf("removal reasons = %v, want %v", domain.RemovalReasons, want)
	}
	for _, c := range domain.RemovalReasons {
		if domain.RemovalReasonLabel(c) == "" {
			t.Errorf("%s has no label", c)
		}
	}
	if domain.RemovalReasonLabel("moved_away") != "" {
		t.Error("an unlisted code has a label")
	}
	if !domain.RemovalNeedsIndependence(domain.RemovalConfirmedFraud) || !domain.RemovalNeedsIndependence(domain.RemovalRuleBreach) ||
		domain.RemovalNeedsIndependence(domain.RemovalDisputeNonResponse) {
		t.Error("independence applies to fraud and breach only")
	}
}

// GOV-011 acceptance 1: a removal without a reason from the documented list
// is refused and changes nothing; every listed reason is accepted; the
// free-text detail is optional and bounded.
func TestRemovalWithoutListedReasonRefused(t *testing.T) {
	w := newWorld(t, nil)
	for _, in := range []struct{ code, detail string }{
		{"", ""}, {"", "left town"}, {"moved_away", "left town"}, {"Confirmed_Fraud", ""}, {" rule_breach", ""},
	} {
		_, err := w.remove(w.admin, w.aM, in.code, in.detail, "")
		if !fieldRefused(err, "reason_code") {
			t.Errorf("code %q detail %q: %v", in.code, in.detail, err)
		}
	}
	_, err := w.remove(w.admin, w.aM, domain.RemovalRuleBreach, strings.Repeat("x", 501), "")
	if !fieldRefused(err, "reason") {
		t.Errorf("over-long detail: %v", err)
	}
	if st := w.memberStatus(w.aM); st != domain.MemberActive {
		t.Fatalf("refused removals changed the member: %s", st)
	}
	if n := len(w.departures(w.aM.ID)); n != 0 {
		t.Fatalf("refused removals wrote %d departure events", n)
	}
	// Every documented reason is accepted, with or without detail.
	for i, code := range domain.RemovalReasons {
		_, m := w.Member2(w.admin, code+"@example.org")
		detail := ""
		if i%2 == 0 {
			detail = "minutes of the meeting, item 4"
		}
		got, err := w.remove(w.admin, m, code, detail, "")
		if err != nil || got.Status != domain.MemberDeparted {
			t.Fatalf("%s: %v", code, err)
		}
	}
}

// GOV-011 acceptance 2: removal produces a notice to the member and an audit
// event. The notice names the listed reason (fixed text), never the
// administrator's free-text detail; the audit event carries both.
func TestRemovalNoticeAndAudit(t *testing.T) {
	w := newWorld(t, nil)
	const detail = "SECRET-DETAIL-TEXT"
	tu.Must(w.remove(w.admin, w.bM, domain.RemovalConfirmedFraud, detail, ""))
	if k := w.inbox(w.b); !hasKind(k, service.NotifMemberRemoved) || !hasKind(k, service.NotifAdminRemoved) {
		t.Fatalf("removed member inbox = %v", k)
	}
	named := false
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, w.b, w.net.ID, service.NotificationQuery{})) {
		if strings.Contains(n.Title+n.Body, detail) {
			t.Errorf("notice leaks the administrator's detail: %+v", n)
		}
		named = named || strings.Contains(n.Body, domain.RemovalReasonLabel(domain.RemovalConfirmedFraud))
	}
	if !named {
		t.Error("no notice names the listed reason")
	}
	evs := w.departures(w.bM.ID)
	if len(evs) != 1 {
		t.Fatalf("departure events = %d", len(evs))
	}
	e := evs[0]
	var after map[string]any
	if err := json.Unmarshal([]byte(e.SummaryAfter), &after); err != nil {
		t.Fatal(err)
	}
	if e.ActorType != domain.ActorAdmin || e.ActorID != w.adminM.ID || after["reason_code"] != domain.RemovalConfirmedFraud ||
		after["status"] != string(domain.MemberDeparted) || !strings.Contains(e.Reason, detail) ||
		!strings.Contains(e.Reason, domain.RemovalReasonLabel(domain.RemovalConfirmedFraud)) {
		t.Fatalf("audit event = %+v", e)
	}
	if strings.Contains(e.SummaryAfter, detail) {
		t.Error("free text in the audit summary")
	}
	// Without detail the audit reason is the listed reason itself.
	tu.Must(w.remove(w.admin, w.aM, domain.RemovalRuleBreach, "", ""))
	if evs := w.departures(w.aM.ID); len(evs) != 1 || evs[0].Reason != domain.RemovalReasonLabel(domain.RemovalRuleBreach) {
		t.Fatalf("audit without detail = %+v", evs)
	}
	// Voluntary departure is unchanged: no reason code needed.
	c, cM := w.Member2(w.admin, "carol@example.org")
	if got := tu.Must(w.Svc.DepartMember(w.Ctx, c, w.net.ID, service.DepartInput{})); got.Status != domain.MemberDeparted || got.ID != cM.ID {
		t.Fatalf("voluntary departure: %+v", got)
	}
}

// GOV-011 acceptance 3 with SEC-010: a removal for fraud or breach is refused
// when the deciding administrator is the counterparty in the matter: a trade
// with the member that is under an undecided dispute or a pending re-review,
// or the trade the removal names as its matter. Another administrator may
// decide; a removal for another listed reason is not covered by the rule.
func TestRemovalByConflictedAdminRefused(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ReReviewWindowDays = 14 })
	adm2, adm2M := w.Member2(w.admin, "adm2@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	conflict := func(err error) bool { return errors.Is(err, domain.ErrForbidden) && service.IsRemovalConflict(err) }

	// (1) An undecided dispute on a trade between the administrator and B.
	tx := w.Trade(w.net.ID, w.admin, w.b, w.adminM, w.bM, 400)
	d := tu.Must(w.open(w.admin, tx.ID))
	for _, code := range []string{domain.RemovalConfirmedFraud, domain.RemovalRuleBreach} {
		if _, err := w.remove(w.admin, w.bM, code, "", ""); !conflict(err) {
			t.Fatalf("%s by the counterparty: %v", code, err)
		}
	}
	// The other listed reason is not a conflict case; the open dispute still blocks departure.
	_, err := w.remove(w.admin, w.bM, domain.RemovalDisputeNonResponse, "", "")
	wantCode(t, err, domain.ErrBlocked)
	// An independent administrator is not refused for a conflict (the dispute still blocks).
	_, err = w.remove(adm2, w.bM, domain.RemovalConfirmedFraud, "", "")
	wantCode(t, err, domain.ErrBlocked)
	if service.IsRemovalConflict(err) {
		t.Fatal("independent administrator refused as conflicted")
	}

	// (2) The dispute is decided, then the member asks for the one re-review:
	// the matter is still open and the trade is no longer disputed.
	rev, _ := w.reviewer()
	tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "no evidence"))
	noErr(t, w.rereview(w.b, d.ID, "please look again"))
	if st := w.txStatus(tx.ID).Status; st != domain.TxCompleted {
		t.Fatalf("precondition: trade %s", st)
	}
	if _, err := w.remove(w.admin, w.bM, domain.RemovalConfirmedFraud, "", ""); !conflict(err) {
		t.Fatalf("fraud removal by the counterparty during a re-review: %v", err)
	}
	if st := w.memberStatus(w.bM); st != domain.MemberActive || len(w.departures(w.bM.ID)) != 0 {
		t.Fatalf("refused removal changed the member: %s", st)
	}

	// (3) A named matter: the trade the removal is about.
	own := w.Trade(w.net.ID, w.a, w.admin, w.aM, w.adminM, 300)
	if _, err := w.remove(w.admin, w.aM, domain.RemovalRuleBreach, "", own.ID); !conflict(err) {
		t.Fatalf("breach removal naming the administrator's own trade: %v", err)
	}
	for _, ref := range []string{"no-such-trade", tx.ID} { // unknown, or not a trade of A
		if _, err := w.remove(w.admin, w.aM, domain.RemovalRuleBreach, "", ref); !fieldRefused(err, "transaction_id") {
			t.Fatalf("matter %q: %v", ref, err)
		}
	}
	if st := w.memberStatus(w.aM); st != domain.MemberActive || len(w.departures(w.aM.ID)) != 0 {
		t.Fatalf("refused removal changed the member: %s", st)
	}
	// The second administrator took no part in that trade and may decide.
	got := tu.Must(w.remove(adm2, w.aM, domain.RemovalRuleBreach, "", own.ID))
	if got.Status != domain.MemberDeparted {
		t.Fatalf("independent removal: %s", got.Status)
	}
	evs := w.departures(w.aM.ID)
	if len(evs) != 1 || !strings.Contains(evs[0].SummaryAfter, `"matter_transaction":"`+own.ID+`"`) {
		t.Fatalf("matter not recorded: %+v", evs)
	}
	// A trade the administrator took no part in is not a conflict.
	c, cM := w.Member2(w.admin, "carol@example.org")
	ct := w.Trade(w.net.ID, c, w.b, cM, w.bM, 50)
	if _, err := w.remove(w.admin, cM, domain.RemovalConfirmedFraud, "", ct.ID); err != nil {
		t.Fatalf("removal naming an unrelated trade: %v", err)
	}
}
