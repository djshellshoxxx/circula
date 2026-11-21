package api_test

import (
	"net/http"
	"testing"

	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestPasswordAdmissionAPIReturnsRetryableServiceBusy(t *testing.T) {
	w := newWorld(t)
	session, _, err := w.env.Svc.Login(w.env.Ctx, "alice", replayTestPassword)
	if err != nil {
		t.Fatal(err)
	}
	headers := map[string]string{"Cookie": (&http.Cookie{Name: httpx.SessionCookie, Value: session}).String(), "X-CSRF-Token": csrfFor(session)}
	var held []*passwordwork.Ticket
	for range passwordwork.DefaultLimit {
		work := testutil.Must(passwordwork.Process().Acquire(w.env.Ctx))
		defer work.Release()
		held = append(held, work)
	}
	request := req{method: "POST", path: "/api/v1/me/password", hdr: headers,
		body: m{"current_password": replayTestPassword, "new_password": "a distinct new password 42"}}
	response := w.do(request)
	wantProblem(t, response, 503, "unavailable")
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("busy password change omitted retry guidance")
	}
	// Session authority survives a capacity refusal; the request can be
	// retried normally when one process slot becomes available.
	wantStatus(t, w.do(req{method: "GET", path: "/api/v1/me", hdr: headers}), 200)
	held[0].Release()
	wantStatus(t, w.do(request), 204)
}
