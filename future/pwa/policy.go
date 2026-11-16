// Package pwa defines a public-asset allowlist; it never handles private responses.
package pwa

import (
	"net/url"
	"regexp"
	"strings"
)

// Cacheable rejects navigation, credentials, queries and all non-fingerprinted assets.
// The embedding service worker must also verify response headers before storing bytes.
func Cacheable(origin, rawURL, method, destination string, publicAssets map[string]bool) bool {
	return CacheableAt(origin, "/assets/", rawURL, method, destination, publicAssets)
}

var publicFingerprint = regexp.MustCompile(`\.[0-9a-f]{8,64}\.(?:css|js|png|woff2?)$`)

// CacheableAt supports an explicit public prefix such as /community/static/.
func CacheableAt(origin, prefix, rawURL, method, destination string, publicAssets map[string]bool) bool {
	if !basePattern.MatchString(prefix) || strings.ContainsAny(origin, "?#") || strings.ContainsAny(rawURL, "?#") {
		return false
	}
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
	return strings.HasPrefix(u.Path, prefix) && publicFingerprint.MatchString(u.Path) && !strings.Contains(u.Path, "..") && publicAssets[u.Path]
}
