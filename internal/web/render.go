package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

type pageTemplate = template.Template

type fieldErr struct{ Field, Text string }

type navItem struct {
	Label, Href string
	Current     bool
}

// page is the data every template receives.
type page struct {
	Title   string // screen name; also the h1
	NetName string
	Slug    string
	Authed  bool
	Nav     []navItem
	CSRF    string
	Flash   string
	ReqID   string
	Notice  string // persistent notice (suspended, departed)
	Summary string // error summary lead text
	Errors  []fieldErr
	V       map[string]string // submitted form values
	D       any               // screen data
	Dec     int
	Unit    string
	Idem    string
	Next    string
	Status  int
	Text    string
	// Session expiry warning (UX-030): Unix time and whether activity extends it.
	ExpiresAt  int64
	Extendable bool
	// Unread is the inbox badge count (FR-038); 0 shows no badge.
	Unread int
	// Warn is the credit-limit notice shown on Home and the propose form (UX-012).
	Warn string
	// SecUnseen counts unacknowledged security notices (SEC-012), shown in the header.
	SecUnseen int
	// FrozenSince is separate from notice text so its time element stays markup.
	FrozenSince template.HTML
	// SnapshotAt labels network figures even when scripting or connectivity fails (PER-022).
	SnapshotAt template.HTML

	displayZone *time.Location
}

// Time renders a stored instant in the viewer's display zone.
func (p *page) Time(t time.Time) template.HTML {
	return fmtTime(p.localTime(t))
}

// SessionExpiryText shares the catalogue formatter and the viewer's selected
// display zone with ordinary page timestamps. The Unix expiry still drives timers.
func (p *page) SessionExpiryText() string {
	return messages.DateTime(p.localTime(time.Unix(p.ExpiresAt, 0)))
}

// ISODate preserves date-only precision for membership join dates.
func (p *page) ISODate(t time.Time) string {
	return p.localTime(t).Format(time.DateOnly)
}

func (p *page) localTime(t time.Time) time.Time {
	if p.displayZone == nil {
		return t.UTC()
	}
	return t.In(p.displayZone)
}

func (p *page) Val(k string) string { return p.V[k] }

// Err returns the error text for a field, or "".
func (p *page) Err(k string) string {
	for _, e := range p.Errors {
		if e.Field == k {
			return e.Text
		}
	}
	return ""
}

func mustLoadTemplates() map[string]*pageTemplate {
	funcs := template.FuncMap{
		"msg":                 messages.Text,
		"msgAround":           messages.Around,
		"msgAroundCount":      messages.AroundCount,
		"msgAroundTwo":        messages.AroundTwo,
		"msgAroundTwoVersion": messages.AroundTwoVersion,
		"msgAroundTwoCount":   messages.AroundTwoCount,
		"amt":                 func(v int64, dec int) string { return minus(domain.FormatAmount(v, dec)) },
		"plain":               func(v int64, dec int) string { return domain.FormatAmount(abs64(v), dec) },
		"unit":                unitFor,
		"time":                fmtTime,
		"upper":               strings.ToUpper,
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				m[fmt.Sprint(kv[i])] = kv[i+1]
			}
			return m
		},
		"eq2": func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) },
		// No query-escape helper: html/template already escapes a value placed in
		// an href query, and a second escape corrupts return links ("next").
		"categoryLabel": categoryLabel,
		"deref": func(v any) any {
			switch x := v.(type) {
			case *int64:
				return *x
			case *time.Time:
				return *x
			}
			return v
		},
	}
	base, err := fs.ReadFile(assets, "templates/base.html")
	if err != nil {
		panic(err)
	}
	files, _ := fs.Glob(assets, "templates/*.html")
	out := map[string]*pageTemplate{}
	for _, f := range files {
		name := strings.TrimSuffix(path.Base(f), ".html")
		if name == "base" {
			continue
		}
		src, _ := fs.ReadFile(assets, f)
		t := template.New(name).Funcs(funcs)
		template.Must(t.Parse(string(base)))
		template.Must(t.Parse(string(src)))
		if name == "history" {
			template.Must(t.Parse(histBalanceSrc))
		}
		out[name] = t
	}
	return out
}

