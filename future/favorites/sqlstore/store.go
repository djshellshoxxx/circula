// Package sqlstore is a database/sql implementation of favorites.Store for
// SQLite. It imports no driver: the host (or a test fixture) opens the
// *sql.DB with its own SQLite driver.
//
// Requirements on the *sql.DB (adapter responsibilities, one process per
// database file as in the host application):
//
//   - SQLite 3.35 or newer (RETURNING, ON CONFLICT DO NOTHING).
//   - WAL journal mode and a busy timeout, set per connection through the
//     driver DSN, so readers do not block the single writer.
//   - Foreign keys on, if the integration migration adds them.
//
// Write transactions are serialized in-process and start with BEGIN IMMEDIATE,
// so a read-then-write sequence cannot fail late with a snapshot conflict.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
)

// DBTX is the subset of *sql.DB, *sql.Conn and *sql.Tx used here. Host code
// can pass its own *sql.Tx to PurgeMemberTx to purge inside a departure
// transaction.
type DBTX interface {
	ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row
}

// Store implements favorites.Store over a *sql.DB.
type Store struct {
	db *sql.DB
	mu sync.Mutex // serializes in-process writers
}

// New wraps db. Call Migrate first.
func New(db *sql.DB) *Store { return &Store{db: db} }

var errReadOnly = errors.New("sqlstore: write in read-only transaction")

