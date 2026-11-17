package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

// KindTest is an operator diagnostic: it is delivered only to a member who has
// at least one enabled preference on the channel, through the same gates.
const KindTest = "test"

// Hooks are test seams for crash/race injection. Production leaves them nil.
type Hooks struct {
	AfterClaim     func(ctx context.Context, deliveryID string)
	AfterAuthorize func(ctx context.Context, deliveryID string)
	// AfterSend runs after the provider accepted or rejected a message but
	// before the outcome is recorded; a non-nil return aborts processing as a
	// process crash would (no outcome recorded, lease left to expire).
	AfterSend func(ctx context.Context, deliveryID, targetID string) error
	// BeforeFinalize may abort before the row is acknowledged.
	BeforeFinalize func(ctx context.Context, deliveryID string) error
}

// DispatcherOptions configures a worker. Channels with a nil sender are never
// claimed: rows for an unconfigured channel wait and then expire, and core
// operation is unaffected.
type DispatcherOptions struct {
	Owner     string // unique per worker process; default random
	Email     EmailSender
	Push      PushSender
	BatchSize int           // default 20
	Lease     time.Duration // default 2m; must exceed the longest send
	MaxClaims int           // default 8: abandoned-claim cap (poison rows)
	Hooks     Hooks
}

// Dispatcher claims due outbox rows and sends them. It never touches ledger
// state: a failure only changes the delivery row.
type Dispatcher struct {
	s    *Service
	o    DispatcherOptions
	chs  []string
	host string
}

// RunStats summarises one pass.
type RunStats struct {
	Claimed, Sent, Retried, Failed, Cancelled, Expired, Deferred, LeaseLost int
}

// NewDispatcher validates options.
func (s *Service) NewDispatcher(o DispatcherOptions) (*Dispatcher, error) {
	if o.Owner == "" {
		o.Owner = newID()
	}
	if o.BatchSize <= 0 {
		o.BatchSize = 20
	}
	if o.Lease <= 0 {
		o.Lease = 2 * time.Minute
	}
	if o.MaxClaims <= 0 {
		o.MaxClaims = 8
	}
	d := &Dispatcher{s: s, o: o}
	if o.Email != nil {
		if s.cfg.BaseURL == "" {
			return nil, errors.New("notifications: e-mail delivery needs Config.BaseURL for unsubscribe links")
		}
		d.chs = append(d.chs, ChannelEmail)
		u, _ := url.Parse(s.cfg.BaseURL)
		d.host = u.Hostname()
	}
	if o.Push != nil {
		d.chs = append(d.chs, ChannelPush)
	}
	return d, nil
}

type row struct {
	ID, Network, Member, EventID, Channel, Kind, TargetID string
	Attempts, Claims                                      int
	Expiry                                                time.Time
}

// Run loops RunOnce every interval until ctx ends. Errors are logged, not
// fatal, so a transient database busy does not stop delivery.
func (d *Dispatcher) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if _, err := d.RunOnce(ctx); err != nil && ctx.Err() == nil {
			d.s.log.Error("dispatch pass failed", "err_class", "store")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}

// RunOnce claims up to BatchSize due rows and processes them.
func (d *Dispatcher) RunOnce(ctx context.Context) (RunStats, error) {
	var st RunStats
	rows, err := d.claim(ctx, &st)
	if err != nil {
		return st, err
	}
	st.Claimed = len(rows)
	for _, r := range rows {
		if ctx.Err() != nil {
			return st, ctx.Err()
		}
		if h := d.o.Hooks.AfterClaim; h != nil {
			h(ctx, r.ID)
		}
		d.process(ctx, r, &st)
	}
	return st, nil
}

