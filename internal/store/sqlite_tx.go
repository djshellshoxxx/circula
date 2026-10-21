package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const txCols = `t.id, t.network_id, t.kind, t.status, t.payer_account_id, t.payee_account_id,
	COALESCE(pa.member_id,''), COALESCE(pb.member_id,''), t.amount, COALESCE(t.listing_id,''),
	t.note, t.reason, COALESCE(t.original_tx_id,''), COALESCE(t.created_by,''), COALESCE(t.accepted_by,''),
	t.reversed_total, t.proposed_at, t.expires_at, t.completed_at, COALESCE(t.ledger_seq,0)`

const txFrom = ` FROM transactions t
	JOIN accounts pa ON pa.id = t.payer_account_id
	JOIN accounts pb ON pb.id = t.payee_account_id`

func (t *txImpl) InsertTransaction(x *domain.Transaction) error {
	_, err := t.exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,
		listing_id,note,reason,original_tx_id,created_by,reversed_total,proposed_at,expires_at)
		VALUES (?,?,?,'proposed',?,?,?,?,?,?,?,?,0,?,?)`,
		x.ID, x.NetworkID, string(x.Kind), x.PayerAccountID, x.PayeeAccountID, x.Amount,
		nullStr(x.ListingID), x.Note, x.Reason, nullStr(x.OriginalTxID), nullStr(x.CreatedBy),
		ms(x.ProposedAt), msPtr(x.ExpiresAt))
	return err
}

func scanTx(r rowScanner) (*domain.Transaction, error) {
	var x domain.Transaction
	var kind, status string
	var proposed int64
	var exp, done sql.NullInt64
	if err := r.Scan(&x.ID, &x.NetworkID, &kind, &status, &x.PayerAccountID, &x.PayeeAccountID,
		&x.PayerMemberID, &x.PayeeMemberID, &x.Amount, &x.ListingID, &x.Note, &x.Reason,
		&x.OriginalTxID, &x.CreatedBy, &x.AcceptedBy, &x.ReversedTotal, &proposed, &exp, &done,
		&x.LedgerSeq); err != nil {
		return nil, err
	}
	x.Kind, x.Status = domain.TxKind(kind), domain.TxStatus(status)
	x.ProposedAt, x.ExpiresAt, x.CompletedAt = fromMS(proposed), ptrTime(exp), ptrTime(done)
	return &x, nil
}

func (t *txImpl) GetTransaction(netID, id string) (*domain.Transaction, error) {
	x, err := scanTx(t.row(`SELECT `+txCols+txFrom+` WHERE t.network_id = ? AND t.id = ?`, netID, id))
	return x, notFound(err, "transaction")
}

func (t *txImpl) UpdateTransaction(x *domain.Transaction) error {
	res, err := t.exec(`UPDATE transactions SET status = ?, accepted_by = ?, note = ?, reversed_total = ?,
		completed_at = ?, ledger_seq = ? WHERE network_id = ? AND id = ?`,
		string(x.Status), nullStr(x.AcceptedBy), x.Note, x.ReversedTotal, msPtr(x.CompletedAt),
		nullInt(x.LedgerSeq), x.NetworkID, x.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return domain.NotFound("transaction")
	}
	return nil
}

func nullInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func (t *txImpl) InsertStatusEvent(e *domain.StatusEvent) error {
	_, err := t.exec(`INSERT INTO transaction_status_events(id,network_id,tx_id,from_status,to_status,actor_id,request_id,occurred_at)
		VALUES (?,?,?,?,?,?,?,?)`, e.ID, e.NetworkID, e.TxID, string(e.From), string(e.To), e.ActorID, e.RequestID, ms(e.OccurredAt))
	return err
}

func (t *txImpl) ListStatusEvents(netID, txID string) ([]*domain.StatusEvent, error) {
	rows, err := t.query(`SELECT id,network_id,tx_id,from_status,to_status,actor_id,request_id,occurred_at
		FROM transaction_status_events WHERE network_id = ? AND tx_id = ? ORDER BY occurred_at, id`, netID, txID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.StatusEvent
	for rows.Next() {
		var e domain.StatusEvent
		var from, to string
		var at int64
		if err := rows.Scan(&e.ID, &e.NetworkID, &e.TxID, &from, &to, &e.ActorID, &e.RequestID, &at); err != nil {
			return nil, err
		}
		e.From, e.To, e.OccurredAt = domain.TxStatus(from), domain.TxStatus(to), fromMS(at)
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (t *txImpl) queryTxs(q string, args []any) ([]*domain.Transaction, error) {
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Transaction
	for rows.Next() {
		x, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (t *txImpl) ListTransactions(netID string, f TxFilter) ([]*domain.Transaction, error) {
	q := `SELECT ` + txCols + txFrom + ` WHERE t.network_id = ?`
	args := []any{netID}
	if f.MemberID != "" {
		q += ` AND (pa.member_id = ? OR pb.member_id = ?)`
		args = append(args, f.MemberID, f.MemberID)
	}
	if len(f.Statuses) > 0 {
		in, a := inClause(f.Statuses)
		q += ` AND t.status IN ` + in
		args = append(args, a...)
	}
	if f.Kind != "" {
		q += ` AND t.kind = ?`
		args = append(args, string(f.Kind))
	}
	if f.OriginalTxID != "" {
		q += ` AND t.original_tx_id = ?`
		args = append(args, f.OriginalTxID)
	}
	if f.CreatedBy != "" {
		q += ` AND t.created_by = ?`
		args = append(args, f.CreatedBy)
	}
	if f.NotCreatedBy != "" {
		q += ` AND (t.created_by IS NULL OR t.created_by <> ?)`
		args = append(args, f.NotCreatedBy)
	}
	if !f.LiveAt.IsZero() {
		q += ` AND (t.expires_at IS NULL OR t.expires_at > ?)`
		args = append(args, ms(f.LiveAt))
	}
	if f.AfterID != "" {
		q += ` AND t.id < ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY t.id DESC LIMIT ?`
	args = append(args, limitOf(f.Limit))
	return t.queryTxs(q, args)
}

func (t *txImpl) ListExpiredProposals(now time.Time, limit int) ([]*domain.Transaction, error) {
	return t.queryTxs(`SELECT `+txCols+txFrom+` WHERE t.status = 'proposed' AND t.expires_at <= ?
		ORDER BY t.expires_at, t.id LIMIT ?`, []any{ms(now), limitOf(limit)})
}

func (t *txImpl) InsertEntry(e *domain.Entry) error {
	_, err := t.exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?,?,?,?,?,?,?)`,
		e.ID, e.NetworkID, e.TxID, e.AccountID, e.Amount, e.LedgerSeq, ms(e.PostedAt))
	return err
}

