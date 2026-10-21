package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// Account security pages: second factor (SEC-007) and security notices
// (SEC-012). Pages hold no rules; they call the service.

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /account/security", s.auth(s.securityPage))
		s.mux.HandleFunc("POST /account/security/ack", s.auth(s.securityAck))
		s.mux.HandleFunc("GET /account/security/mfa/enrol", s.auth(s.mfaEnrolPage))
		s.mux.HandleFunc("POST /account/security/mfa/enrol", s.auth(s.mfaEnrolBegin))
		s.mux.HandleFunc("POST /account/security/mfa/enrol/confirm", s.auth(s.mfaEnrolConfirm))
		s.mux.HandleFunc("POST /account/security/mfa/disable", s.auth(s.mfaDisable))
		s.mux.HandleFunc("POST /account/security/mfa/recovery", s.auth(s.mfaRecovery))
		s.mux.HandleFunc("GET /login/mfa", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/login", http.StatusSeeOther) // the challenge only travels in the POST
		})
		s.mux.HandleFunc("POST /login/mfa", s.loginMFAPost)
	})
}

var noticeText = map[string]string{
	domain.NoticePasswordChanged:  "Your password was changed and you were signed out everywhere.",
	domain.NoticeMFAEnabled:       "Two-step sign-in was turned on.",
	domain.NoticeMFADisabled:      "Two-step sign-in was turned off.",
	domain.NoticeRecoveryRegen:    "Your recovery codes were replaced. Older codes no longer work.",
	domain.NoticeRecoveryCodeUsed: "A recovery code was used to sign in.",
	domain.NoticeTokenCreated:     "An API token was created.",
	domain.NoticeTokenRevoked:     "An API token was revoked.",
}

type noticeView struct {
	Text   string
	Detail string
	When   time.Time
	New    bool
}

type securityData struct {
	MFA     *service.MFAStatus
	Notices []noticeView
	Unseen  int
}

// secBanner fills the "new security notices" count shown in the page header.
func (s *server) secBanner(r *http.Request, p *page) {
	if a, _, ok := s.authenticate(r); ok {
		if n, err := s.svc.CountUnseenSecurityNotices(r.Context(), a); err == nil {
			p.SecUnseen = n
		}
	}
}

