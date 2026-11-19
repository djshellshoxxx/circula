package web

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-022, FR-023, FR-025: following a listing continuation must keep the
// owner's private inventory and the exact search/type filters. A full final
// page must not send the member to an empty page.
func TestListingPaginationPreservesView(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mine  bool
		count int
	}{
		{name: "own paused listings", mine: true, count: 51},
		{name: "filtered public listings", count: 51},
		{name: "full final page", count: 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			create := func(actor domain.Actor, kind domain.ListingType, title string) *domain.Listing {
				return testutil.Must(w.env.Svc.CreateListing(w.env.Ctx, actor, w.net.ID, service.ListingInput{
					Type: kind, Title: title, IdempotencyKey: w.env.Key(),
				}))
			}
			// These older rows belong after the first page by ID, but outside
			// the requested view. A broadened continuation exposes them.
			create(w.bob, domain.ListingOffer, "Other member's unrelated listing")
			create(w.alice, domain.ListingRequest, "repair & care wrong kind")
			create(w.alice, domain.ListingOffer, "Unrelated search result")
			var oldest string
			for i := 0; i < tc.count; i++ {
				l := create(w.alice, domain.ListingOffer, fmt.Sprintf("repair & care %02d", i))
				if i == 0 {
					oldest = l.Title
				}
				if tc.mine {
					paused := domain.ListingPaused
					testutil.Must(w.env.Svc.UpdateListing(w.env.Ctx, w.alice, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &paused}))
				}
			}
			b := w.browser(t)
			b.login("alice@example.org")
			query := url.Values{"type": {"offer"}, "q": {"repair & care"}}
			if tc.mine {
				query.Set("mine", "1")
			}
			first := b.get(w.base() + "/listings?" + query.Encode())
			if first.Status != 200 {
				t.Fatalf("first page: status %d", first.Status)
			}
			match := nextLinkRe.FindStringSubmatch(first.Body)
			if tc.count == 50 {
				if match != nil {
					t.Fatalf("full final page offers an empty continuation: %s", match[1])
				}
				return
			}
			if match == nil {
				t.Fatal("first page omitted continuation")
			}
			next := html.UnescapeString(match[1])
			u, err := url.Parse(next)
			if err != nil {
				t.Fatal(err)
			}
			for key, values := range query {
				if got := u.Query().Get(key); got != values[0] {
					t.Errorf("continuation %s=%q, want %q", key, got, values[0])
				}
			}
			second := b.get(next)
			if second.Status != 200 {
				t.Fatalf("second page: status %d", second.Status)
			}
			body := html.UnescapeString(second.Body)
			if !strings.Contains(body, oldest) {
				t.Error("oldest matching listing disappeared on continuation")
			}
			for _, unrelated := range []string{"Other member's unrelated listing", "repair & care wrong kind", "Unrelated search result"} {
				if strings.Contains(body, unrelated) {
					t.Errorf("continuation broadened view to %q", unrelated)
				}
			}
			if nextLinkRe.MatchString(second.Body) {
				t.Error("last page offers an empty continuation")
			}
		})
	}
}
