package web

import (
	"html/template"
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// Administrator recovery (GOV-021): the members' page. Any member may read the
// requests and the recorded steps; an eligible member may ask to be appointed.
// Approvals happen only in the operator CLI; this page has no operator form.
func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/admin-recovery", s.netH(s.adminRecoveryPage))
		s.mux.HandleFunc("POST /n/{slug}/admin-recovery", s.netH(s.adminRecoveryPost))
	})
}

type adminRecoveryRow struct {
	Requester, Status         string
	Requested, Expires, First template.HTML
	Closed                    template.HTML
	HasFirst, HasClosed, Mine bool
}

type adminRecoveryEventRow struct {
	When     template.HTML
	Step, By string
}

type adminRecoveryData struct {
	AdminRequired bool
	CanRequest    bool
	Rows          []adminRecoveryRow
	Events        []adminRecoveryEventRow
}

func (s *server) adminRecoveryPage(w http.ResponseWriter, r *http.Request, c *rc) {
	s.adminRecoveryRender(w, r, c, http.StatusOK, &page{})
}

func (s *server) adminRecoveryRender(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	ctx := r.Context()
	views, err := s.svc.ListAdminRecoveryRequests(ctx, c.actor, c.netID(), service.Page{Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	evs, err := s.svc.ListAdminRecoveryEvents(ctx, c.actor, c.netID(), service.AuditQuery{Limit: 200})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	can, err := s.svc.CanRequestAdminRecovery(ctx, c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &adminRecoveryData{AdminRequired: c.Net.Status == domain.NetworkActive && c.Net.AdminRequired, CanRequest: can}
	names := c.memberNames()
	for _, v := range views {
		row := adminRecoveryRow{Requester: names.of(v.RequesterID), Status: messages.Text("admin_recovery.status." + string(v.Status)),
			Requested: c.when(v.RequestedAt), Expires: c.when(v.ExpiresAt), Mine: v.Mine}
		if v.Mine {
			row.Requester = messages.Text("admin_recovery.you")
		}
		if v.FirstApprovedAt != nil {
			row.First, row.HasFirst = c.when(*v.FirstApprovedAt), true
		}
		if v.ClosedAt != nil {
			row.Closed, row.HasClosed = c.when(*v.ClosedAt), true
		}
		d.Rows = append(d.Rows, row)
	}
	for _, e := range evs {
		by := messages.Text("admin_recovery.by_member")
		switch e.ActorType {
		case domain.ActorOperator:
			by = messages.Text("admin_recovery.by_operator")
		case domain.ActorSystem:
			by = messages.Text("admin_recovery.by_system")
		}
		d.Events = append(d.Events, adminRecoveryEventRow{When: c.when(e.OccurredAt), Step: messages.Text("admin_recovery.event." + e.Action), By: by})
	}
	p.Title = messages.Text("admin_recovery.title")
	p.D = d
	s.render(w, r, c, "admin_recovery", status, p)
}

func (s *server) adminRecoveryPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	if v["confirm"] != "1" {
		s.adminRecoveryRender(w, r, c, http.StatusUnprocessableEntity, &page{V: v, Summary: messages.Text("admin_recovery.not_confirmed"),
			Errors: []fieldErr{{"confirm", messages.Text("admin_recovery.confirm_error")}}})
		return
	}
	if _, err := s.svc.RequestAdminRecovery(r.Context(), c.actor, c.netID(), service.AdminRecoveryInput{IdempotencyKey: idemKey(r)}); err != nil {
		sum, fe, ok := c.refuse(err)
		if !ok {
			s.fail(w, r, err)
			return
		}
		if domain.CodeOf(err) == domain.CodeConflict || domain.CodeOf(err) == domain.CodeInvalidState {
			sum = "Not sent. " + messages.Text("admin_recovery.cannot_ask") + " Nothing changed."
		}
		s.adminRecoveryRender(w, r, c, http.StatusConflict, &page{V: v, Summary: sum, Errors: fe})
		return
	}
	s.setFlash(w, r, messages.Text("admin_recovery.sent"))
	http.Redirect(w, r, c.base()+"/admin-recovery", http.StatusSeeOther)
}
