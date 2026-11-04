package store

import (
	"database/sql"
	"errors"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func zfrom(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return fromMS(v)
}

const closureCols = `network_id,reason,scheduled_by,scheduled_at,closing_at,notice_days,read_only_days,closed_at,
	final_ledger_seq,final_sha256,final_balance_total,final_entries_sum,erased_at`

func scanClosure(r rowScanner) (*domain.Closure, error) {
	var c domain.Closure
	var sched, closing, closed, erased int64
	if err := r.Scan(&c.NetworkID, &c.Reason, &c.ScheduledBy, &sched, &closing, &c.NoticeDays, &c.ReadOnlyDays, &closed,
		&c.FinalLedgerSeq, &c.FinalSHA256, &c.FinalBalanceTotal, &c.FinalEntriesSum, &erased); err != nil {
		return nil, err
	}
	c.ScheduledAt, c.ClosingAt, c.ClosedAt, c.ErasedAt = fromMS(sched), fromMS(closing), zfrom(closed), zfrom(erased)
	return &c, nil
}

func (t *txImpl) GetClosure(netID string) (*domain.Closure, error) {
	c, err := scanClosure(t.row(`SELECT `+closureCols+` FROM network_closures WHERE network_id = ?`, netID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (t *txImpl) PutClosure(c domain.Closure) (bool, error) {
	res, err := t.exec(`INSERT OR IGNORE INTO network_closures(network_id,reason,scheduled_by,scheduled_at,closing_at,notice_days,read_only_days)
		VALUES (?,?,?,?,?,?,?)`, c.NetworkID, c.Reason, c.ScheduledBy, ms(c.ScheduledAt), ms(c.ClosingAt), c.NoticeDays, c.ReadOnlyDays)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) DeleteOpenClosure(netID string) (bool, error) {
	res, err := t.exec(`DELETE FROM network_closures WHERE network_id = ? AND closed_at = 0`, netID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) SetClosureDone(netID string, at time.Time) error {
	_, err := t.exec(`UPDATE network_closures SET closed_at = ? WHERE network_id = ?`, ms(at), netID)
	return err
}

func (t *txImpl) SetClosureFinal(netID string, seq int64, sha string, bal, sum int64) error {
	_, err := t.exec(`UPDATE network_closures SET final_ledger_seq = ?, final_sha256 = ?, final_balance_total = ?, final_entries_sum = ?
		WHERE network_id = ?`, seq, sha, bal, sum, netID)
	return err
}

func (t *txImpl) SetClosureErased(netID string, at time.Time) error {
	_, err := t.exec(`UPDATE network_closures SET erased_at = ? WHERE network_id = ?`, ms(at), netID)
	return err
}

func (t *txImpl) ListClosures(pending bool) ([]domain.Closure, error) {
	q := `SELECT ` + closureCols + ` FROM network_closures WHERE closed_at = 0 ORDER BY network_id`
	if !pending {
		q = `SELECT ` + closureCols + ` FROM network_closures WHERE closed_at <> 0 AND erased_at = 0 ORDER BY network_id`
	}
	rows, err := t.query(q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Closure
	for rows.Next() {
		c, err := scanClosure(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (t *txImpl) InsertChallenge(c Challenge) error {
	_, err := t.exec(`INSERT INTO challenges(id,network_id,member_id,kind,subject_id,reason,involved_id,status,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`, c.ID, c.NetworkID, c.MemberID, c.Kind, c.SubjectID, c.Reason, c.InvolvedID, c.Status, ms(c.CreatedAt))
	return err
}

const challengeCols = `id,network_id,member_id,kind,subject_id,reason,involved_id,status,outcome,decided_by,rationale,created_at,decided_at`

func scanChallenge(r rowScanner) (*Challenge, error) {
	var c Challenge
	var created, decided int64
	if err := r.Scan(&c.ID, &c.NetworkID, &c.MemberID, &c.Kind, &c.SubjectID, &c.Reason, &c.InvolvedID, &c.Status,
		&c.Outcome, &c.DecidedBy, &c.Rationale, &created, &decided); err != nil {
		return nil, err
	}
	c.CreatedAt, c.DecidedAt = fromMS(created), zfrom(decided)
	return &c, nil
}

func (t *txImpl) GetChallenge(netID, id string) (*Challenge, error) {
	c, err := scanChallenge(t.row(`SELECT `+challengeCols+` FROM challenges WHERE network_id = ? AND id = ?`, netID, id))
	return c, notFound(err, "challenge")
}

func (t *txImpl) DecideChallenge(c Challenge) error {
	_, err := t.exec(`UPDATE challenges SET status = 'decided', outcome = ?, decided_by = ?, rationale = ?, decided_at = ?
		WHERE network_id = ? AND id = ? AND status = 'requested'`, c.Outcome, c.DecidedBy, c.Rationale, ms(c.DecidedAt), c.NetworkID, c.ID)
	return err
}

func (t *txImpl) ListChallenges(netID, memberID, status string, limit int) ([]Challenge, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	rows, err := t.query(`SELECT `+challengeCols+` FROM challenges WHERE network_id = ? AND (? = '' OR member_id = ?)
		AND (? = '' OR status = ?) ORDER BY created_at DESC, id DESC LIMIT ?`, netID, memberID, memberID, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Challenge
	for rows.Next() {
		c, err := scanChallenge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (t *txImpl) RecordDeadlineNotice(netID, disputeID, kind string, at time.Time) (bool, error) {
	res, err := t.exec(`INSERT OR IGNORE INTO dispute_deadline_notices(network_id,dispute_id,kind,notified_at) VALUES (?,?,?,?)`,
		netID, disputeID, kind, ms(at))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

var _ ClosureRepo = (*txImpl)(nil)
