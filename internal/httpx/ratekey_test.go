package httpx

import "testing"

// SA2-15: limiter keys are /32 for IPv4 and /64 for IPv6; v4-mapped is v4.
func TestRateKey(t *testing.T) {
	for in, want := range map[string]string{
		"203.0.113.9":                      "203.0.113.9",
		"::ffff:203.0.113.9":               "203.0.113.9",
		"2001:db8:1:2:aaaa:bbbb:cccc:dddd": "2001:db8:1:2::/64",
		"2001:db8:1:2:1::1":                "2001:db8:1:2::/64",
		"2001:db8:1:3::1":                  "2001:db8:1:3::/64",
		"fe80::1%eth0":                     "fe80::/64",
		"not-an-ip":                        "not-an-ip",
	} {
		if got := RateKey(in); got != want {
			t.Errorf("RateKey(%q) = %q, want %q", in, got, want)
		}
	}
	if RateKey("2001:db8:1:2::1") != RateKey("2001:db8:1:2:ffff::9") {
		t.Error("addresses of one /64 differ")
	}
	if RateKey("2001:db8:1:2::1") == RateKey("2001:db8:1:3::1") {
		t.Error("different /64s share a key")
	}
	e := edge("10.0.0.0/8")
	r := req("10.0.0.1:1", "X-Forwarded-For", "2001:db8:1:2:3:4:5:6")
	if got := e.RateKey(r); got != "2001:db8:1:2::/64" {
		t.Errorf("Edge.RateKey = %q", got)
	}
	if got := e.ClientIP(r); got != "2001:db8:1:2:3:4:5:6" {
		t.Errorf("ClientIP must stay exact, got %q", got)
	}
}
