package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func init() {
	// The members list offers this action for departed members; the generic
	// confirmation route is shadowed by the more specific one below.
	memberActions["erase"] = memberActionDef{messages.Text("erasure.action"), messages.Text("erasure.consequence"), true, false}
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/admin/members/{id}/erase", s.adminH(s.eraseMemberForm))
		s.mux.HandleFunc("POST /n/{slug}/admin/members/{id}/erase", s.adminH(s.eraseMemberPost))
		s.mux.HandleFunc("GET /n/{slug}/erase-my-data", s.netH(s.eraseMeForm))
		s.mux.HandleFunc("POST /n/{slug}/erase-my-data", s.netH(s.eraseMePost))
	})
}

type eraseData struct {
	MemberID string
	Name     string
	Self     bool
}

func (s *server) eraseMemberForm(w http.ResponseWriter, r *http.Request, c *rc) {
	m := c.findMember(r.PathValue("id"))
	if m == nil || m.Status != domain.MemberDeparted {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	s.render(w, r, c, "erase_confirm", http.StatusOK, &page{Title: messages.Text("erasure.member_title", m.DisplayName),
		D: &eraseData{MemberID: m.ID, Name: m.DisplayName}})
}

func (s *server) eraseMemberPost(w http.ResponseWriter, r *http.Request, c *rc) {
	m := c.findMember(r.PathValue("id"))
	if m == nil || m.Status != domain.MemberDeparted {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	v := formValues(r)
	p := &page{Title: messages.Text("erasure.member_title", m.DisplayName), V: v, D: &eraseData{MemberID: m.ID, Name: m.DisplayName}}
	if v["confirm"] != "yes" {
		p.Summary, p.Errors = messages.Text("erasure.confirm_required"), []fieldErr{{"confirm", messages.Text("erasure.confirm_error")}}
		s.render(w, r, c, "erase_confirm", http.StatusUnprocessableEntity, p)
		return
	}
	if _, err := s.svc.AnonymiseMember(r.Context(), c.actor, c.netID(), m.ID, strings.TrimSpace(v["reason"])); err != nil {
		s.formFail(w, r, c, "erase_confirm", p, err)
		return
	}
	s.setFlash(w, r, messages.Text("erasure.member_done"))
	http.Redirect(w, r, c.base()+"/admin/members?status=departed", http.StatusSeeOther)
}

func (s *server) eraseMeForm(w http.ResponseWriter, r *http.Request, c *rc) {
	if c.Me.Status != domain.MemberDeparted {
		s.fail(w, r, domain.ErrForbidden)
		return
	}
	s.render(w, r, c, "erase_confirm", http.StatusOK, &page{Title: messages.Text("erasure.self_title"), D: &eraseData{Self: true}})
}

func (s *server) eraseMePost(w http.ResponseWriter, r *http.Request, c *rc) {
	if c.Me.Status != domain.MemberDeparted {
		s.fail(w, r, domain.ErrForbidden)
		return
	}
	v := formValues(r)
	p := &page{Title: messages.Text("erasure.self_title"), V: v, D: &eraseData{Self: true}}
	if v["confirm"] != "yes" {
		p.Summary, p.Errors = messages.Text("erasure.confirm_required"), []fieldErr{{"confirm", messages.Text("erasure.confirm_error")}}
		s.render(w, r, c, "erase_confirm", http.StatusUnprocessableEntity, p)
		return
	}
	if _, err := s.svc.EraseMyData(r.Context(), c.actor, c.netID(), strings.TrimSpace(v["reason"])); err != nil {
		s.formFail(w, r, c, "erase_confirm", p, err)
		return
	}
	s.setFlash(w, r, messages.Text("erasure.self_done"))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
