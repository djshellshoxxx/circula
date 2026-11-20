package onboarding

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// SQLStore is the SQLite Store adapter over database/sql. The caller supplies
// the *sql.DB (and therefore the driver); this package imports no driver in
// normal builds. SQL uses "?" placeholders and SQLite upserts.
//
// Every write is a single transaction whose first statement is a write, so
// SQLite acquires the write lock immediately and never needs a read-to-write
// upgrade. Compare-and-set is expressed in the WHERE clause, so concurrent
// updaters lose cleanly with ErrConflict instead of overwriting.
type SQLStore struct{ db *sql.DB }

// NewSQLStore wraps db. It does not migrate; call Migrate explicitly.
func NewSQLStore(db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, errors.New("onboarding: nil db")
	}
	return &SQLStore{db}, nil
}

// Migrate applies SchemaSQL idempotently and records the version. Existing
// unrelated tables and rows are never touched.
func Migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, SchemaSQL); err != nil {
		return fmt.Errorf("onboarding migrate: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO onboarding_schema_migrations(version, applied_at) VALUES (1, ?)`, ts(time.Now())); err != nil {
		return err
	}
	return tx.Commit()
}

// Rollback removes the module's tables (see RollbackSQL).
func Rollback(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, RollbackSQL); err != nil {
		return err
	}
	return tx.Commit()
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTS(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: bad timestamp", ErrInvalidState)
	}
	return t, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *SQLStore) LoadProgress(ctx context.Context, k Key) (Progress, bool, error) {
	if err := k.validate(); err != nil {
		return Progress{}, false, err
	}
	var (
		p               Progress
		done, skip, dis int64
		share           int64
		updated         string
	)
	err := s.db.QueryRowContext(ctx, `SELECT step_version, completed_mask, skipped_mask, dismissed, share_aggregate, revision, updated_at
		FROM member_onboarding_progress WHERE network_id = ? AND member_id = ?
		ORDER BY step_version DESC LIMIT 1`, k.NetworkID, k.MemberID).
		Scan(&p.Version, &done, &skip, &dis, &share, &p.Revision, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Progress{}, false, nil
	}
	if err != nil {
		return Progress{}, false, err
	}
	if done < 0 || done > 255 || skip < 0 || skip > 255 || dis > 1 || share > 1 {
		return Progress{}, false, fmt.Errorf("%w: column out of range", ErrInvalidState)
	}
	p.Completed, p.Skipped, p.Dismissed, p.Share = uint8(done), uint8(skip), dis == 1, share == 1
	if p.UpdatedAt, err = parseTS(updated); err != nil {
		return Progress{}, false, err
	}
	if err := p.Validate(); err != nil {
		return Progress{}, false, err
	}
	return p, true, nil
}

