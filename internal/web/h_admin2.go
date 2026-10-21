package web

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

var moneyFields = []string{"default_credit_limit", "max_tx_amount", "signup_grant"}

func policyValues(p domain.Policy) map[string]string {
	d := p.Decimals
	v := map[string]string{
		"currency_name": p.CurrencyName, "default_credit_limit": domain.FormatAmount(p.DefaultCreditLimit, d),
		"max_tx_amount": domain.FormatAmount(p.MaxTxAmount, d), "signup_grant": domain.FormatAmount(p.SignupGrant, d),
		"proposal_expiry_days": itoa(p.ProposalExpiryDays), "dispute_window_days": itoa(p.DisputeWindowDays),
		"limit_warning_days": itoa(p.LimitWarningDays), "join_mode": string(p.JoinMode), "departure_rule": string(p.DepartureRule),
	}
	if p.DefaultFlagBlocksPayer {
		v["default_flag_blocks_payer"] = "1"
	}
	return v
}

func (s *server) policyForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.render(w, r, c, "admin_policy", 200, &page{Title: "Network policy", V: policyValues(c.Net.Policy), D: c.Net.PolicyVersion})
}

func (s *server) policyPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: "Network policy", V: v, D: c.Net.PolicyVersion}
	pol := c.Net.Policy
	dec := pol.Decimals
	var fe []fieldErr
	pol.CurrencyName = strings.TrimSpace(v["currency_name"])
	pol.JoinMode, pol.DepartureRule = domain.JoinMode(v["join_mode"]), domain.DepartureRule(v["departure_rule"])
	pol.DefaultFlagBlocksPayer = v["default_flag_blocks_payer"] == "1"
	dst := map[string]*int64{"default_credit_limit": &pol.DefaultCreditLimit, "max_tx_amount": &pol.MaxTxAmount, "signup_grant": &pol.SignupGrant}
	for _, f := range moneyFields {
		n, err := domain.ParseAmount(v[f], dec)
		if err != nil {
			fe = append(fe, fieldErr{f, fmt.Sprintf("%s: enter a number with up to %d decimal places, for example %s.", label(f), dec, domain.FormatAmount(1250, dec))})
			continue
		}
		*dst[f] = n
	}
	days := map[string]*int{"proposal_expiry_days": &pol.ProposalExpiryDays, "dispute_window_days": &pol.DisputeWindowDays, "limit_warning_days": &pol.LimitWarningDays}
	for f, d := range days {
		n, err := strconv.Atoi(strings.TrimSpace(v[f]))
		if err != nil {
			fe = append(fe, fieldErr{f, label(f) + ": enter a whole number."})
			continue
		}
		*d = n
	}
	if len(fe) > 0 {
		p.Summary, p.Errors = "Not saved. Fix the problems below. Nothing changed.", fe
		s.render(w, r, c, "admin_policy", 422, p)
		return
	}
	_, err := s.svc.UpdatePolicy(r.Context(), c.actor, c.netID(), service.PolicyChange{Policy: pol, Reason: strings.TrimSpace(v["reason"]), IdempotencyKey: idemKey(r)})
	if err != nil {
		s.formFail(w, r, c, "admin_policy", p, err)
		return
	}
	s.setFlash(w, r, "Policy saved as a new version.")
	http.Redirect(w, r, c.base()+"/admin/policy", http.StatusSeeOther)
}

type adjustData struct {
	Members []option
	Review  *adjustReview
}

type adjustReview struct {
	Name, Dir, Amount, Reason string
	Before, After             balanceVM
}

// activeMemberOptions lists every active member except the caller, paging without a cap.
func (c *rc) activeMemberOptions() []option {
	o := []option{{"", "Choose a member"}}
	after := ""
	for {
		pg, err := c.s.svc.AdminMemberPage(c.ctx, c.actor, c.netID(), []domain.MemberStatus{domain.MemberActive},
			service.MemberPageQuery{AfterID: after, Limit: 500})
		if err != nil {
			return o
		}
		for _, m := range pg.Members {
			if m.ID != c.Me.ID {
				o = append(o, option{m.ID, m.DisplayName})
			}
		}
		if !pg.HasNext || len(pg.Members) == 0 {
			return o
		}
		after = pg.Members[len(pg.Members)-1].ID
	}
}

func (s *server) adjustForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderAdjust(w, r, c, 200, &page{V: map[string]string{"direction": "credit"}})
}

func (s *server) renderAdjust(w http.ResponseWriter, r *http.Request, c *rc, st int, p *page) {
	p.Title = "Adjustments"
	if d, ok := p.D.(*adjustData); ok {
		d.Members = c.activeMemberOptions()
	} else {
		p.D = &adjustData{Members: c.activeMemberOptions()}
	}
	s.render(w, r, c, "admin_adjust", st, p)
}

