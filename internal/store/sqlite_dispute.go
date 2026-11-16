package store

import (
	"database/sql"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const disputeCols = `id,network_id,tx_id,opened_by,category,statement,status,COALESCE(outcome,''),
	reversal_amount,COALESCE(reviewer_id,''),rationale,opened_at,decided_at`

func (t *txImpl) InsertDispute(d *domain.Dispute) error {
	_, err := t.exec(`INSERT INTO disputes(id,network_id,tx_id,opened_by,category,statement,status,opened_at)
		VALUES (?,?,?,?,?,?,?,?)`, d.ID, d.NetworkID, d.TxID, d.OpenedBy, d.Category, d.Statement,
		string(d.Status), ms(d.OpenedAt))
	return err
}

func scanDispute(r rowScanner) (*domain.Dispute, error) {
	var d domain.Dispute
	var status, outcome string
	var opened int64
	var decided sql.NullInt64
	if err := r.Scan(&d.ID, &d.NetworkID, &d.TxID, &d.OpenedBy, &d.Category, &d.Statement, &status,
		&outcome, &d.ReversalAmount, &d.ReviewerID, &d.Rationale, &opened, &decided); err != nil {
		return nil, err
	}
	d.Status, d.Outcome = domain.DisputeStatus(status), domain.DisputeOutcome(outcome)
	d.OpenedAt, d.DecidedAt = fromMS(opened), ptrTime(decided)
	return &d, nil
}

func (t *txImpl) GetDispute(netID, id string) (*domain.Dispute, error) {
	d, err := scanDispute(t.row(`SELECT `+disputeCols+` FROM disputes WHERE network_id = ? AND id = ?`, netID, id))
	return d, notFound(err, "dispute")
}

func (t *txImpl) UpdateDispute(d *domain.Dispute) error {
	_, err := t.exec(`UPDATE disputes SET status=?, outcome=?, reversal_amount=?, reviewer_id=?, rationale=?, decided_at=?
		WHERE network_id = ? AND id = ?`, string(d.Status), nullStr(string(d.Outcome)), d.ReversalAmount,
		nullStr(d.ReviewerID), d.Rationale, msPtr(d.DecidedAt), d.NetworkID, d.ID)
	return err
}

func (t *txImpl) ListDisputes(netID string, f DisputeFilter) ([]*domain.Dispute, error) {
	q := `SELECT ` + disputeCols + ` FROM disputes WHERE network_id = ?`
	args := []any{netID}
	if f.PartyID != "" {
		q += ` AND tx_id IN (SELECT t.id FROM transactions t
			JOIN accounts pa ON pa.id = t.payer_account_id JOIN accounts pb ON pb.id = t.payee_account_id
			WHERE pa.member_id = ? OR pb.member_id = ?)`
		args = append(args, f.PartyID, f.PartyID)
	}
	if len(f.Statuses) > 0 {
		in, a := inClause(f.Statuses)
		q += ` AND status IN ` + in
		args = append(args, a...)
	}
	if f.TxID != "" {
		q += ` AND tx_id = ?`
		args = append(args, f.TxID)
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

func (t *txImpl) InsertDisputeEvent(e DisputeEvent) error {
	_, err := t.exec(`INSERT INTO dispute_events(id,network_id,dispute_id,actor_id,action,outcome,occurred_at)
		VALUES (?,?,?,?,?,?,?)`, e.ID, e.NetworkID, e.DisputeID, e.ActorID, e.Action, e.Outcome, ms(e.OccurredAt))
	return err
}

func (t *txImpl) ListDisputeEvents(netID, disputeID string) ([]DisputeEvent, error) {
	rows, err := t.query(`SELECT id,network_id,dispute_id,actor_id,action,outcome,occurred_at FROM dispute_events
		WHERE network_id = ? AND dispute_id = ? ORDER BY occurred_at, id`, netID, disputeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []DisputeEvent
	for rows.Next() {
		var e DisputeEvent
		var at int64
		if err := rows.Scan(&e.ID, &e.NetworkID, &e.DisputeID, &e.ActorID, &e.Action, &e.Outcome, &at); err != nil {
			return nil, err
		}
		e.OccurredAt = fromMS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}
