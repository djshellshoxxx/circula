package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

var _ FeatureRepo = (*txImpl)(nil)

func (t *txImpl) InsertDisputeResponse(r DisputeResponse) error {
	_, err := t.exec(`INSERT INTO dispute_responses(id,network_id,dispute_id,responder_id,body,submitted_at)
		VALUES (?,?,?,?,?,?)`, r.ID, r.NetworkID, r.DisputeID, r.ResponderID, r.Body, ms(r.SubmittedAt))
	return err
}

func (t *txImpl) GetDisputeResponse(netID, disputeID string) (*DisputeResponse, error) {
	var r DisputeResponse
	var at int64
	err := t.row(`SELECT id,network_id,dispute_id,responder_id,body,submitted_at FROM dispute_responses
		WHERE network_id = ? AND dispute_id = ?`, netID, disputeID).
		Scan(&r.ID, &r.NetworkID, &r.DisputeID, &r.ResponderID, &r.Body, &at)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	r.SubmittedAt = fromMS(at)
	return &r, nil
}

func (t *txImpl) ListDisputesAwaitingResponse(netID, memberID string, limit int) ([]*domain.Dispute, error) {
	rows, err := t.query(`SELECT `+disputeCols+` FROM disputes d
		WHERE network_id = ? AND status IN ('open','under_review') AND opened_by <> ?
		  AND tx_id IN (SELECT t.id FROM transactions t
			JOIN accounts pa ON pa.id = t.payer_account_id JOIN accounts pb ON pb.id = t.payee_account_id
			WHERE pa.member_id = ? OR pb.member_id = ?)
		  AND NOT EXISTS (SELECT 1 FROM dispute_responses r WHERE r.network_id = d.network_id AND r.dispute_id = d.id)
		ORDER BY id DESC LIMIT ?`, netID, memberID, memberID, memberID, limitOf(limit))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Dispute
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

const reReviewCols = `id,network_id,dispute_id,requested_by,reason,requested_at,status,COALESCE(reviewer_id,''),
	COALESCE(outcome,''),reversal_amount,rationale,decided_at`

func (t *txImpl) InsertReReview(r ReReview) error {
	_, err := t.exec(`INSERT INTO dispute_rereviews(id,network_id,dispute_id,requested_by,reason,requested_at,status)
		VALUES (?,?,?,?,?,?,?)`, r.ID, r.NetworkID, r.DisputeID, r.RequestedBy, r.Reason, ms(r.RequestedAt), r.Status)
	return err
}

func (t *txImpl) GetReReview(netID, disputeID string) (*ReReview, error) {
	var r ReReview
	var req int64
	var dec sql.NullInt64
	err := t.row(`SELECT `+reReviewCols+` FROM dispute_rereviews WHERE network_id = ? AND dispute_id = ?`, netID, disputeID).
		Scan(&r.ID, &r.NetworkID, &r.DisputeID, &r.RequestedBy, &r.Reason, &req, &r.Status, &r.ReviewerID,
			&r.Outcome, &r.ReversalAmount, &r.Rationale, &dec)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	r.RequestedAt, r.DecidedAt = fromMS(req), ptrTime(dec)
	return &r, nil
}

func (t *txImpl) UpdateReReview(r ReReview) error {
	_, err := t.exec(`UPDATE dispute_rereviews SET status=?, reviewer_id=?, outcome=?, reversal_amount=?, rationale=?, decided_at=?
		WHERE network_id = ? AND id = ?`, r.Status, nullStr(r.ReviewerID), nullStr(r.Outcome), r.ReversalAmount,
		r.Rationale, msPtr(r.DecidedAt), r.NetworkID, r.ID)
	return err
}

const reportCols = `id,network_id,reporter_id,target_type,target_id,category,note,status,created_at,
	COALESCE(handled_by,''),handled_at,handling_note`

func (t *txImpl) InsertReport(r Report) error {
	_, err := t.exec(`INSERT INTO reports(id,network_id,reporter_id,target_type,target_id,category,note,status,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`, r.ID, r.NetworkID, r.ReporterID, r.TargetType, r.TargetID, r.Category, r.Note,
		r.Status, ms(r.CreatedAt))
	return err
}

func scanReport(s rowScanner) (*Report, error) {
	var r Report
	var created int64
	var handled sql.NullInt64
	if err := s.Scan(&r.ID, &r.NetworkID, &r.ReporterID, &r.TargetType, &r.TargetID, &r.Category, &r.Note,
		&r.Status, &created, &r.HandledBy, &handled, &r.HandlingNote); err != nil {
		return nil, err
	}
	r.CreatedAt, r.HandledAt = fromMS(created), ptrTime(handled)
	return &r, nil
}

func (t *txImpl) GetReport(netID, id string) (*Report, error) {
	r, err := scanReport(t.row(`SELECT `+reportCols+` FROM reports WHERE network_id = ? AND id = ?`, netID, id))
	return r, notFound(err, "report")
}

func (t *txImpl) UpdateReport(r Report) error {
	_, err := t.exec(`UPDATE reports SET status=?, handled_by=?, handled_at=?, handling_note=? WHERE network_id = ? AND id = ?`,
		r.Status, nullStr(r.HandledBy), msPtr(r.HandledAt), r.HandlingNote, r.NetworkID, r.ID)
	return err
}

func (t *txImpl) ListReports(netID string, f ReportFilter) ([]*Report, error) {
	q := `SELECT ` + reportCols + ` FROM reports WHERE network_id = ?`
	args := []any{netID}
	if f.Status != "" {
		q += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limitOf(f.Limit))
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Report
	for rows.Next() {
		r, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (t *txImpl) CountReportsSince(netID, reporterID string, since time.Time) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM reports WHERE network_id = ? AND reporter_id = ? AND created_at >= ?`,
		netID, reporterID, ms(since)).Scan(&n)
	return n, mapErr(err)
}

func (t *txImpl) InsertNotification(n Notification) (bool, error) {
	res, err := t.exec(`INSERT INTO notifications(id,network_id,member_id,kind,title,body,link,dedupe_key,created_at)
		VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, n.ID, n.NetworkID, n.MemberID, n.Kind, n.Title, n.Body,
		n.Link, n.DedupeKey, ms(n.CreatedAt))
	if err != nil {
		return false, err
	}
	c, err := res.RowsAffected()
	return c > 0, err
}

func (t *txImpl) ListNotifications(netID, memberID string, f NotificationFilter) ([]*Notification, error) {
	q := `SELECT id,network_id,member_id,kind,title,body,link,dedupe_key,created_at,read_at FROM notifications
		WHERE network_id = ? AND member_id = ?`
	args := []any{netID, memberID}
	if f.UnreadOnly {
		q += ` AND read_at IS NULL`
	}
	if f.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limitOf(f.Limit))
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Notification
	for rows.Next() {
		var n Notification
		var created int64
		var read sql.NullInt64
		if err := rows.Scan(&n.ID, &n.NetworkID, &n.MemberID, &n.Kind, &n.Title, &n.Body, &n.Link, &n.DedupeKey,
			&created, &read); err != nil {
			return nil, err
		}
		n.CreatedAt, n.ReadAt = fromMS(created), ptrTime(read)
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (t *txImpl) CountUnreadNotifications(netID, memberID string) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM notifications WHERE network_id = ? AND member_id = ? AND read_at IS NULL`,
		netID, memberID).Scan(&n)
	return n, mapErr(err)
}

func (t *txImpl) DeleteReadNotificationsBefore(cutoff time.Time, limit int) (int64, error) {
	res, err := t.exec(`DELETE FROM notifications WHERE rowid IN
		(SELECT rowid FROM notifications WHERE read_at IS NOT NULL AND read_at <= ? LIMIT ?)`, ms(cutoff), limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) MarkNotificationsRead(netID, memberID string, ids []string, at time.Time) (int64, error) {
	q := `UPDATE notifications SET read_at = ? WHERE network_id = ? AND member_id = ? AND read_at IS NULL`
	args := []any{ms(at), netID, memberID}
	if len(ids) > 0 {
		in, a := inClause(ids)
		q += ` AND id IN ` + in
		args = append(args, a...)
	}
	res, err := t.exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *txImpl) ListNetworkIDs() ([]string, error) {
	rows, err := t.query(`SELECT id FROM networks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
