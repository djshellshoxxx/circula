package recurring

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Safety writes must actually persist. A trigger's RAISE(IGNORE), a missing lock
// row or an unexpected schema must not acknowledge a successful operation.
func execOne(ctx context.Context, tx *sql.Tx, q string, args ...any) error {
	res, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrStorage
	}
	return nil
}
func insertFiring(ctx context.Context, tx *sql.Tx, network, id string, occurrence Occurrence, state string) (bool, error) {
	res, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO future_template_firings(network,template,window,scheduled,state) VALUES(?,?,?,?,?)`, network, id, occurrence.Key, occurrence.Scheduled.UTC().Format(time.RFC3339Nano), state)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 1 {
		return true, nil
	}
	if n != 0 {
		return false, ErrStorage
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM future_template_firings WHERE network=? AND template=? AND window=?)`, network, id, occurrence.Key).Scan(&exists); err != nil {
		return false, err
	}
	if !exists {
		return false, ErrStorage
	}
	return false, nil
}
func canonicalSchema(q string) string {
	return strings.ReplaceAll(strings.ToLower(strings.Join(strings.Fields(q), " ")), " if not exists ", " ")
}
func checkSchema(ctx context.Context, tx *sql.Tx) error {
	for _, expected := range schemaStatements {
		normalized := canonicalSchema(expected)
		fields := strings.Fields(normalized)
		if len(fields) < 3 || fields[0] != "create" {
			continue
		}
		name, _, _ := strings.Cut(fields[2], "(")
		var actual string
		if err := tx.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type=? AND name=?`, fields[1], name).Scan(&actual); err != nil {
			return err
		}
		if canonicalSchema(actual) != normalized {
			return ErrStorage
		}
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_recurring_lock WHERE id=1`).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return ErrStorage
	}
	return nil
}

// Bulk retirement is permitted to match zero records, but must not acknowledge
// rows silently left intact by a failed/ignored mutation. Predicates are static
// SQL supplied only by the maintenance methods in this module.
func retireResponses(ctx context.Context, tx *sql.Tx, predicate string, args ...any) error {
	where := "response<>'' AND (" + predicate + ")"
	if _, err := tx.ExecContext(ctx, `UPDATE future_template_operations SET response='' WHERE `+where, args...); err != nil {
		return err
	}
	var remaining bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM future_template_operations WHERE `+where+`)`, args...).Scan(&remaining); err != nil {
		return err
	}
	if remaining {
		return ErrStorage
	}
	return nil
}