func (d *Dispatcher) claim(ctx context.Context, st *RunStats) ([]row, error) {
	var out []row
	err := d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := d.s.now().Unix()
		// Expiry always wins over everything else.
		res, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='expired',last_error='expired',updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE expiry<=? AND (status='ready' OR (status='in_flight' AND lease_until<=?))`, now, now, now)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		st.Expired += int(n)
		d.s.met.Expired.Add(n)
		// Rows whose lease expired too many times are poison: stop them.
		res, err = q.ExecContext(ctx, `UPDATE notification_delivery SET status='failed',last_error='poison',updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE status='in_flight' AND lease_until<=? AND claims>=?`, now, now, d.o.MaxClaims)
		if err != nil {
			return err
		}
		n, _ = res.RowsAffected()
		st.Failed += int(n)
		d.s.met.Failed.Add(n)
		d.s.met.Poisoned.Add(n)
		if len(d.chs) == 0 {
			return nil
		}
		ph := strings.TrimSuffix(strings.Repeat("?,", len(d.chs)), ",")
		args := []any{now, now}
		for _, c := range d.chs {
			args = append(args, c)
		}
		args = append(args, d.o.BatchSize)
		rs, err := q.QueryContext(ctx, `SELECT id,network_id,member_id,event_id,channel,kind,COALESCE(target_id,''),attempts,claims,expiry FROM notification_delivery
			WHERE ((status='ready' AND next_attempt_at<=?) OR (status='in_flight' AND lease_until<=?)) AND channel IN (`+ph+`)
			ORDER BY next_attempt_at,id LIMIT ?`, args...)
		if err != nil {
			return err
		}
		for rs.Next() {
			var r row
			var exp int64
			if err := rs.Scan(&r.ID, &r.Network, &r.Member, &r.EventID, &r.Channel, &r.Kind, &r.TargetID, &r.Attempts, &r.Claims, &exp); err != nil {
				rs.Close()
				return err
			}
			r.Expiry = time.Unix(exp, 0).UTC()
			out = append(out, r)
		}
		if err := rs.Err(); err != nil {
			rs.Close()
			return err
		}
		rs.Close()
		until := d.s.now().Add(d.o.Lease).Unix()
		for i := range out {
			if _, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='in_flight',lease_owner=?,lease_until=?,claims=claims+1,updated_at=? WHERE id=?`,
				d.o.Owner, until, now, out[i].ID); err != nil {
				return err
			}
			out[i].Claims++
		}
		return nil
	})
	return out, err
}

type sendTarget struct {
	ID        string
	Email     string
	Push      pushSecret
	SessionID string
}

type decision struct {
	act     string // send | done | lost
	targets []sendTarget
}

const (
	actSend = "send"
	actDone = "done" // the row was finalised (cancelled / expired / deferred / already sent)
	actLost = "lost"
)

func (d *Dispatcher) finishCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
}

func (d *Dispatcher) process(ctx context.Context, r row, st *RunStats) {
	log := d.s.log.With("delivery", r.ID, "channel", r.Channel)
	active, err := d.s.cfg.Directory.MemberActive(ctx, r.Network, r.Member)
	if err != nil {
		// Fail closed: do not send. Release the lease for a later pass.
		d.release(ctx, r, 1*time.Minute, "directory_unavailable")
		log.Warn("directory unavailable; deferred")
		st.Deferred++
		d.s.met.Deferred.Add(1)
		return
	}
	if !active {
		d.finalize(ctx, r, "cancelled", "member_inactive", st)
		return
	}
	dec, err := d.authorize(ctx, r, st)
	if err != nil {
		log.Error("authorize failed", "err_class", "store")
		return // lease expires; row is retried
	}
	if dec.act != actSend {
		if dec.act == actLost {
			st.LeaseLost++
			d.s.met.LeaseLost.Add(1)
		}
		return
	}
	if h := d.o.Hooks.AfterAuthorize; h != nil {
		h(ctx, r.ID)
	}
	var (
		sent, gone, perm, trans int
		retryAfter              time.Duration
		providerID              string
	)
	for _, t := range dec.targets {
		if t.SessionID != "" {
			ok, err := d.s.cfg.Directory.SessionActive(ctx, r.Network, r.Member, t.SessionID)
			if err != nil {
				trans++ // fail closed
				continue
			}
			if !ok {
				d.revokePush(ctx, r, t.ID, "logout")
				gone++
				continue
			}
		}
		if !d.stillAllowed(ctx, r, t) {
			gone++
			continue
		}
		msgID, serr := d.send(ctx, r, t)
		if h := d.o.Hooks.AfterSend; h != nil {
			if err := h(ctx, r.ID, t.ID); err != nil {
				return // simulated crash: nothing recorded
			}
		}
		if msgID != "" {
			providerID = msgID
		}
		status := "sent"
		if serr != nil {
			switch serr.Class {
			case ClassTargetGone:
				status = "gone"
				gone++
				d.dropTarget(ctx, r, t)
			case ClassPermanent:
				status = "failed"
				perm++
			default:
				status = "failed"
				trans++
				if serr.RetryAfter > retryAfter {
					retryAfter = serr.RetryAfter
				}
			}
			log.Warn("send failed", "class", serr.Class.String(), "code", serr.Code)
		} else {
			sent++
		}
		d.recordTarget(ctx, r, t.ID, status)
	}
	if h := d.o.Hooks.BeforeFinalize; h != nil {
		if err := h(ctx, r.ID); err != nil {
			return
		}
	}
	switch {
	case trans > 0:
		d.retry(ctx, r, retryAfter, st)
	case sent > 0:
		d.finalizeSent(ctx, r, providerID, st)
	case perm > 0:
		d.finalize(ctx, r, "failed", "permanent", st)
	default:
		d.finalize(ctx, r, "cancelled", "no_target", st)
	}
}

