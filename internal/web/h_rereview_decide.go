package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

type reReviewDecisionData struct {
	decisionData
	Proof                    string
	PayerChange, PayeeChange string
	ShowBalances             bool
	PayerBefore, PayerAfter  balanceVM
	PayeeBefore, PayeeAfter  balanceVM
}

func (s *server) reReviewProof(c *rc, in service.ReReviewDecideInput) string {
	b, _ := json.Marshal([]string{c.netID(), c.actor.PersonID, in.DisputeID, in.Outcome,
		strconv.FormatInt(in.Amount, 10), in.Rationale})
	return s.mac("rereview-decision:" + string(b))
}

// reReviewDecisionReview shows the exact proposed effect without deciding or
// posting. The service still revalidates authorization and live state inside
// the confirm transaction; a signed proof prevents altering reviewed values.
func (s *server) reReviewDecisionReview(w http.ResponseWriter, r *http.Request, c *rc, id string, in service.ReReviewDecideInput) {
	v, err := s.svc.GetDispute(r.Context(), c.actor, c.netID(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if (!c.isReviewer() && !c.isAdmin()) || v.Tx.IsParty(c.Me.ID) || v.Tx.CreatedBy == c.Me.ID ||
		v.Tx.AcceptedBy == c.Me.ID || v.Dispute.ReviewerID == c.Me.ID {
		s.fail(w, r, domain.Forbidden("%s", messages.Text("dispute.rereview.independent_required")))
		return
	}
	if v.ReReview == nil || v.ReReview.Status != service.ReReviewRequested {
		s.disputeFail(w, r, c, id, domain.InvalidState("%s", messages.Text("dispute.rereview.none_pending")))
		return
	}
	pol := c.Net.Policy
	remaining := v.Tx.Amount - v.Tx.ReversedTotal
	var words string
	switch in.Outcome {
	case service.ReReviewConfirmed:
		in.Amount, words = 0, messages.Text("dispute.rereview.confirm_option")
	case string(domain.OutcomeUpheld):
		in.Amount, words = remaining, messages.Text("dispute.rereview.uphold_option")
	case string(domain.OutcomePartiallyUpheld):
		words = messages.Text("dispute.rereview.partial_option")
	default:
		s.disputeFail(w, r, c, id, domain.Invalid("outcome", messages.Text("dispute.rereview.outcome_required")))
		return
	}
	if in.Outcome != service.ReReviewConfirmed && (in.Amount <= 0 || in.Amount > remaining) {
		s.disputeFail(w, r, c, id, domain.Invalid("amount", messages.Text("dispute.rereview.amount_max", domain.FormatAmount(remaining, pol.Decimals))))
		return
	}
	if in.Rationale == "" || utf8.RuneCountInString(in.Rationale) > 2000 {
		s.disputeFail(w, r, c, id, domain.Invalid("rationale", messages.Text("dispute.rereview.rationale_required")))
		return
	}
	if in.Amount > 0 && v.Tx.Status == domain.TxDisputed {
		s.disputeFail(w, r, c, id, domain.InvalidState("%s", messages.Text("dispute.rereview.other_undecided")))
		return
	}
	d := &reReviewDecisionData{decisionData: decisionData{ID: id, Outcome: words, Amount: domain.FormatAmount(in.Amount, pol.Decimals),
		Unit: unitFor(pol.CurrencyName, in.Amount, pol.Decimals), Payer: v.PayerName, Payee: v.PayeeName,
		Rationale: in.Rationale, Reverses: in.Amount > 0}, Proof: s.reReviewProof(c, in),
		PayerChange: makeBalance(in.Amount, pol.Decimals, pol.CurrencyName).Signed,
		PayeeChange: makeBalance(-in.Amount, pol.Decimals, pol.CurrencyName).Signed}
	// PER-003: a reviewer sees only the dispute and signed effects. Full
	// account balances stay admin-only, using the existing audited read.
	if c.isAdmin() {
		payer, err := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), v.Tx.PayerMemberID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		payee, err := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), v.Tx.PayeeMemberID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		payerAfter, payerOK := domain.AddBounded(payer.Balance, in.Amount)
		payeeAfter, payeeOK := domain.AddBounded(payee.Balance, -in.Amount)
		if !payerOK || !payeeOK {
			s.disputeFail(w, r, c, id, domain.Invalid("amount", messages.Text("dispute.rereview.balance_range")))
			return
		}
		d.ShowBalances = true
		d.PayerBefore, d.PayerAfter = makeBalance(payer.Balance, pol.Decimals, pol.CurrencyName), makeBalance(payerAfter, pol.Decimals, pol.CurrencyName)
		d.PayeeBefore, d.PayeeAfter = makeBalance(payee.Balance, pol.Decimals, pol.CurrencyName), makeBalance(payeeAfter, pol.Decimals, pol.CurrencyName)
	}
	values := formValues(r)
	values["amount"], values["rationale"] = domain.FormatAmount(in.Amount, pol.Decimals), in.Rationale
	s.render(w, r, c, "dispute_rereview_confirm", 200, &page{Title: messages.Text("dispute.rereview.title"), V: values, D: d})
}
