package api_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// ARC-010 (TB1 into TB2): a JSON command body and a pagination cursor are
// attacker input. Whatever the bytes, on any of these routes the API answers
// without a 5xx (no panic, no internal error), and a refused request (4xx)
// leaves the ledger, the audit log and every other table exactly as it was
// (testutil.Snapshot). Seeds include a body each route accepts, so the seed
// run (go test) also passes through the accepting path.

type fuzzRoute struct {
	method, path string
	seed         string // a well-formed body: accepted, except where noted
}

func FuzzAPICommandBody(f *testing.F) {
	fx := testutil.NewFixture(f)
	h := api.NewHandlerWithOptions(fx.Svc, api.Options{CSRFKey: testCSRFKey, Now: fx.Clock.Now,
		PrincipalPerMinute: -1, IPPerMinute: -1, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	tok, _, err := fx.Svc.CreateAPIToken(fx.Ctx, fx.Admin, service.TokenInput{Name: "fuzz", Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		f.Fatal(err)
	}
	propose := func(by domain.Actor, to *domain.Member) *domain.Transaction {
		tx, err := fx.Svc.ProposeTransaction(fx.Ctx, by, fx.Net.ID, service.ProposeInput{CounterpartyID: to.ID,
			Direction: service.ProposerPays, Amount: 100, IdempotencyKey: fx.Key()})
		if err != nil {
			f.Fatal(err)
		}
		return tx
	}
	complete := func(by, to domain.Actor, toM *domain.Member) *domain.Transaction {
		tx := propose(by, toM)
		if _, err := fx.Svc.AcceptTransaction(fx.Ctx, to, fx.Net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: fx.Key()}); err != nil {
			f.Fatal(err)
		}
		return tx
	}
	done := complete(fx.Admin, fx.Alice, fx.AliceM) // the admin is a party: disputes
	other := complete(fx.Alice, fx.Bob, fx.BobM)    // the admin is not a party: reversal
	pending := propose(fx.Admin, fx.BobM)
	lst, err := fx.Svc.CreateListing(fx.Ctx, fx.Admin, fx.Net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bike repair", IdempotencyKey: fx.Key()})
	if err != nil {
		f.Fatal(err)
	}

	routes := []fuzzRoute{
		{"POST", "/transactions", `{"counterparty_member_id":"` + fx.AliceM.ID + `","direction":"pay","amount_minor":"100","description":"eggs"}`},
		{"POST", "/transactions/" + pending.ID + "/cancel", `{"reason":"changed my mind"}`},
		{"POST", "/listings", `{"kind":"offer","title":"Eggs","description":"fresh","price_minor":"100"}`},
		{"PATCH", "/listings/" + lst.ID, `{"title":"Bike repairs"}`},
		{"POST", "/disputes", `{"transaction_id":"` + done.ID + `","category":"quality","statement":"broken"}`},
		{"POST", "/admin/adjustments", `{"member_id":"` + fx.BobM.ID + `","direction":"credit","amount_minor":"5","reason":"gift"}`},
		{"POST", "/admin/transactions/" + other.ID + "/reversal", `{"reason":"agreed undo"}`},
		{"PATCH", "/admin/policy", `{"reason":"raise limit","default_credit_limit_minor":"20000"}`},
		{"PUT", "/admin/accounts/" + fx.AliceM.ID + "/credit-limit", `{"credit_limit_minor":"500","reason":"risk review"}`},
		{"POST", "/admin/invitations", `{}`},
		{"POST", "/admin/announcements", `{"title":"Market day","body":"Saturday"}`},
		{"POST", "/reports", `{"target_type":"member","target_id":"` + fx.BobM.ID + `","category":"harassment"}`},
		{"POST", "/challenges", `{"kind":"credit_limit","reason":"too low"}`}, // refused: no limit decision to challenge
		{"PATCH", "/me", `{"time_zone":"Europe/Berlin"}`},
		{"GET", "/transactions?cursor=", ""}, // the body bytes are the cursor
		{"GET", "/listings?limit=1&cursor=", ""},
	}
	malformed := []string{
		``, ` `, `{}`, `[]`, `null`, `"x"`, `{`, `}`, `{}}`, `{} {}`, `{"a":1}`, `{"reason":1}`, `{"reason":"a","reason":"b"}`,
		`{"amount_minor":"-1"}`, `{"amount_minor":"1e3"}`, `{"amount_minor":"99999999999999999999"}`, `{"amount_minor":100}`,
		"{\"title\":\"\xff\xfe\"}", `{"title":"\ud800"}`, "\xef\xbb\xbf{}", `{"x":` + strings.Repeat("[", 200) + strings.Repeat("]", 200) + `}`,
		"Ey4.Aa", "%00", strings.Repeat("A", 2048) + "." + strings.Repeat("B", 64),
	}
	// Few seeds: under -fuzz every execution walks the coverage map of the
	// whole binary (SQLite included), so a large seed set would use up a
	// short CI fuzz run before any mutation starts.
	for i, rt := range routes {
		f.Add(uint8(i), []byte(rt.seed))
		if strings.HasPrefix(rt.seed, "{") {
			f.Add(uint8(i), []byte(rt.seed[:len(rt.seed)-1]+`,"unknown":true}`))
			f.Add(uint8(i), []byte(rt.seed+" x"))
		}
	}
	for i, b := range malformed {
		f.Add(uint8(i%len(routes)), []byte(b))
		f.Add(uint8((i+7)%len(routes)), []byte(b))
	}

	db := fx.Store.DB()
	f.Fuzz(func(t *testing.T, route uint8, body []byte) {
		rt := routes[int(route)%len(routes)]
		var hr *http.Request
		if rt.method == "GET" {
			hr = httptest.NewRequest("GET", "/api/v1"+rt.path+url.QueryEscape(string(body)), nil)
		} else {
			hr = httptest.NewRequest(rt.method, "/api/v1"+rt.path, bytes.NewReader(body))
			ct := "application/json; charset=utf-8"
			if rt.method == http.MethodPatch {
				ct = "application/merge-patch+json"
			}
			hr.Header.Set("Content-Type", ct)
			hr.Header.Set("Idempotency-Key", fx.Key())
			hr.Header.Set("If-Match", "*")
		}
		hr.Header.Set("Authorization", "Bearer "+tok)
		before := testutil.Snapshot(t, db)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, hr)
		after := testutil.Snapshot(t, db)
		if rec.Code >= 500 {
			t.Fatalf("%s %s with %q: %d %s", rt.method, rt.path, body, rec.Code, rec.Body.String())
		}
		if ch := before.Changed(after); len(ch) > 0 && (rec.Code >= 400 || rt.method == "GET") {
			t.Fatalf("%s %s refused %q with %d but changed %v", rt.method, rt.path, body, rec.Code, ch)
		}
	})
}
