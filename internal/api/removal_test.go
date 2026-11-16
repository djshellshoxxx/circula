package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// GOV-011 through the JSON API: removal needs a reason_code from the
// documented list (free-text reason optional); a removal for fraud or breach
// naming a trade the administrator took part in is refused as a conflict.
func TestRemoveMemberNeedsListedReasonAPI(t *testing.T) {
	w := newWorld(t)
	path := "/admin/members/" + w.aliceM.ID + "/remove"
	for _, body := range []m{nil, {"reason": "left"}, {"reason_code": "moved_away"}, {"reason_code": ""}} {
		p := wantProblem(t, w.post(w.adminT, path, body, w.key()), 422, "validation-failed")
		if len(p.Errors) != 1 || p.Errors[0].Pointer != "/reason_code" {
			t.Fatalf("body %v: errors %+v", body, p.Errors)
		}
	}
	// The field belongs to removal only.
	wantProblem(t, w.post(w.adminT, "/admin/members/"+w.aliceM.ID+"/suspend", m{"reason": "x", "reason_code": "rule_breach"}, w.key()), 400, "invalid-body")

	own := w.env.Trade(w.net.ID, w.alice, w.admin, w.aliceM, w.adminM, 100)
	p := wantProblem(t, w.post(w.adminT, path, m{"reason_code": "confirmed_fraud", "transaction_id": own.ID}, w.key()), 403, "forbidden")
	if !strings.Contains(p.Detail, "conflict of interest") {
		t.Fatalf("detail = %q", p.Detail)
	}
	p = wantProblem(t, w.post(w.adminT, path, m{"reason_code": "confirmed_fraud", "transaction_id": "nope"}, w.key()), 422, "validation-failed")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/transaction_id" {
		t.Fatalf("errors %+v", p.Errors)
	}
	if got, err := w.env.Svc.GetMember(context.Background(), w.admin, w.net.ID, w.aliceM.ID); err != nil || got.Status != domain.MemberActive {
		t.Fatalf("refused removals changed the member: %+v %v", got, err)
	}

	rec := w.post(w.adminT, path, m{"reason_code": "dispute_non_response"}, w.key())
	wantStatus(t, rec, 200)
	if got := decode[struct {
		Status string `json:"status"`
	}](t, rec); got.Status != string(domain.MemberDeparted) {
		t.Fatalf("status = %s", got.Status)
	}
	rec = w.post(w.adminT, "/admin/members/"+w.bobM.ID+"/remove", m{"reason_code": "rule_breach", "reason": "minutes item 4", "transaction_id": own.ID}, w.key())
	wantProblem(t, rec, 422, "validation-failed") // bob is not a party to that trade
}
