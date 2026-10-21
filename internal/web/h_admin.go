package web

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) adminRoutes() {
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
	pend, err := s.svc.CountMembers(r.Context(), c.actor, c.netID(), []domain.MemberStatus{domain.MemberPending})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "admin_home", 200, &page{Title: "Network admin", D: pend})
}

type memberRow struct {
	*domain.Member
	Roles   string
	Actions []option
}

type memberActionDef struct {
	Verb, Consequence string
	Reason            bool // reason mandatory
	Amount            bool
}

var memberActions = map[string]memberActionDef{
	"approve":         {"Approve", "The person becomes an active member and can trade.", false, false},
	"deny":            {"Deny", "The request is closed and the person does not join.", true, false},
	"suspend":         {"Suspend", "The member can no longer propose, accept or post. They can still view history and decline trades.", true, false},
	"restore":         {"Restore", "The member becomes active again.", true, false},
	"remove":          {"Remove", "The membership ends. Open proposals are cancelled and the member keeps read-only access to their own history. This cannot be undone: a departed member cannot rejoin.", true, false},
	"grant-admin":     {"Make administrator", "The member can change policy, limits and memberships and see other members' balances.", false, false},
	"revoke-admin":    {"Remove administrator role", "The member loses administrator rights. A network must keep at least one administrator.", false, false},
	"grant-reviewer":  {"Make dispute reviewer", "The member can decide disputes they are not a party to.", false, false},
	"revoke-reviewer": {"Remove reviewer role", "The member can no longer decide disputes.", false, false},
	"flag":            {"Mark as in default", "The member is marked as in default (a governance marker); the network policy decides what that blocks.", true, false},
	"unflag":          {"Clear default marker", "The default marker is removed.", true, false},
	"limit":           {"Set credit limit", "Sets this member's credit limit. Leave the amount empty to go back to the network default.", true, true},
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
		var roles []string
		for _, ro := range m.Roles {
			roles = append(roles, string(ro))
		}
		rows = append(rows, memberRow{Member: m, Roles: strings.Join(roles, ", "), Actions: actionsFor(m)})
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
	s.render(w, r, c, "admin_members", 200, &page{Title: "Members", D: map[string]any{"Rows": rows, "Pages": pn},
		V: map[string]string{"status": f}})
}

// pageNav carries the links and totals for a keyset-paged list. Empty hrefs mean no such page.
type pageNav struct {
	Total             int
	First, Prev, Next string
}

type memberAct struct {
	Key  string
	Def  memberActionDef
	M    *domain.Member
	Unit string
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
	s.render(w, r, c, "admin_member_act", 200, &page{Title: def.Verb + ": " + m.DisplayName + "?",
		D: &memberAct{Key: r.PathValue("action"), Def: def, M: m}})
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
	p := &page{Title: def.Verb + ": " + m.DisplayName + "?", D: &memberAct{Key: key, Def: def, M: m}, V: v}
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
		_, err = s.svc.DepartMember(ctx, c.actor, nid, service.DepartInput{MemberID: id, Reason: reason})
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
				p.Summary = "Not saved. Fix the problems below. Nothing changed."
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
	s.setFlash(w, r, def.Verb+": done for "+m.DisplayName+".")
	http.Redirect(w, r, c.base()+"/admin/members", http.StatusSeeOther)
}

type invData struct {
	List    []invRow
	NewLink string
}

type invRow struct {
	ID, Status string
	Expires    any
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
		status := "Unused"
		switch {
		case i.UsedAt != nil:
			status = "Used"
		case i.RevokedAt != nil:
			status = "Revoked"
		case i.ExpiresAt.Before(now):
			status = "Expired"
		}
		d.List = append(d.List, invRow{i.ID, status, fmtTime(i.ExpiresAt)})
	}
	p.Title, p.D = "Invitations", d
	s.render(w, r, c, "admin_invites", st, p)
}

func (s *server) invitationCreate(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{V: v}
	days, err := parseDays(v["days"])
	if err != nil {
		p.Summary, p.Errors = "Not created. Fix the problems below. Nothing changed.", []fieldErr{{"days", "Valid for: enter a whole number of days from 1 to 90."}}
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
	s.setFlash(w, r, "Invitation revoked.")
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
