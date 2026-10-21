package api_test

import (
	"strings"
	"testing"
)

func TestStrictBodyHandling(t *testing.T) {
	w := newWorld(t)
	post := func(raw, ctype string) (code int, p problem) {
		rec := w.do(req{method: "POST", path: "/api/v1/transactions", token: w.aliceT, raw: raw, ctype: ctype,
			hdr: map[string]string{"Idempotency-Key": w.key()}})
		if rec.Code >= 400 {
			p = decode[problem](t, rec)
		}
		return rec.Code, p
	}
	good := `{"counterparty_member_id":"` + w.bobM.ID + `","direction":"pay","amount_minor":"100"`
	cases := []struct {
		name, raw, ctype string
		status           int
		code             string
	}{
		{"unknown field", good + `,"extra":1}`, "", 400, "invalid-body"},
		{"client-supplied balance", good + `,"balance_minor":"5"}`, "", 400, "invalid-body"},
		{"duplicate key", `{"direction":"pay","direction":"receive"}`, "", 400, "invalid-body"},
		{"malformed", `{"direction":`, "", 400, "invalid-body"},
		{"trailing data", good + `} {}`, "", 400, "invalid-body"},
		{"number for money", strings.Replace(good, `"100"`, `100`, 1) + `}`, "", 400, "invalid-body"},
		{"wrong content type", good + `}`, "text/plain", 415, "unsupported-media-type"},
		{"wrong charset", good + `}`, "application/json; charset=latin1", 415, "unsupported-media-type"},
		{"decimal point", strings.Replace(good, `"100"`, `"12.50"`, 1) + `}`, "", 422, "validation-failed"},
		{"leading zero", strings.Replace(good, `"100"`, `"0100"`, 1) + `}`, "", 422, "validation-failed"},
		{"negative", strings.Replace(good, `"100"`, `"-100"`, 1) + `}`, "", 422, "validation-failed"},
		{"plus sign", strings.Replace(good, `"100"`, `"+100"`, 1) + `}`, "", 422, "validation-failed"},
		{"exponent", strings.Replace(good, `"100"`, `"1e3"`, 1) + `}`, "", 422, "validation-failed"},
		{"whitespace", strings.Replace(good, `"100"`, `" 100"`, 1) + `}`, "", 422, "validation-failed"},
		{"zero", strings.Replace(good, `"100"`, `"0"`, 1) + `}`, "", 422, "validation-failed"},
		{"overflow", strings.Replace(good, `"100"`, `"9223372036854775808"`, 1) + `}`, "", 422, "validation-failed"},
		{"bad direction", strings.Replace(good, `"pay"`, `"give"`, 1) + `}`, "", 422, "validation-failed"},
	}
	for _, tc := range cases {
		code, p := post(tc.raw, tc.ctype)
		if code != tc.status || p.Code != tc.code {
			t.Errorf("%s: got %d %s (%s), want %d %s", tc.name, code, p.Code, p.Detail, tc.status, tc.code)
		}
	}
	// Validation errors carry JSON pointers into the body (API-030).
	_, p := post(strings.Replace(good, `"100"`, `"12.50"`, 1)+`}`, "")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/amount_minor" || p.Errors[0].Code == "" {
		t.Fatalf("errors = %+v", p.Errors)
	}
	// Service-level field errors are translated to API field names.
	_, p = post(`{"counterparty_member_id":"`+w.aliceM.ID+`","direction":"pay","amount_minor":"5"}`, "")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/counterparty_member_id" {
		t.Fatalf("service errors = %+v", p.Errors)
	}
}

func TestBodySizeLimit(t *testing.T) {
	w := newWorld(t)
	big := `{"kind":"offer","title":"` + strings.Repeat("a", 70<<10) + `"}`
	rec := w.do(req{method: "POST", path: "/api/v1/listings", token: w.aliceT, raw: big, hdr: map[string]string{"Idempotency-Key": w.key()}})
	wantProblem(t, rec, 413, "payload-too-large")
}

func TestPatchRequiresMergePatchType(t *testing.T) {
	w := newWorld(t)
	rec := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, raw: `{"reason":"x"}`, ctype: "application/json",
		hdr: map[string]string{"If-Match": `"x"`}})
	wantProblem(t, rec, 415, "unsupported-media-type")
}

func TestNotImplementedFeatures(t *testing.T) {
	w := newWorld(t)
	wantProblem(t, w.get(w.adminT, "/admin/announcements"), 501, "not-implemented")
	wantProblem(t, w.post(w.adminT, "/admin/announcements", m{"title": "x"}, w.key()), 501, "not-implemented")
	wantProblem(t, w.get(w.aliceT, "/network/announcements"), 501, "not-implemented")
}

func TestRateLimit(t *testing.T) {
	w := newWorldOpts(t, optsRate(3))
	for i := 0; i < 3; i++ {
		wantStatus(t, w.get(w.aliceT, "/network"), 200)
	}
	rec := w.get(w.aliceT, "/network")
	wantProblem(t, rec, 429, "rate-limited")
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	// Another principal has its own budget.
	wantStatus(t, w.get(w.bobT, "/network"), 200)
}
