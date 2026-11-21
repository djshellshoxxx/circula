package api

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// These are the listing validators issued before the atomic revision check.
// Sharing the fingerprint with the service must retain the wire format,
// millisecond UTC timestamps and exact minor-unit values.
func TestListingETagRetainsSerializedRevision(t *testing.T) {
	l := &domain.Listing{
		ID: "018f0000-0000-7000-8000-000000000001", Title: "Bike repair",
		Description: "Owner's tools\nBring the bicycle.", Status: domain.ListingPaused,
		UpdatedAt: time.Date(2026, 11, 20, 14, 15, 16, 123456789, time.FixedZone("UTC+2", 2*60*60)),
	}
	if got := listingETag(l); got != `"bfbd7542ee498ebb82edfc45"` {
		t.Fatalf("unpriced listing ETag changed: %s", got)
	}
	price := int64(9007199254740993)
	l.Price = &price
	if got := listingETag(l); got != `"84e1561ed38d96a3087176dd"` {
		t.Fatalf("priced listing ETag changed: %s", got)
	}
}
