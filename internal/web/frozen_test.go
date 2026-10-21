package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// CMD-07: a write refused because the network is frozen shows a clear
// read-only page with status 423, not a generic failure.
func TestFrozenNetworkPage(t *testing.T) {
	w := newWorld(t)
	s := newServer(w.env.Svc, Options{})
	rec := httptest.NewRecorder()
	s.fail(rec, httptest.NewRequest("POST", "/", nil), domain.Errf(domain.CodeNetworkFrozen, "x"))
	if rec.Code != 423 || !strings.Contains(rec.Body.String(), "read-only") {
		t.Fatalf("frozen page: %d %q", rec.Code, rec.Body.String())
	}
}
