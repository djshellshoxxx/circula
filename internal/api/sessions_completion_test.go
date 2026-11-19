package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
)

func TestOwnSessionsAPIRejectsMalformedSignedPosition(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	w := newWorldOpts(t, api.Options{CSRFKey: testCSRFKey, CursorKey: key})
	raw, actor := apiSession(t, w, "alice")
	for _, position := range []string{"malformed", "01:" + strings.Repeat("a", 64), "1:" + strings.Repeat("G", 64)} {
		payload, err := json.Marshal(map[string]string{
			"p": position,
			"f": "/api/v1/me/sessions||" + actor.PersonID + "|session-public-id-v2",
		})
		if err != nil {
			t.Fatal(err)
		}
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write(payload)
		token := base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		wantProblem(t, sessionRequest(w, raw, "GET", "/me/sessions?cursor="+url.QueryEscape(token), false),
			http.StatusBadRequest, "invalid-cursor")
	}
}

func TestOwnSessionsAPIBulkRevocationKeepsCurrentAndRejectsTokens(t *testing.T) {
	w := newWorld(t)
	raw, _ := apiSession(t, w, "alice")
	otherRaw, _ := apiSession(t, w, "alice")
	foreignRaw, _ := apiSession(t, w, "bob")
	const path = "/me/sessions/revoke-others"
	if r := sessionRequest(w, raw, "POST", path, false); r.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF: %d", r.Code)
	}
	if r := w.do(req{method: "POST", path: "/api/v1" + path, token: w.aliceT}); r.Code != http.StatusForbidden {
		t.Fatalf("API token revoked other sessions: %d", r.Code)
	}
	if r := w.do(req{method: "POST", path: "/api/v1" + path}); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous revocation: %d", r.Code)
	}
	for i := 0; i < 2; i++ {
		if r := sessionRequest(w, raw, "POST", path, true); r.Code != http.StatusNoContent {
			t.Fatalf("bulk revocation %d: %d %s", i, r.Code, r.Body.String())
		}
	}
	if _, err := w.env.Svc.Authenticate(w.env.Ctx, otherRaw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Fatalf("other session survived: %v", err)
	}
	for _, live := range []string{raw, foreignRaw} {
		if _, err := w.env.Svc.Authenticate(w.env.Ctx, live); err != nil {
			t.Fatalf("current or foreign session ended: %v", err)
		}
	}
	page := decode[ownSessionPage](t, sessionRequest(w, raw, "GET", "/me/sessions", false))
	if len(page.Data) != 1 || !page.Data[0].Current || page.NextCursor != nil {
		t.Fatalf("inventory after bulk revoke: %+v", page)
	}
	if r := w.get(w.aliceT, "/me"); r.Code != http.StatusOK {
		t.Fatalf("API token stopped working after session revocation: %d", r.Code)
	}
}
