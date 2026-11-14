package web

import (
	"net/url"
	"strings"
	"testing"
)

// GOV-010 (QA-023): the leave page offers the member's self-export (SEC-017) before the
// departure completes, the offered form really downloads it, and it still works afterwards.
func TestLeaveOffersSelfExport(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	exportForm := `<form method="post" action="` + b + `/export">`

	al := w.browser(t)
	al.login("alice@example.org")
	f := al.get(b + "/leave")
	al.mustContain(f, "Download your data first", exportForm, `name="format"`, "Download my data", `href="`+b+`/history"`)
	if i, j := strings.Index(f.Body, exportForm), strings.Index(f.Body, `action="`+b+`/leave"`); i < 0 || j < 0 || i > j {
		t.Errorf("the self-export is not offered ahead of the leave form (export at %d, leave at %d)", i, j)
	}
	for _, format := range []string{"csv", "json"} {
		dl := al.post(b+"/export", f, url.Values{"format": {format}})
		if dl.Status != 200 || dl.Header.Get("Content-Type") != "application/zip" || !strings.HasPrefix(dl.Body, "PK") {
			t.Errorf("%s export from the leave page: status %d type %q", format, dl.Status, dl.Header.Get("Content-Type"))
		}
		if !strings.Contains(dl.Header.Get("Content-Disposition"), "-mine-"+format+".zip") {
			t.Errorf("%s export is not the member self-export: %q", format, dl.Header.Get("Content-Disposition"))
		}
	}

	// A member who cannot leave yet is offered the export too.
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get(b+"/leave"), "You cannot leave yet", exportForm, "Download my data")

	// After leaving, the export the page promised is still there.
	done := al.post(b+"/leave", f, url.Values{"confirm": {"1"}})
	al.mustContain(done, "You have left")
	h := al.get(b + "/history")
	if dl := al.post(b+"/export", h, url.Values{"format": {"csv"}}); dl.Status != 200 || !strings.HasPrefix(dl.Body, "PK") {
		t.Errorf("export after leaving: status %d", dl.Status)
	}
}
