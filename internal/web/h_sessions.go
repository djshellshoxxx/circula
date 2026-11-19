package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /account/sessions", s.auth(s.sessionsPage))
		s.mux.HandleFunc("GET /account/sessions/{session}/revoke", s.auth(s.sessionRevokePage))
		s.mux.HandleFunc("POST /account/sessions/{session}/revoke", s.auth(s.sessionRevokePost))
	})
}

func (s *server) sessionsPage(w http.ResponseWriter, r *http.Request, c *rc) {
	rows, err := s.svc.ListOwnSessions(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "sessions", http.StatusOK, &page{Title: messages.Text("sessions.title"), D: rows})
}

func (s *server) ownSessionTarget(r *http.Request, c *rc) (*service.SessionInfo, error) {
	rows, err := s.svc.ListOwnSessions(r.Context(), c.actor)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.ID == r.PathValue("session") {
			return &row, nil
		}
	}
	return nil, domain.NotFound("session")
}

func (s *server) sessionRevokePage(w http.ResponseWriter, r *http.Request, c *rc) {
	target, err := s.ownSessionTarget(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "session_revoke", http.StatusOK, &page{Title: messages.Text("sessions.confirm_title"), D: target})
}

func (s *server) sessionRevokePost(w http.ResponseWriter, r *http.Request, c *rc) {
	target, err := s.ownSessionTarget(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if r.PostFormValue("confirm") != "yes" {
		s.render(w, r, nil, "session_revoke", http.StatusUnprocessableEntity,
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
