package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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
		row := inboxRow{ID: n.ID, Title: n.Title, Body: n.Body, Unread: n.ReadAt == nil, When: c.when(n.CreatedAt)}
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
		d.Waiting = append(d.Waiting, c.row(t, names))
	}
	ds, err := s.svc.ListDisputesAwaitingMyResponse(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for _, x := range ds {
		d.Disputes = append(d.Disputes, awaitingDispute{ID: x.ID, Category: categoryLabel(x.Category), Opened: c.when(x.OpenedAt)})
	}
	s.render(w, r, c, "inbox", 200, &page{Title: messages.Text("inbox.title"), D: d, V: map[string]string{"show": d.Filter}})
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
		s.setFlash(w, r, messages.Text("inbox.all_read"))
	}
	http.Redirect(w, r, c.base()+"/inbox", http.StatusSeeOther)
}

type reportFormData struct {
	Type, ID, Label                      string
	Cats                                 []option
	DisputeID, CaseDescription, CaseHref string
}

var reportCats = []option{
	{"spam", messages.Text("report.category.spam")}, {"scam", messages.Text("report.category.scam")},
	{"inappropriate", messages.Text("report.category.inappropriate")},
	{"harassment", messages.Text("report.category.harassment")}, {"other", messages.Text("report.category.other")},
}

// reportTarget resolves what is being reported and its display label.
func (c *rc) reportTarget(typ, id string) (string, bool) {
	switch typ {
	case service.ReportListing:
		l, err := c.s.svc.GetListing(c.ctx, c.actor, c.netID(), id)
		if err != nil || l.MemberID == c.Me.ID {
			return "", false
		}
		return messages.Text("report.target_listing", l.Title), true
	case service.ReportMember:
		m, err := c.s.svc.GetMember(c.ctx, c.actor, c.netID(), id)
		if err != nil || m.ID == c.Me.ID {
			return "", false
		}
		return messages.Text("report.target_member", m.DisplayName), true
	case service.ReportTransaction:
		v, err := c.s.svc.GetTransaction(c.ctx, c.actor, c.netID(), id)
		if err != nil || !v.Tx.IsParty(c.Me.ID) || v.Tx.PayerMemberID == "" ||
			v.Tx.PayeeMemberID == "" || v.Tx.PayerMemberID == v.Tx.PayeeMemberID {
			return "", false
		}
		return messages.Text("report.target_transaction", v.Tx.ID), true
	}
	return "", false
}

func (s *server) reportFormData(r *http.Request, c *rc, typ, id, disputeID string) (*reportFormData, error) {
	if !c.isActive() {
		return nil, domain.ErrForbidden
	}
	if disputeID != "" {
		caseContext, err := s.svc.GetReportDisputeContext(r.Context(), c.actor, c.netID(), disputeID)
		if err != nil {
			return nil, err
		}
		var label string
		switch {
		case typ == service.ReportMember && id == caseContext.MemberID:
			label = messages.Text("report.target_member", caseContext.MemberName)
		case typ == service.ReportListing && id != "" && id == caseContext.ListingID:
			label = messages.Text("report.target_case_listing")
			if caseContext.ListingTitle != "" {
				label = messages.Text("report.target_listing", caseContext.ListingTitle)
			}
		default:
			return nil, domain.Invalid("dispute_id", "the selected case does not match the report target")
		}
		return &reportFormData{Type: typ, ID: id, Label: label, Cats: reportCats, DisputeID: caseContext.DisputeID,
			CaseDescription: messages.Text("report.case_description", caseContext.MemberName, messages.Date(c.at(caseContext.OpenedAt))),
			CaseHref:        c.base() + "/disputes/" + caseContext.DisputeID}, nil
	}
	label, ok := c.reportTarget(typ, id)
	if !ok {
		return nil, domain.NotFound("report target")
	}
	return &reportFormData{Type: typ, ID: id, Label: label, Cats: reportCats}, nil
}

