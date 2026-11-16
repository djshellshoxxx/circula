package service_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A3-08: control and invisible format characters are refused in listing text.
func TestA3_08_ListingTextRejectsInvisibleRunes(t *testing.T) {
	w := newWorld(t, nil)
	create := func(title, desc string) error {
		_, err := w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: title, Description: desc, IdempotencyKey: w.Key()})
		return err
	}
	for _, c := range [][2]string{
		{"Bike\u200brepair", ""},            // zero width space
		{"Bike\u200drepair", ""},            // zero width joiner
		{"Bike \u202erepair", ""},           // bidi override
		{"\ufeffBike", ""},                  // BOM
		{"Bike\x07repair", ""},              // control
		{"Bike\nrepair", ""},                // newline in a one-line field
		{"Bike repair", "a\x00b"},           // NUL
		{"Bike repair", "pay me\u2060 now"}, // word joiner
		{"Bike repair", "a\u2028b"},         // line separator
		{"Bike repair", "a\x1b[31mb"},       // escape
		{"Bike repair", "a\vb"},             // vertical tab
	} {
		wantCode(t, create(c[0], c[1]), domain.ErrValidation)
	}
	// Ordinary multi-line text is fine.
	if err := create("Bike repair", "Line one\r\nLine two\twith tab, naïve café 日本語"); err != nil {
		t.Fatalf("plain text refused: %v", err)
	}
	// Update goes through the same rules.
	l := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Ok", IdempotencyKey: w.Key()}))
	bad := "x\u200by"
	_, err := w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: l.ID, Title: &bad})
	wantCode(t, err, domain.ErrValidation)
}

// A3-08: at most MaxActiveListingsPerMember active listings per member; paused
// or archived ones do not count, and re-activating counts again.
func TestA3_08_ActiveListingCap(t *testing.T) {
	w := newWorld(t, nil)
	create := func(who domain.Actor) (*domain.Listing, error) {
		return w.Svc.CreateListing(w.Ctx, who, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Item", IdempotencyKey: w.Key()})
	}
	var first *domain.Listing
	for i := 0; i < service.MaxActiveListingsPerMember; i++ {
		l, err := create(w.a)
		if err != nil {
			t.Fatalf("listing %d: %v", i, err)
		}
		if first == nil {
			first = l
		}
	}
	_, err := create(w.a)
	wantCode(t, err, domain.ErrValidation)
	if _, err := create(w.b); err != nil { // the cap is per member
		t.Fatalf("other member blocked: %v", err)
	}
	paused := domain.ListingPaused
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: first.ID, Status: &paused}))
	extra := tu.Must(create(w.a))
	active := domain.ListingActive
	_, err = w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: first.ID, Status: &active})
	wantCode(t, err, domain.ErrValidation)
	// Editing text of an already active listing is not blocked by the cap.
	title := "Renamed"
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: extra.ID, Title: &title}))
}
