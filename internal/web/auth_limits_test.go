package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// SEC-006: without Options.Limits the HTML adapter keeps the limits it had
// before they became configurable.
func TestDefaultRateLimitsUnchanged(t *testing.T) {
	w := newWorld(t)
	s := newServer(w.env.Svc, Options{})
	for name, got := range map[string]*limiter{"ipFail": s.ipFail, "acctFail": s.acctFail, "pwFail": s.pwFail,
		"reg": s.reg, "setup": s.setup, "join": s.join} {
		want := map[string]Limit{"ipFail": {20, time.Minute}, "acctFail": {5, time.Minute}, "pwFail": {5, 10 * time.Minute},
			"reg": {10, time.Minute}, "setup": {10, 10 * time.Minute}, "join": {10, 10 * time.Minute}}[name]
		if got.max != want.Max || got.window != want.Window {
			t.Errorf("%s = %d/%s, want %d/%s", name, got.max, got.window, want.Max, want.Window)
		}
	}
}

// SEC-006: the operator's limits reach the adapter; the join limit's
// Retry-After follows its window.
func TestConfiguredRateLimits(t *testing.T) {
	w := newWorldWith(t, func(o *Options) {
		o.Limits.Register = Limit{Max: 1, Window: time.Minute}
		o.Limits.Join = Limit{Max: 1, Window: 90 * time.Second}
	})
	b := w.browser(t)
	f := b.get("/register")
	vals := url.Values{"login": {"new1@example.org"}, "password": {testPW}, "password2": {testPW}}
	if p := b.post("/register", f, vals); p.Status == 429 {
		t.Fatal("first registration throttled")
	}
	vals.Set("login", "new2@example.org")
	if p := b.post("/register", f, vals); p.Status != 429 {
		t.Errorf("second registration within the configured window: %d, want 429", p.Status)
	}

	w.env.Person("zed@example.org")
	w.env.Network(w.admin, "appr", func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
	z := w.browser(t)
	z.login("zed@example.org")
	jf := z.get("/n/appr/join")
	z.post("/n/appr/join", jf, url.Values{"display_name": {"Zed"}})
	p := z.post("/n/appr/join", jf, url.Values{"display_name": {"Zed"}})
	if p.Status != 429 || p.Header.Get("Retry-After") != "90" {
		t.Errorf("second join: %d Retry-After %q, want 429 and 90", p.Status, p.Header.Get("Retry-After"))
	}
}

// SEC-032 in the web forms: the policy form and the setup form refuse open
// joining with a sign-up grant until the confirmation box is ticked.
func TestOpenGrantConfirmationForms(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	pf := adm.get(b + "/admin/policy")
	np := url.Values{"currency_name": {"credit"}, "default_credit_limit": {"100.00"}, "max_tx_amount": {"1000.00"}, "signup_grant": {"5.00"},
		"proposal_expiry_days": {"7"}, "dispute_window_days": {"30"}, "limit_warning_days": {"0"}, "response_period_days": {"0"},
		"re_review_window_days": {"14"}, "limit_warning_percent": {"80"}, "join_mode": {"open"}, "departure_rule": {"retain"},
		"reason": {"grow the network"}, "notice_days": {"0"}, "notice_reason": {"test network"}}
	bad := adm.post(b+"/admin/policy", pf, np)
	adm.mustContain(bad, `href="#f-confirm_open_grant_risk"`, `id="e-confirm_open_grant_risk"`, `aria-invalid="true"`)
	if n, _ := w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID); n.PolicyVersion != 1 {
		t.Fatalf("refused policy saved: version %d", n.PolicyVersion)
	}
	np.Set("confirm_open_grant_risk", "1")
	adm.mustContain(adm.post(b+"/admin/policy", pf, np), "Policy saved as a new version")
	if n, _ := w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID); n.Policy.JoinMode != domain.JoinOpen || n.Policy.SignupGrant != 500 {
		t.Errorf("confirmed policy: %+v", n.Policy)
	}

	w.env.Person("founder@example.org")
	f := w.browser(t)
	f.login("founder@example.org")
	form := f.get("/setup")
	vals := setupVals("", "opengrant")
	vals.Set("join_mode", "open")
	vals.Set("signup_grant", "1.00")
	p := f.post("/setup", form, vals)
	if p.Status != 422 || !strings.Contains(p.Body, `href="#f-confirm_open_grant_risk"`) || hasNetwork(w, "opengrant") {
		t.Fatalf("setup without confirmation: %d, network created %v", p.Status, hasNetwork(w, "opengrant"))
	}
	vals.Set("confirm_open_grant_risk", "1")
	if p := f.post("/setup", form, vals); p.URL != "/n/opengrant/" {
		t.Errorf("setup with confirmation: %d %s", p.Status, p.URL)
	}
}
