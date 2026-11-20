package federation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Queue is an explicit owner action. A fresh content snapshot and owner consent
// are checked in the same transaction as the bounded immutable outbox record.
func (s Store) Queue(ctx context.Context, a Actor, remote, listing, eventID string, revoke bool) (string, error) {
	if !identifier(remote) || !identifier(listing) || !identifier(eventID) {
		return "", ErrInvalid
	}
	now, err := s.now()
	if err != nil {
		return "", err
	}
	wire := ""
	err = s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, false); e != nil {
			return e
		}
		if s.Project == nil || s.SigningKey == nil {
			return ErrInvalid
		}
		if _, e := peerEnabled(ctx, c, a.Network, remote); e != nil {
			return e
		}
		p, e := s.Project(ctx, c, a, listing)
		if e != nil || p.Owner != a.Member || !p.OwnerActive || p.Revision < 1 {
			return ErrInvalid
		}
		kind := "listing.publish"
		if revoke {
			kind = "listing.revoke"
		} else {
			if !p.Active {
				return ErrInvalid
			}
			var consent bool
			var owner string
			if e = c.QueryRowContext(ctx, `SELECT owner,enabled FROM discovery_consent WHERE local=? AND remote=? AND listing=?`, a.Network, remote, listing).Scan(&owner, &consent); e != nil || !consent || owner != a.Member {
				return ErrInvalid
			}
		}
		var oldOwner, oldListing, oldKind string
		var oldRevision int64
		e = c.QueryRowContext(ctx, `SELECT wire,owner,listing,kind,source_revision FROM discovery_outbox WHERE local=? AND remote=? AND id=?`, a.Network, remote, eventID).Scan(&wire, &oldOwner, &oldListing, &oldKind, &oldRevision)
		if e == nil {
			if wire == "" {
				return errors.New("expired idempotency record")
			}
			if oldOwner != a.Member || oldListing != listing || oldKind != kind || oldRevision != p.Revision {
				return errors.New("idempotency conflict")
			}
			return nil
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		var count int
		if e = c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_outbox WHERE local=? AND remote=?`, a.Network, remote).Scan(&count); e != nil {
			return e
		}
		if count >= 1000 {
			return errors.New("outbox quota")
		}
		var hourly int
		if e = c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_outbox WHERE local=? AND owner=? AND expires>?`, a.Network, a.Member, now.Add(23*time.Hour).Unix()).Scan(&hourly); e != nil {
			return e
		}
		if hourly >= 120 {
			return errors.New("publish rate limit")
		}
		var revision int64
		e = c.QueryRowContext(ctx, `SELECT revision FROM discovery_sequence WHERE local=? AND remote=? AND listing=?`, a.Network, remote, listing).Scan(&revision)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if revision >= 1<<60 {
			return ErrInvalid
		}
		revision++
		if e = execOne(ctx, c, `INSERT INTO discovery_sequence VALUES(?,?,?,?) ON CONFLICT(local,remote,listing) DO UPDATE SET revision=excluded.revision`, a.Network, remote, listing, revision); e != nil {
			return e
		}
		event := Discovery{Version: 2, ID: eventID, Issuer: a.Network, Audience: remote, Kind: kind, Listing: listing, Revision: revision, Issued: now.Unix(), Expires: now.Add(24 * time.Hour).Unix()}
		if !revoke {
			event.Content = &p.Content
		}
		kid, key, e := s.SigningKey(ctx, a.Network, remote)
		if e != nil {
			return e
		}
		wire, e = SignDiscovery(event, kid, key)
		if e != nil {
			return e
		}
		if e = execOne(ctx, c, `INSERT INTO discovery_outbox VALUES(?,?,?,?,?,?,?,?,?,?,'pending',0,?)`, a.Network, remote, eventID, a.Member, listing, revision, p.Revision, kind, wire, event.Expires, now.Unix()); e != nil {
			return e
		}
		if revoke {
			if _, e = c.ExecContext(ctx, `UPDATE discovery_consent SET enabled=0 WHERE local=? AND remote=? AND listing=? AND owner=?`, a.Network, remote, listing, a.Member); e != nil {
				return e
			}
			if _, e = c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND remote=? AND listing=? AND kind='listing.publish' AND state IN('pending','sending')`, a.Network, remote, listing); e != nil {
				return e
			}
		}
		return s.audit(ctx, c, a, "listing-queue", remote)
	})
	if err != nil {
		return "", err
	}
	return wire, nil
}

// Delivery sends only to a configured fixed endpoint. HTTPSDelivery supplies the
// production SSRF policy; tests inject a synthetic transport explicitly.
type Delivery interface {
	Send(context.Context, string, string) error
}

// Dispatch processes one explicit outbox item with bounded attempts. A process
// crash leaves a lease for at-least-once retry; the receiver deduplicates events.
func (s Store) Dispatch(ctx context.Context, local, remote, id string, delivery Delivery) error {
	if !identifier(local) || !identifier(remote) || !identifier(id) || delivery == nil {
		return ErrInvalid
	}
	now, err := s.now()
	if err != nil {
		return err
	}
	var endpoint, wire, owner, listing, kind, state string
	var sourceRevision, revision, expires, next int64
	var attempts int
	err = s.transaction(ctx, func(c *sql.Conn) error {
		var e error
		endpoint, e = peerEnabled(ctx, c, local, remote)
		if e != nil {
			return e
		}
		e = c.QueryRowContext(ctx, `SELECT wire,owner,listing,kind,state,source_revision,revision,expires,attempts,next_at FROM discovery_outbox WHERE local=? AND remote=? AND id=?`, local, remote, id).Scan(&wire, &owner, &listing, &kind, &state, &sourceRevision, &revision, &expires, &attempts, &next)
		if e != nil {
			return e
		}
		if attempts < 0 || attempts > 6 || (kind != "listing.publish" && kind != "listing.revoke") {
			return ErrInvalid
		}
		if state == "sent" {
			return nil
		}
		if state == "unsealed" {
			if s.SigningKey == nil {
				return ErrInvalid
			}
			kid, key, e := s.SigningKey(ctx, local, remote)
			if e != nil {
				return e
			}
			event := Discovery{Version: 2, ID: id, Issuer: local, Audience: remote, Kind: "listing.revoke", Listing: listing, Revision: revision, Issued: now.Unix(), Expires: now.Add(24 * time.Hour).Unix()}
			wire, e = SignDiscovery(event, kid, key)
			if e != nil {
				return e
			}
			expires = event.Expires
			state = "pending"
			if e = execOne(ctx, c, `UPDATE discovery_outbox SET wire=?,expires=?,state='pending' WHERE local=? AND remote=? AND id=?`, wire, expires, local, remote, id); e != nil {
				return e
			}
		}

		if (state != "pending" && state != "sending") || next > now.Unix() || expires <= now.Unix() || attempts >= 6 {
			return errors.New("outbox unavailable")
		}
		event, parseErr := decodeOutgoing(wire)
		if parseErr != nil || !validDiscovery(event, remote, local, now) || event.ID != id || event.Listing != listing || event.Kind != kind || event.Revision != revision || event.Expires != expires {
			return ErrInvalid
		}
		a := Actor{local, owner}
		if kind == "listing.publish" {
			if e = s.auth(ctx, c, a, false); e != nil {
				return e
			}
		}
		if kind == "listing.publish" {
			if s.Project == nil {
				return ErrInvalid
			}
			p, e := s.Project(ctx, c, a, listing)
			if e != nil || !p.Active || !p.OwnerActive || p.Owner != owner || p.Revision != sourceRevision {
				return ErrInvalid
			}
			var consent bool
			if e = c.QueryRowContext(ctx, `SELECT enabled FROM discovery_consent WHERE local=? AND remote=? AND listing=? AND owner=?`, local, remote, listing, owner).Scan(&consent); e != nil || !consent {
				return ErrInvalid
			}
			var event Discovery
			parts, parseErr := decodeOutgoing(wire)
			if parseErr != nil {
				return parseErr
			}
			event = parts
			encoded, _ := json.Marshal(p.Content)
			prior, _ := json.Marshal(event.Content)
			if string(encoded) != string(prior) {
				return errors.New("projection revision mismatch")
			}
		}
		// Reserve a bounded lease after the fresh authorization check.
		if e = execOne(ctx, c, `UPDATE discovery_outbox SET state='sending',attempts=attempts+1,next_at=? WHERE local=? AND remote=? AND id=?`, now.Add(time.Minute).Unix(), local, remote, id); e != nil {
			return e
		}
		return nil
	})
	if err != nil || state == "sent" {
		return err
	}

	// A delivery is considered in flight once this bounded operation starts. Local
	// revocation cancels not-yet-started records; a network send cannot be retracted.
	// Never hold the SQLite writer lock during remote I/O.
	sendErr := delivery.Send(ctx, endpoint, wire)
	err = s.transaction(ctx, func(c *sql.Conn) error {
		var currentState string
		var currentAttempts int
		if e := c.QueryRowContext(ctx, `SELECT state,attempts FROM discovery_outbox WHERE local=? AND remote=? AND id=?`, local, remote, id).Scan(&currentState, &currentAttempts); e != nil {
			return e
		}
		if currentState != "sending" || currentAttempts != attempts+1 {
			return nil
		}
		if sendErr == nil {
			_, e := c.ExecContext(ctx, `UPDATE discovery_outbox SET state='sent' WHERE local=? AND remote=? AND id=?`, local, remote, id)
			return e
		}
		newState := "pending"
		if attempts+1 >= 6 || expires <= now.Add(time.Duration(1<<uint(attempts))*time.Minute).Unix() {
			newState = "dead"
		}
		_, e := c.ExecContext(ctx, `UPDATE discovery_outbox SET state=?,next_at=? WHERE local=? AND remote=? AND id=?`, newState, now.Add(time.Duration(1<<uint(attempts))*time.Minute).Unix(), local, remote, id)
		return e
	})
	if err != nil {
		return err
	}
	if sendErr != nil {
		return fmt.Errorf("delivery failed: %w", sendErr)
	}
	return nil
}
