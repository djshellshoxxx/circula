package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// Listing text search folds case and accents beyond ASCII and treats LIKE wildcards literally.
func TestListingSearchUnicodeFold(t *testing.T) {
	w := newWorld(t, nil)
	mk := func(title, desc string) {
		tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: title, Description: desc, IdempotencyKey: w.Key()}))
	}
	mk("Crème brûlée lessons", "Pâtisserie basics")
	mk("İstanbul tours", "walking")
	mk("Plain", "100% organic_veg")
	for q, want := range map[string]int{
		"creme": 1, "CRÈME": 1, "brulee": 1, "patisserie": 1, "ISTANBUL": 1, "ıstanbul": 1,
		"100%": 1, "organic_": 1, "%": 1, "_": 1, "zzz": 0,
	} {
		got := tu.Must(w.Svc.ListListings(w.Ctx, w.b, w.net.ID, service.ListingQuery{Query: q}))
		if len(got) != want {
			t.Errorf("query %q: got %d, want %d", q, len(got), want)
		}
	}
}
