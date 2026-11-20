package service_test

import (
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-01: a role grant must notify every other active administrator even
// when ordinary members fill the first store page. Pagination must not
// duplicate notices or add inactive, non-admin, grantee or foreign recipients.
func TestAdminGrantNoticesBeyondFirstMemberPage(t *testing.T) {
	w := newWorld(t, nil)
	inactive, inactiveM := w.Member(w.admin, w.net, "inactive@example.org")
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, inactiveM.ID, domain.RoleAdmin))
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, inactiveM.ID, "fixture inactive administrator"))
	foreign := w.Person("foreign@example.org")
	w.Network(foreign, "other-network", nil)

	var early domain.Actor
	// These valid fixture IDs sort before the normal deterministic IDs.
	// Seed rows directly to avoid 501 unnecessary password hashes and
	// unrelated invitation throughput limits.
	err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
		for i := 0; i < 501; i++ {
			pid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i)
			mid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 2*i+1)
			if err := tx.InsertPerson(&domain.Person{ID: pid, Login: fmt.Sprintf("filler%d@example.org", i), CreatedAt: w.Clock.Now()}, "unused-fixture"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: mid, NetworkID: w.net.ID, PersonID: pid,
				DisplayName: "Filler", Status: domain.MemberActive, JoinMethod: "invitation", JoinedAt: w.Clock.Now()}); err != nil {
				return err
			}
			if i == 0 {
				early = domain.Actor{PersonID: pid, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
				if err := tx.GrantRole("00000000-0000-4000-9000-000000000001", w.net.ID, mid, domain.RoleAdmin, w.adminM.ID, w.Clock.Now()); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	count := func(actor domain.Actor) int {
		t.Helper()
		n := 0
		for _, notice := range tu.Must(w.Svc.ListSecurityNotices(w.Ctx, actor, false)) {
			if notice.Kind == domain.NoticeAdminGranted {
				n++
			}
		}
		return n
	}
	cases := []struct {
		name   string
		actor  domain.Actor
		want   int
		before int
	}{
		{name: "administrator on first page", actor: early, want: 1},
		{name: "administrator after first page", actor: w.admin, want: 1},
		{name: "ordinary member", actor: w.a},
		{name: "grantee", actor: w.b},
		{name: "inactive administrator", actor: inactive},
		{name: "administrator of another network", actor: foreign},
	}
	for i := range cases {
		cases[i].before = count(cases[i].actor)
	}
	tu.Must(0, w.Svc.AssignRole(w.Ctx, w.admin, w.net.ID, w.bM.ID, domain.RoleAdmin))
	for _, tc := range cases {
		if got := count(tc.actor) - tc.before; got != tc.want {
			t.Errorf("%s received %d new admin-grant notices, want %d", tc.name, got, tc.want)
		}
	}
}
