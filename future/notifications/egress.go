package notifications

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// DefaultPushProviders is the built-in Web Push provider allowlist: the
// endpoint hosts browsers' push services are known to use. Operators extend or
// replace it explicitly; an empty policy rejects everything.
var DefaultPushProviders = []string{
	"fcm.googleapis.com",
	"updates.push.services.mozilla.com",
	"*.push.services.mozilla.com",
	"*.notify.windows.com",
	"web.push.apple.com",
	"*.push.apple.com",
}

// EndpointPolicy is the explicit provider-host policy for member-supplied Web
// Push endpoints. Only https on port 443 to an allowlisted DNS name is valid.
// IP literals, userinfo, fragments and non-ASCII or encoded hosts never are.
type EndpointPolicy struct {
	// AllowedHosts entries are exact lowercase names ("fcm.googleapis.com") or
	// "*.suffix" which matches one or more extra leading labels of a suffix that
	// itself has at least two labels.
	AllowedHosts []string
}

// MaxEndpointLen bounds stored endpoint URLs.
const MaxEndpointLen = 2048

// ErrEndpointRejected is returned for any endpoint the policy refuses. It
// never includes the endpoint.
var ErrEndpointRejected = fmt.Errorf("%w: push endpoint rejected by policy", ErrInvalid)

// Validate returns the normalised endpoint URL or ErrEndpointRejected.
func (p EndpointPolicy) Validate(raw string) (*url.URL, error) {
	if raw == "" || len(raw) > MaxEndpointLen {
		return nil, ErrEndpointRejected
	}
	for i := 0; i < len(raw); i++ { // printable ASCII only: no spaces, controls, raw unicode
		if raw[i] <= 0x20 || raw[i] >= 0x7f || raw[i] == '\\' {
			return nil, ErrEndpointRejected
		}
	}
	if strings.Contains(raw, "#") {
		return nil, ErrEndpointRejected
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.Host == "" {
		return nil, ErrEndpointRejected
	}
	if port := u.Port(); port != "" && port != "443" {
		return nil, ErrEndpointRejected
	}
	host := u.Hostname()
	if !p.hostAllowed(host) {
		return nil, ErrEndpointRejected
	}
	u.Host = host // drops an explicit :443 so equal endpoints hash equally
	return u, nil
}

// HostAllowed reports whether a bare host name passes the policy.
func (p EndpointPolicy) HostAllowed(host string) bool { return p.hostAllowed(host) }

func (p EndpointPolicy) hostAllowed(host string) bool {
	if host == "" || len(host) > 253 || host != strings.ToLower(host) || strings.HasSuffix(host, ".") {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}
	for i := 0; i < len(host); i++ {
		c := host[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.') {
			return false // rejects %, [, :, @, _, unicode
		}
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
	}
	for _, pat := range p.AllowedHosts {
		if rest, ok := strings.CutPrefix(pat, "*."); ok {
			if strings.Count(rest, ".") >= 1 && strings.HasSuffix(host, "."+rest) {
				return true
			}
			continue
		}
		if pat == host {
			return true
		}
	}
	return false
}

var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24",
	"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "255.255.255.255/32",
	"64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20", "5f00::/16",
)

func mustPrefixes(s ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(s))
	for i, v := range s {
		out[i] = netip.MustParsePrefix(v)
	}
	return out
}

// IsPublicIP reports whether addr is a globally routable unicast address.
// IPv4-mapped IPv6 is unmapped first, so ::ffff:127.0.0.1 is judged as
// 127.0.0.1. Transition ranges that embed IPv4 (NAT64, 6to4, Teredo) are
// refused outright rather than parsed.
func IsPublicIP(addr netip.Addr) bool {
	addr = addr.Unmap().WithZone("")
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() || addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast() || !addr.IsGlobalUnicast() {
		return false
	}
	for _, p := range blockedPrefixes {
		if p.Contains(addr) {
			return false
		}
	}
	return true
}

// Resolver is the DNS seam (net.DefaultResolver satisfies it).
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Dialer is the connect seam. Production leaves it nil and gets a dialer that
// refuses non-public destinations at socket level.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Egress builds the only HTTP client used for member-controlled destinations.
type Egress struct {
	Policy   EndpointPolicy
	Resolver Resolver // default net.DefaultResolver
	Dialer   Dialer   // default: public-only dialer
	RootCAs  *x509.CertPool
	// Timeouts: DialTimeout bounds resolve+connect, Timeout the whole request.
	DialTimeout time.Duration
	Timeout     time.Duration
}

// ErrBlockedDestination is returned when resolution or connection targets a
// non-public address or a host outside the policy.
var ErrBlockedDestination = errors.New("notifications: egress destination blocked")

func publicOnlyControl(_, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil || !IsPublicIP(ap.Addr()) {
		return ErrBlockedDestination
	}
	return nil
}

// Client returns an http.Client that: ignores proxy environment variables,
// never follows redirects, re-validates the host against the policy at dial
// time, resolves the name itself and refuses if ANY answer is non-public
// (mixed answers are how rebinding is smuggled), then connects to the checked
// IP literal while TLS still verifies the original host name.
func (e Egress) Client() *http.Client {
	res := e.Resolver
	if res == nil {
		res = net.DefaultResolver
	}
	dialTimeout, timeout := e.DialTimeout, e.Timeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	d := e.Dialer
	if d == nil {
		d = &net.Dialer{Timeout: dialTimeout, Control: publicOnlyControl}
	}
	policy := e.Policy
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil || port != "443" || !policy.hostAllowed(host) {
				return nil, ErrBlockedDestination
			}
			ctx, cancel := context.WithTimeout(ctx, dialTimeout)
			defer cancel()
			ips, err := res.LookupNetIP(ctx, "ip", host)
			if err != nil || len(ips) == 0 {
				return nil, ErrBlockedDestination
			}
			for _, ip := range ips {
				if !IsPublicIP(ip) {
					return nil, ErrBlockedDestination
				}
			}
			var lastErr error
			for _, ip := range ips {
				c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip.Unmap().String(), port))
				if err == nil {
					return c, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: e.RootCAs},
		TLSHandshakeTimeout:    dialTimeout,
		ResponseHeaderTimeout:  timeout,
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: 16 << 10,
	}
	return &http.Client{
		Transport:     guardTransport{policy: policy, next: tr},
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// guardTransport rejects requests the policy forbids before any DNS or dial.
type guardTransport struct {
	policy EndpointPolicy
	next   http.RoundTripper
}

func (g guardTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if _, err := g.policy.Validate(r.URL.String()); err != nil {
		return nil, ErrBlockedDestination
	}
	return g.next.RoundTrip(r)
}
