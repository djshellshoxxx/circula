// Package qrfixture is a TEST FIXTURE standing in for Circula's core: a tiny
// SQLite ledger with proposals, acceptance and two-entry sum-zero posting that
// implements qr.Core. It is not production code, is never registered anywhere
// and imports no driver; tests supply the database handle.
package qrfixture

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	qr "github.com/djshellshoxxx/circula/future/qr"
)

// Schema is the fixture's DDL (separate from the module's own migrations).
const Schema = `
CREATE TABLE IF NOT EXISTS fx_network (id TEXT PRIMARY KEY, name TEXT NOT NULL, active INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS fx_member (
  network_id TEXT NOT NULL, id TEXT NOT NULL, name TEXT NOT NULL,
  active INTEGER NOT NULL, terms INTEGER NOT NULL DEFAULT 1,
  credit_limit INTEGER NOT NULL, balance INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (network_id, id));
CREATE TABLE IF NOT EXISTS fx_tx (
  seq INTEGER PRIMARY KEY AUTOINCREMENT, network_id TEXT NOT NULL,
  payer_id TEXT NOT NULL, payee_id TEXT NOT NULL, amount INTEGER NOT NULL CHECK (amount > 0),
  note TEXT NOT NULL, origin TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS fx_entry (
  seq INTEGER PRIMARY KEY AUTOINCREMENT, network_id TEXT NOT NULL, tx_seq INTEGER NOT NULL,
  member_id TEXT NOT NULL, delta INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS fx_audit (seq INTEGER PRIMARY KEY AUTOINCREMENT, network_id TEXT NOT NULL, event TEXT NOT NULL, subject TEXT NOT NULL);
`

// Fixture implements qr.Core against the fixture schema.
type Fixture struct {
	// MaxTx is the per-transaction cap (default 1e9), like the core's policy.
	MaxTx int64
	// AfterPropose, when set, runs after the proposal rows are written and may
	// return an error to simulate a failure between the proposal and the claim.
	AfterPropose func() error
	// Proposed counts Propose calls that reached the insert (including rolled-back ones).
	Proposed atomic.Int64
}

var _ qr.Core = (*Fixture)(nil)

func txID(seq int64) string { return "tx_" + strconv.FormatInt(seq, 10) }

func parseTx(id string) (int64, bool) {
	if !strings.HasPrefix(id, "tx_") {
		return 0, false
	}
	n, err := strconv.ParseInt(id[3:], 10, 64)
	return n, err == nil && n > 0
}

// Migrate creates the fixture tables.
func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, Schema)
	return err
}

// AddNetwork inserts a network.
func AddNetwork(ctx context.Context, db *sql.DB, id, name string, active bool) error {
	_, err := db.ExecContext(ctx, `INSERT INTO fx_network VALUES(?,?,?)`, id, name, b2i(active))
	return err
}

// AddMember inserts a member with a credit limit and zero balance.
func AddMember(ctx context.Context, db *sql.DB, network, id, name string, creditLimit int64) error {
	_, err := db.ExecContext(ctx, `INSERT INTO fx_member(network_id,id,name,active,terms,credit_limit,balance) VALUES(?,?,?,1,1,?,0)`, network, id, name, creditLimit)
	return err
}

// SetMember updates status flags (active membership, terms) and the credit limit.
func SetMember(ctx context.Context, db *sql.DB, network, id string, active, terms bool, creditLimit int64) error {
	_, err := db.ExecContext(ctx, `UPDATE fx_member SET active=?, terms=?, credit_limit=? WHERE network_id=? AND id=?`, b2i(active), b2i(terms), creditLimit, network, id)
	return err
}

