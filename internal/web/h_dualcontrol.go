package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// Dual control of administrator adjustments (SEC-011): the pending list and
// the approve, reject and cancel forms. The service holds every rule; this
// file only decides which buttons to show.

func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/admin/adjustments/pending", s.adminH(s.pendingList))
		s.mux.HandleFunc("POST /n/{slug}/admin/adjustments/{id}/{action}", s.adminH(s.pendingAct))
	})
}

type pendingRow struct {
	ID, Target, Proposer, Decider string
	Dir, Amount, Reason, Status   string
	RelatedTxID                   string
	Created, Expires              time.Time
	Expired                       bool
	CanApprove, CanReject         bool
	CanCancel                     bool
}

type pendingData struct {
	Rows      []pendingRow
	Next      string
	Threshold string // "" when dual control is off
}

var pendingFilters = map[string]domain.PendingStatus{
	"pending": domain.PendingOpen, "approved": domain.PendingApproved, "rejected": domain.PendingRejected,
	"expired": domain.PendingExpired, "cancelled": domain.PendingCancelled,
}

func (s *server) pendingList(w http.ResponseWriter, r *http.Request, c *rc) {
	s.renderPending(w, r, c, 200, &page{})
}

func (s *server) renderPending(w http.ResponseWriter, r *http.Request, c *rc, st int, p *page) {
	f := r.URL.Query().Get("status")
	if _, ok := pendingFilters[f]; !ok {
		f = "pending"
	}
	rows, err := s.svc.ListPendingAdjustments(r.Context(), c.actor, c.netID(), []domain.PendingStatus{pendingFilters[f]},
		service.Page{AfterID: r.URL.Query().Get("after"), Limit: 100})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// Offer approve/reject only where the service would accept them (A3-01).
	ids := make([]string, 0, len(rows))
	for _, x := range rows {
		ids = append(ids, x.ID)
	}
	decidable, err := s.svc.DecidableAdjustments(r.Context(), c.actor, c.netID(), ids)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	nm := c.memberNames()
	dec, now := c.Net.Policy.Decimals, s.opts.Clock.Now()
	d := &pendingData{}
	if t := c.Net.Policy.DualControlThreshold; t > 0 {
		d.Threshold = domain.FormatAmount(t, dec)
	}
	for _, x := range rows {
		row := pendingRow{ID: x.ID, Target: nm.of(x.Target), Proposer: nm.of(x.Proposer), Amount: domain.FormatAmount(x.Amount, dec),
			Reason: x.Reason, RelatedTxID: x.RelatedTxID, Status: string(x.Status), Created: x.CreatedAt, Expires: x.ExpiresAt, Expired: !now.Before(x.ExpiresAt)}
		row.Dir = "Credit (balance goes up)"
		if x.Direction == string(service.AdjustDebit) {
			row.Dir = "Debit (balance goes down)"
		}
		if x.DecidedBy != "" {
			row.Decider = nm.of(x.DecidedBy)
		}
		if x.Status == domain.PendingOpen && !row.Expired {
			mine := x.Proposer == c.Me.ID
			row.CanCancel = mine
			row.CanApprove = decidable[x.ID]
			row.CanReject = row.CanApprove
		}
		d.Rows = append(d.Rows, row)
	}
	if len(rows) == 100 {
		d.Next = rows[99].ID
	}
	p.Title, p.D = "Adjustments awaiting approval", d
	if p.V == nil {
		p.V = map[string]string{}
	}
	p.V["status"] = f
	s.render(w, r, c, "admin_adjust_pending", st, p)
}

var pendingVerbs = map[string]string{"approve": "approved", "reject": "rejected", "cancel": "cancelled"}

