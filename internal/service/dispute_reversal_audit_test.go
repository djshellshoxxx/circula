package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A1-09 / FR-081: each posted dispute reversal has its own target event,
// linked to the dispute and original transaction within the same atomic unit.
func TestDisputeReversalAuditTargetsAndAtomicity(t *testing.T) {
	for _, path := range []string{"resolve", "retry", "re-review"} {
		t.Run(path, func(t *testing.T) {
			var fs *reversalAuditFailStore
			e := tu.NewEnvWith(t, func(st store.Store) store.Store {
				fs = &reversalAuditFailStore{Store: st}
				return fs
			})
			w := &world{Env: e}
			w.admin = e.Person("admin@example.org")
			w.net, w.adminM = e.Network(w.admin, "audit-target", nil)
			w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
			w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
			reviewer, reviewerM := w.reviewerAs("first@example.org")
			original := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
			if path == "retry" {
				other, otherM := w.Member2(w.admin, "other@example.org")
				w.Trade(w.net.ID, w.b, other, w.bM, otherM, 10100)
			}
			d := tu.Must(w.open(w.a, original.ID))
			in := service.ResolveInput{DisputeID: d.ID, Outcome: domain.OutcomeUpheld, Rationale: "private evidence", IdempotencyKey: w.Key()}
			var attempt func() error
			switch path {
			case "resolve":
				attempt = func() error { _, err := w.Svc.ResolveDispute(w.Ctx, reviewer, w.net.ID, in); return err }
			case "retry":
				blocked := tu.Must(w.Svc.ResolveDispute(w.Ctx, reviewer, w.net.ID, in))
				if blocked.Status != domain.DisputePostingPending {
					t.Fatalf("not posting pending: %+v", blocked)
				}
				limit := int64(20000)
				noErr(t, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.bM.ID, Limit: &limit, Reason: "allow reversal"}))
				attempt = func() error { _, err := w.Svc.RetryDisputePosting(w.Ctx, reviewer, w.net.ID, d.ID); return err }
			case "re-review":
				tu.Must(w.resolve(reviewer, d.ID, domain.OutcomeDismissed, 0, "private evidence"))
				noErr(t, w.rereview(w.a, d.ID, "reconsider"))
				reviewer, reviewerM = w.reviewerAs("second@example.org")
				rr := service.ReReviewDecideInput{DisputeID: d.ID, Outcome: "upheld", Rationale: "private re-review evidence", IdempotencyKey: w.Key()}
				attempt = func() error { _, err := w.Svc.DecideReReview(w.Ctx, reviewer, w.net.ID, rr); return err }
			}
			reviewer.RequestID = "audit-link-test"
			if events := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, service.AuditQuery{Action: domain.ActTxReversed})); len(events) != 0 {
				t.Fatal("blocked/dismissed decision claimed a reversal that did not post")
			}
			before := dbChecksum(t, w.Env)
			fs.fail = true
			if err := attempt(); !errors.Is(err, errReversalAudit) {
				t.Fatalf("missing target audit failure: %v", err)
			}
			fs.fail = false
			if after := dbChecksum(t, w.Env); after != before {
				t.Fatal("target-audit failure left posting, decision, notices or audit changes")
			}
			noErr(t, attempt())
			reversals := tu.Must(w.Svc.ReversalsOf(w.Ctx, w.a, w.net.ID, original.ID))
			if len(reversals) != 1 {
				t.Fatalf("reversal count %d", len(reversals))
			}
			query := service.AuditQuery{Action: domain.ActTxReversed, TargetID: reversals[0].ID}
			events := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, query))
			if len(events) != 1 {
				t.Fatalf("audit events targeting reversal: %d", len(events))
			}
			event := events[0]
			if event.TargetType != "transaction" || event.ActorID != reviewerM.ID || event.ActorType != domain.ActorReviewer || event.RequestID != reviewer.RequestID {
				t.Fatalf("target/actor attribution: %+v", event)
			}
			var beforeSummary, afterSummary map[string]any
			noErr(t, json.Unmarshal([]byte(event.SummaryBefore), &beforeSummary))
			noErr(t, json.Unmarshal([]byte(event.SummaryAfter), &afterSummary))
			if beforeSummary["original"] != original.ID || afterSummary["dispute"] != d.ID || afterSummary["amount"] != float64(100) {
				t.Fatalf("missing reversal links: before=%v after=%v", beforeSummary, afterSummary)
			}
			for _, party := range []domain.Actor{w.a, w.b} {
				if visible := tu.Must(w.Svc.ListAudit(w.Ctx, party, w.net.ID, query)); len(visible) != 1 || visible[0].TargetID != reversals[0].ID {
					t.Fatalf("party cannot find its reversal target: %+v", visible)
				}
			}
			outsider, _ := w.Member2(w.admin, "outsider@example.org")
			if visible := tu.Must(w.Svc.ListAudit(w.Ctx, outsider, w.net.ID, query)); len(visible) != 0 {
				t.Fatal("unrelated member can see reversal event")
			}
			if path == "retry" {
				if err := attempt(); err == nil {
					t.Fatal("already-posted retry unexpectedly succeeded")
				}
			} else {
				noErr(t, attempt()) // durable command replay must not emit another event
			}
			if events := tu.Must(w.Svc.ListAudit(w.Ctx, w.admin, w.net.ID, query)); len(events) != 1 {
				t.Fatal("retry duplicated reversal audit event")
			}
			if report := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !report.OK {
				t.Fatal(report.Findings)
			}
		})
	}
}

var errReversalAudit = errors.New("injected reversal target audit failure")

type reversalAuditFailStore struct {
	store.Store
	fail bool
}

func (s *reversalAuditFailStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	if !s.fail {
		return s.Store.Update(ctx, fn)
	}
	return s.Store.Update(ctx, func(tx store.Tx) error {
		return fn(&reversalAuditFailTx{Tx: tx, GovRepo: tx.(store.GovRepo)})
	})
}

type reversalAuditFailTx struct {
	store.Tx
	store.GovRepo
}

func (t *reversalAuditFailTx) InsertAudit(event *domain.AuditEvent) error {
	if event.Action == domain.ActTxReversed {
		return errReversalAudit
	}
	return t.Tx.InsertAudit(event)
}