// SetNetworkActive toggles a network.
func SetNetworkActive(ctx context.Context, db *sql.DB, id string, active bool) error {
	_, err := db.ExecContext(ctx, `UPDATE fx_network SET active=? WHERE id=?`, b2i(active), id)
	return err
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (f *Fixture) max() int64 {
	if f.MaxTx > 0 {
		return f.MaxTx
	}
	return 1_000_000_000
}

// Member implements qr.Core.
func (f *Fixture) Member(ctx context.Context, q qr.DBTX, network, id string) (qr.Member, error) {
	var m qr.Member
	var active, terms, netActive int
	err := q.QueryRowContext(ctx, `SELECT m.id, m.name, n.name, m.active, m.terms, n.active FROM fx_member m JOIN fx_network n ON n.id=m.network_id
		WHERE m.network_id=? AND m.id=?`, network, id).Scan(&m.ID, &m.DisplayName, &m.NetworkName, &active, &terms, &netActive)
	if errors.Is(err, sql.ErrNoRows) {
		return qr.Member{}, qr.ErrNotFound
	}
	if err != nil {
		return qr.Member{}, err
	}
	m.Active = active == 1 && terms == 1 && netActive == 1
	return m, nil
}

func (f *Fixture) available(ctx context.Context, q qr.DBTX, network, id string) (int64, error) {
	var bal, lim, reserved int64
	err := q.QueryRowContext(ctx, `SELECT balance, credit_limit FROM fx_member WHERE network_id=? AND id=?`, network, id).Scan(&bal, &lim)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, qr.ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM fx_tx WHERE network_id=? AND payer_id=? AND status='proposed'`, network, id).Scan(&reserved); err != nil {
		return 0, err
	}
	return bal + lim - reserved, nil
}

// AvailableCredit implements qr.Core.
func (f *Fixture) AvailableCredit(ctx context.Context, q qr.DBTX, network, id string) (int64, error) {
	return f.available(ctx, q, network, id)
}

// Propose implements qr.Core: the same checks the real proposal performs, run
// inside the caller's transaction. Nothing is posted until Accept.
func (f *Fixture) Propose(ctx context.Context, q qr.DBTX, in qr.ProposalInput) (qr.ProposalResult, error) {
	payer, err := f.Member(ctx, q, in.NetworkID, in.PayerID)
	if err != nil {
		return qr.ProposalResult{}, err
	}
	if !payer.Active {
		return qr.ProposalResult{}, fmt.Errorf("%w: payer", qr.ErrForbidden)
	}
	payee, err := f.Member(ctx, q, in.NetworkID, in.PayeeID)
	if err != nil {
		return qr.ProposalResult{}, err
	}
	if !payee.Active {
		return qr.ProposalResult{}, qr.ErrCounterpartyInactive
	}
	if in.PayerID == in.PayeeID || in.AmountMinor <= 0 || in.AmountMinor > f.max() {
		return qr.ProposalResult{}, qr.ErrInvalid
	}
	avail, err := f.available(ctx, q, in.NetworkID, in.PayerID)
	if err != nil {
		return qr.ProposalResult{}, err
	}
	if in.AmountMinor > avail {
		return qr.ProposalResult{}, qr.ErrInsufficientCredit
	}
	f.Proposed.Add(1)
	res, err := q.ExecContext(ctx, `INSERT INTO fx_tx(network_id,payer_id,payee_id,amount,note,origin,status) VALUES(?,?,?,?,?,?,'proposed')`,
		in.NetworkID, in.PayerID, in.PayeeID, in.AmountMinor, in.Note, in.Origin)
	if err != nil {
		return qr.ProposalResult{}, err
	}
	seq, _ := res.LastInsertId()
	if _, err := q.ExecContext(ctx, `INSERT INTO fx_audit(network_id,event,subject) VALUES(?,?,?)`, in.NetworkID, "tx.proposed", txID(seq)); err != nil {
		return qr.ProposalResult{}, err
	}
	if f.AfterPropose != nil {
		if err := f.AfterPropose(); err != nil {
			return qr.ProposalResult{}, err
		}
	}
	return qr.ProposalResult{TransactionID: txID(seq), State: qr.TxProposed}, nil
}

// TxStatus implements qr.Core.
func (f *Fixture) TxStatus(ctx context.Context, q qr.DBTX, network, id string) (qr.TxState, error) {
	seq, ok := parseTx(id)
	if !ok {
		return "", qr.ErrNotFound
	}
	var st string
	err := q.QueryRowContext(ctx, `SELECT status FROM fx_tx WHERE network_id=? AND seq=?`, network, seq).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return "", qr.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	switch st {
	case "completed":
		return qr.TxCompleted, nil
	case "proposed":
		return qr.TxProposed, nil
	}
	return qr.TxClosed, nil
}

// Accept is the ordinary consent step: only the payee may accept; it posts two
// balanced entries and marks the transaction completed, all in the caller's transaction.
func (f *Fixture) Accept(ctx context.Context, q qr.DBTX, network, id, actor string) error {
	seq, ok := parseTx(id)
	if !ok {
		return qr.ErrNotFound
	}
	var payer, payee, status string
	var amount int64
	err := q.QueryRowContext(ctx, `SELECT payer_id,payee_id,amount,status FROM fx_tx WHERE network_id=? AND seq=?`, network, seq).Scan(&payer, &payee, &amount, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return qr.ErrNotFound
	}
	if err != nil {
		return err
	}
	if actor != payee {
		return qr.ErrForbidden
	}
	if status != "proposed" {
		return qr.ErrInvalid
	}
	var bal, lim int64
	if err := q.QueryRowContext(ctx, `SELECT balance, credit_limit FROM fx_member WHERE network_id=? AND id=?`, network, payer).Scan(&bal, &lim); err != nil {
		return err
	}
	if bal-amount < -lim {
		return qr.ErrInsufficientCredit
	}
	for _, e := range []struct {
		m string
		d int64
	}{{payer, -amount}, {payee, amount}} {
		if _, err := q.ExecContext(ctx, `UPDATE fx_member SET balance=balance+? WHERE network_id=? AND id=?`, e.d, network, e.m); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO fx_entry(network_id,tx_seq,member_id,delta) VALUES(?,?,?,?)`, network, seq, e.m, e.d); err != nil {
			return err
		}
	}
	if _, err := q.ExecContext(ctx, `UPDATE fx_tx SET status='completed' WHERE seq=?`, seq); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `INSERT INTO fx_audit(network_id,event,subject) VALUES(?,?,?)`, network, "tx.completed", id)
	return err
}

