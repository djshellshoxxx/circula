package store

import "database/sql"

func (t *txImpl) LastPlatformAuditSeq(action string) (int64, bool, error) {
	var seq int64
	err := t.row(`SELECT seq FROM audit_events WHERE network_id = '' AND action = ? ORDER BY seq DESC LIMIT 1`, action).Scan(&seq)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return seq, err == nil, err
}
