package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-012: actions that change a member's rights, obligations or balance notify that member,
// with fixed text that never carries the administrator's reason.
func TestAdminActionsNotifyMember(t *testing.T) {
	w := newWorld(t, nil)
	const secret = "SECRET-REASON-TEXT"
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, secret))
	tu.Must(0, w.Svc.RestoreMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, secret))
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.aM.ID, domain.RoleReviewer))
	lim := int64(5000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lim, Reason: secret}))
	tu.Must(w.Svc.CreateAdjustment(w.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aM.ID, Amount: 100, Direction: service.AdjustCredit,
		Reason: secret, IdempotencyKey: w.Key()}))
	tx := w.Trade(w.net.ID, w.b, w.a, w.bM, w.aM, 300)
	tu.Must(w.Svc.ReverseTransaction(w.Ctx, w.admin, w.net.ID, service.ReverseInput{TxID: tx.ID, Reason: secret, IdempotencyKey: w.Key()}))
	for _, k := range []string{service.NotifAdminSuspended, service.NotifAdminReinstate, service.NotifAdminRole,
		service.NotifAdminLimit, service.NotifAdminAdjust, service.NotifAdminReversal} {
		if !hasKind(w.inbox(w.a), k) {
			t.Errorf("member not notified of %s", k)
		}
	}
	for _, n := range tu.Must(w.Svc.ListNotifications(w.Ctx, w.a, w.net.ID, service.NotificationQuery{})) {
		if strings.Contains(n.Title+n.Body, secret) {
			t.Errorf("notice leaks the administrator's reason: %+v", n)
		}
	}
	// SEC-012: removal of a member produces a notice to that member.
	tu.Must(w.Svc.DepartMember(w.Ctx, w.admin, w.net.ID, service.DepartInput{MemberID: w.bM.ID, Reason: "rules"}))
	if !hasKind(w.inbox(w.b), service.NotifAdminRemoved) {
		t.Error("removed member not notified")
	}
}
