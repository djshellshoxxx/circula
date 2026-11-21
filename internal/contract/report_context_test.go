package contract

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025: optional case input, full administrator relation and the distinct
// redacted projection agree with the published request/response schemas.
func TestReportDisputeContextMatchesOpenAPI(t *testing.T) {
	h := newHarness(t)
	t.Cleanup(func() { api.CloseHandler(h.h) })
	e := h.env
	admin := e.Person("context-administrator")
	net, _ := e.Network(admin, "report-context-contract", nil)
	alice, aliceM := e.Member(admin, net, "context-alice")
	bob, bobM := e.Member(admin, net, "context-bob")
	trade := e.Trade(net.ID, alice, bob, aliceM, bobM, 100)
	dispute := tu.Must(e.Svc.OpenDispute(e.Ctx, alice, net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "Ordinary case", IdempotencyKey: e.Key(),
	}))
	adminT, aliceT := h.token(admin, domain.ScopeRead, domain.ScopeWrite), h.token(alice, domain.ScopeRead, domain.ScopeWrite)
	filed := h.do(201, call{method: "POST", path: "/reports", token: aliceT, idem: true,
		body: m{"target_type": "member", "target_id": bobM.ID, "dispute_id": dispute.ID, "category": "other", "description": "Private context fixture"}})
	r := get[struct {
		ID                string
		DisputeID         string `json:"dispute_id"`
		RelatedDisputeURL string `json:"related_dispute_url"`
	}](t, filed)
	if r.DisputeID != dispute.ID || r.RelatedDisputeURL != "/api/v1/disputes/"+dispute.ID {
		t.Fatal("report response omitted the exact case relation")
	}
	h.do(200, call{method: "GET", path: "/admin/reports", token: adminT})
	h.do(200, call{method: "GET", path: strings.TrimPrefix(r.RelatedDisputeURL, "/api/v1"), token: adminT})
	h.do(200, call{method: "POST", path: "/admin/reports/" + r.ID + "/handle", token: adminT, idem: true, body: m{"note": "Reviewed"}})
	reviewer, reviewerM := e.Member(admin, net, "context-reviewer")
	tu.Must(0, e.Svc.AssignRole(e.Ctx, admin, net.ID, reviewerM.ID, domain.RoleReviewer))
	h.do(200, call{method: "GET", path: "/disputes/" + dispute.ID + "/reports", token: h.token(reviewer, domain.ScopeRead)})
	h.do(403, call{method: "GET", path: "/disputes/" + dispute.ID + "/reports", token: aliceT})
}
