package messaging

import (
	"context"
	"database/sql"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Search returns a sequence page of visible messages containing a literal,
// case-sensitive Unicode substring. Query text is never a SQL pattern or code.
// Search does not alter read positions, retention, or any message content.
func (s *Store) Search(ctx context.Context, a Actor, id, query string, after int64, limit int) ([]Message, error) {
	if len(query) > 128 || !utf8.ValidString(query) || strings.TrimSpace(query) == "" || after < 0 || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	for _, r := range query {
		if unicode.IsControl(r) {
			return nil, ErrInvalid
		}
	}
	out := []Message{}
	err := s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if _, err := s.thread(ctx, tx, a, id, false); err != nil {
			return err
		}
		var hidden int64
		if err := tx.QueryRowContext(ctx, `SELECT hidden FROM message_positions WHERE network=? AND thread=? AND member=?`, a.Network, id, a.Member).Scan(&hidden); err != nil {
			return err
		}
		if hidden > after {
			after = hidden
		}
		rows, err := tx.QueryContext(ctx, `SELECT id,sequence,sender,body,created FROM message_entries WHERE network=? AND thread=? AND sequence>? AND instr(body,?)>0 ORDER BY sequence LIMIT ?`, a.Network, id, after, query, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m Message
			if err := rows.Scan(&m.ID, &m.Sequence, &m.Sender, &m.Body, &m.Created); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}
