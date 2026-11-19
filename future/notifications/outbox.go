package notifications

import (
	"context"
	"database/sql"
	"time"
	"unicode/utf8"
)

// Event is the source event a host service commits. NetworkID, EventID and
// MemberID come from the host's own committed state, not from clients.
type Event struct {
	NetworkID string
	EventID   string // stable source identity, e.g. the proposal/transaction event id
	MemberID  string // recipient
	Kind      string // one of Kinds
	ExpiresAt time.Time
}

// Enqueued reports one outbox row for an event.
type Enqueued struct {
	DeliveryID string
	Channel    string
	Created    bool // false: the row already existed (duplicate event)
}

const (
	defaultTTL = 24 * time.Hour
	maxTTL     = 7 * 24 * time.Hour
)

func (e Event) validate() error {
	if !ValidKind(e.Kind) {
		return invalid("kind")
	}
	for _, v := range []string{e.NetworkID, e.MemberID} {
		if v == "" || len(v) > 128 || !utf8.ValidString(v) {
			return invalid("event ids")
		}
	}
	if e.EventID == "" || len(e.EventID) > 256 || !utf8.ValidString(e.EventID) {
		return invalid("event id")
	}
	return nil
}

// EnqueueTx records the outbox rows for ev inside the caller's transaction q,
// so rows commit with the source event and vanish if it rolls back. It creates
// a row only for each channel on which the recipient has opted in to ev.Kind
// and is currently active; repeated calls for the same
// (network,event,member,channel,kind) return the original rows. Neither this
// call nor any later delivery attempt writes to or retries ledger state.
func (s *Service) EnqueueTx(ctx context.Context, q DBTX, ev Event) ([]Enqueued, error) {
	if err := ev.validate(); err != nil {
		return nil, err
	}
	now := s.now()
	if ev.ExpiresAt.IsZero() {
		ev.ExpiresAt = now.Add(defaultTTL)
	}
	if !ev.ExpiresAt.After(now) {
		return nil, invalid("event already expired")
	}
	if max := now.Add(maxTTL); ev.ExpiresAt.After(max) {
		ev.ExpiresAt = max
	}
	active, err := s.cfg.Directory.MemberActive(ctx, ev.NetworkID, ev.MemberID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, nil
	}
	var out []Enqueued
	for _, ch := range Channels {
		var en int
		err := q.QueryRowContext(ctx, `SELECT enabled FROM notification_preferences WHERE network_id=? AND member_id=? AND event_type=? AND channel=?`,
			ev.NetworkID, ev.MemberID, ev.Kind, ch).Scan(&en)
		if err == sql.ErrNoRows || (err == nil && en != 1) {
			continue
		}
		if err != nil {
			return nil, err
		}
		r, err := s.insertDelivery(ctx, q, now, ev, ch, "")
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) insertDelivery(ctx context.Context, q DBTX, now time.Time, ev Event, channel, targetID string) (Enqueued, error) {
	key, err := DedupeKey(ev.NetworkID, ev.EventID, ev.MemberID, channel, ev.Kind)
	if err != nil {
		return Enqueued{}, invalid("event")
	}
	id := newID()
	var tgt any
	if targetID != "" {
		tgt = targetID
	}
	res, err := q.ExecContext(ctx, `INSERT INTO notification_delivery(id,network_id,member_id,event_id,dedupe_key,channel,kind,status,attempts,claims,next_attempt_at,expiry,target_id,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,'ready',0,0,?,?,?,?,?) ON CONFLICT(dedupe_key) DO NOTHING`,
		id, ev.NetworkID, ev.MemberID, ev.EventID, key, channel, ev.Kind, now.Unix(), ev.ExpiresAt.Unix(), tgt, now.Unix(), now.Unix())
	if err != nil {
		return Enqueued{}, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		s.met.Enqueued.Add(1)
		return Enqueued{DeliveryID: id, Channel: channel, Created: true}, nil
	}
	var existing string
	if err := q.QueryRowContext(ctx, `SELECT id FROM notification_delivery WHERE dedupe_key=?`, key).Scan(&existing); err != nil {
		return Enqueued{}, err
	}
	s.met.Duplicates.Add(1)
	return Enqueued{DeliveryID: existing, Channel: channel}, nil
}

// ---- inspection ----

// DeliveryCount is one row of the status matrix.
type DeliveryCount struct {
	Channel, Status string
	Count           int
}

// StatusReport is redacted operator metadata: counts and ages only. It never
// holds member ids, addresses, endpoints or keys.
type StatusReport struct {
	Deliveries            []DeliveryCount
	OldestReadyAgeSeconds int64
	ActivePushSubs        int
	VerifiedEmailTargets  int
	BouncedEmailTargets   int
	SchemaVersion         int
}

// Status summarises the outbox for the CLI and metrics.
func (s *Service) Status(ctx context.Context) (StatusReport, error) {
	var r StatusReport
	now := s.now().Unix()
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		rows, err := q.QueryContext(ctx, `SELECT channel,status,COUNT(*) FROM notification_delivery GROUP BY channel,status ORDER BY channel,status`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var d DeliveryCount
			if err := rows.Scan(&d.Channel, &d.Status, &d.Count); err != nil {
				rows.Close()
				return err
			}
			r.Deliveries = append(r.Deliveries, d)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		var oldest sql.NullInt64
		if err := q.QueryRowContext(ctx, `SELECT MIN(next_attempt_at) FROM notification_delivery WHERE status='ready' AND next_attempt_at<=?`, now).Scan(&oldest); err != nil {
			return err
		}
		if oldest.Valid {
			r.OldestReadyAgeSeconds = now - oldest.Int64
		}
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE revoked_at IS NULL`).Scan(&r.ActivePushSubs); err != nil {
			return err
		}
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_email_targets WHERE revoked_at IS NULL AND verified_at IS NOT NULL AND bounced_at IS NULL`).Scan(&r.VerifiedEmailTargets); err != nil {
			return err
		}
		return q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_email_targets WHERE revoked_at IS NULL AND bounced_at IS NOT NULL`).Scan(&r.BouncedEmailTargets)
	})
	r.SchemaVersion = SchemaVersion()
	return r, err
}

// DeliveryInfo is a redacted row for diagnostics: no member, network, event,
// address or endpoint values.
type DeliveryInfo struct {
	ID        string
	Channel   string
	Kind      string
	Status    string
	Attempts  int
	Claims    int
	NextAt    time.Time
	Expiry    time.Time
	LastError string
}

// ListDeliveries returns up to limit redacted rows, optionally by status.
func (s *Service) ListDeliveries(ctx context.Context, status string, limit int) ([]DeliveryInfo, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	var out []DeliveryInfo
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		query := `SELECT id,channel,kind,status,attempts,claims,next_attempt_at,expiry,COALESCE(last_error,'') FROM notification_delivery`
		args := []any{}
		if status != "" {
			query += ` WHERE status=?`
			args = append(args, status)
		}
		query += ` ORDER BY created_at DESC,id LIMIT ?`
		args = append(args, limit)
		rows, err := q.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d DeliveryInfo
			var n, e int64
			if err := rows.Scan(&d.ID, &d.Channel, &d.Kind, &d.Status, &d.Attempts, &d.Claims, &n, &e, &d.LastError); err != nil {
				return err
			}
			d.NextAt, d.Expiry = time.Unix(n, 0).UTC(), time.Unix(e, 0).UTC()
			out = append(out, d)
		}
		return rows.Err()
	})
	return out, err
}

// RetryDelivery re-queues a failed/expired/cancelled delivery for the operator
// (CLI). It resets attempts but not the expiry: an expired row stays expired.
func (s *Service) RetryDelivery(ctx context.Context, id string) error {
	if id == "" || len(id) > 64 {
		return ErrNotFound
	}
	return s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := s.now()
		res, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='ready',attempts=0,claims=0,next_attempt_at=?,last_error=NULL,updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE id=? AND status='failed' AND expiry>?`, now.Unix(), now.Unix(), id, now.Unix())
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		_, err = q.ExecContext(ctx, `DELETE FROM notification_delivery_target WHERE delivery_id=? AND status='failed'`, id)
		return err
	})
}

