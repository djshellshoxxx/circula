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
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	// Free text the member wrote in later features (migration 0008): the
	// response to a dispute, a re-review reason and abuse-report notes.
	for _, q := range []string{
		`UPDATE dispute_responses SET body = '' WHERE network_id = ? AND responder_id = ? AND body <> ''`,
		`UPDATE dispute_rereviews SET reason = '' WHERE network_id = ? AND requested_by = ? AND reason <> ''`,
		`UPDATE reports SET note = '' WHERE network_id = ? AND reporter_id = ? AND note <> ''`,
	} {
		r, err := t.exec(q, netID, memberID)
		if err != nil {
			return 0, err
		}
		c, err := r.RowsAffected()
		if err != nil {
			return 0, err
		}
		n += c
	}
	return n, nil
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

// RedactNotificationNames clears the member's inbox and notifications linked
// to their member record, trades or disputes. Resource relations identify the
// affected notices; a shared name or name substring never selects other mail.
func (t *txImpl) RedactNotificationNames(netID, memberID, _ string) (int64, error) {
	res, err := t.exec(`WITH member_trades AS (
		SELECT tr.id FROM transactions tr
		JOIN accounts a ON a.network_id = tr.network_id AND a.member_id = ?
		WHERE tr.network_id = ? AND (a.id = tr.payer_account_id OR a.id = tr.payee_account_id)
	), related_links AS (
		SELECT '/trades/' || id AS link FROM member_trades
		UNION SELECT '/disputes/' || id FROM disputes
			WHERE network_id = ? AND tx_id IN (SELECT id FROM member_trades)
		UNION SELECT '/members/' || ?
		UNION SELECT '/admin/members/' || ?
	)
	UPDATE notifications SET body = '', event_date = NULL WHERE network_id = ? AND body <> ''
		AND (member_id = ? OR link IN (SELECT link FROM related_links))`,
		memberID, netID, netID, memberID, memberID, netID, memberID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
