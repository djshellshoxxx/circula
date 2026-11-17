package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

// Store implements discovery.Repository on SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex // serializes write transactions (one process per file)
}

var _ d.Repository = (*Store)(nil)

// New wraps an opened, migrated database.
func New(db *sql.DB) *Store { return &Store{db: db} }

func unix(t time.Time) int64 { return t.UTC().Unix() }

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Unix()
}

func fromNull(n sql.NullInt64) time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return time.Unix(n.Int64, 0).UTC()
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// write runs fn in one serialized transaction whose first statement bumps the
// global revision (acquiring the write lock up front, so a deferred
// read-to-write upgrade can never fail with a snapshot conflict).
func (s *Store) write(ctx context.Context, fn func(tx *sql.Tx, seq int64) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var seq int64
	if err := tx.QueryRowContext(ctx, `UPDATE discovery_revision SET seq = seq + 1 WHERE id = 1 RETURNING seq`).Scan(&seq); err != nil {
		tx.Rollback()
		return err
	}
	if err := fn(tx, seq); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func audit(ctx context.Context, tx *sql.Tx, network, actor, action, target, detail string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO discovery_audit (network_id, actor, action, target, detail, at) VALUES (?,?,?,?,?,?)`,
		network, actor, action, target, detail, unix(at))
	return err
}

// Audit returns the newest audit entries for a network.
func (s *Store) Audit(ctx context.Context, network string, limit int) ([]d.AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, actor, action, target, detail, at FROM discovery_audit WHERE network_id=? ORDER BY id DESC LIMIT ?`, network, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []d.AuditEntry
	for rows.Next() {
		var e d.AuditEntry
		var id, at int64
		if err := rows.Scan(&id, &e.Actor, &e.Action, &e.Target, &e.Detail, &at); err != nil {
			return nil, err
		}
		e.ID, e.Network, e.At = fmt.Sprint(id), network, time.Unix(at, 0).UTC()
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertListing mirrors one core listing into the read model.
func (s *Store) UpsertListing(ctx context.Context, rec d.ListingRecord) error {
	return s.write(ctx, func(tx *sql.Tx, seq int64) error {
		var owner string
		var archived sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT owner_id, archived_at FROM discovery_listing_projection WHERE network_id=? AND id=?`, rec.Network, rec.ID).Scan(&owner, &archived)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return err
		case owner != rec.OwnerID:
			return &d.Error{Kind: d.ErrConflict, Field: "owner", Detail: "listing owner is immutable"}
		}
		var arch any
		if rec.Status == d.StatusArchived {
			switch {
			case archived.Valid:
				arch = archived.Int64
			case !rec.ArchivedAt.IsZero():
				arch = unix(rec.ArchivedAt)
			default:
				arch = unix(rec.Updated)
			}
		}
		owneract := 0
		if rec.OwnerActive {
			owneract = 1
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO discovery_listing_projection
		 (id, network_id, owner_id, kind, title, description, title_fold, desc_fold, status, owner_active, created_at, updated_at, archived_at, seq)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT (network_id, id) DO UPDATE SET kind=excluded.kind, title=excluded.title, description=excluded.description,
		   title_fold=excluded.title_fold, desc_fold=excluded.desc_fold, status=excluded.status, owner_active=excluded.owner_active,
		   updated_at=excluded.updated_at, archived_at=excluded.archived_at, seq=excluded.seq`,
			rec.ID, rec.Network, rec.OwnerID, rec.Kind, rec.Title, rec.Description, d.Fold(rec.Title), d.Fold(rec.Description),
			rec.Status, owneract, unix(rec.Created), unix(rec.Updated), arch, seq)
		return err
	})
}

const listingCols = `p.id, p.owner_id, p.kind, p.title, p.description, p.status, p.owner_active, p.created_at, p.updated_at, p.archived_at,
 c.category_id, c.listing_type, c.condition, c.area_id, a.availability, a.available_from, a.expires_at, a.renewed_at, a.renew_count, a.version`

type scanner interface{ Scan(...any) error }

// scanListing scans listingCols (+ optional trailing extras).
func scanListing(network string, sc scanner, extra ...any) (d.ListingRecord, d.Meta, error) {
	var rec d.ListingRecord
	var m d.Meta
	var active int
	var created, updated int64
	var archived, from, exp, renewed, renewCount, version sql.NullInt64
	var cat, typ, cond, area, avail sql.NullString
	dest := []any{&rec.ID, &rec.OwnerID, &rec.Kind, &rec.Title, &rec.Description, &rec.Status, &active, &created, &updated, &archived,
		&cat, &typ, &cond, &area, &avail, &from, &exp, &renewed, &renewCount, &version}
	dest = append(dest, extra...)
	if err := sc.Scan(dest...); err != nil {
		return rec, m, err
	}
	rec.Network, rec.OwnerActive = network, active == 1
	rec.Created, rec.Updated, rec.ArchivedAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC(), fromNull(archived)
	m = d.Meta{CategoryID: cat.String, Type: typ.String, Condition: cond.String, AreaID: area.String, Availability: avail.String,
		AvailableFrom: fromNull(from), ExpiresAt: fromNull(exp), RenewedAt: fromNull(renewed), RenewCount: int(renewCount.Int64), Version: version.Int64}
	return rec, m, nil
}

const listingJoin = ` FROM discovery_listing_projection p
 LEFT JOIN listing_category c ON c.network_id=p.network_id AND c.listing_id=p.id
 LEFT JOIN listing_availability a ON a.network_id=p.network_id AND a.listing_id=p.id `

// GetListing loads one listing with all tags and pending/active media.
func (s *Store) GetListing(ctx context.Context, network, id string) (d.ListingView, error) {
	var v d.ListingView
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return v, err
	}
	defer tx.Rollback()
	rec, m, err := scanListing(network, tx.QueryRowContext(ctx, `SELECT `+listingCols+listingJoin+` WHERE p.network_id=? AND p.id=?`, network, id))
	if errors.Is(err, sql.ErrNoRows) {
		return v, d.ErrNotFound
	}
	if err != nil {
		return v, err
	}
	v.ListingRecord, v.Meta = rec, m
	rows, err := tx.QueryContext(ctx, `SELECT lt.tag, COALESCE(t.status,'pending') FROM listing_tags lt
	  LEFT JOIN taxonomy_tag t ON t.network_id=lt.network_id AND t.name=lt.tag
	  WHERE lt.network_id=? AND lt.listing_id=? ORDER BY lt.tag`, network, id)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var t d.ListingTag
		var st string
		if err := rows.Scan(&t.Name, &st); err != nil {
			rows.Close()
			return v, err
		}
		t.Status = d.TagStatus(st)
		v.Tags = append(v.Tags, t)
	}
	rows.Close()
	mrows, err := tx.QueryContext(ctx, mediaSelect+` WHERE network_id=? AND listing_id=? AND status IN ('pending','active') ORDER BY position, id`, network, id)
	if err != nil {
		return v, err
	}
	defer mrows.Close()
	for mrows.Next() {
		mi, err := scanMedia(mrows)
		if err != nil {
			return v, err
		}
		v.Media = append(v.Media, mi)
	}
	return v, mrows.Err()
}

