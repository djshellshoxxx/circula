package web

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func (s *server) adminRoutes() {
	s.adminOverviewRoutes()
	s.mux.HandleFunc("GET /n/{slug}/admin", s.adminH(s.adminHome))
	s.mux.HandleFunc("GET /n/{slug}/admin/members", s.adminH(s.adminMembers))
	s.mux.HandleFunc("GET /n/{slug}/admin/members/{id}/{action}", s.adminH(s.memberActionForm))
	s.mux.HandleFunc("POST /n/{slug}/admin/members/{id}/{action}", s.adminH(s.memberActionPost))
	s.mux.HandleFunc("GET /n/{slug}/admin/invitations", s.adminH(s.invitations))
	s.mux.HandleFunc("POST /n/{slug}/admin/invitations", s.adminH(s.invitationCreate))
	s.mux.HandleFunc("POST /n/{slug}/admin/invitations/{id}/revoke", s.adminH(s.invitationRevoke))
	s.mux.HandleFunc("GET /n/{slug}/admin/policy", s.adminH(s.policyForm))
	s.mux.HandleFunc("POST /n/{slug}/admin/policy", s.adminH(s.policyPost))
	s.mux.HandleFunc("GET /n/{slug}/admin/adjust", s.adminH(s.adjustForm))
	s.mux.HandleFunc("POST /n/{slug}/admin/adjust", s.adminH(s.adjustPost))
	s.mux.HandleFunc("GET /n/{slug}/admin/audit", s.adminH(s.auditPage))
	s.mux.HandleFunc("POST /n/{slug}/admin/check", s.adminH(s.ledgerCheck))
	s.mux.HandleFunc("GET /n/{slug}/admin/exports", s.adminH(s.exportsPage))
	s.mux.HandleFunc("POST /n/{slug}/admin/exports", s.adminH(s.exportsPost))
}

func (s *server) adminHome(w http.ResponseWriter, r *http.Request, c *rc) {
	s.overview(w, r, c)
}

type memberRow struct {
	*domain.Member
	StatusLabel string
	Roles       []string
	Actions     []option
}

func memberStatusLabel(status domain.MemberStatus) string {
	switch status {
	case domain.MemberPending, domain.MemberActive, domain.MemberSuspended, domain.MemberDeparted:
		return messages.Text("admin.members.status." + string(status))
	}
	return string(status)
}

func memberRoleLabel(role domain.Role) string {
	switch role {
	case domain.RoleAdmin, domain.RoleReviewer:
		return messages.Text("admin.members.role." + string(role))
	}
	return string(role)
}

func removalReasonLabel(code string) string {
	switch code {
	case domain.RemovalRuleBreach, domain.RemovalConfirmedFraud, domain.RemovalDisputeNonResponse:
		return messages.Text("admin.member_action.removal_reason." + code)
	}
	return code
}

func newMemberRow(m *domain.Member) memberRow {
	var roles []string
	for _, role := range m.Roles {
		roles = append(roles, memberRoleLabel(role))
	}
	return memberRow{Member: m, StatusLabel: memberStatusLabel(m.Status), Roles: roles, Actions: actionsFor(m)}
}

type memberActionDef struct {
	Verb, Consequence string
	Reason            bool // reason mandatory
	Amount            bool
}

