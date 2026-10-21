package web

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func (s *server) adminOverviewRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/admin/reverse", s.adminH(s.reverseForm))
	s.mux.HandleFunc("POST /n/{slug}/admin/reverse", s.adminH(s.reversePost))
	s.mux.HandleFunc("GET /n/{slug}/admin/announcements", s.adminH(s.announcementsPage))
	s.mux.HandleFunc("POST /n/{slug}/admin/announcements", s.adminH(s.announcementCreate))
	s.mux.HandleFunc("POST /n/{slug}/admin/announcements/{id}/delete", s.adminH(s.announcementDelete))
}

// attentionRow is one line of the administrator overview: a count in words, with a link when
// there is a page to act on it.
type attentionRow struct {
	Count int
	Text  string
	Href  string
	Label string // link text
}

type overviewData struct {
	Rows  []attentionRow
	AsOf  template.HTML
	Check string
	// Frozen is set while the network is read-only; FrozenReason is the administrator-only cause.
	Frozen       bool
	FrozenSince  template.HTML
	FrozenReason string
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// overview renders the administrator home: what needs attention, computed when the page loads
// (FR-067).
func (s *server) overview(w http.ResponseWriter, r *http.Request, c *rc) {
	ov, err := s.svc.AdminOverview(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	b := c.base() + "/admin"
	stale := int(ov.StaleAfter / time.Hour)
	soon := int(ov.InviteWithin / time.Hour)
	d := &overviewData{AsOf: c.when(ov.AsOf), Rows: []attentionRow{
		{ov.PendingMembers, plural(ov.PendingMembers, "membership request is waiting", "membership requests are waiting"), b + "/members?status=pending", "Review requests"},
		{ov.StaleProposals, plural(ov.StaleProposals, "open proposal has", "open proposals have") + " waited more than " + itoa(stale) + " hours for an answer", "", ""},
		{ov.OpenDisputes, plural(ov.OpenDisputes, "dispute is", "disputes are") + " open or under review", c.base() + "/review", "Open the review queue"},
		{ov.PostingPending, plural(ov.PostingPending, "dispute is", "disputes are") + " decided but the reversal is not yet posted", c.base() + "/review?status=posting_pending", "See them"},
		{ov.NearLimit, plural(ov.NearLimit, "active member is", "active members are") + " at or beyond their credit limit", b + "/members?status=active", "Open the member list"},
		{ov.ExpiringInvites, plural(ov.ExpiringInvites, "invitation ends", "invitations end") + " within " + itoa(soon) + " hours", b + "/invitations", "Open invitations"},
	}}
	if ov.Frozen != nil {
		d.Frozen, d.FrozenSince, d.FrozenReason = true, c.when(ov.Frozen.Since), ov.Frozen.Reason
	}
	switch {
	case ov.LastCheck == nil:
		d.Check = "No ledger check has been recorded yet."
	case ov.LastCheck.OK:
		d.Check = "The last ledger check passed."
	default:
		d.Check = "The last ledger check found problems."
	}
	if ov.LastCheck != nil {
		d.Check += " " + string(c.when(ov.LastCheck.At))
	}
	s.render(w, r, c, "admin_home", 200, &page{Title: "Network admin", D: d})
}

// ---- administrative reversal (FR-066) ----

type reverseForm struct {
	Trade  *reverseTrade
	Review *reverseReview
}

type reverseTrade struct {
	ID, Payer, Payee, Amount, Reversed, Remaining string
}

type reverseReview struct {
	reverseTrade
	Reverse, Reason, Unit   string
	PayerBefore, PayerAfter balanceVM
	PayeeBefore, PayeeAfter balanceVM
	PayerName, PayeeName    string
	Full                    bool
}

// reversibleTrade loads a trade an administrator may reverse and explains refusals in words.
func (c *rc) reversibleTrade(id string) (*service.TxView, *reverseTrade, string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil, "Trade reference: enter the reference of the completed trade to reverse. You can copy it from the trade page address."
	}
	v, err := c.s.svc.GetTransaction(c.ctx, c.actor, c.netID(), id)
	if err != nil {
		return nil, nil, "Trade reference: no trade with this reference was found in this network."
	}
	t := v.Tx
	n := c.memberNames()
	dec := c.Net.Policy.Decimals
	rt := &reverseTrade{ID: t.ID, Payer: n.of(t.PayerMemberID), Payee: n.of(t.PayeeMemberID), Amount: domain.FormatAmount(t.Amount, dec),
		Reversed: domain.FormatAmount(t.ReversedTotal, dec), Remaining: domain.FormatAmount(t.Amount-t.ReversedTotal, dec)}
	switch {
	case t.IsParty(c.Me.ID):
		return v, rt, "Trade reference: you took part in this trade, so you cannot reverse it. Ask another administrator."
	case t.Kind == domain.KindReversal:
		return v, rt, "Trade reference: this is itself a reversal, and a reversal cannot be reversed."
	case t.Status == domain.TxDisputed:
		return v, rt, "Trade reference: this trade is under dispute. A reviewer decides disputed trades from the review queue."
	case t.ReversedTotal >= t.Amount:
		return v, rt, "Trade reference: nothing is left to reverse on this trade."
	case t.Status != domain.TxCompleted:
		return v, rt, "Trade reference: only a completed trade can be reversed. This one is " + statusWord(t.Status) + "."
	}
	return v, rt, ""
}

func (s *server) reverseForm(w http.ResponseWriter, r *http.Request, c *rc) {
	id := r.URL.Query().Get("tx")
	p := &page{Title: "Reverse a trade", V: map[string]string{"tx": id}, D: &reverseForm{}}
	if id != "" {
		if _, rt, msg := c.reversibleTrade(id); rt != nil {
			p.D = &reverseForm{Trade: rt}
			if msg != "" {
				p.Summary, p.Errors = "This trade cannot be reversed.", []fieldErr{{"tx", msg}}
			}
		}
	}
	s.render(w, r, c, "admin_reverse", 200, p)
}

func (s *server) reversePost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: "Reverse a trade", V: v, D: &reverseForm{}}
	step := r.PostFormValue("step")
	view, rt, msg := c.reversibleTrade(v["tx"])
	if rt != nil {
		p.D = &reverseForm{Trade: rt}
	}
	if step == "back" {
		s.render(w, r, c, "admin_reverse", 200, p)
		return
	}
	dec := c.Net.Policy.Decimals
	var fe []fieldErr
	if msg != "" {
		fe = append(fe, fieldErr{"tx", msg})
	}
	var amt int64
	if view != nil && msg == "" {
		remaining := view.Tx.Amount - view.Tx.ReversedTotal
		amt = remaining
		if strings.TrimSpace(v["amount"]) != "" {
			a, err := domain.ParseAmount(v["amount"], dec)
			switch {
			case err != nil || a <= 0:
				fe = append(fe, fieldErr{"amount", c.amountHelp() + " Leave it empty to reverse everything that is left."})
			case a > remaining:
				fe = append(fe, fieldErr{"amount", "Amount: at most " + domain.FormatAmount(remaining, dec) + " is left to reverse on this trade."})
			default:
				amt = a
			}
		}
	}
	reason := strings.TrimSpace(v["reason"])
	switch {
	case reason == "":
		fe = append(fe, fieldErr{"reason", "Reason: a reason is required for every reversal. Write why it is needed."})
	case len([]rune(reason)) > 500:
		fe = append(fe, fieldErr{"reason", "Reason: this is too long. Use 500 characters or fewer."})
	}
	if len(fe) > 0 {
		p.Summary, p.Errors = "Not reversed. Fix the problems below. Nothing changed.", fe
		s.render(w, r, c, "admin_reverse", 422, p)
		return
	}
	t := view.Tx
	unit := c.Net.Policy.CurrencyName
	rv := &reverseReview{reverseTrade: *rt, Reverse: domain.FormatAmount(amt, dec), Reason: reason, Unit: unitFor(unit, amt, dec),
		PayerName: rt.Payer, PayeeName: rt.Payee, Full: amt == t.Amount-t.ReversedTotal}
	// Unaudited reads: the audited balance view is for pages that show a member's balance as the
	// point of the page; here the numbers only explain the effect of the reversal.
	pb, perr := s.svc.PeekMemberBalance(r.Context(), c.actor, c.netID(), t.PayerMemberID)
	yb, yerr := s.svc.PeekMemberBalance(r.Context(), c.actor, c.netID(), t.PayeeMemberID)
	if perr != nil || yerr != nil {
		s.fail(w, r, firstErr(perr, yerr))
		return
	}
	rv.PayerBefore, rv.PayerAfter = makeBalance(pb.Balance, dec, unit), makeBalance(pb.Balance+amt, dec, unit)
	rv.PayeeBefore, rv.PayeeAfter = makeBalance(yb.Balance, dec, unit), makeBalance(yb.Balance-amt, dec, unit)
	if step != "confirm" {
		p.Title, p.D = "Review reversal", &reverseForm{Trade: rt, Review: rv}
		s.render(w, r, c, "admin_reverse_review", 200, p)
		return
	}
	rev, err := s.svc.ReverseTransaction(r.Context(), c.actor, c.netID(), service.ReverseInput{TxID: t.ID, Amount: amt, Reason: reason,
		IdempotencyKey: idemKey(r)})
	if err != nil {
		p.Title, p.D = "Review reversal", &reverseForm{Trade: rt, Review: rv}
		if domain.CodeOf(err) == domain.CodeInsufficientCredit {
			p.Summary = "Not reversed. " + rt.Payee + " does not have enough available credit to give this amount back. Nothing changed."
			s.render(w, r, c, "admin_reverse_review", 422, p)
			return
		}
		if sum, f, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, f
			s.render(w, r, c, "admin_reverse_review", 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Reversal posted: "+rv.Reverse+" "+rv.Unit+" went back from "+rt.Payee+" to "+rt.Payer+".")
	http.Redirect(w, r, c.base()+"/trades/"+rev.ID, http.StatusSeeOther) //nolint:gosec // G710: fixed local prefix, id is a service-issued identifier
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// ---- announcements (API-064) ----

type annAdminRow struct {
	annRow
	Scheduled bool
}

type annAdminData struct{ Rows []annAdminRow }

func (s *server) announcementsPage(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderAnnouncements(w, r, c, 200, &page{})
}

func (s *server) renderAnnouncements(w http.ResponseWriter, r *http.Request, c *rc, status int, p *page) {
	as, err := s.svc.AdminListAnnouncements(r.Context(), c.actor, c.netID(), service.Page{Limit: 100})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &annAdminData{}
	now := s.opts.Clock.Now()
	for _, a := range as {
		d.Rows = append(d.Rows, annAdminRow{annRow{ID: a.ID, Title: a.Title, Body: a.Body, At: c.when(a.EffectiveAt)}, a.EffectiveAt.After(now)})
	}
	p.Title, p.D = "Announcements", d
	s.render(w, r, c, "admin_announcements", status, p)
}

func (s *server) announcementCreate(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{V: v}
	in := service.AnnouncementInput{Title: v["title"], Body: v["body"], IdempotencyKey: idemKey(r)}
	if d := strings.TrimSpace(v["days"]); d != "" {
		n, ok := atoiRange(d, 0, 365)
		if !ok {
			p.Summary, p.Errors = "Not published. Fix the problems below. Nothing changed.",
				[]fieldErr{{"days", "Start in (days): enter a whole number from 0 to 365, or leave it empty to publish now."}}
			s.renderAnnouncements(w, r, c, 422, p)
			return
		}
		in.EffectiveAt = s.opts.Clock.Now().Add(time.Duration(n) * 24 * time.Hour)
	}
	if _, err := s.svc.CreateAnnouncement(r.Context(), c.actor, c.netID(), in); err != nil {
		if sum, fe, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			s.renderAnnouncements(w, r, c, 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Announcement published.")
	http.Redirect(w, r, c.base()+"/admin/announcements", http.StatusSeeOther)
}

func (s *server) announcementDelete(w http.ResponseWriter, r *http.Request, c *rc) {
	if err := s.svc.DeleteAnnouncement(r.Context(), c.actor, c.netID(), r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, "Announcement deleted. Members no longer see it; the audit log keeps the record.")
	http.Redirect(w, r, c.base()+"/admin/announcements", http.StatusSeeOther)
}

func atoiRange(s string, lo, hi int) (int, bool) {
	n := 0
	if s == "" || len(s) > 4 {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, n >= lo && n <= hi
}
