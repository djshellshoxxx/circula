package web

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

// SEC-006 / AUD4-F04: the durable service budget can be exhausted before the
// adapter's in-memory budget, including after the handler has restarted.
func TestPasswordChangeDurableThrottleStatus(t *testing.T) {
	w := newWorldCfg(t, func(c *service.Config) {
		c.AuthThrottleAttempts = 4
		c.AuthThrottleGlobalAttempts = 8
	}, nil)
	b := w.browser(t)
	b.login("alice@example.org")
	form := b.get("/account/password")
	const replacement = "a new correct password 2026"
	values := url.Values{"current_password": {"wrong current password"}, "password": {replacement}, "password2": {replacement}}
	for i := 0; i < 4; i++ {
		if p := b.post("/account/password", form, values); p.Status != http.StatusUnprocessableEntity {
			t.Fatalf("wrong password %d: status %d, want 422", i, p.Status)
		}
	}
	values.Set("current_password", testPW)
	p := b.post("/account/password", form, values)
	if p.Status != http.StatusTooManyRequests {
		t.Fatalf("durable throttle: status %d, want 429", p.Status)
	}
	b.mustContain(p, "Too many attempts", "Nothing changed")
	if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", testPW); err != nil {
		t.Fatalf("throttled request changed the existing password: %v", err)
	}

	// The finite service lock expires; the same valid request can then succeed
	// and must revoke the original browser session as ordinary changes do.
	w.env.Clock.Advance(service.DefaultAuthThrottleLock + time.Second)
	p = b.post("/account/password", form, values)
	if p.Status != http.StatusOK || p.URL != "/login" {
		t.Fatalf("change after lock expiry: status %d at %s", p.Status, p.URL)
	}
	if _, _, err := w.env.Svc.Login(w.env.Ctx, "alice@example.org", replacement); err != nil {
		t.Fatalf("replacement password after lock expiry: %v", err)
	}
}
