package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

// CreateCategory adds a category; names are unique per network after folding.
func (s *Store) CreateCategory(ctx context.Context, c d.Category, actor string, max int) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM taxonomy_category WHERE network_id=?`, c.Network).Scan(&n); err != nil {
			return err
		}
		if n >= max {
			return &d.Error{Kind: d.ErrQuota, Field: "name", Detail: "category limit reached"}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO taxonomy_category (network_id, id, name, name_fold, kind, enabled, created_at) VALUES (?,?,?,?,?,1,?)`,
			c.Network, c.ID, c.Name, d.Fold(c.Name), c.Kind, unix(c.Created))
		if isUnique(err) {
			return &d.Error{Kind: d.ErrDuplicate, Field: "name", Detail: "a category with this name already exists"}
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, c.Network, actor, "category.create", c.ID, c.Name, c.Created)
	})
}

// UpdateCategory renames and/or toggles a category.
func (s *Store) UpdateCategory(ctx context.Context, network, id string, name *string, enabled *bool, actor string) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		var x int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM taxonomy_category WHERE network_id=? AND id=?`, network, id).Scan(&x); errors.Is(err, sql.ErrNoRows) {
			return d.ErrNotFound
		} else if err != nil {
			return err
		}
		if name != nil {
			_, err := tx.ExecContext(ctx, `UPDATE taxonomy_category SET name=?, name_fold=? WHERE network_id=? AND id=?`, *name, d.Fold(*name), network, id)
			if isUnique(err) {
				return &d.Error{Kind: d.ErrDuplicate, Field: "name", Detail: "a category with this name already exists"}
			}
			if err != nil {
				return err
			}
		}
		if enabled != nil {
			v := 0
			if *enabled {
				v = 1
			}
			if _, err := tx.ExecContext(ctx, `UPDATE taxonomy_category SET enabled=? WHERE network_id=? AND id=?`, v, network, id); err != nil {
				return err
			}
		}
		return audit(ctx, tx, network, actor, "category.update", id, "", time.Now())
	})
}

// ListCategories lists a network's categories with listing counts.
func (s *Store) ListCategories(ctx context.Context, network string, includeDisabled bool) ([]d.Category, error) {
	q := `SELECT c.id, c.name, c.kind, c.enabled, c.created_at,
	  (SELECT count(*) FROM listing_category lc WHERE lc.network_id=c.network_id AND lc.category_id=c.id)
	  FROM taxonomy_category c WHERE c.network_id=?`
	if !includeDisabled {
		q += ` AND c.enabled=1`
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY c.name_fold, c.id`, network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []d.Category
	for rows.Next() {
		c := d.Category{Network: network}
		var en int
		var cr int64
		if err := rows.Scan(&c.ID, &c.Name, &c.Kind, &en, &cr, &c.Listings); err != nil {
			return nil, err
		}
		c.Enabled, c.Created = en == 1, time.Unix(cr, 0).UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateArea adds a coarse area.
func (s *Store) CreateArea(ctx context.Context, a d.Area, actor string, max int) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM coarse_area WHERE network_id=?`, a.Network).Scan(&n); err != nil {
			return err
		}
		if n >= max {
			return &d.Error{Kind: d.ErrQuota, Field: "name", Detail: "area limit reached"}
		}
		var lat, lon any
		if a.Cell != nil {
			lat, lon = a.Cell.Lat10, a.Cell.Lon10
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO coarse_area (network_id, id, name, name_fold, enabled, lat10, lon10, created_at) VALUES (?,?,?,?,1,?,?,?)`,
			a.Network, a.ID, a.Name, d.Fold(a.Name), lat, lon, time.Now().Unix())
		if isUnique(err) {
			return &d.Error{Kind: d.ErrDuplicate, Field: "name", Detail: "an area with this name already exists"}
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, a.Network, actor, "area.create", a.ID, a.Name, time.Now())
	})
}

