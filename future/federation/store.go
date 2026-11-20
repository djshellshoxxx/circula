package federation

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Actor struct{ Network, Member string }

// Auth must enforce current active membership, admin/dual-control policy and
// CSRF at the host boundary. SQL reads may share this exact transaction.
type Auth func(context.Context, *sql.Conn, Actor, bool) error
type LocalListing struct {
	Owner               string
	Active, OwnerActive bool
	Revision            int64
	Content             SharedListing
}
type Project func(context.Context, *sql.Conn, Actor, string) (LocalListing, error)
type SigningKey func(context.Context, string, string) (string, ed25519.PrivateKey, error)
type Store struct {
	DB         *sql.DB
	Authorize  Auth
	Project    Project
	SigningKey SigningKey
	Now        func() time.Time
}
type PeerConfig struct {
	Local, Remote, Endpoint string
	Enabled                 bool
}
type TrustedKey struct {
	ID                  string
	Key                 ed25519.PublicKey
	NotBefore, NotAfter time.Time
	Revoked             bool
}

func (s Store) now() (time.Time, error) {
	if s.DB == nil || s.Now == nil {
		return time.Time{}, errors.New("missing federation dependencies")
	}
	now := s.Now()
	if now.IsZero() {
		return now, ErrInvalid
	}
	return now, nil
}
func (s Store) transaction(ctx context.Context, fn func(*sql.Conn) error) error {
	if s.DB == nil {
		return ErrInvalid
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	if err = fn(conn); err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}
func (s Store) Migrate(ctx context.Context) error {
	return s.transaction(ctx, func(c *sql.Conn) error {
		for _, query := range []string{
			`CREATE TABLE IF NOT EXISTS discovery_network(local TEXT PRIMARY KEY,enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN(0,1)))`,
			`CREATE TABLE IF NOT EXISTS discovery_peers(local TEXT NOT NULL,remote TEXT NOT NULL,endpoint TEXT NOT NULL,enabled INTEGER NOT NULL CHECK(enabled IN(0,1)),PRIMARY KEY(local,remote))`,
			`CREATE TABLE IF NOT EXISTS discovery_keys(local TEXT NOT NULL,remote TEXT NOT NULL,kid TEXT NOT NULL,key BLOB NOT NULL,valid_from INTEGER NOT NULL,valid_to INTEGER NOT NULL,revoked INTEGER NOT NULL CHECK(revoked IN(0,1)),PRIMARY KEY(local,remote,kid))`,
			`CREATE TABLE IF NOT EXISTS discovery_consent(local TEXT NOT NULL,remote TEXT NOT NULL,listing TEXT NOT NULL,owner TEXT NOT NULL,enabled INTEGER NOT NULL CHECK(enabled IN(0,1)),PRIMARY KEY(local,remote,listing))`,
			`CREATE TABLE IF NOT EXISTS discovery_inbox(local TEXT NOT NULL,remote TEXT NOT NULL,id TEXT NOT NULL,hash TEXT NOT NULL,received INTEGER NOT NULL,PRIMARY KEY(local,remote,id))`,
			`CREATE TABLE IF NOT EXISTS discovery_projection(local TEXT NOT NULL,remote TEXT NOT NULL,listing TEXT NOT NULL,revision INTEGER NOT NULL,kid TEXT NOT NULL,kind TEXT NOT NULL,content TEXT NOT NULL,expires INTEGER NOT NULL,PRIMARY KEY(local,remote,listing))`,
			`CREATE TABLE IF NOT EXISTS discovery_outbox(local TEXT NOT NULL,remote TEXT NOT NULL,id TEXT NOT NULL,owner TEXT NOT NULL,listing TEXT NOT NULL,revision INTEGER NOT NULL,source_revision INTEGER NOT NULL,kind TEXT NOT NULL,wire TEXT NOT NULL,expires INTEGER NOT NULL,state TEXT NOT NULL,attempts INTEGER NOT NULL,next_at INTEGER NOT NULL,PRIMARY KEY(local,remote,id))`,
			`CREATE TABLE IF NOT EXISTS discovery_sequence(local TEXT NOT NULL,remote TEXT NOT NULL,listing TEXT NOT NULL,revision INTEGER NOT NULL,PRIMARY KEY(local,remote,listing))`,
			`CREATE INDEX IF NOT EXISTS discovery_due ON discovery_outbox(state,next_at)`,
			`CREATE INDEX IF NOT EXISTS discovery_listing_order ON discovery_outbox(local,remote,listing,kind,revision)`,
			`CREATE TABLE IF NOT EXISTS discovery_rate(local TEXT NOT NULL,remote TEXT NOT NULL,window INTEGER NOT NULL,requests INTEGER NOT NULL,PRIMARY KEY(local,remote))`,
			`CREATE TABLE IF NOT EXISTS discovery_audit(local TEXT NOT NULL,actor TEXT NOT NULL,action TEXT NOT NULL,remote TEXT NOT NULL,at INTEGER NOT NULL)`,
		} {
			if err := validateSchemaObject(ctx, c, query); err != nil {
				return err
			}
			if _, err := c.ExecContext(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s Store) auth(ctx context.Context, c *sql.Conn, a Actor, admin bool) error {
	if !identifier(a.Network) || !identifier(a.Member) || s.Authorize == nil {
		return ErrInvalid
	}
	return s.Authorize(ctx, c, a, admin)
}
func (s Store) audit(ctx context.Context, c *sql.Conn, a Actor, action, remote string) error {
	now, e := s.now()
	if e != nil {
		return e
	}
	e = execOne(ctx, c, `INSERT INTO discovery_audit(local,actor,action,remote,at)VALUES(?,?,?,?,?)`, a.Network, a.Member, action, remote, now.Unix())
	return e
}
func (s Store) EnableNetwork(ctx context.Context, a Actor, enabled bool) error {
	return s.transaction(ctx, func(c *sql.Conn) error {
		if err := s.auth(ctx, c, a, true); err != nil {
			return err
		}
		if err := execOne(ctx, c, `INSERT INTO discovery_network(local,enabled)VALUES(?,?) ON CONFLICT(local) DO UPDATE SET enabled=excluded.enabled`, a.Network, enabled); err != nil {
			return err
		}
		if !enabled {
			if _, err := c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND state IN('pending','sending')`, a.Network); err != nil {
				return err
			}
		}
		return s.audit(ctx, c, a, "network-setting", "")
	})
}
func (s Store) ApprovePeer(ctx context.Context, a Actor, peer PeerConfig, key TrustedKey) error {
	if peer.Local != a.Network || peer.Remote == a.Network || !identifier(peer.Remote) || !identifier(key.ID) || len(key.Key) != ed25519.PublicKeySize || !key.NotBefore.Before(key.NotAfter) || key.Revoked || ValidateEndpoint(peer.Endpoint) != nil {
		return ErrInvalid
	}
	return s.transaction(ctx, func(c *sql.Conn) error {
		if err := s.auth(ctx, c, a, true); err != nil {
			return err
		}
		var count int
		if err := c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_peers WHERE local=?`, a.Network).Scan(&count); err != nil {
			return err
		}
		var exists int
		if err := c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_peers WHERE local=? AND remote=?`, a.Network, peer.Remote).Scan(&exists); err != nil {
			return err
		}
		if count >= 100 && exists == 0 {
			return errors.New("peer quota")
		}
		if err := execOne(ctx, c, `INSERT INTO discovery_peers(local,remote,endpoint,enabled)VALUES(?,?,?,?) ON CONFLICT(local,remote) DO UPDATE SET endpoint=excluded.endpoint,enabled=excluded.enabled`, a.Network, peer.Remote, peer.Endpoint, peer.Enabled); err != nil {
			return err
		}
		var existing []byte
		err := c.QueryRowContext(ctx, `SELECT key FROM discovery_keys WHERE local=? AND remote=? AND kid=?`, a.Network, peer.Remote, key.ID).Scan(&existing)
		if err == nil {
			return errors.New("key id already configured; rotate with a new id")
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err = execOne(ctx, c, `INSERT INTO discovery_keys VALUES(?,?,?,?,?,?,0)`, a.Network, peer.Remote, key.ID, []byte(key.Key), key.NotBefore.Unix(), key.NotAfter.Unix()); err != nil {
			return err
		}
		return s.audit(ctx, c, a, "peer-approve", peer.Remote)
	})
}
func (s Store) RevokePeer(ctx context.Context, a Actor, remote string) error {
	return s.transaction(ctx, func(c *sql.Conn) error {
		if !identifier(remote) {
			return ErrInvalid
		}
		if err := s.auth(ctx, c, a, true); err != nil {
			return err
		}
		if _, err := c.ExecContext(ctx, `UPDATE discovery_peers SET enabled=0 WHERE local=? AND remote=?`, a.Network, remote); err != nil {
			return err
		}
		if _, err := c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND remote=? AND state IN('pending','sending')`, a.Network, remote); err != nil {
			return err
		}
		if _, err := c.ExecContext(ctx, `UPDATE discovery_keys SET revoked=1 WHERE local=? AND remote=?`, a.Network, remote); err != nil {
			return err
		}

		return s.audit(ctx, c, a, "peer-revoke", remote)
	})
}
func (s Store) RevokeKey(ctx context.Context, a Actor, remote, kid string) error {
	return s.transaction(ctx, func(c *sql.Conn) error {
		if err := s.auth(ctx, c, a, true); err != nil {
			return err
		}
		if !identifier(remote) || !identifier(kid) {
			return ErrInvalid
		}
		if _, err := c.ExecContext(ctx, `UPDATE discovery_keys SET revoked=1 WHERE local=? AND remote=? AND kid=?`, a.Network, remote, kid); err != nil {
			return err
		}
		return s.audit(ctx, c, a, "key-revoke", remote)
	})
}
func (s Store) Consent(ctx context.Context, a Actor, remote, listing string, enabled bool) error {
	return s.transaction(ctx, func(c *sql.Conn) error {
		if err := s.auth(ctx, c, a, false); err != nil {
			return err
		}
		if !identifier(remote) || !identifier(listing) || s.Project == nil {
			return ErrInvalid
		}
		projection, err := s.Project(ctx, c, a, listing)
		if err != nil {
			return err
		}
		if projection.Owner != a.Member || !projection.OwnerActive {
			return ErrInvalid
		}
		var previous bool
		previousErr := c.QueryRowContext(ctx, `SELECT enabled FROM discovery_consent WHERE local=? AND remote=? AND listing=?`, a.Network, remote, listing).Scan(&previous)
		if previousErr != nil && !errors.Is(previousErr, sql.ErrNoRows) {
			return previousErr
		}

		if enabled {
			if _, e := peerEnabled(ctx, c, a.Network, remote); e != nil {
				return e
			}
			if errors.Is(previousErr, sql.ErrNoRows) {
				var count int
				if e := c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_consent WHERE local=? AND remote=?`, a.Network, remote).Scan(&count); e != nil {
					return e
				}
				if count >= 1000 {
					return errors.New("consent quota")
				}
			}
		} else if errors.Is(previousErr, sql.ErrNoRows) {
			return nil
		}
		if err = execOne(ctx, c, `INSERT INTO discovery_consent VALUES(?,?,?,?,?) ON CONFLICT(local,remote,listing) DO UPDATE SET owner=excluded.owner,enabled=excluded.enabled`, a.Network, remote, listing, a.Member, enabled); err != nil {
			return err
		}
		if !enabled {
			now, e := s.now()
			if e != nil {
				return e
			}
			if previous {
				if e = s.queueWithdrawalTx(ctx, c, a.Network, remote, listing, a.Member, now); e != nil {
					return e
				}
			}
			if _, err = c.ExecContext(ctx, `UPDATE discovery_outbox SET state='cancelled' WHERE local=? AND remote=? AND listing=? AND kind='listing.publish' AND state IN('pending','sending')`, a.Network, remote, listing); err != nil {
				return err
			}
		}
		return s.audit(ctx, c, a, "listing-consent", remote)
	})
}
func peerEnabled(ctx context.Context, c *sql.Conn, local, remote string) (string, error) {
	var endpoint string
	err := c.QueryRowContext(ctx, `SELECT p.endpoint FROM discovery_peers p JOIN discovery_network n ON n.local=p.local WHERE p.local=? AND p.remote=? AND p.enabled=1 AND n.enabled=1`, local, remote).Scan(&endpoint)
	if err != nil {
		return "", ErrInvalid
	}
	return endpoint, nil
}
func (s Store) Receive(ctx context.Context, local, remote, wire string) (bool, error) {
	if !identifier(local) || !identifier(remote) || local == remote {
		return false, ErrInvalid
	}
	kid, err := WireKeyID(wire)
	if err != nil {
		return false, err
	}
	now, err := s.now()
	if err != nil {
		return false, err
	}
	duplicate := false
	err = s.transaction(ctx, func(c *sql.Conn) error {
		if _, e := peerEnabled(ctx, c, local, remote); e != nil {
			return e
		}
		var key []byte
		var from, to int64
		var revoked bool
		if e := c.QueryRowContext(ctx, `SELECT key,valid_from,valid_to,revoked FROM discovery_keys WHERE local=? AND remote=? AND kid=?`, local, remote, kid).Scan(&key, &from, &to, &revoked); e != nil || revoked || now.Unix() < from || now.Unix() >= to {
			return ErrInvalid
		}
		event, e := VerifyDiscovery(wire, local, remote, kid, key, now)
		if e != nil || event.Issued < from || event.Expires > to {
			return ErrInvalid
		}
		var window int64
		var requests int
		e = c.QueryRowContext(ctx, `SELECT window,requests FROM discovery_rate WHERE local=? AND remote=?`, local, remote).Scan(&window, &requests)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		current := now.Unix() / 60
		if window != current {
			requests = 0
		}
		if requests >= 120 {
			return errors.New("peer rate limit")
		}
		if e = execOne(ctx, c, `INSERT INTO discovery_rate VALUES(?,?,?,?) ON CONFLICT(local,remote) DO UPDATE SET window=excluded.window,requests=excluded.requests`, local, remote, current, requests+1); e != nil {
			return e
		}
		hash := sha256.Sum256([]byte(wire))
		digest := hex.EncodeToString(hash[:])
		var prior string
		e = c.QueryRowContext(ctx, `SELECT hash FROM discovery_inbox WHERE local=? AND remote=? AND id=?`, local, remote, event.ID).Scan(&prior)
		if e == nil {
			if prior != digest {
				return errors.New("event equivocation")
			}
			duplicate = true
			return nil
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		var count int
		if e = c.QueryRowContext(ctx, `SELECT count(*) FROM discovery_inbox WHERE local=? AND remote=?`, local, remote).Scan(&count); e != nil {
			return e
		}
		if count >= 10000 {
			return errors.New("inbox quota")
		}
		content, _ := json.Marshal(event.Content)
		var revision int64
		var kind, oldContent string
		e = c.QueryRowContext(ctx, `SELECT revision,kind,content FROM discovery_projection WHERE local=? AND remote=? AND listing=?`, local, remote, event.Listing).Scan(&revision, &kind, &oldContent)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if e == nil && revision == event.Revision && (kind != event.Kind || oldContent != string(content)) {
			return errors.New("revision equivocation")
		}
		if e = execOne(ctx, c, `INSERT INTO discovery_inbox VALUES(?,?,?,?,?)`, local, remote, event.ID, digest, now.Unix()); e != nil {
			return e
		}
		if event.Revision > revision {
			e = execOne(ctx, c, `INSERT INTO discovery_projection VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(local,remote,listing) DO UPDATE SET revision=excluded.revision,kid=excluded.kid,kind=excluded.kind,content=excluded.content,expires=excluded.expires`, local, remote, event.Listing, event.Revision, kid, event.Kind, string(content), event.Expires)
		}
		return e
	})
	return duplicate, err
}

type RemoteListing struct {
	Peer, Listing string
	Revision      int64
	Content       SharedListing
}

func (s Store) Browse(ctx context.Context, a Actor) ([]RemoteListing, error) {
	return s.BrowsePage(ctx, a, "", "", 100)
}

// BrowsePage is a tenant-authorized keyset page; cursor values are public opaque
// IDs, not authorization. Host HTTP adapter may sign/bind a cursor if caching.
func (s Store) BrowsePage(ctx context.Context, a Actor, afterPeer, afterListing string, limit int) ([]RemoteListing, error) {
	if limit < 1 || limit > 100 || (afterPeer != "" && !identifier(afterPeer)) || (afterListing != "" && !identifier(afterListing)) {
		return nil, ErrInvalid
	}
	out := []RemoteListing{}
	now, err := s.now()
	if err != nil {
		return nil, err
	}
	err = s.transaction(ctx, func(c *sql.Conn) error {
		if e := s.auth(ctx, c, a, false); e != nil {
			return e
		}
		rows, e := c.QueryContext(ctx, `SELECT d.remote,d.listing,d.revision,d.content FROM discovery_projection d JOIN discovery_network n ON n.local=d.local JOIN discovery_peers p ON p.local=d.local AND p.remote=d.remote JOIN discovery_keys k ON k.local=d.local AND k.remote=d.remote AND k.kid=d.kid WHERE d.local=? AND n.enabled=1 AND p.enabled=1 AND k.revoked=0 AND d.kind='listing.publish' AND d.expires>? AND (d.remote>? OR (d.remote=? AND d.listing>?)) ORDER BY d.remote,d.listing LIMIT ?`, a.Network, now.Unix(), afterPeer, afterPeer, afterListing, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var r RemoteListing
			var content string
			if e = rows.Scan(&r.Peer, &r.Listing, &r.Revision, &content); e != nil {
				return e
			}
			canonicalErr := json.Unmarshal([]byte(content), &r.Content)
			canonical, _ := json.Marshal(r.Content)
			if canonicalErr != nil || string(canonical) != content || !identifier(r.Peer) || !identifier(r.Listing) || r.Revision < 1 || !validShared(r.Content) {
				return ErrInvalid
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// Schema validation is deliberately strict for this first candidate schema. An
// incompatible prior table/index requires an explicit reviewed migration rather
// than a successful IF NOT EXISTS that leaves uniqueness or columns absent.
func validateSchemaObject(ctx context.Context, c *sql.Conn, query string) error {
	kind := "table"
	if strings.HasPrefix(query, "CREATE INDEX") {
		kind = "index"
	}
	tail := strings.SplitN(query, "IF NOT EXISTS ", 2)
	if len(tail) != 2 {
		return ErrInvalid
	}
	name := strings.FieldsFunc(tail[1], func(r rune) bool { return r == ' ' || r == '(' })[0]
	var prior, actualKind string
	err := c.QueryRowContext(ctx, `SELECT sql,type FROM sqlite_schema WHERE name=? COLLATE NOCASE`, name).Scan(&prior, &actualKind)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	normalize := func(v string) string {
		return strings.ToUpper(strings.Join(strings.Fields(strings.Replace(v, " IF NOT EXISTS", "", 1)), " "))
	}
	if actualKind != kind || normalize(prior) != normalize(query) {
		return errors.New("incompatible federation schema: " + name)
	}
	return nil
}
func execOne(ctx context.Context, c *sql.Conn, q string, args ...any) error {
	result, err := c.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("federation safety write was not recorded")
	}
	return nil
}
