package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PrefRow is one (kind, channel) preference. The zero value is the default:
// disabled, UTC, no quiet hours. Nothing is ever delivered by default.
type PrefRow struct {
	EventType  string
	Channel    string
	Enabled    bool
	QuietStart *int // minutes after local midnight; nil with QuietEnd nil = none
	QuietEnd   *int
	TZ         string
	UpdatedAt  time.Time
}

// PrefUpdate replaces one preference row. QuietStart and QuietEnd must be both
// set or both nil; equal values mean no quiet hours and are stored as nil.
type PrefUpdate struct {
	EventType  string
	Channel    string
	Enabled    bool
	QuietStart *int
	QuietEnd   *int
	TZ         string // "" means UTC
}

const maxPrefUpdates = 10 // kinds x channels

func (u PrefUpdate) normalise() (PrefUpdate, error) {
	if !ValidKind(u.EventType) || !ValidChannel(u.Channel) {
		return u, invalid("unknown notification kind or channel")
	}
	if u.TZ == "" {
		u.TZ = "UTC"
	}
	if len(u.TZ) > 64 || u.TZ == "Local" || strings.ContainsAny(u.TZ, "\x00\r\n\\ ") || strings.Contains(u.TZ, "..") {
		return u, invalid("time zone")
	}
	if _, err := time.LoadLocation(u.TZ); err != nil {
		return u, invalid("time zone")
	}
	if (u.QuietStart == nil) != (u.QuietEnd == nil) {
		return u, invalid("quiet hours need both start and end")
	}
	if u.QuietStart != nil {
		if *u.QuietStart < 0 || *u.QuietStart > 1439 || *u.QuietEnd < 0 || *u.QuietEnd > 1439 {
			return u, invalid("quiet hours out of range")
		}
		if *u.QuietStart == *u.QuietEnd {
			u.QuietStart, u.QuietEnd = nil, nil
		}
	}
	return u, nil
}

func fmtQuiet(s, e *int) string {
	if s == nil {
		return "none"
	}
	return fmt.Sprintf("%02d:%02d-%02d:%02d", *s/60, *s%60, *e/60, *e%60)
}

