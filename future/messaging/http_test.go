package messaging

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func testHTTP(t *testing.T, s *Store) http.Handler {
	t.Helper()
	h, e := Handler(s, HTTPAuth{
		Resolve: func(r *http.Request) (Actor, error) {
			m := r.Header.Get("Test-Member")
			if m == "" {
				return Actor{}, ErrDenied
			}
			return Actor{"network", m}, nil
		},
		VerifyCSRF: func(r *http.Request) error {
			if r.Header.Get("Test-CSRF") == "valid" || r.PostFormValue("csrf") == "valid" {
				return nil
			}
			return ErrDenied
		},
		Token: func(*http.Request) (string, error) { return "valid", nil },
	})
	if e != nil {
		t.Fatal(e)
	}
	return h
}
func requestHTTP(h http.Handler, method, path, member, body, contentType, key string, csrf bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Test-Member", member)
	r.Header.Set("Content-Type", contentType)
	r.Header.Set("Idempotency-Key", key)
	if csrf {
		r.Header.Set("Test-CSRF", "valid")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestNoJSMessagingFormsAndEscaping(t *testing.T) {
	f := setup(t)
	h := testHTTP(t, f.s)
	th := f.thread()
	f.send(th.ID, `<script>alert("x")</script><img src="https://attacker.invalid">`, "message-html")
	w := requestHTTP(h, "GET", "/conversations/"+th.ID, "bob", "", "", "", false)
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") || !strings.Contains(w.Body.String(), "not end-to-end encrypted") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, want := range []string{`name="csrf" value="valid"`, `/send`, "/read", "/hide", "/report", "/close"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatal(want)
		}
	}
	data := url.Values{"csrf": {"valid"}, "key": {"message-form"}, "body": {"no JS needed"}}
	w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/send", "bob", data.Encode(), "application/x-www-form-urlencoded", "", false)
	if w.Code != http.StatusSeeOther {
		t.Fatal(w.Code, w.Body.String())
	}
	msgs, e := f.s.Messages(ctx, alice, th.ID, 0, 100)
	if e != nil || len(msgs) != 2 || msgs[1].Sender != "bob" {
		t.Fatal(msgs, e)
	}
	w = requestHTTP(h, "GET", "/conversations", "bob", "", "", "", false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "unread") {
		t.Fatal(w.Code)
	}
	data = url.Values{"csrf": {"valid"}, "sequence": {"2"}}
	w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/read", "bob", data.Encode(), "application/x-www-form-urlencoded", "", false)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	data = url.Values{"csrf": {"valid"}}
	w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/hide", "bob", data.Encode(), "application/x-www-form-urlencoded", "", false)
	if w.Code != 303 {
		t.Fatal(w.Code)
	}
	w = requestHTTP(h, "GET", "/api/v1/conversations/"+th.ID+"/export", "bob", "", "", "", false)
	if w.Code != 200 || strings.Contains(w.Body.String(), "no JS needed") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestHTTPAPIAuthCSRFBoundariesAndPrivateErrors(t *testing.T) {
	f := setup(t)
	h := testHTTP(t, f.s)
	th := f.thread()
	for _, tc := range []struct {
		method, path, member, body, content string
		csrf                                bool
		status                              int
	}{
		{"GET", "/api/v1/conversations", "", "", "", false, 403},
		{"GET", "/conversations/" + th.ID, "carol", "", "", false, 403},
		{"GET", "/api/v1/conversations/" + th.ID + "/messages", "carol", "", "", false, 403},
		{"POST", "/api/v1/conversations/" + th.ID + "/messages", "alice", `{"body":"hi"}`, "application/json", false, 403},
		{"POST", "/api/v1/conversations/" + th.ID + "/messages", "alice", `{"body":"hi","actor":"bob"}`, "application/json", true, 400},
		{"POST", "/api/v1/conversations/" + th.ID + "/messages", "alice", `{"body":"hi"} {}`, "application/json", true, 400},
		{"POST", "/api/v1/conversations/" + th.ID + "/messages", "alice", strings.Repeat("x", 65537), "application/json", true, 400},
		{"POST", "/api/v1/conversations/" + th.ID + "/messages", "alice", `{"body":"hi"}`, "text/plain", true, 400},
		{"GET", "/api/v1/conversations/" + th.ID + "/messages?after=-1", "alice", "", "", false, 400},
		{"POST", "/api/v1/purge", "alice", `{}`, "application/json", true, 403},
		{"POST", "/block", "alice", "target=bob&enabled=true", "application/x-www-form-urlencoded", false, 403},
		{"DELETE", "/api/v1/conversations", "alice", "", "", true, 405},
	} {
		w := requestHTTP(h, tc.method, tc.path, tc.member, tc.body, tc.content, "message-api", tc.csrf)
		if w.Code != tc.status {
			t.Fatal(tc.method, tc.path, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal(w.Header())
		}
		if strings.Contains(w.Body.String(), "sql") || strings.Contains(w.Body.String(), "hi") {
			t.Fatal("raw detail leaked", w.Body.String())
		}
	}
	w := requestHTTP(h, "POST", "/api/v1/conversations/"+th.ID+"/messages", "alice", `{"body":"API body"}`, "application/json", "message-api", true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result Message
	if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil || result.Sequence != 1 {
		t.Fatal(result, e)
	}
}
func TestHTTPFactoriesFailClosed(t *testing.T) {
	f := setup(t)
	if _, e := Handler(f.s, HTTPAuth{}); e == nil {
		t.Fatal("missing auth accepted")
	}
}

func TestMigrationBackupRestoreAndEmptyHide(t *testing.T) {
	f := setup(t)
	if e := f.s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	th := f.thread()
	if e := f.s.Hide(ctx, bob, th.ID); e != nil {
		t.Fatal(e)
	}
	if threads, e := f.s.Threads(ctx, bob, "", 100); e != nil || len(threads) != 0 {
		t.Fatal(threads, e)
	}
	f.send(th.ID, "backup survives", "message-backup")
	if _, e := f.s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); e != nil {
		t.Fatal(e)
	}
	// SQLite backup API equivalent: a consistent VACUUM INTO snapshot while live.
	backup := f.path + ".backup"
	if _, e := f.s.db.Exec(`VACUUM INTO ?`, backup); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(backup, 0600); e != nil {
		t.Fatal(e)
	}
	assertPrivateBackup(t, backup)
	oldPath := f.path
	f.s.Close()
	f.path = backup
	f.open()
	var b bytes.Buffer
	if e := f.s.Export(ctx, bob, th.ID, &b); e != nil {
		t.Fatal(e)
	}
	data, e := io.ReadAll(&b)
	if e != nil || !strings.Contains(string(data), "backup survives") {
		t.Fatal(string(data), e)
	}
	f.s.Close()
	f.path = oldPath
	f.open()
}

func FuzzPrepare(f *testing.F) {
	for _, s := range []string{"hello", "<script>", "\x00", "\xff", "\n\t"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		th := Thread{Network: "n", A: "a", B: "b"}
		out, e := Prepare(th, Access{"n", "a", true}, body)
		if e == nil && out != body {
			t.Fatal("body changed")
		}
	})
}
