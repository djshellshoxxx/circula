package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const pendCols = `id,network_id,proposer_member_id,target_member_id,amount,direction,reason,status,created_at,expires_at,
	COALESCE(decided_by,''),decided_at,COALESCE(tx_id,'')`

func scanPending(r rowScanner) (*domain.PendingAdjustment, error) {
	var p domain.PendingAdjustment
	var created, exp int64
	var dec sql.NullInt64
	var st string
	if err := r.Scan(&p.ID, &p.NetworkID, &p.Proposer, &p.Target, &p.Amount, &p.Direction, &p.Reason, &st,
		&created, &exp, &p.DecidedBy, &dec, &p.TxID); err != nil {
		return nil, err
	}
	p.Status = domain.PendingStatus(st)
	p.CreatedAt, p.ExpiresAt, p.DecidedAt = fromMS(created), fromMS(exp), ptrTime(dec)
	return &p, nil
}

func (t *txImpl) InsertPendingAdjustment(p *domain.PendingAdjustment) error {
	_, err := t.exec(`INSERT INTO pending_adjustments(id,network_id,proposer_member_id,target_member_id,amount,direction,reason,status,created_at,expires_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, p.ID, p.NetworkID, p.Proposer, p.Target, p.Amount, p.Direction, p.Reason,
		string(domain.PendingOpen), ms(p.CreatedAt), ms(p.ExpiresAt))
	return err
}

func (t *txImpl) GetPendingAdjustment(netID, id string) (*domain.PendingAdjustment, error) {
	p, err := scanPending(t.row(`SELECT `+pendCols+` FROM pending_adjustments WHERE network_id = ? AND id = ?`, netID, id))
	return p, notFound(err, "adjustment request")
}

func (t *txImpl) ListPendingAdjustments(netID string, statuses []domain.PendingStatus, afterID string, limit int) ([]*domain.PendingAdjustment, error) {
	q := `SELECT ` + pendCols + ` FROM pending_adjustments WHERE network_id = ?`
	args := []any{netID}
	if len(statuses) > 0 {
		in, a := inClause(statuses)
		q += ` AND status IN ` + in
		args = append(args, a...)
	}
	if afterID != "" {
		q += ` AND id < ?`
		args = append(args, afterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limitOf(limit))
	return t.listPending(q, args)
}

func (t *txImpl) listPending(q string, args []any) ([]*domain.PendingAdjustment, error) {
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.PendingAdjustment
	for rows.Next() {
		p, err := scanPending(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (t *txImpl) DecidePendingAdjustment(netID, id string, to domain.PendingStatus, by string, at time.Time, txID string) (bool, error) {
	var byV, txV any
	if by != "" {
		byV = by
	}
	if txID != "" {
		txV = txID
	}
	res, err := t.exec(`UPDATE pending_adjustments SET status = ?, decided_by = ?, decided_at = ?, tx_id = ?
		WHERE network_id = ? AND id = ? AND status = 'pending'`, string(to), byV, ms(at), txV, netID, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (t *txImpl) ListExpiredPendingAdjustments(now time.Time, limit int) ([]*domain.PendingAdjustment, error) {
	return t.listPending(`SELECT `+pendCols+` FROM pending_adjustments WHERE status = 'pending' AND expires_at <= ?
		ORDER BY expires_at, id LIMIT ?`, []any{ms(now), limit})
}
