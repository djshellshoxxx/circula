package web

import (
	"net/http"
	"strconv"
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
