package web

import (
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-025 / WEB-02: a party can report from the transaction page, and an
// independent administrator can follow every page of that transaction's
// related disputes without showing report content in the dispute list.
func TestTransactionReportHTMLJourneyAndDisputePages(t *testing.T) {
	w := newWorld(t)
	tx := w.propose(w.alice, w.bobM, 100)
	alice, admin := w.browser(t), w.browser(t)
	alice.login("alice@example.org")
	admin.login("admin@example.org")
	reportPath := w.base() + "/report?type=transaction&id=" + tx.ID
	trade := alice.get(w.base() + "/trades/" + tx.ID)
	alice.mustContain(trade, reportPath, "Report a problem with this transaction")
	adminTrade := admin.get(w.base() + "/trades/" + tx.ID)
	if strings.Contains(adminTrade.Body, "Report a problem with this transaction") {
		t.Fatal("nonparty administrator received a transaction report action")
	}
	form := alice.get(reportPath)
	alice.mustContain(form, "Transaction: "+tx.ID)
	const privateNote = "Private transaction report form fixture"
	sent := alice.post(w.base()+"/report", form, url.Values{
		"type": {"transaction"}, "id": {tx.ID}, "category": {"other"}, "note": {privateNote},
	})
	alice.mustContain(sent, "Report sent")
	related := w.base() + "/disputes?transaction_id=" + tx.ID
	queue := admin.get(w.base() + "/admin/reports")
	admin.mustContain(queue, related, w.base()+"/trades/"+tx.ID, "Related disputes on this transaction", privateNote)
	tu.Must(w.env.Svc.AcceptTransaction(w.env.Ctx, w.bob, w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: w.env.Key()}))
	var ids []string
	for i := 0; i <= perPage; i++ {
		d := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
			TxID: tx.ID, Category: "other", Statement: "Ordinary historical case", IdempotencyKey: w.env.Key(),
		}))
		tu.Must(w.env.Svc.WithdrawDispute(w.env.Ctx, w.alice, w.net.ID, d.ID))
		ids = append(ids, d.ID)
	}
	otherTx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 200)
	other := tu.Must(w.env.Svc.OpenDispute(w.env.Ctx, w.alice, w.net.ID, service.OpenDisputeInput{
		TxID: otherTx.ID, Category: "other", Statement: "Unrelated case", IdempotencyKey: w.env.Key(),
	}))
	first := admin.get(related)
	admin.mustContain(first, "Disputes for transaction "+tx.ID, related+"&after="+ids[1], ids[len(ids)-1])
	for _, absent := range []string{ids[0], other.ID, privateNote} {
		if strings.Contains(first.Body, absent) {
			t.Fatalf("related dispute page contained another page, transaction or report note: %q", absent)
		}
	}
	last := admin.get(related + "&after=" + ids[1])
	admin.mustContain(last, ids[0], "Disputes for transaction "+tx.ID)
	if strings.Contains(last.Body, other.ID) || strings.Contains(last.Body, related+"&amp;after=") || strings.Contains(last.Body, privateNote) {
		t.Fatal("last related-dispute page lost its transaction scope or exposed report content")
	}
}
