package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/terms", s.netH(s.termsPage))
		s.mux.HandleFunc("POST /n/{slug}/terms", s.netH(s.termsAccept))
		s.mux.HandleFunc("GET /n/{slug}/policy", s.netH(s.rulesPage))
		s.mux.HandleFunc("GET /n/{slug}/leave", s.netH(s.leaveForm))
		s.mux.HandleFunc("POST /n/{slug}/leave", s.netH(s.leavePost))
		s.mux.HandleFunc("GET /n/{slug}/admin/terms", s.adminH(s.adminTermsForm))
		s.mux.HandleFunc("POST /n/{slug}/admin/terms", s.adminH(s.adminTermsPost))
		s.mux.HandleFunc("POST /n/{slug}/admin/policy/cancel", s.adminH(s.policyCancel))
	})
}

// paragraphs splits free text into paragraphs on blank lines, for display.
func paragraphs(s string) []string {
	var out []string
	for _, p := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

type termsData struct {
	Terms           *domain.Terms
	Paras           []string
	Accepted        int
	NeedsAcceptance bool
	Next            string
}

func (s *server) termsPage(w http.ResponseWriter, r *http.Request, c *rc) {
	v, err := s.svc.CurrentTerms(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &termsData{Terms: v.Current, Accepted: v.Accepted, NeedsAcceptance: v.NeedsAcceptance() && c.Me.Status == domain.MemberActive,
		Next: safeNext(r.URL.Query().Get("next"))}
	if v.Current != nil {
		d.Paras = paragraphs(v.Current.Body)
	}
	s.render(w, r, c, "terms", 200, &page{Title: "Network rules and terms", D: d})
}

func (s *server) termsAccept(w http.ResponseWriter, r *http.Request, c *rc) {
	ver, _ := strconv.Atoi(r.PostFormValue("version"))
	if r.PostFormValue("accept") != "1" {
		v, err := s.svc.CurrentTerms(r.Context(), c.actor, c.netID())
		if err != nil {
			s.fail(w, r, err)
			return
		}
		d := &termsData{Terms: v.Current, Accepted: v.Accepted, NeedsAcceptance: true}
		if v.Current != nil {
			d.Paras = paragraphs(v.Current.Body)
		}
		s.render(w, r, c, "terms", 422, &page{Title: "Network rules and terms", D: d,
			Summary: "Not saved. Tick the box to say you accept these terms. Nothing changed.",
			Errors:  []fieldErr{{"accept", "Accept: tick the box to accept these terms."}}})
		return
	}
	if err := s.svc.AcceptTerms(r.Context(), c.actor, c.netID(), ver); err != nil {
		if domain.CodeOf(err) == domain.CodeValidation || domain.CodeOf(err) == domain.CodeInvalidState {
			s.setFlash(w, r, "These terms changed while you were reading. Please read the new version.")
			http.Redirect(w, r, c.base()+"/terms", http.StatusSeeOther)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "You accepted version "+strconv.Itoa(ver)+" of the network terms.")
	dest := c.base() + "/trades"
	if n := safeNext(r.PostFormValue("next")); n != "" {
		dest = n
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// ---- change log (GOV-002, GOV-003) ----

type ruleRow struct {
	Version                  int
	Lines                    []string
	Reason, NoticeReason     string
	Announced, Effective     any
	NoticeDays               int
	Pending, Withdrawn, Init bool
}

func fmtChangeValue(key string, v any, dec int) string {
	switch key {
	case "default_credit_limit", "max_tx_amount", "signup_grant":
		if n, ok := v.(int64); ok {
			return domain.FormatAmount(n, dec)
		}
	}
	return fmtAny(v)
}

func fmtAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		if x {
			return "yes"
		}
		return "no"
	}
	return ""
}

func ruleRows(log []service.PolicyLogEntry, dec int) []ruleRow {
	out := make([]ruleRow, 0, len(log))
	for _, e := range log {
		row := ruleRow{Version: e.Version, Reason: e.Reason, NoticeReason: e.NoticeReason, NoticeDays: e.NoticeDays,
			Announced: fmtTime(e.AnnouncedAt), Effective: fmtTime(e.EffectiveAt), Pending: e.Pending, Withdrawn: e.Withdrawn,
			Init: e.Previous == nil && e.Version == 1}
		keys := make([]string, 0, len(e.Changes))
		for k := range e.Changes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if prev, ok := e.Previous[k]; ok {
				row.Lines = append(row.Lines, label(k)+": "+fmtChangeValue(k, prev, dec)+" to "+fmtChangeValue(k, e.Changes[k], dec))
			} else {
				row.Lines = append(row.Lines, label(k)+": "+fmtChangeValue(k, e.Changes[k], dec))
			}
		}
		out = append(out, row)
	}
	return out
}

type rulesData struct {
	Rows     []ruleRow
	Terms    *domain.Terms
	CanLeave bool
}

func (s *server) rulesPage(w http.ResponseWriter, r *http.Request, c *rc) {
	log, err := s.svc.PolicyLog(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	tv, err := s.svc.CurrentTerms(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "rules", 200, &page{Title: "Network rules", D: &rulesData{Rows: ruleRows(log, c.Net.Policy.Decimals), Terms: tv.Current,
		CanLeave: c.Me.Status == domain.MemberActive || c.Me.Status == domain.MemberSuspended}})
}

// ---- administrator: terms and scheduled policy changes ----

func (s *server) adminTermsForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.adminTermsRender(w, r, c, 200, &page{})
}

func (s *server) adminTermsRender(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	v, err := s.svc.CurrentTerms(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p.Title = "Network terms"
	d := &termsData{Terms: v.Current}
	if v.Current != nil {
		d.Paras = paragraphs(v.Current.Body)
		if p.V == nil {
			p.V = map[string]string{"body": v.Current.Body}
		}
	}
	p.D = d
	s.render(w, r, c, "admin_terms", status, p)
}

func (s *server) adminTermsPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	t, err := s.svc.PublishTerms(r.Context(), c.actor, c.netID(), v["body"])
	if err != nil {
		sum, fe, ok := c.refuse(err)
		if !ok {
			s.fail(w, r, err)
			return
		}
		s.adminTermsRender(w, r, c, 422, &page{V: v, Summary: sum, Errors: fe})
		return
	}
	s.setFlash(w, r, "Published version "+strconv.Itoa(t.Version)+". Members were told and must accept it before their next trade.")
	http.Redirect(w, r, c.base()+"/admin/terms", http.StatusSeeOther)
}

func (s *server) policyCancel(w http.ResponseWriter, r *http.Request, c *rc) {
	reason := strings.TrimSpace(r.PostFormValue("reason"))
	if reason == "" {
		reason = "Scheduled change cancelled"
	}
	_, err := s.svc.UpdatePolicy(r.Context(), c.actor, c.netID(), service.PolicyChange{Policy: c.Net.Policy, Reason: reason, IdempotencyKey: idemKey(r)})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "The scheduled change was cancelled. The rules in force stay as they are.")
	http.Redirect(w, r, c.base()+"/admin/policy", http.StatusSeeOther)
}

// ---- leaving the network (FR-014, GOV-010) ----

func (s *server) leaveForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.leaveRender(w, r, c, 200, &page{})
}

func (s *server) leaveRender(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	pv, err := s.svc.PreviewDeparture(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	p.Title = "Leave this network"
	p.D = pv
	s.render(w, r, c, "leave", status, p)
}

func (s *server) leavePost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	if v["confirm"] != "1" {
		s.leaveRender(w, r, c, 422, &page{V: v, Summary: "Not done. Tick the box to confirm that you want to leave. Nothing changed.",
			Errors: []fieldErr{{"confirm", "Confirm: tick the box to leave this network."}}})
		return
	}
	_, err := s.svc.DepartMember(r.Context(), c.actor, c.netID(), service.DepartInput{Reason: strings.TrimSpace(v["reason"])})
	if err != nil {
		sum, fe, ok := c.refuse(err)
		if !ok {
			s.fail(w, r, err)
			return
		}
		s.leaveRender(w, r, c, http.StatusConflict, &page{V: v, Summary: sum, Errors: fe})
		return
	}
	s.setFlash(w, r, "You have left "+c.Net.Name+". You can still read your history and download your data from the History page.")
	http.Redirect(w, r, c.base()+"/history", http.StatusSeeOther)
}
