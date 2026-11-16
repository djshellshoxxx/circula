package store

import "github.com/djshellshoxxx/circula/internal/domain"

func (t *txImpl) PutSuspension(netID string, s domain.Suspension) (bool, error) {
	res, err := t.exec(`INSERT OR IGNORE INTO network_suspensions(network_id,kind,category,reason,suspended_by,suspended_at)
		VALUES (?,?,?,?,?,?)`, netID, string(s.Kind), s.Category, s.Reason, s.By, ms(s.At))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) DeleteSuspension(netID string, kind domain.SuspensionKind) (bool, error) {
	res, err := t.exec(`DELETE FROM network_suspensions WHERE network_id = ? AND kind = ?`, netID, string(kind))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) ListSuspensions(netID string) ([]domain.Suspension, error) {
	rows, err := t.query(`SELECT kind,category,reason,suspended_by,suspended_at FROM network_suspensions
		WHERE network_id = ? ORDER BY kind`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Suspension
	for rows.Next() {
		var s domain.Suspension
		var kind string
		var at int64
		if err := rows.Scan(&kind, &s.Category, &s.Reason, &s.By, &at); err != nil {
			return nil, err
		}
		s.Kind, s.At = domain.SuspensionKind(kind), fromMS(at)
		out = append(out, s)
	}
	return out, rows.Err()
}
