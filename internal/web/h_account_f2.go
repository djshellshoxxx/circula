package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) timeZoneRoutes() {
	s.mux.HandleFunc("GET /account/time-zone", s.auth(s.timeZoneForm))
	s.mux.HandleFunc("POST /account/time-zone", s.auth(s.timeZonePost))
}

type timeZoneData struct {
	Current string
	Example any
}

func (s *server) timeZonePage(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	zone, err := s.svc.GetTimeZone(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p.Title = "Time zone"
	cur := zone
	if cur == "" {
		cur = "UTC"
	}
	p.D = &timeZoneData{Current: cur, Example: fmtTime(s.opts.Clock.Now().In(service.Location(zone)))}
	if p.V == nil {
		p.V = map[string]string{"time_zone": zone}
	}
	s.render(w, r, nil, "time_zone", status, p)
}

func (s *server) timeZoneForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.timeZonePage(w, r, c, 200, &page{})
}

// timeZonePost saves the display time zone. Pages show times in it; the API and the audit log
// stay in UTC.
func (s *server) timeZonePost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	zone := strings.TrimSpace(v["time_zone"])
	if err := s.svc.SetTimeZone(r.Context(), c.actor, zone); err != nil {
		if domain.CodeOf(err) == domain.CodeValidation {
			s.timeZonePage(w, r, c, 422, &page{V: v, Summary: "Not saved. Fix the problem below. Nothing changed.",
				Errors: []fieldErr{{"time_zone", "Time zone: enter a name such as Europe/Berlin or America/Mexico_City, or leave it empty for UTC."}}})
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Time zone saved.")
	http.Redirect(w, r, "/account/time-zone", http.StatusSeeOther)
}
