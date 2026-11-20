package api

import "github.com/djshellshoxxx/circula/internal/service"

// Deliberately separate from reportDTO: no reporter, note, handling identity
// or timestamp can reach the reviewer projection through shared conversion.
type disputeReportDTO struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

func (s *server) disputeReports(c *reqCtx) error {
	if err := c.only("limit", "cursor"); err != nil {
		return err
	}
	p, err := c.page("") // the path binds this dispute, plus network and principal
	if err != nil {
		return err
	}
	rows, next, err := collect(c, p, func(pg service.Page) ([]service.DisputeReport, error) {
		return s.svc.ListDisputeReports(c.r.Context(), c.actor, c.netID, c.r.PathValue("id"), pg)
	}, func(r service.DisputeReport) string { return r.ID }, nil)
	if err != nil {
		return err
	}
	out := make([]disputeReportDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, disputeReportDTO{ID: r.ID, Category: r.Category, Status: r.Status})
	}
	return listOK(c, out, next)
}
