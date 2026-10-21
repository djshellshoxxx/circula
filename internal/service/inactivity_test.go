package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

const day = 24 * time.Hour

// GOV-012: an inactive member is flagged and notified and remains a member.
func TestInactivityRuleFlagsAndNeverRemoves(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.InactivityDays = 60 })
	w.Clock.Advance(59 * day)
	if n := tu.Must(w.Svc.RunInactivityRule(w.Ctx)); n != 0 {
		t.Fatalf("flagged inside the period: %d", n)
	}
	w.Clock.Advance(day)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100) // a and b are active now
	w.Clock.Advance(25 * day)                    // day 85: admin idle 85 days, a and b 25
	n := tu.Must(w.Svc.RunInactivityRule(w.Ctx))
	if n != 1 {
		t.Fatalf("flagged = %d, want 1 (the administrator)", n)
	}
	if !hasKind(w.inbox(w.admin), service.NotifMemberInactive) {
		t.Error("inactive member not notified")
	}
	if hasKind(w.inbox(w.a), service.NotifMemberInactive) || hasKind(w.inbox(w.b), service.NotifMemberInactive) {
		t.Error("active members notified")
	}
	if n := tu.Must(w.Svc.RunInactivityRule(w.Ctx)); n != 0 {
		t.Errorf("second run flagged %d again", n)
	}
	// Still an active member, with the same balance and all rights.
	m := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.adminM.ID))
	if m.Status != domain.MemberActive {
		t.Errorf("status = %s", m.Status)
	}
	// The administrator sees the flagged list; a plain member does not.
	list := tu.Must(w.Svc.ListInactiveMembers(w.Ctx, w.admin, w.net.ID, service.Page{}))
	if len(list) != 1 || list[0].ID != w.adminM.ID {
		t.Errorf("list = %v", list)
	}
	if _, err := w.Svc.ListInactiveMembers(w.Ctx, w.a, w.net.ID, service.Page{}); err == nil {
		t.Error("member could list inactive members")
	}
}

// GOV-012: with no rule (the default) nobody is flagged however long they idle.
func TestInactivityDefaultIsNoRule(t *testing.T) {
	w := newWorld(t, nil)
	w.Clock.Advance(1000 * day)
	if n := tu.Must(w.Svc.RunInactivityRule(w.Ctx)); n != 0 {
		t.Fatalf("flagged %d without a rule", n)
	}
	if l := tu.Must(w.Svc.ListInactiveMembers(w.Ctx, w.admin, w.net.ID, service.Page{})); len(l) != 0 {
		t.Errorf("list = %v", l)
	}
}

// GOV-012: a sign-in counts as activity, and it still counts after the idle
// session itself has been purged.
func TestInactivityCountsSignInAfterSessionPurge(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.InactivityDays = 100 })
	w.Clock.Advance(80 * day)
	tu.Must2(w.Svc.Login(w.Ctx, "admin@example.org", "correct horse battery"))
	w.Clock.Advance(day)
	tu.Must(w.Svc.RunInactivityRule(w.Ctx)) // saves the activity
	w.Clock.Advance(40 * day)
	tu.Must(w.Svc.Purge(w.Ctx)) // the idle session is deleted
	w.Clock.Advance(29 * day)   // day 150: admin signed in 69 days ago, the others never
	n := tu.Must(w.Svc.RunInactivityRule(w.Ctx))
	if n != 2 {
		t.Fatalf("flagged = %d, want 2 (the two members who never signed in)", n)
	}
	if hasKind(w.inbox(w.admin), service.NotifMemberInactive) {
		t.Error("administrator who signed in was flagged")
	}
}

// The policy bound: 0 or 30 to 3650.
func TestInactivityDaysBounds(t *testing.T) {
	for days, ok := range map[int]bool{0: true, 29: false, 30: true, 3650: true, 3651: false, -1: false} {
		p := domain.DefaultPolicy()
		p.InactivityDays = days
		if err := p.Validate(); (err == nil) != ok {
			t.Errorf("inactivity_days %d: err=%v", days, err)
		}
	}
}