// withImmediate runs fn inside BEGIN IMMEDIATE ... COMMIT on one connection.
func withImmediate(ctx context.Context, db *sql.DB, fn func(DBTX) error) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	done := false
	defer func() {
		if !done {
			// Roll back even when ctx is cancelled or fn panicked.
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if err := fn(conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	done = true
	return nil
}

func (s *Store) Update(ctx context.Context, fn func(favorites.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return withImmediate(ctx, s.db, func(x DBTX) error { return fn(&tx{x: x}) })
}

func (s *Store) View(ctx context.Context, fn func(favorites.Tx) error) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	return fn(&tx{x: conn, ro: true})
}

type tx struct {
	x  DBTX
	ro bool
}

func (t *tx) Revision(ctx context.Context, network, owner string) (int64, error) {
	var r int64
	err := t.x.QueryRowContext(ctx, `SELECT revision FROM favorites_owner_state WHERE network_id=? AND owner_member_id=?`, network, owner).Scan(&r)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return r, err
}

func (t *tx) BumpRevision(ctx context.Context, network, owner string) (int64, error) {
	if t.ro {
		return 0, errReadOnly
	}
	var r int64
	err := t.x.QueryRowContext(ctx, `INSERT INTO favorites_owner_state(network_id, owner_member_id, revision) VALUES (?,?,1)
		ON CONFLICT(network_id, owner_member_id) DO UPDATE SET revision = revision + 1 RETURNING revision`, network, owner).Scan(&r)
	return r, err
}

const favCols = `network_id, owner_member_id, target_member_id, nickname, pinned, position, created_at, updated_at`

func scanFav(sc interface{ Scan(...any) error }) (favorites.Entry, error) {
	var e favorites.Entry
	var pinned int
	var c, u string
	if err := sc.Scan(&e.Network, &e.Owner, &e.Target, &e.Nickname, &pinned, &e.Order, &c, &u); err != nil {
		return e, err
	}
	e.Pinned = pinned == 1
	var err error
	if e.CreatedAt, err = parseTime(c); err != nil {
		return e, err
	}
	e.UpdatedAt, err = parseTime(u)
	return e, err
}

func (t *tx) ListFavorites(ctx context.Context, network, owner string, limit int) ([]favorites.Entry, error) {
	if limit <= 0 {
		limit = favorites.HardMaxFavorites + 1
	}
	rows, err := t.x.QueryContext(ctx, `SELECT `+favCols+` FROM member_favorites WHERE network_id=? AND owner_member_id=?
		ORDER BY pinned DESC, position ASC, target_member_id ASC LIMIT ?`, network, owner, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []favorites.Entry
	for rows.Next() {
		e, err := scanFav(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (t *tx) GetFavorite(ctx context.Context, network, owner, target string) (favorites.Entry, bool, error) {
	e, err := scanFav(t.x.QueryRowContext(ctx, `SELECT `+favCols+` FROM member_favorites WHERE network_id=? AND owner_member_id=? AND target_member_id=?`, network, owner, target))
	if errors.Is(err, sql.ErrNoRows) {
		return favorites.Entry{}, false, nil
	}
	return e, err == nil, err
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (t *tx) InsertFavorite(ctx context.Context, e favorites.Entry) error {
	if t.ro {
		return errReadOnly
	}
	res, err := t.x.ExecContext(ctx, `INSERT INTO member_favorites(`+favCols+`) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`,
		e.Network, e.Owner, e.Target, e.Nickname, b2i(e.Pinned), e.Order, fmtTime(e.CreatedAt), fmtTime(e.UpdatedAt))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return favorites.ErrDuplicate
	}
	return nil
}

func (t *tx) UpdateFavorite(ctx context.Context, e favorites.Entry) (bool, error) {
	if t.ro {
		return false, errReadOnly
	}
	res, err := t.x.ExecContext(ctx, `UPDATE member_favorites SET nickname=?, pinned=?, position=?, updated_at=?
		WHERE network_id=? AND owner_member_id=? AND target_member_id=?`,
		e.Nickname, b2i(e.Pinned), e.Order, fmtTime(e.UpdatedAt), e.Network, e.Owner, e.Target)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *tx) DeleteFavorite(ctx context.Context, network, owner, target string) (bool, error) {
	if t.ro {
		return false, errReadOnly
	}
	res, err := t.x.ExecContext(ctx, `DELETE FROM member_favorites WHERE network_id=? AND owner_member_id=? AND target_member_id=?`, network, owner, target)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *tx) GetIntent(ctx context.Context, network, owner, key string) (favorites.Intent, bool, error) {
	var in favorites.Intent
	var c, x string
	err := t.x.QueryRowContext(ctx, `SELECT network_id, owner_member_id, request_key, trade_id, target_member_id, description, intent, created_at, expires_at
		FROM favorite_repeat_intents WHERE network_id=? AND owner_member_id=? AND request_key=?`, network, owner, key).
		Scan(&in.Network, &in.Owner, &in.Key, &in.TradeID, &in.Target, &in.Description, &in.Intent, &c, &x)
	if errors.Is(err, sql.ErrNoRows) {
		return in, false, nil
	}
	if err != nil {
		return in, false, err
	}
	if in.CreatedAt, err = parseTime(c); err != nil {
		return in, false, err
	}
	in.ExpiresAt, err = parseTime(x)
	return in, err == nil, err
}

func (t *tx) InsertIntent(ctx context.Context, in favorites.Intent) error {
	if t.ro {
		return errReadOnly
	}
	res, err := t.x.ExecContext(ctx, `INSERT INTO favorite_repeat_intents(network_id, owner_member_id, request_key, trade_id, target_member_id, description, intent, created_at, expires_at)
		VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`,
		in.Network, in.Owner, in.Key, in.TradeID, in.Target, in.Description, in.Intent, fmtTime(in.CreatedAt), fmtTime(in.ExpiresAt))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return favorites.ErrDuplicate
	}
	return nil
}

func (t *tx) DeleteIntent(ctx context.Context, network, owner, key string) error {
	if t.ro {
		return errReadOnly
	}
	_, err := t.x.ExecContext(ctx, `DELETE FROM favorite_repeat_intents WHERE network_id=? AND owner_member_id=? AND request_key=?`, network, owner, key)
	return err
}

func (t *tx) CountIntents(ctx context.Context, network, owner string, now time.Time) (int, error) {
	var n int
	err := t.x.QueryRowContext(ctx, `SELECT COUNT(*) FROM favorite_repeat_intents WHERE network_id=? AND owner_member_id=? AND expires_at > ?`, network, owner, fmtTime(now)).Scan(&n)
	return n, err
}

func (t *tx) PurgeMember(ctx context.Context, network, member string) (favorites.PurgeResult, error) {
	if t.ro {
		return favorites.PurgeResult{}, errReadOnly
	}
	return PurgeMemberTx(ctx, t.x, network, member)
}

// PurgeMemberTx removes every row owned by or pointing at member in network,
// using x (for example the host's departure *sql.Tx). Owners who lose a
// favorite get a new list revision. It is atomic only as far as x is.
func PurgeMemberTx(ctx context.Context, x DBTX, network, member string) (favorites.PurgeResult, error) {
	var r favorites.PurgeResult
	if _, err := x.ExecContext(ctx, `INSERT INTO favorites_owner_state(network_id, owner_member_id, revision)
		SELECT DISTINCT network_id, owner_member_id, 1 FROM member_favorites WHERE network_id=? AND target_member_id=? AND owner_member_id<>?
		ON CONFLICT(network_id, owner_member_id) DO UPDATE SET revision = revision + 1`, network, member, member); err != nil {
		return r, err
	}
	run := func(q string, dst *int64) error {
		res, err := x.ExecContext(ctx, q, network, member, member)
		if err != nil {
			return err
		}
		*dst, _ = res.RowsAffected()
		return nil
	}
	if err := run(`DELETE FROM member_favorites WHERE network_id=? AND (owner_member_id=? OR target_member_id=?)`, &r.Favorites); err != nil {
		return r, err
	}
	if err := run(`DELETE FROM favorite_repeat_intents WHERE network_id=? AND (owner_member_id=? OR target_member_id=?)`, &r.Intents); err != nil {
		return r, err
	}
	res, err := x.ExecContext(ctx, `DELETE FROM favorites_owner_state WHERE network_id=? AND owner_member_id=?`, network, member)
	if err != nil {
		return r, err
	}
	r.States, _ = res.RowsAffected()
	return r, nil
}

func (t *tx) PurgeExpiredIntents(ctx context.Context, cutoff time.Time) (int64, error) {
	if t.ro {
		return 0, errReadOnly
	}
	res, err := t.x.ExecContext(ctx, `DELETE FROM favorite_repeat_intents WHERE expires_at <= ?`, fmtTime(cutoff))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
