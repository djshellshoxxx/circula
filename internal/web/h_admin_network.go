package web

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Network status page: suspension (GOV-023) and closure (FR-068, GOV-025).

func (s *server) adminNetworkRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/admin/network", s.adminH(s.networkPage))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/suspend", s.adminH(s.networkSuspend))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/resume", s.adminH(s.networkResume))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/close", s.adminH(s.networkClose))
}

type suspensionRow struct {
	Who, Category, Reason string
	At                    template.HTML
}

type networkData struct {
	Status          string
	Closed          bool
	Suspensions     []suspensionRow
	AdminSuspended  bool
	PlatformHolds   bool
	Blockers        []string
	NonZeroBalances int
	Members         int
}

func (s *server) renderNetworkPage(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	pv, err := s.svc.PreviewClosure(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n, err := s.svc.GetNetwork(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &networkData{Status: string(n.Status), Closed: n.Status == domain.NetworkClosed,
		Blockers: pv.Blockers, NonZeroBalances: pv.NonZeroBalances, Members: pv.Members,
		AdminSuspended: n.SuspensionOf(domain.SuspendedByAdmin) != nil, PlatformHolds: n.SuspensionOf(domain.SuspendedByPlatform) != nil}
	for _, sp := range n.Suspensions {
		who := "An administrator"
		if sp.Kind == domain.SuspendedByPlatform {
			who = "The platform operator"
		}
		d.Suspensions = append(d.Suspensions, suspensionRow{Who: who, Category: sp.Category, Reason: sp.Reason, At: c.when(sp.At)})
	}
	p.Title, p.D = "Suspension and closure", d
	s.render(w, r, c, "admin_network", status, p)
}

func (s *server) networkPage(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderNetworkPage(w, r, c, 200, &page{})
}

func (s *server) networkAction(w http.ResponseWriter, r *http.Request, c *rc, done string, call func(reason string) error) {
	v := formValues(r)
	p := &page{V: v}
	if err := call(v["reason"]); err != nil {
		if sum, fe, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			s.renderNetworkPage(w, r, c, 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, done)
	http.Redirect(w, r, c.base()+"/admin/network", http.StatusSeeOther)
}

func (s *server) networkSuspend(w http.ResponseWriter, r *http.Request, c *rc) {
	s.networkAction(w, r, c, "Trading is paused. Members were told. Balances, history and exports stay available.",
		func(reason string) error { return s.svc.SuspendNetwork(r.Context(), c.actor, c.netID(), reason) })
}

func (s *server) networkResume(w http.ResponseWriter, r *http.Request, c *rc) {
	s.networkAction(w, r, c, "Your suspension is lifted.",
		func(reason string) error { return s.svc.ResumeNetwork(r.Context(), c.actor, c.netID(), reason) })
}

func (s *server) networkClose(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	if strings.TrimSpace(v["confirm"]) != c.Net.Slug {
		p := &page{V: v}
		p.Summary = "Not closed. Type the network address name exactly to confirm. Nothing changed."
		p.Errors = []fieldErr{{"confirm", "Confirmation: type " + c.Net.Slug + " exactly."}}
		s.renderNetworkPage(w, r, c, 422, p)
		return
	}
	s.networkAction(w, r, c, "The network is closed. History stays readable and exports remain available.",
		func(reason string) error { return s.svc.CloseNetwork(r.Context(), c.actor, c.netID(), reason) })
}
