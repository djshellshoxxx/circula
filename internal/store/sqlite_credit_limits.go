package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
)

func (t *txImpl) InsertPostingCreditCheck(c repository.PostingCreditCheck) error {
	_, err := t.exec(`INSERT INTO posting_credit_checks(entry_id,network_id,legacy_unverified,checked_at,
		policy_version,balance_before,is_system,limit_override,limit_override_until,effective_limit)
		VALUES (?,?,0,?,?,?,?,?,?,?)`, c.EntryID, c.NetworkID, ms(c.CheckedAt), c.PolicyVersion,
		c.BalanceBefore, b2i(c.IsSystem), c.LimitOverride, msPtr(c.LimitOverrideUntil), c.EffectiveLimit)
	return err
}

func (t *txImpl) CreditLimitHistory(netID string, visit func(repository.CreditLimitHistoryRow) error) error {
	rows, err := t.query(`SELECT e.id,e.network_id,e.tx_id,e.account_id,e.amount,e.ledger_seq,e.posted_at,
		a.is_system,w.entry_id,w.network_id,w.legacy_unverified,w.checked_at,w.policy_version,
		w.balance_before,w.is_system,w.limit_override,w.limit_override_until,w.effective_limit,p.policy_json
		FROM entries e LEFT JOIN accounts a ON a.id=e.account_id AND a.network_id=e.network_id
		LEFT JOIN posting_credit_checks w ON w.entry_id=e.id
		LEFT JOIN network_policies p ON p.network_id=w.network_id AND p.version=w.policy_version
		WHERE e.network_id=? ORDER BY e.account_id,e.ledger_seq,e.id`, netID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var r repository.CreditLimitHistoryRow
		var posted int64
		var accountSystem, legacy, checked, version, before, system, override, until, effective sql.NullInt64
		var entryID, networkID, policyJSON sql.NullString
		if err := rows.Scan(&r.Entry.ID, &r.Entry.NetworkID, &r.Entry.TxID, &r.Entry.AccountID,
			&r.Entry.Amount, &r.Entry.LedgerSeq, &posted, &accountSystem, &entryID, &networkID,
			&legacy, &checked, &version, &before, &system, &override, &until, &effective, &policyJSON); err != nil {
			return err
		}
		r.Entry.PostedAt = fromMS(posted)
		if !accountSystem.Valid || accountSystem.Int64 != 0 && accountSystem.Int64 != 1 {
			r.Problem = "entry account has a missing or invalid system marker"
		} else {
			r.AccountIsSystem = accountSystem.Int64 == 1
		}
		if entryID.Valid {
			if !networkID.Valid || networkID.String != r.Entry.NetworkID {
				r.Problem = "credit evidence belongs to a different network"
			}
			if legacy.Valid && legacy.Int64 == 1 {
				r.LegacyUnverified = true
				if checked.Valid || version.Valid || before.Valid || system.Valid || override.Valid || until.Valid || effective.Valid {
					r.Problem = "legacy credit marker contains invented decision inputs"
				}
			} else if !legacy.Valid || legacy.Int64 != 0 || !checked.Valid || !version.Valid || version.Int64 <= 0 ||
				int64(int(version.Int64)) != version.Int64 || !before.Valid || !system.Valid ||
				system.Int64 != 0 && system.Int64 != 1 || !effective.Valid || effective.Int64 < 0 ||
				override.Valid && override.Int64 < 0 || until.Valid && !override.Valid {
				r.Problem = "credit witness has missing or invalid decision inputs"
			} else {
				r.Check = &repository.PostingCreditCheck{EntryID: entryID.String, NetworkID: networkID.String,
					CheckedAt: fromMS(checked.Int64), PolicyVersion: int(version.Int64), BalanceBefore: before.Int64,
					IsSystem: system.Int64 == 1, EffectiveLimit: effective.Int64, LimitOverrideUntil: ptrTime(until)}
				if override.Valid {
					value := override.Int64
					r.Check.LimitOverride = &value
				}
				if policyJSON.Valid {
					policy := domain.DefaultPolicy()
					if err := json.Unmarshal([]byte(policyJSON.String), &policy); err != nil {
						return fmt.Errorf("credit witness policy for entry %s: %w", r.Entry.ID, err)
					}
					r.Policy = &policy
				}
			}
		}
		if err := visit(r); err != nil {
			return err
		}
	}
	return rows.Err()
}
