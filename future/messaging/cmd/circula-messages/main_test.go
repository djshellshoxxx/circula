package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// The test substitutes a TLS transport trusted only for its local fixture.
func TestCLIRequestAndNoCredentialRedirect(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	var method, path, network, key, body string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, network, key = r.Method, r.URL.Path, r.Header.Get("X-Network"), r.Header.Get("Idempotency-Key")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		if r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("missing token")
		}
		if strings.Contains(body, "redirect") {
			http.Redirect(w, r, "https://never-called.invalid", 302)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	var out bytes.Buffer
	if e := run([]string{"-url", server.URL, "-network", "n", "-thread", "t", "-key", "key-1234", "send"}, strings.NewReader("message from stdin"), &out); e != nil {
		t.Fatal(e)
	}
	if method != "POST" || path != "/api/v1/conversations/t/messages" || network != "n" || key != "key-1234" || !strings.Contains(body, "message from stdin") || out.String() != `{"ok":true}` {
		t.Fatal(method, path, network, key, body, out.String())
	}
	if e := run([]string{"-url", server.URL, "-network", "n", "-thread", "t", "send"}, strings.NewReader("redirect"), &out); e == nil || !strings.Contains(e.Error(), "302") {
		t.Fatal(e)
	}
}
func TestCLIValidation(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "token")
	for _, args := range [][]string{{}, {"-url", "http://example.test", "-network", "n", "list"}, {"-url", "https://user:pw@example.test", "-network", "n", "list"}, {"-url", "https://example.test/path", "-network", "n", "list"}, {"-url", "https://example.test", "list"}, {"-url", "https://example.test", "-network", "n", "unknown"}} {
		if e := run(args, strings.NewReader(""), io.Discard); e == nil {
			t.Fatal(args)
		}
	}
	os.Unsetenv("CIRCULA_TOKEN")
	if e := run([]string{"-url", "https://example.test", "-network", "n", "list"}, strings.NewReader(""), io.Discard); e == nil {
		t.Fatal("missing token")
	}
}

func TestCLILargePageCompleteOrError(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	response := strings.Repeat("<", 1500000)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, response) }))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	var out bytes.Buffer
	args := []string{"-url", server.URL, "-network", "n", "list"}
	if e := run(args, strings.NewReader(""), &out); e != nil || out.String() != response {
		t.Fatal("large page truncated", out.Len(), e)
	}
	response = strings.Repeat("x", (4<<20)+1)
	out.Reset()
	if e := run(args, strings.NewReader(""), &out); e == nil || out.Len() != 0 {
		t.Fatal("oversized response reported success", out.Len(), e)
	}
}

func TestCLIAdminCommandsAndConversationCursor(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	var method, path, body string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.RequestURI()
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	for _, c := range []struct {
		args               []string
		method, path, body string
	}{{[]string{"-cursor", "abc", "list"}, "GET", "/api/v1/conversations?after=abc", ""}, {[]string{"-thread", "t", "-after", "2", "-reporter", "bob", "reports"}, "GET", "/api/v1/conversations/t/reports?after=2&reporter=bob", ""}, {[]string{"-thread", "t", "held"}, "GET", "/api/v1/conversations/t/held?after=0&reporter=", ""}, {[]string{"-thread", "t", "-moderation-action", "hold", "moderate"}, "POST", "/api/v1/conversations/t/moderate", `"action":"hold"`}, {[]string{"-days", "1", "retention"}, "POST", "/api/v1/retention", `"days":1`}, {[]string{"purge"}, "POST", "/api/v1/purge", `{}`}} {
		args := append([]string{"-url", server.URL, "-network", "n"}, c.args...)
		if e := run(args, strings.NewReader(""), io.Discard); e != nil || method != c.method || path != c.path || !strings.Contains(body, c.body) {
			t.Fatal(c, method, path, body, e)
		}
	}
}

func TestCLISearchUsesBoundedStdinAndPOST(t *testing.T) {
	t.Setenv("CIRCULA_TOKEN", "fixture-token")
	called := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.URL.RequestURI() != "/api/v1/conversations/t/search" || !strings.Contains(string(body), `"query":"private phrase"`) || !strings.Contains(string(body), `"after":7`) {
			t.Error(r.Method, r.URL.RequestURI(), string(body))
		}
		io.WriteString(w, `[]`)
	}))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = previous }()
	args := []string{"-url", server.URL, "-network", "n", "-thread", "t", "-after", "7", "search"}
	if err := run(args, strings.NewReader("private phrase"), io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := run(args, strings.NewReader(strings.Repeat("x", 129)), io.Discard); err == nil || called != 1 {
		t.Fatal("oversized search sent", called, err)
	}
}
