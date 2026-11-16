package api_test

import "testing"

// SEC-032 over the API: open join mode with a sign-up grant is 422 at
// /confirm_open_grant_risk until confirm_open_grant_risk is true; the flag is
// not a policy field and must be a boolean.
func TestPolicyOpenGrantConfirmationAPI(t *testing.T) {
	w := newWorld(t)
	etag := w.get(w.adminT, "/admin/policy").Header().Get("ETag")
	body := m{"join_mode": "open", "signup_grant_minor": "250", "reason": "grow", "notice_days": 0, "notice_reason": "test network"}
	p := wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body,
		hdr: map[string]string{"If-Match": etag}}), 422, "validation-failed")
	if len(p.Errors) != 1 || p.Errors[0].Pointer != "/confirm_open_grant_risk" {
		t.Fatalf("errors = %+v", p.Errors)
	}
	body["confirm_open_grant_risk"] = "yes"
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body,
		hdr: map[string]string{"If-Match": etag}}), 422, "validation-failed")
	body["confirm_open_grant_risk"] = true
	rec := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, body: body, hdr: map[string]string{"If-Match": etag}})
	wantStatus(t, rec, 200)
	got := decode[struct {
		Policy map[string]any `json:"policy"`
	}](t, rec).Policy
	if got["join_mode"] != "open" || got["signup_grant_minor"] != "250" {
		t.Errorf("policy = %+v", got)
	}
	if _, leaked := got["confirm_open_grant_risk"]; leaked {
		t.Error("confirm_open_grant_risk returned as a policy field")
	}
}
