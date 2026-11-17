// Package notifytest holds in-memory fixtures for the notification module's
// tests and for host adapters' own acceptance harnesses: a controllable
// Directory and Clock, a throw-away PKI, a fake SMTP server and a fake Web Push
// service that decrypts what it receives. Nothing here talks to the real
// network, and none of it may be linked into production code.
package notifytest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/netip"
	"sync"
	"time"
)

// Clock is a settable clock.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

// NewClock starts at t.
func NewClock(t time.Time) *Clock { return &Clock{t: t} }

// Now implements notifications.Clock.
func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

// Add advances time.
func (c *Clock) Add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// Set jumps to t.
func (c *Clock) Set(t time.Time) { c.mu.Lock(); c.t = t; c.mu.Unlock() }

// Directory implements notifications.Directory from in-memory state. Unknown
// members are inactive (fail closed).
type Directory struct {
	mu       sync.Mutex
	active   map[[2]string]bool
	sessions map[[3]string]bool
	Err      error // returned by every call when set
}

// NewDirectory returns an empty directory.
func NewDirectory() *Directory {
	return &Directory{active: map[[2]string]bool{}, sessions: map[[3]string]bool{}}
}

// SetActive marks a member active or not.
func (d *Directory) SetActive(network, member string, v bool) {
	d.mu.Lock()
	d.active[[2]string{network, member}] = v
	d.mu.Unlock()
}

// SetSession marks a login session valid or not.
func (d *Directory) SetSession(network, member, session string, v bool) {
	d.mu.Lock()
	d.sessions[[3]string{network, member, session}] = v
	d.mu.Unlock()
}

// MemberActive implements notifications.Directory.
func (d *Directory) MemberActive(_ context.Context, network, member string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active[[2]string{network, member}], d.Err
}

// SessionActive implements notifications.Directory. Unregistered sessions are
// treated as valid so tests only need to register the ones they end.
func (d *Directory) SessionActive(_ context.Context, network, member, session string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.sessions[[3]string{network, member, session}]
	if !ok {
		v = true
	}
	return v, d.Err
}

// PKI is a throw-away certificate authority.
type PKI struct {
	CA   *x509.Certificate
	key  *ecdsa.PrivateKey
	Pool *x509.CertPool
}

// NewPKI creates a CA.
func NewPKI() *PKI {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "notifytest CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, _ := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	ca, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(ca)
	return &PKI{CA: ca, key: key, Pool: pool}
}

// Leaf issues a server certificate for the given DNS names.
func (p *PKI) Leaf(names ...string) tls.Certificate {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: names[0]},
		DNSNames:  names,
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tpl, p.CA, &key.PublicKey, p.key)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// StaticResolver answers LookupNetIP from a map; unknown names fail.
type StaticResolver map[string][]netip.Addr

// LookupNetIP implements notifications.Resolver.
func (r StaticResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	a, ok := r[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	return a, nil
}

// RedirectDialer connects every dial to a fixed local address, recording the
// IP the production code asked for. It stands in for the network so tests can
// run the full resolve-validate-dial path against a loopback server.
type RedirectDialer struct {
	Target string
	mu     sync.Mutex
	dialed []string
}

// DialContext implements notifications.Dialer.
func (d *RedirectDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	d.dialed = append(d.dialed, address)
	d.mu.Unlock()
	var nd net.Dialer
	return nd.DialContext(ctx, network, d.Target)
}

// Dialed lists the addresses production code tried to reach.
func (d *RedirectDialer) Dialed() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.dialed...)
}

type tlsCertificate = tls.Certificate

func tlsConfig(c tls.Certificate) *tls.Config {
	return &tls.Config{Certificates: []tls.Certificate{c}, MinVersion: tls.VersionTLS12}
}
