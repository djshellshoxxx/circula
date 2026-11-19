package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchVisibilityPagingRestartAndRetention(t *testing.T) {
	f := setup(t)
	th := f.thread()
	for i, body := range []string{"private 100%_ café", "unmatched", "private café second", "PRIVATE distinct"} {
		f.send(th.ID, body, fmt.Sprintf("search-%02d", i))
	}
	first, err := f.s.Search(ctx, bob, th.ID, "café", 0, 1)
	if err != nil || len(first) != 1 || first[0].Sequence != 1 {
		t.Fatal(first, err)
	}
	if err = f.s.Close(); err != nil {
		t.Fatal(err)
	}
	f.open()
	second, err := f.s.Search(ctx, bob, th.ID, "café", first[0].Sequence, 1)
	if err != nil || len(second) != 1 || second[0].Sequence != 3 {
		t.Fatal(second, err)
	}
	for _, query := range []string{"%_", "100%_"} {
		rows, err := f.s.Search(ctx, bob, th.ID, query, 0, 100)
		if err != nil || len(rows) != 1 || rows[0].Sequence != 1 {
			t.Fatal(rows, err)
		}
	}
	rows, err := f.s.Search(ctx, bob, th.ID, "' OR 1=1 --", 0, 100)
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	for _, actor := range []Actor{{"other", "alice"}, {"network", "carol"}, {}} {
		if _, err := f.s.Search(ctx, actor, th.ID, "private", 0, 100); !errors.Is(err, ErrDenied) {
			t.Fatal(actor, err)
		}
	}
	if err := f.s.Hide(ctx, bob, th.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = f.s.Search(ctx, bob, th.ID, "private", 0, 100)
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	rows, err = f.s.Search(ctx, alice, th.ID, "private", 0, 100)
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	if _, err = f.s.db.Exec(`UPDATE memberships SET active=0 WHERE member='alice'`); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Search(ctx, alice, th.ID, "private", 0, 100); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
	if _, err = f.s.db.Exec(`UPDATE memberships SET active=1 WHERE member='alice'`); err != nil {
		t.Fatal(err)
	}
	if err = f.s.Moderate(ctx, mod, th.ID, "hold", 0, ""); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(30 * 24 * time.Hour)
	if _, err = f.s.Search(ctx, alice, th.ID, "private", 0, 100); !errors.Is(err, ErrDenied) {
		t.Fatal("expired held content searchable", err)
	}
}

func TestSearchValidation(t *testing.T) {
	f := setup(t)
	th := f.thread()
	for _, query := range []string{"", "  ", "\xff", "\x00", "line\nbreak", strings.Repeat("é", 65)} {
		if _, err := f.s.Search(ctx, alice, th.ID, query, 0, 100); !errors.Is(err, ErrInvalid) {
			t.Fatal(query, err)
		}
	}
	for _, bounds := range [][2]int{{-1, 100}, {0, 0}, {0, 101}} {
		if _, err := f.s.Search(ctx, alice, th.ID, "valid", int64(bounds[0]), bounds[1]); !errors.Is(err, ErrInvalid) {
			t.Fatal(bounds, err)
		}
	}
}

func TestSearchDoesNotExtendRetention(t *testing.T) {
	f := setup(t)
	th := f.thread()
	f.send(th.ID, "private content", "retention-search")
	f.now = f.now.Add(30*24*time.Hour - time.Second)
	rows, err := f.s.Search(ctx, bob, th.ID, "private", 0, 100)
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	f.now = f.now.Add(time.Second)
	rows, err = f.s.Search(ctx, bob, th.ID, "private", 0, 100)
	if !errors.Is(err, ErrDenied) || len(rows) != 0 {
		t.Fatal("search must not prolong content visibility", rows, err)
	}
}

func TestSearchAPIAndNoJSPrivacy(t *testing.T) {
	f := setup(t)
	th := f.thread()
	query := `<script>private</script>`
	for i := 0; i < 101; i++ {
		f.now = f.now.Add(2 * time.Second)
		f.send(th.ID, query, fmt.Sprintf("search-%03d", i))
	}
	h := testHTTP(t, f.s)
	path := "/api/v1/conversations/" + th.ID + "/search"
	body, _ := json.Marshal(map[string]any{"query": query, "after": 100})
	w := requestHTTP(h, "POST", path, "bob", string(body), "application/json", "", true)
	var messages []Message
	if err := json.Unmarshal(w.Body.Bytes(), &messages); err != nil || w.Code != 200 || len(messages) != 1 || messages[0].Sequence != 101 {
		t.Fatal(w.Code, messages, err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Header())
	}
	for _, tc := range []struct {
		method, member, body string
		csrf                 bool
		status               int
	}{
		{"GET", "bob", "", true, 405},
		{"POST", "carol", string(body), true, 403},
		{"POST", "bob", string(body), false, 403},
		{"POST", "bob", `{"query":"private","sender":"alice"}`, true, 400},
		{"POST", "bob", `{"query":"private","after":-1}`, true, 400},
	} {
		w := requestHTTP(h, tc.method, path, tc.member, tc.body, "application/json", "", tc.csrf)
		if w.Code != tc.status {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	form := url.Values{"csrf": {"valid"}, "query": {query}}
	w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/search", "bob", form.Encode(), "application/x-www-form-urlencoded", "", false)
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") || !strings.Contains(w.Body.String(), "Next search results") || strings.Contains(w.Body.String(), "?query=") || w.Header().Get("Location") != "" {
		t.Fatal(w.Code, w.Body.String())
	}
	form.Set("after", "100")
	w = requestHTTP(h, "POST", "/conversations/"+th.ID+"/search", "bob", form.Encode(), "application/x-www-form-urlencoded", "", false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `value="101"`) || strings.Contains(w.Body.String(), "Next search results") {
		t.Fatal(w.Code, w.Body.String())
	}
	threads, err := f.s.Threads(ctx, bob, "", 100)
	if err != nil || len(threads) != 1 || threads[0].Unread != 101 {
		t.Fatal("search changed read position", threads, err)
	}
}
