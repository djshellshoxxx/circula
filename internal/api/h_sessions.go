package api

import (
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

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
	p, err := c.page("")
	if err != nil {
		return err
	}
	var afterTime time.Time
	var afterID string
	if p.after != "" {
		stamp, id, ok := strings.Cut(p.after, ":")
		millis, err := strconv.ParseInt(stamp, 10, 64)
		if !ok || err != nil || len(id) != 64 {
			return &apiError{status: 400, code: "invalid-cursor", detail: "cursor is not valid for this request"}
		}
		afterTime, afterID = time.UnixMilli(millis), id
	}
	rows, err := s.svc.ListOwnSessions(c.r.Context(), c.actor)
	if err != nil {
		return err
	}
	var page []service.SessionInfo
	for _, row := range rows {
		if afterID != "" && (row.CreatedAt.After(afterTime) || row.CreatedAt.Equal(afterTime) && row.ID >= afterID) {
			continue
		}
		page = append(page, row)
		if len(page) > p.limit {
			break
		}
	}
	var next *string
	if len(page) > p.limit {
		page = page[:p.limit]
		last := page[len(page)-1]
		token := s.sealCursor(strconv.FormatInt(last.CreatedAt.UnixMilli(), 10)+":"+last.ID, p.filter)
		next = &token
	}
	out := make([]sessionDTO, 0, len(page))
	for _, row := range page {
		out = append(out, toSession(row))
	}
	return listOK(c, out, next)
}
