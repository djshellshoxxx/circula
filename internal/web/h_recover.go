package web

import (
	"github.com/djshellshoxxx/circula/internal/service"
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// Account recovery without email (SEC-008). Rules live in the service; this
// file maps forms to calls. Registered through extraRoutes.

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /recover", s.recoverForm)
		s.mux.HandleFunc("POST /recover", s.recoverPost)
		s.mux.HandleFunc("GET /account/recovery-codes", s.auth(s.recoveryCodesForm))
		s.mux.HandleFunc("POST /account/recovery-codes", s.auth(s.recoveryCodesPost))
		s.mux.HandleFunc("GET /n/{slug}/admin/members/{id}/reset-password", s.adminH(s.resetPasswordForm))
		s.mux.HandleFunc("POST /n/{slug}/admin/members/{id}/reset-password", s.adminH(s.resetPasswordPost))
	})
	// The generic confirm page renders this action; the dedicated route above
	// performs it.
	memberActions["reset-password"] = memberActionDef{messages.Text("admin.reset.action"),
		messages.Text("admin.reset.consequence"), true, false}
}

func (s *server) recoverForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "recover", 200, &page{Title: messages.Text("auth.title.recover")})
}

func (s *server) recoverPost(w http.ResponseWriter, r *http.Request) {
	login := r.PostFormValue("login")
	p := &page{Title: messages.Text("auth.title.recover"), V: map[string]string{"login": login}}
	show := func(st int, sum string, fe []fieldErr) {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "recover", st, p)
	}
	ip, acct := s.clientIP(r), acctKey(s.clientIP(r), login)
	if s.ipFail.blocked(ip) || s.acctFail.blocked(acct) {
		show(429, messages.Text("auth.throttle.few_minutes"), nil)
		return
	}
	pw := r.PostFormValue("password")
	if pw != r.PostFormValue("password2") {
		show(422, messages.Text("auth.not_saved"),
			[]fieldErr{{"password2", messages.Text("auth.mismatch.confirm_new")}})
		return
	}
	err := s.svc.RecoverPassword(service.WithClientIP(r.Context(), s.clientIP(r)), login, r.PostFormValue("recovery_code"), pw)
	switch domain.CodeOf(err) {
	case "":
		s.acctFail.reset(acct)
		s.setFlash(w, r, messages.Text("auth.flash.recovered"))
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	case domain.CodeUnauthenticated:
		s.ipFail.fail(ip)
		s.acctFail.fail(acct)
		// One message for unknown login, wrong code and used code.
		show(401, messages.Text("auth.recover.failed"), nil)
	case domain.CodeRateLimited:
		s.ipFail.fail(ip)
		show(429, messages.Text("auth.throttle.login_nothing"), nil)
	case domain.CodeValidation:
		c := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
		if sum, fe, ok := c.refuse(err); ok {
			show(422, sum, fe)
			return
		}
		s.fail(w, r, err)
	default:
		s.fail(w, r, err)
	}
}

type recoveryCodesData struct {
	Remaining int
	Codes     []string
}

func (s *server) recoveryCodesForm(w http.ResponseWriter, r *http.Request, c *rc) {
	n, err := s.svc.RecoveryCodesRemaining(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "recovery_codes", 200, &page{Title: messages.Text("auth.title.recovery_codes"), D: &recoveryCodesData{Remaining: n}})
}

func (s *server) recoveryCodesPost(w http.ResponseWriter, r *http.Request, c *rc) {
	p := &page{Title: messages.Text("auth.title.recovery_codes")}
	key := "recgen|" + c.actor.PersonID
	again := func(st int, sum string, fe []fieldErr) {
		n, _ := s.svc.RecoveryCodesRemaining(r.Context(), c.actor)
		p.D = &recoveryCodesData{Remaining: n}
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "recovery_codes", st, p)
	}
	if s.pwFail.blocked(key) {
		again(429, messages.Text("auth.throttle.few_minutes"), nil)
		return
	}
	codes, err := s.svc.GenerateRecoveryCodes(r.Context(), c.actor, r.PostFormValue("password"))
	switch domain.CodeOf(err) {
	case "":
		s.pwFail.reset(key)
		p.D = &recoveryCodesData{Codes: codes}
		s.render(w, r, nil, "recovery_codes", 200, p)
	case domain.CodeUnauthenticated:
		s.pwFail.fail(key)
		again(422, messages.Text("auth.not_saved"),
			[]fieldErr{{"password", messages.Text("auth.field.current_password_wrong")}})
	case domain.CodeRateLimited:
		again(429, messages.Text("auth.throttle.few_minutes"), nil)
	default:
		s.fail(w, r, err)
	}
}

func (s *server) resetPasswordForm(w http.ResponseWriter, r *http.Request, c *rc) {
	m := c.findMember(r.PathValue("id"))
	if m == nil {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	def := memberActions["reset-password"]
	s.render(w, r, c, "admin_member_act", 200, &page{Title: messages.Text("admin.reset.confirm_title", m.DisplayName),
		D: &memberAct{Key: "reset-password", Def: def, M: m}})
}

type resetDone struct{ Name, Temp string }

func (s *server) resetPasswordPost(w http.ResponseWriter, r *http.Request, c *rc) {
	m := c.findMember(r.PathValue("id"))
	if m == nil {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	def := memberActions["reset-password"]
	p := &page{Title: messages.Text("admin.reset.confirm_title", m.DisplayName), D: &memberAct{Key: "reset-password", Def: def, M: m}, V: formValues(r)}
	temp, err := s.svc.AdminResetPassword(r.Context(), c.actor, c.netID(), m.ID, strings.TrimSpace(r.PostFormValue("reason")))
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeForbidden:
			p.Summary = messages.Text("admin.reset.self_refused")
			s.render(w, r, c, "admin_member_act", 403, p)
		case domain.CodeRateLimited:
			p.Summary = messages.Text("admin.reset.throttled")
			s.render(w, r, c, "admin_member_act", 429, p)
		default:
			s.formFail(w, r, c, "admin_member_act", p, err)
		}
		return
	}
	s.render(w, r, c, "admin_reset_done", 200, &page{Title: messages.Text("admin.reset.done_title"), D: &resetDone{Name: m.DisplayName, Temp: temp}})
}

// mustChange sends a person whose password was reset by someone else to the
// password page, the only screen they may use until they change it.
func (s *server) mustChange(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.setFlash(w, r, messages.Text("auth.flash.must_change"))
	}
	http.Redirect(w, r, "/account/password", http.StatusSeeOther)
}
