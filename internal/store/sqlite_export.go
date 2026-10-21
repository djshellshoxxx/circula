package store

import "github.com/djshellshoxxx/circula/internal/domain"

// exportQuery returns SQL and args for a table. Admin scope (memberID == "")
// is network-wide; member scope limits to the member's own data and exposes
// counterparties by display name only (FR-090, SEC-017). No query can reach
// password hashes, token hashes, invitation hashes or sessions.
func exportQuery(tbl ExportTable, netID, memberID string) (string, []any, bool) {
	own := memberID != ""
	switch tbl {
	case TblMembers:
		if own {
			return `SELECT id, display_name, status, join_method, joined_at, departed_at FROM members
				WHERE network_id = ? AND id = ? ORDER BY id`, []any{netID, memberID}, true
		}
		return `SELECT m.id, m.display_name, m.status, m.join_method, m.default_flag, m.joined_at, m.departed_at,
			a.limit_override, a.limit_override_until
			FROM members m LEFT JOIN accounts a ON a.member_id = m.id WHERE m.network_id = ? ORDER BY m.id`, []any{netID}, true
	case TblRoles:
		if own {
			return "", nil, false
		}
		return `SELECT id, member_id, role, granted_at, granted_by, revoked_at FROM role_assignments
			WHERE network_id = ? ORDER BY id`, []any{netID}, true
	case TblAccounts:
		if own {
			return `SELECT id, balance FROM accounts WHERE network_id = ? AND member_id = ? ORDER BY id`, []any{netID, memberID}, true
		}
		return `SELECT id, COALESCE(member_id,'') AS member_id, is_system, balance, limit_override, limit_override_until
			FROM accounts WHERE network_id = ? ORDER BY id`, []any{netID}, true
	case TblListings:
		q := `SELECT id, member_id, type, title, description, price, status, created_at, updated_at
			FROM listings WHERE network_id = ?`
		if own {
			return q + ` AND member_id = ? ORDER BY id`, []any{netID, memberID}, true
		}
		return q + ` ORDER BY id`, []any{netID}, true
	case TblTransactions:
		if own {
			return `SELECT t.id, t.kind, t.status,
				CASE WHEN pa.member_id = ?1 THEN 'payer' ELSE 'payee' END AS my_role,
				t.amount, ma.display_name AS payer_name, mb.display_name AS payee_name,
				t.note, t.reason, COALESCE(t.original_tx_id,'') AS original_tx_id, t.proposed_at, t.completed_at,
				COALESCE(t.ledger_seq,0) AS ledger_seq
				FROM transactions t JOIN accounts pa ON pa.id = t.payer_account_id
				JOIN accounts pb ON pb.id = t.payee_account_id
				LEFT JOIN members ma ON ma.id = pa.member_id LEFT JOIN members mb ON mb.id = pb.member_id
				WHERE t.network_id = ?2 AND (pa.member_id = ?1 OR pb.member_id = ?1) ORDER BY t.id`,
				[]any{memberID, netID}, true
		}
		return `SELECT id, kind, status, payer_account_id, payee_account_id, amount, COALESCE(listing_id,'') AS listing_id,
			note, reason, COALESCE(original_tx_id,'') AS original_tx_id, COALESCE(created_by,'') AS created_by,
			COALESCE(accepted_by,'') AS accepted_by, reversed_total, proposed_at, expires_at, completed_at,
			COALESCE(ledger_seq,0) AS ledger_seq FROM transactions WHERE network_id = ? ORDER BY id`, []any{netID}, true
	case TblStatusEvents:
		if own {
			return `SELECT s.id, s.tx_id, s.from_status, s.to_status, s.occurred_at FROM transaction_status_events s
				JOIN transactions t ON t.id = s.tx_id JOIN accounts pa ON pa.id = t.payer_account_id
				JOIN accounts pb ON pb.id = t.payee_account_id
				WHERE s.network_id = ?2 AND (pa.member_id = ?1 OR pb.member_id = ?1) ORDER BY s.id`,
				[]any{memberID, netID}, true
		}
		return `SELECT id, tx_id, from_status, to_status, actor_id, request_id, occurred_at
			FROM transaction_status_events WHERE network_id = ? ORDER BY id`, []any{netID}, true
	case TblEntries:
		if own {
			return `SELECT e.id, e.tx_id, e.amount, e.ledger_seq, e.posted_at FROM entries e
				JOIN accounts a ON a.id = e.account_id WHERE e.network_id = ? AND a.member_id = ? ORDER BY e.ledger_seq, e.id`,
				[]any{netID, memberID}, true
		}
		return `SELECT id, tx_id, account_id, amount, ledger_seq, posted_at FROM entries
			WHERE network_id = ? ORDER BY ledger_seq, id`, []any{netID}, true
	case TblDisputes:
		if own {
			return `SELECT d.id, d.tx_id, d.category, d.status, COALESCE(d.outcome,'') AS outcome, d.reversal_amount, d.opened_at, d.decided_at
				FROM disputes d JOIN transactions t ON t.id = d.tx_id JOIN accounts pa ON pa.id = t.payer_account_id
				JOIN accounts pb ON pb.id = t.payee_account_id
				WHERE d.network_id = ?2 AND (pa.member_id = ?1 OR pb.member_id = ?1) ORDER BY d.id`,
				[]any{memberID, netID}, true
		}
		return `SELECT id, tx_id, opened_by, category, statement, status, COALESCE(outcome,'') AS outcome, reversal_amount,
			COALESCE(reviewer_id,'') AS reviewer_id, rationale, opened_at, decided_at FROM disputes
			WHERE network_id = ? ORDER BY id`, []any{netID}, true
	case TblPolicies:
		if own {
			return "", nil, false
		}
		return `SELECT version, policy_json, changed_by, reason, effective_at FROM network_policies
			WHERE network_id = ? ORDER BY version`, []any{netID}, true
	case TblAudit:
		if own {
			return "", nil, false
		}
		return `SELECT id, seq, occurred_at, actor_type, actor_id, action, target_type, target_id, request_id,
			idem_ref, summary_before, summary_after, reason, prev_hash, hash FROM audit_events
			WHERE network_id = ? ORDER BY seq`, []any{netID}, true
	}
	return "", nil, false
}

func (t *txImpl) StreamTable(tbl ExportTable, netID, memberID string, fn func([]string, []any) error) error {
	q, args, ok := exportQuery(tbl, netID, memberID)
	if !ok {
		return domain.Forbidden("table %s is not exportable in this scope", tbl)
	}
	rows, err := t.query(q, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	if err := fn(cols, nil); err != nil { // header first, even for empty tables
		return err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		out := make([]any, len(vals))
		for i, v := range vals {
			if b, isB := v.([]byte); isB {
				v = string(b)
			}
			out[i] = v
		}
		if err := fn(cols, out); err != nil {
			return err
		}
	}
	return rows.Err()
}
