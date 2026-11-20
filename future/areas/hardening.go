package areas

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode"
)

func execOne(ctx context.Context, tx *sql.Tx, q string, args ...any) (sql.Result, error) {
	r, e := tx.ExecContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return nil, e
	}
	if n != 1 {
		return nil, errors.New("safety write was not persisted")
	}
	return r, nil
}
func normalizeSQL(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.ReplaceAll(s, "IF NOT EXISTS ", ""))
}
func validateSchema(ctx context.Context, tx *sql.Tx, schema []string) error {
	for _, q := range schema {
		fields := strings.Fields(q)
		kind := strings.ToLower(fields[1])
		name := fields[5]
		if i := strings.IndexByte(name, '('); i >= 0 {
			name = name[:i]
		}
		var typ, actual string
		e := tx.QueryRowContext(ctx, `SELECT type,sql FROM sqlite_schema WHERE name=?`, name).Scan(&typ, &actual)
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			return e
		}
		if typ != kind || normalizeSQL(actual) != normalizeSQL(q) {
			return errors.New("incompatible areas schema object: " + name)
		}
	}
	return nil
}
func validateReplay(a Identity, object string, payload, out any) error {
	switch v := out.(type) {
	case *Record:
		input, ok := payload.(Edit)
		if !ok || v.Network != a.Network || v.ID != object || v.Label != strings.TrimSpace(input.Label) || v.Visible != input.Visible || v.Archived != input.Archived || v.Revision != input.Revision+1 || Validate(v.Area) != nil {
			return errors.New("invalid stored area reply")
		}
	case *Assignment:
		input, ok := payload.(Assign)
		if !ok || v.Network != a.Network || v.Listing != object || v.AreaID != input.AreaID || v.Active != (v.AreaID != "") || v.Revision != input.Revision+1 {
			return errors.New("invalid stored assignment reply")
		}
	case *Sharing:
		input, ok := payload.(struct {
			Enabled  bool
			Revision int64
		})
		if !ok || v.Revision != input.Revision+1 || v.Enabled != input.Enabled {
			return errors.New("invalid stored sharing reply")
		}
	}
	return nil
}

func requireForeignKeys(ctx context.Context, tx *sql.Tx) error {
	var enabled int
	if e := tx.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&enabled); e != nil {
		return e
	}
	if enabled != 1 {
		return errors.New("foreign keys must be enabled on every host connection")
	}
	return nil
}
