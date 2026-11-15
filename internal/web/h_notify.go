package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) notifyRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/inbox", s.netH(s.inbox))
	s.mux.HandleFunc("POST /n/{slug}/inbox/read", s.netH(s.inboxRead))
	s.mux.HandleFunc("GET /n/{slug}/report", s.netH(s.reportForm))
	s.mux.HandleFunc("POST /n/{slug}/report", s.netH(s.reportPost))
	s.mux.HandleFunc("GET /n/{slug}/admin/reports", s.adminH(s.reportQueue))
	s.mux.HandleFunc("POST /n/{slug}/admin/reports/{id}/handle", s.adminH(s.reportHandle))
}

// unreadCount is the inbox badge number; any failure shows no badge rather than breaking the page.
func (c *rc) unreadCount() int {
	if c.Me == nil || c.Me.Status == domain.MemberPending {
		return 0
	}
	n, err := c.s.svc.CountUnreadNotifications(c.ctx, c.actor, c.netID())
	if err != nil {
		return 0
	}
	return n
}

type inboxRow struct {
	ID, Title, Body, Href string
	Unread                bool
	When                  any
}

type awaitingDispute struct {
	ID, Category string
	Opened       any
}

type inboxData struct {
	Items    []inboxRow
	Waiting  []txRow
	Disputes []awaitingDispute
	Unread   int
	Next     string
	Filter   string
}

