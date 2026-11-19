package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type recoveryTransport func(*http.Request) (*http.Response, error)

func (fn recoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestRecoveryCLIRejectsMalformedUTF8BeforeRequest(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	for _, command := range []string{"send", "report", "search"} {
		for i, malformed := range []string{"\xff", "\x80", "\xc0\xaf", "\xed\xa0\x80", "\xe2\x82", "\xf4\x90\x80\x80"} {
			t.Run(fmt.Sprintf("%s/%d", command, i), func(t *testing.T) {
				calls := 0
				http.DefaultTransport = recoveryTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
				})
				var out strings.Builder
				args := []string{"-url", "https://fixture.invalid", "-network", "n", "-thread", "t", "-key", "key-1234", command}
				err := run(args, strings.NewReader("prefix"+malformed), &out)
				if err == nil || calls != 0 || out.Len() != 0 {
					t.Fatalf("malformed stdin reached transport: calls=%d output=%q err=%v", calls, out.String(), err)
				}
			})
		}
	}
}

func TestRecoveryCLIPreservesValidUnicodeStdin(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	const text = "literal � café 北 🧭"
	for command, field := range map[string]string{"send": "body", "report": "reason", "search": "query"} {
		t.Run(command, func(t *testing.T) {
			calls := 0
			http.DefaultTransport = recoveryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload[field] != text {
					t.Fatalf("valid stdin changed: %#v, %v", payload, err)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
			})
			args := []string{"-url", "https://fixture.invalid", "-network", "n", "-thread", "t", "-key", "key-1234", command}
			if err := run(args, strings.NewReader(text), io.Discard); err != nil || calls != 1 {
				t.Fatalf("valid stdin rejected: calls=%d err=%v", calls, err)
			}
		})
	}
}
