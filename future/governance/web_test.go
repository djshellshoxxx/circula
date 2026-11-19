package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type webFixture struct{}

func (webFixture) Authorize(r *http.Request, mutate bool) (Identity, error) {
	member := r.Header.Get("X-Test-Member")
	if member == "" {
		return Identity{}, errors.New("unauthenticated")
	}
	if mutate && r.Header.Get("X-Test-CSRF") != "valid" {
		return Identity{}, errors.New("csrf")
	}
	return Identity{"n", member}, nil
}
func (webFixture) CSRF(*http.Request) string { return "csrf" }
func TestWebCLIContracts(t *testing.T) {
	s, _, now, _ := newFixture(t)
	r := opened(t, s, *now)
	web := Web{s, webFixture{}, []byte(strings.Repeat("k", 32))}
	for _, test := range []struct {
		method, query, body, member, csrf, intent string
		status                                    int
	}{{"GET", "?proposal=" + r.Proposal, "", "", "", "", 403}, {"GET", "?proposal=" + r.Proposal, "", "b", "", "", 200}, {"GET", "?proposal=" + r.Proposal + "&format=html", "", "b", "", "", 200}, {"DELETE", "", "", "b", "", "", 405}, {"POST", "", `{"Action":"vote","Proposal":"` + r.Proposal + `","Intent":"x","Revision":2,"Choice":"yes"}`, "b", "", "x", 403}, {"POST", "", `{"Action":"vote","Proposal":"` + r.Proposal + `","Intent":"x","Revision":2,"Choice":"yes"}`, "b", "valid", "different", 400}, {"GET", "?cursor=garbage", "", "b", "", "", 400}} {
		request := httptest.NewRequest(test.method, "/"+test.query, strings.NewReader(test.body))
		request.Header.Set("X-Test-Member", test.member)
		request.Header.Set("X-Test-CSRF", test.csrf)
		request.Header.Set("Idempotency-Key", test.intent)
		writer := httptest.NewRecorder()
		web.ServeHTTP(writer, request)
		if writer.Code != test.status {
			t.Fatalf("%+v: %d %s", test, writer.Code, writer.Body.String())
		}
		if writer.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cache")
		}
	}
	form := url.Values{"action": {"vote"}, "proposal": {r.Proposal}, "intent": {"form-vote"}, "revision": {"2"}, "choice": {"abstain"}, "csrf": {"csrf"}}
	request := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Test-Member", "b")
	request.Header.Set("X-Test-CSRF", "valid")
	writer := httptest.NewRecorder()
	web.ServeHTTP(writer, request)
	if writer.Code != 303 || !strings.Contains(writer.Header().Get("Location"), r.Proposal) {
		t.Fatal(writer.Code, writer.Body.String())
	}
	request = httptest.NewRequest("GET", "/?cursor="+web.cursor(Identity{"n", "a"}, r.Proposal), nil)
	request.Header.Set("X-Test-Member", "b")
	writer = httptest.NewRecorder()
	web.ServeHTTP(writer, request)
	if writer.Code != 400 {
		t.Fatal("cursor crossed account", writer.Code)
	}
	var out bytes.Buffer
	if e := RunCLI(context.Background(), s, Identity{"n", "a"}, []string{"show", r.Proposal}, nil, &out); e != nil {
		t.Fatal(e)
	}
	var d Detail
	if e := json.Unmarshal(out.Bytes(), &d); e != nil || d.OwnVote != nil {
		t.Fatal(d, e)
	}
	if e := RunCLI(context.Background(), s, Identity{"n", "a"}, []string{"command"}, strings.NewReader(strings.Repeat(" ", 65537)), &out); e == nil {
		t.Fatal("oversized cli")
	}
}
func TestWebBodyTenantAndDisabled(t *testing.T) {
	s, auth, _, _ := newFixture(t)
	web := Web{s, webFixture{}, []byte(strings.Repeat("k", 32))}
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"Body":"`+strings.Repeat("x", 66000)+`"}`))
	request.Header.Set("X-Test-Member", "a")
	request.Header.Set("X-Test-CSRF", "valid")
	writer := httptest.NewRecorder()
	web.ServeHTTP(writer, request)
	if writer.Code != 413 {
		t.Fatal("body policy", writer.Code, writer.Body.String())
	}
	request = httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Test-Member", "a")
	request.Header.Set("X-Network", "other")
	writer = httptest.NewRecorder()
	web.ServeHTTP(writer, request)
	if writer.Code != 403 {
		t.Fatal("network switch")
	}
	auth.mu.Lock()
	auth.members["a"] = Principal{true, false, false}
	auth.mu.Unlock()
	if _, e := s.List(context.Background(), Identity{"n", "a"}, "", 10); e == nil {
		t.Fatal("disabled bypass")
	}
}