func (s *server) inbox(w http.ResponseWriter, r *http.Request, c *rc) {
	unreadOnly := r.URL.Query().Get("show") == "unread"
	const per = 25
	ns, err := s.svc.ListNotifications(r.Context(), c.actor, c.netID(), service.NotificationQuery{UnreadOnly: unreadOnly,
		Page: service.Page{AfterID: r.URL.Query().Get("after"), Limit: per}})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &inboxData{Unread: c.unreadCount()}
	if unreadOnly {
		d.Filter = "unread"
	}
	for _, n := range ns {
		row := inboxRow{ID: n.ID, Title: n.Title, Body: n.Body, Unread: n.ReadAt == nil, When: fmtTime(n.CreatedAt)}
		if n.Link != "" {
			row.Href = c.base() + n.Link
		}
		d.Items = append(d.Items, row)
	}
	if len(ns) == per {
		d.Next = ns[len(ns)-1].ID
	}
	// "Waiting for you" (FR-038): proposals to answer and disputes to respond to.
	names := c.memberNames()
	pending, err := s.svc.ListPendingForActor(r.Context(), c.actor, c.netID(), service.PendingQuery{AwaitingMe: true, Page: service.Page{Limit: 100}})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for _, t := range pending {
		d.Waiting = append(d.Waiting, makeRow(t, c.Me, names, c.Net.Policy.CurrencyName, c.Net.Policy.Decimals))
	}
	ds, err := s.svc.ListDisputesAwaitingMyResponse(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for _, x := range ds {
		d.Disputes = append(d.Disputes, awaitingDispute{ID: x.ID, Category: categoryLabel(x.Category), Opened: fmtTime(x.OpenedAt)})
	}
	s.render(w, r, c, "inbox", 200, &page{Title: "Inbox", D: d, V: map[string]string{"show": d.Filter}})
}

func (s *server) inboxRead(w http.ResponseWriter, r *http.Request, c *rc) {
	var ids []string
	if id := strings.TrimSpace(r.PostFormValue("id")); id != "" {
		ids = []string{id}
	}
	if _, err := s.svc.MarkNotificationsRead(r.Context(), c.actor, c.netID(), ids); err != nil {
		s.fail(w, r, err)
		return
	}
	if len(ids) == 0 {
		s.setFlash(w, r, "All notifications are marked as read.")
	}
	http.Redirect(w, r, c.base()+"/inbox", http.StatusSeeOther)
}

type reportFormData struct {
	Type, ID, Label string
	Cats            []option
}

var reportCats = []option{
	{"spam", "Spam or advertising"}, {"scam", "Looks like a scam"}, {"inappropriate", "Inappropriate content"},
	{"harassment", "Harassment or abuse"}, {"other", "Something else"},
}

// reportTarget resolves what is being reported and its display label.
func (c *rc) reportTarget(typ, id string) (string, bool) {
	switch typ {
	case service.ReportListing:
		l, err := c.s.svc.GetListing(c.ctx, c.actor, c.netID(), id)
		if err != nil || l.MemberID == c.Me.ID {
			return "", false
		}
		return "Listing: " + l.Title, true
	case service.ReportMember:
		m, err := c.s.svc.GetMember(c.ctx, c.actor, c.netID(), id)
		if err != nil || m.ID == c.Me.ID {
			return "", false
		}
		return "Member: " + m.DisplayName, true
	}
	return "", false
}

func (s *server) reportFormData(r *http.Request, c *rc, typ, id string) (*reportFormData, error) {
	if !c.isActive() {
		return nil, domain.ErrForbidden
	}
	label, ok := c.reportTarget(typ, id)
	if !ok {
		return nil, domain.NotFound("report target")
	}
	return &reportFormData{Type: typ, ID: id, Label: label, Cats: reportCats}, nil
}

func (s *server) reportForm(w http.ResponseWriter, r *http.Request, c *rc) {
	d, err := s.reportFormData(r, c, r.URL.Query().Get("type"), r.URL.Query().Get("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "report_form", 200, &page{Title: "Report a problem", D: d, V: map[string]string{"category": "spam"}})
}

func (s *server) reportPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	d, err := s.reportFormData(r, c, v["type"], v["id"])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	_, err = s.svc.CreateReport(r.Context(), c.actor, c.netID(), service.ReportInput{TargetType: d.Type, TargetID: d.ID,
		Category: v["category"], Note: v["note"], IdempotencyKey: idemKey(r)})
	if err != nil {
		p := &page{Title: "Report a problem", D: d, V: v}
		switch domain.CodeOf(err) {
		case domain.CodeConflict:
			p.Summary = "Not sent. You already have an open report about this. An administrator will look at it. Nothing changed."
		case domain.CodeRateLimited:
			p.Summary = "Not sent. You have sent many reports today. Try again tomorrow. Nothing changed."
		default:
			sum, fe, ok := c.refuse(err)
			if !ok {
				s.fail(w, r, err)
				return
			}
			p.Summary, p.Errors = sum, fe
		}
		s.render(w, r, c, "report_form", 422, p)
		return
	}
	s.setFlash(w, r, "Report sent. Administrators will review it. It does not change anything for the person you reported, and they are not told who reported.")
	http.Redirect(w, r, c.base(), http.StatusSeeOther)
}

type reportRow struct {
	ID, Reporter, Target, Category, Note, Status, HandlingNote string
	TargetType                                                 string
	TargetHref                                                 string
	Open                                                       bool
	Created                                                    any
}

type reportsData struct {
	Rows   []reportRow
	Filter string
	Next   string
}

func (s *server) reportQueue(w http.ResponseWriter, r *http.Request, c *rc) {
	status := r.URL.Query().Get("status")
	if status != "handled" && status != "all" {
		status = "open"
	}
	f := status
	if f == "all" {
		f = ""
	}
	const per = 25
	rs, err := s.svc.ListReports(r.Context(), c.actor, c.netID(), f, service.Page{AfterID: r.URL.Query().Get("after"), Limit: per})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &reportsData{Filter: status}
	for _, x := range rs {
		row := reportRow{ID: x.ID, Reporter: x.ReporterName, Target: x.TargetLabel, Category: categoryOf(x.Category), Note: x.Note,
			Status: x.Status, HandlingNote: x.HandlingNote, TargetType: x.TargetType, Open: x.Status == "open", Created: fmtTime(x.CreatedAt)}
		if x.TargetType == service.ReportMember {
			row.TargetHref = c.base() + "/members/" + x.TargetID
		}
		d.Rows = append(d.Rows, row)
	}
	if len(rs) == per {
		d.Next = rs[len(rs)-1].ID
	}
	s.render(w, r, c, "admin_reports", 200, &page{Title: "Reports", D: d})
}

func categoryOf(v string) string {
	for _, o := range reportCats {
		if o.Value == v {
			return o.Label
		}
	}
	return v
}

func (s *server) reportHandle(w http.ResponseWriter, r *http.Request, c *rc) {
	_, err := s.svc.HandleReport(r.Context(), c.actor, c.netID(), service.HandleReportInput{ReportID: r.PathValue("id"),
		Note: r.PostFormValue("note"), IdempotencyKey: idemKey(r)})
	if err != nil {
		if sum, _, ok := c.refuse(err); ok {
			s.setFlash(w, r, sum)
			http.Redirect(w, r, c.base()+"/admin/reports", http.StatusSeeOther)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Report marked as handled. The reporter was told it was reviewed.")
	http.Redirect(w, r, c.base()+"/admin/reports", http.StatusSeeOther)
}

// limitNotice is the UX-012 inline warning, or "" when the member is below the threshold.
func (c *rc) limitNotice() string {
	b, err := c.s.svc.GetBalance(c.ctx, c.actor, c.netID())
	if err != nil {
		return ""
	}
	used, warn := service.LimitWarning(b, c.Net.Policy.LimitWarningPercent)
	if !warn {
		return ""
	}
	return "You are using " + plainAmt(used, b.Decimals) + " of your " + plainAmt(b.CreditLimit, b.Decimals) +
		" credit limit (" + itoa(c.Net.Policy.LimitWarningPercent) + " percent or more). A trade that would take you past the limit is refused."
}
