package governance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type masterAuth struct {
	authorize func(*http.Request, bool) (Identity, error)
	token     func(*http.Request) string
}

func (a masterAuth) Authorize(r *http.Request, b bool) (Identity, error) { return a.authorize(r, b) }
func (a masterAuth) CSRF(r *http.Request) string                         { return a.token(r) }

func TestMasterStrictQueryBeforeHost(t *testing.T) {
	calls := 0
	auth := masterAuth{authorize: func(*http.Request, bool) (Identity, error) { calls++; return Identity{"n", "a"}, nil }, token: func(*http.Request) string { return "token" }}
	w := Web{Auth: auth, CursorKey: []byte(strings.Repeat("k", 32))}
	for _, method := range []string{"GET", "POST"} {
		for _, query := range []string{"proposal=a&proposal=b", "format=html&unknown=1", "proposal=a&bad=%zz", "format=xml", "proposal=a&cursor=b", "cursor=" + strings.Repeat("a", 8193)} {
			req := httptest.NewRequest(method, "/governance?"+query, strings.NewReader("action=create"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			out := httptest.NewRecorder()
			w.ServeHTTP(out, req)
			if out.Code != 400 || calls != 0 || req.Form != nil {
				t.Fatalf("%s query validation late: status=%d calls=%d", method, out.Code, calls)
			}
		}
	}
}

func TestMasterCallbacksReauthorizePrivatePages(t *testing.T) {
	for _, page := range []string{"detail", "list", "error"} {
		for _, change := range []string{"revoke", "replace", "member", "cancel", "unchanged"} {
			t.Run(page+"/"+change, func(t *testing.T) {
				s, authority, now, _ := newFixture(t)
				proposal := opened(t, s, *now)
				command(t, s, "b", Command{Action: "vote", Proposal: proposal.Proposal, Intent: "ballot", Choice: Yes, Revision: proposal.Revision})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				current := Identity{"n", "b"}
				denied := false
				auth := masterAuth{authorize: func(*http.Request, bool) (Identity, error) {
					if denied {
						return Identity{}, errors.New("denied")
					}
					return current, nil
				}, token: func(*http.Request) string {
					switch change {
					case "revoke":
						denied = true
					case "replace":
						current.Member = "a"
					case "member":
						authority.mu.Lock()
						authority.members["b"] = Principal{false, false, true}
						authority.mu.Unlock()
					case "cancel":
						cancel()
					}
					return "token"
				}}
				w := Web{s, auth, []byte(strings.Repeat("k", 32))}
				path := "/governance?format=html"
				method := "GET"
				body := ""
				if page != "list" {
					path += "&proposal=" + proposal.Proposal
				}
				if page == "error" {
					method = "POST"
					body = url.Values{"action": {"invalid"}, "proposal": {proposal.Proposal}, "intent": {"failure"}}.Encode()
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				out := httptest.NewRecorder()
				w.ServeHTTP(out, req)
				if change != "unchanged" {
					if out.Code != 403 || strings.Contains(out.Body.String(), "Should we update?") || strings.Contains(out.Body.String(), "Your recorded choice") {
						t.Fatalf("private page exposed: %d %s", out.Code, out.Body.String())
					}
				} else {
					want := 200
					if page == "error" {
						want = 409
					}
					if out.Code != want || !strings.Contains(out.Body.String(), "Should we update?") {
						t.Fatalf("valid page failed: %d %s", out.Code, out.Body.String())
					}
				}
			})
		}
	}
}

func TestMasterNativeFormQueryCompatibility(t *testing.T) {
	for _, queryKind := range []string{"same", "different", "badcursor"} {
		t.Run(queryKind, func(t *testing.T) {
			s, _, now, _ := newFixture(t)
			p := opened(t, s, *now)
			w := Web{s, webFixture{}, []byte(strings.Repeat("k", 32))}
			query := "?proposal=" + p.Proposal + "&format=html"
			if queryKind == "different" {
				query = "?proposal=other&format=html"
			}
			if queryKind == "badcursor" {
				query = "?cursor=bad&format=html"
			}
			body := url.Values{"action": {"vote"}, "proposal": {p.Proposal}, "intent": {"form-vote"}, "revision": {"2"}, "choice": {"abstain"}, "csrf": {"csrf"}}
			req := httptest.NewRequest("POST", "/governance"+query, strings.NewReader(body.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Test-Member", "b")
			req.Header.Set("X-Test-CSRF", "valid")
			out := httptest.NewRecorder()
			w.ServeHTTP(out, req)
			d, err := s.Read(context.Background(), Identity{"n", "b"}, p.Proposal)
			if err != nil {
				t.Fatal(err)
			}
			if queryKind == "same" {
				if out.Code != 303 || d.OwnVote == nil {
					t.Fatalf("native form failed:%d %s", out.Code, out.Body.String())
				}
			} else if out.Code != 400 || d.OwnVote != nil {
				t.Fatalf("invalid query mutated:%d vote=%v", out.Code, d.OwnVote)
			}
		})
	}
	s, _, _, _ := newFixture(t)
	w := Web{s, webFixture{}, []byte(strings.Repeat("k", 32))}
	req := httptest.NewRequest("GET", "/governance?cursor=&format=html", nil)
	req.Header.Set("X-Test-Member", "a")
	out := httptest.NewRecorder()
	w.ServeHTTP(out, req)
	if out.Code != 200 {
		t.Fatalf("empty cursor compatibility: %d", out.Code)
	}
}

func TestMasterHostCannotHideMalformedForm(t *testing.T) {
	for _, suffix := range []string{"&ignored=%zz", "&title=duplicate", "&unexpected=1"} {
		s, _, _, _ := newFixture(t)
		calls := 0
		auth := masterAuth{authorize: func(r *http.Request, _ bool) (Identity, error) {
			calls++
			_ = r.PostFormValue("csrf") // Typical cookie-CSRF lookup discards ParseForm error.
			return Identity{"n", "a"}, nil
		}, token: func(*http.Request) string { return "token" }}
		web := Web{s, auth, []byte(strings.Repeat("k", 32))}
		body := url.Values{"action": {"create"}, "intent": {"create-master"}, "title": {"Private title"}, "body": {"Body"}, "category": {"policy"}, "csrf": {"token"}}.Encode() + suffix
		request := httptest.NewRequest("POST", "/governance", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		web.ServeHTTP(out, request)
		proposals, err := s.List(context.Background(), Identity{"n", "a"}, "", 20)
		if err != nil {
			t.Fatal(err)
		}
		if out.Code != 400 || calls != 0 || len(proposals) != 0 {
			t.Fatalf("malformed form reached host/mutation: status=%d calls=%d proposals=%d", out.Code, calls, len(proposals))
		}
	}
}