func (t *txImpl) ListEntries(netID, txID string) ([]*domain.Entry, error) {
	rows, err := t.query(`SELECT id,network_id,tx_id,account_id,amount,ledger_seq,posted_at FROM entries
		WHERE network_id = ? AND tx_id = ? ORDER BY amount, id`, netID, txID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Entry
	for rows.Next() {
		var e domain.Entry
		var at int64
		if err := rows.Scan(&e.ID, &e.NetworkID, &e.TxID, &e.AccountID, &e.Amount, &e.LedgerSeq, &at); err != nil {
			return nil, err
		}
		e.PostedAt = fromMS(at)
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (t *txImpl) SumEntries(accountID string, uptoSeq int64) (int64, error) {
	var v int64
	err := t.row(`SELECT COALESCE(SUM(amount),0) FROM entries WHERE account_id = ? AND (? <= 0 OR ledger_seq <= ?)`,
		accountID, uptoSeq, uptoSeq).Scan(&v)
	return v, err
}

// ListAccountEntries lists an account's entries newest first by ledger_seq.
// beforeSeq > 0 returns entries strictly older than that sequence (keyset).
// (account_id, ledger_seq) is unique: an account has one entry per posting.
func (t *txImpl) ListAccountEntries(netID, accountID string, beforeSeq int64, limit int) ([]*domain.Entry, error) {
	rows, err := t.query(`SELECT id,network_id,tx_id,account_id,amount,ledger_seq,posted_at FROM entries
		WHERE network_id = ? AND account_id = ? AND (? <= 0 OR ledger_seq < ?)
		ORDER BY ledger_seq DESC LIMIT ?`, netID, accountID, beforeSeq, beforeSeq, limitOf(limit))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Entry
	for rows.Next() {
		var e domain.Entry
		var at int64
		if err := rows.Scan(&e.ID, &e.NetworkID, &e.TxID, &e.AccountID, &e.Amount, &e.LedgerSeq, &at); err != nil {
			return nil, err
		}
		e.PostedAt = fromMS(at)
		out = append(out, &e)
	}
	return out, rows.Err()
}

// LiveProposalStats counts and sums unexpired proposed trades created by
// createdBy. counterparty, when set, limits to proposals with that member as
// the other party; requestsOnly limits to those where the creator is the payee.
func (t *txImpl) LiveProposalStats(netID, createdBy, counterparty string, requestsOnly bool, now time.Time) (int, int64, error) {
	q := `SELECT COUNT(*), COALESCE(SUM(t.amount),0)` + txFrom + `
		WHERE t.network_id = ? AND t.status = 'proposed' AND t.kind = 'trade' AND t.expires_at > ? AND t.created_by = ?`
	args := []any{netID, ms(now), createdBy}
	if counterparty != "" {
		q += ` AND (pa.member_id = ? OR pb.member_id = ?)`
		args = append(args, counterparty, counterparty)
	}
	if requestsOnly {
		q += ` AND pb.member_id = t.created_by`
	}
	var n int
	var sum int64
	err := t.row(q, args...).Scan(&n, &sum)
	return n, sum, err
}