var memberActions = map[string]memberActionDef{
	"approve":         {messages.Text("admin.member_action.approve"), messages.Text("admin.member_action.approve_consequence"), false, false},
	"deny":            {messages.Text("admin.member_action.deny"), messages.Text("admin.member_action.deny_consequence"), true, false},
	"suspend":         {messages.Text("admin.member_action.suspend"), messages.Text("admin.member_action.suspend_consequence"), true, false},
	"restore":         {messages.Text("admin.member_action.restore"), messages.Text("admin.member_action.restore_consequence"), true, false},
	"remove":          {messages.Text("admin.member_action.remove"), messages.Text("admin.member_action.remove_consequence"), false, false}, // reason code instead (GOV-011)
	"grant-admin":     {messages.Text("admin.member_action.grant_admin"), messages.Text("admin.member_action.grant_admin_consequence"), false, false},
	"revoke-admin":    {messages.Text("admin.member_action.revoke_admin"), messages.Text("admin.member_action.revoke_admin_consequence"), false, false},
	"grant-reviewer":  {messages.Text("admin.member_action.grant_reviewer"), messages.Text("admin.member_action.grant_reviewer_consequence"), false, false},
	"revoke-reviewer": {messages.Text("admin.member_action.revoke_reviewer"), messages.Text("admin.member_action.revoke_reviewer_consequence"), false, false},
	"flag":            {messages.Text("admin.member_action.flag"), messages.Text("admin.member_action.flag_consequence"), true, false},
	"unflag":          {messages.Text("admin.member_action.unflag"), messages.Text("admin.member_action.unflag_consequence"), true, false},
	"limit":           {messages.Text("admin.member_action.limit"), messages.Text("admin.member_action.limit_consequence"), true, true},
}

func actionsFor(m *domain.Member) []option {
	var keys []string
	switch m.Status {
	case domain.MemberPending:
		keys = []string{"approve", "deny"}
	case domain.MemberActive:
		keys = []string{"limit", "suspend", "remove", "reset-password"}
		if m.HasRole(domain.RoleAdmin) {
			keys = append(keys, "revoke-admin")
		} else {
			keys = append(keys, "grant-admin")
		}
		if m.HasRole(domain.RoleReviewer) {
			keys = append(keys, "revoke-reviewer")
		} else {
			keys = append(keys, "grant-reviewer")
		}
		if m.DefaultFlag {
			keys = append(keys, "unflag")
		} else {
			keys = append(keys, "flag")
		}
	case domain.MemberSuspended:
		keys = []string{"restore", "remove"}
	case domain.MemberDeparted:
		keys = []string{"erase"} // SEC-018; see h_erasure.go
	}
	var o []option
	for _, k := range keys {
		o = append(o, option{k, memberActions[k].Verb})
	}
	return o
}