// authorize is the fresh authorization gate. It runs in one write transaction
// that also owns the lease, so a preference change, revocation or bounce that
// committed before it is always seen, and the decision to send is made against
// exactly the state that was read.
func (d *Dispatcher) authorize(ctx context.Context, r row, st *RunStats) (decision, error) {
	var dec decision
	s := d.s
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := s.now()
		var status, owner string
		var expiry int64
		if err := q.QueryRowContext(ctx, `SELECT status,COALESCE(lease_owner,''),expiry FROM notification_delivery WHERE id=?`, r.ID).Scan(&status, &owner, &expiry); err != nil {
			return err
		}
		if status != "in_flight" || owner != d.o.Owner {
			dec.act = actLost
			return nil
		}
		fin := func(status, reason string, next int64) error {
			dec.act = actDone
			_, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status=?,last_error=?,next_attempt_at=CASE WHEN ?>0 THEN ? ELSE next_attempt_at END,
				claims=CASE WHEN status='in_flight' AND ?='ready' THEN claims-1 ELSE claims END,
				updated_at=?,lease_owner=NULL,lease_until=NULL WHERE id=? AND lease_owner=?`,
				status, reason, next, next, status, now.Unix(), r.ID, d.o.Owner)
			return err
		}
		if expiry <= now.Unix() {
			st.Expired++
			s.met.Expired.Add(1)
			return fin("expired", "expired", 0)
		}
		cancel := func(reason string) error {
			st.Cancelled++
			s.met.Cancelled.Add(1)
			return fin("cancelled", reason, 0)
		}
		var pref Preference
		pref.Channel, pref.Active = r.Channel, true
		if r.Kind == KindEmailVerification {
			var sealed []byte
			err := q.QueryRowContext(ctx, `SELECT sealed FROM notification_email_targets WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL AND bounced_at IS NULL AND verified_at IS NULL`,
				r.TargetID, r.Network, r.Member).Scan(&sealed)
			if err == sql.ErrNoRows {
				return cancel("target_gone")
			}
			if err != nil {
				return err
			}
			addr, _, oerr := s.cfg.Secrets.Keyring.Open(sealed, sealAAD("email", r.Network, r.Member, r.TargetID))
			if oerr != nil {
				return cancel("target_unreadable")
			}
			dec.targets = []sendTarget{{ID: r.TargetID, Email: string(addr)}}
		} else {
			var qs, qe sql.NullInt64
			var tz string
			var err error
			if r.Kind == KindTest {
				err = q.QueryRowContext(ctx, `SELECT tz,quiet_start,quiet_end FROM notification_preferences WHERE network_id=? AND member_id=? AND channel=? AND enabled=1 ORDER BY event_type LIMIT 1`,
					r.Network, r.Member, r.Channel).Scan(&tz, &qs, &qe)
			} else {
				err = q.QueryRowContext(ctx, `SELECT tz,quiet_start,quiet_end FROM notification_preferences WHERE network_id=? AND member_id=? AND event_type=? AND channel=? AND enabled=1`,
					r.Network, r.Member, r.Kind, r.Channel).Scan(&tz, &qs, &qe)
			}
			if err == sql.ErrNoRows {
				return cancel("opt_out")
			}
			if err != nil {
				return err
			}
			pref.Enabled, pref.Timezone = true, tz
			if qs.Valid {
				pref.QuietStart, pref.QuietEnd = int(qs.Int64), int(qe.Int64)
			}
			allowed, aerr := pref.Allowed(now)
			if aerr != nil {
				// Unknown zone (e.g. no tzdata): fail closed, look again later.
				st.Deferred++
				s.met.Deferred.Add(1)
				return fin("ready", "zone_unavailable", now.Add(time.Hour).Unix())
			}
			if !allowed {
				next, _ := pref.NextAllowed(now)
				if next.IsZero() || !next.Before(time.Unix(expiry, 0)) {
					st.Expired++
					s.met.Expired.Add(1)
					return fin("expired", "quiet_hours", 0)
				}
				st.Deferred++
				s.met.Deferred.Add(1)
				return fin("ready", "quiet_hours", next.Unix())
			}
			if r.Channel == ChannelEmail {
				var id string
				var sealed []byte
				err := q.QueryRowContext(ctx, `SELECT id,sealed FROM notification_email_targets WHERE network_id=? AND member_id=? AND revoked_at IS NULL AND bounced_at IS NULL AND verified_at IS NOT NULL`,
					r.Network, r.Member).Scan(&id, &sealed)
				if err == sql.ErrNoRows {
					return cancel("no_target")
				}
				if err != nil {
					return err
				}
				addr, _, oerr := s.cfg.Secrets.Keyring.Open(sealed, sealAAD("email", r.Network, r.Member, id))
				if oerr != nil {
					return cancel("target_unreadable")
				}
				dec.targets = []sendTarget{{ID: id, Email: string(addr)}}
			} else {
				rs, err := q.QueryContext(ctx, `SELECT id,sealed,COALESCE(session_id,'') FROM push_subscriptions WHERE network_id=? AND member_id=? AND revoked_at IS NULL ORDER BY created_at,id`, r.Network, r.Member)
				if err != nil {
					return err
				}
				type raw struct {
					id, sess string
					sealed   []byte
				}
				var raws []raw
				for rs.Next() {
					var x raw
					if err := rs.Scan(&x.id, &x.sealed, &x.sess); err != nil {
						rs.Close()
						return err
					}
					raws = append(raws, x)
				}
				if err := rs.Err(); err != nil {
					rs.Close()
					return err
				}
				rs.Close()
				for _, x := range raws {
					b, _, oerr := s.cfg.Secrets.Keyring.Open(x.sealed, sealAAD("push", r.Network, r.Member, x.id))
					var ps pushSecret
					if oerr != nil || json.Unmarshal(b, &ps) != nil {
						if _, err := q.ExecContext(ctx, `UPDATE push_subscriptions SET revoked_at=?,revoke_reason='unreadable' WHERE id=?`, now.Unix(), x.id); err != nil {
							return err
						}
						continue
					}
					dec.targets = append(dec.targets, sendTarget{ID: x.id, Push: ps, SessionID: x.sess})
				}
			}
		}
		if len(dec.targets) == 0 {
			return cancel("no_target")
		}
		// Drop targets that already accepted this delivery on an earlier pass.
		done := map[string]string{}
		rs, err := q.QueryContext(ctx, `SELECT target_id,status FROM notification_delivery_target WHERE delivery_id=?`, r.ID)
		if err != nil {
			return err
		}
		for rs.Next() {
			var id, stt string
			if err := rs.Scan(&id, &stt); err != nil {
				rs.Close()
				return err
			}
			done[id] = stt
		}
		rs.Close()
		keep := dec.targets[:0]
		already := 0
		for _, t := range dec.targets {
			if done[t.ID] == "sent" || done[t.ID] == "gone" {
				already++
				continue
			}
			keep = append(keep, t)
		}
		dec.targets = keep
		if len(dec.targets) == 0 {
			if already > 0 {
				st.Sent++
				s.met.Sent.Add(1)
				return fin("sent", "", 0)
			}
			return cancel("no_target")
		}
		dec.act = actSend
		return nil
	})
	return dec, err
}

// stillAllowed re-reads the opt-in and target state immediately before each
// send, narrowing the window in which a concurrent opt-out, unsubscribe,
// revocation or bounce can lose the race. A send already handed to a provider
// cannot be recalled; that residual window is documented in README.md.
func (d *Dispatcher) stillAllowed(ctx context.Context, r row, t sendTarget) bool {
	good := false
	_ = d.s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		if r.Kind != KindEmailVerification {
			var n int
			var err error
			if r.Kind == KindTest {
				err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_preferences WHERE network_id=? AND member_id=? AND channel=? AND enabled=1`, r.Network, r.Member, r.Channel).Scan(&n)
			} else {
				err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_preferences WHERE network_id=? AND member_id=? AND event_type=? AND channel=? AND enabled=1`, r.Network, r.Member, r.Kind, r.Channel).Scan(&n)
			}
			if err != nil || n == 0 {
				return err
			}
		}
		var m int
		var err error
		if r.Channel == ChannelPush {
			err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL`, t.ID, r.Network, r.Member).Scan(&m)
		} else {
			err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_email_targets WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL AND bounced_at IS NULL`, t.ID, r.Network, r.Member).Scan(&m)
		}
		good = err == nil && m == 1
		return err
	})
	return good
}

func (d *Dispatcher) send(ctx context.Context, r row, t sendTarget) (providerID string, se *SendError) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if r.Channel == ChannelEmail {
		m := EmailMessage{To: t.Email, MessageID: newID() + "@" + d.host}
		if r.Kind == KindEmailVerification {
			m.Subject = VerifySubject
			link := d.s.cfg.BaseURL + "/notifications/confirm-email?t=" + d.s.confirmToken(r.Network, r.Member, t.ID, r.Expiry)
			m.Body = "Someone asked to send Circula notifications to this address.\n\nIf that was you, confirm here:\n" + link +
				"\n\nIf not, ignore this message and nothing will be sent.\n"
		} else {
			m.Subject = GenericSubject
			m.UnsubscribeURL = d.s.UnsubscribeURL(r.Network, r.Member, ChannelEmail)
			m.Body = GenericBody + "\n\nOpen Circula: " + d.s.cfg.BaseURL + PushOpenPath + "\n\nStop these emails: " + m.UnsubscribeURL + "\n"
		}
		if err := d.o.Email.SendEmail(ctx, m); err != nil {
			return m.MessageID, classify(err)
		}
		return m.MessageID, nil
	}
	ttl := r.Expiry.Sub(d.s.now())
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	if ttl < time.Minute {
		ttl = time.Minute
	}
	pm := PushMessage{Endpoint: t.Push.Endpoint, P256dh: t.Push.P256dh, Auth: t.Push.Auth, Payload: PushPayload(), TTL: ttl, Topic: ""}
	if err := d.o.Push.SendPush(ctx, pm); err != nil {
		return "", classify(err)
	}
	return "", nil
}

// PushPayload is the only push body ever sent: fixed text, no identifiers.
func PushPayload() []byte {
	b, _ := json.Marshal(map[string]any{"v": 1, "title": PushTitle, "body": GenericBody, "url": PushOpenPath, "tag": PushTag})
	return b
}

func (d *Dispatcher) recordTarget(ctx context.Context, r row, targetID, status string) {
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	_ = d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		_, err := q.ExecContext(ctx, `INSERT INTO notification_delivery_target(delivery_id,target_id,status,attempts,updated_at) VALUES(?,?,?,1,?)
			ON CONFLICT(delivery_id,target_id) DO UPDATE SET status=excluded.status,attempts=attempts+1,updated_at=excluded.updated_at`, r.ID, targetID, status, d.s.now().Unix())
		return err
	})
}

func (d *Dispatcher) revokePush(ctx context.Context, r row, id, reason string) {
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	_ = d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		_, err := q.ExecContext(ctx, `UPDATE push_subscriptions SET revoked_at=?,revoke_reason=? WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL`, d.s.now().Unix(), reason, id, r.Network, r.Member)
		return err
	})
}

// dropTarget disables a destination the provider says is permanently invalid.
func (d *Dispatcher) dropTarget(ctx context.Context, r row, t sendTarget) {
	if r.Channel == ChannelPush {
		d.revokePush(ctx, r, t.ID, "provider_gone")
		return
	}
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	var ab auditBatch
	err := d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		return d.s.bounceTx(ctx, q, &ab, r.Network, t.ID, true)
	})
	if err == nil {
		d.s.flushAudit(ctx, &ab)
	}
}

func (d *Dispatcher) release(ctx context.Context, r row, after time.Duration, reason string) {
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	next := d.s.now().Add(after).Unix()
	_ = d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		_, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='ready',next_attempt_at=?,claims=claims-1,last_error=?,updated_at=?,lease_owner=NULL,lease_until=NULL WHERE id=? AND lease_owner=? AND status='in_flight'`,
			next, reason, d.s.now().Unix(), r.ID, d.o.Owner)
		return err
	})
}

