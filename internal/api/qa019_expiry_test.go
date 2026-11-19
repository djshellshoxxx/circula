package api_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-019 / OPS-027 / FR-038: real API proposal and decline requests, followed
// by the scheduled expiry service entry point, expose terminal states and
// exactly one expiry notice per party without changing either ledger.
func TestQA019DeclineAndExpiryAPI(t *testing.T) {
	w := newWorld(t)
	before := map[string]acct{w.aliceT: w.acct(w.aliceT), w.bobT: w.acct(w.bobT)}
	propose := func(token, counterparty, direction, amount string) txResp {
		t.Helper()
		r := w.post(token, "/transactions", m{"counterparty_member_id": counterparty,
			"direction": direction, "amount_minor": amount}, w.key())
		wantStatus(t, r, 201)
		return decode[txResp](t, r)
	}
	declined := propose(w.aliceT, w.bobM.ID, "pay", "700")
	r := w.post(w.bobT, "/transactions/"+declined.ID+"/decline", nil, w.key())
	wantStatus(t, r, 200)
	if got := decode[txResp](t, r); got.Status != "declined" {
		t.Fatalf("decline response = %+v", got)
	}
	expiring := propose(w.bobT, w.aliceM.ID, "receive", "400")

	w.env.Clock.Advance(time.Duration(w.net.Policy.ProposalExpiryDays)*24*time.Hour - time.Minute)
	if n := testutil.Must(w.env.Svc.ExpireProposals(w.env.Ctx, domain.SystemActor("qa019-expiry"), 100)); n != 0 {
		t.Fatalf("expired %d before the deadline", n)
	}
	w.env.Clock.Advance(2 * time.Minute)
	for _, want := range []int{1, 0} {
		if n := testutil.Must(w.env.Svc.ExpireProposals(w.env.Ctx, domain.SystemActor("qa019-expiry"), 100)); n != want {
			t.Fatalf("expiry sweep = %d, want %d", n, want)
		}
	}

	for _, token := range []string{w.aliceT, w.bobT} {
		for id, status := range map[string]string{declined.ID: "declined", expiring.ID: "expired"} {
			r := w.get(token, "/transactions/"+id)
			wantStatus(t, r, 200)
			if got := decode[txResp](t, r); got.Status != status || got.Seq != nil {
				t.Errorf("transaction %s = %+v, want unposted %s", id, got, status)
			}
		}
		if got := w.acct(token); got != before[token] {
			t.Errorf("account changed: %+v, before %+v", got, before[token])
		}
		r := w.get(token, "/accounts/me/entries")
		wantStatus(t, r, 200)
		if got := decode[page](t, r); len(got.Data) != 0 {
			t.Errorf("decline or expiry posted entries: %+v", got.Data)
		}
		r = w.get(token, "/notifications")
		wantStatus(t, r, 200)
		expiryNotices, declineNotices := 0, 0
		for _, n := range decode[page](t, r).Data {
			switch n["kind"] {
			case service.NotifTradeExpired:
				expiryNotices++
				if n["link"] != "/trades/"+expiring.ID {
					t.Errorf("expiry notice links to %v", n["link"])
				}
			case service.NotifTradeDeclined:
				declineNotices++
			}
		}
		if expiryNotices != 1 {
			t.Errorf("expiry notices = %d, want exactly 1 after repeat sweep", expiryNotices)
		}
		wantDecline := 0
		if token == w.aliceT {
			wantDecline = 1
		}
		if declineNotices != wantDecline {
			t.Errorf("decline notices = %d, want %d", declineNotices, wantDecline)
		}
	}
	if report := testutil.Must(w.env.Svc.VerifyLedger(w.env.Ctx, w.admin, w.net.ID)); !report.OK {
		t.Errorf("ledger verification failed: %v", report.Findings)
	}
}
