package federation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

func ValidateEndpoint(raw string) error {
	if len(raw) > 2048 {
		return ErrInvalid
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || u.Port() != "" && u.Port() != "443" || !strings.HasPrefix(u.Path, "/") {
		return ErrInvalid
	}
	if ip, e := netip.ParseAddr(u.Hostname()); e == nil && !PublicAddress(ip) {
		return ErrInvalid
	}
	return nil
}

var forbiddenPrefixes = []netip.Prefix{netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("3fff::/20")}

func PublicAddress(a netip.Addr) bool {
	a = a.Unmap()
	if a.Is6() && !netip.MustParsePrefix("2000::/3").Contains(a) {
		return false
	}
	if !a.IsValid() || !a.IsGlobalUnicast() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() || a.IsMulticast() || a.Zone() != "" {
		return false
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// HTTPSDelivery always validates every DNS answer and dials a validated address
// directly. TLS retains the configured hostname; redirects/proxies are disabled.
type HTTPSDelivery struct{ Resolver Resolver }

func (h HTTPSDelivery) Send(ctx context.Context, endpoint, wire string) error {
	if ValidateEndpoint(endpoint) != nil || len(wire) > MaxWire {
		return ErrInvalid
	}
	client := h.client(endpoint)
	defer client.CloseIdleConnections()
	return sendBounded(ctx, endpoint, wire, client)
}
func (h HTTPSDelivery) client(endpoint string) *http.Client {
	u, _ := url.Parse(endpoint)
	resolver := h.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, TLSHandshakeTimeout: 3 * time.Second, ResponseHeaderTimeout: 3 * time.Second, DisableKeepAlives: true, MaxResponseHeaderBytes: 8192}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil || !strings.EqualFold(host, u.Hostname()) || port != "443" {
			return nil, ErrInvalid
		}
		ips, e := resolver.LookupNetIP(ctx, "ip", host)
		if e != nil || len(ips) == 0 || len(ips) > 16 {
			return nil, ErrInvalid
		}
		for _, ip := range ips {
			if !PublicAddress(ip) {
				return nil, ErrInvalid
			}
		}
		dialer := net.Dialer{Timeout: 3 * time.Second}
		return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ips[0].String(), port))
	}
	return &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }}

}
func sendBounded(ctx context.Context, endpoint, wire string, client *http.Client) error {
	request, e := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(wire))
	if e != nil {
		return e
	}
	request.Header.Set("Content-Type", "application/circula-discovery+jws")
	response, e := client.Do(request)
	if e != nil {
		return errors.New("peer transport failed")
	}
	defer response.Body.Close()
	body, e := io.ReadAll(io.LimitReader(response.Body, 4097))
	if e != nil || len(body) > 4096 {
		return errors.New("peer response rejected")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("peer status %d", response.StatusCode)
	}
	return nil
}
func decodeOutgoing(wire string) (Discovery, error) {
	if _, err := WireKeyID(wire); err != nil {
		return Discovery{}, err
	}
	parts := strings.Split(wire, ".")
	if len(parts) != 3 || len(wire) > MaxWire {
		return Discovery{}, ErrInvalid
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return Discovery{}, ErrInvalid
	}
	var event Discovery
	if !utf8.Valid(raw) || json.Unmarshal(raw, &event) != nil {
		return Discovery{}, ErrInvalid
	}
	canonical, _ := json.Marshal(event)
	if !bytes.Equal(canonical, raw) {
		return Discovery{}, ErrInvalid
	}
	return event, nil
}
