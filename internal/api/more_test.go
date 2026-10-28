package api_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
)

type page struct {
	Data       []map[string]any `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

func TestCursorPagination(t *testing.T) {
	w := newWorld(t)
	for i := 0; i < 7; i++ {
		wantStatus(t, w.post(w.aliceT, "/listings", m{"kind": "offer", "title": "item " + itoa(i)}, w.key()), 201)
	}
	seen := map[string]bool{}
	path := "/listings?limit=3"
	pages := 0
	var firstCursor string
	for path != "" {
		rec := w.get(w.bobT, path)
		wantStatus(t, rec, 200)
		pg := decode[page](t, rec)
		if len(pg.Data) > 3 {
			t.Fatalf("page too big: %d", len(pg.Data))
		}
		for _, it := range pg.Data {
			id := it["id"].(string)
			if seen[id] {
				t.Fatalf("duplicate %s", id)
			}
			seen[id] = true
		}
		pages++
		path = ""
		if pg.NextCursor != nil {
			if firstCursor == "" {
				firstCursor = *pg.NextCursor
			}
			path = "/listings?limit=3&cursor=" + *pg.NextCursor
		}
	}
	if len(seen) != 7 || pages != 3 {
		t.Fatalf("seen %d items in %d pages", len(seen), pages)
	}
	// An exact multiple ends with a null cursor.
	pg := decode[page](t, w.get(w.bobT, "/listings?limit=7"))
	if pg.NextCursor != nil || len(pg.Data) != 7 {
		t.Fatalf("limit=7: %d items cursor=%v", len(pg.Data), pg.NextCursor)
	}
	// Forged, truncated, or re-used with other filters -> invalid-cursor.
	wantProblem(t, w.get(w.bobT, "/listings?cursor=garbage"), 400, "invalid-cursor")
	forged := strings.SplitN(firstCursor, ".", 2)[0] + ".AAAA"
	wantProblem(t, w.get(w.bobT, "/listings?limit=3&cursor="+forged), 400, "invalid-cursor")
	wantProblem(t, w.get(w.bobT, "/listings?limit=3&kind=request&cursor="+firstCursor), 400, "invalid-cursor")
	wantProblem(t, w.get(w.aliceT, "/listings?limit=3&cursor="+firstCursor), 400, "invalid-cursor") // other caller
	// Bad parameters.
	for _, q := range []string{"limit=0", "limit=101", "limit=abc", "bogus=1", "kind=thing"} {
		wantProblem(t, w.get(w.bobT, "/listings?"+q), 400, "invalid-query")
	}
	// Filters.
	wantStatus(t, w.post(w.bobT, "/listings", m{"kind": "request", "title": "need ladder", "price_minor": "500"}, w.key()), 201)
	pg = decode[page](t, w.get(w.aliceT, "/listings?kind=request"))
	if len(pg.Data) != 1 || pg.Data[0]["price_minor"] != "500" {
		t.Fatalf("kind filter = %v", pg.Data)
	}
	pg = decode[page](t, w.get(w.aliceT, "/listings?q=ladder"))
	if len(pg.Data) != 1 {
		t.Fatalf("text filter = %v", pg.Data)
	}
	// Directory.
	pg = decode[page](t, w.get(w.aliceT, "/members?limit=2"))
	if len(pg.Data) != 2 || pg.NextCursor == nil || pg.Data[0]["status"] != "active" {
		t.Fatalf("members = %+v", pg)
	}
	wantStatus(t, w.get(w.aliceT, "/members/"+w.bobM.ID), 200)
	wantProblem(t, w.get(w.aliceT, "/members/nope"), 404, "not-found")
}

func TestPolicyAndListingETags(t *testing.T) {
	w := newWorld(t)
	rec := w.get(w.adminT, "/admin/policy")
	wantStatus(t, rec, 200)
	etag := rec.Header().Get("ETag")
	if !strings.HasPrefix(etag, `"`) || strings.HasPrefix(etag, "W/") {
		t.Fatalf("ETag = %q", etag)
	}
	patch := func(ifm string, body any) *bytes.Buffer {
		h := map[string]string{}
		if ifm != "" {
			h["If-Match"] = ifm
		}
		r := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body, hdr: h})
		return bytes.NewBuffer(append([]byte{byte('0' + r.Code/100), byte('0' + r.Code/10%10), byte('0' + r.Code%10)}, r.Body.Bytes()...))
	}
	_ = patch
	body := m{"reason": "raise limit", "default_credit_limit_minor": "20000"}
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body}), 428, "precondition-required")
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body, hdr: map[string]string{"If-Match": `"stale"`}}), 412, "precondition-failed")
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.aliceT, body: body, hdr: map[string]string{"If-Match": etag}}), 403, "forbidden")
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: m{"reason": "r", "nope": 1}, hdr: map[string]string{"If-Match": etag}}), 400, "invalid-body")
	p := wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: m{"default_credit_limit_minor": "20000"},
		hdr: map[string]string{"If-Match": etag}}), 422, "validation-failed") // reason is mandatory
	if len(p.Errors) == 0 {
		t.Fatal("no field errors")
	}
	rec = w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body, hdr: map[string]string{"If-Match": etag}})
	wantStatus(t, rec, 200)
	out := decode[struct {
		Version     int
		EffectiveAt string `json:"effective_at"`
		Policy      map[string]any
	}](t, rec)
	if out.Version != 2 || out.EffectiveAt == "" || out.Policy["default_credit_limit_minor"] != "20000" || out.Policy["max_tx_amount_minor"] != "100000000" {
		t.Fatalf("patched = %s", rec.Body.String())
	}
	if rec.Header().Get("ETag") == etag {
		t.Fatal("ETag must change")
	}
	// Reusing the old ETag now fails.
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body, hdr: map[string]string{"If-Match": etag}}), 412, "precondition-failed")
	// New members get the new default (policy applies to new calls).
	if a := w.acct(w.aliceT); a.CreditLimitMinor != "20000" {
		t.Fatalf("limit after policy change = %s", a.CreditLimitMinor)
	}

	// Listing PATCH with If-Match.
	rec = w.post(w.aliceT, "/listings", m{"kind": "offer", "title": "bike repair", "description": "d"}, w.key())
	wantStatus(t, rec, 201)
	l := decode[map[string]any](t, rec)
	le := rec.Header().Get("ETag")
	lp := "/api/v1/listings/" + l["id"].(string)
	wantProblem(t, w.do(req{method: "PATCH", path: lp, token: w.aliceT, body: m{"title": "x"}}), 428, "precondition-required")
	wantProblem(t, w.do(req{method: "PATCH", path: lp, token: w.aliceT, body: m{"title": "x"}, hdr: map[string]string{"If-Match": `"zzz"`}}), 412, "precondition-failed")
	wantProblem(t, w.do(req{method: "PATCH", path: lp, token: w.bobT, body: m{"title": "x"}, hdr: map[string]string{"If-Match": le}}), 404, "not-found")
	rec = w.do(req{method: "PATCH", path: lp, token: w.aliceT, body: m{"title": "bike fix", "price_minor": "700"}, hdr: map[string]string{"If-Match": le}})
	wantStatus(t, rec, 200)
	if decode[map[string]any](t, rec)["title"] != "bike fix" || rec.Header().Get("ETag") == le {
		t.Fatalf("patched listing: %s", rec.Body.String())
	}
	le2 := rec.Header().Get("ETag")
	rec = w.do(req{method: "PATCH", path: lp, token: w.aliceT, body: m{"price_minor": nil, "status": "paused"}, hdr: map[string]string{"If-Match": le2}})
	wantStatus(t, rec, 200)
	if got := decode[map[string]any](t, rec); got["price_minor"] != nil || got["status"] != "paused" {
		t.Fatalf("merge-patch null = %v", got)
	}
}

