package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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
	a.Sentence = messages.Text("trade.sentence.other", a.Payer, a.Payee, amt, u)
	if t.PayerMemberID == c.Me.ID {
		a.Sentence = messages.Text("trade.sentence.pay", a.Payee, amt, u)
	} else if t.PayeeMemberID == c.Me.ID {
		a.Sentence = messages.Text("trade.sentence.receive", a.Payer, amt, u)
	}
	switch action {
	case "accept":
		a.Verb = messages.Text("trade.action.accept")
		b, err := c.s.svc.GetBalance(c.ctx, c.actor, c.netID())
		if err != nil {
			return nil, err
		}
		a.Before = makeBalance(b.Balance, b.Decimals, pol.CurrencyName)
		a.AvailNow = minus(plainAmt(b.Available, b.Decimals))
		a.After, a.AvailAfter = a.Before, a.AvailNow
		if t.Status != domain.TxProposed {
			// A repeated confirmation may read an already posted transaction.
			// Do not project its amount a second time before the service checks
			// idempotency or rejects another terminal state.
			break
		}
		delta, availableDelta := t.Amount, t.Amount
		if t.PayerMemberID == c.Me.ID {
			delta, availableDelta = -t.Amount, -t.Amount
			if t.CreatedBy == t.PayerMemberID {
				availableDelta = 0 // only payer-created proposals reserve credit
			}
		}
		after, balanceOK := domain.AddBounded(b.Balance, delta)
		available, availableOK := domain.AddChecked(b.Available, availableDelta)
		if !balanceOK || !availableOK {
			return nil, domain.Invalid("amount", messages.Text("trade.review.range_detail"))
		}
		a.After = makeBalance(after, b.Decimals, pol.CurrencyName)
		a.AvailAfter = minus(plainAmt(available, b.Decimals))
	case "decline":
		a.Verb = messages.Text("trade.action.decline")
		a.Consequence = messages.Text("trade.action.release")
	case "cancel":
		a.Verb = messages.Text("trade.action.cancel")
		a.AskReason = !t.IsParty(c.Me.ID)
		a.Consequence = messages.Text("trade.action.release")
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
	p.Title = messages.Text("trade.action.title", a.Verb)
	p.D = a
	tm := "trade_confirm"
	if a.Action == "accept" {
		p.Title, tm = messages.Text("trade.accept.title"), "trade_accept"
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
		msg = messages.Text("trade.action.completed")
	case "decline":
		_, err = s.svc.DeclineTransaction(r.Context(), c.actor, c.netID(), ref)
		msg = messages.Text("trade.action.declined")
	case "cancel":
		_, err = s.svc.CancelTransaction(r.Context(), c.actor, c.netID(), service.CancelInput{TxID: ref.TxID,
			Reason: r.PostFormValue("reason"), IdempotencyKey: ref.IdempotencyKey})
		msg = messages.Text("trade.action.cancelled")
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
			msg = messages.Text("trade.action.completed_balance", bv.Signed, bv.State)
		}
	}
	s.setFlash(w, r, msg)
	http.Redirect(w, r, c.base()+"/trades/"+a.Tx.ID, http.StatusSeeOther) //nolint:gosec // G710: fixed local prefix, server-generated id
}
