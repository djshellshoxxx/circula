package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SA2-18: a trade may reference only an active listing of the counterparty,
// in the matching direction.
func TestProposeListingValidation(t *testing.T) {
	w := newWorld(t, nil)
	_, cM := w.Member(w.admin, w.net, "cy@example.org")
	offer := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike repair", IdempotencyKey: w.Key()}))
	want := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingRequest, Title: "Need a ladder", IdempotencyKey: w.Key()}))
	propose := func(who domain.Actor, cp string, d service.Direction, listing string) error {
		_, err := w.Svc.ProposeTransaction(w.Ctx, who, w.net.ID, service.ProposeInput{CounterpartyID: cp,
			Direction: d, Amount: 100, ListingID: listing, IdempotencyKey: w.Key()})
		return err
	}
	// unknown id
	wantCode(t, propose(w.b, w.aM.ID, service.ProposerPays, "nope"), domain.ErrValidation)
	// the listing belongs to A, not to C
	wantCode(t, propose(w.b, cM.ID, service.ProposerPays, offer.ID), domain.ErrValidation)
	// wrong direction: an offer is paid by the proposer, a request by the counterparty
	wantCode(t, propose(w.b, w.aM.ID, service.ProposerIsPaid, offer.ID), domain.ErrValidation)
	wantCode(t, propose(w.b, w.aM.ID, service.ProposerPays, want.ID), domain.ErrValidation)
	// right ones
	if err := propose(w.b, w.aM.ID, service.ProposerPays, offer.ID); err != nil {
		t.Fatalf("offer: %v", err)
	}
	if err := propose(w.b, w.aM.ID, service.ProposerIsPaid, want.ID); err != nil {
		t.Fatalf("request: %v", err)
	}
	// paused and archived listings are refused, indistinguishable from unknown ones
	for _, st := range []domain.ListingStatus{domain.ListingPaused, domain.ListingArchived} {
		st := st
		tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: offer.ID, Status: &st}))
		wantCode(t, propose(w.b, w.aM.ID, service.ProposerPays, offer.ID), domain.ErrValidation)
	}
}
