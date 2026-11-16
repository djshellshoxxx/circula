package web

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// reviewData fills the response (FR-076) and re-review (FR-078) parts of the dispute page.
func (s *server) reviewData(c *rc, v *service.DisputeView, out *disputeData) {
	d := v.Dispute
	party := v.Tx.IsParty(c.Me.ID)
	live := d.Status == domain.DisputeOpen || d.Status == domain.DisputeUnderReview
	if v.Response != nil {
		out.Response = v.Response.Body
		out.ResponseBy = v.PayerName
		if v.Response.ResponderID != v.Tx.PayerMemberID {
			out.ResponseBy = v.PayeeName
		}
	}
	out.CanRespond = live && party && d.OpenedBy != c.Me.ID && v.Response == nil &&
		(c.Me.Status == domain.MemberActive || c.Me.Status == domain.MemberSuspended)

	rr := v.ReReview
	if rr != nil {
		out.ReReviewWhy = rr.Reason
		out.ReReviewText = "Re-review requested, waiting for a different reviewer."
		if rr.Status == store.ReReviewDecided {
			out.ReReviewDone = rr.Rationale
			out.ReReviewText = reReviewWords(rr.Outcome, domain.FormatAmount(rr.ReversalAmount, c.Net.Policy.Decimals), c.Net.Policy.CurrencyName)
		}
	}
	if d.Status == domain.DisputeResolved && d.DecidedAt != nil {
		until := d.DecidedAt.Add(time.Duration(c.Net.Policy.ReReviewWindowDays) * 24 * time.Hour)
		out.ReReviewUntil = fmtTime(until)
		out.CanReReview = rr == nil && party && c.isActive() && s.opts.Clock.Now().Before(until)
	}
	out.DecideReRev = rr != nil && rr.Status == store.ReReviewRequested && (c.isReviewer() || c.isAdmin()) && !party &&
		c.Me.ID != d.ReviewerID
}

func reReviewWords(outcome, amt, unit string) string {
	switch outcome {
	case service.ReReviewConfirmed:
		return "Re-review decided: the original decision stands."
	case string(domain.OutcomeUpheld):
		return "Re-review decided, upheld: a further reversal of " + amt + " " + unit + "s has been posted."
	case string(domain.OutcomePartiallyUpheld):
		return "Re-review decided, partly upheld: a further reversal of " + amt + " " + unit + "s has been posted."
	}
	return "Re-review decided."
}