func (s *server) pendingAct(w http.ResponseWriter, r *http.Request, c *rc) {
	action, id := r.PathValue("action"), r.PathValue("id")
	done, ok := pendingVerbs[action]
	if !ok {
		s.fail(w, r, domain.NotFound("page"))
		return
	}
	var err error
	switch action {
	case "approve":
		switch r.PostFormValue("step") {
		case "back":
			s.renderPending(w, r, c, 200, &page{})
			return
		case "confirm":
			var pa *domain.PendingAdjustment
			pa, err = s.svc.GetPendingAdjustment(r.Context(), c.actor, c.netID(), id)
			if err == nil && !secretEqual(r.PostFormValue("review_proof"), s.pendingApprovalProof(c, pa)) {
				err = domain.Invalid("approval", "review this adjustment again before confirming; nothing changed")
			}
			if err == nil {
				_, err = s.svc.ApproveAdjustment(r.Context(), c.actor, c.netID(), id)
			}
		default:
			err = s.pendingApprovalReview(w, r, c, id)
			if err == nil {
				return
			}
		}
	case "reject":
		_, err = s.svc.RejectAdjustment(r.Context(), c.actor, c.netID(), id)
	default:
		_, err = s.svc.CancelAdjustment(r.Context(), c.actor, c.netID(), id)
	}
	if err != nil {
		p := &page{}
		sum, fe, handled := c.refuse(err)
		switch domain.CodeOf(err) {
		case domain.CodeForbidden:
			sum, handled = "Not done. You cannot decide this request: it needs a different administrator, and never the member it adjusts. Nothing changed.", true
		case domain.CodeInvalidState:
			sum, handled = "Not done. This request has expired or was already decided. Nothing changed. The list below shows its current state.", true
		}
		if !handled {
			s.fail(w, r, err)
			return
		}
		p.Summary, p.Errors = sum, fe
		s.renderPending(w, r, c, http.StatusConflict, p)
		return
	}
	s.setFlash(w, r, fmt.Sprintf("Adjustment request %s.", done))
	http.Redirect(w, r, c.base()+"/admin/adjustments/pending", http.StatusSeeOther)
}

type pendingApprovalData struct {
	adjustReview
	ID, Proposer, Proof string
}

func (s *server) pendingApprovalProof(c *rc, pa *domain.PendingAdjustment) string {
	b, _ := json.Marshal([]string{c.netID(), c.actor.PersonID, pa.ID, pa.Proposer, pa.Target,
		pa.Direction, strconv.FormatInt(pa.Amount, 10), pa.Reason, pa.RelatedTxID})
	return s.mac("adjustment-approval:" + string(b))
}

// pendingApprovalReview uses the stored request, never amounts or member IDs
// supplied by the form. Only the later confirmed service call may post it.
func (s *server) pendingApprovalReview(w http.ResponseWriter, r *http.Request, c *rc, id string) error {
	pa, err := s.svc.GetPendingAdjustment(r.Context(), c.actor, c.netID(), id)
	if err != nil {
		return err
	}
	if pa.Status != domain.PendingOpen || !s.opts.Clock.Now().Before(pa.ExpiresAt) {
		return domain.InvalidState("this request has expired or was already decided")
	}
	allowed, err := s.svc.DecidableAdjustments(r.Context(), c.actor, c.netID(), []string{id})
	if err != nil {
		return err
	}
	if !allowed[id] {
		return domain.Forbidden("a different independent administrator must decide this request")
	}
	b, err := s.svc.AdminGetMemberBalance(r.Context(), c.actor, c.netID(), pa.Target)
	if err != nil {
		return err
	}
	delta, direction := pa.Amount, "Credit (balance goes up)"
	if pa.Direction == string(service.AdjustDebit) {
		delta, direction = -pa.Amount, "Debit (balance goes down)"
	}
	after, ok := domain.AddBounded(b.Balance, delta)
	if !ok {
		return domain.Invalid("amount", "the resulting balance would exceed the supported range")
	}
	names, pol := c.memberNames(), c.Net.Policy
	d := &pendingApprovalData{ID: id, Proposer: names.of(pa.Proposer), Proof: s.pendingApprovalProof(c, pa),
		adjustReview: adjustReview{Name: names.of(pa.Target), Dir: direction, Amount: domain.FormatAmount(pa.Amount, pol.Decimals), Reason: pa.Reason, RelatedTxID: pa.RelatedTxID,
			Before: makeBalance(b.Balance, pol.Decimals, pol.CurrencyName), After: makeBalance(after, pol.Decimals, pol.CurrencyName)}}
	s.render(w, r, c, "admin_adjust_approve", 200, &page{Title: "Review adjustment approval", D: d})
	return nil
}
