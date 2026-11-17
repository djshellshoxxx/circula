package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

const (
	idemTTL      = 24 * time.Hour
	idemInflight = 5 * time.Minute
)

// IdemBegin claims or replays an idempotency key scoped to network+actor.
func (s *Store) IdemBegin(ctx context.Context, network, actor, key, reqHash string, now time.Time) (d.IdemResult, error) {
	var res d.IdemResult
	err := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		var hash, state, ct string
		var status int
		var body []byte
		var created int64
		err := tx.QueryRowContext(ctx, `SELECT req_hash, state, status, content_type, body, created_at FROM discovery_idempotency WHERE network_id=? AND actor=? AND idem_key=?`,
			network, actor, key).Scan(&hash, &state, &status, &ct, &body, &created)
		switch {
		case err == nil:
			age := now.Sub(time.Unix(created, 0))
			expired := age > idemTTL || (state == "inflight" && age > idemInflight)
			if !expired {
				switch {
				case hash != reqHash:
					res.State = "mismatch"
				case state == "inflight":
					res.State = "inflight"
				default:
					res = d.IdemResult{State: "replay", Status: status, ContentType: ct, Body: body}
				}
				return nil
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM discovery_idempotency WHERE network_id=? AND actor=? AND idem_key=?`, network, actor, key); err != nil {
				return err
			}
		case errors.Is(err, sql.ErrNoRows):
		default:
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO discovery_idempotency (network_id, actor, idem_key, req_hash, state, created_at) VALUES (?,?,?,?, 'inflight', ?)`,
			network, actor, key, reqHash, now.Unix()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM discovery_idempotency WHERE created_at < ?`, now.Add(-idemTTL).Unix()); err != nil {
			return err
		}
		res.State = "new"
		return nil
	})
	return res, err
}

// IdemFinish stores the response for replays.
func (s *Store) IdemFinish(ctx context.Context, network, actor, key string, status int, contentType string, body []byte) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		_, err := tx.ExecContext(ctx, `UPDATE discovery_idempotency SET state='done', status=?, content_type=?, body=? WHERE network_id=? AND actor=? AND idem_key=?`,
			status, contentType, body, network, actor, key)
		return err
	})
}

// IdemAbort releases an in-flight key after a failed attempt so it can be retried.
func (s *Store) IdemAbort(ctx context.Context, network, actor, key string) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM discovery_idempotency WHERE network_id=? AND actor=? AND idem_key=? AND state='inflight'`, network, actor, key)
		return err
	})
}