func TestPendingCurrencyNamePolicyPatchChangesETag(t *testing.T) {
	fixedNow := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	w := newWorldOpts(t, api.Options{CSRFKey: testCSRFKey, Now: func() time.Time { return fixedNow }})
	if !w.env.Clock.Now().Equal(fixedNow) {
		t.Fatalf("service clock = %s, want %s", w.env.Clock.Now(), fixedNow)
	}
	initial := w.get(w.adminT, "/admin/policy")
	wantStatus(t, initial, 200)
	oldETag := initial.Header().Get("ETag")
	if !strings.HasPrefix(oldETag, `"`) || strings.HasPrefix(oldETag, "W/") {
		t.Fatalf("initial ETag is not strong: %q", oldETag)
	}
	initialPolicy := decode[struct {
		Version int `json:"version"`
		Policy  struct {
			CurrencyName string `json:"currency_name"`
		} `json:"policy"`
	}](t, initial)

	body := m{"reason": "rename currency", "currency_name": "credits", "notice_days": 14}
	updated := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT,
		body: body, hdr: map[string]string{"If-Match": oldETag}})
	wantStatus(t, updated, 200)
	newETag := updated.Header().Get("ETag")
	out := decode[struct {
		Version int `json:"version"`
		Policy  struct {
			CurrencyName string `json:"currency_name"`
		} `json:"policy"`
		Pending *struct {
			Changes     map[string]any `json:"changes"`
			EffectiveAt string         `json:"effective_at"`
			Pending     bool           `json:"pending"`
		} `json:"pending"`
	}](t, updated)
	if out.Pending == nil || !out.Pending.Pending {
		t.Fatalf("pending policy missing from response: %s", updated.Body.String())
	}
	effectiveAt, err := time.Parse(time.RFC3339Nano, out.Pending.EffectiveAt)
	if err != nil || !effectiveAt.Equal(fixedNow.AddDate(0, 0, 14)) {
		t.Fatalf("pending effective_at = %q, want fixed-clock +14 days", out.Pending.EffectiveAt)
	}
	if out.Version != initialPolicy.Version || out.Policy.CurrencyName != initialPolicy.Policy.CurrencyName {
		t.Fatalf("active policy changed before effective_at: version %d->%d currency %q->%q",
			initialPolicy.Version, out.Version, initialPolicy.Policy.CurrencyName, out.Policy.CurrencyName)
	}
	if out.Pending.Changes["currency_name"] != "credits" {
		t.Fatalf("pending currency_name = %v, want credits", out.Pending.Changes["currency_name"])
	}
	if newETag == "" {
		t.Errorf("pending response omitted ETag")
	} else if newETag == oldETag {
		t.Errorf("pending currency-name-only update did not change ETag: old=%q new=%q", oldETag, newETag)
	}
	if !strings.HasPrefix(newETag, `"`) || strings.HasPrefix(newETag, "W/") {
		t.Errorf("updated ETag is not strong: %q", newETag)
	}

	current := w.get(w.adminT, "/admin/policy")
	wantStatus(t, current, 200)
	if current.Header().Get("ETag") != newETag {
		t.Errorf("GET ETag = %q, want pending ETag %q", current.Header().Get("ETag"), newETag)
	}

	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT,
		body: m{"reason": "stale rename", "currency_name": "marks", "notice_days": 14},
		hdr:  map[string]string{"If-Match": oldETag}}), 412, "precondition-failed")
	afterStale := w.get(w.adminT, "/admin/policy")
	wantStatus(t, afterStale, 200)
	if afterStale.Header().Get("ETag") != newETag {
		t.Fatalf("stale update changed ETag: got %q, want %q", afterStale.Header().Get("ETag"), newETag)
	}
	after := decode[struct {
		Version int `json:"version"`
		Policy  struct {
			CurrencyName string `json:"currency_name"`
		} `json:"policy"`
		Pending *struct {
			Changes map[string]any `json:"changes"`
		} `json:"pending"`
	}](t, afterStale)
	if after.Pending == nil || after.Pending.Changes["currency_name"] != "credits" ||
		after.Version != initialPolicy.Version || after.Policy.CurrencyName != initialPolicy.Policy.CurrencyName {
		t.Fatalf("stale request changed active or pending policy: %s", afterStale.Body.String())
	}
}

