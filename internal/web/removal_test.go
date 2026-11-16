package web

import (
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// GOV-011 in the web interface: the removal form offers exactly the documented
// reasons; removing without one is refused with a field error; a removal for
// fraud naming a trade the administrator took part in is refused as a
// conflict; a listed reason with optional detail removes the member.
func TestRemoveMemberNeedsListedReasonWeb(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	path := b + "/admin/members/" + w.aliceM.ID + "/remove"
	status := func() domain.MemberStatus {
		m, err := w.env.Svc.GetMember(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID)
		if err != nil {
			t.Fatal(err)
		}
		return m.Status
	}

	f := adm.get(path)
	adm.mustContain(f, `<select id="f-reason_code" name="reason_code"`, "Reason for removal", `<option value="" selected>Choose a reason</option>`,
		`value="rule_breach">Breach of network rules`, `value="confirmed_fraud">Confirmed fraud`,
		`value="dispute_non_response">Prolonged non-response to a dispute`, `name="transaction_id"`, "Details (optional)")
	checkPage(t, "remove member", f)

	for _, vals := range []url.Values{{"reason": {"left town"}}, {"reason_code": {"moved_away"}, "reason": {"left town"}}} {
		bad := adm.post(path, f, vals)
		if bad.Status != 422 {
			t.Fatalf("%v: status %d", vals, bad.Status)
		}
		adm.mustContain(bad, `id="error-summary"`, `href="#f-reason_code"`, "Reason for removal")
		checkPage(t, "remove member error", bad)
	}

	own := w.env.Trade(w.net.ID, w.alice, w.admin, w.aliceM, w.adminM, 100)
	conflict := adm.post(path, f, url.Values{"reason_code": {"confirmed_fraud"}, "transaction_id": {own.ID}})
	if conflict.Status != 403 {
		t.Fatalf("conflict: status %d", conflict.Status)
	}
	adm.mustContain(conflict, `id="error-summary"`, "another administrator must decide", `value="confirmed_fraud" selected`)
	checkPage(t, "remove member conflict", conflict)
	if st := status(); st != domain.MemberActive {
		t.Fatalf("refused removals changed the member: %s", st)
	}

	done := adm.post(path, f, url.Values{"reason_code": {"rule_breach"}, "reason": {"minutes item 4"}})
	adm.mustContain(done, "Remove: done for alice@example.org")
	if st := status(); st != domain.MemberDeparted {
		t.Fatalf("after removal: %s", st)
	}
}
