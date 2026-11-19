package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-019 / OPS-027 / FR-038: exercise HTML forms and pages over HTTP. This
// verifies the server-rendered flow, not a real-browser accessibility run.
func TestQA019DeclineAndExpiryPages(t *testing.T) {
	w := newWorld(t)
	alice, bob := w.browser(t), w.browser(t)
	alice.login("alice@example.org")
	bob.login("bob@example.org")
	base := w.base()
	before := map[string]service.BalanceView{}
	for _, actor := range []domain.Actor{w.alice, w.bob} {
		before[actor.PersonID] = *testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, actor, w.net.ID))
	}
	propose := func(b *browser, counterparty, direction, amount string) string {
		t.Helper()
		form := b.get(base + "/trades/new")
		values := url.Values{"counterparty_id": {counterparty}, "direction": {direction},
			"amount": {amount}, "step": {"review"}}
		review := b.post(base+"/trades/new", form, values)
		b.mustContain(review, "Review your trade", "Send trade")
		values.Set("step", "confirm")
		sent := b.post(base+"/trades/new", review, values)
		if sent.Status != 200 || !strings.HasPrefix(sent.URL, base+"/trades/") || sent.URL == base+"/trades/new" {
			t.Fatalf("proposal failed: %d %s %s", sent.Status, sent.URL, snippet(sent.Body))
		}
		return strings.TrimPrefix(sent.URL, base+"/trades/")
	}
	declined := propose(alice, w.bobM.ID, "pay", "7.00")
	declinePath := base + "/trades/" + declined + "/decline"
	done := bob.post(declinePath, bob.get(declinePath), nil)
	if done.Status != 200 {
		t.Fatalf("decline returned %d", done.Status)
	}
	bob.mustContain(done, "Trade declined. Nothing was posted.", `<span class="tag">Declined</span>`)
	expiring := propose(bob, w.aliceM.ID, "request", "4.00")

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

	for _, viewer := range []struct {
		b     *browser
		actor domain.Actor
	}{{alice, w.alice}, {bob, w.bob}} {
		for id, status := range map[string]string{declined: "Declined", expiring: "Expired"} {
			page := viewer.b.get(base + "/trades/" + id)
			if page.Status != 200 {
				t.Fatalf("trade page returned %d", page.Status)
			}
			viewer.b.mustContain(page, `<span class="tag">`+status+`</span>`, "Nothing was posted.")
			if strings.Contains(page.Body, "Review and accept") || strings.Contains(page.Body, "Cancel this proposal") {
				t.Errorf("terminal trade still offers a pending action: %s", snippet(page.Body))
			}
			view := testutil.Must(w.env.Svc.GetTransaction(w.env.Ctx, viewer.actor, w.net.ID, id))
			if len(view.Entries) != 0 {
				t.Errorf("terminal trade has %d ledger entries", len(view.Entries))
			}
		}
		if got := *testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, viewer.actor, w.net.ID)); got != before[viewer.actor.PersonID] {
			t.Errorf("account changed: %+v, before %+v", got, before[viewer.actor.PersonID])
		}
		inbox := viewer.b.get(base + "/inbox")
		if inbox.Status != 200 {
			t.Fatalf("inbox returned %d", inbox.Status)
		}
		viewer.b.mustContain(inbox, "Nothing is waiting for you", "A trade proposal expired")
		if n := strings.Count(inbox.Body, `>A trade proposal expired</a>`); n != 1 {
			t.Errorf("inbox renders %d expiry notices, want exactly 1", n)
		}
		notices := testutil.Must(w.env.Svc.ListNotifications(w.env.Ctx, viewer.actor, w.net.ID, service.NotificationQuery{}))
		count := 0
		for _, notice := range notices {
			if notice.Kind == service.NotifTradeExpired {
				count++
				if notice.Link != "/trades/"+expiring {
					t.Errorf("expiry notice links to %q", notice.Link)
				}
			}
		}
		if count != 1 {
			t.Errorf("expiry notices = %d after repeat sweep, want exactly 1", count)
		}
	}
	if report := testutil.Must(w.env.Svc.VerifyLedger(w.env.Ctx, w.admin, w.net.ID)); !report.OK {
		t.Errorf("ledger verification failed: %v", report.Findings)
	}
}
