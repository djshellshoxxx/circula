package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

type disputeReportRow struct {
	ID, Category, Status string
}

type disputeReportsData struct {
	DisputeID string
	Rows      []disputeReportRow
	Next      string
}

func (s *server) disputeReports(w http.ResponseWriter, r *http.Request, c *rc) {
	id := r.PathValue("id")
	reports, err := s.svc.ListDisputeReports(r.Context(), c.actor, c.netID(), id,
		service.Page{AfterID: r.URL.Query().Get("after"), Limit: perPage + 1})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &disputeReportsData{DisputeID: id}
	if len(reports) > perPage {
		reports = reports[:perPage]
		d.Next = reports[perPage-1].ID
	}
	for _, report := range reports {
		d.Rows = append(d.Rows, disputeReportRow{ID: report.ID, Category: categoryOf(report.Category),
			Status: messages.Text("report.status." + report.Status)})
	}
	s.render(w, r, c, "dispute_reports", 200, &page{Title: messages.Text("report.review_title"), D: d})
}
