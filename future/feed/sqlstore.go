package feed

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"unicode/utf8"
)

// SQLStore is an optional, explicitly initialized SQLite source-reference store.
// It stores no listing or announcement content. Callers own the database and its
// lifetime. No constructor opens a database, initializes schema, or registers the
// module in Core. A separately reviewed integration must project only committed
// source changes and recheck current source authorization before showing rows.
//
// Source creation time is immutable. Deletion preserves a tombstone, preventing
// delayed event deliveries from resurrecting a deleted source. Sequence is an
// insertion watermark, not an authorization or content revision.
type SQLStore struct{ db *sql.DB }

var _ Store = (*SQLStore)(nil)

// NewSQLStore wraps an existing database without querying or changing it.
func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, ErrInvalidQuery
	}
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) valid(ctx context.Context) bool { return s != nil && s.db != nil && ctx != nil }

var feedSchema = []string{
	`CREATE TABLE IF NOT EXISTS fnet204_sources (
		sequence INTEGER PRIMARY KEY AUTOINCREMENT,
		network TEXT NOT NULL CHECK(length(CAST(network AS BLOB)) BETWEEN 1 AND 256),
		kind TEXT NOT NULL CHECK(kind IN ('announcement','listing')),
		source_id TEXT NOT NULL CHECK(length(CAST(source_id AS BLOB)) BETWEEN 1 AND 128),
		member TEXT NOT NULL CHECK(length(CAST(member AS BLOB)) <= 256),
		order_time INTEGER NOT NULL CHECK(order_time BETWEEN -9223372036854775807 AND -1),
		visible INTEGER NOT NULL CHECK(visible IN (0,1)),
		active INTEGER NOT NULL CHECK(active IN (0,1)),
		deleted INTEGER NOT NULL DEFAULT 0 CHECK(deleted IN (0,1)),
		UNIQUE(network, kind, source_id)
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS fnet204_sources_network_sequence ON fnet204_sources(network, sequence DESC)`,
	`CREATE INDEX IF NOT EXISTS fnet204_sources_all_order ON fnet204_sources(network, order_time, kind, source_id)
		WHERE visible = 1 AND active = 1 AND deleted = 0`,
	`CREATE INDEX IF NOT EXISTS fnet204_sources_kind_order ON fnet204_sources(network, kind, order_time, source_id)
		WHERE visible = 1 AND active = 1 AND deleted = 0`,
}

// InitSchema explicitly creates only this module's tables and indexes. It is an
// isolated store setup operation, not a Core migration. Repeated calls are safe.
func (s *SQLStore) InitSchema(ctx context.Context) error {
	if !s.valid(ctx) {
		return ErrInvalidQuery
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return hideStoreError(err)
	}
	defer tx.Rollback()
	for _, statement := range feedSchema {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return hideStoreError(err)
		}
	}
	return hideStoreError(tx.Commit())
}

func validStoreText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && utf8.ValidString(value)
}

func validStoreSource(source Source) bool {
	return validStoreText(source.Network, maxBindingLen) && validStoreText(source.Member, maxBindingLen) && validPosition(positionOf(source))
}

// Upsert records a trusted current source snapshot. Repeating the same event
// preserves its sequence; visibility, membership ownership and active status
// may change. A creation-time change or previously deleted source fails closed.
func (s *SQLStore) Upsert(ctx context.Context, source Source) error {
	if !s.valid(ctx) {
		return ErrInvalidQuery
	}
	if !validStoreSource(source) {
		return ErrInvalidSource
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO fnet204_sources
		(network, kind, source_id, member, order_time, visible, active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(network, kind, source_id) DO UPDATE SET
			member = excluded.member, visible = excluded.visible, active = excluded.active
		WHERE fnet204_sources.deleted = 0 AND fnet204_sources.order_time = excluded.order_time`,
		source.Network, source.Kind, source.ID, source.Member, -source.Created.UnixNano(), source.Visible, source.Active)
	if err != nil {
		return hideStoreError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return hideStoreError(err)
	}
	if count != 1 {
		return ErrInvalidSource
	}
	return nil
}

// Delete hides a source permanently within this store. An unknown source gets
// a tombstone as well, so an older delayed create event cannot recreate it.
func (s *SQLStore) Delete(ctx context.Context, network, kind, id string) error {
	if !s.valid(ctx) {
		return ErrInvalidQuery
	}
	if !validStoreText(network, maxBindingLen) || !isSourceKind(kind) || !validStoreText(id, maxSourceIDLen) {
		return ErrInvalidSource
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO fnet204_sources
		(network, kind, source_id, member, order_time, visible, active, deleted)
		VALUES (?, ?, ?, '', -1, 0, 0, 1)
		ON CONFLICT(network, kind, source_id) DO UPDATE SET
			visible = 0, active = 0, deleted = 1`, network, kind, id)
	return hideStoreError(err)
}

// HighWater returns the largest committed insertion sequence in one network.
// The dedicated index answers this with one seek, even with a large history.
func (s *SQLStore) HighWater(ctx context.Context, network string) (int64, error) {
	if !s.valid(ctx) || !validStoreText(network, maxBindingLen) {
		return 0, ErrInvalidQuery
	}
	var sequence int64
	err := s.db.QueryRowContext(ctx, `SELECT sequence FROM fnet204_sources
		INDEXED BY fnet204_sources_network_sequence WHERE network = ? ORDER BY sequence DESC LIMIT 1`, network).Scan(&sequence)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, hideStoreError(err)
	}
	if sequence <= 0 {
		return 0, ErrInvalidSource
	}
	return sequence, nil
}

