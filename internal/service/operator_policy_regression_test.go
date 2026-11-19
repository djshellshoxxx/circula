package service_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A1-08 / A3-F11: an operator threshold edit must neither collide with a
// scheduled version nor cancel its unrelated changes or be reverted by it.
func TestOperatorDualControlPreservesScheduledPolicy(t *testing.T) {
	d := newDC(t)
	tu.Must(d.setPolicy(func(p *domain.Policy) {
		p.DefaultCreditLimit = 5000
		p.DisputeWindowDays = 3
	}, iptr(7), "announced shorter notice"))
	before := operatorPolicyRows(t, d.world)
	scheduled := before[len(before)-1]
	beforeNotices := len(tu.Must(d.Svc.ListNotifications(d.Ctx, d.a, d.net.ID, service.NotificationQuery{})))
	d.Clock.Advance(2 * 24 * time.Hour)

	for _, threshold := range []int64{5000, 0, 2000} {
		noErr(t, d.Svc.OperatorSetDualControl(d.Ctx, domain.SystemActor("operator-test"), d.net.ID, threshold, "threshold correction"))
		current := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
		if current.Policy.DefaultCreditLimit != 100000 || current.Policy.DisputeWindowDays == 3 || current.Policy.DualControlThreshold != threshold {
			t.Fatalf("immediate policy changed unrelated values: %+v", current.Policy)
		}
		rows := operatorPolicyRows(t, d.world)
		if !reflect.DeepEqual(rows[:len(before)], before) {
			t.Fatal("operator edit changed existing immutable policy rows")
		}
		replacement := rows[len(rows)-1]
		want := scheduled.Policy
		want.DualControlThreshold = threshold
		if replacement.Policy != want || !replacement.EffectiveAt.Equal(scheduled.EffectiveAt) || !replacement.AnnouncedAt.Equal(scheduled.AnnouncedAt) || replacement.NoticeDays != scheduled.NoticeDays || replacement.NoticeReason != scheduled.NoticeReason {
			t.Fatalf("scheduled policy/notice not preserved: %+v", replacement)
		}
		if replacement.ChangedBy != "" || !strings.HasPrefix(replacement.Reason, "operator: ") {
			t.Fatalf("operator-authored revision attributed to a member: %+v", replacement)
		}
		log := tu.Must(d.Svc.PolicyLog(d.Ctx, d.a, d.net.ID))
		pending := 0
		for _, entry := range log {
			if entry.Pending {
				pending++
				if entry.Version != replacement.Version {
					t.Fatalf("old pending version remains live: %+v", entry)
				}
			}
		}
		if pending != 1 {
			t.Fatalf("pending policy count = %d", pending)
		}
	}
	if after := len(tu.Must(d.Svc.ListNotifications(d.Ctx, d.a, d.net.ID, service.NotificationQuery{}))); after != beforeNotices {
		t.Fatal("preserving the original notice duplicated notifications")
	}
	d.Clock.Advance(5 * 24 * time.Hour)
	current := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
	if current.Policy.DefaultCreditLimit != 5000 || current.Policy.DisputeWindowDays != 3 || current.Policy.DualControlThreshold != 2000 {
		t.Fatalf("scheduled activation lost policy or operator threshold: %+v", current.Policy)
	}
	if report := tu.Must(d.Svc.VerifyLedger(d.Ctx, d.admin, d.net.ID)); !report.OK {
		t.Fatalf("verification after policy replacement: %+v", report)
	}
}

func TestOperatorDualControlAfterCancelledPolicy(t *testing.T) {
	d := newDC(t)
	tu.Must(d.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit = 5000 }, nil, ""))
	tu.Must(d.setPolicy(func(*domain.Policy) {}, nil, ""))
	before := operatorPolicyRows(t, d.world)
	noErr(t, d.Svc.OperatorSetDualControl(d.Ctx, domain.SystemActor("operator-test"), d.net.ID, 0, "disable"))
	rows := operatorPolicyRows(t, d.world)
	if len(rows) != len(before)+1 {
		t.Fatalf("cancelled policy was recreated: %d -> %d", len(before), len(rows))
	}
	d.Clock.Advance(30 * 24 * time.Hour)
	current := tu.Must(d.Svc.GetNetwork(d.Ctx, d.admin, d.net.ID))
	if current.Policy.DefaultCreditLimit != 100000 || current.Policy.DualControlThreshold != 0 {
		t.Fatalf("cancelled policy became effective: %+v", current.Policy)
	}
}

func operatorPolicyRows(t *testing.T, w *world) []store.PolicyVersionRow {
	t.Helper()
	var rows []store.PolicyVersionRow
	if err := w.Store.View(w.Ctx, func(tx store.Tx) error {
		var err error
		rows, err = tx.(store.GovRepo).ListPolicyVersions(w.net.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return rows
}
