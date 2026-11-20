package web

import (
	"html/template"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

func (s *server) disputeRoutes() {
	s.mux.HandleFunc("GET /n/{slug}/disputes", s.netH(s.disputes))
	s.mux.HandleFunc("GET /n/{slug}/review", s.netH(s.reviewQueue))
	s.mux.HandleFunc("GET /n/{slug}/trades/{id}/dispute", s.netH(s.disputeForm))
	s.mux.HandleFunc("POST /n/{slug}/trades/{id}/dispute", s.netH(s.disputePost))
	s.mux.HandleFunc("GET /n/{slug}/disputes/{id}", s.netH(s.disputeView))
	s.mux.HandleFunc("GET /n/{slug}/disputes/{id}/reports", s.netH(s.disputeReports))
	s.mux.HandleFunc("POST /n/{slug}/disputes/{id}/{act}", s.netH(s.disputeAct))
}

var categories = []option{
	{"not_delivered", messages.Text("dispute.category.not_delivered")}, {"quality", messages.Text("dispute.category.quality")},
	{"wrong_amount", messages.Text("dispute.category.wrong_amount")}, {"unauthorised", messages.Text("dispute.category.unauthorised")}, {"other", messages.Text("dispute.category.other")},
}

func categoryLabel(v string) string {
	for _, o := range categories {
		if o.Value == v {
			return o.Label
		}
	}
	return v
}

type disputeRow struct {
	ID, Category, Status, Tx string
	Opened                   any
}

func statusWords(s domain.DisputeStatus) string {
	switch s {
	case domain.DisputeOpen:
		return messages.Text("dispute.status.open")
	case domain.DisputeUnderReview:
		return messages.Text("dispute.status.under_review")
	case domain.DisputePostingPending:
		return messages.Text("dispute.status.posting_pending")
	case domain.DisputeResolved:
		return messages.Text("dispute.status.resolved")
	case domain.DisputeWithdrawn:
		return messages.Text("dispute.status.withdrawn")
	}
	return string(s)
}

func (c *rc) disputeRows(ds []*domain.Dispute) []disputeRow {
	var out []disputeRow
	for _, d := range ds {
		out = append(out, disputeRow{ID: d.ID, Category: categoryLabel(d.Category), Status: statusWords(d.Status), Tx: d.TxID, Opened: c.when(d.OpenedAt)})
	}
	return out
}

// disputeListData is one page of disputes. Next is the cursor of the following page, "" at the end.
type disputeListData struct {
	Rows          []disputeRow
	Next          string
	Filter        string
	TransactionID string
}

// disputePage reads one page of disputes (newest first) with a status filter. It asks for one row
// more than the page size to learn whether a further page exists, so no fixed window hides
// older disputes (WEB-02).
func (c *rc) disputePage(r *http.Request, statuses []domain.DisputeStatus, txID string) (*disputeListData, error) {
	ds, err := c.s.svc.FindDisputes(r.Context(), c.actor, c.netID(), service.DisputeQuery{Statuses: statuses,
		TxID: txID, Page: service.Page{AfterID: r.URL.Query().Get("after"), Limit: perPage + 1}})
	if err != nil {
		return nil, err
	}
	d := &disputeListData{TransactionID: txID}
	if len(ds) > perPage {
		ds = ds[:perPage]
		d.Next = ds[perPage-1].ID
	}
	d.Rows = c.disputeRows(ds)
	return d, nil
}

func (s *server) disputes(w http.ResponseWriter, r *http.Request, c *rc) {
	d, err := c.disputePage(r, nil, r.URL.Query().Get("transaction_id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "disputes", 200, &page{Title: messages.Text("dispute.list_title"), D: d})
}

var queueFilters = []option{
	{"", messages.Text("dispute.queue.all")}, {string(domain.DisputeOpen), messages.Text("dispute.queue.waiting")},
	{string(domain.DisputeUnderReview), messages.Text("dispute.status.under_review")}, {string(domain.DisputePostingPending), messages.Text("dispute.status.posting_pending")},
}

type queueData struct {
	*disputeListData
	Filters []option
}

func (s *server) reviewQueue(w http.ResponseWriter, r *http.Request, c *rc) {
	if !c.isReviewer() && !c.isAdmin() {
		s.fail(w, r, domain.ErrForbidden)
		return
	}
	f := r.URL.Query().Get("status")
	sts := []domain.DisputeStatus{domain.DisputeOpen, domain.DisputeUnderReview, domain.DisputePostingPending}
	for _, o := range queueFilters[1:] {
		if o.Value == f {
			sts = []domain.DisputeStatus{domain.DisputeStatus(f)}
		}
	}
	d, err := c.disputePage(r, sts, "")
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d.Filter = f
	s.render(w, r, c, "review_queue", 200, &page{Title: messages.Text("dispute.queue_title"), D: &queueData{d, queueFilters}, V: map[string]string{"status": f}})
}

type disputeForm struct {
	TxID string
	Cats []option
	Row  txRow
	// Deadline is the dispute deadline stored when the trade completed
	// (DOM-017), formatted for display; empty for a trade not completed.
	Deadline     template.HTML
	DeadlineText string
	Confirm      bool
}

func (s *server) disputeFormData(r *http.Request, c *rc) (*disputeForm, error) {
	v, err := s.svc.GetTransaction(r.Context(), c.actor, c.netID(), r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	if !v.Tx.IsParty(c.Me.ID) {
		return nil, domain.NotFound("trade")
	}
	row := makeRow(v.Tx, c.Me, c.memberNames(), c.Net.Policy.CurrencyName, c.Net.Policy.Decimals)
	d := &disputeForm{TxID: v.Tx.ID, Cats: categories, Row: row}
	if dl := v.Tx.DisputeDeadlineAt; dl != nil {
		at := c.at(*dl)
		d.Deadline, d.DeadlineText = fmtTime(at), at.Format("2 Jan 2006, 15:04 MST")
	}
	return d, nil
}

func (s *server) disputeForm(w http.ResponseWriter, r *http.Request, c *rc) {
	d, err := s.disputeFormData(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "dispute_form", 200, &page{Title: messages.Text("dispute.open_title"), D: d, V: map[string]string{"category": "not_delivered"}})
}

func (s *server) disputePost(w http.ResponseWriter, r *http.Request, c *rc) {
	d, err := s.disputeFormData(r, c)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	v := formValues(r)
	p := &page{Title: messages.Text("dispute.open_title"), D: d, V: v}
	step := r.PostFormValue("step")
	if step == "back" {
		s.render(w, r, c, "dispute_form", 200, p)
		return
	}
	st := strings.TrimSpace(v["statement"])
	if st == "" || utf8.RuneCountInString(st) > 2000 || categoryLabel(v["category"]) == v["category"] {
		p.Summary = messages.Text("dispute.invalid")
		if st == "" {
			p.Errors = append(p.Errors, fieldErr{"statement", messages.Text("dispute.statement.required")})
		}
		if utf8.RuneCountInString(st) > 2000 {
			p.Errors = append(p.Errors, fieldErr{"statement", messages.Text("dispute.statement.too_long")})
		}
		if categoryLabel(v["category"]) == v["category"] {
			p.Errors = append(p.Errors, fieldErr{"category", messages.Text("dispute.category.required")})
		}
		s.render(w, r, c, "dispute_form", 422, p)
		return
	}
	if step != "confirm" {
		d.Confirm = true
		p.Title = messages.Text("dispute.review_title")
		p.D = d
		s.render(w, r, c, "dispute_form", 200, p)
		return
	}
	dp, err := s.svc.OpenDispute(r.Context(), c.actor, c.netID(), service.OpenDisputeInput{TxID: d.TxID,
		Category: v["category"], Statement: st, IdempotencyKey: idemKey(r)})
	if err != nil {
		d.Confirm = true
		p.Title = messages.Text("dispute.review_title")
		if sum, fe, ok := c.refuse(err); ok {
			p.Summary, p.Errors = sum, fe
			if domain.CodeOf(err) == domain.CodeForbidden || domain.CodeOf(err) == domain.CodeInvalidState {
				p.Summary = messages.Text("dispute.not_eligible")
				if d.DeadlineText != "" {
					p.Summary = messages.Text("dispute.not_eligible_deadline", d.DeadlineText)
				}
			}
			s.render(w, r, c, "dispute_form", 422, p)
			return
		}
		s.fail(w, r, err)
		return
	}
	s.setFlash(w, r, messages.Text("dispute.opened"))
	http.Redirect(w, r, c.base()+"/disputes/"+dp.ID, http.StatusSeeOther)
}

type disputeData struct {
	V        *service.DisputeView
	Category string
	Status   string
	Outcome  string
	Opener   string
	Party    bool
	Withdraw bool
	Review   bool
	Decide   bool
	Retry    bool
	Events   []evRow
	Amount   string
	Unit     string
	// FR-076 response and FR-078 re-review.
	Response      string // response text, "" until submitted
	ResponseBy    string
	CanRespond    bool
	ReReviewText  string // one-line state of the re-review, "" when none
	ReReviewWhy   string
	ReReviewDone  string // decision rationale
	CanReReview   bool
	ReReviewUntil any // last day to ask, shown with the request form
	DecideReRev   bool
	OpenedAt      template.HTML
	// GOV-014: deadlines shown while the dispute is unresolved.
	Deadlines    bool
	ResponseDue  template.HTML
	DecisionBy   template.HTML
	ResponseLate bool
	DecisionLate bool
}

type evRow struct {
	Text string
	At   any
}

func outcomeWords(d *domain.Dispute, amt string, unit string) string {
	switch d.Outcome {
	case domain.OutcomeDismissed:
		return messages.Text("dispute.outcome.dismissed")
	case domain.OutcomeUpheld:
		if d.Status == domain.DisputePostingPending {
			return messages.Text("dispute.outcome.upheld_pending", amt, unit)
		}
		return messages.Text("dispute.outcome.upheld_posted", amt, unit)
	case domain.OutcomePartiallyUpheld:
		if d.Status == domain.DisputePostingPending {
			return messages.Text("dispute.outcome.partial_pending", amt, unit)
		}
		return messages.Text("dispute.outcome.partial_posted", amt, unit)
	}
	return ""
}

// Known stored event values have catalogue labels. Unknown values remain plain
// data: they are never used as catalogue keys or interpreted as formatting.
func disputeEventWords(action, outcome string) string {
	label := action
	switch action {
	case "opened", "under_review", "posting_pending", "resolved", "withdrawn", "responded",
		"rereview_requested", "rereview_confirmed", "rereview_upheld", "rereview_partially_upheld":
		label = messages.Text("dispute.event." + action)
	}
	if outcome == "" {
		return label
	}
	switch outcome {
	case "dismissed", "upheld", "partially_upheld":
		outcome = messages.Text("dispute.event_outcome." + outcome)
	}
	return messages.Text("dispute.event_with_outcome", label, outcome)
}

func (s *server) disputeView(w http.ResponseWriter, r *http.Request, c *rc) {
	s.showDispute(w, r, c, r.PathValue("id"), &page{}, 200)
}

func (s *server) showDispute(w http.ResponseWriter, r *http.Request, c *rc, id string, p *page, status int) {
	v, err := s.svc.GetDispute(r.Context(), c.actor, c.netID(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := v.Dispute
	pol := c.Net.Policy
	out := &disputeData{V: v, Category: categoryLabel(d.Category), Status: statusWords(d.Status), Party: v.Tx.IsParty(c.Me.ID)}
	out.Amount = domain.FormatAmount(v.Tx.Amount, pol.Decimals)
	out.Unit = unitFor(pol.CurrencyName, v.Tx.Amount, pol.Decimals)
	out.OpenedAt = c.when(d.OpenedAt)
	if d.Status == domain.DisputeOpen || d.Status == domain.DisputeUnderReview {
		out.Deadlines, out.ResponseDue, out.DecisionBy = true, c.when(v.Deadlines.ResponseDeadline), c.when(v.Deadlines.DecisionTarget)
		out.ResponseLate, out.DecisionLate = v.Deadlines.ResponseOverdue, v.Deadlines.DecisionOverdue
	}
	if d.Outcome != "" {
		out.Outcome = outcomeWords(d, domain.FormatAmount(d.ReversalAmount, pol.Decimals), unitFor(pol.CurrencyName, d.ReversalAmount, pol.Decimals))
	}
	if d.OpenedBy == v.Tx.PayerMemberID {
		out.Opener = v.PayerName
	} else {
		out.Opener = v.PayeeName
	}
	active := d.Status == domain.DisputeOpen || d.Status == domain.DisputeUnderReview
	reviewer := (c.isReviewer() || c.isAdmin()) && !out.Party
	out.Withdraw = active && d.OpenedBy == c.Me.ID
	out.Review = reviewer && d.Status == domain.DisputeOpen
	out.Decide = reviewer && active
	out.Retry = reviewer && d.Status == domain.DisputePostingPending
	s.reviewData(c, v, out)
	for _, e := range v.Events {
		out.Events = append(out.Events, evRow{Text: disputeEventWords(e.Action, e.Outcome), At: c.when(e.OccurredAt)})
	}
	p.Title, p.D = messages.Text("dispute.title"), out
	s.render(w, r, c, "dispute_view", status, p)
}

func (s *server) disputeAct(w http.ResponseWriter, r *http.Request, c *rc) {
	id, act := r.PathValue("id"), r.PathValue("act")
	var err error
	msg := ""
	switch act {
	case "withdraw":
		_, err = s.svc.WithdrawDispute(r.Context(), c.actor, c.netID(), id)
		msg = messages.Text("dispute.withdrawn")
	case "start-review":
		_, err = s.svc.StartDisputeReview(r.Context(), c.actor, c.netID(), id)
		msg = messages.Text("dispute.review_started")
	case "retry":
		_, err = s.svc.RetryDisputePosting(r.Context(), c.actor, c.netID(), id)
		msg = messages.Text("dispute.posting_retried")
	case "respond":
		_, err = s.svc.RespondToDispute(r.Context(), c.actor, c.netID(), service.RespondInput{DisputeID: id,
			Response: r.PostFormValue("response"), IdempotencyKey: idemKey(r)})
		msg = messages.Text("dispute.responded")
	case "rereview":
		_, err = s.svc.RequestReReview(r.Context(), c.actor, c.netID(), service.ReReviewRequestInput{DisputeID: id,
			Reason: r.PostFormValue("reason"), IdempotencyKey: idemKey(r)})
		msg = messages.Text("dispute.rereview_requested")
	case "rereview-decide":
		if r.PostFormValue("step") == "back" {
			s.showDispute(w, r, c, id, &page{V: formValues(r)}, 200)
			return
		}
		in := service.ReReviewDecideInput{DisputeID: id, Outcome: r.PostFormValue("outcome"),
			Rationale: strings.TrimSpace(r.PostFormValue("rationale")), IdempotencyKey: idemKey(r)}
		confirm := r.PostFormValue("step") == "confirm"
		if in.Outcome == string(domain.OutcomePartiallyUpheld) || confirm {
			amt, perr := domain.ParseAmount(r.PostFormValue("amount"), c.Net.Policy.Decimals)
			if perr != nil || (in.Outcome != service.ReReviewConfirmed && amt <= 0) {
				s.disputeFail(w, r, c, id, domain.Invalid("amount", messages.Text("dispute.amount_invalid", c.Net.Policy.Decimals)))
				return
			}
			in.Amount = amt
		}
		if !confirm {
			s.reReviewDecisionReview(w, r, c, id, in)
			return
		}
		if !secretEqual(r.PostFormValue("review_proof"), s.reReviewProof(c, in)) {
			s.disputeFail(w, r, c, id, domain.Invalid("outcome", messages.Text("dispute.review_again")))
			return
		}
		_, err = s.svc.DecideReReview(r.Context(), c.actor, c.netID(), in)
		msg = messages.Text("dispute.rereview_recorded")
	case "resolve":
		in := service.ResolveInput{DisputeID: id, Outcome: domain.DisputeOutcome(r.PostFormValue("outcome")),
			Rationale: r.PostFormValue("rationale"), WaiveResponse: r.PostFormValue("waive_response") == "1",
			WaiveReason: r.PostFormValue("waive_reason"), IdempotencyKey: idemKey(r)}
		if in.Outcome == domain.OutcomePartiallyUpheld {
			amt, perr := domain.ParseAmount(r.PostFormValue("amount"), c.Net.Policy.Decimals)
			if perr != nil || amt <= 0 {
				s.disputeFail(w, r, c, id, domain.Invalid("amount", messages.Text("dispute.amount_invalid", c.Net.Policy.Decimals)))
				return
			}
			in.Amount = amt
		}
		// A decision can move credit, so it gets a review step like every other money form: the
		// form posts step=review (or nothing) and only step=confirm records it.
		switch r.PostFormValue("step") {
		case "back":
			s.showDispute(w, r, c, id, &page{V: formValues(r)}, 200)
			return
		case "confirm":
		default:
			s.decisionReview(w, r, c, id, in)
			return
		}
		_, err = s.svc.ResolveDispute(r.Context(), c.actor, c.netID(), in)
		msg = messages.Text("dispute.decision_recorded")
	default:
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	if err != nil {
		s.disputeFail(w, r, c, id, err)
		return
	}
	s.setFlash(w, r, msg)
	http.Redirect(w, r, c.base()+"/disputes/"+id, http.StatusSeeOther) //nolint:gosec // G710: fixed local prefix, id is a single path segment
}

// disputeFail shows the refusal on the dispute page itself.
func (s *server) disputeFail(w http.ResponseWriter, r *http.Request, c *rc, id string, err error) {
	sum, fe, ok := c.refuse(err)
	if !ok {
		s.fail(w, r, err)
		return
	}
	s.showDispute(w, r, c, id, &page{Summary: sum, Errors: fe, V: formValues(r)}, 422)
}
