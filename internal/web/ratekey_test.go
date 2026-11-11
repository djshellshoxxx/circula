package web

import (
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

// SA2-15: rotating addresses inside one IPv6 /64 shares one failure budget.
func TestIPv6RotationSharesBudget(t *testing.T) {
	w := newWorldWith(t, func(o *Options) {
		o.Edge.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	})
	b := w.browser(t)
	login := func(xff, name string) int {
		p := b.get("/login")
		req, _ := http.NewRequest("POST", b.url+"/login", strings.NewReader(url.Values{"login": {name},
			"password": {"wrong password here"}, "csrf": {p.csrf()}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Forwarded-For", xff)
		return b.do(req).Status
	}
	// Distinct login names so only the per-address budget (20/min) can trip.
	last := 0
	for i := 0; i < 25; i++ {
		last = login(fmt.Sprintf("2001:db8:5:6:%x::%x", i+1, i+7), fmt.Sprintf("nobody%d@example.org", i))
	}
	if last != 429 {
		t.Errorf("rotation inside one /64 was not limited: last status %d", last)
	}
	if got := login("2001:db8:5:7::1", "other@example.org"); got == 429 {
		t.Errorf("a different /64 shares the budget: %d", got)
	}
	if got := login("198.51.100.7", "other@example.org"); got == 429 {
		t.Errorf("an IPv4 client shares the budget: %d", got)
	}
}
