package web

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// Network status page: suspension (GOV-023) and closure (FR-068, GOV-025).

func (s *server) adminNetworkRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/admin/network", s.adminH(s.networkPage))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/suspend", s.adminH(s.networkSuspend))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/resume", s.adminH(s.networkResume))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/close", s.adminH(s.networkClose))
	s.mux.HandleFunc("POST /n/{slug}/admin/network/close/cancel", s.adminH(s.networkCloseCancel))
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
	OpenProposals   int
	Closing         bool // a closure is announced and has not completed
	ClosingAt       template.HTML
	ClosureReason   string
	CanCancel       bool
	ReadOnlyUntil   template.HTML
	FinalSHA256     string
	MinNoticeDays   int
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
		Blockers: pv.Blockers, NonZeroBalances: pv.NonZeroBalances, Members: pv.Members, OpenProposals: pv.OpenProposals,
		MinNoticeDays:  domain.MinClosureNoticeDays,
		AdminSuspended: n.SuspensionOf(domain.SuspendedByAdmin) != nil, PlatformHolds: n.SuspensionOf(domain.SuspendedByPlatform) != nil}
	if cl := n.Closure; cl != nil {
		d.ClosureReason, d.FinalSHA256 = cl.Reason, cl.FinalSHA256
		d.ClosingAt = c.when(cl.ClosingAt)
		d.Closing = !cl.Done()
		d.CanCancel = n.Status == domain.NetworkActive && !cl.Done()
		if cl.Done() {
			d.ReadOnlyUntil = c.when(cl.ReadOnlyUntil())
		}
	}
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
	days, _ := strconv.Atoi(strings.TrimSpace(v["notice_days"]))
	s.networkAction(w, r, c, "The closure date is announced. Members were told. Trading continues until that date.",
		func(reason string) error {
			_, err := s.svc.ScheduleClosure(r.Context(), c.actor, c.netID(), service.ClosureInput{Reason: reason, NoticeDays: days})
			return err
		})
}

func (s *server) networkCloseCancel(w http.ResponseWriter, r *http.Request, c *rc) {
	s.networkAction(w, r, c, "The closure is cancelled. Members were told.",
		func(reason string) error { return s.svc.CancelClosure(r.Context(), c.actor, c.netID(), reason) })
}
