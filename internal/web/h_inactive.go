package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
)

// GOV-012: administrators see the members the network's inactivity rule flags.
// Nothing here changes a membership; the ordinary member actions stay the way
// to act.
func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/admin/inactive", s.adminH(s.adminInactive))
	})
}

func (s *server) adminInactive(w http.ResponseWriter, r *http.Request, c *rc) {
	ms, err := s.svc.ListInactiveMembers(r.Context(), c.actor, c.netID(), service.Page{Limit: 100})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var rows []memberRow
	for _, m := range ms {
		rows = append(rows, memberRow{Member: m, Actions: actionsFor(m)})
	}
	s.render(w, r, c, "admin_inactive", 200, &page{Title: "Inactive members",
		D: map[string]any{"Rows": rows, "Days": c.Net.Policy.InactivityDays}})
}
