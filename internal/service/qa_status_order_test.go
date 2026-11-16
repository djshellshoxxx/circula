package service_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// legalStatusSteps is every (from, to) pair the DOM-010 state machine allows,
// plus creation ("" to proposed).
func legalStatusSteps() map[[2]string]bool {
	ok := map[[2]string]bool{{"", string(domain.TxProposed)}: true}
	for _, from := range domain.AllTxStatuses {
		for _, cmd := range domain.AllTxCommands {
			if to, err := domain.NextTxStatus(from, cmd); err == nil {
				ok[[2]string{string(from), string(to)}] = true
			}
		}
	}
	return ok
}

// statusChains checks L12 ordering for every transaction of the networks in
// `in` (a "(?,?)" list bound to ids): in insertion (rowid) order its status
// events start at "" -> proposed, each starts where the previous ended,
// every step is a legal transition, times never go backwards, and the last
// event ends at the transaction's current status. It returns the first
// violation, or "".
func statusChains(db *sql.DB, in string, ids []any) string {
	legal := legalStatusSteps()
	rows, err := db.Query(`SELECT t.id, t.status, e.from_status, e.to_status, e.occurred_at
		FROM transactions t LEFT JOIN transaction_status_events e ON e.tx_id = t.id AND e.network_id = t.network_id
		WHERE t.network_id IN `+in+` ORDER BY t.rowid, e.rowid`, ids...)
	if err != nil {
		return err.Error()
	}
	defer func() { _ = rows.Close() }()
	type chain struct {
		status, last string
		at           int64
		n            int
	}
	var cur *chain
	var curID string
	end := func() string {
		if cur != nil && (cur.n == 0 || cur.last != cur.status) {
			return fmt.Sprintf("L12: tx %s has %d status events ending at %q, status is %q", curID, cur.n, cur.last, cur.status)
		}
		return ""
	}
	for rows.Next() {
		var id, status string
		var from, to sql.NullString
		var at sql.NullInt64
		if err := rows.Scan(&id, &status, &from, &to, &at); err != nil {
			return err.Error()
		}
		if id != curID {
			if msg := end(); msg != "" {
				return msg
			}
			cur, curID = &chain{status: status}, id
		}
		if !to.Valid {
			continue // no events at all; end reports it
		}
		if from.String != cur.last {
			return fmt.Sprintf("L12: tx %s event %d starts at %q, previous ended at %q", id, cur.n+1, from.String, cur.last)
		}
		if !legal[[2]string{from.String, to.String}] {
			return fmt.Sprintf("L12: tx %s has illegal step %q -> %q", id, from.String, to.String)
		}
		if at.Int64 < cur.at {
			return fmt.Sprintf("L12: tx %s event %d goes back in time", id, cur.n+1)
		}
		cur.last, cur.at = to.String, at.Int64
		cur.n++
	}
	if err := rows.Err(); err != nil {
		return err.Error()
	}
	return end()
}

// L12: with the production id generator (UUIDv7) and the wall clock, a
// trade taken through dispute, withdrawal, partial and full reversal, plus
// the reversal and adjustment transactions it creates, has status histories
// that the read path (GetTransaction) returns in the order the events were
// written, forming legal chains that end at the current status. Events of
// one unit share a millisecond, so the order cannot come from time alone.
func TestStatusEventOrderMatchesWriteOrder(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.IDs, c.Clock = nil, nil })
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "order", func(p *domain.Policy) { p.DisputeWindowDays = 30 })
	a, aM := e.Member(admin, net, "ana@example.org")
	b, bM := e.Member(admin, net, "ben@example.org")
	tx := e.Trade(net.ID, a, b, aM, bM, 1000)
	d := tu.Must(e.Svc.OpenDispute(e.Ctx, a, net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "other", IdempotencyKey: e.Key()}))
	tu.Must(e.Svc.WithdrawDispute(e.Ctx, a, net.ID, d.ID))
	tu.Must(e.Svc.ReverseTransaction(e.Ctx, admin, net.ID, service.ReverseInput{TxID: tx.ID, Amount: 300, Reason: "part", IdempotencyKey: e.Key()}))
	tu.Must(e.Svc.ReverseTransaction(e.Ctx, admin, net.ID, service.ReverseInput{TxID: tx.ID, Amount: 700, Reason: "rest", IdempotencyKey: e.Key()}))
	tu.Must(e.Svc.CreateAdjustment(e.Ctx, admin, net.ID, service.AdjustInput{MemberID: aM.ID, Amount: 5, Direction: service.AdjustCredit,
		Reason: "r", IdempotencyKey: e.Key()}))

	db := e.Store.DB()
	if msg := statusChains(db, "(?)", []any{net.ID}); msg != "" {
		t.Fatal(msg)
	}
	strs := func(q string, args ...any) []string {
		t.Helper()
		r, err := db.Query(q, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = r.Close() }()
		var out []string
		for r.Next() {
			var s string
			if err := r.Scan(&s); err != nil {
				t.Fatal(err)
			}
			out = append(out, s)
		}
		return out
	}
	txIDs := strs(`SELECT id FROM transactions WHERE network_id = ? ORDER BY rowid`, net.ID)
	if len(txIDs) != 4 {
		t.Fatalf("%d transactions, want trade, two reversals and an adjustment", len(txIDs))
	}
	for _, id := range txIDs {
		want := strs(`SELECT from_status || '>' || to_status FROM transaction_status_events WHERE tx_id = ? ORDER BY rowid`, id)
		v := tu.Must(e.Svc.GetTransaction(e.Ctx, admin, net.ID, id))
		var got []string
		for _, h := range v.History {
			got = append(got, string(h.From)+">"+string(h.To))
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("tx %s (%s): history %v, written %v", id, v.Tx.Kind, got, want)
		}
	}
	if v := tu.Must(e.Svc.GetTransaction(e.Ctx, admin, net.ID, tx.ID)); v.Tx.Status != domain.TxReversed || len(v.History) < 4 {
		t.Errorf("trade status %s with %d events", v.Tx.Status, len(v.History))
	}
}
