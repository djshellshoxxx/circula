package web

import (
	"net/url"
	"testing"
	"time"
)

// GOV-012: the administrator page lists members the inactivity rule flags, and a
// plain member cannot open it.
func TestInactivePage(t *testing.T) {
	w := newWorld(t)
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get("/n/town/admin/inactive"), "no inactivity rule")

	f := adm.get("/n/town/admin/policy")
	vals := url.Values{"currency_name": {"credit"}, "default_credit_limit": {"100.00"}, "max_tx_amount": {"1000000.00"},
		"signup_grant": {"0"}, "proposal_expiry_days": {"7"}, "dispute_window_days": {"30"}, "limit_warning_days": {"0"},
		"response_period_days": {"0"}, "re_review_window_days": {"14"}, "limit_warning_percent": {"80"}, "inactivity_days": {"60"},
		"join_mode": {"invite"}, "departure_rule": {"retain"}, "reason": {"test"}, "notice_days": {"0"}, "notice_reason": {"test"}}
	adm.mustContain(adm.post("/n/town/admin/policy", f, vals), "Policy saved")

	w.env.Clock.Advance(90 * 24 * time.Hour)
	adm2 := w.browser(t) // the first session has idled out
	adm2.login("admin@example.org")
	adm2.mustContain(adm2.get("/n/town/admin/inactive"), "60 days", "Inactive members of", "bob")

	al := w.browser(t)
	al.login("alice@example.org")
	if r := al.get("/n/town/admin/inactive"); r.Status != 403 && r.Status != 404 {
		t.Errorf("member status %d", r.Status)
	}
}
