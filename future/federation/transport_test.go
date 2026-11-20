package federation

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTransportResourceLimitsAndRedirectPolicy(t *testing.T) {
	client := (HTTPSDelivery{}).client("https://example.com/federation/v2/events")
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || transport.TLSClientConfig.InsecureSkipVerify || transport.TLSClientConfig.MinVersion < tls.VersionTLS12 || client.Timeout != 5*time.Second {
		t.Fatal("unsafe TLS/proxy/timeout")
	}
	for _, c := range []struct {
		code    int
		body    string
		failure bool
	}{{200, "ok", false}, {204, "", false}, {500, "failed", true}, {200, strings.Repeat("a", 4097), true}, {302, "redirect", true}} {
		calls := 0
		client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("Content-Type") != "application/circula-discovery+jws" {
				t.Fatal("wire content type")
			}
			return &http.Response{StatusCode: c.code, Header: http.Header{"Location": []string{"https://127.0.0.1/metadata"}}, Body: io.NopCloser(strings.NewReader(c.body)), Request: r}, nil
		})
		err := sendBounded(context.Background(), "https://example.com/federation/v2/events", "wire", client)
		if (err != nil) != c.failure || calls != 1 {
			t.Fatal(c, err, calls)
		}
	}
	client.Timeout = 10 * time.Millisecond
	client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	if err := sendBounded(context.Background(), "https://example.com/federation/v2/events", "wire", client); err == nil {
		t.Fatal("slow transport")
	}
	client.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic TLS verification failure")
	})
	if err := sendBounded(context.Background(), "https://example.com/federation/v2/events", "wire", client); err == nil {
		t.Fatal("TLS failure propagated")
	}
}
