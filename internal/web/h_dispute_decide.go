package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// decisionData is the review-your-decision screen.
type decisionData struct {
	ID, Outcome, Amount, Unit, Payer, Payee, Rationale string
	Reverses                                           bool
}

// decisionReview shows what a decision would do before it is recorded. Nothing changes here.
func (s *server) decisionReview(w http.ResponseWriter, r *http.Request, c *rc, id string, in service.ResolveInput) {
	v, err := s.svc.GetDispute(r.Context(), c.actor, c.netID(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	pol := c.Net.Policy
	rat := strings.TrimSpace(in.Rationale)
	remaining := v.Tx.Amount - v.Tx.ReversedTotal
	var amt int64
	var words string
	switch in.Outcome {
	case domain.OutcomeDismissed:
		words = "Dismissed: the balances stand as they are."
	case domain.OutcomeUpheld:
		amt, words = remaining, "Upheld: the whole remaining trade is reversed."
	case domain.OutcomePartiallyUpheld:
		amt, words = in.Amount, "Partly upheld: part of the trade is reversed."
		if amt > remaining {
			s.disputeFail(w, r, c, id, domain.Invalid("amount", "must be at most the remaining reversible amount "+domain.FormatAmount(remaining, pol.Decimals)))
			return
		}
	default:
		s.disputeFail(w, r, c, id, domain.Invalid("outcome", "choose dismissed, upheld or partly upheld"))
		return
	}
	switch {
	case rat == "":
		s.disputeFail(w, r, c, id, domain.Invalid("rationale", "required: both parties will see it"))
		return
	case len(rat) > 2000:
		s.disputeFail(w, r, c, id, domain.Invalid("rationale", "too long: use 2000 characters or fewer"))
		return
	}
	p := &page{Title: "Review your decision", V: formValues(r), D: &decisionData{ID: id, Outcome: words,
		Amount: domain.FormatAmount(amt, pol.Decimals), Unit: unitFor(pol.CurrencyName, amt, pol.Decimals), Payer: v.PayerName, Payee: v.PayeeName,
		Rationale: rat, Reverses: amt > 0}}
	s.render(w, r, c, "dispute_decide_review", 200, p)
}
