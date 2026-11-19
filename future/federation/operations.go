package federation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

// Maintenance trims expired shared text/wire bodies while preserving high-water
// marks, replay hashes and idempotency records. It never evicts safety metadata.
func (s Store) Maintenance(ctx context.Context, a Actor) error {
	now, err := s.now()
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, true); e != nil {
			return e
		}
		if _, e := c.ExecContext(ctx, `UPDATE discovery_projection SET content='null' WHERE local=? AND expires<=?`, a.Network, now.Unix()); e != nil {
			return e
		}
		if _, e := c.ExecContext(ctx, `UPDATE discovery_outbox SET wire='',state='expired' WHERE local=? AND expires<=? AND state!='unsealed'`, a.Network, now.Unix()); e != nil {
			return e
		}
		if _, e := c.ExecContext(ctx, `DELETE FROM discovery_audit WHERE local=? AND at<?`, a.Network, now.Add(-90*24*time.Hour).Unix()); e != nil {
			return e
		}
		return nil
	})
}

// Depart locally revokes all consent, cancels queued publishes and keeps an
// immutable withdrawal-required audit marker. No private profile is exported.
// A system adapter must call this during the existing departure transaction.
func (s Store) Depart(ctx context.Context, a Actor, member string) error {
	if !identifier(member) {
		return ErrInvalid
	}
	now, e := s.now()
	if e != nil {
		return e
	}
	return s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, true); e != nil {
			return e
		}
		rows, e := c.QueryContext(ctx, `SELECT remote,listing FROM discovery_consent WHERE local=? AND owner=? AND enabled=1`, a.Network, member)
		if e != nil {
			return e
		}
		type target struct{ peer, listing string }
		targets := []target{}
		for rows.Next() {
			var item target
			if e = rows.Scan(&item.peer, &item.listing); e != nil {
				rows.Close()
				return e
			}
			targets = append(targets, item)
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return e
		}
		rows.Close()
		for _, item := range targets {
			if e = s.queueWithdrawalTx(ctx, c, a.Network, item.peer, item.listing, member, now); e != nil {
				return e
			}
		}
		if _, e = c.ExecContext(ctx, `UPDATE discovery_consent SET enabled=0 WHERE local=? AND owner=?`, a.Network, member); e != nil {
			return e
		}
		if _, e = c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND owner=? AND kind='listing.publish' AND state IN('pending','sending')`, a.Network, member); e != nil {
			return e
		}
		return s.audit(ctx, c, a, "departure-withdrawal-queued", "")
	})
}

// Retry is an explicit admin operation; expired or exhausted deliveries cannot be
// resurrected. It respects peer enablement and fresh source checks in Dispatch.
func (s Store) Retry(ctx context.Context, a Actor, remote, id string) error {
	if !identifier(remote) || !identifier(id) {
		return ErrInvalid
	}
	now, e := s.now()
	if e != nil {
		return e
	}
	return s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, true); e != nil {
			return e
		}
		if _, e := peerEnabled(ctx, c, a.Network, remote); e != nil {
			return e
		}
		res, e := c.ExecContext(ctx, `UPDATE discovery_outbox SET state='pending',next_at=? WHERE local=? AND remote=? AND id=? AND state IN('pending','dead') AND attempts<6 AND expires>?`, now.Unix(), a.Network, remote, id, now.Unix())
		if e != nil {
			return e
		}
		count, e := res.RowsAffected()
		if e != nil {
			return e
		}
		if count != 1 {
			return errors.New("retry unavailable")
		}
		return s.audit(ctx, c, a, "delivery-retry", remote)
	})
}

// queueWithdrawalTx needs no signing key. Withdrawal survives a signer outage or
// owner departure and is sealed at dispatch; it contains no public text.
func (s Store) queueWithdrawalTx(ctx context.Context, c *sql.Conn, local, remote, listing, owner string, now time.Time) error {
	// No remote content exists unless a publish was queued. Coalesce repeated
	// opt-out/departure for an already withdrawn publication and never let a
	// toggle-only consent consume the safety reserve.
	var published, withdrawn int64
	if e := c.QueryRowContext(ctx, `SELECT COALESCE(max(CASE WHEN kind='listing.publish' THEN revision ELSE 0 END),0),COALESCE(max(CASE WHEN kind='listing.revoke' THEN revision ELSE 0 END),0) FROM discovery_outbox WHERE local=? AND remote=? AND listing=?`, local, remote, listing).Scan(&published, &withdrawn); e != nil {
		return e
	}
	if published == 0 || withdrawn >= published {
		return nil
	}
	var count int
	if e := c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_outbox WHERE local=? AND remote=?`, local, remote).Scan(&count); e != nil {
		return e
	}
	// Ordinary creation stops at 1,000 lifetime rows. Each such publish can
	// need at most one coalesced automatic withdrawal, so 2,000 total rows
	// reserves enough safety capacity for every existing publication.
	if count >= 2000 {
		return errors.New("withdrawal reserve quota")
	}
	var revision int64
	e := c.QueryRowContext(ctx, `SELECT revision FROM discovery_sequence WHERE local=? AND remote=? AND listing=?`, local, remote, listing).Scan(&revision)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	if revision >= 1<<60 {
		return ErrInvalid
	}
	revision++
	if e = execOne(ctx, c, `INSERT INTO discovery_sequence VALUES(?,?,?,?) ON CONFLICT(local,remote,listing) DO UPDATE SET revision=excluded.revision`, local, remote, listing, revision); e != nil {
		return e
	}
	seed := make([]byte, 16)
	if _, e = rand.Read(seed); e != nil {
		return e
	}
	id := "withdraw-" + hex.EncodeToString(seed)
	e = execOne(ctx, c, `INSERT INTO discovery_outbox VALUES(?,?,?,?,?,?,0,'listing.revoke','',?,'unsealed',0,?)`, local, remote, id, owner, listing, revision, now.Add(24*time.Hour).Unix(), now.Unix())
	return e
}

