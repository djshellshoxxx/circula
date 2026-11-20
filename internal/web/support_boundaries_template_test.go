package web

import (
	"bytes"
	"html/template"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/web/messages"
)

type supportBoundaryPageData struct {
	D    any
	Slug string
}

func renderSupportBoundaryTemplate(t *testing.T, name string, data any) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate the test source")
	}
	path := filepath.Join(filepath.Dir(source), "templates", name)
	// Use the same catalogue function as production mustLoadTemplates.
	tmpl, err := template.New(name).Funcs(template.FuncMap{"msg": messages.Text}).ParseFiles(path)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "content", data); err != nil {
		t.Fatalf("render %s: %v", name, err)
	}
	return output.String()
}

func TestHelpDocumentsSupportBoundaries(t *testing.T) {
	body := renderSupportBoundaryTemplate(t, "help.html", nil)
	if !strings.Contains(body, `<h2 id="sessions">Review and end your sign-ins</h2>`) {
		t.Error("help page did not render the localized session guidance")
	}
	for _, required := range []string{
		`<h2 id="support-boundaries">Support boundaries</h2>`,
		"network administrators",
		"network rules and trades",
		"software, deployment and platform policy",
		"support request",
		"time-limited break-glass (emergency) access",
		"documented reason",
		"second staff approver",
		"record readable by network administrators",
		"notice when required",
		"operator procedures",
		"technical enforcement is incomplete",
	} {
		if !strings.Contains(body, required) {
			t.Errorf("help page is missing support-boundary text %q", required)
		}
	}
	for _, unsupportedClaim := range []string{"cannot technically access", "cryptographically prevented", "guaranteed response time"} {
		if strings.Contains(strings.ToLower(body), unsupportedClaim) {
			t.Errorf("help page makes unsupported security or service claim %q", unsupportedClaim)
		}
	}
}

func TestAdminHomeLinksToSupportBoundaries(t *testing.T) {
	body := renderSupportBoundaryTemplate(t, "admin_home.html", supportBoundaryPageData{Slug: "demo"})
	if !strings.Contains(body, `href="/help#support-boundaries"`) {
		t.Fatal("admin home does not link to the support-boundaries heading")
	}
}
