package api

import (
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// notificationDTO is one inbox entry (FR-038). link is a path relative to the
// network, for example "/trades/<id>" or "/disputes/<id>".
type notificationDTO struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	Link      string  `json:"link"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"created_at"`
	ReadAt    *string `json:"read_at"`
}

func toNotification(n *store.Notification) notificationDTO {
	return notificationDTO{ID: n.ID, Kind: n.Kind, Title: n.Title, Body: n.Body, Link: n.Link, Read: n.ReadAt != nil,
		CreatedAt: ts(n.CreatedAt), ReadAt: tsPtr(n.ReadAt)}
}

func (s *server) listNotifications(c *reqCtx) error { // FR-038
	if err := c.only("limit", "cursor", "unread"); err != nil {
		return err
	}
	unread := false
	switch c.query.Get("unread") {
	case "", "false":
	case "true":
		unread = true
	default:
		return invalidQuery("unread must be true or false")
	}
	p, err := c.page(c.query.Get("unread"))
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]*store.Notification, error) {
		return s.svc.ListNotifications(c.r.Context(), c.actor, c.netID, service.NotificationQuery{UnreadOnly: unread, Page: pg})
	}, func(n *store.Notification) string { return n.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]notificationDTO, 0, len(rows))
	for _, n := range rows {
		out = append(out, toNotification(n))
	}
	return listOK(c, out, next)
}

func (s *server) unreadCount(c *reqCtx) error { // FR-038
	n, err := s.svc.CountUnreadNotifications(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	return ok(c, 200, struct {
		Unread int `json:"unread"`
	}{n})
}

func (s *server) markNotificationsRead(c *reqCtx) error { // FR-038
	var in struct {
		IDs []string `json:"ids"`
		All bool     `json:"all"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	if in.All == (len(in.IDs) > 0) { // exactly one of ids / all
		return validation("ids", "invalid", "send either a non-empty ids list or all=true")
	}
	if in.All {
		in.IDs = nil
	}
	n, err := s.svc.MarkNotificationsRead(c.r.Context(), c.actor, c.netID, in.IDs)
	if err != nil {
		return err
	}
	return ok(c, 200, struct {
		Marked int64 `json:"marked"`
	}{n})
}
