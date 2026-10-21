package store

import "fmt"

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
	var b, e int64
	if err := t.row(`SELECT COALESCE(SUM(balance),0) FROM accounts WHERE network_id = ?`, netID).Scan(&b); err != nil {
		return 0, 0, err
	}
	err := t.row(`SELECT COALESCE(SUM(amount),0) FROM entries WHERE network_id = ?`, netID).Scan(&e)
	return b, e, err
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
