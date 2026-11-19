package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestCreditLimitRevisionInheritedDefaultAndUnrelatedPostings(t *testing.T) {
	w := newWorld(t, nil)
	original := tu.Must(w.Svc.PeekMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if original.CreditLimitRevision == "" {
		t.Fatal("missing internal revision")
	}
	// A policy default increase is immediate and changes inherited limits.
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit += 100 }, nil, ""))
	changed := tu.Must(w.Svc.PeekMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if changed.CreditLimitRevision == original.CreditLimitRevision {
		t.Fatal("changed inherited default reused the revision")
	}
	limit := int64(1000)
	beforeAudits := countRows(t, w.Env, "audit_events")
	_, err := w.Svc.SetCreditLimitView(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{
		MemberID: w.aM.ID, Limit: &limit, Reason: "stale", ExpectedRevision: original.CreditLimitRevision,
	})
	if !errors.Is(err, service.ErrCreditLimitChanged) {
		t.Fatalf("stale service precondition = %v", err)
	}
	if after := countRows(t, w.Env, "audit_events"); beforeAudits != after {
		t.Fatal("refused conditional write appended an audit event")
	}
	// An active override is unaffected by an unrelated default-policy edit.
	expiry := w.Clock.Now().Add(time.Hour)
	set := tu.Must(w.Svc.SetCreditLimitView(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{
		MemberID: w.aM.ID, Limit: &limit, ExpiresAt: &expiry, Reason: "active override", ExpectedRevision: changed.CreditLimitRevision,
	}))
	tu.Must(w.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit += 100 }, nil, ""))
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	unchanged := tu.Must(w.Svc.PeekMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if unchanged.CreditLimitRevision != set.CreditLimitRevision {
		t.Fatal("postings or a default hidden by an override invalidated the limit edit")
	}
	w.Clock.Advance(time.Hour)
	expired := tu.Must(w.Svc.PeekMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if expired.CreditLimitRevision == set.CreditLimitRevision {
		t.Fatal("expiry did not invalidate the limit edit")
	}
}
