package web

import (
	"net/http"
	"net/url"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /account/sessions", s.auth(s.sessionsPage))
		s.mux.HandleFunc("GET /account/sessions/revoke-others", s.auth(s.sessionsRevokeOthersPage))
		s.mux.HandleFunc("POST /account/sessions/revoke-others", s.auth(s.sessionsRevokeOthersPost))
		s.mux.HandleFunc("GET /account/sessions/{session}/revoke", s.auth(s.sessionRevokePage))
		s.mux.HandleFunc("POST /account/sessions/{session}/revoke", s.auth(s.sessionRevokePost))
	})
}

func (s *server) sessionsPage(w http.ResponseWriter, r *http.Request, c *rc) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err == nil {
		for key, values := range query {
			if (key != "after" && key != "before") || len(values) != 1 {
				err = domain.Invalid("cursor", "invalid session page query")
				break
			}
		}
	}
	if err != nil {
		s.fail(w, r, domain.Invalid("cursor", "invalid session page query"))
		return
	}
	rows, err := s.svc.ListOwnSessions(r.Context(), c.actor, service.Page{
		AfterID: query.Get("after"), BeforeID: query.Get("before"),
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "sessions", http.StatusOK, &page{Title: messages.Text("sessions.title"), D: rows})
}

func (s *server) sessionsRevokeOthersPage(w http.ResponseWriter, r *http.Request, _ *rc) {
	s.render(w, r, nil, "sessions_revoke_others", http.StatusOK,
		&page{Title: messages.Text("sessions.bulk_title")})
}

func (s *server) sessionsRevokeOthersPost(w http.ResponseWriter, r *http.Request, c *rc) {
	if r.PostFormValue("confirm") != "yes" {
		s.render(w, r, nil, "sessions_revoke_others", http.StatusUnprocessableEntity,
			&page{Title: messages.Text("sessions.bulk_title"), Summary: messages.Text("sessions.bulk_required"),
				Errors: []fieldErr{{"confirm", messages.Text("sessions.bulk_confirm")}}})
		return
	}
	if _, err := s.svc.RevokeOtherOwnSessions(r.Context(), c.actor); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, messages.Text("sessions.bulk_ended"))
	http.Redirect(w, r, "/account/sessions", http.StatusSeeOther)
}

func (s *server) ownSessionTarget(r *http.Request, c *rc) (*service.SessionInfo, error) {
	return s.svc.GetOwnSession(r.Context(), c.actor, r.PathValue("session"))
}

func (s *server) sessionRevokePage(w http.ResponseWriter, r *http.Request, c *rc) {
	target, err := s.ownSessionTarget(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "session_revoke", http.StatusOK, &page{Title: messages.Text("sessions.confirm_title"), D: target})
}

func (s *server) sessionRevokePost(w http.ResponseWriter, r *http.Request, c *rc) {
	target, err := s.ownSessionTarget(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if r.PostFormValue("confirm") != "yes" {
		s.render(w, r, c, "session_revoke", http.StatusUnprocessableEntity,
			&page{Title: messages.Text("sessions.confirm_title"), D: target,
				Summary: messages.Text("sessions.confirm_required"),
				Errors:  []fieldErr{{"confirm", messages.Text("sessions.confirm_label")}}})
		return
	}
	if err := s.svc.RevokeOwnSession(r.Context(), c.actor, target.ID); err != nil {
		s.fail(w, r, err)
		return
	}
	if target.Current {
		s.clearSession(w, r)
		s.rotateCSRF(w, r)
		s.setFlash(w, r, messages.Text("sessions.current_ended"))
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.setFlash(w, r, messages.Text("sessions.other_ended"))
	http.Redirect(w, r, "/account/sessions", http.StatusSeeOther)
}
