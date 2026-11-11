package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const accountCols = `id, network_id, COALESCE(member_id,''), is_system, balance, limit_override, limit_override_until`

func (t *txImpl) InsertAccount(a *domain.Account) error {
	_, err := t.exec(`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES (?,?,?,?,0)`,
		a.ID, a.NetworkID, nullStr(a.MemberID), b2i(a.IsSystem))
	return err
}

func scanAccount(r *sql.Row) (*domain.Account, error) {
	var a domain.Account
	var sys int
	var lim, until sql.NullInt64
	if err := r.Scan(&a.ID, &a.NetworkID, &a.MemberID, &sys, &a.Balance, &lim, &until); err != nil {
		return nil, notFound(err, "account")
	}
	a.IsSystem, a.LimitOverride, a.LimitOverrideExpires = sys == 1, ptrInt(lim), ptrTime(until)
	return &a, nil
}

func (t *txImpl) GetAccount(netID, id string) (*domain.Account, error) {
	return scanAccount(t.row(`SELECT `+accountCols+` FROM accounts WHERE network_id = ? AND id = ?`, netID, id))
}

func (t *txImpl) GetAccountByMember(netID, memberID string) (*domain.Account, error) {
	return scanAccount(t.row(`SELECT `+accountCols+` FROM accounts WHERE network_id = ? AND member_id = ?`, netID, memberID))
}

func (t *txImpl) GetSystemAccount(netID string) (*domain.Account, error) {
	return scanAccount(t.row(`SELECT `+accountCols+` FROM accounts WHERE network_id = ? AND is_system = 1`, netID))
}

func (t *txImpl) AddBalance(netID, id string, delta int64) error {
	res, err := t.exec(`UPDATE accounts SET balance = balance + ? WHERE network_id = ? AND id = ?`, delta, netID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return domain.NotFound("account")
	}
	return nil
}

func (t *txImpl) SetLimitOverride(netID, id string, limit *int64, until *time.Time) error {
	var l any
	if limit != nil {
		l = *limit
	}
	_, err := t.exec(`UPDATE accounts SET limit_override = ?, limit_override_until = ? WHERE network_id = ? AND id = ?`,
		l, msPtr(until), netID, id)
	return err
}

func (t *txImpl) sumProposed(col, netID, accountID string, now time.Time) (int64, error) {
	var v int64
	err := t.row(`SELECT COALESCE(SUM(amount),0) FROM transactions
		WHERE network_id = ? AND `+col+` = ? AND status = 'proposed' AND kind = 'trade' AND expires_at > ?`,
		netID, accountID, ms(now)).Scan(&v)
	return v, err
}

// ReservedOutgoing sums the payer's own unexpired proposals. Proposals made by
// the other party ("requests") are not consented to by the payer yet and
// reserve nothing (DOM-011, DOM-012, SA-08).
func (t *txImpl) ReservedOutgoing(netID, accountID string, now time.Time) (int64, error) {
	var v int64
	err := t.row(`SELECT COALESCE(SUM(t.amount),0) FROM transactions t
		JOIN accounts pa ON pa.id = t.payer_account_id
		WHERE t.network_id = ? AND t.payer_account_id = ? AND t.status = 'proposed' AND t.kind = 'trade'
		AND t.expires_at > ? AND t.created_by = pa.member_id`, netID, accountID, ms(now)).Scan(&v)
	return v, err
}

func (t *txImpl) PendingIncoming(netID, accountID string, now time.Time) (int64, error) {
	return t.sumProposed("payee_account_id", netID, accountID, now)
}
