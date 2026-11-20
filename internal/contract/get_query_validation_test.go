package contract

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestGetQueryErrorsMatchOpenAPI(t *testing.T) {
	h := newHarness(t)
	admin := h.env.Person("admin")
	h.env.Network(admin, "acme", nil)
	token := h.token(admin, domain.ScopeRead)
	for _, path := range []string{"/me", "/admin/network/closure", "/notifications/unread-count", "/tokens"} {
		h.do(400, call{method: "GET", path: path + "?bogus=1", token: token, skipRequest: true})
	}
}
