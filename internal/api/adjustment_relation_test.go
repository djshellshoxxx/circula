package api_test

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
	"testing"
	"time"
)

func TestAdjustmentRelationAPI(t *testing.T) {
	w := newWorld(t)
	original := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	adm2, adm2M := w.env.Member(w.admin, w.net, "second-admin")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, domain.SystemActor("fixture"), w.net.ID, adm2M.ID, domain.RoleAdmin))
	w.env.Clock.Advance(25 * time.Hour)
	pol := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)).Policy
	pol.DualControlThreshold = 1000
	testutil.Must(w.env.Svc.UpdatePolicy(w.env.Ctx, w.admin, w.net.ID, service.PolicyChange{Policy: pol, Reason: "two admins", IdempotencyKey: w.env.Key()}))
	second := w.token(adm2, domain.ScopeRead, domain.ScopeWrite)
	for _, amount := range []string{"100", "2000"} {
		body := m{"member_id": w.aliceM.ID, "direction": "credit", "amount_minor": amount, "reason": "correction", "related_transaction_id": original.ID}
		key := w.key()
		res := w.post(w.adminT, "/admin/adjustments", body, key)
		code := 201
		if amount == "2000" {
			code = 202
		}
		wantStatus(t, res, code)
		row := decode[m](t, res)
		if row["related_transaction_id"] != original.ID {
			t.Fatalf("missing response relation: %+v", row)
		}
		replay := decode[m](t, w.post(w.adminT, "/admin/adjustments", body, key))
		if replay["id"] != row["id"] {
			t.Fatal("duplicate request")
		}
		body["related_transaction_id"] = ""
		wantProblem(t, w.post(w.adminT, "/admin/adjustments", body, key), 422, "idempotency-key-reused")
		if code == 202 {
			pending := decode[m](t, w.get(second, "/admin/adjustments/"+row["id"].(string)))
			if pending["related_transaction_id"] != original.ID {
				t.Fatal("pending read lost relation")
			}
			approved := w.post(second, "/admin/adjustments/"+row["id"].(string)+"/approve", nil, "")
			wantStatus(t, approved, 200)
			row = decode[m](t, approved)
		}
		if row["related_transaction_id"] != original.ID || row["original_transaction_id"] != nil {
			t.Fatalf("approval relation/reversal wrong: %+v", row)
		}
		id := row["id"].(string)
		own := decode[m](t, w.get(w.aliceT, "/transactions/"+id))
		if own["related_transaction_id"] != original.ID {
			t.Fatal("member detail lost relation")
		}
		wantProblem(t, w.get(w.bobT, "/transactions/"+id), 404, "not-found")
	}
	body := m{"member_id": w.aliceM.ID, "direction": "credit", "amount_minor": "1", "reason": "standalone"}
	standalone := w.post(w.adminT, "/admin/adjustments", body, w.key())
	wantStatus(t, standalone, 201)
	if _, ok := decode[m](t, standalone)["related_transaction_id"]; ok {
		t.Fatal("standalone response changed shape")
	}
	body["related_transaction_id"] = "missing"
	wantProblem(t, w.post(w.adminT, "/admin/adjustments", body, w.key()), 422, "validation-failed")
}
