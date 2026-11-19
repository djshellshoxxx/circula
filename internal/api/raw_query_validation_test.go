package api_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// A2-18: URL.Query silently drops malformed pairs. A broken historical query
// must not silently become a request for the current balance.
func TestMalformedRawQueriesAreRejected(t *testing.T) {
	w := newWorld(t)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	for _, path := range []string{
		"/network?bogus=%ZZ",
		"/network?%ZZ=1",
		"/network?bogus=1;ignored=2",
		"/accounts/me?at=%ZZ",
		"/accounts/me?at=2000-01-01T00:00:00Z;bogus=1",
		"/accounts/me?at=2000-01-01T00:00:00Z&bogus=%ZZ",
	} {
		t.Run(path, func(t *testing.T) {
			wantProblem(t, w.get(w.aliceT, path), 400, "invalid-query")
		})
	}

	// Valid percent escapes, including a timezone plus sign and literal
	// semicolons/percent signs in search text, remain supported.
	r := w.get(w.aliceT, "/accounts/me?at=2000-01-01T00%3A00%3A00%2B00%3A00")
	wantStatus(t, r, 200)
	history := decode[struct {
		AsOf    string `json:"as_of"`
		Balance string `json:"balance_minor"`
	}](t, r)
	if history.AsOf != "2000-01-01T00:00:00.000Z" || history.Balance != "0" {
		t.Fatalf("escaped historical query returned current data: %+v", history)
	}
	wantStatus(t, w.get(w.aliceT, "/listings?q=50%25%3B+repair&limit=1"), 200)
}

func TestMalformedRawQueriesPreserveGuardsAndDoNotConsumeIdempotency(t *testing.T) {
	w := newWorld(t)
	for _, token := range []string{"", "cir_invalid"} {
		wantProblem(t, w.get(token, "/network?bogus=%ZZ"), 401, "unauthenticated")
	}
	session, _ := tu.Must2(w.env.Svc.Login(w.env.Ctx, "alice", "correct horse battery"))
	wantProblem(t, w.do(req{method: "POST", path: "/api/v1/listings?bogus=%ZZ", body: m{"kind": "offer", "title": "Repair"},
		hdr: map[string]string{"Cookie": "circula_session=" + session}}), 403, "csrf-invalid")
	wantProblem(t, w.do(req{method: "GET", path: "/api/v1/network?bogus=%ZZ", token: w.aliceT,
		hdr: map[string]string{"X-Network": "not-a-membership"}}), 404, "not-found")
	outsider := w.env.Person("outsider")
	wantProblem(t, w.get(w.token(outsider, domain.ScopeRead), "/network?bogus=%ZZ"), 404, "not-found")

	key := w.key()
	body := m{"kind": "offer", "title": "Repair"}
	wantProblem(t, w.post(w.aliceT, "/listings?bogus=%ZZ", body, key), 400, "invalid-query")
	var count int
	if err := w.env.Store.DB().QueryRow("SELECT count(*) FROM listings").Scan(&count); err != nil || count != 0 {
		t.Fatalf("malformed query wrote %d listings: %v", count, err)
	}
	wantStatus(t, w.post(w.aliceT, "/listings", body, key), 201)
}
