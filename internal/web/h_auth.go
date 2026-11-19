package web

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strconv"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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
	s.render(w, r, nil, "login", 200, &page{Title: messages.Text("auth.title.sign_in"), Next: safeNext(r.URL.Query().Get("next"))})
}

func (s *server) loginPost(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: messages.Text("auth.title.sign_in"), V: formValues(r), Next: safeNext(r.PostFormValue("next"))}
	ip, acct := s.clientIP(r), acctKey(s.clientIP(r), r.PostFormValue("login"))
	if s.ipFail.blocked(ip) || s.acctFail.blocked(acct) {
		p.Summary = messages.Text("auth.throttle.minute")
		s.render(w, r, nil, "login", 429, p)
		return
	}
	res, err := s.svc.LoginStep(service.WithClientIP(r.Context(), s.clientIP(r)), r.PostFormValue("login"), r.PostFormValue("password"))
	if err != nil {
		if c := domain.CodeOf(err); c == domain.CodeValidation || c == domain.CodeUnauthenticated || c == domain.CodeRateLimited {
			s.ipFail.fail(ip)
			s.acctFail.fail(acct)
		}
		switch domain.CodeOf(err) {
		case domain.CodeRateLimited:
			p.Summary = messages.Text("auth.throttle.login")
			s.render(w, r, nil, "login", 429, p)
		case domain.CodeValidation, domain.CodeUnauthenticated:
			// One message for unknown login and wrong password (UX-052).
			p.Summary = messages.Text("auth.login.failed")
			p.Errors = []fieldErr{{"login", messages.Text("auth.login.failed_field")}}
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
		if err := s.svc.Logout(r.Context(), tok); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	s.clearSession(w, r)
	s.rotateCSRF(w, r)
	s.setFlash(w, r, messages.Text("auth.flash.signed_out"))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *server) registerForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "register", 200, &page{Title: messages.Text("auth.title.register"), Next: safeNext(r.URL.Query().Get("next"))})
}

