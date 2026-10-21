package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) SetMemberDisplayName(netID, memberID, name string) error {
	_, err := t.exec(`UPDATE members SET display_name = ? WHERE network_id = ? AND id = ?`, name, netID, memberID)
	return err
}

func (t *txImpl) AnonymiseListings(netID, memberID, title string, at time.Time) (int64, error) {
	res, err := t.exec(`UPDATE listings SET title = ?, description = '', status = 'archived', updated_at = ?
		WHERE network_id = ? AND member_id = ?`, title, ms(at), netID, memberID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) RedactTransactionNotes(netID, memberID string) (int64, error) {
	res, err := t.exec(`UPDATE transactions SET note = '' WHERE network_id = ? AND created_by = ? AND note <> ''`, netID, memberID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) RedactDisputeStatements(netID, memberID string) (int64, error) {
	res, err := t.exec(`UPDATE disputes SET statement = '' WHERE network_id = ? AND opened_by = ? AND statement <> ''`, netID, memberID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) AnonymisePerson(personID, login, unusableHash string) error {
	_, err := t.exec(`UPDATE persons SET login = ?, password_hash = ? WHERE id = ?`, login, unusableHash, personID)
	return err
}

func (t *txImpl) RedactTokensOf(personID string, at time.Time) error {
	_, err := t.exec(`UPDATE api_tokens SET name = 'erased', revoked_at = COALESCE(revoked_at, ?) WHERE person_id = ?`, ms(at), personID)
	return err
}

func (t *txImpl) InsertErasure(e *domain.ErasureRecord) error {
	_, err := t.exec(`INSERT INTO member_erasures(member_id,network_id,person_id,erased_at,person_erased) VALUES (?,?,?,?,?)`,
		e.MemberID, e.NetworkID, e.PersonID, ms(e.ErasedAt), b2i(e.PersonErased))
	return err
}

func (t *txImpl) GetErasure(netID, memberID string) (*domain.ErasureRecord, error) {
	var e domain.ErasureRecord
	var at int64
	var pe int
	err := t.row(`SELECT member_id,network_id,person_id,erased_at,person_erased FROM member_erasures
		WHERE network_id = ? AND member_id = ?`, netID, memberID).Scan(&e.MemberID, &e.NetworkID, &e.PersonID, &at, &pe)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	e.ErasedAt, e.PersonErased = fromMS(at), pe == 1
	return &e, nil
}
