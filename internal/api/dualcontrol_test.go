package api_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type pendingResp struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	AmountMinor   string  `json:"amount_minor"`
	TransactionID *string `json:"transaction_id"`
}

func TestDualControlAPI(t *testing.T) {
	w := newWorld(t)
	adm2, adm2M := w.env.Member(w.admin, w.net, "admin2")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, adm2M.ID, domain.RoleAdmin))
	t2 := w.token(adm2, domain.ScopeRead, domain.ScopeWrite)

	// Enabling through the policy API needs two administrators (present); it is audited like any policy change.
	rec := w.get(w.adminT, "/admin/policy")
	etag := rec.Header().Get("ETag")
	patch := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT,
		body: m{"dual_control_threshold_minor": "1000", "reason": "dual control"}, hdr: map[string]string{"If-Match": etag}})
	wantStatus(t, patch, 200)
	if got := decode[struct {
		Policy map[string]any `json:"policy"`
	}](t, patch).Policy["dual_control_threshold_minor"]; got != "1000" {
		t.Fatalf("threshold = %v", got)
	}

	body := m{"member_id": w.bobM.ID, "direction": "credit", "amount_minor": "5000", "reason": "big"}
	rec = w.post(w.adminT, "/admin/adjustments", body, w.key())
	wantStatus(t, rec, 202)
	pa := decode[pendingResp](t, rec)
	if pa.Status != "pending" || pa.AmountMinor != "5000" || rec.Header().Get("Location") == "" {
		t.Fatalf("pending = %+v", pa)
	}
	if a := w.acct(w.bobT); a.BalanceMinor != "0" {
		t.Fatalf("posted early: %+v", a)
	}
	// Below the threshold still returns 201 immediately.
	wantStatus(t, w.post(w.adminT, "/admin/adjustments", m{"member_id": w.bobM.ID, "direction": "credit", "amount_minor": "1000", "reason": "small"}, w.key()), 201)

	wantProblem(t, w.post(w.adminT, "/admin/adjustments/"+pa.ID+"/approve", nil, ""), 403, "forbidden") // proposer
	wantProblem(t, w.post(w.bobT, "/admin/adjustments/"+pa.ID+"/approve", nil, ""), 403, "forbidden")   // member
	wantProblem(t, w.post(t2, "/admin/adjustments/nope/approve", nil, ""), 404, "not-found")
	lst := decode[page](t, w.get(t2, "/admin/adjustments?status=pending"))
	if len(lst.Data) != 1 {
		t.Fatalf("list = %+v", lst)
	}
	wantProblem(t, w.get(t2, "/admin/adjustments?status=weird"), 400, "invalid-query")

	wantStatus(t, w.post(t2, "/admin/adjustments/"+pa.ID+"/approve", nil, ""), 200)
	if a := w.acct(w.bobT); a.BalanceMinor != "6000" {
		t.Fatalf("after approve: %+v", a)
	}
	got := decode[pendingResp](t, w.get(w.adminT, "/admin/adjustments/"+pa.ID))
	if got.Status != "approved" || got.TransactionID == nil {
		t.Fatalf("after = %+v", got)
	}
	wantProblem(t, w.post(t2, "/admin/adjustments/"+pa.ID+"/reject", nil, ""), 409, "invalid-transaction-state")

	// The legacy single-step service call fails safe above the threshold.
	_, err := w.env.Svc.CreateAdjustment(w.env.Ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.bobM.ID, Amount: 5000,
		Direction: service.AdjustCredit, Reason: "x", IdempotencyKey: w.env.Key()})
	if domain.CodeOf(err) != domain.CodeApprovalRequired {
		t.Errorf("legacy call: %v", err)
	}
}
