package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) tradeRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/trades", s.netH(s.trades))
	s.mux.HandleFunc("GET /n/{slug}/trades/new", s.netH(s.proposeForm))
	s.mux.HandleFunc("POST /n/{slug}/trades/new", s.netH(s.proposePost))
	s.mux.HandleFunc("GET /n/{slug}/trades/{id}", s.netH(s.tradeDetail))
	s.mux.HandleFunc("GET /n/{slug}/trades/{id}/{action}", s.netH(s.tradeActionForm))
	s.mux.HandleFunc("POST /n/{slug}/trades/{id}/{action}", s.netH(s.tradeActionPost))
}

type option struct{ Value, Label string }

type tradesData struct {
	Needs, Waiting, Past []txRow
	Filter               string
	Next                 string
}

func (s *server) trades(w http.ResponseWriter, r *http.Request, c *rc) {
	f := r.URL.Query().Get("status")
	var st []domain.TxStatus
	for _, x := range domain.AllTxStatuses {
		if string(x) == f && x != domain.TxAccepted {
			st = []domain.TxStatus{x}
		}
	}
	txs, err := s.svc.ListTransactions(r.Context(), c.actor, c.netID(), st, service.Page{AfterID: r.URL.Query().Get("after"), Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	d := &tradesData{Filter: f}
	for _, t := range txs {
		if t.Kind != domain.KindTrade {
			continue
		}
		row := c.row(t, n)
		switch {
		case row.Needs:
			d.Needs = append(d.Needs, row)
		case row.Pending:
			d.Waiting = append(d.Waiting, row)
		default:
			d.Past = append(d.Past, row)
		}
	}
	if len(txs) == 50 {
		d.Next = txs[49].ID
	}
	s.render(w, r, c, "trades", 200, &page{Title: "Trades", D: d, V: map[string]string{"status": f}})
}

type proposeData struct {
	Members []option
	Review  *review
}

type review struct {
	Sentence   string
	Counter    string
	Amount     string
	Unit       string
	Before     balanceVM
	After      balanceVM
	AvailNow   string
	AvailAfter string
	Reserve    bool
	Expires    int
	TooMuch    bool
}

func (c *rc) counterpartyOptions() []option {
	var o []option
	all, err := c.activeMembers()
	if err != nil {
		return nil
	}
	for _, e := range all {
		if e.MemberID != c.Me.ID {
			o = append(o, option{e.MemberID, e.DisplayName})
		}
	}
	return o
}

func (s *server) proposeForm(w http.ResponseWriter, r *http.Request, c *rc) {
	q := r.URL.Query()
	v := map[string]string{"counterparty_id": q.Get("to"), "direction": "pay", "listing_id": q.Get("listing")}
	if q.Get("direction") == "request" {
		v["direction"] = "request"
	}
	s.renderPropose(w, r, c, 200, &page{Title: "Propose a trade", V: v})
}

func (s *server) renderPropose(w http.ResponseWriter, r *http.Request, c *rc, st int, p *page) {
	opts := c.counterpartyOptions()
	p.D = &proposeData{Members: append([]option{{"", "Choose a member"}}, opts...)}
	p.Warn = c.limitNotice()
	s.render(w, r, c, "propose", st, p)
}

func (c *rc) amountHelp() string {
	d := c.Net.Policy.Decimals
	return fmt.Sprintf("Amount: enter a number with up to %d decimal places, for example %s.", d, domain.FormatAmount(1250, d))
}

func (s *server) proposePost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: "Propose a trade", V: v}
	step := r.PostFormValue("step")
	if step == "back" {
		s.renderPropose(w, r, c, 200, p)
		return
	}
	dec := c.Net.Policy.Decimals
	var fe []fieldErr
	cp, _ := c.activeOther(v["counterparty_id"])
	if cp == "" {
		fe = append(fe, fieldErr{"counterparty_id", "Counterparty: choose another active member from the list."})
	}
	dir := service.Direction(v["direction"])
	if dir != service.ProposerPays && dir != service.ProposerIsPaid {
		fe = append(fe, fieldErr{"direction", "Direction: choose whether you pay or are paid."})
	}
	amount, err := domain.ParseAmount(v["amount"], dec)
	if err != nil || amount <= 0 {
		msg := c.amountHelp()
		if de, ok := err.(*domain.Error); ok && len(de.Fields) > 0 && strings.Contains(de.Fields[0].Detail, "decimal") {
			msg = fmt.Sprintf("Amount: this network allows at most %d decimal places. Remove the extra digits.", dec)
		}
		fe = append(fe, fieldErr{"amount", msg})
	}
	if len(fe) > 0 {
		p.Errors, p.Summary = fe, "Not sent. Fix the problems below. Nothing changed."
		s.renderPropose(w, r, c, 422, p)
		return
	}
	if step != "confirm" {
		rv, err := c.buildReview(cp, dir, amount)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		p.Title = "Review your trade"
		p.D = rv
		s.render(w, r, c, "propose_review", 200, p)
		return
	}
	t, err := s.svc.ProposeTransaction(r.Context(), c.actor, c.netID(), service.ProposeInput{CounterpartyID: v["counterparty_id"],
		Direction: dir, Amount: amount, ListingID: v["listing_id"], Note: v["note"], IdempotencyKey: idemKey(r)})
	if err != nil {
		if sum, f, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, f
			rv, _ := c.buildReview(cp, dir, amount)
			p.Title, p.D = "Review your trade", rv
			if domain.CodeOf(err) == domain.CodeValidation {
				s.renderPropose(w, r, c, 422, p)
				return
			}
			s.render(w, r, c, "propose_review", 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, fmt.Sprintf("Trade sent to %s. It is pending until they respond; pending trades are not part of your balance.", cp))
	http.Redirect(w, r, c.base()+"/trades/"+t.ID, http.StatusSeeOther)
}

func (c *rc) buildReview(cp string, dir service.Direction, amount int64) (*review, error) {
	b, err := c.s.svc.GetBalance(c.ctx, c.actor, c.netID())
	if err != nil {
		return nil, err
	}
	pol := c.Net.Policy
	u := unitFor(pol.CurrencyName, amount, pol.Decimals)
	a := domain.FormatAmount(amount, pol.Decimals)
	rv := &review{Counter: cp, Amount: a, Unit: u, Before: makeBalance(b.Balance, b.Decimals, pol.CurrencyName),
		AvailNow: minus(plainAmt(b.Available, b.Decimals)), Expires: pol.ProposalExpiryDays}
	if dir == service.ProposerPays {
		rv.Sentence = fmt.Sprintf("You pay %s %s %s", cp, a, u)
		rv.After = makeBalance(b.Balance-amount, b.Decimals, pol.CurrencyName)
		rv.AvailAfter = minus(plainAmt(b.Available-amount, b.Decimals))
		rv.Reserve = true
		rv.TooMuch = amount > b.Available
	} else {
		rv.Sentence = fmt.Sprintf("%s pays you %s %s", cp, a, u)
		rv.After = makeBalance(b.Balance+amount, b.Decimals, pol.CurrencyName)
		rv.AvailAfter = minus(plainAmt(b.Available+amount, b.Decimals))
	}
	return rv, nil
}

type tradeDetail struct {
	Tx        *domain.Transaction
	Row       txRow
	Payer     string
	Payee     string
	Explain   string
	History   []histRow
	Accept    bool
	Decline   bool
	Cancel    bool
	Dispute   bool
	DisputeID string
	WhyNot    string
	Party     bool
	Reverse   bool // an administrator who is not a party may reverse it (FR-066)
	// Times in the viewer's display zone.
	Proposed, Expires, Posted template.HTML
}

type histRow struct {
	Label string
	At    any
}

func (s *server) tradeDetail(w http.ResponseWriter, r *http.Request, c *rc) {
	v, err := s.svc.GetTransaction(r.Context(), c.actor, c.netID(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	t := v.Tx
	d := &tradeDetail{Tx: t, Payer: n.of(t.PayerMemberID), Payee: n.of(t.PayeeMemberID), Explain: statusExplain(t.Status),
		Party: t.IsParty(c.Me.ID)}
	if d.Party {
		d.Row = c.row(t, n)
	} else {
		d.Row.Status, _ = statusLabel(t, c.Me, n)
	}
	for _, h := range v.History {
		d.History = append(d.History, histRow{Label: historyWord(h.From, h.To), At: c.when(h.OccurredAt)})
	}
	if t.Status == domain.TxProposed {
		switch {
		case d.Party && t.CreatedBy != c.Me.ID && c.isActive():
			d.Accept, d.Decline = true, true
		case d.Party && t.CreatedBy != c.Me.ID && c.Me.Status == domain.MemberSuspended:
			d.Decline = true
			d.WhyNot = "Accept is not available while your membership is suspended. You can still decline."
		case d.Party && t.CreatedBy == c.Me.ID:
			d.Cancel = true
		case !d.Party && c.isAdmin():
			d.Cancel = true
		}
	}
	if t.Status == domain.TxCompleted && t.Kind == domain.KindTrade && d.Party {
		if c.isActive() {
			d.Dispute = true
		} else {
			d.WhyNot = "Raising a dispute is not available while your membership is not active."
		}
	}
	d.Proposed = c.when(t.ProposedAt)
	if t.ExpiresAt != nil {
		d.Expires = c.when(*t.ExpiresAt)
	}
	if t.CompletedAt != nil {
		d.Posted = c.when(*t.CompletedAt)
	}
	d.Reverse = c.isAdmin() && !d.Party && t.Kind != domain.KindReversal && t.Status == domain.TxCompleted && t.ReversedTotal < t.Amount
	if t.Status == domain.TxDisputed && (d.Party || c.isAdmin() || c.isReviewer()) {
		if dv, err := s.svc.GetDisputeForTx(r.Context(), c.actor, c.netID(), t.ID); err == nil {
			d.DisputeID = dv.Dispute.ID
		}
	}
	s.render(w, r, c, "trade_detail", 200, &page{Title: "Trade detail", D: d})
}

func historyWord(from, to domain.TxStatus) string {
	w := statusWord(to)
	if from == "" {
		return "Proposed"
	}
	return w
}

func statusWord(s domain.TxStatus) string {
	switch s {
	case domain.TxProposed:
		return "Pending"
	case domain.TxAccepted, domain.TxCompleted:
		return "Completed"
	}
	t := string(s)
	return strings.ToUpper(t[:1]) + t[1:]
}
