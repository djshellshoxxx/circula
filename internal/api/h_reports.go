package api

import (
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// reportDTO is a report. Reporter fields are filled only for administrators
// (FR-024: the reporter is never shown to the reported member).
type reportDTO struct {
	ID               string  `json:"id"`
	TargetType       string  `json:"target_type"`
	TargetID         string  `json:"target_id"`
	Category         string  `json:"category"`
	Description      string  `json:"description"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	ReporterMemberID string  `json:"reporter_member_id"`
	ReporterName     *string `json:"reporter_display_name"`
	TargetLabel      *string `json:"target_label"`
	HandledBy        *string `json:"handled_by"`
	HandledAt        *string `json:"handled_at"`
	HandlingNote     string  `json:"handling_note"`
}

func toReport(r *store.Report) reportDTO {
	return reportDTO{ID: r.ID, TargetType: r.TargetType, TargetID: r.TargetID, Category: r.Category, Description: r.Note,
		Status: r.Status, CreatedAt: ts(r.CreatedAt), ReporterMemberID: r.ReporterID, HandledBy: strPtr(r.HandledBy),
		HandledAt: tsPtr(r.HandledAt), HandlingNote: r.HandlingNote}
}

func (s *server) fileReport(c *reqCtx) error { // FR-024
	var in struct {
		TargetType  string `json:"target_type"`
		TargetID    string `json:"target_id"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	r, err := s.svc.CreateReport(c.r.Context(), c.actor, c.netID, service.ReportInput{TargetType: in.TargetType,
		TargetID: in.TargetID, Category: in.Category, Note: in.Description, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	c.w.Header().Set("Location", v1+"/reports/"+r.ID)
	return ok(c, 201, toReport(r))
}

func (s *server) reportQueue(c *reqCtx) error { // FR-024 administrator queue
	if err := c.only("limit", "cursor", "status"); err != nil {
		return err
	}
	status := c.query.Get("status")
	if status != "" && status != "open" && status != "handled" {
		return invalidQuery("status must be open or handled")
	}
	p, err := c.page(status)
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.ReportView, error) {
		return s.svc.ListReports(c.r.Context(), c.actor, c.netID, status, pg)
	}, func(v service.ReportView) string { return v.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]reportDTO, 0, len(rows))
	for _, v := range rows {
		d := toReport(v.Report)
		d.ReporterName, d.TargetLabel = strPtr(v.ReporterName), strPtr(v.TargetLabel)
		out = append(out, d)
	}
	return listOK(c, out, next)
}

func (s *server) closeReport(c *reqCtx) error { // FR-024 mark handled
	var in struct {
		Note string `json:"note"`
	}
	if err := c.decodeOptional(&in); err != nil {
		return err
	}
	r, err := s.svc.HandleReport(c.r.Context(), c.actor, c.netID, service.HandleReportInput{ReportID: c.r.PathValue("id"),
		Note: in.Note, IdempotencyKey: c.idemKey})
	if err != nil {
		return err
	}
	return ok(c, 200, toReport(r))
}
