package pwa

import "testing"

func TestNeverCachePrivate(t *testing.T) {
	assets := map[string]bool{"/assets/app.0123abcd.css": true}
	for _, tc := range []struct {
		u, m, d string
		want    bool
	}{
		{"https://local.test/assets/app.0123abcd.css", "GET", "style", true},
		{"https://evil.test/assets/app.0123abcd.css", "GET", "style", false},
		{"https://local.test/api/v1/balances", "GET", "", false},
		{"https://local.test/assets/app.0123abcd.css?token=secret", "GET", "style", false},
		{"https://local.test/assets/app.0123abcd.css", "POST", "style", false},
		{"https://local.test/assets/app.0123abcd.css", "GET", "document", false},
		{"https://local.test/assets/missing.css", "GET", "style", false},
	} {
		if got := Cacheable("https://local.test", tc.u, tc.m, tc.d, assets); got != tc.want {
			t.Fatalf("%s: %v", tc.u, got)
		}
	}
}
