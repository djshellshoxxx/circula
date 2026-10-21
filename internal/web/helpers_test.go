package web

import (
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

const testPW = "correct horse battery"

type world struct {
	env          *testutil.Env
	srv          *httptest.Server
	net          *domain.Network
	admin, alice domain.Actor
	adminM       *domain.Member
	aliceM       *domain.Member
	bob          domain.Actor
	bobM         *domain.Member
}

// newWorld builds a network (founder "admin") with members alice and bob.
func newWorld(t *testing.T) *world {
	t.Helper()
	env := testutil.NewEnv(t)
	w := &world{env: env}
	w.admin = env.Person("admin@example.org")
	w.net, w.adminM = env.Network(w.admin, "town", nil)
	w.alice, w.aliceM = env.Member(w.admin, w.net, "alice@example.org")
	w.bob, w.bobM = env.Member(w.admin, w.net, "bob@example.org")
	w.srv = httptest.NewServer(NewHandlerWith(env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: env.Clock}))
	t.Cleanup(w.srv.Close)
	return w
}

type browser struct {
	t   *testing.T
	w   *world
	c   *http.Client
	url string
}

func (w *world) browser(t *testing.T) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, w: w, url: w.srv.URL, c: &http.Client{Jar: jar}}
}

type pg struct {
	Status int
	Body   string
	Header http.Header
	URL    string
}

func (b *browser) do(req *http.Request) *pg {
	b.t.Helper()
	resp, err := b.c.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return &pg{resp.StatusCode, string(body), resp.Header, resp.Request.URL.Path}
}

func (b *browser) get(path string) *pg {
	b.t.Helper()
	req, _ := http.NewRequest("GET", b.url+path, nil)
	return b.do(req)
}

var csrfRe = regexp.MustCompile(`name="csrf" value="([^"]+)"`)
var idemRe = regexp.MustCompile(`name="idem" value="([^"]+)"`)

func (p *pg) csrf() string {
	m := csrfRe.FindStringSubmatch(p.Body)
	if m == nil {
		return ""
	}
	return m[1]
}

func (p *pg) idem() string {
	m := idemRe.FindStringSubmatch(p.Body)
	if m == nil {
		return ""
	}
	return m[1]
}

// post submits vals to path using the csrf (and idem) token of from.
func (b *browser) post(path string, from *pg, vals url.Values) *pg {
	b.t.Helper()
	if vals == nil {
		vals = url.Values{}
	}
	if vals.Get("csrf") == "" && from != nil {
		vals.Set("csrf", from.csrf())
	}
	if vals.Get("idem") == "" && from != nil && from.idem() != "" {
		vals.Set("idem", from.idem())
	}
	req, _ := http.NewRequest("POST", b.url+path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return b.do(req)
}

// login signs in through the real form.
func (b *browser) login(login string) *pg {
	b.t.Helper()
	f := b.get("/login")
	p := b.post("/login", f, url.Values{"login": {login}, "password": {testPW}})
	if p.Status != 200 || strings.Contains(p.Body, "id=\"error-summary\"") {
		b.t.Fatalf("login %s failed: %d %s", login, p.Status, p.Body)
	}
	return p
}

func (b *browser) mustContain(p *pg, subs ...string) {
	b.t.Helper()
	body := html.UnescapeString(p.Body)
	for _, s := range subs {
		if !strings.Contains(body, s) && !strings.Contains(p.Body, s) {
			b.t.Errorf("page %s (status %d) lacks %q\n%s", p.URL, p.Status, s, snippet(p.Body))
		}
	}
}

func snippet(s string) string {
	if i := strings.Index(s, "<main"); i >= 0 {
		s = s[i:]
	}
	if len(s) > 1800 {
		s = s[:1800]
	}
	return s
}

func (w *world) base() string { return "/n/" + w.net.Slug }
