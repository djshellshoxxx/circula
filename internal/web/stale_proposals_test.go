package web

import (
	"net/url"
	"regexp"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-067: follow the attention link through every page to the underlying
// trades; a regular member cannot read the network-wide queue.
func TestAdminStaleProposalQueue(t *testing.T) {
	w := newWorld(t)
	want := map[string]bool{}
	for i := 0; i < 51; i++ {
		tx := tu.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{
			CounterpartyID: w.bobM.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.env.Key(),
		}))
		want[tx.ID] = true
	}
	w.env.Clock.Advance(service.OverviewStaleAfter)
	adm := w.browser(t)
	adm.login("admin@example.org")
	path := w.base() + "/admin/proposals/stale"
	adm.mustContain(adm.get(w.base()+"/admin"), `href="`+path+`"`, "51 open proposals have waited at least 48 hours")
	p := adm.get(path)
	if p.Status != 200 {
		t.Fatalf("queue status=%d", p.Status)
	}
	adm.mustContain(p, "Open proposals waiting at least 48 hours", "Expired proposals are excluded", "Next page")
	adm.mustContain(p, "alice@example.org proposes to pay bob@example.org 1.00 credit", "Waiting for an answer")
	adm.mustNotContain(p, "You will receive", "+1.00", "Waiting for you to respond")
	tradeLinks := regexp.MustCompile(`/trades/([0-9a-f-]{36})`)
	seen := map[string]bool{}
	collect := func(p *pg, count int) {
		t.Helper()
		matches := tradeLinks.FindAllStringSubmatch(p.Body, -1)
		if len(matches) != count {
			t.Fatalf("trade links=%d, want %d", len(matches), count)
		}
		for _, m := range matches {
			if !want[m[1]] || seen[m[1]] {
				t.Fatalf("unexpected or repeated trade %s", m[1])
			}
			seen[m[1]] = true
		}
	}
	collect(p, 50)
	last := tradeLinks.FindAllStringSubmatch(p.Body, -1)[49][1]
	p = adm.get(path + "?after=" + url.QueryEscape(last))
	collect(p, 1)
	adm.mustNotContain(p, "Next page")
	if detail := adm.get(w.base() + "/trades/" + last); detail.Status != 200 {
		t.Fatalf("administrator cannot follow proposal detail: %d", detail.Status)
	}
	if bad := adm.get(path + "?after=invalid"); bad.Status != 422 {
		t.Fatalf("invalid cursor status=%d", bad.Status)
	}
	member := w.browser(t)
	member.login("alice@example.org")
	if denied := member.get(path); denied.Status != 403 {
		t.Fatalf("member queue status=%d", denied.Status)
	}
	// At expiry, the same stored proposed rows disappear without a sweep.
	w.env.Clock.Advance(service.OverviewStaleAfter * 10)
	adm.login("admin@example.org")
	p = adm.get(path)
	adm.mustContain(p, "No proposals are waiting this long", "Return to the administrator overview")
	adm.mustNotContain(p, "Next page")
}
