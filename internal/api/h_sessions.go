package api

import "github.com/djshellshoxxx/circula/internal/service"

type sessionDTO struct {
	ID         string `json:"id"`
	Current    bool   `json:"current"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
}

func toSession(s service.SessionInfo) sessionDTO {
	return sessionDTO{ID: s.ID, Current: s.Current, CreatedAt: ts(s.CreatedAt), LastSeenAt: ts(s.LastSeenAt), ExpiresAt: ts(s.ExpiresAt)}
}

// Session selectors are not UUID resource IDs or credentials. The service
// alone maps them to a session belonging to the authenticated person.
func (s *server) revokeOwnSession(c *reqCtx) error {
	if err := s.svc.RevokeOwnSession(c.r.Context(), c.actor, c.r.PathValue("session")); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}

func (s *server) listOwnSessions(c *reqCtx) error {
	// The persisted public selector has a different ordering from the former
	// derived selector. Old signed positions must not resume in that new order.
	p, err := c.page("session-public-id-v2")
	if err != nil {
		return err
	}
	page, err := s.svc.ListOwnSessions(c.r.Context(), c.actor, service.Page{AfterID: p.after, Limit: p.limit})
	if err != nil {
		return err
	}
	var next *string
	if page.Next != "" {
		// A bounded scan may have no active rows and still need continuation.
		token := s.sealCursor(page.Next, p.filter)
		next = &token
	}
	out := make([]sessionDTO, 0, len(page.Rows))
	for _, row := range page.Rows {
		out = append(out, toSession(row))
	}
	return listOK(c, out, next)
}
