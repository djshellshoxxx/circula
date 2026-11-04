package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// UX-017: a reversal is its own line, linked to the original; the original keeps its own
// line and gains only a "Reversed by …" link to each reversal, on History and trade detail.
func TestReversalLinksHistoryAndDetail(t *testing.T) {
	w := newWorld(t)
	b := w.base()
	ctx := w.env.Ctx
	orig := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 300)
	plain := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 50)
	rev1 := testutil.Must(w.env.Svc.ReverseTransaction(ctx, w.admin, w.net.ID, service.ReverseInput{TxID: orig.ID, Amount: 100,
		Reason: "entered twice", IdempotencyKey: w.env.Key()}))
	rev2 := testutil.Must(w.env.Svc.ReverseTransaction(ctx, w.admin, w.net.ID, service.ReverseInput{TxID: orig.ID,
		Reason: "rest of it", IdempotencyKey: w.env.Key()}))
	adj := testutil.Must(w.env.Svc.CreateAdjustment(ctx, w.admin, w.net.ID, service.AdjustInput{MemberID: w.aliceM.ID, Amount: 25,
		Direction: service.AdjustCredit, Reason: "welcome correction", IdempotencyKey: w.env.Key()}))

	origLink := fmt.Sprintf(`Reverses <a href="%s/trades/%s">the original trade</a>`, b, orig.ID)
	by1 := fmt.Sprintf(`<a href="%s/trades/%s">Reversed by ledger reference %d (1.00 `, b, rev1.ID, rev1.LedgerSeq)
	by2 := fmt.Sprintf(`<a href="%s/trades/%s">Reversed by ledger reference %d (2.00 `, b, rev2.ID, rev2.LedgerSeq)

	for _, who := range []string{"alice@example.org", "bob@example.org"} {
		br := w.browser(t)
		br.login(who)
		h := br.get(b + "/history")
		br.mustContain(h, origLink, by1, by2, "Reversal: ")
		if strings.Count(h.Body, origLink) != 2 {
			t.Errorf("%s: want both reversal lines linked to the original, got %d", who, strings.Count(h.Body, origLink))
		}
		// Each reversal keeps its own line, date and signed effect; the original line is still there.
		for _, id := range []string{orig.ID, rev1.ID, rev2.ID, plain.ID} {
			br.mustContain(h, fmt.Sprintf(`<p><a href="%s/trades/%s">`, b, id))
		}
		if strings.Index(h.Body, by1) > strings.Index(h.Body, by2) {
			t.Errorf("%s: reversals are not listed oldest first", who)
		}
		// A trade that was never reversed carries no link.
		if li := rowOf(h.Body, b+"/trades/"+plain.ID); strings.Contains(li, "Reversed by") || strings.Contains(li, "Reverses") {
			t.Errorf("%s: unreversed row has a reversal link: %s", who, li)
		}
		// Each reversal line carries its own reason; the original line stays as it was.
		for id, reason := range map[string]string{rev1.ID: "Reason: entered twice", rev2.ID: "Reason: rest of it"} {
			if li := rowOf(h.Body, b+"/trades/"+id); !strings.Contains(li, reason) || !strings.Contains(li, origLink) {
				t.Errorf("%s: reversal row lacks %q or its link: %s", who, reason, li)
			}
		}
		if li := rowOf(h.Body, b+"/trades/"+orig.ID); strings.Contains(li, "Reason:") || strings.Contains(li, "Reversal: ") ||
			!strings.Contains(li, by1) || !strings.Contains(li, by2) {
			t.Errorf("%s: original row changed beyond its Reversed by links: %s", who, li)
		}

		d := br.get(b + "/trades/" + orig.ID)
		br.mustContain(d, "<dt>Reversals</dt>", by1, by2)
		if strings.Contains(d.Body, "The original trade") {
			t.Errorf("%s: original links to an original", who)
		}
		r := br.get(b + "/trades/" + rev1.ID)
		br.mustContain(r, "<dt>Reverses</dt>", fmt.Sprintf(`<a href="%s/trades/%s">The original trade</a>`, b, orig.ID))
		if strings.Contains(r.Body, "Reversed by") {
			t.Errorf("%s: a reversal shows a Reversed by link", who)
		}
		if p := br.get(b + "/trades/" + plain.ID); strings.Contains(p.Body, "Reversed by") || strings.Contains(p.Body, "The original trade") {
			t.Errorf("%s: unreversed trade detail has reversal links", who)
		}
	}

	// An adjustment is its own line with its date, reason and signed effect. It corrects a
	// balance, not a transaction, so it has no original to link to (0001_init: only a
	// reversal carries original_tx_id).
	al := w.browser(t)
	al.login("alice@example.org")
	li := rowOf(al.get(b+"/history").Body, b+"/trades/"+adj.ID)
	for _, need := range []string{"Adjustment: ", `<span class="num">&#43;0.25</span>`, "Reason: welcome correction", "<time datetime="} {
		if !strings.Contains(li, need) {
			t.Errorf("adjustment row lacks %q: %s", need, li)
		}
	}
	if strings.Contains(li, "Reverses") || strings.Contains(li, "Reversed by") {
		t.Errorf("adjustment row links to an original: %s", li)
	}

	// An administrator who is not a party still reaches the original from the reversal.
	adm := w.browser(t)
	adm.login("admin@example.org")
	adm.mustContain(adm.get(b+"/trades/"+rev2.ID), fmt.Sprintf(`<a href="%s/trades/%s">The original trade</a>`, b, orig.ID))
}

// rowOf returns the History list item whose first link is href.
func rowOf(body, href string) string {
	i := strings.Index(body, `<p><a href="`+href+`">`)
	if i < 0 {
		return ""
	}
	end := strings.Index(body[i:], "</li>")
	if end < 0 {
		return body[i:]
	}
	return body[i : i+end]
}