// QueryBatch returns at most Limit currently visible, active source references
// after a keyset position. Limit is 1..101. It does not apply a sequence watermark
// or viewer authorization: the paging layer applies both to these bounded
// candidates. Filtering them before SQL LIMIT would permit unbounded scanning
// after backdated inserts or under sparse viewer permissions.
//
// Negated nanoseconds make every order component ascending. Row-value seeks
// then use the complete index, including huge groups with equal timestamps.
// A category-specific index avoids scanning unrelated categories.
func (s *SQLStore) QueryBatch(ctx context.Context, request StoreQuery) ([]StoredSource, error) {
	if !s.valid(ctx) || !validStoreText(request.Network, maxBindingLen) || request.Limit < 1 || request.Limit > maxPageLimit+1 {
		return nil, ErrInvalidQuery
	}
	kinds, _, err := (Filter{Kinds: request.Kinds}).canonical()
	if err != nil || (request.After != nil && (!validPosition(*request.After) || !hasKind(kinds, request.After.Kind))) {
		return nil, ErrInvalidQuery
	}
	statement, args := storeBatchQuery(request, kinds)
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, hideStoreError(err)
	}
	defer rows.Close()
	result := make([]StoredSource, 0, request.Limit)
	previous := request.After
	for rows.Next() {
		if len(result) >= request.Limit {
			return nil, ErrInvalidSource
		}
		var item StoredSource
		var orderTime int64
		if err := rows.Scan(&item.Sequence, &item.Source.Network, &item.Source.Kind, &item.Source.ID, &item.Source.Member, &orderTime, &item.Source.Visible, &item.Source.Active); err != nil {
			return nil, hideStoreError(err)
		}
		item.Source.Created = time.Unix(0, -orderTime).UTC()
		position := positionOf(item.Source)
		if item.Sequence <= 0 || !validStoreSource(item.Source) || item.Source.Network != request.Network || !hasKind(kinds, item.Source.Kind) || !item.Source.Visible || !item.Source.Active || (previous != nil && !precedes(*previous, position)) {
			return nil, ErrInvalidSource
		}
		result = append(result, item)
		previous = &position
	}
	if err := rows.Err(); err != nil {
		return nil, hideStoreError(err)
	}
	return result, nil
}

func storeBatchQuery(request StoreQuery, kinds []string) (string, []any) {
	selectColumns := `SELECT sequence, network, kind, source_id, member, order_time, visible, active FROM fnet204_sources `
	args := []any{request.Network}
	var statement string
	if len(kinds) == 1 {
		statement = selectColumns + `INDEXED BY fnet204_sources_kind_order WHERE network = ? AND kind = ? AND visible = 1 AND active = 1 AND deleted = 0`
		args = append(args, kinds[0])
		if request.After != nil {
			statement += ` AND (order_time, source_id) > (?, ?)`
			args = append(args, -request.After.Created.UnixNano(), request.After.ID)
		}
		statement += ` ORDER BY order_time, source_id LIMIT ?`
	} else {
		statement = selectColumns + `INDEXED BY fnet204_sources_all_order WHERE network = ? AND visible = 1 AND active = 1 AND deleted = 0`
		if request.After != nil {
			statement += ` AND (order_time, kind, source_id) > (?, ?, ?)`
			args = append(args, -request.After.Created.UnixNano(), request.After.Kind, request.After.ID)
		}
		statement += ` ORDER BY order_time, kind, source_id LIMIT ?`
	}
	return statement, append(args, request.Limit)
}

func hideStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return ErrStore
}
