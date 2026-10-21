package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (s *server) accountRoutes() {
	s.mux.HandleFunc("GET /account/password", s.auth(s.passwordForm))
	s.mux.HandleFunc("POST /account/password", s.auth(s.passwordPost))
	s.mux.HandleFunc("POST /session/extend", s.auth(s.sessionExtend))
}

func (s *server) passwordForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.render(w, r, nil, "password", 200, &page{Title: "Change your password"})
}

// passwordPost changes the password (AUTH-03, SEC-003). The service revokes
// every session of the person, including this one, so the person signs in again.
func (s *server) passwordPost(w http.ResponseWriter, r *http.Request, c *rc) {
	p := &page{Title: "Change your password"}
	show := func(st int, sum string, fe []fieldErr) {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "password", st, p)
	}
	key := "pw|" + c.actor.PersonID
	if s.pwFail.blocked(key) {
		show(429, "Too many attempts. Wait a few minutes and try again. Nothing changed.", nil)
		return
	}
	oldPW, newPW := r.PostFormValue("current_password"), r.PostFormValue("password")
	if newPW != r.PostFormValue("password2") {
		show(422, "Not saved. Fix the problems below. Nothing changed.",
			[]fieldErr{{"password2", "Confirm new password: the two passwords do not match. Type the same password in both fields."}})
		return
	}
	err := s.svc.ChangePassword(r.Context(), c.actor, oldPW, newPW)
	switch {
	case err == nil:
	case domain.CodeOf(err) == domain.CodeUnauthenticated:
		s.pwFail.fail(key)
		show(422, "Not saved. Fix the problems below. Nothing changed.",
			[]fieldErr{{"current_password", "Current password: not correct. Check it and try again."}})
		return
	default:
		cc := &rc{s: s, ctx: r.Context(), actor: c.actor, Net: &domain.Network{}}
		if sum, fe, ok := cc.refuse(err); ok {
			show(422, sum, fe)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.pwFail.reset(key)
	s.clearSession(w, r)
	s.rotateCSRF(w, r)
	s.setFlash(w, r, "Password changed. You were signed out everywhere. Sign in with the new password.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// expiry returns the Unix time at which the current session is expected to
// end and whether staying active can extend it (UX-030). It is advisory: the
// service decides. The absolute limit comes from the signed start cookie.
func (s *server) expiry(r *http.Request) (at int64, extendable, ok bool) {
	if s.opts.SessionIdle <= 0 || s.opts.Edge.SessionToken(r) == "" {
		return 0, false, false
	}
	now := s.opts.Clock.Now()
	end, extendable := now.Add(s.opts.SessionIdle), true
	if c, err := r.Cookie(cookieStart); err == nil && s.opts.SessionMax > 0 {
		v, sig, found := strings.Cut(c.Value, ".")
		if n, perr := strconv.ParseInt(v, 10, 64); found && perr == nil && sig == s.mac("start"+v) {
			if abs := time.Unix(n, 0).Add(s.opts.SessionMax); abs.Before(end) {
				end, extendable = abs, false
			}
		}
	}
	return end.Unix(), extendable, true
}

// sessionExtend is the target of the "Stay signed in" button. Reaching it
// already refreshed the session's idle timer (Authenticate); it reports the
// new expected end.
func (s *server) sessionExtend(w http.ResponseWriter, r *http.Request, c *rc) {
	at, ext, ok := s.expiry(r)
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		http.Error(w, `{"error":"unavailable"}`, http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"expires": at, "extendable": ext})
}
