package web

import (
	"html"
	"net/url"
	"regexp"
	"testing"

	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A validation error must keep the local return destination through the
// rendered retry form, rather than sending the member to the trades index.
func TestTermsAcceptanceRetryPreservesSafeNext(t *testing.T) {
	w := newWorld(t)
	base := w.base()
	admin := w.browser(t)
	admin.login("admin@example.org")
	publish := admin.get(base + "/admin/terms")
	saved := admin.post(base+"/admin/terms", publish, url.Values{
		"body": {"Terms for the retry test."},
	})
	admin.mustContain(saved, "Published version 1")

	member := w.browser(t)
	member.login("alice@example.org")
	want := base + "/trades/new"
	form := member.get(base + "/terms?next=" + url.QueryEscape(want))
	if form.Status != 200 {
		t.Fatalf("GET terms: status %d", form.Status)
	}

	invalid := member.post(base+"/terms", form, url.Values{
		"version": {"1"},
		"next":    {want},
		// Deliberately omit the required acceptance checkbox.
	})
	if invalid.Status != 422 {
		t.Fatalf("POST terms without acceptance: status %d, want 422", invalid.Status)
	}

	nextField := regexp.MustCompile(`name="next" value="([^"]*)"`)
	m := nextField.FindStringSubmatch(invalid.Body)
	if m == nil {
		t.Fatalf("422 terms form has no hidden next input")
	}
	retryNext := html.UnescapeString(m[1])
	if retryNext != want {
		t.Fatalf("422 terms form next=%q, want %q", retryNext, want)
	}

	accepted := member.post(base+"/terms", invalid, url.Values{
		"version": {"1"},
		"accept":  {"1"},
		"next":    {retryNext},
	})
	if accepted.URL != want {
		t.Fatalf("accepted retry redirected to %q, want %q", accepted.URL, want)
	}
	if accepted.Status != 200 {
		t.Fatalf("accepted retry: status %d, want 200", accepted.Status)
	}
	if v := testutil.Must(w.env.Svc.CurrentTerms(w.env.Ctx, w.alice, w.net.ID)); v.NeedsAcceptance() {
		t.Error("terms acceptance was not recorded")
	}
}

// Re-rendering a rejected form must not make an external destination usable.
func TestTermsAcceptanceRetryRejectsExternalNext(t *testing.T) {
	w := newWorld(t)
	base := w.base()
	admin := w.browser(t)
	admin.login("admin@example.org")
	form := admin.get(base + "/admin/terms")
	admin.post(base+"/admin/terms", form, url.Values{"body": {"Terms."}})

	member := w.browser(t)
	member.login("alice@example.org")
	terms := member.get(base + "/terms")
	rejected := member.post(base+"/terms", terms, url.Values{
		"version": {"1"},
		"next":    {"https://example.invalid/redirect"},
	})
	if rejected.Status != 422 {
		t.Fatalf("POST with missing acceptance: status %d, want 422", rejected.Status)
	}
	if m := regexp.MustCompile(`name="next" value="([^"]*)"`).FindStringSubmatch(rejected.Body); m == nil || m[1] != "" {
		t.Fatalf("unsafe next was not cleared from retry form")
	}
}