// finalize moves a row this worker still owns to a terminal status.
func (d *Dispatcher) finalize(ctx context.Context, r row, status, reason string, st *RunStats) {
	if !d.update(ctx, r, status, reason, 0, "", false) {
		st.LeaseLost++
		return
	}
	switch status {
	case "cancelled":
		st.Cancelled++
		d.s.met.Cancelled.Add(1)
	case "failed":
		st.Failed++
		d.s.met.Failed.Add(1)
	}
}

func (d *Dispatcher) finalizeSent(ctx context.Context, r row, providerID string, st *RunStats) {
	if !d.update(ctx, r, "sent", "", 0, providerID, false) {
		st.LeaseLost++
		return
	}
	st.Sent++
	d.s.met.Sent.Add(1)
}

func (d *Dispatcher) retry(ctx context.Context, r row, retryAfter time.Duration, st *RunStats) {
	now := d.s.now()
	attempts := r.Attempts + 1
	next, ok := Retry(now, r.Expiry, attempts)
	if retryAfter > MaxRetryAfter {
		retryAfter = MaxRetryAfter
	}
	if retryAfter > 0 {
		if ra := now.Add(retryAfter); ra.After(next) {
			next = ra
			ok = next.Before(r.Expiry)
		}
	}
	if ok {
		if d.updateRetry(ctx, r, attempts, next) {
			st.Retried++
			d.s.met.Retried.Add(1)
		} else {
			st.LeaseLost++
		}
		return
	}
	status, reason := "failed", "exhausted"
	if attempts < 6 {
		status, reason = "expired", "expired"
	}
	if !d.update(ctx, r, status, reason, attempts, "", true) {
		st.LeaseLost++
		return
	}
	if status == "failed" {
		st.Failed++
		d.s.met.Failed.Add(1)
	} else {
		st.Expired++
		d.s.met.Expired.Add(1)
	}
}

