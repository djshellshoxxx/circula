package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
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
	memberActions["reset-password"] = memberActionDef{"Reset password",
		"A one-time temporary password is created and shown to you once. The member's sessions and API tokens end, their recovery codes stop working, and they must choose a new password at next sign-in. Give them the temporary password in person or by a channel you trust.", true, false}
}

func (s *server) recoverForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "recover", 200, &page{Title: "Recover your account"})
}

func (s *server) recoverPost(w http.ResponseWriter, r *http.Request) {
	login := r.PostFormValue("login")
	p := &page{Title: "Recover your account", V: map[string]string{"login": login}}
	show := func(st int, sum string, fe []fieldErr) {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "recover", st, p)
	}
	ip, acct := s.clientIP(r), acctKey(s.clientIP(r), login)
	if s.ipFail.blocked(ip) || s.acctFail.blocked(acct) {
		show(429, "Too many attempts. Wait a few minutes and try again. Nothing changed.", nil)
		return
	}
	pw := r.PostFormValue("password")
	if pw != r.PostFormValue("password2") {
		show(422, "Not saved. Fix the problems below. Nothing changed.",
			[]fieldErr{{"password2", "Confirm new password: the two passwords do not match. Type the same password in both fields."}})
		return
	}
	err := s.svc.RecoverPassword(r.Context(), login, r.PostFormValue("recovery_code"), pw)
	switch domain.CodeOf(err) {
	case "":
		s.acctFail.reset(acct)
		s.setFlash(w, r, "Your password was changed and you were signed out everywhere. Sign in with the new password. Make new recovery codes after you sign in.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	case domain.CodeUnauthenticated:
		s.ipFail.fail(ip)
		s.acctFail.fail(acct)
		// One message for unknown login, wrong code and used code.
		show(401, "The login or recovery code is not correct, or the code was already used. Check both and try again. Nothing changed.", nil)
	case domain.CodeRateLimited:
		s.ipFail.fail(ip)
		show(429, "Too many attempts for this login. Wait a few minutes and try again. Nothing changed.", nil)
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
	s.render(w, r, nil, "recovery_codes", 200, &page{Title: "Recovery codes", D: &recoveryCodesData{Remaining: n}})
}

func (s *server) recoveryCodesPost(w http.ResponseWriter, r *http.Request, c *rc) {
	p := &page{Title: "Recovery codes"}
	key := "recgen|" + c.actor.PersonID
	again := func(st int, sum string, fe []fieldErr) {
		n, _ := s.svc.RecoveryCodesRemaining(r.Context(), c.actor)
		p.D = &recoveryCodesData{Remaining: n}
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "recovery_codes", st, p)
	}
	if s.pwFail.blocked(key) {
		again(429, "Too many attempts. Wait a few minutes and try again. Nothing changed.", nil)
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
		again(422, "Not saved. Fix the problems below. Nothing changed.",
			[]fieldErr{{"password", "Current password: not correct. Check it and try again."}})
	case domain.CodeRateLimited:
		again(429, "Too many attempts. Wait a few minutes and try again. Nothing changed.", nil)
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
	s.render(w, r, c, "admin_member_act", 200, &page{Title: def.Verb + ": " + m.DisplayName + "?",
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
	p := &page{Title: def.Verb + ": " + m.DisplayName + "?", D: &memberAct{Key: "reset-password", Def: def, M: m}, V: formValues(r)}
	temp, err := s.svc.AdminResetPassword(r.Context(), c.actor, c.netID(), m.ID, strings.TrimSpace(r.PostFormValue("reason")))
	if err != nil {
		switch domain.CodeOf(err) {
		case domain.CodeForbidden:
			p.Summary = "Not done. You cannot reset your own password here; use Change password. Nothing changed."
			s.render(w, r, c, "admin_member_act", 403, p)
		case domain.CodeRateLimited:
			p.Summary = "Not done. You have reset too many passwords in the last hour. Try again later. Nothing changed."
			s.render(w, r, c, "admin_member_act", 429, p)
		default:
			s.formFail(w, r, c, "admin_member_act", p, err)
		}
		return
	}
	s.render(w, r, c, "admin_reset_done", 200, &page{Title: "Temporary password", D: &resetDone{Name: m.DisplayName, Temp: temp}})
}

// mustChange sends a person whose password was reset by someone else to the
// password page, the only screen they may use until they change it.
func (s *server) mustChange(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.setFlash(w, r, "Your password was reset for you. Choose a new password to continue. Your current password is the temporary one you were given.")
	}
	http.Redirect(w, r, "/account/password", http.StatusSeeOther)
}
