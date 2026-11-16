// Package pwa defines a public-asset allowlist; it never handles private responses.
package pwa

import (
	"net/url"
	"strings"
)

// Cacheable rejects navigation, credentials, queries and all non-fingerprinted assets.
// The embedding service worker must also verify response headers before storing bytes.
func Cacheable(origin, rawURL, method, destination string, publicAssets map[string]bool) bool {
	base, e := url.Parse(origin)
	if e != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.Path != "" || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" {
		return false
	}
	u, e := url.Parse(rawURL)
	if e != nil || u.Scheme != base.Scheme || u.Host != base.Host || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || method != "GET" {
		return false
	}
	if destination != "style" && destination != "script" && destination != "image" && destination != "font" {
		return false
	}
	return strings.HasPrefix(u.Path, "/assets/") && !strings.Contains(u.Path, "..") && publicAssets[u.Path]
}
