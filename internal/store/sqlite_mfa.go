package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) GetMFA(personID string) (*domain.MFARecord, error) {
	var r domain.MFARecord
	var en int
	var created int64
	var enabledAt sql.NullInt64
	err := t.row(`SELECT person_id,secret,enabled,last_step,created_at,enabled_at FROM person_mfa WHERE person_id = ?`,
		personID).Scan(&r.PersonID, &r.Secret, &en, &r.LastStep, &created, &enabledAt)
	if err != nil {
		return nil, notFound(err, "mfa")
	}
	r.Enabled, r.CreatedAt, r.EnabledAt = en == 1, fromMS(created), ptrTime(enabledAt)
	return &r, nil
}

func (t *txImpl) PutMFA(r *domain.MFARecord) error {
	en := 0
	if r.Enabled {
		en = 1
	}
	var enabledAt any
	if r.EnabledAt != nil {
		enabledAt = ms(*r.EnabledAt)
	}
	_, err := t.exec(`INSERT INTO person_mfa(person_id,secret,enabled,last_step,created_at,enabled_at) VALUES (?,?,?,?,?,?)
		ON CONFLICT(person_id) DO UPDATE SET secret=excluded.secret, enabled=excluded.enabled,
		last_step=excluded.last_step, created_at=excluded.created_at, enabled_at=excluded.enabled_at`,
		r.PersonID, r.Secret, en, r.LastStep, ms(r.CreatedAt), enabledAt)
	return err
}

func (t *txImpl) DeleteMFA(personID string) error {
	_, err := t.exec(`DELETE FROM person_mfa WHERE person_id = ?`, personID)
	return err
}

func (t *txImpl) AdvanceMFAStep(personID string, step int64) (bool, error) {
	res, err := t.exec(`UPDATE person_mfa SET last_step = ? WHERE person_id = ? AND last_step < ?`, step, personID, step)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (t *txImpl) ReplaceRecoveryCodes(personID string, hashes []string) error {
	if _, err := t.exec(`DELETE FROM recovery_codes WHERE person_id = ?`, personID); err != nil {
		return err
	}
	for _, h := range hashes {
		if _, err := t.exec(`INSERT INTO recovery_codes(person_id,code_hash) VALUES (?,?)`, personID, h); err != nil {
			return err
		}
	}
	return nil
}

func (t *txImpl) ConsumeRecoveryCode(personID, hash string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE recovery_codes SET used_at = ? WHERE person_id = ? AND code_hash = ? AND used_at IS NULL`,
		ms(at), personID, hash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (t *txImpl) CountRecoveryCodes(personID string) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM recovery_codes WHERE person_id = ? AND used_at IS NULL`, personID).Scan(&n)
	return n, err
}

func (t *txImpl) InsertMFAChallenge(c *domain.MFAChallenge) error {
	_, err := t.exec(`INSERT INTO mfa_challenges(id_hash,person_id,created_at,expires_at) VALUES (?,?,?,?)`,
		c.IDHash, c.PersonID, ms(c.CreatedAt), ms(c.ExpiresAt))
	return err
}

func (t *txImpl) GetMFAChallenge(idHash string) (*domain.MFAChallenge, error) {
	var c domain.MFAChallenge
	var created, exp int64
	var used sql.NullInt64
	err := t.row(`SELECT id_hash,person_id,created_at,expires_at,used_at FROM mfa_challenges WHERE id_hash = ?`,
		idHash).Scan(&c.IDHash, &c.PersonID, &created, &exp, &used)
	if err != nil {
		return nil, notFound(err, "mfa challenge")
	}
	c.CreatedAt, c.ExpiresAt, c.UsedAt = fromMS(created), fromMS(exp), ptrTime(used)
	return &c, nil
}

func (t *txImpl) UseMFAChallenge(idHash string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE mfa_challenges SET used_at = ? WHERE id_hash = ? AND used_at IS NULL`, ms(at), idHash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (t *txImpl) DeleteMFAChallengesOf(personID string) error {
	_, err := t.exec(`DELETE FROM mfa_challenges WHERE person_id = ?`, personID)
	return err
}

func (t *txImpl) DeleteDeadMFAChallenges(personID string, cutoff time.Time) error {
	_, err := t.exec(`DELETE FROM mfa_challenges WHERE person_id = ? AND (expires_at <= ? OR used_at IS NOT NULL)`,
		personID, ms(cutoff))
	return err
}

func (t *txImpl) InsertNotice(n *domain.SecurityNotice) error {
	_, err := t.exec(`INSERT INTO security_notices(id,person_id,kind,created_at,detail) VALUES (?,?,?,?,?)`,
		n.ID, n.PersonID, n.Kind, ms(n.CreatedAt), n.Detail)
	return err
}

func (t *txImpl) ListNotices(personID string, unseenOnly bool, limit int) ([]*domain.SecurityNotice, error) {
	q := `SELECT id,person_id,kind,created_at,seen_at,detail FROM security_notices WHERE person_id = ?`
	if unseenOnly {
		q += ` AND seen_at IS NULL`
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	rows, err := t.query(q, personID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.SecurityNotice
	for rows.Next() {
		var n domain.SecurityNotice
		var created int64
		var seen sql.NullInt64
		if err := rows.Scan(&n.ID, &n.PersonID, &n.Kind, &created, &seen, &n.Detail); err != nil {
			return nil, err
		}
		n.CreatedAt, n.SeenAt = fromMS(created), ptrTime(seen)
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (t *txImpl) CountUnseenNotices(personID string) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM security_notices WHERE person_id = ? AND seen_at IS NULL`, personID).Scan(&n)
	return n, err
}

func (t *txImpl) AckNotices(personID string, at time.Time) (int64, error) {
	res, err := t.exec(`UPDATE security_notices SET seen_at = ? WHERE person_id = ? AND seen_at IS NULL`, ms(at), personID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) DeleteNoticesOf(personID string) error {
	_, err := t.exec(`DELETE FROM security_notices WHERE person_id = ?`, personID)
	return err
}
