package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) InsertInvitation(inv *domain.Invitation, tokenHash string) error {
	_, err := t.exec(`INSERT INTO invitations(id,network_id,token_hash,created_by,created_at,expires_at) VALUES (?,?,?,?,?,?)`,
		inv.ID, inv.NetworkID, tokenHash, inv.CreatedBy, ms(inv.CreatedAt), ms(inv.ExpiresAt))
	return err
}

const invCols = `id,network_id,created_by,created_at,expires_at,used_at,COALESCE(used_by,''),revoked_at`

func scanInvitation(r rowScanner) (*domain.Invitation, error) {
	var i domain.Invitation
	var created, exp int64
	var used, rev sql.NullInt64
	if err := r.Scan(&i.ID, &i.NetworkID, &i.CreatedBy, &created, &exp, &used, &i.UsedBy, &rev); err != nil {
		return nil, err
	}
	i.CreatedAt, i.ExpiresAt, i.UsedAt, i.RevokedAt = fromMS(created), fromMS(exp), ptrTime(used), ptrTime(rev)
	return &i, nil
}

func (t *txImpl) GetInvitationByHash(h string) (*domain.Invitation, error) {
	i, err := scanInvitation(t.row(`SELECT `+invCols+` FROM invitations WHERE token_hash = ?`, h))
	return i, notFound(err, "invitation")
}

func (t *txImpl) MarkInvitationUsed(id, memberID string, at time.Time) error {
	res, err := t.exec(`UPDATE invitations SET used_at = ?, used_by = ? WHERE id = ? AND used_at IS NULL`, ms(at), memberID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return domain.NotFound("invitation")
	}
	return nil
}

func (t *txImpl) RevokeInvitation(netID, id string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE invitations SET revoked_at = ? WHERE network_id = ? AND id = ? AND revoked_at IS NULL AND used_at IS NULL`,
		ms(at), netID, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) ListInvitations(netID string) ([]*domain.Invitation, error) {
	rows, err := t.query(`SELECT `+invCols+` FROM invitations WHERE network_id = ? ORDER BY id DESC`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Invitation
	for rows.Next() {
		i, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
