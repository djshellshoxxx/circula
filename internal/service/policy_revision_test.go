package service_test

import (
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

func TestPolicyRevisionRefusalDoesNotConsumeKey(t *testing.T) {
	w := newWorld(t, nil)
	original := service.PolicyRevision{Version: w.net.PolicyVersion}
	current := tu.Must(w.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit += 100 }, nil, ""))
	pol := current.Policy
	pol.CurrencyName = "announced credits"
	days := 14
	in := service.PolicyChange{Policy: pol, Reason: "announce conditional change", NoticeDays: &days,
		ExpectedRevision: &original, IdempotencyKey: w.Key()}
	tables := []string{"network_policies", "audit_events", "notifications", "idempotency_keys"}
	before := make(map[string]int)
	for _, table := range tables {
		before[table] = countRows(t, w.Env, table)
	}
	_, err := w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, in)
	if !errors.Is(err, service.ErrPolicyChanged) {
		t.Fatalf("stale service precondition = %v", err)
	}
	for _, table := range tables {
		if got := countRows(t, w.Env, table); got != before[table] {
			t.Fatalf("refused update changed %s: %d -> %d", table, before[table], got)
		}
	}
	// Correcting only the precondition may retry the same durable key: the
	// refused command created neither a result nor a conflicting fingerprint.
	in.ExpectedRevision = &service.PolicyRevision{Version: current.PolicyVersion}
	tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, in))
	if got := countRows(t, w.Env, "network_policies"); got != before["network_policies"]+1 {
		t.Fatalf("fresh precondition did not append one scheduled policy: %d", got)
	}
	if got := countRows(t, w.Env, "notifications"); got <= before["notifications"] {
		t.Fatal("accepted scheduled policy did not notify members")
	}
}

func TestPolicyRevisionPreservesDurableReplay(t *testing.T) {
	w := newWorld(t, nil)
	pol := w.net.Policy
	pol.DefaultCreditLimit += 100
	in := service.PolicyChange{Policy: pol, Reason: "conditional increase", IdempotencyKey: w.Key(),
		ExpectedRevision: &service.PolicyRevision{Version: w.net.PolicyVersion}}
	first := tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, in))
	if first.PolicyVersion != 2 {
		t.Fatalf("first version = %d", first.PolicyVersion)
	}
	// A later command makes the original expected revision stale. A durable
	// retry still loads its result without executing the old command again.
	current := tu.Must(w.setPolicy(func(p *domain.Policy) { p.DefaultCreditLimit += 100 }, nil, ""))
	beforePolicies, beforeAudits := countRows(t, w.Env, "network_policies"), countRows(t, w.Env, "audit_events")
	for _, expected := range []*service.PolicyRevision{in.ExpectedRevision, nil, {Version: current.PolicyVersion}} {
		in.ExpectedRevision = expected
		got := tu.Must(w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, in))
		if got.PolicyVersion != current.PolicyVersion || got.Policy != current.Policy {
			t.Fatalf("durable retry overwrote the later policy: %+v", got)
		}
	}
	if countRows(t, w.Env, "network_policies") != beforePolicies || countRows(t, w.Env, "audit_events") != beforeAudits {
		t.Fatal("durable retry appended another policy or audit event")
	}
	in.Policy.DefaultCreditLimit++
	_, err := w.Svc.UpdatePolicy(w.Ctx, w.admin, w.net.ID, in)
	wantCode(t, err, domain.ErrKeyReused)
}
