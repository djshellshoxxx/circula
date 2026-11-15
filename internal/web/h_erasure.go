package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func init() {
	// The members list offers this action for departed members; the generic
	// confirmation route is shadowed by the more specific one below.
	memberActions["erase"] = memberActionDef{"Erase personal data", "Irreversible. The member's name, listings, trade notes and dispute statements are replaced or cleared. Balances and the ledger stay exactly as they are.", true, false}
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
	s.render(w, r, c, "erase_confirm", http.StatusOK, &page{Title: "Erase personal data of " + m.DisplayName + "?",
		D: &eraseData{MemberID: m.ID, Name: m.DisplayName}})
}

func (s *server) eraseMemberPost(w http.ResponseWriter, r *http.Request, c *rc) {
	m := c.findMember(r.PathValue("id"))
	if m == nil || m.Status != domain.MemberDeparted {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	v := formValues(r)
	p := &page{Title: "Erase personal data of " + m.DisplayName + "?", V: v, D: &eraseData{MemberID: m.ID, Name: m.DisplayName}}
	if v["confirm"] != "yes" {
		p.Summary, p.Errors = "Not done. Tick the box to confirm. Nothing changed.", []fieldErr{{"confirm", "Confirm: tick the box to show you understand this cannot be undone."}}
		s.render(w, r, c, "erase_confirm", http.StatusUnprocessableEntity, p)
		return
	}
	if _, err := s.svc.AnonymiseMember(r.Context(), c.actor, c.netID(), m.ID, strings.TrimSpace(v["reason"])); err != nil {
		s.formFail(w, r, c, "erase_confirm", p, err)
		return
	}
	s.setFlash(w, r, "Personal data erased. The member now appears as a former member.")
	http.Redirect(w, r, c.base()+"/admin/members?status=departed", http.StatusSeeOther)
}

func (s *server) eraseMeForm(w http.ResponseWriter, r *http.Request, c *rc) {
	if c.Me.Status != domain.MemberDeparted {
		s.fail(w, r, domain.ErrForbidden)
		return
	}
	s.render(w, r, c, "erase_confirm", http.StatusOK, &page{Title: "Erase my personal data?", D: &eraseData{Self: true}})
}

func (s *server) eraseMePost(w http.ResponseWriter, r *http.Request, c *rc) {
	if c.Me.Status != domain.MemberDeparted {
		s.fail(w, r, domain.ErrForbidden)
		return
	}
	v := formValues(r)
	p := &page{Title: "Erase my personal data?", V: v, D: &eraseData{Self: true}}
	if v["confirm"] != "yes" {
		p.Summary, p.Errors = "Not done. Tick the box to confirm. Nothing changed.", []fieldErr{{"confirm", "Confirm: tick the box to show you understand this cannot be undone."}}
		s.render(w, r, c, "erase_confirm", http.StatusUnprocessableEntity, p)
		return
	}
	if _, err := s.svc.EraseMyData(r.Context(), c.actor, c.netID(), strings.TrimSpace(v["reason"])); err != nil {
		s.formFail(w, r, c, "erase_confirm", p, err)
		return
	}
	s.setFlash(w, r, "Your personal data in this network has been erased.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
