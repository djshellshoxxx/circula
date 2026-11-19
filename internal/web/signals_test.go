package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-026: the activity signal page shows a network's signals to its
// administrators and dispute reviewers in complete catalogue sentences, with
// member names escaped, and refuses plain members. Administrators are told in
// their inbox and find the page from the overview and the review queue.
func TestSEC026ActivitySignalsPage(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	if _, err := w.env.Store.DB().Exec(`UPDATE members SET display_name = ? WHERE id = ?`, "<b>Alice</b>", w.aliceM.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < domain.DefaultVelocitySignalProposals+1; i++ {
		testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{CounterpartyID: w.bobM.ID,
			Direction: service.ProposerPays, Amount: 1, IdempotencyKey: w.env.Key()}))
	}
	adm := w.browser(t)
	adm.login("admin@example.org")
	p := adm.get(b + "/review/activity-signals")
	if p.Status != 200 {
		t.Fatalf("administrator: status %d", p.Status)
	}
	adm.mustContain(p, `<h1 id="page-title">Activity signals</h1>`,
		"&lt;b&gt;Alice&lt;/b&gt; made 21 trade proposals within a 60-minute window, above the review threshold of 20.",
		`<span class="tag">Many proposals</span>`,
		"exceed 20, or when the trades two members complete with each other, in both directions, within a 30-day window exceed 10.",
		"A signal is not a finding")
	checkStructure(t, "activity signals", p)
	if strings.Contains(p.Body, "<b>Alice</b>") {
		t.Fatal("member name rendered as markup")
	}
	adm.mustContain(adm.get(b+"/review/activity-signals?kind=reciprocal"), "No activity signals to show.", `aria-current="page">Back-and-forth trades`)
	adm.mustContain(adm.get(b+"/admin"), `href="/n/town/review/activity-signals"`)
	adm.mustContain(adm.get(b+"/review"), `href="/n/town/review/activity-signals"`)
	adm.mustContain(adm.get(b+"/inbox"), "Activity to review", `href="/n/town/review/activity-signals"`)

	bob := w.browser(t)
	bob.login("bob@example.org")
	if p := bob.get(b + "/review/activity-signals"); p.Status != 403 {
		t.Fatalf("plain member: status %d", p.Status)
	}
	bob.mustNotContain(bob.get(b+"/inbox"), "Activity to review")
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.bobM.ID, domain.RoleReviewer))
	bob.mustContain(bob.get(b+"/review/activity-signals"), "made 21 trade proposals")
	// The subject never sees their own signal, even as a reviewer.
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, domain.RoleReviewer))
	al := w.browser(t)
	al.login("alice@example.org")
	al.mustContain(al.get(b+"/review/activity-signals"), "No activity signals to show.")
}

// SEC-026: the policy form shows and changes the signal settings. A form
// without the signal fieldset leaves them unchanged; out-of-bounds values are
// refused with the field named and nothing saved.
func TestSEC026PolicyFormSignalSettings(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	pf := adm.get(b + "/admin/policy")
	adm.mustContain(pf, "<legend>Activity signals</legend>", `name="activity_signals_form" value="1"`,
		`name="activity_signals_enabled" value="1" checked`, `id="f-velocity_signal_proposals" name="velocity_signal_proposals" type="text" value="20"`,
		`name="velocity_signal_window_minutes" type="text" value="60"`, `name="reciprocal_signal_trades" type="text" value="10"`,
		`name="reciprocal_signal_window_days" type="text" value="30"`)
	base := url.Values{"currency_name": {"credit"}, "default_credit_limit": {"100.00"}, "max_tx_amount": {"1000000.00"},
		"signup_grant": {"0"}, "proposal_expiry_days": {"7"}, "dispute_window_days": {"30"}, "limit_warning_days": {"0"},
		"response_period_days": {"0"}, "re_review_window_days": {"14"}, "limit_warning_percent": {"80"},
		"join_mode": {"invite"}, "departure_rule": {"retain"}, "reason": {"signal review"}}
	with := func(kv ...string) url.Values {
		v := url.Values{}
		for k, x := range base {
			v[k] = x
		}
		for i := 0; i+1 < len(kv); i += 2 {
			v.Set(kv[i], kv[i+1])
		}
		return v
	}
	policy := func() domain.Policy {
		return testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)).Policy
	}
	// Out of bounds: refused, nothing saved.
	bad := adm.post(b+"/admin/policy", pf, with("activity_signals_form", "1", "activity_signals_enabled", "1",
		"velocity_signal_proposals", "0", "velocity_signal_window_minutes", "60", "reciprocal_signal_trades", "10", "reciprocal_signal_window_days", "30"))
	adm.mustContain(bad, `id="error-summary"`, `href="#f-velocity_signal_proposals"`)
	notNumber := adm.post(b+"/admin/policy", pf, with("activity_signals_form", "1", "velocity_signal_proposals", "many",
		"velocity_signal_window_minutes", "60", "reciprocal_signal_trades", "10", "reciprocal_signal_window_days", "30"))
	adm.mustContain(notNumber, `id="error-summary"`, "enter a whole number")
	if p := policy(); p.VelocitySignalProposals != 20 || !p.ActivitySignalsEnabled {
		t.Fatalf("a refused form changed the policy: %+v", p)
	}
	// The rendered form, box unticked and thresholds changed.
	saved := adm.post(b+"/admin/policy", pf, with("activity_signals_form", "1", "velocity_signal_proposals", "5",
		"velocity_signal_window_minutes", "15", "reciprocal_signal_trades", "4", "reciprocal_signal_window_days", "7"))
	adm.mustContain(saved, "Policy saved as a new version", `name="velocity_signal_proposals" type="text" value="5"`)
	if p := policy(); p.ActivitySignalsEnabled || p.VelocitySignalProposals != 5 || p.VelocitySignalWindowMinutes != 15 ||
		p.ReciprocalSignalTrades != 4 || p.ReciprocalSignalWindowDays != 7 {
		t.Fatalf("saved policy %+v", p)
	}
	// Without the fieldset marker the signal settings stay as they are.
	pf = adm.get(b + "/admin/policy")
	adm.mustContain(adm.post(b+"/admin/policy", pf, with("currency_name", "hours")), "Policy saved as a new version")
	if p := policy(); p.CurrencyName != "hours" || p.ActivitySignalsEnabled || p.VelocitySignalProposals != 5 {
		t.Fatalf("a form without the signal fieldset changed signal settings: %+v", p)
	}
	// Members' rules page does not show the signal thresholds.
	al := w.browser(t)
	al.login("alice@example.org")
	al.mustNotContain(al.get(b+"/policy"), "velocity_signal", "reciprocal_signal", "activity_signals", "Activity signals")
}
