package store

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// Activity signals (SEC-026). The table, its uniqueness and its append-only
// guards are in migration 0019.

func (t *txImpl) CountProposalsSince(netID, memberID string, since time.Time) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM transactions
		WHERE network_id = ? AND kind = 'trade' AND created_by = ? AND proposed_at > ?`, netID, memberID, ms(since)).Scan(&n)
	return n, mapErr(err)
}

func (t *txImpl) CountCompletedTradesBetween(netID, accountA, accountB string, since time.Time) (int, int, error) {
	var ab, ba int
	err := t.row(`SELECT
		(SELECT COUNT(*) FROM transactions WHERE network_id = ? AND kind = 'trade' AND completed_at IS NOT NULL
			AND payer_account_id = ? AND payee_account_id = ? AND completed_at > ?),
		(SELECT COUNT(*) FROM transactions WHERE network_id = ? AND kind = 'trade' AND completed_at IS NOT NULL
			AND payer_account_id = ? AND payee_account_id = ? AND completed_at > ?)`,
		netID, accountA, accountB, ms(since), netID, accountB, accountA, ms(since)).Scan(&ab, &ba)
	return ab, ba, mapErr(err)
}

func (t *txImpl) InsertActivitySignal(s ActivitySignal) (bool, error) {
	// ON CONFLICT DO NOTHING (and the once-only trigger) make a second signal
	// for the same key a no-op inside the caller's unit of work, never a
	// check-then-insert race.
	res, err := t.exec(`INSERT INTO activity_signals(id,network_id,kind,member_id,other_member_id,period_start,window_ms,
		observed,threshold,raised_at) VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`,
		s.ID, s.NetworkID, s.Kind, s.MemberID, nullStr(s.OtherMemberID), ms(s.PeriodStart), s.Window.Milliseconds(),
		s.Observed, s.Threshold, ms(s.RaisedAt))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

const signalCols = `id,network_id,kind,member_id,COALESCE(other_member_id,''),period_start,window_ms,observed,threshold,raised_at`

func (t *txImpl) ListActivitySignals(netID string, f ActivitySignalFilter) ([]*ActivitySignal, error) {
	q := `SELECT ` + signalCols + ` FROM activity_signals WHERE network_id = ?`
	args := []any{netID}
	if f.Kind != "" {
		q += ` AND kind = ?`
		args = append(args, f.Kind)
	}
	if f.Viewer != "" {
		q += ` AND member_id <> ? AND COALESCE(other_member_id,'') <> ?`
		args = append(args, f.Viewer, f.Viewer)
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
	var out []*ActivitySignal
	for rows.Next() {
		var s ActivitySignal
		var period, window, raised int64
		if err := rows.Scan(&s.ID, &s.NetworkID, &s.Kind, &s.MemberID, &s.OtherMemberID, &period, &window,
			&s.Observed, &s.Threshold, &raised); err != nil {
			return nil, err
		}
		s.PeriodStart, s.Window, s.RaisedAt = fromMS(period), time.Duration(window)*time.Millisecond, fromMS(raised)
		out = append(out, &s)
	}
	return out, rows.Err()
}

var savepointSeq atomic.Uint64

// Savepoint runs fn inside SAVEPOINT ... RELEASE. On failure it rolls back to
// the savepoint, so fn's writes, including audit events and the chain head,
// disappear while the enclosing unit continues; the post-commit audit sink
// only sees events that survive.
func (t *txImpl) Savepoint(fn func() error) (err error) {
	name := fmt.Sprintf("sp_%d", savepointSeq.Add(1))
	if _, err := t.q.ExecContext(t.ctx, "SAVEPOINT "+name); err != nil {
		return mapErr(err)
	}
	audits := len(t.audits)
	ok := false
	defer func() {
		if ok {
			return
		}
		t.audits = t.audits[:audits]
		_, rerr := t.q.ExecContext(t.ctx, "ROLLBACK TO "+name)
		_, xerr := t.q.ExecContext(t.ctx, "RELEASE "+name)
		if p := recover(); p != nil {
			panic(p)
		}
		if rerr != nil || xerr != nil {
			err = errors.Join(err, mapErr(rerr), mapErr(xerr))
		}
	}()
	if err = fn(); err != nil {
		return err
	}
	if _, err = t.q.ExecContext(t.ctx, "RELEASE "+name); err != nil {
		return mapErr(err)
	}
	ok = true
	return nil
}
