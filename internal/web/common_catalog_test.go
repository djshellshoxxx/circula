package web

import (
	"bytes"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// UX-047: the actual dashboard and shared shell resolve catalogue labels
// without corrupting a network's already-plural or HTML-sensitive unit name.
func TestDashboardCatalogueAndUnitNames(t *testing.T) {
	for _, unit := range []string{"credit", "Hours", "<hours>&"} {
		t.Run(unit, func(t *testing.T) {
			p := &page{Title: "Home", Slug: "town", Unit: unit, Authed: true, Unread: 1000, SecUnseen: 1000,
				D: &dash{Card: makeCard(&service.BalanceView{Decimals: 2, CreditLimit: 10000, Available: 10000}, unit)}}
			var b bytes.Buffer
			if err := mustLoadTemplates()["dashboard"].ExecuteTemplate(&b, "base", p); err != nil {
				t.Fatal(err)
			}
			body := b.String()
			for _, label := range []string{"Skip to main content", "All networks", "Your balance", "Available to spend", "Propose a trade", "Help: credits, balances and statuses", "1,000 unread", "1,000 new"} {
				if !strings.Contains(body, label) {
					t.Fatalf("missing catalogue label %q", label)
				}
			}
			if strings.Contains(body, "Hourss") || strings.Contains(body, "<hours>") || strings.Contains(body, "%!") {
				t.Fatal("unit inflection, escaping or catalogue interpolation was corrupted")
			}
			if unit == "Hours" && !strings.Contains(body, "100.00 Hours") {
				t.Fatal("dashboard did not use the already-plural unit name")
			}
			if unit == "<hours>&" && !strings.Contains(body, "&lt;hours&gt;&amp;") {
				t.Fatal("unit name was not escaped")
			}
		})
	}
}
