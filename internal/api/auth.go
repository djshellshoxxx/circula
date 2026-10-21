package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// SessionCookieNames are the cookie names the sign-in flow may use.
var SessionCookieNames = []string{"__Host-circula_session", "circula_session"}

// CSRFToken derives the per-session CSRF token for a session token. The HTML
// sign-in flow may hand it to scripts; the API also returns it in the
// X-CSRF-Token response header of every cookie-authenticated response (API-014).
func CSRFToken(sessionToken string) string {
	m := hmac.New(sha256.New, []byte("circula-csrf-v1"))
	m.Write([]byte(sessionToken))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func isUnsafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

// authenticate resolves the caller from a bearer API token or a session cookie.
func (s *server) authenticate(r *http.Request, c *reqCtx) error {
	if h := r.Header.Get("Authorization"); h != "" {
		scheme, tok, ok := strings.Cut(h, " ")
		tok = strings.TrimSpace(tok)
		if !ok || !strings.EqualFold(scheme, "Bearer") || tok == "" {
			return errUnauth
		}
		a, err := s.svc.AuthenticateToken(r.Context(), tok)
		if err != nil {
			return errUnauth
		}
		a.RequestID = c.rid
		c.actor = a
		return nil
	}
	for _, name := range SessionCookieNames {
		ck, err := r.Cookie(name)
		if err != nil || ck.Value == "" {
			continue
		}
		a, err := s.svc.Authenticate(r.Context(), ck.Value)
		if err != nil {
			return errUnauth
		}
		a.RequestID = c.rid
		c.actor, c.cookie, c.session = a, true, ck.Value
		return nil
	}
	return errUnauth
}

// checkCSRF enforces API-014 for cookie-authenticated unsafe requests.
func checkCSRF(r *http.Request, c *reqCtx) error {
	if !c.cookie || !isUnsafe(r.Method) {
		return nil
	}
	got := r.Header.Get("X-CSRF-Token")
	want := CSRFToken(c.session)
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return &apiError{status: 403, code: "csrf-invalid", detail: "missing or invalid CSRF token"}
	}
	return nil
}

// resolveNetwork picks the network of the call: header X-Network (id or slug),
// else the caller's only membership (OQ-API-02: deployment-level context).
func (s *server) resolveNetwork(r *http.Request, c *reqCtx) error {
	ms, err := s.svc.MyMemberships(r.Context(), c.actor)
	if err != nil {
		return err
	}
	var cand []*domain.Member
	for _, m := range ms {
		if c.actor.NetworkScope == "" || c.actor.NetworkScope == m.NetworkID {
			cand = append(cand, m)
		}
	}
	c.memberships = cand
	want := strings.TrimSpace(r.Header.Get("X-Network"))
	if want != "" {
		for _, m := range cand {
			if m.NetworkID == want {
				c.netID, c.member = m.NetworkID, m
				return nil
			}
		}
		if info, err := s.svc.NetworkBySlug(r.Context(), want); err == nil {
			for _, m := range cand {
				if m.NetworkID == info.ID {
					c.netID, c.member = m.NetworkID, m
					return nil
				}
			}
		}
		return notFoundErr()
	}
	switch len(cand) {
	case 0:
		return notFoundErr()
	case 1:
		c.netID, c.member = cand[0].NetworkID, cand[0]
		return nil
	}
	return &apiError{status: 400, code: "network-required", detail: "member of several networks: send the X-Network header"}
}
