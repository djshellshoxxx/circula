package web

import (
	"strconv"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-08: the listing edit page finds a listing by id, not by scanning the
// first page of the member's listings, so an old listing stays editable after
// the member has archived more than 500 newer ones.
func TestA3_08_EditFindsOldListingBeyondPageCap(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	first := testutil.Must(w.env.Svc.CreateListing(w.env.Ctx, w.alice, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Oldest", IdempotencyKey: w.env.Key()}))
	archived := domain.ListingArchived
	for i := 0; i < 520; i++ {
		l := testutil.Must(w.env.Svc.CreateListing(w.env.Ctx, w.alice, w.net.ID, service.ListingInput{
			Type: domain.ListingOffer, Title: "Filler " + strconv.Itoa(i), IdempotencyKey: w.env.Key()}))
		testutil.Must(w.env.Svc.UpdateListing(w.env.Ctx, w.alice, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &archived}))
	}
	a := w.browser(t)
	a.login("alice@example.org")
	ef := a.get(b + "/listings/" + first.ID + "/edit")
	if ef.Status != 200 {
		t.Fatalf("edit form for the oldest listing: %d", ef.Status)
	}
	a.mustContain(ef, `value="Oldest"`)
}