func (s *server) registerPost(w http.ResponseWriter, r *http.Request) {
	p := &page{Title: messages.Text("auth.title.register"), V: formValues(r), Next: safeNext(r.PostFormValue("next"))}
	if !s.reg.allow(s.clientIP(r)) {
		p.Summary = messages.Text("auth.throttle.minute")
		s.render(w, r, nil, "register", 429, p)
		return
	}
	pw := r.PostFormValue("password")
	if pw != r.PostFormValue("password2") {
		p.Errors = []fieldErr{{"password2", messages.Text("auth.mismatch.confirm")}}
		p.Summary = messages.Text("auth.not_saved")
		s.render(w, r, nil, "register", 422, p)
		return
	}
	// RegisterPerson answers identically for a new and a taken login (SA-10), so
	// this page must not sign in or branch on the outcome: it always sends the
	// person to the sign-in form, which works only with the right password.
	_, err := s.svc.RegisterPerson(r.Context(), service.RegisterInput{Login: r.PostFormValue("login"), Password: pw})
	if err == nil {
		s.setFlash(w, r, messages.Text("auth.flash.registered"))
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
	p := &page{Title: messages.Text("setup.title"), Dec: 2, V: map[string]string{"currency_name": "credit", "decimals": "2",
		"default_credit_limit": "0.00", "join_mode": "invite", "signup_grant": "0.00"}}
	p.D = s.opts.SetupSecret != ""
	s.render(w, r, nil, "setup", 200, p)
}

func (s *server) setupPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: messages.Text("setup.title"), V: v, Dec: 2, D: s.opts.SetupSecret != ""}
	fail := func(sum string, fe []fieldErr, st int) {
		p.Summary, p.Errors = sum, fe
		s.render(w, r, nil, "setup", st, p)
	}
	if s.opts.SetupSecret != "" {
		// One secret creates one network, even across restarts and concurrent posts.
		s.setupMu.Lock()
		defer s.setupMu.Unlock()
		usedMsg := messages.Text("setup.secret.used")
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
			fail(messages.Text("setup.throttle"), nil, 429)
			return
		}
		if !secretEqual(r.PostFormValue("setup_secret"), s.opts.SetupSecret) {
			s.setup.fail(ip)
			fail(messages.Text("setup.invalid"), []fieldErr{{"setup_secret", messages.Text("setup.secret.invalid")}}, 422)
			return
		}
	}
	// Validate the scale before using it in an amount formatter. ParseAmount
	// rejects invalid scales, but formatting its error example would otherwise
	// allocate according to an attacker-controlled number of decimal places.
	dec, err := strconv.Atoi(v["decimals"])
	if err != nil || dec < 0 || dec > domain.MaxDecimals {
		fail(messages.Text("setup.invalid"), []fieldErr{{"decimals", messages.Text("setup.decimals.invalid")}}, http.StatusUnprocessableEntity)
		return
	}
	pol := domain.DefaultPolicy()
	pol.CurrencyName, pol.Decimals, pol.JoinMode = v["currency_name"], dec, domain.JoinMode(v["join_mode"])
	var fe []fieldErr
	parse := func(field, labelKey string, dst *int64) {
		n, err := domain.ParseAmount(v[field], dec)
		if err != nil {
			fe = append(fe, fieldErr{field, messages.Text("setup.amount.invalid", messages.Text(labelKey), dec, domain.FormatAmount(1250, dec))})
			return
		}
		*dst = n
	}
	parse("default_credit_limit", "setup.limit.error_label", &pol.DefaultCreditLimit)
	parse("signup_grant", "setup.grant.label", &pol.SignupGrant)
	if len(fe) > 0 {
		fail(messages.Text("setup.invalid"), fe, 422)
		return
	}
	net, _, err := s.svc.CreateNetwork(r.Context(), c.actor, service.CreateNetworkInput{Name: v["name"], Slug: v["slug"],
		Policy: &pol, CreatorDisplayName: v["display_name"],
		ConsumeSetupSecret:   s.opts.SetupSecret != "",             // SA2-16: spent in the same transaction
		ConfirmOpenGrantRisk: v["confirm_open_grant_risk"] == "1"}) // SEC-032
	if err == nil {
		_, err = s.svc.ActivateNetwork(r.Context(), c.actor, net.ID)
	}
	if err != nil {
		cc := &rc{s: s, ctx: r.Context(), actor: c.actor, Net: &domain.Network{Policy: pol}}
		if sum, f, ok := cc.refuse(err); ok {
			if domain.CodeOf(err) == domain.CodeConflict {
				f = []fieldErr{{"slug", messages.Text("setup.slug.unavailable")}}
			}
			fail(sum, f, 422)
			return
		}
		s.fail(w, r, err)
		return
	}
	if s.opts.SetupSecret != "" {
		s.setupUsed.Store(true) // FR-002, SEC-028: spent durably by CreateNetwork
	}
	s.setFlash(w, r, messages.Text("setup.flash.created"))
	http.Redirect(w, r, "/n/"+net.Slug, http.StatusSeeOther)
}

func (s *server) joinInviteForm(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	a, _, ok := s.authenticate(r)
	if !ok {
		s.render(w, r, nil, "join_anon", 200, &page{Title: messages.Text("join.invitation.title"), Next: "/join/" + tok})
		return
	}
	if a.MustChangePassword { // Anonymous invitation information stays public.
		s.mustChange(w, r)
		return
	}
	s.render(w, r, nil, "join", 200, &page{Title: messages.Text("join.invitation.title"), D: s.inviteData(r, tok)})
}

// joinData is the join pages' data: the invitation token or network, and the terms to show (GOV-009).
type joinData struct {
	Token string
	Terms *domain.Terms
	Paras []string
}

func newJoinData(tok string, t *domain.Terms) *joinData {
	d := &joinData{Token: tok, Terms: t}
	if t != nil {
		d.Paras = paragraphs(t.Body)
	}
	return d
}

func (s *server) inviteData(r *http.Request, tok string) *joinData {
	t, _ := s.svc.InvitationTerms(r.Context(), tok) // unusable tokens show no terms; the post explains
	return newJoinData(tok, t)
}

// joinNetData adds the terms to the network description.
type joinNetData struct {
	*service.NetworkInfo
	*joinData
}

func (s *server) netJoinData(r *http.Request, info *service.NetworkInfo) *joinNetData {
	t, _ := s.svc.NetworkTerms(r.Context(), info.ID)
	return &joinNetData{info, newJoinData("", t)}
}

// termsVersion is the version the person ticked, 0 when the box is not ticked.
func termsVersion(r *http.Request) int {
	if r.PostFormValue("accept_terms") != "1" {
		return 0
	}
	n, _ := strconv.Atoi(r.PostFormValue("terms_version"))
	return n
}

