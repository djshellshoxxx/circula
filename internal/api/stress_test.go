package api_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// call is goroutine-safe (the shared world helpers use a counter and t.Helper).
func (w *world) call(method, path, token, key string, body string) *httptest.ResponseRecorder {
	hr := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	if body != "" {
		hr.Header.Set("Content-Type", "application/json")
	}
	hr.Header.Set("Authorization", "Bearer "+token)
	if key != "" {
		hr.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	w.h.ServeHTTP(rec, hr)
	return rec
}

// QA-011: under in-process contention the API never answers 500 or 503; racing
// accepts of one proposal give exactly one 200; same-key retries replay one result.
func TestStressAPIParallelTradesNo5xx(t *testing.T) {
	w := newWorld(t)
	var fiveXX atomic.Int32
	check := func(what string, rec *httptest.ResponseRecorder) {
		if rec.Code >= 500 {
			fiveXX.Add(1)
			t.Errorf("%s: status %d body %s", what, rec.Code, rec.Body.String())
		}
	}

	// A: racing double accept
	rec := w.call("POST", "/transactions", w.aliceT, "st-p0", fmt.Sprintf(`{"counterparty_member_id":%q,"direction":"pay","amount_minor":"1000"}`, w.bobM.ID))
	if rec.Code != 201 {
		t.Fatalf("propose: %d %s", rec.Code, rec.Body.String())
	}
	tx := decode[txResp](t, rec)
	var wins, conflicts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := w.call("POST", "/transactions/"+tx.ID+"/accept", w.bobT, fmt.Sprintf("st-a-%d", i), "")
			check("accept", r)
			switch r.Code {
			case 200:
				wins.Add(1)
			case 409, 422:
				conflicts.Add(1)
			default:
				t.Errorf("accept: unexpected %d %s", r.Code, r.Body.String())
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d accepts returned 200, want 1 (conflicts %d)", wins.Load(), conflicts.Load())
	}

	// B: same key retried in parallel replays one transaction
	body := fmt.Sprintf(`{"counterparty_member_id":%q,"direction":"pay","amount_minor":"100"}`, w.bobM.ID)
	ids := make(chan string, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := w.call("POST", "/transactions", w.aliceT, "st-same", body)
			check("same-key propose", r)
			if r.Code == 201 {
				ids <- decode[txResp](t, r).ID
			} else if r.Code != 409 && r.Code != 425 {
				t.Errorf("same-key propose: %d %s", r.Code, r.Body.String())
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != 1 {
		t.Errorf("%d distinct transactions from one idempotency key", len(seen))
	}

	// C: many disjoint propose+accept pairs through the HTTP stack
	const n = 30
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := fmt.Sprintf(`{"counterparty_member_id":%q,"direction":"pay","amount_minor":"%d"}`, w.bobM.ID, 10+i)
			r := w.call("POST", "/transactions", w.aliceT, fmt.Sprintf("st-c-%d", i), b)
			check("propose", r)
			if r.Code != 201 {
				return
			}
			id := decode[txResp](t, r).ID
			r = w.call("POST", "/transactions/"+id+"/accept", w.bobT, fmt.Sprintf("st-ca-%d", i), "")
			check("accept", r)
		}()
	}
	wg.Wait()
	if fiveXX.Load() != 0 {
		t.Fatalf("%d server errors under contention", fiveXX.Load())
	}
	a, b := w.acct(w.aliceT), w.acct(w.bobT)
	if a.BalanceMinor == "" || a.BalanceMinor[0] != '-' || b.BalanceMinor != a.BalanceMinor[1:] {
		t.Fatalf("balances not equal and opposite: alice %s bob %s", a.BalanceMinor, b.BalanceMinor)
	}
	var sum int64
	if err := w.env.Store.DB().QueryRow(`SELECT COALESCE(SUM(balance),0) FROM accounts`).Scan(&sum); err != nil || sum != 0 {
		t.Fatalf("ledger sum %d (%v)", sum, err)
	}
}