func (s *server) adminMembers(w http.ResponseWriter, r *http.Request, c *rc) {
	f := r.URL.Query().Get("status")
	var st []domain.MemberStatus
	for _, x := range []domain.MemberStatus{domain.MemberPending, domain.MemberActive, domain.MemberSuspended, domain.MemberDeparted} {
		if string(x) == f {
			st = []domain.MemberStatus{x}
		}
	}
	total, err := s.svc.CountMembers(r.Context(), c.actor, c.netID(), st)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	cur := r.URL.Query()
	mq := service.MemberPageQuery{Limit: 100}
	paged := false
	switch {
	case cur.Get("after") != "":
		mq.AfterID, paged = cur.Get("after"), true
	case cur.Get("before") != "":
		mq.BeforeID, paged = cur.Get("before"), true
	}
	pg, err := s.svc.AdminMemberPage(r.Context(), c.actor, c.netID(), st, mq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var rows []memberRow
	for _, m := range pg.Members {
		rows = append(rows, newMemberRow(m))
	}
	base := c.base() + "/admin/members?" + url.Values{"status": {f}}.Encode()
	pn := pageNav{Total: total}
	if paged {
		pn.First = base
	}
	if n := len(pg.Members); n > 0 {
		if pg.HasPrev {
			pn.Prev = base + "&before=" + url.QueryEscape(pg.Members[0].ID)
		}
		if pg.HasNext {
			pn.Next = base + "&after=" + url.QueryEscape(pg.Members[n-1].ID)
		}
	}
	s.render(w, r, c, "admin_members", 200, &page{Title: messages.Text("admin.members.title"), D: map[string]any{"Rows": rows, "Pages": pn},
		V: map[string]string{"status": f}})
}

// pageNav carries the links and totals for a keyset-paged list. Empty hrefs mean no such page.
type pageNav struct {
	Total             int
	First, Prev, Next string
}

type memberAct struct {
	Key   string
	Def   memberActionDef
	M     *domain.Member
	Unit  string
	Codes []option // remove: the documented removal reasons (GOV-011)
}

func newMemberAct(key string, def memberActionDef, m *domain.Member) *memberAct {
	a := &memberAct{Key: key, Def: def, M: m}
	if key == "remove" {
		a.Codes = []option{{"", messages.Text("admin.member_action.choose_reason")}}
		for _, c := range domain.RemovalReasons {
			a.Codes = append(a.Codes, option{c, removalReasonLabel(c)})
		}
	}
	return a
}

func (c *rc) findMember(id string) *domain.Member {
	m, err := c.s.svc.GetMember(c.ctx, c.actor, c.netID(), id)
	if err != nil {
		return nil
	}
	return m
}

func (s *server) memberActionForm(w http.ResponseWriter, r *http.Request, c *rc) {
	def, ok := memberActions[r.PathValue("action")]
	m := c.findMember(r.PathValue("id"))
	if !ok || m == nil {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	s.render(w, r, c, "admin_member_act", 200, &page{Title: messages.Text("admin.member_action.confirm_title", def.Verb, m.DisplayName),
		D: newMemberAct(r.PathValue("action"), def, m)})
}

func (s *server) memberActionPost(w http.ResponseWriter, r *http.Request, c *rc) {
	key := r.PathValue("action")
	def, ok := memberActions[key]
	m := c.findMember(r.PathValue("id"))
	if !ok || m == nil {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	v := formValues(r)
	p := &page{Title: messages.Text("admin.member_action.confirm_title", def.Verb, m.DisplayName), D: newMemberAct(key, def, m), V: v}
	reason := strings.TrimSpace(v["reason"])
	ctx, id, nid := r.Context(), m.ID, c.netID()
	var err error
	switch key {
	case "approve":
		_, err = s.svc.DecideJoinRequest(ctx, c.actor, nid, id, true, reason)
	case "deny":
		_, err = s.svc.DecideJoinRequest(ctx, c.actor, nid, id, false, reason)
	case "suspend":
		err = s.svc.SuspendMember(ctx, c.actor, nid, id, reason)
	case "restore":
		err = s.svc.RestoreMember(ctx, c.actor, nid, id, reason)
	case "remove":
		_, err = s.svc.DepartMember(ctx, c.actor, nid, service.DepartInput{MemberID: id, ReasonCode: v["reason_code"],
			Reason: reason, MatterTxID: strings.TrimSpace(v["transaction_id"])})
		if service.IsRemovalConflict(err) { // GOV-011, SEC-010: say why, keep the form
			p.Summary = messages.Text("admin.member_action.removal_conflict")
			s.render(w, r, c, "admin_member_act", http.StatusForbidden, p)
			return
		}
	case "grant-admin":
		err = s.svc.AssignRole(ctx, c.actor, nid, id, domain.RoleAdmin)
	case "revoke-admin":
		err = s.svc.RevokeRole(ctx, c.actor, nid, id, domain.RoleAdmin)
	case "grant-reviewer":
		err = s.svc.AssignRole(ctx, c.actor, nid, id, domain.RoleReviewer)
	case "revoke-reviewer":
		err = s.svc.RevokeRole(ctx, c.actor, nid, id, domain.RoleReviewer)
	case "flag", "unflag":
		err = s.svc.SetDefaultFlag(ctx, c.actor, nid, id, key == "flag", reason)
	case "limit":
		in := service.SetLimitInput{MemberID: id, Reason: reason}
		if strings.TrimSpace(v["amount"]) != "" {
			n, perr := domain.ParseAmount(v["amount"], c.Net.Policy.Decimals)
			if perr != nil {
				p.Errors = []fieldErr{{"amount", c.amountHelp()}}
				p.Summary = messages.Text("admin.member_action.invalid")
				s.render(w, r, c, "admin_member_act", 422, p)
				return
			}
			in.Limit = &n
		}
		err = s.svc.SetCreditLimit(ctx, c.actor, nid, in)
	}
	if err != nil {
		s.formFail(w, r, c, "admin_member_act", p, err)
		return
	}
	s.setFlash(w, r, messages.Text("admin.member_action.done", def.Verb, m.DisplayName))
	http.Redirect(w, r, c.base()+"/admin/members", http.StatusSeeOther)
}

type invData struct {
	List    []invRow
	NewLink string
}

type invRow struct {
	ID, Status   string
	Expires      any
	ExpiresLabel string
	CanRevoke    bool
}

// invitationStatus preserves the existing used/revoked/expired precedence and
// strict expiry boundary. Display labels never decide whether revocation is shown.
func invitationStatus(i *domain.Invitation, now time.Time) (string, bool) {
	switch {
	case i.UsedAt != nil:
		return messages.Text("admin.invitations.status.used"), false
	case i.RevokedAt != nil:
		return messages.Text("admin.invitations.status.revoked"), false
	case i.ExpiresAt.Before(now):
		return messages.Text("admin.invitations.status.expired"), false
	default:
		return messages.Text("admin.invitations.status.unused"), true
	}
}

func (c *rc) invitationRow(i *domain.Invitation, now time.Time) invRow {
	status, canRevoke := invitationStatus(i, now)
	// Status is always one of the catalogue labels above, never submitted text.
	// The plain accessible name uses the same viewer zone/layout as the time element.
	return invRow{ID: i.ID, Status: status, Expires: c.when(i.ExpiresAt),
		ExpiresLabel: messages.DateTime(c.at(i.ExpiresAt)), CanRevoke: canRevoke}
}

func (s *server) invitations(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderInvites(w, r, c, 200, &page{V: map[string]string{"days": "7"}}, "")
}

func (s *server) renderInvites(w http.ResponseWriter, r *http.Request, c *rc, st int, p *page, link string) {
	d := &invData{NewLink: link}
	ls, err := s.svc.ListInvitations(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	now := s.opts.Clock.Now()
	for _, i := range ls {
		d.List = append(d.List, c.invitationRow(i, now))
	}
	p.Title, p.D = messages.Text("admin.invitations.title"), d
	s.render(w, r, c, "admin_invites", st, p)
}

func (s *server) invitationCreate(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{V: v}
	days, err := parseDays(v["days"])
	if err != nil {
		p.Summary, p.Errors = messages.Text("admin.invitations.invalid"), []fieldErr{{"days", messages.Text("admin.invitations.days_error")}}
		s.renderInvites(w, r, c, 422, p, "")
		return
	}
	tok, _, err := s.svc.CreateInvitation(r.Context(), c.actor, c.netID(), service.InviteInput{TTL: time.Duration(days) * 24 * time.Hour})
	if err != nil {
		if sum, fe, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			s.renderInvites(w, r, c, 422, p, "")
			return
		}
		s.fail(w, r, err)
		return
	}
	s.renderInvites(w, r, c, 200, &page{V: map[string]string{"days": "7"}}, s.baseURL(r)+"/join/"+tok)
}

func parseDays(s string) (int, error) {
	n := 0
	if s == "" || len(s) > 3 {
		return 0, domain.Invalid("days", "bad")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, domain.Invalid("days", "bad")
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 || n > 90 {
		return 0, domain.Invalid("days", "bad")
	}
	return n, nil
}

func (s *server) invitationRevoke(w http.ResponseWriter, r *http.Request, c *rc) {
	if err := s.svc.RevokeInvitation(r.Context(), c.actor, c.netID(), r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, messages.Text("admin.invitations.revoked"))
	http.Redirect(w, r, c.base()+"/admin/invitations", http.StatusSeeOther)
}

// baseURL is the origin used in links shown to users: the configured public
// base URL, else the scheme and Host of this request.
func (s *server) baseURL(r *http.Request) string {
	if s.opts.PublicBaseURL != "" {
		return strings.TrimRight(s.opts.PublicBaseURL, "/")
	}
	scheme := "http"
	if s.secure(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
