package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

// SEC-006 network-join-limit (batch-2 review): when a network's membership
// window is full, accepting an invitation in the web gets a throttled answer
// (429 with Retry-After for the configured window) and plain wording, not the
// generic 422 refusal that tells the person to wait a minute.
func TestNetworkJoinLimitRefusalIsThrottledInWeb(t *testing.T) {
	w := newWorldCfg(t, func(c *service.Config) {
		c.NetworkJoinLimit, c.NetworkJoinWindow = 2, 24*time.Hour // alice and bob fill it
	}, nil)
	tok, _, err := w.env.Svc.CreateInvitation(w.env.Ctx, w.admin, w.net.ID, service.InviteInput{})
	if err != nil {
		t.Fatal(err)
	}
	w.env.Person("carol@example.org")
	b := w.browser(t)
	b.login("carol@example.org")
	form := b.get("/join/" + tok)
	p := b.post("/join/"+tok, form, url.Values{"display_name": {"Carol"}})
	if p.Status != 429 {
		t.Fatalf("status %d, want 429", p.Status)
	}
	if got := p.Header.Get("Retry-After"); got != "86400" {
		t.Errorf("Retry-After %q, want 86400", got)
	}
	b.mustContain(p, "not accepting new members right now", "Nothing changed")
	if strings.Contains(p.Body, "Wait a minute") {
		t.Error("network limit refusal uses the sign-in throttle wording")
	}
	// The invitation was not consumed: it still works once the window passes.
	w.env.Clock.Advance(25 * time.Hour)
	j := b.post("/join/"+tok, b.get("/join/"+tok), url.Values{"display_name": {"Carol"}})
	b.mustContain(j, "You have joined")
}
