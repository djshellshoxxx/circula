package web

import (
	"fmt"
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type actData struct {
	Action      string
	Verb        string
	Tx          *domain.Transaction
	Payer       string
	Payee       string
	Sentence    string
	Before      balanceVM
	After       balanceVM
	AvailNow    string
	AvailAfter  string
	AskReason   bool
	Consequence string
}

func (c *rc) actView(r *http.Request, action string) (*actData, error) {
	v, err := c.s.svc.GetTransaction(c.ctx, c.actor, c.netID(), r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	if action != "accept" && action != "decline" && action != "cancel" {
		return nil, domain.NotFound("page")
	}
	n := c.memberNames()
	t := v.Tx
	pol := c.Net.Policy
	a := &actData{Action: action, Tx: t, Payer: n.of(t.PayerMemberID), Payee: n.of(t.PayeeMemberID)}
	amt := domain.FormatAmount(t.Amount, pol.Decimals)
	u := unitFor(pol.CurrencyName, t.Amount, pol.Decimals)
	a.Sentence = fmt.Sprintf("%s pays %s %s %s", a.Payer, a.Payee, amt, u)
	if t.PayerMemberID == c.Me.ID {
		a.Sentence = fmt.Sprintf("You pay %s %s %s", a.Payee, amt, u)
	} else if t.PayeeMemberID == c.Me.ID {
		a.Sentence = fmt.Sprintf("%s pays you %s %s", a.Payer, amt, u)
	}
	switch action {
	case "accept":
		a.Verb = "Accept"
		b, err := c.s.svc.GetBalance(c.ctx, c.actor, c.netID())
		if err != nil {
			return nil, err
		}
		a.Before = makeBalance(b.Balance, b.Decimals, pol.CurrencyName)
		a.AvailNow = minus(plainAmt(b.Available, b.Decimals))
		if t.PayerMemberID == c.Me.ID {
			a.After = makeBalance(b.Balance-t.Amount, b.Decimals, pol.CurrencyName)
			a.AvailAfter = a.AvailNow // the amount is already set aside while pending
		} else {
			a.After = makeBalance(b.Balance+t.Amount, b.Decimals, pol.CurrencyName)
			a.AvailAfter = minus(plainAmt(b.Available+t.Amount, b.Decimals))
		}
	case "decline":
		a.Verb = "Decline"
		a.Consequence = "Nothing is posted and the amount set aside for this trade is released."
	case "cancel":
		a.Verb = "Cancel"
		a.AskReason = !t.IsParty(c.Me.ID)
		a.Consequence = "Nothing is posted and the amount set aside for this trade is released."
	}
	return a, nil
}

func (s *server) tradeActionForm(w http.ResponseWriter, r *http.Request, c *rc) {
	act := r.PathValue("action")
	a, err := c.actView(r, act)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.renderAct(w, r, c, 200, a, &page{})
}

func (s *server) renderAct(w http.ResponseWriter, r *http.Request, c *rc, st int, a *actData, p *page) {
	p.Title = a.Verb + " this trade?"
	p.D = a
	tm := "trade_confirm"
	if a.Action == "accept" {
		p.Title, tm = "Review before accepting", "trade_accept"
	}
	s.render(w, r, c, tm, st, p)
}

func (s *server) tradeActionPost(w http.ResponseWriter, r *http.Request, c *rc) {
	act := r.PathValue("action")
	a, err := c.actView(r, act)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ref := service.TxRef{TxID: a.Tx.ID, IdempotencyKey: idemKey(r)}
	var msg string
	switch act {
	case "accept":
		_, err = s.svc.AcceptTransaction(r.Context(), c.actor, c.netID(), ref)
		msg = "Trade completed."
	case "decline":
		_, err = s.svc.DeclineTransaction(r.Context(), c.actor, c.netID(), ref)
		msg = "Trade declined. Nothing was posted."
	case "cancel":
		_, err = s.svc.CancelTransaction(r.Context(), c.actor, c.netID(), service.CancelInput{TxID: ref.TxID,
			Reason: r.PostFormValue("reason"), IdempotencyKey: ref.IdempotencyKey})
		msg = "Trade cancelled. Nothing was posted."
	}
	if err != nil {
		p := &page{V: formValues(r)}
		if sum, fe, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			st := 422
			if domain.CodeOf(err) == domain.CodeInvalidState {
				st = 409
			}
			s.renderAct(w, r, c, st, a, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	if act == "accept" {
		if b, e := s.svc.GetBalance(r.Context(), c.actor, c.netID()); e == nil {
			bv := makeBalance(b.Balance, b.Decimals, c.Net.Policy.CurrencyName)
			msg += " Your balance is now " + bv.Signed + " \u00b7 " + bv.State + "."
		}
	}
	s.setFlash(w, r, msg)
	http.Redirect(w, r, c.base()+"/trades/"+a.Tx.ID, http.StatusSeeOther)
}
