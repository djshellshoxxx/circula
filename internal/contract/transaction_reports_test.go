package contract

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025: the new report target, relation field and dispute query all agree
// with the published contract, including the handled report response.
func TestTransactionReportsMatchOpenAPI(t *testing.T) {
	h := newHarness(t)
	t.Cleanup(func() { api.CloseHandler(h.h) })
	e := h.env
	admin := e.Person("administrator")
	net, _ := e.Network(admin, "transaction-report-contract", nil)
	alice, aliceM := e.Member(admin, net, "alice")
	bob, bobM := e.Member(admin, net, "bob")
	adminToken, aliceToken := h.token(admin, domain.ScopeRead, domain.ScopeWrite), h.token(alice, domain.ScopeRead, domain.ScopeWrite)
	tx := e.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	filed := h.do(201, call{method: "POST", path: "/reports", token: aliceToken, idem: true,
		body: m{"target_type": "transaction", "target_id": tx.ID, "category": "other", "description": "Ordinary report fixture"}})
	r := get[struct {
		ID                 string
		RelatedDisputesURL string `json:"related_disputes_url"`
	}](t, filed)
	if r.RelatedDisputesURL != "/api/v1/disputes?transaction_id="+tx.ID {
		t.Fatalf("wrong related-dispute URL: %q", r.RelatedDisputesURL)
	}
	tu.Must(e.Svc.OpenDispute(e.Ctx, alice, net.ID, service.OpenDisputeInput{
		TxID: tx.ID, Category: "other", Statement: "Ordinary dispute fixture", IdempotencyKey: e.Key(),
	}))
	h.do(200, call{method: "GET", path: "/admin/reports", token: adminToken})
	h.do(200, call{method: "GET", path: strings.TrimPrefix(r.RelatedDisputesURL, "/api/v1"), token: adminToken})
	h.do(200, call{method: "POST", path: "/admin/reports/" + r.ID + "/handle", token: adminToken, idem: true,
		body: m{"note": "Reviewed independently"}})
}
