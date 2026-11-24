package web

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func TestDocumentTitleCatalogue(t *testing.T) {
	const screen = "History 75% & </title>"
	const network = "Community 100% & <North>"
	reordered := func(key string, args ...any) string {
		switch key {
		case "common.title.public":
			return fmt.Sprintf("Circula | %[1]s", args...)
		case "common.title.network":
			return fmt.Sprintf("Circula | %[2]s | %[1]s", args...)
		default:
			return messages.Text(key, args...)
		}
	}
	for _, tc := range []struct {
		name, netName, want string
		reorder             bool
	}{
		{"public English", "", screen + " - Circula", false},
		{"network English", network, screen + " - " + network + " - Circula", false},
		{"public reordered", "", "Circula | " + screen, true},
		{"network reordered", network, "Circula | " + network + " | " + screen, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tpl := mustLoadTemplates()["login"]
			if tc.reorder {
				tpl.Funcs(template.FuncMap{"msg": reordered})
			}
			var body strings.Builder
			if err := tpl.ExecuteTemplate(&body, "base", &page{Title: screen, NetName: tc.netName}); err != nil {
				t.Fatal(err)
			}
			_, after, _ := strings.Cut(body.String(), "<title>")
			title, _, found := strings.Cut(after, "</title>")
			if want := template.HTMLEscapeString(tc.want); !found || title != want {
				t.Errorf("document title %q, want %q", title, want)
			}
		})
	}
}

// These HTTP cases need no database: the public error path must not read one.
func sharedPagesServer() *server {
	return newServer(nil, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func sharedPagesRequest(s *server, h http.HandlerFunc) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/shared-test", nil)
	r.Header.Set("X-Request-Id", "shared-error-ref")
	w := httptest.NewRecorder()
	s.wrap(h).ServeHTTP(w, r)
	return w
}

func TestSharedPagesErrorHTTP(t *testing.T) {
	s := sharedPagesServer()
	for _, tc := range []struct {
		input, status int
		title         string
	}{
		{errCSRF, 403, "Form expired"}, {403, 403, "No access"},
		{404, 404, "Page not found"}, {422, 422, "Request not accepted"},
		{429, 429, "Too many attempts"}, {503, 503, "Busy, try again"},
		{errFrozen, 423, "Network read-only"}, {409, 409, "Conflict"},
		{500, 500, "Something went wrong"}, {599, 500, "Something went wrong"},
	} {
		t.Run(tc.title, func(t *testing.T) {
			w := sharedPagesRequest(s, func(w http.ResponseWriter, r *http.Request) { s.renderError(w, r, tc.input) })
			if w.Code != tc.status || !strings.Contains(w.Body.String(), `<h1 id="page-title">`+tc.title+`</h1>`) {
				t.Fatalf("input %d: status %d, body %s", tc.input, w.Code, w.Body.String())
			}
			wantRetry := ""
			if tc.status == 503 {
				wantRetry = "5"
			}
			if w.Header().Get("Retry-After") != wantRetry || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Fatalf("error response headers changed: %v", w.Header())
			}
			if strings.Contains(w.Body.String(), `href="/login"`) != (tc.status == 403) {
				t.Fatal("error page changed sign-in action visibility")
			}
			if !strings.Contains(w.Body.String(), "shared-error-ref") || w.Header().Get("X-Request-Id") != "shared-error-ref" {
				t.Fatal("error reference lost request correlation")
			}
		})
	}
}

