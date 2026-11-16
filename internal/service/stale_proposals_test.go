package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-067: the attention queue uses exactly the overview's age/expiry boundary,
// pages without duplicates, and authorizes before validating the cursor.
func TestAdminStaleProposalsMatchesOverview(t *testing.T) {
	w := newWorld(t, nil)
	first := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	second := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 200))
	closed := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 300))
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: closed.ID, IdempotencyKey: w.Key()}))
	w.Clock.Advance(service.OverviewStaleAfter - time.Millisecond)
	if got := tu.Must(w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{})); len(got) != 0 {
		t.Fatal("proposal became stale before the stated age")
	}
	w.Clock.Advance(time.Millisecond)
	fresh := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 400))
	ov := tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	page1 := tu.Must(w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{Limit: 1}))
	if ov.StaleProposals != 2 || len(page1) != 1 {
		t.Fatalf("count=%d first page=%v", ov.StaleProposals, page1)
	}
	page2 := tu.Must(w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{Limit: 1, AfterID: page1[0].ID}))
	if len(page2) != 1 || page2[0].ID == page1[0].ID || page1[0].ID == fresh.ID || page2[0].ID == fresh.ID {
		t.Fatalf("incorrect second page: %v", page2)
	}
	seen := map[string]bool{page1[0].ID: true, page2[0].ID: true}
	if !seen[first.ID] || !seen[second.ID] {
		t.Fatal("queue does not contain both aged proposals")
	}
	if got := tu.Must(w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{AfterID: page2[0].ID})); len(got) != 0 {
		t.Fatal("end of queue is not empty")
	}
	_, err := w.Svc.AdminStaleProposals(w.Ctx, w.a, w.net.ID, service.Page{AfterID: "invalid"})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{AfterID: "invalid"})
	wantCode(t, err, domain.ErrValidation)
	other := w.Person("stale-outsider@example.org")
	_, err = w.Svc.AdminStaleProposals(w.Ctx, other, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	// Advance to the original proposals' exact expiry, without a sweep job.
	w.Clock.Advance(first.ExpiresAt.Sub(w.Clock.Now()))
	ov = tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	got := tu.Must(w.Svc.AdminStaleProposals(w.Ctx, w.admin, w.net.ID, service.Page{}))
	if len(got) != ov.StaleProposals || seenID(got, first.ID) || seenID(got, second.ID) {
		t.Fatal("expired proposals stayed in the queue")
	}
}

func seenID(txs []*domain.Transaction, id string) bool {
	for _, tx := range txs {
		if tx.ID == id {
			return true
		}
	}
	return false
}
