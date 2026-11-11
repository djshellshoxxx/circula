package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-076 / OQ-FR-2: a decision waits for the response, the response period, or a reviewer waiver.
func TestResponsePeriodGate(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.ResponsePeriodDays = 7 })
	rev, _ := w.reviewerAs("rev@example.org")
	decide := func(id string, waive bool, why string) error {
		_, err := w.Svc.ResolveDispute(w.Ctx, rev, w.net.ID, service.ResolveInput{DisputeID: id, Outcome: domain.OutcomeDismissed,
			Rationale: "no case", WaiveResponse: waive, WaiveReason: why, IdempotencyKey: w.Key()})
		return err
	}
	open := func() string {
		tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
		return tu.Must(w.open(w.a, tx.ID)).ID
	}
	d1 := open()
	wantCode(t, decide(d1, false, ""), domain.ErrInvalidState)
	wantCode(t, decide(d1, true, "  "), domain.ErrValidation) // waiver needs a reason
	if err := w.respond(w.b, d1, "I delivered."); err != nil {
		t.Fatal(err)
	}
	if err := decide(d1, false, ""); err != nil {
		t.Fatalf("decision after response: %v", err)
	}
	d2 := open()
	if err := decide(d2, true, "party unreachable, parties agree"); err != nil {
		t.Fatalf("waived decision: %v", err)
	}
	d3 := open()
	w.Clock.Advance(7*24*time.Hour + time.Minute)
	if err := decide(d3, false, ""); err != nil {
		t.Fatalf("decision after period: %v", err)
	}
}

// The re-review window and limit-warning percent come from the network policy.
func TestPolicyDrivenWindows(t *testing.T) {
	p := domain.DefaultPolicy()
	if p.ResponsePeriodDays != 7 || p.ReReviewWindowDays != 14 || p.LimitWarningPercent != 80 {
		t.Fatalf("defaults = %d/%d/%d", p.ResponsePeriodDays, p.ReReviewWindowDays, p.LimitWarningPercent)
	}
	for name, mod := range map[string]func(*domain.Policy){
		"response-neg":  func(p *domain.Policy) { p.ResponsePeriodDays = -1 },
		"response-high": func(p *domain.Policy) { p.ResponsePeriodDays = domain.MaxResponsePeriodDays + 1 },
		"rr-zero":       func(p *domain.Policy) { p.ReReviewWindowDays = 0 },
		"rr-high":       func(p *domain.Policy) { p.ReReviewWindowDays = domain.MaxReReviewWindowDays + 1 },
		"pct-zero":      func(p *domain.Policy) { p.LimitWarningPercent = 0 },
		"pct-high":      func(p *domain.Policy) { p.LimitWarningPercent = 101 },
	} {
		q := domain.DefaultPolicy()
		mod(&q)
		if q.Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
	w := newWorld(t, func(p *domain.Policy) { p.ReReviewWindowDays = 3 })
	rev, _ := w.reviewerAs("rev@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	d := tu.Must(w.open(w.a, tx.ID))
	tu.Must(w.resolve(rev, d.ID, domain.OutcomeDismissed, 0, "no case"))
	w.Clock.Advance(4 * 24 * time.Hour)
	wantCode(t, w.rereview(w.a, d.ID, "please look again"), domain.ErrInvalidState)
}