func TestCreditLimitAndMemberAdmin(t *testing.T) {
	w := newWorld(t)
	rec := w.get(w.adminT, "/admin/accounts/"+w.aliceM.ID)
	wantStatus(t, rec, 200)
	etag := rec.Header().Get("ETag")
	put := func(tok string, h map[string]string, body any) int {
		return w.do(req{method: "PUT", path: "/api/v1/admin/accounts/" + w.aliceM.ID + "/credit-limit", token: tok, body: body, hdr: h}).Code
	}
	body := m{"credit_limit_minor": "500", "reason": "risk review"}
	if c := put(w.adminT, nil, body); c != 428 {
		t.Fatalf("no If-Match -> %d", c)
	}
	if c := put(w.adminT, map[string]string{"If-Match": `"old"`}, body); c != 412 {
		t.Fatalf("stale -> %d", c)
	}
	if c := put(w.aliceT, map[string]string{"If-Match": etag}, body); c != 403 {
		t.Fatalf("member -> %d", c)
	}
	wantProblem(t, w.do(req{method: "PUT", path: "/api/v1/admin/accounts/" + w.aliceM.ID + "/credit-limit", token: w.adminT,
		body: m{"credit_limit_minor": "-5", "reason": "x"}, hdr: map[string]string{"If-Match": etag}}), 422, "validation-failed")
	if c := put(w.adminT, map[string]string{"If-Match": etag}, body); c != 200 {
		t.Fatalf("put -> %d", c)
	}
	if a := w.acct(w.aliceT); a.CreditLimitMinor != "500" {
		t.Fatalf("limit = %s", a.CreditLimitMinor)
	}

	// Member lifecycle with audit; suspended members cannot propose.
	wantStatus(t, w.post(w.adminT, "/admin/members/"+w.bobM.ID+"/suspend", m{"reason": "review"}, w.key()), 200)
	wantProblem(t, w.post(w.bobT, "/transactions", m{"counterparty_member_id": w.aliceM.ID, "direction": "pay", "amount_minor": "1"}, w.key()), 403, "forbidden")
	wantStatus(t, w.get(w.bobT, "/accounts/me"), 200) // suspended may read own data
	rec = w.post(w.adminT, "/admin/members/"+w.bobM.ID+"/reinstate", m{"reason": "ok"}, w.key())
	wantStatus(t, rec, 200)
	if decode[map[string]any](t, rec)["status"] != "active" {
		t.Fatal("not reinstated")
	}
	pg := decode[page](t, w.get(w.adminT, "/admin/members?status=active"))
	if len(pg.Data) != 3 {
		t.Fatalf("active members = %d", len(pg.Data))
	}
	wantProblem(t, w.get(w.adminT, "/admin/members?status=weird"), 400, "invalid-query")
	rec = w.post(w.adminT, "/admin/adjustments", m{"member_id": w.bobM.ID, "direction": "credit", "amount_minor": "250", "reason": "gift"}, w.key())
	wantStatus(t, rec, 201)
	if a := w.acct(w.bobT); a.BalanceMinor != "250" {
		t.Fatalf("bob = %+v", a)
	}

	// Audit trail is paginated and filterable, and admin-only.
	pg = decode[page](t, w.get(w.adminT, "/admin/audit-events?limit=2"))
	if len(pg.Data) != 2 || pg.NextCursor == nil {
		t.Fatalf("audit page = %+v", pg)
	}
	next := decode[page](t, w.get(w.adminT, "/admin/audit-events?limit=2&cursor="+*pg.NextCursor))
	if len(next.Data) == 0 || next.Data[0]["seq"].(float64) <= pg.Data[1]["seq"].(float64) {
		t.Fatalf("audit paging not advancing: %v then %v", pg.Data, next.Data)
	}
	pg = decode[page](t, w.get(w.adminT, "/admin/audit-events?action=adjustment.posted"))
	if len(pg.Data) != 1 {
		t.Fatalf("filtered audit = %d", len(pg.Data))
	}
	wantProblem(t, w.get(w.adminT, "/admin/audit-events?from=yesterday"), 400, "invalid-query")
}

