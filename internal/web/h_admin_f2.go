package web

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func (s *server) adminOverviewRoutes() {
	s.adminNetworkRoutes()
	s.mux.HandleFunc("GET /n/{slug}/admin/proposals/stale", s.adminH(s.staleProposals))
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
	Rows             []attentionRow
	AsOf             template.HTML
	Check            string
	CheckAt          template.HTML
	HasCheck         bool
	HistoricalLimits *service.HistoricalLimitCoverage
	// Frozen is set while the network is read-only; FrozenReason is the administrator-only cause.
	Frozen       bool
	FrozenSince  template.HTML
	FrozenReason string
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
		{ov.PendingMembers, messages.Text("overview.pending", ov.PendingMembers), b + "/members?status=pending", messages.Text("overview.review_requests")},
		{ov.StaleProposals, messages.Text("overview.stale", ov.StaleProposals, stale), b + "/proposals/stale", messages.Text("overview.review_waiting")},
		{ov.OpenDisputes, messages.Text("overview.disputes", ov.OpenDisputes), c.base() + "/review", messages.Text("overview.review_queue")},
		{ov.PostingPending, messages.Text("overview.posting", ov.PostingPending), c.base() + "/review?status=posting_pending", messages.Text("overview.see_posting")},
		{ov.NearLimit, messages.Text("overview.limit", ov.NearLimit), b + "/members?status=active", messages.Text("overview.members")},
		{ov.ExpiringInvites, messages.Text("overview.invites", ov.ExpiringInvites, soon), b + "/invitations", messages.Text("overview.invitations")},
	}}
	if ov.Frozen != nil {
		d.Frozen, d.FrozenSince, d.FrozenReason = true, c.when(ov.Frozen.Since), ov.Frozen.Reason
	}
	switch {
	case ov.LastCheck == nil:
		d.Check = messages.Text("overview.check_none")
	case ov.LastCheck.OK:
		d.Check = messages.Text("overview.check_ok")
	default:
		d.Check = messages.Text("overview.check_failed")
	}
	if ov.LastCheck != nil {
		d.CheckAt = c.when(ov.LastCheck.At)
		d.HasCheck, d.HistoricalLimits = true, ov.LastCheck.HistoricalLimits
	}
	s.render(w, r, c, "admin_home", 200, &page{Title: messages.Text("overview.title"), D: d})
}

