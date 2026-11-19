package web

import (
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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
		words = messages.Text("dispute.decision.dismissed")
	case domain.OutcomeUpheld:
		amt, words = remaining, messages.Text("dispute.decision.upheld")
	case domain.OutcomePartiallyUpheld:
		amt, words = in.Amount, messages.Text("dispute.decision.partial")
		if amt > remaining {
			s.disputeFail(w, r, c, id, domain.Invalid("amount", messages.Text("dispute.decision.amount_max", domain.FormatAmount(remaining, pol.Decimals))))
			return
		}
	default:
		s.disputeFail(w, r, c, id, domain.Invalid("outcome", messages.Text("dispute.decision.outcome_required")))
		return
	}
	switch {
	case rat == "":
		s.disputeFail(w, r, c, id, domain.Invalid("rationale", messages.Text("dispute.decision.rationale_required")))
		return
	case len(rat) > 2000:
		s.disputeFail(w, r, c, id, domain.Invalid("rationale", messages.Text("dispute.decision.rationale_too_long")))
		return
	}
	p := &page{Title: messages.Text("dispute.decision.title"), V: formValues(r), D: &decisionData{ID: id, Outcome: words,
		Amount: domain.FormatAmount(amt, pol.Decimals), Unit: unitFor(pol.CurrencyName, amt, pol.Decimals), Payer: v.PayerName, Payee: v.PayeeName,
		Rationale: rat, Reverses: amt > 0}}
	s.render(w, r, c, "dispute_decide_review", 200, p)
}