func TestInvitationsDisputesDepartureExport(t *testing.T) {
	w := newWorld(t)
	rec := w.post(w.adminT, "/admin/invitations", m{}, w.key())
	wantStatus(t, rec, 201)
	inv := decode[struct{ ID, Token string }](t, rec)
	if !strings.HasPrefix(inv.Token, "inv_") {
		t.Fatalf("invitation = %s", rec.Body.String())
	}
	carol := w.env.Person("carol")
	ct := w.token(carol, "read", "write")
	rec = w.post(ct, "/invitations/"+inv.Token+"/accept", m{"display_name": "Carol"}, w.key())
	wantStatus(t, rec, 201)
	dave := w.token(w.env.Person("dave"), "read", "write")
	if rec := w.post(dave, "/invitations/"+inv.Token+"/accept", m{"display_name": "Dave"}, w.key()); rec.Code < 400 {
		t.Fatalf("used invitation accepted twice: %s", rec.Body.String())
	}

	// Dispute lifecycle (reviewer = admin, not a party).
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 1000)
	rec = w.post(w.aliceT, "/disputes", m{"transaction_id": tx.ID, "category": "not_delivered", "statement": "nothing came"}, w.key())
	wantStatus(t, rec, 201)
	d := decode[map[string]any](t, rec)["id"].(string)
	wantProblem(t, w.post(w.aliceT, "/disputes", m{"transaction_id": tx.ID, "category": "other", "statement": "again"}, w.key()), 409, "invalid-transaction-state")
	wantStatus(t, w.get(w.bobT, "/disputes/"+d), 200)
	wantStatus(t, w.get(w.adminT, "/disputes/"+d), 200)
	wantProblem(t, w.get(w.env2Outsider(t, w), "/disputes/"+d), 404, "not-found")
	wantProblem(t, w.post(w.aliceT, "/disputes/"+d+"/resolve", m{"outcome": "reversal", "reason": "me"}, w.key()), 403, "forbidden")
	wantProblem(t, w.post(w.adminT, "/disputes/"+d+"/resolve", m{"outcome": "adjustment", "reason": "x"}, w.key()), 501, "not-implemented")
	wantProblem(t, w.post(w.adminT, "/disputes/"+d+"/resolve", m{"outcome": "bogus", "reason": "x"}, w.key()), 422, "validation-failed")
	rec = w.post(w.adminT, "/disputes/"+d+"/resolve", m{"outcome": "reversal", "reason": "agreed it was not delivered"}, w.key())
	wantStatus(t, rec, 200)
	if got := decode[map[string]any](t, rec); got["status"] != "resolved" || got["outcome"] != "upheld" {
		t.Fatalf("resolved = %v", got)
	}
	if a := w.acct(w.aliceT); a.BalanceMinor != "0" {
		t.Fatalf("alice after dispute = %+v", a)
	}
	pg := decode[page](t, w.get(w.aliceT, "/disputes?status=resolved"))
	if len(pg.Data) != 1 {
		t.Fatalf("disputes = %v", pg.Data)
	}

	// Export (own data) as zip, checksum verified.
	rec = w.post(w.aliceT, "/exports", m{"scope": "mine", "format": "csv"}, "")
	wantStatus(t, rec, 201)
	ex := decode[struct {
		ID, Status, SHA256 string
		ArtefactURL        string `json:"artefact_url"`
	}](t, rec)
	if ex.Status != "complete" {
		t.Fatalf("export = %s", rec.Body.String())
	}
	art := w.do(req{method: "GET", path: ex.ArtefactURL, token: w.aliceT})
	wantStatus(t, art, 200)
	if art.Header().Get("X-Content-SHA256") != ex.SHA256 {
		t.Fatal("checksum header mismatch")
	}
	zr, err := zip.NewReader(bytes.NewReader(art.Body.Bytes()), int64(art.Body.Len()))
	if err != nil || len(zr.File) == 0 {
		t.Fatalf("bad zip: %v", err)
	}
	wantProblem(t, w.do(req{method: "GET", path: ex.ArtefactURL, token: w.bobT}), 404, "not-found")
	wantProblem(t, w.post(w.aliceT, "/exports", m{"scope": "everything"}, ""), 422, "validation-failed")
	wantProblem(t, w.post(w.aliceT, "/exports", m{"scope": "audit"}, ""), 403, "forbidden")

	// Departure.
	rec = w.post(w.bobT, "/me/membership/departure", m{"reason": "moving"}, w.key())
	wantStatus(t, rec, 200)
	if got := decode[map[string]any](t, rec); got["status"] != "departed" || got["settlement"] != "settled" {
		t.Fatalf("departure = %v", got)
	}
	wantProblem(t, w.post(w.adminT, "/me/membership/departure", nil, w.key()), 409, "blocked") // last admin
}

func (w *world) env2Outsider(t *testing.T, _ *world) string {
	eve := w.env.Person("eve2")
	return w.token(eve, "read", "write")
}
