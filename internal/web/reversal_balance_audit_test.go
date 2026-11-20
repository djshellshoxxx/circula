package web

import (
	"net/url"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// DAT-034: the reversal confirmation displays both parties' balances. Every
// such disclosure must be recorded, including a review that is never posted.
func TestReversalReviewAuditsBothDisclosedBalances(t *testing.T) {
	w := newWorld(t)
	tx := w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 2500)
	b := w.browser(t)
	b.login("admin@example.org")
	path := w.base() + "/admin/reverse"
	form := b.get(path + "?tx=" + tx.ID)
	counts := func() map[string]int {
		t.Helper()
		out := map[string]int{}
		rows, err := w.env.Store.DB().Query(`SELECT target_id,COUNT(*) FROM audit_events WHERE network_id=? AND action=? AND actor_id=? GROUP BY target_id`, w.net.ID, domain.ActAccountViewed, w.adminM.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var id string
			var n int
			if err := rows.Scan(&id, &n); err != nil {
				t.Fatal(err)
			}
			out[id] = n
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := counts()
	vals := url.Values{"tx": {tx.ID}, "amount": {"25.00"}, "reason": {"Duplicate"}, "step": {"review"}}
	review := b.post(path, form, vals)
	b.mustContain(review, "alice@example.org now", "bob@example.org now", "Review reversal")
	after := counts()
	for _, id := range []string{w.aliceM.ID, w.bobM.ID} {
		if after[id] != before[id]+1 {
			t.Fatalf("displayed balance %s was not audited once: before=%v after=%v", id, before, after)
		}
	}
	var entries int
	if err := w.env.Store.DB().QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if entries != 2 {
		t.Fatal("review posted the reversal")
	}
	vals.Set("step", "confirm")
	done := b.post(path, review, vals)
	b.mustContain(done, "Reversal posted")
	confirmed := counts()
	for _, id := range []string{w.aliceM.ID, w.bobM.ID} {
		if confirmed[id] != after[id]+1 {
			t.Fatalf("confirmation balance %s was not audited: before=%v after=%v", id, after, confirmed)
		}
	}
}
