package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/httpx"
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
		"amt":   func(v int64, dec int) string { return minus(domain.FormatAmount(v, dec)) },
		"plain": func(v int64, dec int) string { return domain.FormatAmount(abs64(v), dec) },
		"time":  fmtTime,
		"upper": strings.ToUpper,
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				m[fmt.Sprint(kv[i])] = kv[i+1]
			}
			return m
		},
		"eq2":           func(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) },
		"qs":            url.QueryEscape,
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
	u := t.UTC()
	//nolint:gosec // both values are formatted times, no user input
	return template.HTML(fmt.Sprintf(`<time datetime="%s">%s</time>`, u.Format(time.RFC3339), u.Format("2 Jan 2006, 15:04 UTC")))
}

// render writes a page. c may be nil (public pages).
func (s *server) render(w http.ResponseWriter, r *http.Request, c *rc, name string, status int, p *page) {
	t := s.pages[name]
	if t == nil {
		log.Printf("web: missing template %s", name)
		http.Error(w, "internal error", 500)
		return
	}
	p.CSRF = s.csrfToken(r)
	p.ReqID = reqID(r)
	p.Flash = s.takeFlash(w, r)
	p.Idem = randHex(16)
	if c == nil {
		if p.Dec == 0 {
			p.Dec = 2
		}
		if httpx.SessionToken(r) != "" {
			p.Authed = true
		}
	}
	if c != nil {
		p.Authed = true
		p.NetName, p.Slug = c.Net.Name, c.Net.Slug
		p.Dec, p.Unit = c.Net.Policy.Decimals, c.Net.Policy.CurrencyName
		p.Nav = c.nav(r.URL.Path)
		p.Notice = c.notice()
	}
	if p.Authed {
		if at, ext, ok := s.expiry(r); ok {
			p.ExpiresAt, p.Extendable = at, ext
		}
	}
	if p.V == nil {
		p.V = map[string]string{}
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", p); err != nil {
		log.Printf("web: render %s req=%s: %v", name, p.ReqID, err)
		http.Error(w, "internal error, reference "+p.ReqID, 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func (c *rc) base() string { return "/n/" + c.Net.Slug }

func (c *rc) nav(cur string) []navItem {
	b := c.base()
	items := []navItem{{Label: "Home", Href: b}}
	switch c.Me.Status {
	case domain.MemberActive:
		items = append(items, navItem{Label: "Trades", Href: b + "/trades"},
			navItem{Label: "Directory", Href: b + "/directory"}, navItem{Label: "Listings", Href: b + "/listings"},
			navItem{Label: "History", Href: b + "/history"}, navItem{Label: "Disputes", Href: b + "/disputes"})
	case domain.MemberSuspended:
		items = append(items, navItem{Label: "Trades", Href: b + "/trades"}, navItem{Label: "History", Href: b + "/history"})
	default:
		items = append(items, navItem{Label: "History", Href: b + "/history"})
	}
	if c.isReviewer() || c.isAdmin() {
		items = append(items, navItem{Label: "Review queue", Href: b + "/review"})
	}
	if c.isAdmin() {
		items = append(items, navItem{Label: "Network admin", Href: b + "/admin"})
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
		return "Your membership is suspended. You can view your trades and history, and decline or withdraw trades, but you cannot propose, accept or post. Contact the network administrator to ask about restoring it."
	case domain.MemberDeparted:
		return "You have left this network. You can still read your history and download your data."
	}
	if c.Net.Status == domain.NetworkClosed {
		return "This network is closed. History stays readable and exports are available; new trades are not possible."
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
		p.Title, p.Text = "Form expired", "Not sent. This form could not be verified, usually because the page was open for a long time or cookies are blocked. Nothing changed. Go back, reload the page and try again."
	case 403:
		p.Title, p.Text = "No access", "You do not have access to this page."
	case 404:
		p.Title, p.Text = "Page not found", "We could not find that page. It may not exist, or you may not have access to it."
	case 429:
		p.Title, p.Text = "Too many attempts", "Too many attempts. Wait a minute and try again. Nothing changed."
	case 503:
		w.Header().Set("Retry-After", "5")
		p.Title, p.Text = "Busy, try again", "The service is busy and could not complete the request. Nothing changed. Wait a few seconds and try again."
	case 409:
		p.Title, p.Text = "Conflict", "That cannot be done in the current state. Nothing changed. Reload the page to see the latest state."
	default:
		st = 500
		p.Title, p.Text = "Something went wrong", "The server could not complete the request. Nothing is shown here about the cause. If you contact the administrator, quote the reference below."
	}
	p.Status = st
	s.render(w, r, nil, "error", st, p)
}

// fail maps an unexpected service error to a page-level response.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if httpx.IsBusy(err) || domain.CodeOf(err) == domain.CodeUnavailable { // SQLITE_BUSY: retryable, not a fault (OPS-011)
		s.renderError(w, r, 503)
		return
	}
	switch domain.CodeOf(err) {
	case domain.CodeUnauthenticated:
		s.clearSession(w, r)
		s.toLogin(w, r)
	case domain.CodeForbidden, domain.CodeApprovalRequired:
		s.renderError(w, r, 403)
	case domain.CodeNotFound:
		s.renderError(w, r, 404)
	case domain.CodeRateLimited:
		s.renderError(w, r, 429)
	case domain.CodeInvalidState, domain.CodeConflict, domain.CodeBlocked:
		s.renderError(w, r, 409)
	default:
		if domain.CodeOf(err) == domain.CodeInternal {
			log.Printf("web: internal error req=%q: %q", reqID(r), err.Error()) //nolint:gosec // G706: both values are %q-quoted, no control characters pass
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