// SaveDiscovery validates references and writes classification, availability
// and tags in one transaction with optimistic versioning.
func (s *Store) SaveDiscovery(ctx context.Context, in d.SaveInput) (d.Meta, error) {
	var out d.Meta
	err := s.write(ctx, func(tx *sql.Tx, seq int64) error {
		if err := s.requireListing(ctx, tx, in.Network, in.ListingID); err != nil {
			return err
		}
		var ver int64
		var curCat, curArea string
		err := tx.QueryRowContext(ctx, `SELECT version FROM listing_availability WHERE network_id=? AND listing_id=?`, in.Network, in.ListingID).Scan(&ver)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if ver != in.ExpectedVersion {
			return &d.Error{Kind: d.ErrConflict, Field: "version", Detail: "listing changed since it was read"}
		}
		err = tx.QueryRowContext(ctx, `SELECT category_id, area_id FROM listing_category WHERE network_id=? AND listing_id=?`, in.Network, in.ListingID).Scan(&curCat, &curArea)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		m := in.Meta
		if m.CategoryID != "" {
			var kind string
			var enabled int
			err := tx.QueryRowContext(ctx, `SELECT kind, enabled FROM taxonomy_category WHERE network_id=? AND id=?`, in.Network, m.CategoryID).Scan(&kind, &enabled)
			if errors.Is(err, sql.ErrNoRows) {
				return &d.Error{Kind: d.ErrInvalid, Field: "category", Detail: "unknown category"}
			}
			if err != nil {
				return err
			}
			if enabled == 0 && m.CategoryID != curCat {
				return &d.Error{Kind: d.ErrInvalid, Field: "category", Detail: "category is disabled"}
			}
			if kind != d.TypeAny && m.Type != "" && m.Type != kind {
				return &d.Error{Kind: d.ErrInvalid, Field: "type", Detail: "type does not match the category"}
			}
		}
		if m.AreaID != "" {
			var enabled int
			err := tx.QueryRowContext(ctx, `SELECT enabled FROM coarse_area WHERE network_id=? AND id=?`, in.Network, m.AreaID).Scan(&enabled)
			if errors.Is(err, sql.ErrNoRows) {
				return &d.Error{Kind: d.ErrInvalid, Field: "area", Detail: "unknown area"}
			}
			if err != nil {
				return err
			}
			if enabled == 0 && m.AreaID != curArea {
				return &d.Error{Kind: d.ErrInvalid, Field: "area", Detail: "area is disabled"}
			}
		}
		// Tags: rejected tags can never be attached; new tags start pending
		// (approved when an admin proposes them).
		for _, t := range in.Tags {
			var st string
			err := tx.QueryRowContext(ctx, `SELECT status FROM taxonomy_tag WHERE network_id=? AND name=?`, in.Network, t).Scan(&st)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				var n int
				if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM taxonomy_tag WHERE network_id=?`, in.Network).Scan(&n); err != nil {
					return err
				}
				if n >= in.MaxNetworkTags {
					return &d.Error{Kind: d.ErrQuota, Field: "tags", Detail: "network tag limit reached"}
				}
				ns := string(d.TagPending)
				if in.ActorIsAdmin {
					ns = string(d.TagApproved)
				}
				by := ""
				if in.ActorIsAdmin {
					by = in.ActorID
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO taxonomy_tag (network_id, name, status, moderated_by, updated_at) VALUES (?,?,?,?,?)`, in.Network, t, ns, by, unix(in.Now)); err != nil {
					return err
				}
			case err != nil:
				return err
			case st == string(d.TagRejected):
				return &d.Error{Kind: d.ErrRejected, Field: "tags", Detail: "tag " + t + " is not permitted in this network"}
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM listing_tags WHERE network_id=? AND listing_id=?`, in.Network, in.ListingID); err != nil {
			return err
		}
		for _, t := range in.Tags {
			if _, err := tx.ExecContext(ctx, `INSERT INTO listing_tags (network_id, listing_id, tag) VALUES (?,?,?)`, in.Network, in.ListingID, t); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO listing_category (network_id, listing_id, category_id, listing_type, condition, area_id) VALUES (?,?,?,?,?,?)
		  ON CONFLICT (network_id, listing_id) DO UPDATE SET category_id=excluded.category_id, listing_type=excluded.listing_type, condition=excluded.condition, area_id=excluded.area_id`,
			in.Network, in.ListingID, m.CategoryID, m.Type, m.Condition, m.AreaID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO listing_availability (network_id, listing_id, availability, available_from, expires_at, version, updated_at) VALUES (?,?,?,?,?,1,?)
		  ON CONFLICT (network_id, listing_id) DO UPDATE SET availability=excluded.availability, available_from=excluded.available_from,
		    expires_at=excluded.expires_at, version=listing_availability.version+1, updated_at=excluded.updated_at`,
			in.Network, in.ListingID, m.Availability, nullTime(m.AvailableFrom), nullTime(m.ExpiresAt), unix(in.Now)); err != nil {
			return err
		}
		if err := touch(ctx, tx, in.Network, in.ListingID, in.Now, seq); err != nil {
			return err
		}
		out = m
		out.Version = ver + 1
		return audit(ctx, tx, in.Network, in.ActorID, "listing.discovery.save", in.ListingID, fmt.Sprintf("cat=%s type=%s area=%s tags=%d", m.CategoryID, m.Type, m.AreaID, len(in.Tags)), in.Now)
	})
	return out, err
}

func (s *Store) requireListing(ctx context.Context, tx *sql.Tx, network, id string) error {
	var x int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM discovery_listing_projection WHERE network_id=? AND id=?`, network, id).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return d.ErrNotFound
	}
	return err
}