func (s *server) reportForm(w http.ResponseWriter, r *http.Request, c *rc) {
	d, err := s.reportFormData(r, c, r.URL.Query().Get("type"), r.URL.Query().Get("id"), r.URL.Query().Get("dispute_id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "report_form", 200, &page{Title: messages.Text("report.title"), D: d, V: map[string]string{"category": "spam"}})
}

func (s *server) reportPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	d, err := s.reportFormData(r, c, v["type"], v["id"], v["dispute_id"])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	_, err = s.svc.CreateReport(r.Context(), c.actor, c.netID(), service.ReportInput{TargetType: d.Type, TargetID: d.ID,
		Category: v["category"], Note: v["note"], DisputeID: d.DisputeID, IdempotencyKey: idemKey(r)})
	if err != nil {
		p := &page{Title: messages.Text("report.title"), D: d, V: v}
		switch domain.CodeOf(err) {
		case domain.CodeConflict:
			p.Summary = messages.Text("report.duplicate")
		case domain.CodeRateLimited:
			p.Summary = messages.Text("report.throttled")
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
	s.setFlash(w, r, messages.Text("report.sent"))
	if d.DisputeID != "" {
		// Build the local destination from the authorized case identifier;
		// the form's display link is not a redirect destination.
		http.Redirect(w, r, c.base()+"/disputes/"+url.PathEscape(d.DisputeID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, c.base(), http.StatusSeeOther)
}

type reportRow struct {
	ID, Reporter, Target, Note, Status, HandlingNote string
	TargetHref                                       string
	DisputesHref                                     string
	DisputeHref                                      string
	Open                                             bool
	Created                                          any
	About                                            messages.Parts
	PlainAbout                                       string
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
		row := reportRow{ID: x.ID, Reporter: x.ReporterName, Target: x.TargetLabel, Note: x.Note,
			Status: x.Status, HandlingNote: x.HandlingNote, Open: x.Status == "open", Created: c.when(x.CreatedAt)}
		row.About, row.PlainAbout = reportAbout(x.Category, x.TargetType, x.TargetLabel)
		switch x.TargetType {
		case service.ReportMember:
			row.TargetHref = c.base() + "/members/" + x.TargetID
		case service.ReportTransaction:
			row.TargetHref = c.base() + "/trades/" + x.TargetID
			row.DisputesHref = c.base() + "/disputes?transaction_id=" + x.TargetID
		}
		if x.DisputeID != "" {
			row.DisputeHref = c.base() + "/disputes/" + x.DisputeID
		}
		d.Rows = append(d.Rows, row)
	}
	if len(rs) == per {
		d.Next = rs[len(rs)-1].ID
	}
	s.render(w, r, c, "admin_reports", 200, &page{Title: messages.Text("report.queue_title"), D: d})
}

// reportAbout passes only known catalogue enum labels to Around. Target names
// stay in the template-owned slot. Unknown category/type values use plain text
// so arbitrary stored values (including NUL or %!) never reach its sentinel.
func reportAbout(category, typ, target string) (messages.Parts, string) {
	var typeLabel string
	switch typ {
	case service.ReportListing:
		typeLabel = messages.Text("report.target_type.listing")
	case service.ReportMember:
		typeLabel = messages.Text("report.target_type.member")
	case service.ReportTransaction:
		typeLabel = messages.Text("report.target_type.transaction")
	default:
		return messages.Parts{}, messages.Text("report.about", categoryOf(category), typ, target)
	}
	for _, option := range reportCats {
		if option.Value == category {
			return messages.Around("report.about", option.Label, typeLabel), ""
		}
	}
	return messages.Parts{}, messages.Text("report.about", category, typeLabel, target)
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
	s.setFlash(w, r, messages.Text("report.handled"))
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
	return messages.Text("notice.credit_limit_used", plainAmt(used, b.Decimals), plainAmt(b.CreditLimit, b.Decimals),
		itoa(c.Net.Policy.LimitWarningPercent))
}
