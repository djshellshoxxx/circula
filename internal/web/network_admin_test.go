package web

import (
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// GOV-023, FR-068: the administrator pauses and resumes trading and closes the
// network from the web; members see the status banner and clear refusals.
func TestNetworkSuspendAndCloseWeb(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	adm := w.browser(t)
	adm.login("admin@example.org")
	al := w.browser(t)
	al.login("alice@example.org")

	f := adm.get(b + "/admin/network")
	checkStructure(t, "network page", f)
	adm.mustContain(f, "Pause trading", "Close the network", "Download a full export first")
	bad := adm.post(b+"/admin/network/suspend", f, url.Values{"reason": {""}})
	if bad.Status != 422 {
		t.Fatalf("empty reason: %d", bad.Status)
	}
	ok := adm.post(b+"/admin/network/suspend", f, url.Values{"reason": {"security review"}})
	adm.mustContain(ok, "Trading is paused", "security review", "Resume trading")
	al.mustContain(al.get(b), "Trading in this network is paused")
	if r := al.get(b + "/admin/network"); r.Status != 403 {
		t.Errorf("member reached the admin page: %d", r.Status)
	}
	adm.mustContain(adm.post(b+"/admin/network/resume", adm.get(b+"/admin/network"), url.Values{"reason": {"done"}}), "Your suspension is lifted")
	al.mustNotContain(al.get(b), "Trading in this network is paused")

	// closure: blocked by an open proposal, then confirmed by typing the slug
	pend := tu.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{CounterpartyID: w.bobM.ID,
		Direction: service.ProposerPays, Amount: 100, IdempotencyKey: w.env.Key()}))
	adm.mustContain(adm.get(b+"/admin/network"), "Closing is blocked", pend.ID)
	tu.Must(w.env.Svc.CancelTransaction(w.env.Ctx, w.alice, w.net.ID, service.CancelInput{TxID: pend.ID, IdempotencyKey: w.env.Key()}))
	f = adm.get(b + "/admin/network")
	wrong := adm.post(b+"/admin/network/close", f, url.Values{"reason": {"end"}, "confirm": {"nope"}})
	if wrong.Status != 422 {
		t.Fatalf("wrong confirmation: %d", wrong.Status)
	}
	adm.mustContain(adm.post(b+"/admin/network/close", f, url.Values{"reason": {"end"}, "confirm": {w.net.Slug}}), "The network is closed")
	al.mustContain(al.get(b), "This network is closed")
}
