package web

import (
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Invalid setup precision must be rejected before error examples are formatted.
// The moderate large value reproduces the old unbounded formatting path without
// asking the regression process to make an unsafe allocation.
func TestSetupRejectsInvalidPrecisionBeforeFormatting(t *testing.T) {
	w := newWorld(t)
	b := w.browser(t)
	b.login("alice@example.org")
	form := b.get("/setup")
	for i, precision := range []string{"", "not-a-number", "-1", "5", "4096", "999999999999999999999999999"} {
		t.Run(precision, func(t *testing.T) {
			slug := "invalid-precision-" + strconv.Itoa(i)
			values := setupVals("", slug)
			values.Set("decimals", precision)
			p := b.post("/setup", form, values)
			if p.Status != http.StatusUnprocessableEntity {
				t.Fatalf("invalid precision %q: status %d, want 422", precision, p.Status)
			}
			b.mustContain(p, `id="e-decimals"`, "Decimal places: choose a value from 0 to 4.")
			if hasNetwork(w, slug) {
				t.Fatal("invalid precision created a network")
			}
		})
	}
}

// UX-047: real setup refusals retain inputs and associate each complete
// precision-sensitive correction with its field without creating a network.
func TestSetupAmountValidationUsesSelectedPrecision(t *testing.T) {
	for _, tc := range []struct {
		precision string
		want      string
	}{
		{"0", "enter a number with up to 0 decimal places, for example 1250."},
		{"1", "enter a number with up to 1 decimal place, for example 125.0."},
		{"2", "enter a number with up to 2 decimal places, for example 12.50."},
		{"4", "enter a number with up to 4 decimal places, for example 0.1250."},
	} {
		t.Run(tc.precision, func(t *testing.T) {
			w := newWorld(t)
			b := w.browser(t)
			b.login("alice@example.org")
			form := b.get("/setup")
			slug := "amount-precision-" + tc.precision
			values := setupVals("", slug)
			values.Set("decimals", tc.precision)
			values.Set("default_credit_limit", `invalid <credit&"`)
			values.Set("signup_grant", `invalid <grant&"`)
			p := b.post("/setup", form, values)
			if p.Status != http.StatusUnprocessableEntity {
				t.Fatalf("invalid amounts at precision %s: status %d, want 422", tc.precision, p.Status)
			}
			b.mustContain(p, `<option value="`+tc.precision+`" selected>`)
			for _, field := range []struct {
				name  string
				label string
			}{
				{"default_credit_limit", "Default credit limit"},
				{"signup_grant", "Sign-up grant"},
			} {
				want := field.label + ": " + tc.want
				b.mustContain(p, `href="#f-`+field.name+`">`+want+`</a>`, `id="e-`+field.name+`"`)
				if got := strings.Count(p.Body, want); got != 2 {
					t.Errorf("%s correction appears %d times, want summary and inline field error", field.name, got)
				}
				input := regexp.MustCompile(`<input\b[^>]*\bid="f-` + field.name + `"[^>]*>`).FindString(p.Body)
				for _, attr := range []string{
					`value="` + html.EscapeString(values.Get(field.name)) + `"`,
					`aria-invalid="true"`,
					`aria-describedby="h-` + field.name + ` e-` + field.name + `"`,
				} {
					if !strings.Contains(input, attr) {
						t.Errorf("%s input lacks %q: %s", field.name, attr, input)
					}
				}
			}
			if hasNetwork(w, slug) {
				t.Fatal("invalid setup amounts created a network")
			}
		})
	}
}
