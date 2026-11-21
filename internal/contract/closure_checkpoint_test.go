package contract

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-025 e: final-export records and successful or unavailable checkpoint
// verification responses all validate against the published OpenAPI schema.
func TestClosureCheckpointMatchesOpenAPI(t *testing.T) {
	h := newHarness(t)
	e := h.env
	admin := e.Person("admin")
	net, _ := e.Network(admin, "closure-contract", nil)
	token := h.token(admin, domain.ScopeRead, domain.ScopeWrite)
	h.do(200, call{method: "POST", path: "/admin/network/close", token: token, body: m{"reason": "finished"}})
	e.Clock.Advance(31 * 24 * time.Hour)
	testutil.Must(e.Svc.FinaliseClosures(e.Ctx, domain.SystemActor("closure-test")))
	e.Clock.Advance(time.Hour)
	h.do(200, call{method: "GET", path: "/network", token: token})
	h.do(200, call{method: "GET", path: "/admin/network/closure", token: token})
	h.do(200, call{method: "POST", path: "/admin/network/closure/verify", token: token})
	for _, version := range []int{0, 99} {
		if _, err := e.Store.DB().Exec(`UPDATE network_closures SET final_export_version=? WHERE network_id=?`, version, net.ID); err != nil {
			t.Fatal(err)
		}
		h.do(200, call{method: "GET", path: "/admin/network/closure", token: token})
		h.do(409, call{method: "POST", path: "/admin/network/closure/verify", token: token})
	}
}