func TestSharedPagesFailHTTPPrivacyAndRedirects(t *testing.T) {
	s := sharedPagesServer()
	const secret = "private-database-detail <img src=x>"
	for _, tc := range []struct {
		code   domain.Code
		status int
	}{
		{domain.CodeUnavailable, 503}, {domain.CodeNetworkFrozen, 423},
		{domain.CodeForbidden, 403}, {domain.CodeApprovalRequired, 403},
		{domain.CodeNotFound, 404}, {domain.CodeValidation, 422},
		{domain.CodeRateLimited, 429}, {domain.CodeInvalidState, 409},
		{domain.CodeConflict, 409}, {domain.CodeBlocked, 409},
		{domain.CodeTermsRequired, 409}, {domain.CodeInternal, 500},
	} {
		t.Run(string(tc.code), func(t *testing.T) {
			w := sharedPagesRequest(s, func(w http.ResponseWriter, r *http.Request) {
				s.fail(w, r, &domain.Error{Code: tc.code, Msg: secret, Cause: errors.New(secret)})
			})
			if w.Code != tc.status || strings.Contains(w.Body.String(), "private-database-detail") {
				t.Fatalf("refusal changed status or exposed a cause: %d %s", w.Code, w.Body.String())
			}
		})
	}
	for _, tc := range []struct {
		code  domain.Code
		path  string
		next  string
		terms bool
	}{
		{domain.CodeUnauthenticated, "/login", "/shared-test", false},
		{domain.CodePasswordChangeRequired, "/account/password", "", false},
		{domain.CodeTermsRequired, "/n/town/terms", "/n/town/trades", true},
	} {
		w := sharedPagesRequest(s, func(w http.ResponseWriter, r *http.Request) {
			if tc.terms {
				r.SetPathValue("slug", "town")
			}
			s.fail(w, r, &domain.Error{Code: tc.code, Msg: secret})
		})
		location, err := url.Parse(w.Header().Get("Location"))
		if err != nil || w.Code != http.StatusSeeOther || location.Path != tc.path || location.Query().Get("next") != tc.next {
			t.Fatalf("%s redirect changed: status %d, location %q", tc.code, w.Code, w.Header().Get("Location"))
		}
		if tc.terms {
			r := httptest.NewRequest(http.MethodGet, "/n/town/terms", nil)
			for _, cookie := range w.Result().Cookies() {
				r.AddCookie(cookie)
			}
			if got := s.takeFlash(httptest.NewRecorder(), r); got != "The network's rules and terms changed. Please read and accept them before you trade." {
				t.Fatalf("terms flash changed: %q", got)
			}
		}
	}
}

func TestSharedPagesRenderFallbacksAndRecovery(t *testing.T) {
	s := sharedPagesServer()
	s.pages["broken-shared-test"] = template.Must(template.New("broken-shared-test").Funcs(template.FuncMap{
		"breakRender": func() (string, error) { return "", errors.New("private-render-failure") },
	}).Parse(`{{define "base"}}partial-buffer{{breakRender}}{{end}}`))
	for _, tc := range []struct{ name, body string }{
		{"missing-shared-test", "internal error\n"},
		{"broken-shared-test", "internal error, reference shared-error-ref\n"},
	} {
		w := sharedPagesRequest(s, func(w http.ResponseWriter, r *http.Request) { s.render(w, r, nil, tc.name, 200, &page{}) })
		if w.Code != 500 || w.Body.String() != tc.body || w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Fatalf("%s fallback changed: %d %q %v", tc.name, w.Code, w.Body.String(), w.Header())
		}
	}
	w := sharedPagesRequest(s, func(http.ResponseWriter, *http.Request) { panic("private-panic-detail") })
	if w.Code != 500 || !strings.Contains(w.Body.String(), "Something went wrong") || strings.Contains(w.Body.String(), "private-panic-detail") {
		t.Fatal("panic recovery lost its generic response")
	}
	var aborted any
	var body string
	func() {
		defer func() { aborted = recover() }()
		r := httptest.NewRequest(http.MethodGet, "/shared-test", nil)
		w := httptest.NewRecorder()
		defer func() { body = w.Body.String() }()
		s.wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })).ServeHTTP(w, r)
	}()
	if aborted != http.ErrAbortHandler || body != "" {
		t.Fatalf("abort sentinel was consumed or received an error page: %v %q", aborted, body)
	}
}