func (s *server) adjustPost(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{V: v}
	step := r.PostFormValue("step")
	if step == "back" {
		s.renderAdjust(w, r, c, 200, p)
		return
	}
	dec := c.Net.Policy.Decimals
	var fe []fieldErr
	name, _ := c.activeOther(v["member_id"])
	if name == "" {
		fe = append(fe, fieldErr{"member_id", "Member: choose a member other than yourself."})
	}
	dir := service.AdjustDirection(v["direction"])
	if dir != service.AdjustCredit && dir != service.AdjustDebit {
		fe = append(fe, fieldErr{"direction", "Direction: choose credit or debit."})
	}
	amt, err := domain.ParseAmount(v["amount"], dec)
	if err != nil || amt <= 0 {
		fe = append(fe, fieldErr{"amount", c.amountHelp()})
	}
	reason := strings.TrimSpace(v["reason"])
	if reason == "" {
		fe = append(fe, fieldErr{"reason", "Reason: a reason is required for every adjustment. Write why it is needed."})
	}
	if len(fe) > 0 {
		p.Summary, p.Errors = "Not posted. Fix the problems below. Nothing changed.", fe
		s.renderAdjust(w, r, c, 422, p)
		return
	}
	b, err := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), v["member_id"])
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rv := &adjustReview{Name: name, Amount: domain.FormatAmount(amt, dec), Reason: reason,
		Before: makeBalance(b.Balance, dec, c.Net.Policy.CurrencyName)}
	after := b.Balance + amt
	rv.Dir = "Credit (balance goes up)"
	if dir == service.AdjustDebit {
		after, rv.Dir = b.Balance-amt, "Debit (balance goes down)"
	}
	rv.After = makeBalance(after, dec, c.Net.Policy.CurrencyName)
	if step != "confirm" {
		p.Title, p.D = "Review adjustment", rv
		s.render(w, r, c, "admin_adjust_review", 200, p)
		return
	}
	res, err := s.svc.CreateAdjustmentRequest(r.Context(), c.actor, c.netID(), service.AdjustInput{MemberID: v["member_id"], Amount: amt,
		Direction: dir, Reason: reason, IdempotencyKey: idemKey(r)})
	if err != nil {
		if sum, f, ok := c.refuse(err); ok {
			p.Summary, p.Errors, p.Title, p.D = sum, f, "Review adjustment", rv
			s.render(w, r, c, "admin_adjust_review", 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	if res.Pending != nil { // SEC-011: nothing is posted until a second administrator approves
		s.setFlash(w, r, fmt.Sprintf("Adjustment for %s is above the dual-control threshold. Nothing is posted yet: another administrator must approve it before it expires.", name))
		http.Redirect(w, r, c.base()+"/admin/adjustments/pending", http.StatusSeeOther)
		return
	}
	s.setFlash(w, r, fmt.Sprintf("Adjustment posted for %s. Their balance is now %s · %s.", name, rv.After.Signed, rv.After.State))
	http.Redirect(w, r, c.base()+"/admin/adjust", http.StatusSeeOther)
}

type auditRow struct {
	Seq                                int64
	When                               any
	Actor, Action, Target, Reason, Req string
}

func (s *server) auditPage(w http.ResponseWriter, r *http.Request, c *rc) {
	q := r.URL.Query()
	after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
	evs, err := s.svc.ListAudit(r.Context(), c.actor, c.netID(), service.AuditQuery{Action: strings.TrimSpace(q.Get("action")), AfterSeq: after, Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	var rows []auditRow
	for _, e := range evs {
		who := e.ActorID
		if nm, ok := n.lookup(e.ActorID); ok {
			who = nm
		} else if e.ActorType == domain.ActorSystem || who == "" {
			who = "System"
		}
		rows = append(rows, auditRow{e.Seq, c.when(e.OccurredAt), who, e.Action, e.TargetType + " " + e.TargetID, e.Reason, e.RequestID})
	}
	next := ""
	if len(evs) == 50 {
		next = strconv.FormatInt(evs[49].Seq, 10)
	}
	s.render(w, r, c, "admin_audit", 200, &page{Title: "Audit and checks", D: map[string]any{"Rows": rows, "Next": next},
		V: map[string]string{"action": q.Get("action")}})
}

func (s *server) ledgerCheck(w http.ResponseWriter, r *http.Request, c *rc) {
	rep, err := s.svc.VerifyAndRecordLedger(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "admin_check", 200, &page{Title: "Consistency check result", D: rep})
}

func (s *server) exportsPage(w http.ResponseWriter, r *http.Request, c *rc) {
	s.render(w, r, c, "admin_exports", 200, &page{Title: "Exports", V: map[string]string{"scope": "ledger", "format": "csv"}})
}

// zipSink streams export files into one zip response; headers are sent on the
// first file so that early refusals can still render an error page.
type zipSink struct {
	w    http.ResponseWriter
	name string
	zw   *zip.Writer
}

func (z *zipSink) Create(name string) (io.WriteCloser, error) {
	if z.zw == nil {
		z.w.Header().Set("Content-Type", "application/zip")
		z.w.Header().Set("Content-Disposition", `attachment; filename="`+z.name+`"`)
		z.zw = zip.NewWriter(z.w)
	}
	w, err := z.zw.Create(name)
	if err != nil {
		return nil, err
	}
	return nopCloser{w}, nil
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func (s *server) sendExport(w http.ResponseWriter, r *http.Request, c *rc, scope service.ExportScope, f service.ExportFormat) {
	if f != service.FormatCSV && f != service.FormatJSON {
		s.fail(w, r, domain.NotFound("format"))
		return
	}
	z := &zipSink{w: w, name: fmt.Sprintf("circula-%s-%s-%s.zip", c.Net.Slug, scope, f)}
	_, err := s.svc.Export(r.Context(), c.actor, c.netID(), scope, f, z)
	if z.zw != nil {
		_ = z.zw.Close()
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.fail(w, r, domain.Errf(domain.CodeInternal, "empty export"))
}

func (s *server) exportsPost(w http.ResponseWriter, r *http.Request, c *rc) {
	scope := service.ExportScope(r.PostFormValue("scope"))
	switch scope {
	case service.ExportLedger, service.ExportMembers, service.ExportAudit, service.ExportFull:
	default:
		s.fail(w, r, domain.NotFound("scope"))
		return
	}
	s.sendExport(w, r, c, scope, service.ExportFormat(r.PostFormValue("format")))
}

func (s *server) exportMine(w http.ResponseWriter, r *http.Request, c *rc) {
	s.sendExport(w, r, c, service.ExportMine, service.ExportFormat(r.PostFormValue("format")))
}
