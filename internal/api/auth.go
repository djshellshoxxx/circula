package api

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
)

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
	if tok := httpx.SessionToken(r); tok != "" {
		a, err := s.svc.Authenticate(r.Context(), tok)
		if err != nil {
			return errUnauth
		}
		a.RequestID = c.rid
		c.actor, c.cookie, c.session = a, true, tok
		return nil
	}
	return errUnauth
}

// checkCSRF enforces API-014 for cookie-authenticated unsafe requests.
func (s *server) checkCSRF(r *http.Request, c *reqCtx) error {
	if !c.cookie || !isUnsafe(r.Method) {
		return nil
	}
	if !s.csrf.Valid(c.session, r.Header.Get("X-CSRF-Token")) {
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
