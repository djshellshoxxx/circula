package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// selfSigned writes a certificate/key pair and returns the paths and cert.
func selfSigned(t *testing.T, dir string, serial int64) (certPath, keyPath string, cert *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := x509.MarshalECPrivateKey(key)
	certPath, keyPath = filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _ := x509.ParseCertificate(der)
	return certPath, keyPath, c
}

func TestServeNativeTLS(t *testing.T) {
	db := seedDatabase(t, t.TempDir())
	certPath, keyPath, cert := selfSigned(t, t.TempDir(), 1)
	cfg := testConfigFor(db)
	cfg.Listen = freeAddr(t)
	cfg.PublicBaseURL = "https://localhost"
	cfg.MigrateMode = "explicit"
	cfg.MaxBodyBytes = 1 << 20
	cfg.RequestTimeout = 5 * time.Second
	cfg.MetricsAllow = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}
	cfg.TLSCertFile, cfg.TLSKeyFile = certPath, keyPath

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runServe(ctx, cfg, discardLogger()) }()

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	client := func(max uint16) *http.Client {
		return &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MaxVersion: max, MinVersion: tls.VersionTLS10},
		}}
	}
	var resp *http.Response
	var err error
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		select {
		case e := <-done:
			t.Fatalf("serve exited: %v", e)
		default:
		}
		resp, err = client(0).Get("https://" + cfg.Listen + "/readyz")
		if err == nil && resp.StatusCode == 200 {
			break
		}
	}
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("https never ready: %v", err)
	}
	_ = resp.Body.Close()
	if resp.TLS == nil || resp.TLS.Version < tls.VersionTLS12 {
		t.Fatalf("negotiated %+v", resp.TLS)
	}
	if r, err := client(0).Get("https://" + cfg.Listen + "/login"); err != nil {
		t.Fatal(err)
	} else {
		_ = r.Body.Close()
		if !strings.Contains(r.Header.Get("Strict-Transport-Security"), "max-age") {
			t.Errorf("no HSTS over native TLS (status %d)", r.StatusCode)
		}
	}
	// Plain http to the TLS port gets no application response.
	if r, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://" + cfg.Listen + "/readyz"); err == nil {
		if r.StatusCode == 200 {
			t.Error("plain http served on the TLS port")
		}
		_ = r.Body.Close()
	}
	// TLS 1.1 is refused.
	if r, err := client(tls.VersionTLS11).Get("https://" + cfg.Listen + "/readyz"); err == nil {
		_ = r.Body.Close()
		t.Error("TLS 1.1 handshake accepted")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("serve returned %v", err)
	}
}

func TestCertReloaderKeepsOldOnFailure(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath, c1 := selfSigned(t, dir, 11)
	r, err := newCertReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	serial := func() int64 {
		c, _ := r.getCertificate(nil)
		x, _ := x509.ParseCertificate(c.Certificate[0])
		return x.SerialNumber.Int64()
	}
	if serial() != c1.SerialNumber.Int64() {
		t.Fatal("wrong initial cert")
	}
	selfSigned(t, dir, 22) // overwrite with a new pair
	if err := r.reload(); err != nil || serial() != 22 {
		t.Fatalf("reload: %v serial=%d", err, serial())
	}
	if err := os.WriteFile(certPath, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := r.reload(); err == nil || serial() != 22 {
		t.Fatalf("bad reload must fail and keep the old cert: %v %d", err, serial())
	}
	if err := os.Chmod(keyPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newCertReloader(certPath, keyPath); err == nil {
		t.Fatal("group-readable key accepted")
	}
}

func TestTLSSettingsValidation(t *testing.T) {
	cfg := &Config{PublicBaseURL: "https://x.test", TLSCertFile: "c"}
	if _, err := tlsEnabled(cfg); err == nil {
		t.Error("cert without key accepted")
	}
	cfg = &Config{PublicBaseURL: "http://x.test", TLSCertFile: "c", TLSKeyFile: "k"}
	if _, err := tlsEnabled(cfg); err == nil {
		t.Error("tls with http public url accepted")
	}
	if on, err := tlsEnabled(&Config{PublicBaseURL: "http://127.0.0.1"}); on || err != nil {
		t.Error("no tls settings must be off without error")
	}
}

func TestAuditMirrorLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "mirror.jsonl")
	m, err := openAuditMirror(p, discardLogger())
	if err != nil {
		t.Fatal(err)
	}
	for i := int64(1); i <= 3; i++ {
		m.write(&domain.AuditEvent{ID: "id", NetworkID: "n", Seq: i, OccurredAt: time.Unix(i, 0), Action: "a.b",
			SummaryAfter: "SECRET-BODY", Hash: "h", PrevHash: "p"})
	}
	_ = m.close()
	b, _ := os.ReadFile(p)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 3 || strings.Contains(string(b), "SECRET-BODY") {
		t.Fatalf("mirror = %q", b)
	}
	var l mirrorLine
	if err := json.Unmarshal([]byte(lines[2]), &l); err != nil || l.Seq != 3 || l.Hash != "h" {
		t.Fatalf("line = %+v %v", l, err)
	}
}
