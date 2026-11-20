package api_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-18: unsupported query parameters must not silently turn a filtered or
// historical request into a different, apparently successful representation.
func TestGetRepresentationsRejectUnknownQueryParameters(t *testing.T) {
	w := newWorld(t)
	trade := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	dispute := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: trade.ID, Category: "other", Statement: "service not as agreed", IdempotencyKey: w.env.Key(),
	}))
	listing, _ := w.createListing(w.aliceT, "Repair")
	const missing = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/network", 200},
		{"/me", 200},
		{"/network/terms", 200},
		{"/members/" + w.aliceM.ID, 200},
		{"/accounts/me", 200},
		{"/listings/" + listing.ID, 200},
		{"/transactions/" + trade.ID, 200},
		{"/disputes/" + dispute.ID, 200},
		{"/admin/accounts/" + w.aliceM.ID, 200},
		{"/admin/policy", 200},
		{"/admin/network/closure", 200},
		{"/admin/consistency-checks/" + missing, 404},
		{"/admin/adjustments/" + missing, 404},
		{"/notifications/unread-count", 200},
		{"/exports/" + missing, 404},
		{"/exports/" + missing + "/artefact", 404},
		{"/tokens", 200},
	} {
		t.Run(tc.path, func(t *testing.T) {
			wantStatus(t, w.get(w.adminT, tc.path), tc.status)
			for _, query := range []string{"?bogus=1", "?limit=1", "?bogus="} {
				wantProblem(t, w.get(w.adminT, tc.path+query), 400, "invalid-query")
			}
		})
	}
	// Authentication still precedes query validation.
	wantProblem(t, w.get("", "/network?bogus=1"), 401, "unauthenticated")
}

func TestStrictGetQueriesPreserveHistoryAndPagination(t *testing.T) {
	w := newWorld(t)
	const historical = "/accounts/me?at=2000-01-01T00:00:00Z"
	wantStatus(t, w.get(w.aliceT, historical), 200)
	wantProblem(t, w.get(w.aliceT, historical+"&bogus=1"), 400, "invalid-query")
	for _, path := range []string{
		"/members?limit=1", "/accounts/me/entries?limit=1", "/transactions?limit=1",
		"/network/policy-log?limit=1", "/challenges?limit=1&status=requested",
	} {
		wantStatus(t, w.get(w.aliceT, path), 200)
	}
}
