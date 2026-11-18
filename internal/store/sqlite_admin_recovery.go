package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Administrator recovery requests (GOV-021). Every transition is a
// compare-and-set on the current status; migration triggers keep the subject
// immutable and the states final.

const arCols = `id,network_id,requester_member_id,status,requested_at,expires_at,first_operator,first_reason,
	first_approved_at,closed_by,closed_reason,closed_at,COALESCE(role_assignment_id,'')`

func scanAdminRecovery(r rowScanner) (*domain.AdminRecoveryRequest, error) {
	var a domain.AdminRecoveryRequest
	var status string
	var requested, expires int64
	var first, closed sql.NullInt64
	if err := r.Scan(&a.ID, &a.NetworkID, &a.RequesterID, &status, &requested, &expires, &a.FirstOperator, &a.FirstReason,
		&first, &a.ClosedBy, &a.ClosedReason, &closed, &a.RoleAssignment); err != nil {
		return nil, err
	}
	a.Status = domain.AdminRecoveryStatus(status)
	a.RequestedAt, a.ExpiresAt = fromMS(requested), fromMS(expires)
	a.FirstApprovedAt, a.ClosedAt = ptrTime(first), ptrTime(closed)
	return &a, nil
}

func (t *txImpl) InsertAdminRecovery(r *domain.AdminRecoveryRequest) error {
	_, err := t.exec(`INSERT INTO admin_recovery_requests(id,network_id,requester_member_id,status,requested_at,expires_at)
		VALUES (?,?,?,?,?,?)`, r.ID, r.NetworkID, r.RequesterID, string(r.Status), ms(r.RequestedAt), ms(r.ExpiresAt))
	return err
}

func (t *txImpl) GetAdminRecovery(id string) (*domain.AdminRecoveryRequest, error) {
	a, err := scanAdminRecovery(t.row(`SELECT `+arCols+` FROM admin_recovery_requests WHERE id = ?`, id))
	return a, notFound(err, "recovery request")
}

func (t *txImpl) listAdminRecovery(q string, args ...any) ([]*domain.AdminRecoveryRequest, error) {
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.AdminRecoveryRequest
	for rows.Next() {
		a, err := scanAdminRecovery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (t *txImpl) ListAdminRecovery(netID string, openOnly bool, afterID string, limit int) ([]*domain.AdminRecoveryRequest, error) {
	q, args := `SELECT `+arCols+` FROM admin_recovery_requests WHERE 1 = 1`, []any{}
	if netID != "" {
		q += ` AND network_id = ?`
		args = append(args, netID)
	}
	if openOnly {
		q += ` AND status IN ('requested','approved')`
	}
	if afterID != "" {
		q += ` AND id < ?`
		args = append(args, afterID)
	}
	return t.listAdminRecovery(q+` ORDER BY id DESC LIMIT ?`, append(args, limitOf(limit))...)
}

func (t *txImpl) ApproveAdminRecoveryFirst(id, operator, reason string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE admin_recovery_requests SET status = 'approved', first_operator = ?, first_reason = ?,
		first_approved_at = ? WHERE id = ? AND status = 'requested'`, operator, reason, ms(at), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) CloseAdminRecovery(id string, from, to domain.AdminRecoveryStatus, by, reason string, at time.Time, roleAssignmentID string) (bool, error) {
	res, err := t.exec(`UPDATE admin_recovery_requests SET status = ?, closed_by = ?, closed_reason = ?, closed_at = ?,
		role_assignment_id = ? WHERE id = ? AND status = ?`,
		string(to), by, reason, ms(at), nullStr(roleAssignmentID), id, string(from))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) GetOpenAdminRecovery(netID, memberID string) (*domain.AdminRecoveryRequest, error) {
	a, err := scanAdminRecovery(t.row(`SELECT `+arCols+` FROM admin_recovery_requests
		WHERE network_id = ? AND requester_member_id = ? AND status IN ('requested','approved')`, netID, memberID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, mapErr(err)
}

func (t *txImpl) ExpiredOpenAdminRecovery(now time.Time, afterID string, limit int) ([]*domain.AdminRecoveryRequest, error) {
	return t.listAdminRecovery(`SELECT `+arCols+` FROM admin_recovery_requests
		WHERE status IN ('requested','approved') AND expires_at <= ? AND id > ? ORDER BY id LIMIT ?`, ms(now), afterID, limitOf(limit))
}