func (s *server) joinInvitePost(w http.ResponseWriter, r *http.Request, c *rc) {
	tok := r.PathValue("token")
	p := &page{Title: messages.Text("join.invitation.title"), D: s.inviteData(r, tok), V: formValues(r)}
	m, err := s.svc.RedeemInvitationTerms(r.Context(), c.actor, tok, r.PostFormValue("display_name"), termsVersion(r))
	if err != nil {
		if s.networkFull(w, r, err, "join", p) {
			return
		}
		cc := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
		switch domain.CodeOf(err) {
		case domain.CodeNotFound, domain.CodeInvalidState, domain.CodeForbidden:
			p.Summary = messages.Text("join.invitation.invalid")
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

// networkFull renders the per-network membership limit refusal (SEC-006,
// network-join-limit) as a throttled response: 429 with Retry-After, not the
// generic validation refusal, whose sign-in wording ("wait a minute") would be
// wrong for a window of hours. It reports whether err was that refusal; the
// join services return rate-limited for nothing else.
func (s *server) networkFull(w http.ResponseWriter, r *http.Request, err error, tpl string, p *page) bool {
	if domain.CodeOf(err) != domain.CodeRateLimited {
		return false
	}
	secs := int(s.svc.NetworkJoinWindow().Seconds())
	if secs < 1 {
		secs = 60
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	p.Summary = messages.Text("join.network.full")
	s.render(w, r, nil, tpl, http.StatusTooManyRequests, p)
	return true
}

func (s *server) afterJoin(w http.ResponseWriter, r *http.Request, c *rc, m *domain.Member) {
	n, err := s.svc.GetNetwork(r.Context(), c.actor, m.NetworkID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if m.Status == domain.MemberPending {
		s.setFlash(w, r, messages.Text("join.flash.pending"))
	} else {
		s.setFlash(w, r, messages.Text("join.flash.joined", n.Name))
	}
	http.Redirect(w, r, "/n/"+n.Slug, http.StatusSeeOther) //nolint:gosec // G710: fixed local prefix plus validated slug
}

// networkForJoin keeps private network names behind the same membership read
// gate as normal network pages. Invitation links have their own token flow.
func (s *server) networkForJoin(r *http.Request, c *rc) (*service.NetworkInfo, error) {
	info, err := s.svc.NetworkBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		return nil, err
	}
	if info.JoinMode == domain.JoinInvite {
		if _, err := s.svc.GetNetwork(r.Context(), c.actor, info.ID); err != nil {
			return nil, err
		}
	}
	return info, nil
}

func (s *server) joinNetForm(w http.ResponseWriter, r *http.Request, c *rc) {
	info, err := s.networkForJoin(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, nil, "joinnet", 200, &page{Title: messages.Text("join.network.title", info.Name), D: s.netJoinData(r, info)})
}

func (s *server) joinNetPost(w http.ResponseWriter, r *http.Request, c *rc) {
	info, err := s.networkForJoin(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// FR-012: join attempts are rate-limited per person and per address.
	if !s.join.allow("p:"+c.actor.PersonID) || !s.join.allow("i:"+s.clientIP(r)) {
		w.Header().Set("Retry-After", strconv.Itoa(int(s.join.window.Seconds())))
		summary := messages.Text("join.throttle.ten_minutes")
		if s.join.window.Minutes() != 10 {
			summary = messages.Text("join.throttle.other")
		}
		p := &page{Title: messages.Text("join.network.title", info.Name), D: s.netJoinData(r, info), V: formValues(r),
			Summary: summary}
		s.render(w, r, nil, "joinnet", 429, p)
		return
	}
	m, err := s.svc.JoinNetworkTerms(r.Context(), c.actor, info.ID, r.PostFormValue("display_name"), termsVersion(r))
	if err != nil {
		p := &page{Title: messages.Text("join.network.title", info.Name), D: s.netJoinData(r, info), V: formValues(r)}
		if s.networkFull(w, r, err, "joinnet", p) {
			return
		}
		cc := &rc{s: s, ctx: r.Context(), Net: &domain.Network{}}
		if sum, fe, ok := cc.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			s.render(w, r, nil, "joinnet", 422, p)
			return
		}
		if domain.CodeOf(err) == domain.CodeForbidden {
			p.Summary = messages.Text("join.network.closed")
			s.render(w, r, nil, "joinnet", 403, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.afterJoin(w, r, c, m)
}

func (s *server) help(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, nil, "help", 200, &page{Title: messages.Text("help.title")})
}

// secretEqual compares two secrets in constant time regardless of length.
func secretEqual(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
