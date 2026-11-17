package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

const mediaSelect = `SELECT id, network_id, listing_id, owner_id, sha256, mime, alt, width, height, thumb_w, thumb_h, bytes, thumb_bytes, storage_path, thumb_path, status, position, created_at, retention_until FROM listing_media`

func scanMedia(sc scanner) (d.MediaInfo, error) {
	var m d.MediaInfo
	var st string
	var cr int64
	var ret sql.NullInt64
	if err := sc.Scan(&m.ID, &m.Network, &m.ListingID, &m.OwnerID, &m.SHA256, &m.MIME, &m.Alt, &m.Width, &m.Height, &m.ThumbWidth, &m.ThumbHeight,
		&m.Bytes, &m.ThumbBytes, &m.StoragePath, &m.ThumbPath, &st, &m.Position, &cr, &ret); err != nil {
		return m, err
	}
	m.Status, m.Created, m.RetentionUntil = d.MediaStatus(st), time.Unix(cr, 0).UTC(), fromNull(ret)
	return m, nil
}

// ReserveMedia atomically checks per-listing count and byte quotas and inserts
// a pending row in one statement under the write lock, so concurrent uploads
// can never exceed the limits.
func (s *Store) ReserveMedia(ctx context.Context, m d.MediaInfo, q d.MediaQuota) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO listing_media
		 (id, network_id, listing_id, owner_id, sha256, mime, alt, width, height, thumb_w, thumb_h, bytes, thumb_bytes, storage_path, thumb_path, status, position, created_at)
		 SELECT ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'pending',
		   (SELECT COALESCE(MAX(position),0)+1 FROM listing_media WHERE network_id=? AND listing_id=? AND status IN ('pending','active')), ?
		 WHERE EXISTS (SELECT 1 FROM discovery_listing_projection p WHERE p.network_id=? AND p.id=? AND p.owner_id=? AND p.status IN ('active','paused'))
		   AND (SELECT count(*) FROM listing_media WHERE network_id=? AND listing_id=? AND status IN ('pending','active')) < ?
		   AND (SELECT COALESCE(SUM(bytes+thumb_bytes),0) FROM listing_media WHERE network_id=? AND listing_id=? AND status IN ('pending','active')) + ? <= ?`,
			m.ID, m.Network, m.ListingID, m.OwnerID, m.SHA256, m.MIME, m.Alt, m.Width, m.Height, m.ThumbWidth, m.ThumbHeight, m.Bytes, m.ThumbBytes, m.StoragePath, m.ThumbPath,
			m.Network, m.ListingID, unix(m.Created),
			m.Network, m.ListingID, m.OwnerID,
			m.Network, m.ListingID, q.MaxCount,
			m.Network, m.ListingID, m.Bytes+m.ThumbBytes, q.MaxBytes)
		if isUnique(err) {
			return &d.Error{Kind: d.ErrDuplicate, Field: "file", Detail: "this photo is already attached"}
		}
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			return nil
		}
		var x int
		err = tx.QueryRowContext(ctx, `SELECT 1 FROM discovery_listing_projection WHERE network_id=? AND id=? AND owner_id=? AND status IN ('active','paused')`, m.Network, m.ListingID, m.OwnerID).Scan(&x)
		if errors.Is(err, sql.ErrNoRows) {
			return d.ErrNotFound
		}
		if err != nil {
			return err
		}
		return &d.Error{Kind: d.ErrQuota, Field: "file", Detail: "photo count or size limit for this listing reached"}
	})
}

// ActivateMedia publishes a reserved photo once both blobs are durable.
func (s *Store) ActivateMedia(ctx context.Context, network, listingID, mediaID string, now time.Time) error {
	return s.write(ctx, func(tx *sql.Tx, seq int64) error {
		res, err := tx.ExecContext(ctx, `UPDATE listing_media SET status='active' WHERE id=? AND network_id=? AND listing_id=? AND status='pending'`, mediaID, network, listingID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return d.ErrNotFound
		}
		if err := touch(ctx, tx, network, listingID, now, seq); err != nil {
			return err
		}
		return audit(ctx, tx, network, "", "media.add", mediaID, listingID, now)
	})
}

// RemoveMedia hides a photo immediately and schedules GC at retentionUntil.
func (s *Store) RemoveMedia(ctx context.Context, network, listingID, mediaID, actor string, now, retentionUntil time.Time) error {
	return s.write(ctx, func(tx *sql.Tx, seq int64) error {
		res, err := tx.ExecContext(ctx, `UPDATE listing_media SET status='removed', retention_until=? WHERE id=? AND network_id=? AND listing_id=? AND status IN ('pending','active')`,
			unix(retentionUntil), mediaID, network, listingID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return d.ErrNotFound
		}
		if err := touch(ctx, tx, network, listingID, now, seq); err != nil {
			return err
		}
		return audit(ctx, tx, network, actor, "media.remove", mediaID, listingID, now)
	})
}

// GetMedia loads one photo row (any status).
func (s *Store) GetMedia(ctx context.Context, network, listingID, mediaID string) (d.MediaInfo, error) {
	m, err := scanMedia(s.db.QueryRowContext(ctx, mediaSelect+` WHERE id=? AND network_id=? AND listing_id=?`, mediaID, network, listingID))
	if errors.Is(err, sql.ErrNoRows) {
		return m, d.ErrNotFound
	}
	return m, err
}

// ClaimGarbage marks every media row eligible for reclamation as 'deleting'
// and returns up to Limit deleting rows (including those left by a crash).
// Eligible: removed past retention, pending past TTL, active media of a
// listing archived past retention, and media whose listing is gone.
func (s *Store) ClaimGarbage(ctx context.Context, p d.GCParams) ([]d.MediaInfo, error) {
	var out []d.MediaInfo
	err := s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now := p.Now.UTC().Unix()
		_, err := tx.ExecContext(ctx, `UPDATE listing_media SET status='deleting' WHERE id IN (
		  SELECT m.id FROM listing_media m WHERE m.status != 'deleting' AND (
		    (m.status='removed' AND m.retention_until <= ?)
		    OR (m.status='pending' AND m.created_at <= ?)
		    OR (m.status='active' AND EXISTS (SELECT 1 FROM discovery_listing_projection pr WHERE pr.network_id=m.network_id AND pr.id=m.listing_id
		          AND pr.status='archived' AND pr.archived_at IS NOT NULL AND pr.archived_at + ? <= ?))
		    OR NOT EXISTS (SELECT 1 FROM discovery_listing_projection pr WHERE pr.network_id=m.network_id AND pr.id=m.listing_id)
		  ) LIMIT ?)`,
			now, now-int64(p.PendingTTL/time.Second), int64(p.ArchivedRetention/time.Second), now, p.Limit)
		if err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, mediaSelect+` WHERE status='deleting' ORDER BY created_at, id LIMIT ?`, p.Limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMedia(rows)
			if err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}

// PathReferenced reports whether another live row still uses a blob path.
func (s *Store) PathReferenced(ctx context.Context, path, exceptID string) (bool, error) {
	var x int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM listing_media WHERE (storage_path=? OR thumb_path=?) AND id != ? AND status != 'deleting' LIMIT 1`, path, path, exceptID).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// DeleteMediaRow removes a row previously claimed for deletion.
func (s *Store) DeleteMediaRow(ctx context.Context, id string) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM listing_media WHERE id=? AND status='deleting'`, id)
		return err
	})
}
