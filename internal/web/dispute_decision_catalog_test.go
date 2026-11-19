package web

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// UX-047, FR-078: the completed re-review explanation uses the same existing
// singular/custom-unit selection as the review screen, with exact amount text.
func TestReReviewCatalogueKeepsAmountAndUnit(t *testing.T) {
	for _, tc := range []struct {
		outcome, currency string
		amount            int64
		decimals          int
		want              string
	}{
		{string(domain.OutcomeUpheld), "credit", 100, 2, "Re-review decided, upheld: a further reversal of 1.00 credit has been posted."},
		{string(domain.OutcomePartiallyUpheld), "Hours", 9007199254740993, 2, "Re-review decided, partly upheld: a further reversal of 90071992547409.93 Hours has been posted."},
		{string(domain.OutcomeUpheld), "credit", 200, 2, "Re-review decided, upheld: a further reversal of 2.00 credits has been posted."},
		{service.ReReviewConfirmed, "credit", 0, 2, "Re-review decided: the original decision stands."},
		{`unknown_%! "><img src=x>`, "credit", 0, 2, "Re-review decided."},
	} {
		got := reReviewWords(tc.outcome, domain.FormatAmount(tc.amount, tc.decimals), unitFor(tc.currency, tc.amount, tc.decimals))
		if got != tc.want {
			t.Errorf("%s/%s: got %q, want %q", tc.outcome, tc.currency, got, tc.want)
		}
	}
}
