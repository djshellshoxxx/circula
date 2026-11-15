package web

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// UX-061 through the real History page, router, session and service.
func TestHistoryPageHistoricalBalance(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	before := w.env.Clock.Now()
	w.env.Clock.Advance(time.Minute)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1250)
	at := w.env.Clock.Now()
	bob := w.browser(t)
	bob.login("bob@example.org")

	bare := bob.get(b + "/history")
	bob.mustContain(bare, `id="histbal-at"`, "Balance at an exact time")
	if strings.Contains(bare.Body, `id="histbal-result"`) || strings.Contains(bare.Body, `id="histbal-err"`) {
		t.Fatal("bare page shows a result or error")
	}

	ok := bob.get(b + "/history?at=" + url.QueryEscape(at.In(time.FixedZone("", -5*3600)).Format(time.RFC3339)))
	bob.mustContain(ok, `id="histbal-result"`, "+12.50", `datetime="`+at.UTC().Format("2006-01-02T15:04:05.000Z")+`"`, "posted balance of your account at this instant")
	for _, current := range []string{"Available", "credit limit"} {
		if strings.Contains(sectionOf(ok.Body), current) {
			t.Errorf("historical section mentions current value %q", current)
		}
	}

	none := bob.get(b + "/history?at=" + url.QueryEscape(before.Format(time.RFC3339)))
	bob.mustContain(none, "No postings as of this instant")

	for raw, reason := range map[string]string{
		"2026-03-01":                           "use a full RFC 3339 date-time",
		at.Add(time.Hour).Format(time.RFC3339): "not in the future",
		"":                                     "enter a date and time",
	} {
		p := bob.get(b + "/history?at=" + url.QueryEscape(raw))
		bob.mustContain(p, `id="histbal-err"`, `aria-invalid="true"`, reason, `value="`+raw+`"`)
		if strings.Contains(p.Body, `id="histbal-result"`) {
			t.Errorf("%q: result shown with an error", raw)
		}
	}

	// Submitted value is escaped, not interpreted.
	x := bob.get(b + "/history?at=" + url.QueryEscape(`"><script>x</script>`))
	if strings.Contains(x.Body, "<script>x</script>") {
		t.Fatal("submitted value not escaped")
	}
}

func sectionOf(body string) string {
	i := strings.Index(body, `aria-labelledby="histbal-h"`)
	if i < 0 {
		return ""
	}
	j := strings.Index(body[i:], "</section>")
	return body[i : i+j]
}