// Decline ends a proposal without posting (payee only).
func (f *Fixture) Decline(ctx context.Context, q qr.DBTX, network, id, actor string) error {
	seq, ok := parseTx(id)
	if !ok {
		return qr.ErrNotFound
	}
	res, err := q.ExecContext(ctx, `UPDATE fx_tx SET status='declined' WHERE network_id=? AND seq=? AND payee_id=? AND status='proposed'`, network, seq, actor)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return qr.ErrNotFound
	}
	return nil
}

// Counts reports fixture totals for assertions: transactions by status, entry
// rows, the sum of all entry deltas (must be 0) and audit rows.
type Counts struct {
	Proposed, Completed, Declined int
	Entries                       int
	EntrySum                      int64
	Audit                         int
}

// Count returns Counts.
func Count(ctx context.Context, q qr.DBTX) (Counts, error) {
	var c Counts
	rows, err := q.QueryContext(ctx, `SELECT status, COUNT(*) FROM fx_tx GROUP BY status`)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return c, err
		}
		switch st {
		case "proposed":
			c.Proposed = n
		case "completed":
			c.Completed = n
		default:
			c.Declined += n
		}
	}
	if err := rows.Err(); err != nil {
		return c, err
	}
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(delta),0) FROM fx_entry`).Scan(&c.Entries, &c.EntrySum); err != nil {
		return c, err
	}
	err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM fx_audit`).Scan(&c.Audit)
	return c, err
}

// Balance returns a member's balance.
func Balance(ctx context.Context, q qr.DBTX, network, id string) (int64, error) {
	var b int64
	err := q.QueryRowContext(ctx, `SELECT balance FROM fx_member WHERE network_id=? AND id=?`, network, id).Scan(&b)
	return b, err
}