// touch marks a listing as recently updated and part of a new revision.
func touch(ctx context.Context, tx *sql.Tx, network, id string, now time.Time, seq int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE discovery_listing_projection SET updated_at = MAX(updated_at, ?), seq = ? WHERE network_id=? AND id=?`, unix(now), seq, network, id)
	return err
}

// Renew extends expiry with a version check; the listing row's identity and
// created_at (the anchor for trade references) are never modified.
func (s *Store) Renew(ctx context.Context, network, id, actor string, expectedVersion int64, newExpires, now time.Time) (d.Meta, error) {
	var out d.Meta
	err := s.write(ctx, func(tx *sql.Tx, seq int64) error {
		var status string
		err := tx.QueryRowContext(ctx, `SELECT status FROM discovery_listing_projection WHERE network_id=? AND id=?`, network, id).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return d.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != d.StatusActive && status != d.StatusPaused {
			return &d.Error{Kind: d.ErrInvalid, Field: "status", Detail: "only active or paused listings can be renewed"}
		}
		var ver int64
		err = tx.QueryRowContext(ctx, `SELECT version FROM listing_availability WHERE network_id=? AND listing_id=?`, network, id).Scan(&ver)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if ver != expectedVersion {
			return &d.Error{Kind: d.ErrConflict, Field: "version", Detail: "listing was renewed or edited concurrently"}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO listing_availability (network_id, listing_id, availability, expires_at, renewed_at, renew_count, version, updated_at) VALUES (?,?,?,?,?,1,1,?)
		  ON CONFLICT (network_id, listing_id) DO UPDATE SET expires_at=excluded.expires_at, renewed_at=excluded.renewed_at,
		   renew_count=listing_availability.renew_count+1, version=listing_availability.version+1, updated_at=excluded.updated_at`,
			network, id, "", unix(newExpires), unix(now), unix(now)); err != nil {
			return err
		}
		if err := touch(ctx, tx, network, id, now, seq); err != nil {
			return err
		}
		out.Version = ver + 1
		out.ExpiresAt = newExpires
		return audit(ctx, tx, network, actor, "listing.renew", id, "expires="+newExpires.UTC().Format(time.RFC3339), now)
	})
	return out, err
}
