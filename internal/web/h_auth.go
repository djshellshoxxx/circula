package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) authRoutes() {
	s.mux.HandleFunc("GET /login", s.loginForm)
	s.mux.HandleFunc("POST /login", s.loginPost)
	s.mux.HandleFunc("POST /logout", s.logout)
	s.mux.HandleFunc("GET /register", s.registerForm)
	s.mux.HandleFunc("POST /register", s.registerPost)
	s.mux.HandleFunc("GET /setup", s.auth(s.setupForm))
	s.mux.HandleFunc("POST /setup", s.auth(s.setupPost))
	s.mux.HandleFunc("GET /join/{token}", s.joinInviteForm)
	s.mux.HandleFunc("POST /join/{token}", s.auth(s.joinInvitePost))
	s.mux.HandleFunc("GET /n/{slug}/join", s.auth(s.joinNetForm))
	s.mux.HandleFunc("POST /n/{slug}/join", s.auth(s.joinNetPost))
	s.mux.HandleFunc("GET /help", s.help)
}

// formValues copies submitted values for redisplay; secrets are never kept.
func formValues(r *http.Request) map[string]string {
	_ = r.ParseForm()
	v := map[string]string{}
	for k, vs := range r.PostForm {
		if len(vs) > 0 && k != "password" && k != "csrf" && k != "setup_secret" {
			v[k] = vs[0]
		}
	}
	return v
}

func (s *server) loginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "login", 200, &page{Title: "Sign in", Next: safeNext(r.URL.Query().Get("next"))})
}

func (s *server) loginPost(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: "Sign in", V: formValues(r), Next: safeNext(r.PostFormValue("next"))}
	ip, acct := s.clientIP(r), acctKey(s.clientIP(r), r.PostFormValue("login"))
	if s.ipFail.blocked(ip) || s.acctFail.blocked(acct) {
		p.Summary = "Too many attempts. Wait a minute and try again. Nothing changed."
		s.render(w, r, nil, "login", 429, p)
		return
	}
	res, err := s.svc.LoginStep(r.Context(), r.PostFormValue("login"), r.PostFormValue("password"))
	if err != nil {
		if c := domain.CodeOf(err); c == domain.CodeValidation || c == domain.CodeUnauthenticated || c == domain.CodeRateLimited {
			s.ipFail.fail(ip)
			s.acctFail.fail(acct)
		}
		switch domain.CodeOf(err) {
		case domain.CodeRateLimited:
			p.Summary = "Too many attempts for this login. Wait a few minutes and try again."
			s.render(w, r, nil, "login", 429, p)
		case domain.CodeValidation, domain.CodeUnauthenticated:
			// One message for unknown login and wrong password (UX-052).
			p.Summary = "The login or password is not correct. Check both and try again."
			p.Errors = []fieldErr{{"login", "Login or password: check both and try again."}}
			s.render(w, r, nil, "login", 401, p)
		default:
			s.fail(w, r, err)
		}
		return
	}
	s.acctFail.reset(acct)
	if res.MFAChallenge != "" { // second factor enabled: no session until CompleteMFA (SEC-007)
		s.renderMFAStep(w, r, 200, res.MFAChallenge, p.Next, "")
		return
	}
	s.startSession(w, r, res.Token)
	dest := p.Next
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther) //nolint:gosec // G710: dest comes from safeNext (local paths only)
}

