package web

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/service"
)

const (
	cookieCSRF  = httpx.CSRFCookie
	cookieFlash = "cm_flash"
	cookieStart = "cm_sstart" // signed session start time, drives the expiry warning only
)

// errCSRF is the pseudo status renderError uses for a rejected form token.
const errCSRF = -403

func (s *server) secure(r *http.Request) bool { return s.opts.Edge.Secure(r) }

// clientIP is the address used for rate limiting (trusted proxies only).
func (s *server) clientIP(r *http.Request) string { return s.opts.Edge.ClientIP(r) }

func (s *server) setCookie(w http.ResponseWriter, r *http.Request, name, val string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: val, Path: "/", MaxAge: maxAge,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secure(r)})
}

// setSession stores the session token under the name shared with the API.
func (s *server) setSession(w http.ResponseWriter, r *http.Request, tok string) {
	s.setCookie(w, r, httpx.SessionCookieName(s.secure(r)), tok, 0)
	now := strconv.FormatInt(s.opts.Clock.Now().Unix(), 10)
	s.setCookie(w, r, cookieStart, now+"."+s.mac("start"+now), 0)
}

// clearSession removes the session cookies (every spelling).
func (s *server) clearSession(w http.ResponseWriter, r *http.Request) {
	if s.secure(r) {
		s.setCookie(w, r, httpx.SessionCookieSecure, "", -1)
	}
	s.setCookie(w, r, httpx.SessionCookie, "", -1)
	s.setCookie(w, r, cookieStart, "", -1)
}

func (s *server) mac(v string) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(v))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// ensureCSRF guarantees a csrf cookie. Before sign-in the form token is an HMAC
// of that cookie's value; once signed in it is an HMAC of the session token
// (httpx.CSRF, the same scheme the API uses), so a planted cookie cannot mint
// tokens for a session.
func (s *server) ensureCSRF(w http.ResponseWriter, r *http.Request) context.Context {
	if c, err := r.Cookie(cookieCSRF); err == nil && len(c.Value) == 32 {
		return context.WithValue(r.Context(), keyCSRF, c.Value)
	}
	v := randHex(16)
	s.setCookie(w, r, cookieCSRF, v, 0)
	return context.WithValue(r.Context(), keyCSRF, v)
}

// rotateCSRF issues a fresh csrf secret (on login and logout).
func (s *server) rotateCSRF(w http.ResponseWriter, r *http.Request) {
	s.setCookie(w, r, cookieCSRF, randHex(16), 0)
}

func (s *server) csrfToken(r *http.Request) string {
	if t := httpx.SessionToken(r); t != "" {
		return s.csrf.Token(t)
	}
	v, _ := r.Context().Value(keyCSRF).(string)
	return s.csrf.Token(v)
}

func (s *server) csrfOK(r *http.Request) bool {
	if !s.sameOrigin(r) {
		return false
	}
	if c, err := r.Cookie(cookieCSRF); err != nil || len(c.Value) != 32 {
		if httpx.SessionToken(r) == "" {
			return false
		}
	}
	got := r.Header.Get("X-CSRF-Token")
	if got == "" {
		got = r.PostFormValue("csrf")
	}
	return s.csrf.Valid(httpx.Binding(r), got)
}

// setFlash stores a signed one-shot message shown on the next page.
func (s *server) setFlash(w http.ResponseWriter, r *http.Request, msg string) {
	b := base64.RawURLEncoding.EncodeToString([]byte(msg))
	s.setCookie(w, r, cookieFlash, b+"."+s.mac("flash"+b), 300)
}

func (s *server) takeFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie(cookieFlash)
	if err != nil {
		return ""
	}
	s.setCookie(w, r, cookieFlash, "", -1)
	b, sig, ok := strings.Cut(c.Value, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.mac("flash"+b))) {
		return ""
	}
	m, err := base64.RawURLEncoding.DecodeString(b)
	if err != nil {
		return ""
	}
	return string(m)
}

// rc is the per-request context of an authenticated handler.
type rc struct {
	s     *server
	ctx   context.Context
	actor domain.Actor
	token string
	// Set by net():
	Info *service.NetworkInfo
	Net  *domain.Network
	Me   *domain.Member
}

func (c *rc) netID() string { return c.Net.ID }

func (c *rc) isAdmin() bool {
	return c.Me != nil && c.Me.Status == domain.MemberActive && c.Me.HasRole(domain.RoleAdmin)
}
func (c *rc) isReviewer() bool {
	return c.Me != nil && c.Me.Status == domain.MemberActive && c.Me.HasRole(domain.RoleReviewer)
}
func (c *rc) isActive() bool { return c.Me != nil && c.Me.Status == domain.MemberActive }

func (s *server) authenticate(r *http.Request) (domain.Actor, string, bool) {
	tok := httpx.SessionToken(r)
	if tok == "" {
		return domain.Actor{}, "", false
	}
	a, err := s.svc.Authenticate(r.Context(), tok)
	if err != nil {
		return domain.Actor{}, "", false
	}
	a.RequestID = reqID(r)
	return a, tok, true
}

// safeNext returns next when it is a plain local path, else "" (SEC-024).
func safeNext(n string) string { return httpx.SafeNext(n) }

func (s *server) toLogin(w http.ResponseWriter, r *http.Request) {
	next := ""
	if r.Method == http.MethodGet {
		next = "?next=" + url.QueryEscape(r.URL.RequestURI())
	}
	http.Redirect(w, r, "/login"+next, http.StatusSeeOther)
}

// auth wraps a handler that needs a signed-in person.
func (s *server) auth(h func(http.ResponseWriter, *http.Request, *rc)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, tok, ok := s.authenticate(r)
		if !ok {
			s.toLogin(w, r)
			return
		}
		h(w, r, &rc{s: s, ctx: r.Context(), actor: a, token: tok})
	}
}

// netH wraps a handler scoped to /n/{slug}: it resolves the network and the
// caller's membership. A non-member gets the same page as an unknown slug.
func (s *server) netH(h func(http.ResponseWriter, *http.Request, *rc)) http.HandlerFunc {
	return s.auth(func(w http.ResponseWriter, r *http.Request, c *rc) {
		info, err := s.svc.NetworkBySlug(r.Context(), r.PathValue("slug"))
		if err != nil {
			s.fail(w, r, domain.NotFound("network"))
			return
		}
		mems, err := s.svc.MyMemberships(r.Context(), c.actor)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		for _, m := range mems {
			if m.NetworkID == info.ID {
				c.Me = m
			}
		}
		if c.Me == nil {
			s.fail(w, r, domain.NotFound("network"))
			return
		}
		if c.Net, err = s.svc.GetNetwork(r.Context(), c.actor, info.ID); err != nil {
			s.fail(w, r, err)
			return
		}
		c.Info = info
		if c.Me.Status == domain.MemberPending {
			s.render(w, r, c, "pending", 200, &page{Title: "Waiting for approval"})
			return
		}
		h(w, r, c)
	})
}

// adminH additionally requires the administrator role (else the 403 page).
func (s *server) adminH(h func(http.ResponseWriter, *http.Request, *rc)) http.HandlerFunc {
	return s.netH(func(w http.ResponseWriter, r *http.Request, c *rc) {
		if !c.isAdmin() {
			s.fail(w, r, domain.ErrForbidden)
			return
		}
		h(w, r, c)
	})
}

// idemKey returns the form's idempotency key (minted when the form was shown).
func idemKey(r *http.Request) string {
	if k := r.PostFormValue("idem"); k != "" && len(k) <= 64 {
		return k
	}
	return randHex(16)
}
