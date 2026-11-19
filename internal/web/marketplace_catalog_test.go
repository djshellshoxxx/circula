package web

import (
	"bytes"
	"html"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func renderMarketplacePage(t *testing.T, name string, p *page) string {
	t.Helper()
	var out bytes.Buffer
	if err := mustLoadTemplates()[name].ExecuteTemplate(&out, "base", p); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "%!") {
		t.Fatal("marketplace catalogue interpolation was corrupted")
	}
	return out.String()
}

func wantMarketplaceText(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html.UnescapeString(body), want) {
			t.Errorf("marketplace page lacks %q", want)
		}
	}
}

// UX-047, FR-021, FR-023: prices in both listing views use exact minor-unit
// amounts and the existing per-amount unit inflector. They must not append a
// second s to an already-plural unit or pluralize exactly one unit.
func TestMarketplacePriceUnits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unit   string
		amount int64
		want   string
	}{
		{"singular", "credit", 100, "price 1.00 credit"},
		{"plural", "credit", 250, "price 2.50 credits"},
		{"already plural singular", "Hours", 100, "price 1.00 Hours"},
		{"already plural", "Hours", 250, "price 2.50 Hours"},
		{"HTML-sensitive unit", "<hours>&", 100, "price 1.00 <hours>&"},
		{"exact large amount", "credit", 9007199254740993, "price 90071992547409.93 credits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &domain.Listing{ID: "listing-id", MemberID: "member-id", Type: domain.ListingOffer,
				Title: "Repair", Status: domain.ListingActive, Price: &tc.amount}
			for name, data := range map[string]any{
				"listings": &listingsData{Items: []listingRow{{Listing: l, Owner: "Zoë", Price: domain.FormatAmount(tc.amount, 2)}}},
				"profile":  &profileData{ID: "member-id", Name: "Zoë", Listings: []*domain.Listing{l}},
			} {
				body := renderMarketplacePage(t, name, &page{Slug: "town", Unit: tc.unit, Dec: 2, D: data})
				wantMarketplaceText(t, body, tc.want)
				for _, bad := range []string{"price 1.00 credits", "Hourss", "<hours>"} {
					if strings.Contains(body, bad) {
						t.Errorf("%s has invalid price rendering %q", name, bad)
					}
				}
				if tc.unit == "<hours>&" && !strings.Contains(body, "&lt;hours&gt;&amp;") {
					t.Errorf("%s did not escape the unit", name)
				}
			}
		})
	}
}

// UX-047, SEC-016: catalogue interpolation keeps visible labels, complete
// accessible action names, stable references, and contextual HTML escaping.
func TestMarketplacePagesCatalogue(t *testing.T) {
	const unsafe = `<img src=x onerror=alert(1)> & Zoë 100%`
	const memberID = "018f0000-0000-7000-8000-00000000000a"
	for _, status := range []domain.ListingStatus{domain.ListingActive, domain.ListingPaused, domain.ListingArchived} {
		l := &domain.Listing{ID: "listing-id", MemberID: memberID, Type: domain.ListingOffer,
			Title: unsafe, Description: unsafe, Status: status}
		for _, own := range []bool{false, true} {
			body := renderMarketplacePage(t, "listings", &page{Slug: "town", Unit: "credit", Dec: 2,
				V: map[string]string{"q": unsafe, "type": "offer"},
				D: &listingsData{Mine: own, Items: []listingRow{{Listing: l, Owner: unsafe, Own: own}}}})
			wantMarketplaceText(t, body, "New listing", "By "+unsafe, `class="tag">`+string(status)+"</span>")
			if own {
				wantMarketplaceText(t, body, "All listings", `aria-label="Edit `+unsafe+`"`)
			} else {
				wantMarketplaceText(t, body, "My listings", "Offers and requests", "Filter",
					`aria-label="Propose a trade about `+unsafe+`"`, `aria-label="Report `+unsafe+`"`)
			}
			if strings.Contains(body, "<img src=x") || strings.Contains(body, " · price ") {
				t.Fatal("unpriced listing gained a price or unescaped member content")
			}
		}
	}
	for _, name := range []string{"listing_form", "listing_edit"} {
		body := renderMarketplacePage(t, name, &page{Slug: "town", D: "listing-id", CSRF: "csrf-value", Idem: "idem-value",
			V: map[string]string{"title": unsafe, "description": unsafe, "price": "1.00", "type": "request", "status": "paused"}})
		wantMarketplaceText(t, body, ">Title</label>", ">Description (optional)</label>", ">Price (optional)</label>",
			`name="csrf" value="csrf-value"`, ">Cancel</a>")
		if name == "listing_form" {
			wantMarketplaceText(t, body, "Offer: something I provide", "Request: something I want", ">Publish listing</button>",
				`name="idem" value="idem-value"`, "Leave empty if open to offers.")
		} else {
			wantMarketplaceText(t, body, "Kind: <strong>Request</strong>", ">Status</label>", ">Save</button>",
				`value="paused" selected>Paused</option>`, "Archived listings are final and cannot be reopened.")
		}
		if strings.Contains(body, "<img src=x") {
			t.Fatal("retained listing form content crossed the escaping boundary")
		}
	}
	for _, self := range []bool{false, true} {
		body := renderMarketplacePage(t, "profile", &page{Slug: "town", D: &profileData{ID: memberID, Name: unsafe, Self: self}})
		wantMarketplaceText(t, body, "Member reference: <strong>0000-000A</strong>", "Display names can repeat; the reference does not.",
			"Listings by "+unsafe, unsafe+" has no active listings.")
		if self {
			wantMarketplaceText(t, body, "Tell it to the members you trade with so they can check it is you.")
			if strings.Contains(html.UnescapeString(body), "Propose a trade with "+unsafe) {
				t.Fatal("own profile gained a trade action")
			}
		} else {
			wantMarketplaceText(t, body, "Every trade confirmation shows it, so you can check you are trading with this member.",
				"Propose a trade with "+unsafe, "Report "+unsafe)
		}
		if strings.Contains(body, "<img src=x") {
			t.Fatal("profile member name crossed the escaping boundary")
		}
	}
}

func TestDirectoryCatalogueCountsAndReferences(t *testing.T) {
	for _, tc := range []struct {
		total int
		want  string
	}{{0, "0 members found."}, {1, "1 member found."}, {1000, "1,000 members found."}} {
		body := renderMarketplacePage(t, "directory", &page{Slug: "town", D: &dirData{Pages: pageNav{Total: tc.total}}})
		wantMarketplaceText(t, body, tc.want, "Search by name", "No members match. Try a shorter search, or clear it to see everyone.")
	}
	const name = `Zoë 100% <img src=x>`
	body := renderMarketplacePage(t, "directory", &page{Slug: "town", D: &dirData{Q: name, Pages: pageNav{Total: 1, Next: "/n/town/directory?after=example"},
		Entries: []service.DirectoryEntry{{MemberID: "018f0000-0000-7000-8000-00000000000a", DisplayName: name}}}})
	wantMarketplaceText(t, body, "(member reference 0000-000A)", `aria-label="Propose a trade with `+name+`"`, "Directory pages", "Next page")
	if strings.Contains(body, "<img src=x>") {
		t.Fatal("directory name or retained search crossed the escaping boundary")
	}
}
