package web

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// UX-047: only known enum values become catalogue keys. Unknown data keeps its
// exact identity and is escaped by the actual templates tested in messages.
func TestDisputeCatalogueEnumFallbacks(t *testing.T) {
	const unknown = "\x00%!" + ` unknown_type "><img src=x> Zoë 100%`
	if got := categoryLabel(unknown); got != unknown {
		t.Fatalf("unknown category changed: %q", got)
	}
	if got := statusWords(domain.DisputeStatus(unknown)); got != unknown {
		t.Fatalf("unknown status changed: %q", got)
	}
	for _, tc := range []struct{ action, outcome, want string }{
		{"opened", "", "Opened"},
		{"under_review", "", "Under review"},
		{"posting_pending", "partially_upheld", "Posting pending (partially upheld)"},
		{"resolved", "upheld", "Resolved (upheld)"},
		{"rereview_confirmed", "dismissed", "Rereview confirmed (dismissed)"},
		{unknown, "", unknown},
		{unknown, unknown, unknown + " (" + unknown + ")"},
		{"", "", ""},
	} {
		if got := disputeEventWords(tc.action, tc.outcome); got != tc.want {
			t.Errorf("event (%q,%q) = %q, want %q", tc.action, tc.outcome, got, tc.want)
		}
	}
	for _, tc := range []struct{ value, label string }{
		{"not_delivered", "Not delivered"}, {"quality", "Not as described or poor quality"},
		{"wrong_amount", "Wrong amount"}, {"unauthorised", "I did not agree to this trade"}, {"other", "Something else"},
	} {
		if got := categoryLabel(tc.value); got != tc.label {
			t.Errorf("category %q = %q, want %q", tc.value, got, tc.label)
		}
	}
}

func TestDisputeCatalogueOutcomeKeepsExactAmount(t *testing.T) {
	const amount = "90071992547409.93"
	for _, tc := range []struct {
		outcome domain.DisputeOutcome
		status  domain.DisputeStatus
		want    string
	}{
		{domain.OutcomeUpheld, domain.DisputePostingPending, "Upheld: a reversal of " + amount + " Hours is decided but not yet posted."},
		{domain.OutcomePartiallyUpheld, domain.DisputeResolved, "Partly upheld: a reversal of " + amount + " Hours has been posted."},
		{domain.OutcomeDismissed, domain.DisputeResolved, "Dismissed: the balance stands as before."},
		{domain.DisputeOutcome("future-outcome"), domain.DisputeResolved, ""},
	} {
		if got := outcomeWords(&domain.Dispute{Outcome: tc.outcome, Status: tc.status}, amount, "Hours"); got != tc.want {
			t.Errorf("outcome %s/%s = %q, want %q", tc.outcome, tc.status, got, tc.want)
		}
	}
}
