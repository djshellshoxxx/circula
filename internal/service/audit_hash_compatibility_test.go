package service_test

import (
	"slices"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// The persisted audit format predates the shared integrity evaluator. Pin its
// hash independently of fixtures built with the currently compiled helper.
func TestAuditHashPreservesOriginalFormat(t *testing.T) {
	event := &domain.AuditEvent{
		ID: "event-legacy", NetworkID: "network-legacy", Seq: 17,
		OccurredAt: time.UnixMilli(1700000000123).UTC(),
		ActorType:  domain.ActorAdmin, ActorID: "member-admin", Action: "transaction.completed",
		TargetType: "transaction", TargetID: "transaction-legacy", RequestID: "request-legacy", IdemRef: "idem-legacy",
		SummaryBefore: `{"status":"proposed"}`, SummaryAfter: `{"amount":2500,"status":"completed"}`,
		Reason: "fixed audit fixture", SubjectIDs: []string{"member-z", "member-a"}, PrevHash: "previous-hash",
	}
	const want = "b38fd6df574433218ca26bb7d39853153800e7a748e0ba914aa64932c141a706"
	if got := service.AuditHash(event); got != want {
		t.Fatalf("persisted audit hash changed: got %s, want %s", got, want)
	}
	if !slices.Equal(event.SubjectIDs, []string{"member-z", "member-a"}) {
		t.Fatal("hashing mutated the event subjects")
	}
}
