package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// API-024: while the first request with an Idempotency-Key is still running,
// a second request with that key gets a retryable 409 and does not run the
// handler again. Once the first finishes, the key replays its stored response.
func TestInProgressIdempotencyKeyIs409UntilFinished(t *testing.T) {
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	env.Network(admin, "acme", nil)
	raw, _, err := env.Svc.CreateAPIToken(context.Background(), admin, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWithOptions(env.Svc, Options{}).(*server)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	h.add(route{pattern: "POST /api/v1/zz-slow", idem: idemRequired, h: func(c *reqCtx) error {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		return ok(c, 201, map[string]string{"run": "first"})
	}})
	send := func(body, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/zz-slow", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	first := make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- send(`{"a":1}`, "slow-key") }()
	<-started

	busy := send(`{"a":1}`, "slow-key")
	if busy.Code != http.StatusConflict || !strings.Contains(busy.Body.String(), "idempotency-request-in-progress") {
		t.Fatalf("concurrent retry: %d %s", busy.Code, busy.Body.String())
	}
	if busy.Header().Get("Retry-After") == "" {
		t.Error("409 without Retry-After")
	}
	// The same key with a different body is refused as reuse, even while in flight.
	if other := send(`{"a":2}`, "slow-key"); other.Code != http.StatusUnprocessableEntity {
		t.Errorf("different body during flight: %d %s", other.Code, other.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("handler ran %d times while in flight", calls.Load())
	}

	close(release)
	rec := <-first
	if rec.Code != http.StatusCreated {
		t.Fatalf("first request: %d %s", rec.Code, rec.Body.String())
	}
	replay := send(`{"a":1}`, "slow-key")
	if replay.Code != http.StatusCreated || replay.Body.String() != rec.Body.String() || replay.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay after completion: %d %q (first %q)", replay.Code, replay.Body.String(), rec.Body.String())
	}
	if calls.Load() != 1 {
		t.Errorf("replay re-ran the handler: %d calls", calls.Load())
	}
}
