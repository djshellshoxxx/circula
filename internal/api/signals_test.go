package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

type signalPage struct {
	Data []struct {
		ID, Kind, MemberID, MemberName string
		OtherMemberID                  *string `json:"other_member_id"`
		ObservedCount                  int     `json:"observed_count"`
		Threshold                      int     `json:"threshold"`
		WindowMinutes                  int     `json:"window_minutes"`
		PeriodStart                    string  `json:"period_start"`
		PeriodEnd                      string  `json:"period_end"`
	} `json:"data"`
	Next *string `json:"next_cursor"`
}

// SEC-026: GET /activity-signals pages with the signed keyset cursor, bound
// to the filter that produced it; the policy route exposes the thresholds to
// administrators and PATCH changes them under the policy rules. Signals of
// another network never appear, and members of it get not-found.
func TestSEC026ActivitySignalsAPI(t *testing.T) {
	w := newWorld(t)
	burst := func(by domain.Actor, to *domain.Member) {
		for i := 0; i <= domain.DefaultVelocitySignalProposals; i++ {
			testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, by, w.net.ID, service.ProposeInput{CounterpartyID: to.ID,
				Direction: service.ProposerPays, Amount: 1, IdempotencyKey: w.key()}))
		}
	}
	carol, _ := w.env.Member(w.admin, w.net, "carol")
	burst(w.alice, w.bobM)
	burst(w.bob, w.aliceM)
	burst(carol, w.aliceM)
	fran := w.env.Person("fran")
	w.env.Network(fran, "other", nil)
	franT := w.token(fran, domain.ScopeRead)

	var seen []string
	path := "/activity-signals?limit=1"
	for i := 0; i < 5; i++ {
		rec := w.get(w.adminT, path)
		if rec.Code != 200 {
			t.Fatalf("page %d: %d %s", i, rec.Code, rec.Body.String())
		}
		p := decode[signalPage](t, rec)
		for _, s := range p.Data {
			seen = append(seen, s.ID)
			if s.Kind != domain.SignalVelocity || s.OtherMemberID != nil || s.ObservedCount != 21 || s.Threshold != 20 || s.WindowMinutes != 60 {
				t.Errorf("signal %+v", s)
			}
			start, err1 := time.Parse(time.RFC3339, s.PeriodStart)
			end, err2 := time.Parse(time.RFC3339, s.PeriodEnd)
			if err1 != nil || err2 != nil || end.Sub(start) != time.Hour {
				t.Errorf("period %s..%s", s.PeriodStart, s.PeriodEnd)
			}
		}
		if p.Next == nil {
			break
		}
		path = "/activity-signals?limit=1&cursor=" + url.QueryEscape(*p.Next)
	}
	if len(seen) != 3 || seen[0] < seen[1] || seen[1] < seen[2] {
		t.Fatalf("pages returned %v, want three signals newest first", seen)
	}
	first := decode[signalPage](t, w.get(w.adminT, "/activity-signals?limit=1"))
	wantProblem(t, w.get(w.adminT, "/activity-signals?kind=velocity&limit=1&cursor="+url.QueryEscape(*first.Next)), 400, "invalid-cursor")
	wantProblem(t, w.get(w.adminT, "/activity-signals?kind=everything"), 400, "invalid-query")
	wantProblem(t, w.get(w.adminT, "/activity-signals?status=open"), 400, "invalid-query")
	// The subjects see none of their own signals; plain members are refused.
	testutil.Must(0, w.env.Svc.AssignRole(w.env.Ctx, w.admin, w.net.ID, w.aliceM.ID, domain.RoleReviewer))
	if p := decode[signalPage](t, w.get(w.aliceT, "/activity-signals")); len(p.Data) != 2 {
		t.Fatalf("reviewer alice sees %d signals, want the two not about her", len(p.Data))
	}
	wantProblem(t, w.get(w.bobT, "/activity-signals"), 403, "forbidden")
	// Another network's administrator naming this network gets not-found, and
	// sees only her own network's (empty) list.
	wantProblem(t, w.do(req{method: "GET", path: "/api/v1/activity-signals", token: franT, hdr: map[string]string{"X-Network": "acme"}}), 404, "not-found")
	if p := decode[signalPage](t, w.get(franT, "/activity-signals")); len(p.Data) != 0 {
		t.Fatalf("other network's administrator sees %d signals", len(p.Data))
	}
	wantProblem(t, w.get(w.token(w.env.Person("guest"), domain.ScopeRead), "/activity-signals"), 404, "not-found")

	// Policy: thresholds are visible to administrators and patched like any policy field.
	rec := w.get(w.adminT, "/admin/policy")
	pol := decode[struct {
		Policy map[string]any `json:"policy"`
	}](t, rec)
	for k, want := range map[string]any{"activity_signals_enabled": true, "velocity_signal_proposals": 20.0,
		"velocity_signal_window_minutes": 60.0, "reciprocal_signal_trades": 10.0, "reciprocal_signal_window_days": 30.0} {
		if pol.Policy[k] != want {
			t.Errorf("policy %s = %v, want %v", k, pol.Policy[k], want)
		}
	}
	etag := rec.Header().Get("ETag")
	bad := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, hdr: map[string]string{"If-Match": etag},
		body: m{"reason": "tune", "reciprocal_signal_window_days": 366}})
	if p := wantProblem(t, bad, 422, "validation-failed"); len(p.Errors) != 1 || p.Errors[0].Pointer != "/reciprocal_signal_window_days" {
		t.Fatalf("errors %+v", p.Errors)
	}
	ok := w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.adminT, hdr: map[string]string{"If-Match": etag},
		body: m{"reason": "tune", "activity_signals_enabled": false, "velocity_signal_proposals": 50}})
	if ok.Code != 200 {
		t.Fatalf("patch: %d %s", ok.Code, ok.Body.String())
	}
	if n := testutil.Must(w.env.Svc.GetNetwork(w.env.Ctx, w.admin, w.net.ID)); n.Policy.ActivitySignalsEnabled || n.Policy.VelocitySignalProposals != 50 {
		t.Fatalf("patched policy %+v", n.Policy)
	}
	wantProblem(t, w.do(req{method: "PATCH", path: "/api/v1/admin/policy", token: w.aliceT, hdr: map[string]string{"If-Match": "*"},
		body: m{"reason": "x", "activity_signals_enabled": true}}), 403, "forbidden")
}