func (s *SQLStore) SaveProgress(ctx context.Context, k Key, expected int64, next Progress, newly []Step, now time.Time) (Progress, error) {
	if err := k.validate(); err != nil {
		return Progress{}, err
	}
	if err := next.Validate(); err != nil {
		return Progress{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Progress{}, err
	}
	defer tx.Rollback()
	var res sql.Result
	if expected == 0 {
		res, err = tx.ExecContext(ctx, `INSERT INTO member_onboarding_progress
			(network_id, member_id, step_version, completed_mask, skipped_mask, dismissed, share_aggregate, revision, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?) ON CONFLICT DO NOTHING`,
			k.NetworkID, k.MemberID, next.Version, next.Completed, next.Skipped, b2i(next.Dismissed), b2i(next.Share), ts(now))
	} else {
		res, err = tx.ExecContext(ctx, `UPDATE member_onboarding_progress
			SET completed_mask = ?, skipped_mask = ?, dismissed = ?, share_aggregate = ?, revision = revision + 1, updated_at = ?
			WHERE network_id = ? AND member_id = ? AND step_version = ? AND revision = ?`,
			next.Completed, next.Skipped, b2i(next.Dismissed), b2i(next.Share), ts(now),
			k.NetworkID, k.MemberID, next.Version, expected)
	}
	if err != nil {
		return Progress{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Progress{}, err
	} else if n != 1 {
		return Progress{}, ErrConflict
	}
	if next.Share {
		for _, st := range newly {
			if st >= StepCount {
				continue
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO network_onboarding_counts (network_id, step_version, step, completions)
				SELECT ?, ?, ?, 1 WHERE EXISTS (SELECT 1 FROM network_onboarding_metrics WHERE network_id = ? AND enabled = 1)
				ON CONFLICT (network_id, step_version, step) DO UPDATE SET completions = completions + 1`,
				k.NetworkID, next.Version, int(st), k.NetworkID); err != nil {
				return Progress{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Progress{}, err
	}
	next.Revision = expected + 1
	next.UpdatedAt = now.UTC()
	return next, nil
}

func (s *SQLStore) DeleteMember(ctx context.Context, k Key) error {
	if err := k.validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM member_onboarding_progress WHERE network_id = ? AND member_id = ?`, k.NetworkID, k.MemberID)
	return err
}

func (s *SQLStore) PurgeStale(ctx context.Context, before time.Time) (int64, error) {
	cutoff := before.UTC()
	// ts writes canonical UTC RFC3339Nano, whose trimmed fraction does not
	// sort chronologically as TEXT. Pad to nine digits for this comparison;
	// do not round through SQLite's lower-precision date/time functions.
	// The raw bound includes the entire cutoff second and retains an indexed
	// range over updated_at. Both canonical suffixes ('.' and 'Z') precede '~'.
	res, err := s.db.ExecContext(ctx, `DELETE FROM member_onboarding_progress
		WHERE updated_at < ? AND (
			substr(updated_at, 1, 19) || '.' ||
			substr(CASE WHEN substr(updated_at, 20, 1) = '.'
				THEN substr(updated_at, 21, length(updated_at) - 21)
				ELSE '' END || '000000000', 1, 9) || 'Z'
		) < ?`, cutoff.Format("2006-01-02T15:04:05")+"~", cutoff.Format("2006-01-02T15:04:05.000000000Z"))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLStore) LoadCopy(ctx context.Context, network string, version int) (Copy, bool, error) {
	if !validID(network) {
		return Copy{}, false, ErrInvalidIdentifier
	}
	c := Copy{NetworkID: network, Version: version}
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT intro_text, governance_url, terms_url, revision, updated_by, updated_at
		FROM network_onboarding_copy WHERE network_id = ? AND step_version = ?`, network, version).
		Scan(&c.Intro, &c.GovernanceURL, &c.TermsURL, &c.Revision, &c.UpdatedBy, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Copy{}, false, nil
	}
	if err != nil {
		return Copy{}, false, err
	}
	if c.UpdatedAt, err = parseTS(updated); err != nil {
		return Copy{}, false, err
	}
	return c, true, nil
}

func (s *SQLStore) SaveCopy(ctx context.Context, c Copy, expected int64, now time.Time) (Copy, error) {
	if !validID(c.NetworkID) || !validID(c.UpdatedBy) {
		return Copy{}, ErrInvalidIdentifier
	}
	var (
		res sql.Result
		err error
	)
	if expected == 0 {
		res, err = s.db.ExecContext(ctx, `INSERT INTO network_onboarding_copy
			(network_id, step_version, intro_text, governance_url, terms_url, revision, updated_by, updated_at)
			VALUES (?, ?, ?, ?, ?, 1, ?, ?) ON CONFLICT DO NOTHING`,
			c.NetworkID, c.Version, c.Intro, c.GovernanceURL, c.TermsURL, c.UpdatedBy, ts(now))
	} else {
		res, err = s.db.ExecContext(ctx, `UPDATE network_onboarding_copy
			SET intro_text = ?, governance_url = ?, terms_url = ?, revision = revision + 1, updated_by = ?, updated_at = ?
			WHERE network_id = ? AND step_version = ? AND revision = ?`,
			c.Intro, c.GovernanceURL, c.TermsURL, c.UpdatedBy, ts(now), c.NetworkID, c.Version, expected)
	}
	if err != nil {
		return Copy{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return Copy{}, err
	} else if n != 1 {
		return Copy{}, ErrConflict
	}
	c.Revision, c.UpdatedAt = expected+1, now.UTC()
	return c, nil
}

func (s *SQLStore) SetMetricsEnabled(ctx context.Context, network string, enabled bool, by string, now time.Time) error {
	if !validID(network) || !validID(by) {
		return ErrInvalidIdentifier
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO network_onboarding_metrics (network_id, enabled, updated_by, updated_at)
		VALUES (?, ?, ?, ?) ON CONFLICT (network_id) DO UPDATE SET enabled = excluded.enabled, updated_by = excluded.updated_by, updated_at = excluded.updated_at`,
		network, b2i(enabled), by, ts(now)); err != nil {
		return err
	}
	if !enabled {
		if _, err := tx.ExecContext(ctx, `DELETE FROM network_onboarding_counts WHERE network_id = ?`, network); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) MetricsEnabled(ctx context.Context, network string) (bool, error) {
	if !validID(network) {
		return false, ErrInvalidIdentifier
	}
	var e int
	err := s.db.QueryRowContext(ctx, `SELECT enabled FROM network_onboarding_metrics WHERE network_id = ?`, network).Scan(&e)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return e == 1, err
}

func (s *SQLStore) MetricCounts(ctx context.Context, network string, version int) ([StepCount]int64, error) {
	var out [StepCount]int64
	if !validID(network) {
		return out, ErrInvalidIdentifier
	}
	rows, err := s.db.QueryContext(ctx, `SELECT step, completions FROM network_onboarding_counts WHERE network_id = ? AND step_version = ?`, network, version)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var st int
		var n int64
		if err := rows.Scan(&st, &n); err != nil {
			return out, err
		}
		if st >= 0 && st < int(StepCount) {
			out[st] = n
		}
	}
	return out, rows.Err()
}
