package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const bgCols = `id,network_id,operator_label,reason,opened_at,expires_at,closed_at,closed_reason`

func scanBG(r rowScanner) (*domain.BreakGlassSession, error) {
	var b domain.BreakGlassSession
	var opened, exp int64
	var closed sql.NullInt64
	if err := r.Scan(&b.ID, &b.NetworkID, &b.OperatorLabel, &b.Reason, &opened, &exp, &closed, &b.ClosedReason); err != nil {
		return nil, err
	}
	b.OpenedAt, b.ExpiresAt, b.ClosedAt = fromMS(opened), fromMS(exp), ptrTime(closed)
	return &b, nil
}

func (t *txImpl) InsertBreakGlass(b *domain.BreakGlassSession) error {
	_, err := t.exec(`INSERT INTO break_glass_sessions(`+bgCols+`) VALUES (?,?,?,?,?,?,?,?)`,
		b.ID, b.NetworkID, b.OperatorLabel, b.Reason, ms(b.OpenedAt), ms(b.ExpiresAt), msPtr(b.ClosedAt), b.ClosedReason)
	return err
}

func (t *txImpl) GetBreakGlass(id string) (*domain.BreakGlassSession, error) {
	b, err := scanBG(t.row(`SELECT `+bgCols+` FROM break_glass_sessions WHERE id = ?`, id))
	return b, notFound(err, "break-glass session")
}

func (t *txImpl) CloseBreakGlass(id string, at time.Time, reason string) (bool, error) {
	res, err := t.exec(`UPDATE break_glass_sessions SET closed_at = ?, closed_reason = ? WHERE id = ? AND closed_at IS NULL`,
		ms(at), reason, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) listBG(q string, args ...any) ([]*domain.BreakGlassSession, error) {
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.BreakGlassSession
	for rows.Next() {
		b, err := scanBG(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (t *txImpl) ListBreakGlass(netID, afterID string, limit int) ([]*domain.BreakGlassSession, error) {
	q, args := `SELECT `+bgCols+` FROM break_glass_sessions WHERE network_id = ?`, []any{netID}
	if afterID != "" {
		q += ` AND id < ?`
		args = append(args, afterID)
	}
	return t.listBG(q+` ORDER BY id DESC LIMIT ?`, append(args, limitOf(limit))...)
}

func (t *txImpl) LiveBreakGlass(netID string, now time.Time) ([]*domain.BreakGlassSession, error) {
	return t.listBG(`SELECT `+bgCols+` FROM break_glass_sessions
		WHERE network_id = ? AND closed_at IS NULL AND expires_at > ? ORDER BY id DESC`, netID, ms(now))
}

func (t *txImpl) ExpiredOpenBreakGlass(now time.Time, limit int) ([]*domain.BreakGlassSession, error) {
	return t.listBG(`SELECT `+bgCols+` FROM break_glass_sessions
		WHERE closed_at IS NULL AND expires_at <= ? ORDER BY id LIMIT ?`, ms(now), limitOf(limit))
}

func (t *txImpl) InsertBreakGlassGrant(g *domain.BreakGlassGrant) error {
	_, err := t.exec(`INSERT INTO break_glass_grants(id,session_id,network_id,member_id,role,granted_at) VALUES (?,?,?,?,?,?)`,
		g.ID, g.SessionID, g.NetworkID, g.MemberID, string(g.Role), ms(g.GrantedAt))
	return err
}

func (t *txImpl) LiveBreakGlassRoles(netID, memberID string, now time.Time) ([]domain.Role, error) {
	rows, err := t.query(`SELECT DISTINCT g.role FROM break_glass_grants g
		JOIN break_glass_sessions s ON s.id = g.session_id
		WHERE g.network_id = ? AND g.member_id = ? AND s.network_id = g.network_id
		  AND s.closed_at IS NULL AND s.expires_at > ? ORDER BY g.role`, netID, memberID, ms(now))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Role
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		out = append(out, domain.Role(r))
	}
	return out, rows.Err()
}