func minus(s string) string { return strings.Replace(s, "-", "−", 1) }

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func fmtTime(t time.Time) template.HTML {
	if t.IsZero() {
		return ""
	}
	// The text uses t's own zone (UTC unless the handler moved it with rc.at); the machine-readable
	// attribute is always UTC.
	//nolint:gosec // the attribute is UTC RFC3339 and the catalogue text is HTML-escaped.
	return template.HTML(fmt.Sprintf(`<time datetime="%s">%s</time>`, t.UTC().Truncate(time.Millisecond).Format(time.RFC3339Nano), template.HTMLEscapeString(messages.DateTime(t))))
}

// at moves t into the viewer's display time zone (UTC when none is set).
func (c *rc) at(t time.Time) time.Time {
	if c.loc == nil {
		return t.UTC()
	}
	return t.In(c.loc)
}

// when formats a time in the viewer's zone.
func (c *rc) when(t time.Time) template.HTML { return fmtTime(c.at(t)) }

// row builds a transaction row with its time in the viewer's zone.
func (c *rc) row(t *domain.Transaction, n names) txRow {
	r := makeRow(t, c.Me, n, c.Net.Policy.CurrencyName, c.Net.Policy.Decimals)
	r.When = c.at(r.When)
	return r
}

// render writes a page. c may be nil (public pages).
func (s *server) render(w http.ResponseWriter, r *http.Request, c *rc, name string, status int, p *page) {
	t := s.pages[name]
	if t == nil {
		s.log.Error("web template missing", "request_id", reqID(r), "template", name)
		http.Error(w, messages.Text("error.fallback.internal"), 500)
		return
	}
	p.CSRF = s.csrfToken(r)
	p.ReqID = reqID(r)
	p.Flash = s.takeFlash(w, r)
	p.Idem = randHex(16)
	if c != nil && c.Net != nil {
		p.Authed = true
		p.NetName, p.Slug = c.Net.Name, c.Net.Slug
		p.Dec, p.Unit = c.Net.Policy.Decimals, c.Net.Policy.CurrencyName
		p.Nav = c.nav(r.URL.Path)
		p.Notice = c.notice()
		if fi, ok := c.s.svc.FrozenInfo(c.Net.ID); ok {
			p.FrozenSince = c.when(fi.Since)
		}
		p.Unread = c.unreadCount()
		p.SnapshotAt = c.when(s.opts.Clock.Now())
	} else {
		if c != nil {
			c.loadDisplayZone()
		}
		if p.Dec == 0 {
			p.Dec = 2
		}
		p.Authed = s.opts.Edge.SessionToken(r) != "" || p.Authed
	}
	if c != nil {
		p.displayZone = c.loc
	}
	if p.Authed {
		if at, ext, ok := s.expiry(r); ok {
			p.ExpiresAt, p.Extendable = at, ext
		}
		s.secBanner(r, p)
	}
	if p.V == nil {
		p.V = map[string]string{}
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", p); err != nil {
		s.log.Error("web render failed", "request_id", p.ReqID, "template", name, "err", err)
		http.Error(w, messages.Text("error.fallback.reference", p.ReqID), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (c *rc) base() string { return "/n/" + c.Net.Slug }

func (c *rc) nav(cur string) []navItem {
	b := c.base()
	items := []navItem{{Label: messages.Text("navigation.home"), Href: b}}
	switch c.Me.Status {
	case domain.MemberActive:
		items = append(items, navItem{Label: messages.Text("navigation.trades"), Href: b + "/trades"},
			navItem{Label: messages.Text("navigation.directory"), Href: b + "/directory"}, navItem{Label: messages.Text("navigation.listings"), Href: b + "/listings"},
			navItem{Label: messages.Text("navigation.history"), Href: b + "/history"}, navItem{Label: messages.Text("navigation.disputes"), Href: b + "/disputes"},
			navItem{Label: messages.Text("navigation.rules"), Href: b + "/policy"})
	case domain.MemberSuspended, domain.MemberDeparted:
		// Me is resolved by GetNetworkAccess for this request. These members
		// retain their own records and rules even though they cannot trade.
		items = append(items, navItem{Label: messages.Text("navigation.trades"), Href: b + "/trades"}, navItem{Label: messages.Text("navigation.history"), Href: b + "/history"},
			navItem{Label: messages.Text("navigation.disputes"), Href: b + "/disputes"}, navItem{Label: messages.Text("navigation.rules"), Href: b + "/policy"})
	default:
		items = append(items, navItem{Label: messages.Text("navigation.history"), Href: b + "/history"})
	}
	if c.isReviewer() || c.isAdmin() {
		items = append(items, navItem{Label: messages.Text("navigation.review"), Href: b + "/review"})
	}
	if c.isAdmin() {
		items = append(items, navItem{Label: messages.Text("navigation.admin"), Href: b + "/admin"})
	}
	for i := range items {
		it := &items[i]
		it.Current = cur == it.Href || (it.Href != b && strings.HasPrefix(cur, it.Href+"/"))
	}
	return items
}

func (c *rc) notice() string {
	switch c.Me.Status {
	case domain.MemberSuspended:
		return messages.Text("notice.membership_suspended")
	case domain.MemberDeparted:
		return messages.Text("notice.membership_departed")
	}
	if c.Net.Status == domain.NetworkClosed {
		return messages.Text("notice.network_closed")
	}
	if c.Net.Status == domain.NetworkClosing {
		return messages.Text("notice.network_closing")
	}
	if c.Net.AdminRequired {
		return messages.Text("notice.admin_required")
	}
	if cl := c.Net.Closure; cl != nil && !cl.Done() {
		return messages.Text("notice.closing_date", messages.Date(c.at(cl.ClosingAt)))
	}
	if c.Net.Suspended() {
		return messages.Text("notice.trading_paused")
	}
	return ""
}

// renderError writes a generic error page. status errCSRF means a rejected form token.
func (s *server) renderError(w http.ResponseWriter, r *http.Request, status int) {
	p := &page{Status: status}
	st := status
	switch status {
	case errCSRF:
		st = 403
		p.Title, p.Text = messages.Text("error.csrf.title"), messages.Text("error.csrf.body")
	case 403:
		p.Title, p.Text = messages.Text("error.forbidden.title"), messages.Text("error.forbidden.body")
	case 404:
		p.Title, p.Text = messages.Text("error.not_found.title"), messages.Text("error.not_found.body")
	case 422:
		p.Title, p.Text = messages.Text("error.invalid.title"), messages.Text("error.invalid.body")
	case 429:
		p.Title, p.Text = messages.Text("error.rate_limited.title"), messages.Text("error.rate_limited.body")
	case 503:
		w.Header().Set("Retry-After", "5")
		p.Title, p.Text = messages.Text("error.busy.title"), messages.Text("error.busy.body")
	case errFrozen:
		st = 423
		p.Title, p.Text = messages.Text("error.frozen.title"), messages.Text("error.frozen.body")
	case 409:
		p.Title, p.Text = messages.Text("error.conflict.title"), messages.Text("error.conflict.body")
	default:
		st = 500
		p.Title, p.Text = messages.Text("error.internal.title"), messages.Text("error.internal.body")
	}
	p.Status = st
	s.render(w, r, nil, "error", st, p)
}

// fail maps an unexpected service error to a page-level response.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if httpx.IsBusy(err) || domain.CodeOf(err) == domain.CodeUnavailable { // busy resource: retryable, not a fault (OPS-011)
		s.renderError(w, r, 503)
		return
	}
	switch domain.CodeOf(err) {
	case domain.CodeNetworkFrozen:
		s.renderError(w, r, errFrozen)
	case domain.CodeUnauthenticated:
		s.clearSession(w, r)
		s.toLogin(w, r)
	case domain.CodePasswordChangeRequired:
		s.mustChange(w, r)
	case domain.CodeTermsRequired: // GOV-009: show the terms, then come back
		if slug := r.PathValue("slug"); slug != "" {
			s.setFlash(w, r, messages.Text("notice.terms_changed"))
			http.Redirect(w, r, "/n/"+url.PathEscape(slug)+"/terms?next="+url.QueryEscape("/n/"+slug+"/trades"), http.StatusSeeOther)
			return
		}
		s.renderError(w, r, 409)
	case domain.CodeForbidden, domain.CodeApprovalRequired:
		s.renderError(w, r, 403)
	case domain.CodeNotFound:
		s.renderError(w, r, 404)
	case domain.CodeValidation:
		s.renderError(w, r, 422)
	case domain.CodeRateLimited:
		s.renderError(w, r, 429)
	case domain.CodeInvalidState, domain.CodeConflict, domain.CodeBlocked:
		s.renderError(w, r, 409)
	default:
		if domain.CodeOf(err) == domain.CodeInternal {
			// The route pattern, never the raw path; logx escapes and redacts the error text.
			s.log.Error("web internal error", "request_id", reqID(r), "route", r.Pattern, "err", err)
		}
		s.renderError(w, r, 500)
	}
}

func (s *server) notFoundOrRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.renderError(w, r, 404)
		return
	}
	s.home(w, r)
}