// ListPreferences returns the full kind x channel matrix for the actor, filling
// defaults for rows never saved.
func (s *Service) ListPreferences(ctx context.Context, a Actor) ([]PrefRow, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	have := map[[2]string]PrefRow{}
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		rows, err := q.QueryContext(ctx, `SELECT event_type,channel,enabled,quiet_start,quiet_end,tz,updated_at FROM notification_preferences WHERE network_id=? AND member_id=?`, a.NetworkID, a.MemberID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r PrefRow
			var en int
			var qs, qe sql.NullInt64
			var up int64
			if err := rows.Scan(&r.EventType, &r.Channel, &en, &qs, &qe, &r.TZ, &up); err != nil {
				return err
			}
			r.Enabled = en == 1
			if qs.Valid {
				a, b := int(qs.Int64), int(qe.Int64)
				r.QuietStart, r.QuietEnd = &a, &b
			}
			r.UpdatedAt = time.Unix(up, 0).UTC()
			have[[2]string{r.EventType, r.Channel}] = r
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	out := make([]PrefRow, 0, len(Kinds)*len(Channels))
	for _, k := range Kinds {
		for _, c := range Channels {
			if r, ok := have[[2]string{k, c}]; ok {
				out = append(out, r)
			} else {
				out = append(out, PrefRow{EventType: k, Channel: c, TZ: "UTC"})
			}
		}
	}
	return out, nil
}

// SetPreferences applies updates atomically for the actor. Disabling a row
// cancels that member's not-yet-claimed deliveries for it immediately. Inactive
// members cannot change preferences.
func (s *Service) SetPreferences(ctx context.Context, a Actor, ups []PrefUpdate) error {
	if len(ups) == 0 || len(ups) > maxPrefUpdates {
		return invalid("update count")
	}
	norm := make([]PrefUpdate, 0, len(ups))
	seen := map[[2]string]bool{}
	for _, u := range ups {
		n, err := u.normalise()
		if err != nil {
			return err
		}
		k := [2]string{n.EventType, n.Channel}
		if seen[k] {
			return invalid("duplicate preference row")
		}
		seen[k] = true
		norm = append(norm, n)
	}
	if err := s.requireActive(ctx, a); err != nil {
		return err
	}
	var ab auditBatch
	now := s.now()
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		for _, u := range norm {
			var old PrefRow
			var en int
			var qs, qe sql.NullInt64
			err := q.QueryRowContext(ctx, `SELECT enabled,quiet_start,quiet_end,tz FROM notification_preferences WHERE network_id=? AND member_id=? AND event_type=? AND channel=?`,
				a.NetworkID, a.MemberID, u.EventType, u.Channel).Scan(&en, &qs, &qe, &old.TZ)
			exists := err == nil
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			var nqs, nqe any
			if u.QuietStart != nil {
				nqs, nqe = *u.QuietStart, *u.QuietEnd
			}
			enabled := 0
			if u.Enabled {
				enabled = 1
			}
			if _, err := q.ExecContext(ctx, `INSERT INTO notification_preferences(network_id,member_id,event_type,channel,enabled,quiet_start,quiet_end,tz,updated_at)
				VALUES(?,?,?,?,?,?,?,?,?)
				ON CONFLICT(network_id,member_id,event_type,channel) DO UPDATE SET enabled=excluded.enabled,quiet_start=excluded.quiet_start,quiet_end=excluded.quiet_end,tz=excluded.tz,updated_at=excluded.updated_at`,
				a.NetworkID, a.MemberID, u.EventType, u.Channel, enabled, nqs, nqe, u.TZ, now.Unix()); err != nil {
				return err
			}
			changed := !exists || (en == 1) != u.Enabled || old.TZ != u.TZ || qs.Valid != (u.QuietStart != nil) ||
				(u.QuietStart != nil && (int(qs.Int64) != *u.QuietStart || int(qe.Int64) != *u.QuietEnd))
			if changed {
				if err := s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, actorLabel(a), "preference.set",
					u.EventType+"/"+u.Channel, fmt.Sprintf("enabled=%t quiet=%s tz=%s", u.Enabled, fmtQuiet(u.QuietStart, u.QuietEnd), u.TZ)); err != nil {
					return err
				}
			}
			if !u.Enabled {
				if err := cancelReady(ctx, q, now, a.NetworkID, a.MemberID, u.Channel, u.EventType, "opt_out"); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

// cancelReady cancels ready (not in-flight) deliveries. kind "" matches every
// non-verification kind.
func cancelReady(ctx context.Context, q DBTX, now time.Time, network, member, channel, kind, reason string) error {
	if kind == "" {
		_, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='cancelled',last_error=?,updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE network_id=? AND member_id=? AND channel=? AND status='ready' AND kind<>?`, reason, now.Unix(), network, member, channel, KindEmailVerification)
		return err
	}
	_, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='cancelled',last_error=?,updated_at=?,lease_owner=NULL,lease_until=NULL
		WHERE network_id=? AND member_id=? AND channel=? AND kind=? AND status='ready'`, reason, now.Unix(), network, member, channel, kind)
	return err
}

// ListAudit returns the member's own preference/target audit trail, newest
// first.
func (s *Service) ListAudit(ctx context.Context, a Actor, limit int) ([]AuditRecord, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []AuditRecord
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		rows, err := q.QueryContext(ctx, `SELECT actor,action,subject,detail,at FROM notification_audit WHERE network_id=? AND member_id=? ORDER BY id DESC LIMIT ?`, a.NetworkID, a.MemberID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			r := AuditRecord{NetworkID: a.NetworkID, MemberID: a.MemberID}
			var at int64
			if err := rows.Scan(&r.Actor, &r.Action, &r.Subject, &r.Detail, &at); err != nil {
				return err
			}
			r.At = time.Unix(at, 0).UTC()
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}