func (s *server) startSession(w http.ResponseWriter, r *http.Request, tok string) {
	s.setSession(w, r, tok)
	s.rotateCSRF(w, r)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if tok := s.opts.Edge.SessionToken(r); tok != "" {
		_ = s.svc.Logout(r.Context(), tok)
	}
	s.clearSession(w, r)
	s.rotateCSRF(w, r)
	s.setFlash(w, r, "You are signed out.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *server) registerForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "register", 200, &page{Title: "Create an account", Next: safeNext(r.URL.Query().Get("next"))})
}

func (s *server) registerPost(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: "Create an account", V: formValues(r), Next: safeNext(r.PostFormValue("next"))}
	if !s.reg.allow(s.clientIP(r)) {
		p.Summary = "Too many attempts. Wait a minute and try again. Nothing changed."
		s.render(w, r, nil, "register", 429, p)
		return
	}
	pw := r.PostFormValue("password")
	if pw != r.PostFormValue("password2") {
		p.Errors = []fieldErr{{"password2", "Confirm password: the two passwords do not match. Type the same password in both fields."}}
		p.Summary = "Not saved. Fix the problems below. Nothing changed."
		s.render(w, r, nil, "register", 422, p)
		return
	}
	// RegisterPerson answers identically for a new and a taken login (SA-10), so
	// this page must not sign in or branch on the outcome: it always sends the
	// person to the sign-in form, which works only with the right password.
	_, err := s.svc.RegisterPerson(r.Context(), service.RegisterInput{Login: r.PostFormValue("login"), Password: pw})
	if err == nil {
		s.setFlash(w, r, "Thank you. If that login was available the account now exists. Sign in with it and your password.")
		to := "/login"
		if p.Next != "" {
			to += "?next=" + url.QueryEscape(p.Next)
		}
		http.Redirect(w, r, to, http.StatusSeeOther)
		return
	}
	c := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
	if sum, fe, ok := c.refuse(err); ok {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "register", 422, p)
		return
	}
	s.fail(w, r, err)
}

func (s *server) setupForm(w http.ResponseWriter, r *http.Request, c *rc) {
	p := &page{Title: "Create a network", Dec: 2, V: map[string]string{"currency_name": "credit", "decimals": "2",
		"default_credit_limit": "0.00", "join_mode": "invite", "signup_grant": "0.00"}}
	p.D = s.opts.SetupSecret != ""
	s.render(w, r, nil, "setup", 200, p)
}

func (s *server) setupPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: "Create a network", V: v, Dec: 2, D: s.opts.SetupSecret != ""}
	fail := func(sum string, fe []fieldErr, st int) {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "setup", st, p)
	}
	if s.opts.SetupSecret != "" {
		// One secret creates one network, even across restarts and concurrent posts.
		s.setupMu.Lock()
		defer s.setupMu.Unlock()
		const usedMsg = "Not saved. The setup secret has already been used. Ask the person who runs this service to reset it. Nothing changed."
		if s.setupUsed.Load() {
			fail(usedMsg, nil, 403)
			return
		}
		used, err := s.svc.SetupSecretUsed(r.Context())
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if used {
			s.setupUsed.Store(true)
			fail(usedMsg, nil, 403)
			return
		}
		ip := s.clientIP(r)
		if s.setup.blocked(ip) {
			fail("Too many attempts. Wait and try again. Nothing changed.", nil, 429)
			return
		}
		if !secretEqual(r.PostFormValue("setup_secret"), s.opts.SetupSecret) {
			s.setup.fail(ip)
			fail("Not saved. Fix the problems below. Nothing changed.", []fieldErr{{"setup_secret", "Setup secret: not correct. Ask the person who runs this service."}}, 422)
			return
		}
	}
	dec, _ := strconv.Atoi(v["decimals"])
	pol := domain.DefaultPolicy()
	pol.CurrencyName, pol.Decimals, pol.JoinMode = v["currency_name"], dec, domain.JoinMode(v["join_mode"])
	var fe []fieldErr
	parse := func(field string, dst *int64) {
		n, err := domain.ParseAmount(v[field], dec)
		if err != nil {
			fe = append(fe, fieldErr{field, label(field) + ": enter a number with up to " + strconv.Itoa(dec) + " decimal places, for example " + domain.FormatAmount(1250, dec) + "."})
			return
		}
		*dst = n
	}
	parse("default_credit_limit", &pol.DefaultCreditLimit)
	parse("signup_grant", &pol.SignupGrant)
	if len(fe) > 0 {
		fail("Not saved. Fix the problems below. Nothing changed.", fe, 422)
		return
	}
	net, _, err := s.svc.CreateNetwork(r.Context(), c.actor, service.CreateNetworkInput{Name: v["name"], Slug: v["slug"],
		Policy: &pol, CreatorDisplayName: v["display_name"]})
	if err == nil {
		_, err = s.svc.ActivateNetwork(r.Context(), c.actor, net.ID)
	}
	if err != nil {
		cc := &rc{s: s, ctx: r.Context(), actor: c.actor, Net: &domain.Network{Policy: pol}}
		if sum, f, ok := cc.refuse(err); ok {
			if domain.CodeOf(err) == domain.CodeConflict {
				f = []fieldErr{{"slug", "Web address name: already in use. Choose a different one."}}
			}
			fail(sum, f, 422)
			return
		}
		s.fail(w, r, err)
		return
	}
	if s.opts.SetupSecret != "" {
		// FR-002, SEC-028: the secret is invalidated by its first use, durably.
		if err := s.svc.MarkSetupSecretUsed(r.Context(), c.actor); err != nil {
			s.fail(w, r, err)
			return
		}
		s.setupUsed.Store(true)
	}
	s.setFlash(w, r, "Network created and open. You are its first administrator.")
	http.Redirect(w, r, "/n/"+net.Slug, http.StatusSeeOther)
}

