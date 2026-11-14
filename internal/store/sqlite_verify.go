package store

import (
	"fmt"
	"math"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) BalanceMismatches(netID string) ([]BalanceMismatch, error) {
	rows, err := t.query(`SELECT a.id, a.balance, COALESCE(SUM(e.amount),0) FROM accounts a
		LEFT JOIN entries e ON e.account_id = a.id WHERE a.network_id = ?
		GROUP BY a.id HAVING a.balance <> COALESCE(SUM(e.amount),0)`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BalanceMismatch
	for rows.Next() {
		var m BalanceMismatch
		if err := rows.Scan(&m.AccountID, &m.Cached, &m.Summed); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (t *txImpl) TxProblems(netID string) ([]string, error) {
	rows, err := t.query(`SELECT t.id, t.status, COUNT(e.id), COALESCE(SUM(e.amount),0) FROM transactions t
		LEFT JOIN entries e ON e.tx_id = t.id WHERE t.network_id = ? GROUP BY t.id`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id, status string
		var n int
		var sum int64
		if err := rows.Scan(&id, &status, &n, &sum); err != nil {
			return nil, err
		}
		posted := status == "completed" || status == "disputed" || status == "reversed"
		switch {
		case posted && (n != 2 || sum != 0):
			out = append(out, fmt.Sprintf("transaction %s (%s) has %d entries summing to %d", id, status, n, sum))
		case !posted && n != 0:
			out = append(out, fmt.Sprintf("transaction %s (%s) must have no entries but has %d", id, status, n))
		}
	}
	return out, rows.Err()
}

func (t *txImpl) Totals(netID string) (int64, int64, error) {
	b, err := t.sumSaturating("accounts", "balance", netID)
	if err != nil {
		return 0, 0, err
	}
	e, err := t.sumSaturating("entries", "amount", netID)
	return b, e, err
}

// sumSaturating sums an int64 column of one network. It tries SQL SUM first
// and, if that overflows (legacy or corrupt data), sums in Go
// and saturates, so verification reports a mismatch instead of failing (A3-03).
// table and col are package constants, never user input.
func (t *txImpl) sumSaturating(table, col, netID string) (int64, error) {
	var s int64
	err := t.row(`SELECT COALESCE(SUM(`+col+`),0) FROM `+table+` WHERE network_id = ?`, netID).Scan(&s)
	if err == nil {
		return s, nil
	}
	if !strings.Contains(err.Error(), "integer overflow") {
		return 0, err
	}
	rows, err := t.query(`SELECT `+col+` FROM `+table+` WHERE network_id = ?`, netID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	var total int64
	sat := false
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return 0, err
		}
		if sat {
			continue
		}
		next, ok := domain.AddChecked(total, v)
		if !ok { // saturate at the end that overflowed; a mismatch is then reported
			sat = true
			if v > 0 {
				total = math.MaxInt64
			} else {
				total = math.MinInt64
			}
			continue
		}
		total = next
	}
	return total, rows.Err()
}

func (t *txImpl) ReversalMismatches(netID string) ([]string, error) {
	rows, err := t.query(`SELECT o.id, o.reversed_total,
		COALESCE((SELECT SUM(r.amount) FROM transactions r WHERE r.original_tx_id = o.id
		          AND r.kind = 'reversal' AND r.status = 'completed'), 0)
		FROM transactions o WHERE o.network_id = ? AND o.kind <> 'reversal'
		AND o.reversed_total <> COALESCE((SELECT SUM(r.amount) FROM transactions r WHERE r.original_tx_id = o.id
		          AND r.kind = 'reversal' AND r.status = 'completed'), 0)`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		var rt, sum int64
		if err := rows.Scan(&id, &rt, &sum); err != nil {
			return nil, err
		}
		out = append(out, fmt.Sprintf("transaction %s reversed_total %d but reversals sum to %d", id, rt, sum))
	}
	return out, rows.Err()
}
