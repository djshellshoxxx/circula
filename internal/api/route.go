package api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/domain"
)

type idemMode int

const (
	idemNone idemMode = iota
	idemOptional
	idemRequired
)

// reqCtx carries the authenticated call through a handler.
type reqCtx struct {
	s           *server
	w           http.ResponseWriter
	r           *http.Request
	rid         string
	actor       domain.Actor
	cookie      bool
	session     string
	netID       string
	member      *domain.Member
	memberships []*domain.Member
	idemKey     string
	raw         []byte
	rawRead     bool
	query       url.Values
}

type handlerFunc func(c *reqCtx) error

type route struct {
	pattern    string
	public     bool // no authentication
	noNet      bool // no network resolution
	cookieOnly bool // session authentication only (API-011)
	idem       idemMode
	h          handlerFunc
}

func (s *server) add(rt route) { s.mux.HandleFunc(rt.pattern, s.wrap(rt)) }

func (s *server) wrap(rt route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &reqCtx{s: s, w: w, r: r, rid: w.Header().Get("X-Request-Id"), query: r.URL.Query()}
		if err := s.serve(c, rt); err != nil {
			ae, expected := fromErr(err)
			if !expected {
				s.log.Error("api internal error", "request_id", c.rid, "path", r.URL.Path, "err", err)
			}
			writeProblem(w, r, c.rid, ae)
		}
	}
}

func (s *server) serve(c *reqCtx, rt route) error {
	r := c.r
	if !rt.public {
		if err := s.authenticate(r, c); err != nil {
			return err
		}
		if c.cookie {
			c.w.Header().Set("X-CSRF-Token", CSRFToken(c.session))
		}
		if s.opt.PrincipalPerMinute > 0 {
			if ok, retry := s.rl.allow("p:"+c.actor.PersonID, s.opt.PrincipalPerMinute); !ok {
				return rateLimited(retry)
			}
		}
		if err := checkCSRF(r, c); err != nil {
			return err
		}
		if rt.cookieOnly && !c.cookie {
			return domain.Forbidden("session authentication is required for this operation")
		}
		if !rt.noNet {
			if err := s.resolveNetwork(r, c); err != nil {
				return err
			}
		}
	}
	if rt.idem == idemNone {
		return rt.h(c)
	}
	return s.runIdempotent(c, rt)
}

// qInt reads an integer query parameter in [lo, hi].
func (c *reqCtx) qInt(name string, def, lo, hi int) (int, error) {
	v := c.query.Get(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < lo || n > hi {
		return 0, invalidQuery(name + " must be an integer from " + strconv.Itoa(lo) + " to " + strconv.Itoa(hi))
	}
	return n, nil
}

// only rejects unknown query parameters (API: invalid-query).
func (c *reqCtx) only(allowed ...string) error {
outer:
	for k := range c.query {
		for _, a := range allowed {
			if k == a {
				continue outer
			}
		}
		return invalidQuery("unsupported query parameter " + strconv.Quote(k))
	}
	return nil
}
