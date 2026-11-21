package web

import (
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// UX-022, DOM-011, DOM-012: the acceptance screen must reflect whether the
// amount is already reserved. Requests have no reservation before consent.
func TestTradeAcceptanceAvailablePreviewMatchesPosting(t *testing.T) {
	for _, direction := range []service.Direction{service.ProposerPays, service.ProposerIsPaid} {
		t.Run(string(direction), func(t *testing.T) {
			w := newWorld(t)
			// Keep an unrelated outgoing reservation on the accepting member's
			// account so the preview cannot substitute balance plus limit.
			testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.alice, w.net.ID, service.ProposeInput{
				CounterpartyID: w.adminM.ID, Direction: service.ProposerPays, Amount: 200, IdempotencyKey: w.env.Key()}))
			tx := testutil.Must(w.env.Svc.ProposeTransaction(w.env.Ctx, w.bob, w.net.ID, service.ProposeInput{
				CounterpartyID: w.aliceM.ID, Direction: direction, Amount: 500, IdempotencyKey: w.env.Key()}))
			before := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID))
			delta := int64(500)
			if direction == service.ProposerIsPaid {
				delta = -delta
			}
			br := w.browser(t)
			br.login("alice@example.org")
			path := w.base() + "/trades/" + tx.ID + "/accept"
			preview := br.get(path)
			if preview.Status != 200 {
				t.Fatalf("acceptance preview status = %d", preview.Status)
			}
			br.mustContain(preview,
				"<dt>Available to spend now</dt><dd>"+domain.FormatAmount(before.Available, before.Decimals)+"</dd>",
				"<dt>Available to spend after posting</dt><dd>"+domain.FormatAmount(before.Available+delta, before.Decimals)+"</dd>")
			unchanged := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID))
			if unchanged.Balance != before.Balance || unchanged.Available != before.Available {
				t.Fatal("review changed the account")
			}
			posted := br.post(path, preview, url.Values{})
			if posted.Status != 200 {
				t.Fatalf("acceptance status = %d", posted.Status)
			}
			after := testutil.Must(w.env.Svc.GetBalance(w.env.Ctx, w.alice, w.net.ID))
			if after.Balance != before.Balance+delta || after.Available != before.Available+delta || after.ReservedOutgoing != 200 {
				t.Fatalf("posted result differs from preview: before=%+v after=%+v", before, after)
			}
		})
	}
}