func (d *Dispatcher) updateRetry(ctx context.Context, r row, attempts int, next time.Time) bool {
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	ok := false
	_ = d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		res, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='ready',attempts=?,next_attempt_at=?,last_error='transient',updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE id=? AND lease_owner=? AND status='in_flight'`, attempts, next.Unix(), d.s.now().Unix(), r.ID, d.o.Owner)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		ok = n == 1
		return nil
	})
	if !ok {
		d.s.met.LeaseLost.Add(1)
	}
	return ok
}

// update is the fenced terminal write: it only applies while this worker still
// holds the lease, so a worker that lost its lease can never overwrite the
// outcome recorded by the one that took over.
func (d *Dispatcher) update(ctx context.Context, r row, status, reason string, attempts int, providerID string, setAttempts bool) bool {
	ctx, cancel := d.finishCtx(ctx)
	defer cancel()
	ok := false
	_ = d.s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		var prov any
		if providerID != "" {
			prov = providerID
		}
		att := r.Attempts
		if setAttempts {
			att = attempts
		}
		if status == "sent" {
			att = r.Attempts + 1
		}
		res, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status=?,last_error=NULLIF(?,''),attempts=?,provider_message_id=COALESCE(?,provider_message_id),updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE id=? AND lease_owner=? AND status='in_flight'`, status, reason, att, prov, d.s.now().Unix(), r.ID, d.o.Owner)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		ok = n == 1
		return nil
	})
	if !ok {
		d.s.met.LeaseLost.Add(1)
	}
	return ok
}
