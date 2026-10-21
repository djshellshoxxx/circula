package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// getListing reads one listing with its ETag (API-046, gap G-1). A matching If-None-Match gets
// 304 with no body. Active members see active listings of active owners and all of their own;
// anything else is 404.
func (s *server) getListing(c *reqCtx) error {
	if err := c.only(); err != nil {
		return err
	}
	l, err := s.svc.GetListing(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"))
	if err != nil {
		return err
	}
	etag := listingETag(l)
	c.w.Header().Set("ETag", etag)
	if inm := c.r.Header.Get("If-None-Match"); inm != "" && ifNoneMatch(inm, etag) {
		c.w.WriteHeader(304)
		return nil
	}
	return ok(c, 200, toListing(l))
}

// ifNoneMatch reports whether an If-None-Match header matches etag under weak comparison
// (RFC 9110 section 13.1.2): "*" matches anything and a W/ prefix is ignored.
func ifNoneMatch(header, etag string) bool {
	for _, p := range strings.Split(header, ",") {
		p = strings.TrimPrefix(strings.TrimSpace(p), "W/")
		if p == "*" || p == etag {
			return true
		}
	}
	return false
}

type announcementDTO struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	PublishedAt  string `json:"published_at"`
	EffectiveAt  string `json:"effective_at"`
	SupersedesID string `json:"supersedes_id,omitempty"`
}

func toAnnouncement(a *domain.Announcement) announcementDTO {
	return announcementDTO{ID: a.ID, Title: a.Title, Body: a.Body, PublishedAt: ts(a.PublishedAt), EffectiveAt: ts(a.EffectiveAt), SupersedesID: a.SupersedesID}
}

func (s *server) listAnnouncements(admin bool) handlerFunc { // API-064
	return func(c *reqCtx) error {
		if err := c.only("limit", "cursor"); err != nil {
			return err
		}
		p, err := c.page("")
		if err != nil {
			return err
		}
		rows, next, err := collect(c, p, func(pg service.Page) ([]*domain.Announcement, error) {
			if admin {
				return s.svc.AdminListAnnouncements(c.r.Context(), c.actor, c.netID, pg)
			}
			return s.svc.ListAnnouncements(c.r.Context(), c.actor, c.netID, pg)
		}, func(a *domain.Announcement) string { return a.ID }, nil)
		if err != nil {
			return err
		}
		out := make([]announcementDTO, 0, len(rows))
		for _, a := range rows {
			out = append(out, toAnnouncement(a))
		}
		return listOK(c, out, next)
	}
}

func (s *server) createAnnouncement(c *reqCtx) error { // API-064
	var in struct {
		Title        string  `json:"title"`
		Body         string  `json:"body"`
		EffectiveAt  *string `json:"effective_at"`
		SupersedesID string  `json:"supersedes_id"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	ai := service.AnnouncementInput{Title: in.Title, Body: in.Body, SupersedesID: in.SupersedesID, IdempotencyKey: c.idemKey}
	if in.EffectiveAt != nil {
		t, err := parseTime("effective_at", *in.EffectiveAt)
		if err != nil {
			return err
		}
		ai.EffectiveAt = t
	}
	a, err := s.svc.CreateAnnouncement(c.r.Context(), c.actor, c.netID, ai)
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/admin/announcements/"+a.ID)
	return ok(c, 201, toAnnouncement(a))
}

func (s *server) deleteAnnouncement(c *reqCtx) error { // API-064
	if err := s.svc.DeleteAnnouncement(c.r.Context(), c.actor, c.netID, c.r.PathValue("id")); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}

// patchMe updates the caller's preferences. Only time_zone exists: an IANA name, or "" to go
// back to UTC. The API itself always speaks UTC; the zone is for display in pages.
func (s *server) patchMe(c *reqCtx) error {
	var in struct {
		TimeZone json.RawMessage `json:"time_zone"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	if in.TimeZone == nil {
		return validation("time_zone", "required", "time_zone is required")
	}
	var zone string
	if isNull(in.TimeZone) || json.Unmarshal(in.TimeZone, &zone) != nil {
		return validation("time_zone", "invalid-type", "must be a string")
	}
	if err := s.svc.SetTimeZone(c.r.Context(), c.actor, zone); err != nil {
		return err
	}
	z, err := s.svc.GetTimeZone(c.r.Context(), c.actor)
	if err != nil {
		return err
	}
	return ok(c, 200, map[string]any{"person_id": c.actor.PersonID, "time_zone": zonePtr(z)})
}

func zonePtr(z string) *string {
	if z == "" {
		return nil
	}
	return &z
}

func parseTime(field, v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, validation(field, "invalid-format", "must be an RFC 3339 timestamp")
	}
	return t, nil
}
