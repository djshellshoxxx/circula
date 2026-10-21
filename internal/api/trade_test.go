package api_test

import (
	"testing"
)

type acct struct {
	BalanceMinor, CreditLimitMinor, ReservedMinor, AvailableMinor, Direction string
	LedgerSeq                                                                int64
}

func (w *world) acct(tok string) acct {
	w.t.Helper()
	rec := w.get(tok, "/accounts/me")
	wantStatus(w.t, rec, 200)
	raw := decode[struct {
		Balance   string `json:"balance_minor"`
		Limit     string `json:"credit_limit_minor"`
		Reserved  string `json:"reserved_minor"`
		Available string `json:"available_minor"`
		Direction string `json:"direction"`
		Seq       int64  `json:"ledger_seq"`
	}](w.t, rec)
	return acct{raw.Balance, raw.Limit, raw.Reserved, raw.Available, raw.Direction, raw.Seq}
}

type txResp struct {
	ID, Kind, Status string
	Amount           string  `json:"amount_minor"`
	MyDirection      *string `json:"my_direction"`
	Seq              *int64  `json:"posted_ledger_seq"`
	Account          *struct {
		Balance string `json:"balance_minor"`
	} `json:"account"`
}

func TestFullTradeFlow(t *testing.T) {
	w := newWorld(t)
	if a := w.acct(w.aliceT); a.BalanceMinor != "0" || a.CreditLimitMinor != "10000" || a.Direction != "zero" || a.AvailableMinor != "10000" {
		t.Fatalf("initial account = %+v", a)
	}
	// Alice proposes to pay Bob 25.00.
	key := w.key()
	body := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "2500", "description": "garden help"}
	rec := w.post(w.aliceT, "/transactions", body, key)
	wantStatus(t, rec, 201)
	tx := decode[txResp](t, rec)
	if tx.Status != "proposed" || tx.Amount != "2500" || *tx.MyDirection != "pay" || rec.Header().Get("Location") != "/api/v1/transactions/"+tx.ID {
		t.Fatalf("propose = %+v loc=%s", tx, rec.Header().Get("Location"))
	}
	if a := w.acct(w.aliceT); a.ReservedMinor != "2500" || a.AvailableMinor != "7500" || a.BalanceMinor != "0" {
		t.Fatalf("after propose alice = %+v", a)
	}

	// Retry with the same key replays the original response.
	rec2 := w.post(w.aliceT, "/transactions", body, key)
	wantStatus(t, rec2, 201)
	if rec2.Header().Get("Idempotent-Replayed") != "true" || rec2.Body.String() != rec.Body.String() ||
		rec2.Header().Get("Location") != rec.Header().Get("Location") {
		t.Fatalf("replay differs: %s vs %s", rec2.Body.String(), rec.Body.String())
	}
	if rec.Header().Get("Idempotent-Replayed") != "" {
		t.Fatal("first response must not be marked replayed")
	}
	// Same key, different body -> 422; no second transaction.
	body2 := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "2600"}
	wantProblem(t, w.post(w.aliceT, "/transactions", body2, key), 422, "idempotency-key-reused")
	wantProblem(t, w.post(w.aliceT, "/transactions", body, ""), 400, "idempotency-key-required")
	wantProblem(t, w.post(w.aliceT, "/transactions", body, "bad\tkey"), 400, "idempotency-key-required")
	list := w.get(w.aliceT, "/transactions")
	if n := len(decode[struct{ Data []txResp }](t, list).Data); n != 1 {
		t.Fatalf("transactions after retries = %d", n)
	}

	// Visibility: Bob sees it as `receive`; the proposer may not accept.
	rec = w.get(w.bobT, "/transactions/"+tx.ID)
	wantStatus(t, rec, 200)
	if d := decode[txResp](t, rec); *d.MyDirection != "receive" {
		t.Fatalf("bob direction = %v", *d.MyDirection)
	}
	wantProblem(t, w.post(w.aliceT, "/transactions/"+tx.ID+"/accept", nil, w.key()), 403, "forbidden")
	wantProblem(t, w.post(w.adminT, "/transactions/"+tx.ID+"/accept", nil, w.key()), 404, "not-found") // non-party

	// Bob accepts: both entries post, balance view returned (API-069).
	acceptKey := w.key()
	rec = w.post(w.bobT, "/transactions/"+tx.ID+"/accept", nil, acceptKey)
	wantStatus(t, rec, 200)
	done := decode[txResp](t, rec)
	if done.Status != "completed" || done.Seq == nil || *done.Seq < 1 || done.Account == nil || done.Account.Balance != "2500" {
		t.Fatalf("accept = %s", rec.Body.String())
	}
	if rec3 := w.post(w.bobT, "/transactions/"+tx.ID+"/accept", nil, acceptKey); rec3.Header().Get("Idempotent-Replayed") != "true" ||
		rec3.Body.String() != rec.Body.String() {
		t.Fatalf("accept replay: %s", rec3.Body.String())
	}
	// A new key on a completed trade is an invalid state, with the status in the body (API-028).
	p := wantProblem(t, w.post(w.bobT, "/transactions/"+tx.ID+"/accept", nil, w.key()), 409, "invalid-transaction-state")
	if p.CurrentStatus != "completed" {
		t.Fatalf("current_status = %q", p.CurrentStatus)
	}
	wantProblem(t, w.post(w.aliceT, "/transactions/"+tx.ID+"/cancel", nil, w.key()), 409, "invalid-transaction-state")

	// Balances and directions.
	a, b := w.acct(w.aliceT), w.acct(w.bobT)
	if a.BalanceMinor != "-2500" || a.Direction != "in_debt" || a.ReservedMinor != "0" || a.AvailableMinor != "7500" {
		t.Fatalf("alice = %+v", a)
	}
	if b.BalanceMinor != "2500" || b.Direction != "in_credit" || b.AvailableMinor != "12500" {
		t.Fatalf("bob = %+v", b)
	}

	// Entries (signed, newest first) and the status history.
	rec = w.get(w.aliceT, "/accounts/me/entries")
	wantStatus(t, rec, 200)
	raw := decode[struct {
		Data []map[string]any `json:"data"`
	}](t, rec)
	if len(raw.Data) != 1 || raw.Data[0]["amount_minor"] != "-2500" || raw.Data[0]["transaction_id"] != tx.ID ||
		raw.Data[0]["counterparty_display_name"] != "bob" || raw.Data[0]["ledger_seq"].(float64) < 1 {
		t.Fatalf("entries = %v", raw.Data)
	}
	rec = w.get(w.bobT, "/transactions/"+tx.ID+"/events")
	wantStatus(t, rec, 200)
	ev := decode[struct {
		Data []struct{ From, To string }
	}](t, rec)
	if len(ev.Data) != 2 || ev.Data[0].From != "" && ev.Data[0].From != "proposed" || ev.Data[len(ev.Data)-1].To != "completed" {
		t.Fatalf("events = %+v", ev)
	}
	// Filters.
	for q, want := range map[string]int{"?status=completed": 1, "?status=proposed": 0, "?role=payer": 1, "?role=payee": 0} {
		rec = w.get(w.aliceT, "/transactions"+q)
		wantStatus(t, rec, 200)
		if n := len(decode[struct{ Data []txResp }](t, rec).Data); n != want {
			t.Errorf("alice %s -> %d, want %d", q, n, want)
		}
	}
	wantProblem(t, w.get(w.aliceT, "/transactions?status=bogus"), 400, "invalid-query")
	wantProblem(t, w.get(w.aliceT, "/transactions?role=boss"), 400, "invalid-query")

	// Insufficient credit: nothing is created (API-048).
	big := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "9000"}
	wantProblem(t, w.post(w.aliceT, "/transactions", big, w.key()), 422, "insufficient-available-credit")
	over := m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "999999999999"}
	wantProblem(t, w.post(w.aliceT, "/transactions", over, w.key()), 422, "amount-exceeds-policy")
	self := m{"counterparty_member_id": w.aliceM.ID, "direction": "pay", "amount_minor": "5"}
	wantProblem(t, w.post(w.aliceT, "/transactions", self, w.key()), 422, "validation-failed")

	// Decline and cancel paths: no entries, balances unchanged.
	rq := m{"counterparty_member_id": w.aliceM.ID, "direction": "receive", "amount_minor": "300"}
	rec = w.post(w.bobT, "/transactions", rq, w.key()) // Bob asks Alice to pay him
	wantStatus(t, rec, 201)
	t2 := decode[txResp](t, rec)
	rec = w.post(w.aliceT, "/transactions/"+t2.ID+"/decline", nil, w.key())
	wantStatus(t, rec, 200)
	if decode[txResp](t, rec).Status != "declined" {
		t.Fatal("not declined")
	}
	rec = w.post(w.aliceT, "/transactions", m{"counterparty_member_id": w.bobM.ID, "direction": "pay", "amount_minor": "100"}, w.key())
	t3 := decode[txResp](t, rec)
	rec = w.post(w.aliceT, "/transactions/"+t3.ID+"/cancel", nil, w.key())
	wantStatus(t, rec, 200)
	if decode[txResp](t, rec).Status != "cancelled" {
		t.Fatal("not cancelled")
	}
	if a := w.acct(w.aliceT); a.BalanceMinor != "-2500" || a.ReservedMinor != "0" {
		t.Fatalf("alice after decline/cancel = %+v", a)
	}

	// Admin reversal restores balances; a ledger check passes.
	rec = w.post(w.adminT, "/admin/transactions/"+tx.ID+"/reversal", m{"reason": "agreed undo"}, w.key())
	wantStatus(t, rec, 201)
	if a := w.acct(w.aliceT); a.BalanceMinor != "0" {
		t.Fatalf("alice after reversal = %+v", a)
	}
	rec = w.post(w.adminT, "/admin/consistency-checks", nil, w.key())
	wantStatus(t, rec, 201)
	chk := decode[struct {
		ID string
		OK bool
	}](t, rec)
	if !chk.OK {
		t.Fatalf("ledger check failed: %s", rec.Body.String())
	}
	wantStatus(t, w.get(w.adminT, "/admin/consistency-checks/"+chk.ID), 200)
	wantProblem(t, w.get(w.aliceT, "/admin/consistency-checks/"+chk.ID), 403, "forbidden")
	wantProblem(t, w.get(w.adminT, "/admin/consistency-checks/nope"), 404, "not-found")
}
