package store

import "time"

const sessionRevokeOthersSQL = `UPDATE sessions SET revoked_at = ?
	WHERE person_id = ? AND id <> ? AND revoked_at IS NULL
	AND EXISTS (SELECT 1 FROM sessions AS current_session
		WHERE current_session.id = ? AND current_session.person_id = ?
		AND current_session.revoked_at IS NULL)`

func (t *txImpl) RevokeOtherOwnedSessions(personID, keepID string, at time.Time) (int64, error) {
	res, err := t.exec(sessionRevokeOthersSQL, ms(at), personID, keepID, keepID, personID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
