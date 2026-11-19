package messages

import "testing"

// UX-047: marketplace messages preserve exact amount/reference strings and
// full action names, with English plural and count formatting in the catalogue.
func TestMarketplaceMessages(t *testing.T) {
	for _, tc := range []struct {
		key  string
		args []any
		want string
	}{
		{"directory.results", []any{0}, "0 members found."},
		{"directory.results", []any{1}, "1 member found."},
		{"directory.results", []any{1000}, "1,000 members found."},
		{"directory.member_reference", []any{"1234-5678"}, "(member reference 1234-5678)"},
		{"directory.propose_with", []any{"Zoë 100%"}, "Propose a trade with Zoë 100%"},
		{"listing.owner_price", []any{"Zoë <hours> 100%", "90071992547409.93", "Hours"}, "By Zoë <hours> 100% · price 90071992547409.93 Hours"},
		{"listing.by_owner", []any{"Zoë"}, "By Zoë"},
		{"listing.price", []any{"1.00", "credit"}, "price 1.00 credit"},
		{"listing.price_help", []any{1, "125.0"}, "Price: enter a positive amount with up to 1 decimal place, for example 125.0."},
		{"listing.price_help", []any{2, "12.50"}, "Price: enter a positive amount with up to 2 decimal places, for example 12.50."},
		{"listing.edit_named", []any{"Repair & care"}, "Edit Repair & care"},
		{"listing.report_named", []any{"Repair & care"}, "Report Repair & care"},
		{"listing.propose_about", []any{"Repair & care"}, "Propose a trade about Repair & care"},
		{"profile.listings_by", []any{"Zoë 100%"}, "Listings by Zoë 100%"},
		{"profile.empty", []any{"Zoë"}, "Zoë has no active listings."},
		{"profile.propose_with", []any{"Zoë"}, "Propose a trade with Zoë"},
		{"profile.report", []any{"Zoë"}, "Report Zoë"},
	} {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			if got := Text(tc.key, tc.args...); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
