package main

import (
	"net/netip"
	"testing"
)

// SA2-15: warn when the shape suggests an unconfigured reverse proxy.
func TestProxyWarning(t *testing.T) {
	proxies := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	cases := []struct {
		name string
		cfg  Config
		warn bool
	}{
		{"loopback, https, no proxies", Config{Listen: "127.0.0.1:8080", PublicBaseURL: "https://c.example.org"}, true},
		{"ipv6 loopback", Config{Listen: "[::1]:8080", PublicBaseURL: "https://c.example.org"}, true},
		{"proxies configured", Config{Listen: "127.0.0.1:8080", PublicBaseURL: "https://c.example.org", TrustedProxies: proxies}, false},
		{"non-loopback bind", Config{Listen: "0.0.0.0:8080", PublicBaseURL: "https://c.example.org"}, false},
		{"plain http loopback (dev)", Config{Listen: "127.0.0.1:8080", PublicBaseURL: "http://127.0.0.1:8080"}, false},
	}
	for _, c := range cases {
		if got := proxyWarning(&c.cfg) != ""; got != c.warn {
			t.Errorf("%s: warn=%v, want %v", c.name, got, c.warn)
		}
	}
}