func (s *server) staleProposals(w http.ResponseWriter, r *http.Request, c *rc) {
	txs, err := s.svc.AdminStaleProposals(r.Context(), c.actor, c.netID(), service.Page{AfterID: r.URL.Query().Get("after"), Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := struct {
		Rows  []txRow
		Next  string
		Hours int
		AsOf  template.HTML
	}{Hours: int(service.OverviewStaleAfter / time.Hour), AsOf: c.when(s.opts.Clock.Now())}
	n := c.memberNames()
	for _, tx := range txs {
		row := c.row(tx, n)
		amount := domain.FormatAmount(tx.Amount, c.Net.Policy.Decimals)
		// This is a network-wide queue: the viewer need not be either party.
		row.Text = messages.Text("queue.party_proposal", n.of(tx.PayerMemberID), n.of(tx.PayeeMemberID), amount, unitFor(c.Net.Policy.CurrencyName, tx.Amount, c.Net.Policy.Decimals))
		row.Signed = amount
		row.Status = messages.Text("queue.waiting")
		d.Rows = append(d.Rows, row)
	}
	if len(txs) == 50 {
		d.Next = txs[len(txs)-1].ID
	}
	s.render(w, r, c, "admin_stale_proposals", 200, &page{Title: messages.Text("queue.title"), D: d})
}

// ---- administrative reversal (FR-066) ----

type reverseForm struct {
	Trade  *reverseTrade
	Review *reverseReview
}

type reverseTrade struct {
	ID, Payer, Payee, Amount, Reversed, Remaining string
	AmountMinor, ReversedMinor, RemainingMinor    int64
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
		return nil, nil, messages.Text("admin.reverse.reference_required")
	}
	v, err := c.s.svc.GetTransaction(c.ctx, c.actor, c.netID(), id)
	if err != nil {
		return nil, nil, messages.Text("admin.reverse.reference_missing")
	}
	t := v.Tx
	n := c.memberNames()
	dec := c.Net.Policy.Decimals
	rt := &reverseTrade{ID: t.ID, Payer: n.of(t.PayerMemberID), Payee: n.of(t.PayeeMemberID), Amount: domain.FormatAmount(t.Amount, dec),
		Reversed: domain.FormatAmount(t.ReversedTotal, dec), Remaining: domain.FormatAmount(t.Amount-t.ReversedTotal, dec),
		AmountMinor: t.Amount, ReversedMinor: t.ReversedTotal, RemainingMinor: t.Amount - t.ReversedTotal}
	switch {
	case t.IsParty(c.Me.ID):
		return v, rt, messages.Text("admin.reverse.reference_party")
	case t.Kind == domain.KindReversal:
		return v, rt, messages.Text("admin.reverse.reference_reversal")
	case t.Status == domain.TxDisputed:
		return v, rt, messages.Text("admin.reverse.reference_disputed")
	case t.ReversedTotal >= t.Amount:
		return v, rt, messages.Text("admin.reverse.reference_empty")
	case t.Status != domain.TxCompleted:
		return v, rt, messages.Text("admin.reverse.reference_status", statusWord(t.Status))
	}
	return v, rt, ""
}

func (s *server) reverseForm(w http.ResponseWriter, r *http.Request, c *rc) {
	id := r.URL.Query().Get("tx")
	p := &page{Title: messages.Text("admin.reverse.title"), V: map[string]string{"tx": id}, D: &reverseForm{}}
	if id != "" {
		if _, rt, msg := c.reversibleTrade(id); rt != nil {
			p.D = &reverseForm{Trade: rt}
			if msg != "" {
				p.Summary, p.Errors = messages.Text("admin.reverse.unavailable"), []fieldErr{{"tx", msg}}
			}
		}
	}
	s.render(w, r, c, "admin_reverse", 200, p)
}

func (s *server) reversePost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: messages.Text("admin.reverse.title"), V: v, D: &reverseForm{}}
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
				fe = append(fe, fieldErr{"amount", messages.Text("admin.reverse.amount_help", dec, domain.FormatAmount(1250, dec))})
			case a > remaining:
				fe = append(fe, fieldErr{"amount", messages.Text("admin.reverse.amount_max", domain.FormatAmount(remaining, dec))})
			default:
				amt = a
			}
		}
	}
	reason := strings.TrimSpace(v["reason"])
	switch {
	case reason == "":
		fe = append(fe, fieldErr{"reason", messages.Text("admin.reverse.reason_required")})
	case len([]rune(reason)) > 500:
		fe = append(fe, fieldErr{"reason", messages.Text("admin.reverse.reason_long")})
	}
	if len(fe) > 0 {
		p.Summary, p.Errors = messages.Text("admin.reverse.invalid"), fe
		s.render(w, r, c, "admin_reverse", 422, p)
		return
	}
	t := view.Tx
	unit := c.Net.Policy.CurrencyName
	rv := &reverseReview{reverseTrade: *rt, Reverse: domain.FormatAmount(amt, dec), Reason: reason, Unit: unitFor(unit, amt, dec),
		PayerName: rt.Payer, PayeeName: rt.Payee, Full: amt == t.Amount-t.ReversedTotal}
	// The confirmation renders both members' current and resulting balances.
	// Each disclosure uses the audited administrator balance read (DAT-034).
	pb, perr := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), t.PayerMemberID)
	yb, yerr := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), t.PayeeMemberID)
	if perr != nil || yerr != nil {
		s.fail(w, r, firstErr(perr, yerr))
		return
	}
	rv.PayerBefore, rv.PayerAfter = makeBalance(pb.Balance, dec, unit), makeBalance(pb.Balance+amt, dec, unit)
	rv.PayeeBefore, rv.PayeeAfter = makeBalance(yb.Balance, dec, unit), makeBalance(yb.Balance-amt, dec, unit)
	if step != "confirm" {
		p.Title, p.D = messages.Text("admin.reverse.review_title"), &reverseForm{Trade: rt, Review: rv}
		s.render(w, r, c, "admin_reverse_review", 200, p)
		return
	}
	rev, err := s.svc.ReverseTransaction(r.Context(), c.actor, c.netID(), service.ReverseInput{TxID: t.ID, Amount: amt, Reason: reason,
		IdempotencyKey: idemKey(r)})
	if err != nil {
		p.Title, p.D = messages.Text("admin.reverse.review_title"), &reverseForm{Trade: rt, Review: rv}
		if domain.CodeOf(err) == domain.CodeInsufficientCredit {
			p.Summary = messages.Text("admin.reverse.insufficient_credit", rt.Payee)
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
	s.setFlash(w, r, messages.Text("admin.reverse.posted", rv.Reverse, rv.Unit, rt.Payee, rt.Payer))
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

type annAdminData struct {
	Rows    []annAdminRow
	Options []option
}

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
		d.Options = append(d.Options, option{Value: a.ID, Label: a.Title})
		d.Rows = append(d.Rows, annAdminRow{annRow{ID: a.ID, Title: a.Title, Body: a.Body, At: c.when(a.EffectiveAt)}, a.EffectiveAt.After(now)})
	}
	p.Title, p.D = messages.Text("admin.announcements.title"), d
	s.render(w, r, c, "admin_announcements", status, p)
}

func (s *server) announcementCreate(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{V: v}
	in := service.AnnouncementInput{Title: v["title"], Body: v["body"], SupersedesID: v["supersedes_id"], IdempotencyKey: idemKey(r)}
	if d := strings.TrimSpace(v["days"]); d != "" {
		n, ok := atoiRange(d, 0, 365)
		if !ok {
			p.Summary, p.Errors = messages.Text("admin.announcements.invalid"),
				[]fieldErr{{"days", messages.Text("admin.announcements.days_invalid")}}
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
	s.setFlash(w, r, messages.Text("admin.announcements.published"))
	http.Redirect(w, r, c.base()+"/admin/announcements", http.StatusSeeOther)
}

func (s *server) announcementDelete(w http.ResponseWriter, r *http.Request, c *rc) {
	if err := s.svc.DeleteAnnouncement(r.Context(), c.actor, c.netID(), r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, messages.Text("admin.announcements.deleted"))
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
