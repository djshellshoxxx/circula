package store

import "time"

func (t *txImpl) ReplaceRecoveryCodes(personID string, ids, hashes []string, at time.Time) error {
	if _, err := t.exec(`DELETE FROM account_recovery_codes WHERE person_id = ?`, personID); err != nil {
		return err
	}
	for i, h := range hashes {
		if _, err := t.exec(`INSERT INTO account_recovery_codes(id,person_id,code_hash,created_at) VALUES (?,?,?,?)`,
			ids[i], personID, h, ms(at)); err != nil {
			return err
		}
	}
	return nil
}

func (t *txImpl) ConsumeRecoveryCode(personID, hash string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE account_recovery_codes SET used_at = ? WHERE person_id = ? AND code_hash = ? AND used_at IS NULL`,
		ms(at), personID, hash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (t *txImpl) CountRecoveryCodes(personID string) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM account_recovery_codes WHERE person_id = ? AND used_at IS NULL`, personID).Scan(&n)
	return n, err
}

func (t *txImpl) DeleteRecoveryCodes(personID string) error {
	_, err := t.exec(`DELETE FROM account_recovery_codes WHERE person_id = ?`, personID)
	return err
}

func (t *txImpl) MustChangePassword(personID string) (bool, error) {
	var v int
	if err := t.row(`SELECT must_change_password FROM persons WHERE id = ?`, personID).Scan(&v); err != nil {
		return false, notFound(err, "person")
	}
	return v == 1, nil
}

func (t *txImpl) SetMustChangePassword(personID string, v bool) error {
	n := 0
	if v {
		n = 1
	}
	_, err := t.exec(`UPDATE persons SET must_change_password = ? WHERE id = ?`, n, personID)
	return err
}

func (t *txImpl) RevokeTokensOf(personID string, at time.Time) error {
	_, err := t.exec(`UPDATE api_tokens SET revoked_at = ? WHERE person_id = ? AND revoked_at IS NULL`, ms(at), personID)
	return err
}