// UpdateArea renames and/or toggles an area.
func (s *Store) UpdateArea(ctx context.Context, network, id string, name *string, enabled *bool, actor string) error {
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		var x int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM coarse_area WHERE network_id=? AND id=?`, network, id).Scan(&x); errors.Is(err, sql.ErrNoRows) {
			return d.ErrNotFound
		} else if err != nil {
			return err
		}
		if name != nil {
			_, err := tx.ExecContext(ctx, `UPDATE coarse_area SET name=?, name_fold=? WHERE network_id=? AND id=?`, *name, d.Fold(*name), network, id)
			if isUnique(err) {
				return &d.Error{Kind: d.ErrDuplicate, Field: "name", Detail: "an area with this name already exists"}
			}
			if err != nil {
				return err
			}
		}
		if enabled != nil {
			v := 0
			if *enabled {
				v = 1
			}
			if _, err := tx.ExecContext(ctx, `UPDATE coarse_area SET enabled=? WHERE network_id=? AND id=?`, v, network, id); err != nil {
				return err
			}
		}
		return audit(ctx, tx, network, actor, "area.update", id, "", time.Now())
	})
}

// ListAreas lists a network's areas.
func (s *Store) ListAreas(ctx context.Context, network string, includeDisabled bool) ([]d.Area, error) {
	q := `SELECT id, name, enabled, lat10, lon10 FROM coarse_area WHERE network_id=?`
	if !includeDisabled {
		q += ` AND enabled=1`
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY name_fold, id`, network)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []d.Area
	for rows.Next() {
		a := d.Area{Network: network}
		var en int
		var lat, lon sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Name, &en, &lat, &lon); err != nil {
			return nil, err
		}
		a.Enabled = en == 1
		if lat.Valid {
			a.Cell = &d.GeoCell{Lat10: int(lat.Int64), Lon10: int(lon.Int64)}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListTags lists tags of one status ("" for all) with usage counts.
func (s *Store) ListTags(ctx context.Context, network string, status d.TagStatus, limit int) ([]d.Tag, error) {
	q := `SELECT t.name, t.status, t.moderated_by, t.updated_at,
	  (SELECT count(*) FROM listing_tags lt WHERE lt.network_id=t.network_id AND lt.tag=t.name) FROM taxonomy_tag t WHERE t.network_id=?`
	args := []any{network}
	if status != "" {
		q += ` AND t.status=?`
		args = append(args, string(status))
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY t.name LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []d.Tag
	for rows.Next() {
		t := d.Tag{Network: network}
		var st string
		var up int64
		if err := rows.Scan(&t.Name, &st, &t.ModeratedBy, &up, &t.Listings); err != nil {
			return nil, err
		}
		t.Status, t.Updated = d.TagStatus(st), time.Unix(up, 0).UTC()
		out = append(out, t)
	}
	return out, rows.Err()
}

// ModerateTag sets a tag's status (creating admin-curated tags) and moves
// every affected listing into a new revision so open page sequences drop them.
func (s *Store) ModerateTag(ctx context.Context, network, tag string, status d.TagStatus, actor string, now time.Time) error {
	return s.write(ctx, func(tx *sql.Tx, seq int64) error {
		res, err := tx.ExecContext(ctx, `UPDATE taxonomy_tag SET status=?, moderated_by=?, updated_at=? WHERE network_id=? AND name=?`, string(status), actor, unix(now), network, tag)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO taxonomy_tag (network_id, name, status, moderated_by, updated_at) VALUES (?,?,?,?,?)`, network, tag, string(status), actor, unix(now)); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE discovery_listing_projection SET seq=? WHERE network_id=? AND id IN (SELECT listing_id FROM listing_tags WHERE network_id=? AND tag=?)`, seq, network, network, tag); err != nil {
			return err
		}
		return audit(ctx, tx, network, actor, "tag.moderate", tag, string(status), now)
	})
}
