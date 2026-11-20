package web

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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

func networkSuspensionCategory(category string) string {
	switch category {
	case domain.SuspendLegalOrder:
		return messages.Text("admin.network.category_legal_order")
	case domain.SuspendSecurityIncident:
		return messages.Text("admin.network.category_security_incident")
	case domain.SuspendPlatformAbuse:
		return messages.Text("admin.network.category_platform_abuse")
	case domain.SuspendHostingAgreement:
		return messages.Text("admin.network.category_hosting_agreement")
	default:
		return category
	}
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
		who := messages.Text("admin.network.administrator")
		if sp.Kind == domain.SuspendedByPlatform {
			who = messages.Text("admin.network.platform_operator")
		}
		d.Suspensions = append(d.Suspensions, suspensionRow{Who: who, Category: networkSuspensionCategory(sp.Category), Reason: sp.Reason, At: c.when(sp.At)})
	}
	p.Title, p.D = messages.Text("admin.network.title"), d
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
	s.networkAction(w, r, c, messages.Text("admin.network.suspended"),
		func(reason string) error { return s.svc.SuspendNetwork(r.Context(), c.actor, c.netID(), reason) })
}

func (s *server) networkResume(w http.ResponseWriter, r *http.Request, c *rc) {
	s.networkAction(w, r, c, messages.Text("admin.network.resumed"),
		func(reason string) error { return s.svc.ResumeNetwork(r.Context(), c.actor, c.netID(), reason) })
}

func (s *server) networkClose(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	if strings.TrimSpace(v["confirm"]) != c.Net.Slug {
		p := &page{V: v}
		p.Summary = messages.Text("admin.network.close_invalid")
		p.Errors = []fieldErr{{"confirm", messages.Text("admin.network.confirm_invalid", c.Net.Slug)}}
		s.renderNetworkPage(w, r, c, 422, p)
		return
	}
	days, _ := strconv.Atoi(strings.TrimSpace(v["notice_days"]))
	s.networkAction(w, r, c, messages.Text("admin.network.closure_announced"),
		func(reason string) error {
			_, err := s.svc.ScheduleClosure(r.Context(), c.actor, c.netID(), service.ClosureInput{Reason: reason, NoticeDays: days})
			return err
		})
}

func (s *server) networkCloseCancel(w http.ResponseWriter, r *http.Request, c *rc) {
	s.networkAction(w, r, c, messages.Text("admin.network.closure_cancelled"),
		func(reason string) error { return s.svc.CancelClosure(r.Context(), c.actor, c.netID(), reason) })
}
