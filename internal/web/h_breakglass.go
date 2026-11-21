package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/admin/break-glass", s.adminH(s.breakGlassPage))
	})
}

type breakGlassRow struct {
	ID, Operator, Reason, Status string
	Opened, Expires, Closed      any
	ClosedReason                 string
}

func breakGlassStatusText(status string) string {
	switch status {
	case "open":
		return messages.Text("admin.break_glass.status_open")
	case "closed":
		return messages.Text("admin.break_glass.status_closed")
	case "expired":
		return messages.Text("admin.break_glass.status_expired")
	default:
		return status
	}
}

// breakGlassPage shows the network's operator break-glass history, read only
// (SEC-013 b). There is deliberately no form: administrators cannot open,
// extend or close sessions; only the operator CLI can.
func (s *server) breakGlassPage(w http.ResponseWriter, r *http.Request, c *rc) {
	after := r.URL.Query().Get("after")
	bs, err := s.svc.ListBreakGlass(r.Context(), c.actor, c.netID(), service.Page{AfterID: after, Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	now := s.opts.Clock.Now()
	rows := make([]breakGlassRow, 0, len(bs))
	for _, b := range bs {
		row := breakGlassRow{ID: b.ID, Operator: b.OperatorLabel, Reason: b.Reason, Status: breakGlassStatusText(b.Status(now)),
			Opened: fmtTime(b.OpenedAt), Expires: fmtTime(b.ExpiresAt), ClosedReason: b.ClosedReason}
		if b.ClosedAt != nil {
			row.Closed = fmtTime(*b.ClosedAt)
		}
		rows = append(rows, row)
	}
	next := ""
	if len(bs) == 50 {
		next = bs[49].ID
	}
	s.render(w, r, c, "admin_break_glass", http.StatusOK, &page{Title: messages.Text("admin.break_glass.title"), D: map[string]any{"Rows": rows, "Next": next}})
}