// Pending is an admin-only bounded keyset scan. Terminal rows cannot starve
// active entries; callers can persist the last ID for explicit worker batches.
func (s Store) Pending(ctx context.Context, a Actor, afterRemote, afterID string, limit int) ([]QueueStatus, error) {
	if afterID != "" && !identifier(afterID) || afterRemote != "" && !identifier(afterRemote) || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	now, e := s.now()
	if e != nil {
		return nil, e
	}
	out := []QueueStatus{}
	e = s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, true); e != nil {
			return e
		}
		rows, e := c.QueryContext(ctx, `SELECT remote,id,listing,kind,state,attempts FROM discovery_outbox WHERE local=? AND (remote>? OR(remote=? AND id>?)) AND state IN('unsealed','pending','sending') AND next_at<=? ORDER BY remote,id LIMIT ?`, a.Network, afterRemote, afterRemote, afterID, now.Unix(), limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var q QueueStatus
			if e = rows.Scan(&q.Remote, &q.ID, &q.Listing, &q.Kind, &q.State, &q.Attempts); e != nil {
				return e
			}
			out = append(out, q)
		}
		return rows.Err()
	})
	return out, e
}

// WithdrawListing is an admin/system lifecycle hook for listing deletion or
// moderation. It needs no live source projection or signer, so a removed source
// cannot make its own withdrawal impossible. Core must call the corresponding
// adapter inside its deletion transaction; this reference method is standalone.
func (s Store) WithdrawListing(ctx context.Context, a Actor, listing string) error {
	if !identifier(listing) {
		return ErrInvalid
	}
	now, err := s.now()
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(c *sql.Conn) error {
		if err := s.auth(ctx, c, a, true); err != nil {
			return err
		}
		rows, err := c.QueryContext(ctx, `SELECT remote,owner FROM discovery_consent WHERE local=? AND listing=? AND enabled=1`, a.Network, listing)
		if err != nil {
			return err
		}
		type target struct{ remote, owner string }
		targets := []target{}
		for rows.Next() {
			var item target
			if err = rows.Scan(&item.remote, &item.owner); err != nil {
				rows.Close()
				return err
			}
			targets = append(targets, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, item := range targets {
			if err = s.queueWithdrawalTx(ctx, c, a.Network, item.remote, listing, item.owner, now); err != nil {
				return err
			}
		}
		if _, err = c.ExecContext(ctx, `UPDATE discovery_consent SET enabled=0 WHERE local=? AND listing=?`, a.Network, listing); err != nil {
			return err
		}
		if _, err = c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND listing=? AND kind='listing.publish' AND state IN('pending','sending')`, a.Network, listing); err != nil {
			return err
		}
		return s.audit(ctx, c, a, "listing-withdrawal-queued", "")
	})
}