func (s *server) joinInviteForm(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	if _, _, ok := s.authenticate(r); !ok {
		s.render(w, r, nil, "join_anon", 200, &page{Title: "Accept your invitation", Next: "/join/" + tok})
		return
	}
	s.render(w, r, nil, "join", 200, &page{Title: "Accept your invitation", D: tok})
}

func (s *server) joinInvitePost(w http.ResponseWriter, r *http.Request, c *rc) {
	tok := r.PathValue("token")
	p := &page{Title: "Accept your invitation", D: tok, V: formValues(r)}
	m, err := s.svc.RedeemInvitation(r.Context(), c.actor, tok, r.PostFormValue("display_name"))
	if err != nil {
		cc := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
		switch domain.CodeOf(err) {
		case domain.CodeNotFound, domain.CodeInvalidState, domain.CodeForbidden:
			p.Summary = "This invitation is expired, already used or not valid. Ask the network administrator for a new invitation link. Nothing changed."
			s.render(w, r, nil, "join", 422, p)
		default:
			if sum, fe, ok := cc.refuse(err); ok {
				p.Summary, p.Errors = sum, fe
				s.render(w, r, nil, "join", 422, p)
				return
			}
			s.fail(w, r, err)
		}
		return
	}
	s.afterJoin(w, r, c, m)
}

func (s *server) afterJoin(w http.ResponseWriter, r *http.Request, c *rc, m *domain.Member) {
	n, err := s.svc.GetNetwork(r.Context(), c.actor, m.NetworkID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if m.Status == domain.MemberPending {
		s.setFlash(w, r, "Request sent. Your membership is pending until an administrator decides.")
	} else {
		s.setFlash(w, r, "You have joined "+n.Name+".")
	}
	http.Redirect(w, r, "/n/"+n.Slug, http.StatusSeeOther) //nolint:gosec // G710: fixed local prefix plus validated slug
}

func (s *server) joinNetForm(w http.ResponseWriter, r *http.Request, c *rc) {
	info, err := s.svc.NetworkBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "joinnet", 200, &page{Title: "Join " + info.Name, D: info})
}

func (s *server) joinNetPost(w http.ResponseWriter, r *http.Request, c *rc) {
	info, err := s.svc.NetworkBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	m, err := s.svc.JoinNetwork(r.Context(), c.actor, info.ID, r.PostFormValue("display_name"))
	if err != nil {
		p := &page{Title: "Join " + info.Name, D: info, V: formValues(r)}
		cc := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
		if sum, fe, ok := cc.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			s.render(w, r, nil, "joinnet", 422, p)
			return
		}
		if domain.CodeOf(err) == domain.CodeForbidden {
			p.Summary = "Not sent. This network does not accept requests to join. Ask an administrator for an invitation link. Nothing changed."
			s.render(w, r, nil, "joinnet", 403, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.afterJoin(w, r, c, m)
}

func (s *server) help(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "help", 200, &page{Title: "Help: credits, balances and statuses"})
}

// secretEqual compares two secrets in constant time regardless of length.
func secretEqual(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
