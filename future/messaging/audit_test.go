package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAuditStrictJSONRejectsAmbiguity(t *testing.T) {
	f := setup(t)
	th := f.thread()
	h := testHTTP(t, f.s)
	for i, body := range []string{`{"body":"first","body":"second"}`, `{"body":"first","BODY":"second"}`, "{\"body\":\"" + string([]byte{255}) + "\"}", `null`, `[]`, `{"body":"\ud800"}`, `{"body":"\udfff"}`} {
		w := requestHTTP(h, "POST", "/api/v1/conversations/"+th.ID+"/messages", "alice", body, "application/json", fmt.Sprintf("audit-json-%04d", i), true)
		if w.Code != http.StatusBadRequest {
			t.Fatal(i, w.Code, w.Body.String())
		}
	}
	if messages, e := f.s.Messages(ctx, alice, th.ID, 0, 100); e != nil || len(messages) != 0 {
		t.Fatal(messages, e)
	}
	w := requestHTTP(h, "POST", "/api/v1/conversations/"+th.ID+"/messages", "alice", `{"body":"literal � and \\\\ud800"}`, "application/json", "audit-json-valid", true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "literal") {
		t.Fatal(w.Body.String())
	}
}

func TestAuditNoJSModerationAndRetention(t *testing.T) {
	f := setup(t)
	th := f.thread()
	f.send(th.ID, "<script>evidence</script>", "audit-admin-send")
	if e := f.s.Report(ctx, bob, th.ID, 1, "<script>reason</script>"); e != nil {
		t.Fatal(e)
	}
	h := testHTTP(t, f.s)
	path := "/conversations/" + th.ID + "/moderation"
	if w := requestHTTP(h, "GET", path, "bob", "", "", "", false); w.Code != 403 || strings.Contains(w.Body.String(), "evidence") {
		t.Fatal(w.Code, w.Body.String())
	}
	w := requestHTTP(h, "GET", path, "moderator", "", "", "", false)
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") || !strings.Contains(w.Body.String(), "Legal hold") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/moderate", "moderator", "csrf=valid&action=resolve&sequence=1&reporter=bob", "application/x-www-form-urlencoded", "", false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, actor := range []struct {
		name string
		code int
	}{{"bob", 403}, {"moderator", 200}} {
		if w = requestHTTP(h, "GET", "/messaging-maintenance", actor.name, "", "", "", false); w.Code != actor.code {
			t.Fatal(actor, w.Code, w.Body.String())
		}
	}
	f.now = f.now.Add(24 * time.Hour)
	if w = requestHTTP(h, "POST", "/messaging-maintenance", "bob", "csrf=valid&action=shorten&days=1", "application/x-www-form-urlencoded", "", false); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = requestHTTP(h, "POST", "/messaging-maintenance", "moderator", "csrf=valid&action=shorten&days=1", "application/x-www-form-urlencoded", "", false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, e := f.s.Messages(ctx, bob, th.ID, 0, 100); !errors.Is(e, ErrDenied) {
		t.Fatal("shortened deadline ignored", e)
	}
	if e := f.s.ShortenRetention(ctx, mod, 30); e != nil {
		t.Fatal(e)
	}
	var days int
	if e := f.s.db.QueryRow(`SELECT retention FROM message_threads WHERE network='network' AND id=?`, th.ID).Scan(&days); e != nil || days != 1 {
		t.Fatal("retention extended", days, e)
	}
	if w = requestHTTP(h, "POST", "/messaging-maintenance", "moderator", "csrf=valid&action=purge", "application/x-www-form-urlencoded", "", false); w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	var count int
	if e := f.s.db.QueryRow(`SELECT count(*) FROM message_entries`).Scan(&count); e != nil || count != 0 {
		t.Fatal(count, e)
	}
}

func TestAuditMethodContracts(t *testing.T) {
	f := setup(t)
	h := testHTTP(t, f.s)
	th := f.thread()
	for _, c := range []struct{ method, path, allow string }{{"GET", "/api/v1/purge", "POST"}, {"GET", "/api/v1/block", "POST"}, {"POST", "/api/v1/conversations/" + th.ID + "/export", "GET"}, {"GET", "/api/v1/conversations/" + th.ID + "/hide", "POST"}, {"DELETE", "/api/v1/retention", "POST"}} {
		w := requestHTTP(h, c.method, c.path, "alice", `{}`, "application/json", "", true)
		if w.Code != 405 || w.Header().Get("Allow") != c.allow || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatal(c, w.Code, w.Header(), w.Body.String())
		}
	}
}

func FuzzStrictJSONDecode(f *testing.F) {
	for _, seed := range []string{`{"body":"hello"}`, `{"body":"\\ud800"}`, `{"body":"\ud83d\ude00"}`, `{"body":"\ud800"}`, `{"body":"a","BODY":"b"}`, `null`, "{\"body\":\"" + string([]byte{255}) + "\"}"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 65537 {
			t.Skip()
		}
		r := httptest.NewRequest("POST", "/", strings.NewReader(input))
		r.Header.Set("Content-Type", "application/json")
		var out struct {
			Body string `json:"body"`
		}
		if e := decode(r, &out); e == nil && (!json.Valid([]byte(input)) || !utf8.ValidString(input) || !utf8.ValidString(out.Body)) {
			t.Fatal("accepted invalid JSON Unicode")
		}
	})
}

func TestAuditActionsRemainLocalForHostilePaths(t *testing.T) {
	f := setup(t)
	th := f.thread()
	h := testHTTP(t, f.s)
	for _, path := range []string{"//outside.example/private", "//conversations/" + th.ID + "/moderation", "/conversations/" + th.ID + "/moderation"} {
		w := requestHTTP(h, "GET", path, "moderator", "", "", "", false)
		if strings.Contains(w.Body.String(), `action="//`) || strings.Contains(w.Body.String(), `action="https:`) || strings.Contains(w.Body.String(), `action="http:`) {
			t.Fatal("external form action", path, w.Body.String())
		}
		if w.Code == 200 && strings.Contains(w.Body.String(), "Legal hold") && !strings.Contains(w.Body.String(), `action="/conversations/`+th.ID+`/moderate"`) {
			t.Fatal("expected fixed local action", w.Body.String())
		}
	}
}
