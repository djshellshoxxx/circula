package messaging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallbackAuditMalformedFormBeforeResolve(t *testing.T) {
	for _, suffix := range []string{"&discarded=%zz", "&discarded=one;two"} {
		f := setup(t)
		th := f.thread()
		calls := 0
		h, err := Handler(f.s, HTTPAuth{Resolve: func(r *http.Request) (Actor, error) { calls++; _ = r.PostFormValue("csrf"); return bob, nil }, VerifyCSRF: func(*http.Request) error { return nil }, Token: func(*http.Request) (string, error) { return "token", nil }})
		if err != nil {
			t.Fatal(err)
		}
		w := requestHTTP(h, "POST", "/conversations/"+th.ID+"/send", "bob", "csrf=token&key=valid-key&body=malformed-send"+suffix, "application/x-www-form-urlencoded", "", false)
		rows, err := f.s.Messages(ctx, alice, th.ID, 0, 100)
		if w.Code != 400 || calls != 0 || err != nil || len(rows) != 0 {
			t.Fatalf("malformed form reached host/mutation: status=%d calls=%d rows=%d err=%v", w.Code, calls, len(rows), err)
		}
	}
}

func TestCallbackAuditTokenReauthorizesPrivatePages(t *testing.T) {
	for _, page := range []string{"inbox", "thread", "search", "admin"} {
		for _, change := range []string{"revoke", "replace", "member", "cancel", "unchanged"} {
			t.Run(page+"/"+change, func(t *testing.T) {
				f := setup(t)
				th := f.thread()
				f.send(th.ID, "private-callback-canary", "canary-key")
				actor := bob
				if page == "admin" {
					actor = mod
				}
				current := actor
				denied := false
				requestCtx, cancel := context.WithCancel(context.Background())
				defer cancel()
				h, err := Handler(f.s, HTTPAuth{Resolve: func(*http.Request) (Actor, error) {
					if denied {
						return Actor{}, ErrDenied
					}
					return current, nil
				}, VerifyCSRF: func(*http.Request) error { return nil }, Token: func(*http.Request) (string, error) {
					switch change {
					case "revoke":
						denied = true
					case "replace":
						current = alice
					case "member":
						if _, e := f.s.db.Exec(`UPDATE memberships SET active=0 WHERE member=?`, actor.Member); e != nil {
							t.Fatal(e)
						}
					case "cancel":
						cancel()
					}
					return "private-token", nil
				}})
				if err != nil {
					t.Fatal(err)
				}
				method, path, body := "GET", "/conversations", ""
				switch page {
				case "thread":
					path += "/" + th.ID
				case "search":
					method = "POST"
					path += "/" + th.ID + "/search"
					body = "query=private&csrf=token"
				case "admin":
					path = "/messaging-maintenance"
				}
				r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(requestCtx)
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if change == "unchanged" {
					if w.Code != 200 {
						t.Fatalf("valid page failed:%d %s", w.Code, w.Body.String())
					}
				} else if w.Code != 403 || strings.Contains(w.Body.String(), "private-callback-canary") || strings.Contains(w.Body.String(), "private-token") {
					t.Fatalf("stale page exposed:%d %s", w.Code, w.Body.String())
				}
			})
		}
	}
}

func TestCallbackAuditCSRFReauthorizesBeforeSend(t *testing.T) {
	for _, change := range []string{"revoke", "replace", "member", "cancel", "unchanged"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t)
			th := f.thread()
			current := bob
			denied := false
			requestCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h, err := Handler(f.s, HTTPAuth{Resolve: func(*http.Request) (Actor, error) {
				if denied {
					return Actor{}, ErrDenied
				}
				return current, nil
			}, VerifyCSRF: func(*http.Request) error {
				switch change {
				case "revoke":
					denied = true
				case "replace":
					current = alice
				case "member":
					if _, e := f.s.db.Exec(`UPDATE memberships SET active=0 WHERE member='bob'`); e != nil {
						t.Fatal(e)
					}
				case "cancel":
					cancel()
				}
				return nil
			}, Token: func(*http.Request) (string, error) { return "token", nil }})
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/conversations/"+th.ID+"/send", strings.NewReader("body=private-new-message&key=callback-send&csrf=token")).WithContext(requestCtx)
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			var count int
			if err := f.s.db.QueryRow(`SELECT count(*) FROM message_entries`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if change == "unchanged" {
				if w.Code != 303 || count != 1 {
					t.Fatalf("valid send failed:%d count%d", w.Code, count)
				}
			} else if w.Code != 403 || count != 0 {
				t.Fatalf("stale session mutated:%d count%d", w.Code, count)
			}
		})
	}
}

func TestCallbackAuditQueryBoundaryBeforeResolve(t *testing.T) {
	f := setup(t)
	calls := 0
	h, err := Handler(f.s, HTTPAuth{Resolve: func(r *http.Request) (Actor, error) { calls++; _ = r.URL.Query(); return Actor{}, ErrDenied }, VerifyCSRF: func(*http.Request) error { return nil }, Token: func(*http.Request) (string, error) { return "token", nil }})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST"} {
		for _, query := range []string{"after=" + strings.Repeat("x", 8193), "after=1&discarded=%zz", "after=1;discarded=2"} {
			w := requestHTTP(h, method, "/conversations?"+query, "bob", "csrf=token", "application/x-www-form-urlencoded", "", false)
			if w.Code != 400 || calls != 0 {
				t.Fatalf("invalid query reached host: method=%s status=%d calls=%d", method, w.Code, calls)
			}
		}
	}
	// The exact byte boundary still reaches normal host validation. This guard
	// does not invent a new route field allowlist or remove pagination fields.
	w := requestHTTP(h, "GET", "/conversations?after="+strings.Repeat("x", 8192-len("after=")), "bob", "", "", "", false)
	if w.Code != 403 || calls != 1 {
		t.Fatalf("exact query boundary changed: status=%d calls=%d", w.Code, calls)
	}
}