// Purge deletes terminal deliveries older than retention and revoked or
// bounced targets older than retention. Audit rows are append-only and kept.
// Returns the number of deliveries removed.
func (s *Service) Purge(ctx context.Context, retention time.Duration) (int, error) {
	if retention < time.Hour {
		return 0, invalid("retention too short")
	}
	cut := s.now().Add(-retention).Unix()
	var n int64
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM notification_delivery_target WHERE delivery_id IN (SELECT id FROM notification_delivery WHERE status IN ('sent','cancelled','expired','failed') AND updated_at<?)`, cut); err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, `DELETE FROM notification_delivery WHERE status IN ('sent','cancelled','expired','failed') AND updated_at<?`, cut)
		if err != nil {
			return err
		}
		n, _ = res.RowsAffected()
		if _, err := q.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE revoked_at IS NOT NULL AND revoked_at<?`, cut); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM notification_email_targets WHERE (revoked_at IS NOT NULL AND revoked_at<?) OR (bounced_at IS NOT NULL AND bounced_at<?)`, cut, cut); err != nil {
			return err
		}
		_, err = q.ExecContext(ctx, `DELETE FROM notification_idempotency WHERE created_at<?`, cut)
		return err
	})
	return int(n), err
}

// EnqueueTestTx queues an operator diagnostic message for one member's channel
// inside q. It is subject to every gate a real notification passes: the member
// must be active and have opted in to at least one kind on the channel, and a
// confirmed target must exist, so the diagnostic can never reach someone who
// did not opt in.
func (s *Service) EnqueueTestTx(ctx context.Context, q DBTX, networkID, memberID, channel, eventID string) (Enqueued, error) {
	if !ValidChannel(channel) || networkID == "" || memberID == "" || eventID == "" || len(eventID) > 256 {
		return Enqueued{}, invalid("test notification")
	}
	now := s.now()
	return s.insertDelivery(ctx, q, now, Event{NetworkID: networkID, EventID: "test:" + eventID, MemberID: memberID, Kind: KindTest, ExpiresAt: now.Add(time.Hour)}, channel, "")
}
