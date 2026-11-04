package api_test

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/histbal"
)

// API-083 through the real route and service: GET /accounts/me?at= reads the
// caller's own posted balance; without at, API-044 is unchanged.
func TestHistBalanceRouteEndToEnd(t *testing.T) {
	w := newWorldOpts(t, api.Options{CSRFKey: testCSRFKey, Now: func() time.Time { return time.Now() }})
	before := w.env.Clock.Now()
	w.env.Clock.Advance(time.Minute)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 500)
	at := w.env.Clock.Now()

	rec := w.get(w.bobT, "/accounts/me?at="+url.QueryEscape(at.In(time.FixedZone("", 2*3600)).Format(time.RFC3339)))
	wantStatus(t, rec, 200)
	got := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"as_of": histbal.FormatInstant(at), "balance_minor": "500", "direction": "in_credit",
		"decimals": float64(w.net.Policy.Decimals), "ledger_seq": float64(1), "has_postings_by_as_of": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	// The payer's own read; no other member's account is addressable.
	rec = w.get(w.aliceT, "/accounts/me?at="+url.QueryEscape(before.Format(time.RFC3339)))
	wantStatus(t, rec, 200)
	if b := decode[map[string]any](t, rec); b["balance_minor"] != "0" || b["has_postings_by_as_of"] != false {
		t.Fatalf("before any posting: %v", b)
	}
	wantProblem(t, w.get(w.aliceT, "/accounts/me?at="+url.QueryEscape(at.Format(time.RFC3339))+"&member_id="+w.bobM.ID), 400, "invalid-query")
	// Past the adapter's wall clock check but after the service clock: the
	// service refusal still surfaces as 400 invalid-query, not 500.
	wantProblem(t, w.get(w.bobT, "/accounts/me?at="+url.QueryEscape(at.Add(time.Hour).Format(time.RFC3339))), 400, "invalid-query")
	for _, bad := range []string{"2026-03-01", "2026-03-01T12:00:00", "2026-03-01T12:00:00.0001Z", "2026-03-01T12:00:00+02:00", ""} {
		wantProblem(t, w.get(w.bobT, "/accounts/me?at="+bad), 400, "invalid-query") // unencoded + decodes to a space
	}
	// API-044 unchanged without at.
	if b := decode[map[string]any](t, w.get(w.bobT, "/accounts/me")); b["credit_limit_minor"] == nil {
		t.Fatalf("current view lost fields: %v", b)
	}
}
