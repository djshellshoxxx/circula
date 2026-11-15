package api

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type termsDTO struct {
	Version     int    `json:"version"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

type termsStateDTO struct {
	Current         *termsDTO `json:"current"`
	AcceptedVersion int       `json:"accepted_version"`
	NeedsAcceptance bool      `json:"needs_acceptance"`
}

func toTerms(t *domain.Terms) *termsDTO {
	if t == nil {
		return nil
	}
	return &termsDTO{Version: t.Version, Body: t.Body, PublishedAt: ts(t.PublishedAt)}
}

func (s *server) getTerms(c *reqCtx) error { // GOV-009
	v, err := s.svc.CurrentTerms(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	return ok(c, 200, termsStateDTO{Current: toTerms(v.Current), AcceptedVersion: v.Accepted, NeedsAcceptance: v.NeedsAcceptance()})
}

func (s *server) acceptTerms(c *reqCtx) error { // GOV-009
	var in struct {
		Version int `json:"version"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	if err := s.svc.AcceptTerms(c.r.Context(), c.actor, c.netID, in.Version); err != nil {
		return err
	}
	return s.getTerms(c)
}

func (s *server) publishTerms(c *reqCtx) error { // GOV-009
	var in struct {
		Body string `json:"body"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	t, err := s.svc.PublishTerms(c.r.Context(), c.actor, c.netID, in.Body)
	if err != nil {
		return err
	}
	return ok(c, 201, toTerms(t))
}

type policyLogDTO struct {
	Version      int            `json:"version"`
	Changes      map[string]any `json:"changes"`
	Previous     map[string]any `json:"previous,omitempty"`
	Reason       string         `json:"reason"`
	AnnouncedAt  string         `json:"announced_at"`
	EffectiveAt  string         `json:"effective_at"`
	NoticeDays   int            `json:"notice_days"`
	NoticeReason string         `json:"notice_reason,omitempty"`
	Pending      bool           `json:"pending"`
	Withdrawn    bool           `json:"withdrawn"`
}

func toPolicyLog(e service.PolicyLogEntry) policyLogDTO {
	return policyLogDTO{Version: e.Version, Changes: e.Changes, Previous: e.Previous, Reason: e.Reason,
		AnnouncedAt: ts(e.AnnouncedAt), EffectiveAt: ts(e.EffectiveAt), NoticeDays: e.NoticeDays,
		NoticeReason: e.NoticeReason, Pending: e.Pending, Withdrawn: e.Withdrawn}
}

func (s *server) policyLog(c *reqCtx) error { // GOV-002, GOV-003
	rows, err := s.svc.PolicyLog(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	out := make([]policyLogDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, toPolicyLog(e))
	}
	return listOK(c, out, nil)
}