func (s *server) renderSecurity(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	st, err := s.svc.MFAStatus(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ns, err := s.svc.ListSecurityNotices(r.Context(), c.actor, false)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &securityData{MFA: st}
	for _, n := range ns {
		t := noticeText[n.Kind]
		if t == "" {
			t = "A security-related change was made to your account."
		}
		d.Notices = append(d.Notices, noticeView{Text: t, Detail: n.Detail, When: n.CreatedAt, New: n.SeenAt == nil})
		if n.SeenAt == nil {
			d.Unseen++
		}
	}
	p.Title, p.D = "Account security", d
	s.render(w, r, nil, "security", status, p)
}

func (s *server) securityPage(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderSecurity(w, r, c, 200, &page{})
}

func (s *server) securityAck(w http.ResponseWriter, r *http.Request, c *rc) {
	if _, err := s.svc.AckSecurityNotices(r.Context(), c.actor); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Notices marked as read.")
	http.Redirect(w, r, "/account/security", http.StatusSeeOther)
}

// mfaRefusal maps second-factor service errors to page text. ok=false: use fail.
func mfaRefusal(err error, codeField string) (status int, summary string, fe []fieldErr, ok bool) {
	var de *domain.Error
	if !errors.As(err, &de) {
		return 0, "", nil, false
	}
	switch de.Code {
	case domain.CodeValidation:
		for _, f := range de.Fields {
			switch f.Field {
			case "password":
				fe = append(fe, fieldErr{"password", "Password: not correct. Check it and try again."})
			default:
				fe = append(fe, fieldErr{codeField, "Code: not correct. Use the current 6-digit code from your authenticator app" +
					" (or an unused recovery code where allowed). Codes work once."})
			}
		}
		return 422, "Not saved. Fix the problems below. Nothing changed.", fe, true
	case domain.CodeRateLimited:
		return 429, "Too many attempts. Wait a few minutes and try again. Nothing changed.", nil, true
	case domain.CodeInvalidState:
		return 409, "Not done. Two-step sign-in is not in a state that allows that (for example the setup expired, or it is not turned on). Nothing changed.", nil, true
	case domain.CodeConflict:
		return 409, "Not done. Two-step sign-in is already turned on. Nothing changed.", nil, true
	}
	return 0, "", nil, false
}

func (s *server) mfaEnrolPage(w http.ResponseWriter, r *http.Request, c *rc) {
	st, err := s.svc.MFAStatus(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "mfa_enrol", 200, &page{Title: "Turn on two-step sign-in", D: &enrolData{Status: st}})
}

type enrolData struct {
	Status *service.MFAStatus
	Enrol  *service.MFAEnrolment // set only on the response that shows the secret
}

// showEnrol renders the setup key. Enrolment needs the current password (A3-03),
// so it is asked for again here; when it is wrong the start page comes back
// with that error instead.
func (s *server) showEnrol(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page, password string) {
	en, err := s.svc.BeginMFAEnrolment(r.Context(), c.actor, password) // resumes the pending secret
	d := &enrolData{Enrol: en}
	if err != nil {
		st, sum, fe, ok := mfaRefusal(err, "code")
		if !ok {
			s.fail(w, r, err)
			return
		}
		p.Summary, p.Errors, status = sum, fe, st
		if d.Status, err = s.svc.MFAStatus(r.Context(), c.actor); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	p.Title, p.D = "Turn on two-step sign-in", d
	s.render(w, r, nil, "mfa_enrol", status, p)
}

func (s *server) enrolBlocked(w http.ResponseWriter, r *http.Request, c *rc) {
	st, err := s.svc.MFAStatus(r.Context(), c.actor)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "mfa_enrol", 429, &page{Title: "Turn on two-step sign-in", D: &enrolData{Status: st},
		Summary: "Too many attempts. Wait a few minutes and try again. Nothing changed."})
}

func (s *server) mfaEnrolBegin(w http.ResponseWriter, r *http.Request, c *rc) {
	key := "mfa|" + c.actor.PersonID
	if s.pwFail.blocked(key) {
		s.enrolBlocked(w, r, c)
		return
	}
	s.showEnrol(w, r, c, 200, &page{}, r.PostFormValue("password"))
}

func (s *server) mfaEnrolConfirm(w http.ResponseWriter, r *http.Request, c *rc) {
	key := "mfa|" + c.actor.PersonID
	if s.pwFail.blocked(key) {
		s.enrolBlocked(w, r, c)
		return
	}
	pw := r.PostFormValue("password")
	codes, err := s.svc.ConfirmMFAEnrolment(r.Context(), c.actor, pw, r.PostFormValue("code"))
	if err != nil {
		if st, sum, fe, ok := mfaRefusal(err, "code"); ok {
			if domain.CodeOf(err) == domain.CodeValidation {
				s.pwFail.fail(key)
			}
			p := &page{Summary: sum, Errors: fe}
			s.showEnrol(w, r, c, st, p, pw)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.pwFail.reset(key)
	s.render(w, r, nil, "mfa_recovery", 200, &page{Title: "Save your recovery codes", D: codes})
}

func (s *server) mfaDisable(w http.ResponseWriter, r *http.Request, c *rc) {
	key := "mfa|" + c.actor.PersonID
	if s.pwFail.blocked(key) {
		s.renderSecurity(w, r, c, 429, &page{Summary: "Too many attempts. Wait a few minutes and try again. Nothing changed."})
		return
	}
	err := s.svc.DisableMFA(r.Context(), c.actor, r.PostFormValue("password"), r.PostFormValue("code"))
	if err != nil {
		if st, sum, fe, ok := mfaRefusal(err, "disable_code"); ok {
			if domain.CodeOf(err) == domain.CodeValidation {
				s.pwFail.fail(key)
			}
			s.renderSecurity(w, r, c, st, &page{Summary: sum, Errors: fe})
			return
		}
		s.fail(w, r, err)
		return
	}
	s.pwFail.reset(key)
	s.setFlash(w, r, "Two-step sign-in is turned off.")
	http.Redirect(w, r, "/account/security", http.StatusSeeOther)
}

func (s *server) mfaRecovery(w http.ResponseWriter, r *http.Request, c *rc) {
	key := "mfa|" + c.actor.PersonID
	if s.pwFail.blocked(key) {
		s.renderSecurity(w, r, c, 429, &page{Summary: "Too many attempts. Wait a few minutes and try again. Nothing changed."})
		return
	}
	codes, err := s.svc.RegenerateRecoveryCodes(r.Context(), c.actor, r.PostFormValue("code"))
	if err != nil {
		if st, sum, fe, ok := mfaRefusal(err, "regen_code"); ok {
			if domain.CodeOf(err) == domain.CodeValidation {
				s.pwFail.fail(key)
			}
			s.renderSecurity(w, r, c, st, &page{Summary: sum, Errors: fe})
			return
		}
		s.fail(w, r, err)
		return
	}
	s.pwFail.reset(key)
	s.render(w, r, nil, "mfa_recovery", 200, &page{Title: "Save your new recovery codes", D: codes})
}

// renderMFAStep shows the second step of sign-in. The challenge is a
// single-use, five-minute value that only exists in this form.
func (s *server) renderMFAStep(w http.ResponseWriter, r *http.Request, status int, challenge, next, summary string) {
	p := &page{Title: "Enter your authentication code", D: challenge, Next: next, Summary: summary}
	if status == 401 {
		p.Errors = []fieldErr{{"code", "Code: not correct, or the sign-in timed out. Check the code and try again, or sign in again."}}
	}
	s.render(w, r, nil, "login_mfa", status, p)
}

func (s *server) loginMFAPost(w http.ResponseWriter, r *http.Request) {
	challenge, next := r.PostFormValue("challenge"), safeNext(r.PostFormValue("next"))
	ip := s.clientIP(r)
	if s.ipFail.blocked(ip) {
		s.renderMFAStep(w, r, 429, challenge, next, "Too many attempts. Wait a minute and try again. Nothing changed.")
		return
	}
	tok, _, err := s.svc.CompleteMFA(r.Context(), challenge, r.PostFormValue("code"))
	switch domain.CodeOf(err) {
	case "":
		s.startSession(w, r, tok)
		dest := next
		if dest == "" {
			dest = "/"
		}
		http.Redirect(w, r, dest, http.StatusSeeOther) //nolint:gosec // G710: dest comes from safeNext (local paths only)
	case domain.CodeUnauthenticated, domain.CodeValidation:
		s.ipFail.fail(ip)
		s.renderMFAStep(w, r, 401, challenge, next, "The code is not correct, or the sign-in timed out. Nothing changed.")
	case domain.CodeRateLimited:
		s.renderMFAStep(w, r, 429, challenge, next, "Too many attempts for this account. Wait a few minutes and try again.")
	default:
		s.fail(w, r, err)
	}
}
