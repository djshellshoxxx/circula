package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// API-06: a panicking handler must release its idempotency record, so a retry
// with the same key runs again instead of waiting 24 h behind a 409.
func TestPanicReleasesIdempotencyKey(t *testing.T) {
	env := testutil.NewEnv(t)
	admin := env.Person("admin")
	env.Network(admin, "acme", nil)
	raw, _, err := env.Svc.CreateAPIToken(context.Background(), admin, service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWithOptions(env.Svc, Options{}).(*server)
	calls := 0
	h.add(route{pattern: "POST /api/v1/zz-panic", idem: idemRequired, h: func(c *reqCtx) error {
		calls++
		if calls == 1 {
			panic("boom")
		}
		return ok(c, 201, map[string]string{"fine": "yes"})
	}})
	do := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/zz-panic", strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer "+raw)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "same-key")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	if rec := do(); rec.Code != 500 {
		t.Fatalf("panic response: %d %s", rec.Code, rec.Body.String())
	}
	if n := len(h.idem.m); n != 0 {
		t.Fatalf("%d idempotency record(s) left behind", n)
	}
	if rec := do(); rec.Code != 201 {
		t.Fatalf("retry after panic: %d %s", rec.Code, rec.Body.String())
	}
}

// CMD-05: SQLite lock contention maps to a retryable 503.
func TestBusyMapsTo503(t *testing.T) {
	for _, err := range []error{
		errors.New("store: begin: database is locked (5) (SQLITE_BUSY)"),
		fmt.Errorf("wrapped: %w", errors.New("SQLITE_LOCKED")),
	} {
		ae, expected := fromErr(err)
		if ae.status != http.StatusServiceUnavailable || ae.code != "unavailable" || ae.retryAfter == "" || !expected {
			t.Errorf("%v -> %+v", err, ae)
		}
	}
	if ae, _ := fromErr(errors.New("disk on fire")); ae.status != 500 {
		t.Errorf("plain error: %d", ae.status)
	}
}

func TestUnavailableCodeMapsTo503(t *testing.T) {
	ae, expected := fromErr(domain.ErrUnavailable)
	if ae.status != 503 || ae.code != "unavailable" || ae.retryAfter == "" || !expected {
		t.Errorf("%+v", ae)
	}
}
