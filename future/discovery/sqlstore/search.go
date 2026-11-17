package sqlstore

import (
	"context"
	"database/sql"
	"strings"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// Search runs one authorized, bounded page query inside a single read
// transaction (one WAL snapshot). Rows changed after the cursor's snapshot
// revision are excluded, so a page sequence never shows duplicates, post-
// snapshot inserts, or content changed since the first page. Every value is a
// bound parameter; only fixed fragments are concatenated.
func (s *Store) Search(ctx context.Context, p d.SearchParams) (d.SearchPage, error) {
	var page d.SearchPage
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return page, err
	}
	defer tx.Rollback()

	snap := p.Snapshot
	if snap == 0 {
		if err := tx.QueryRowContext(ctx, `SELECT seq FROM discovery_revision WHERE id=1`).Scan(&snap); err != nil {
			return page, err
		}
	}
	page.Snapshot = snap

	// score expression (args come first in the final statement)
	score := "0"
	var sargs []any
	if len(p.Terms) > 0 {
		parts := []string{}
		for _, t := range p.Terms {
			parts = append(parts, `(3*(instr(p.title_fold,?)>0) + (instr(p.desc_fold,?)>0) + 2*EXISTS(SELECT 1 FROM listing_tags lt JOIN taxonomy_tag tg ON tg.network_id=lt.network_id AND tg.name=lt.tag WHERE lt.network_id=p.network_id AND lt.listing_id=p.id AND tg.status='approved' AND instr(lt.tag,?)>0))`)
			sargs = append(sargs, t, t, t)
		}
		score = strings.Join(parts, " + ")
		if len(p.Terms) > 1 {
			score += ` + 4*(instr(p.title_fold,?)>0)`
			sargs = append(sargs, strings.Join(p.Terms, " "))
		}
	}

	where := []string{`p.network_id = ?`, `p.seq <= ?`}
	wargs := []any{p.Network, snap}
	now := p.Now.UTC().Unix()
	switch p.Scope {
	case d.ScopeMine:
		where = append(where, `p.owner_id = ?`)
		wargs = append(wargs, p.Viewer)
	case d.ScopeAll:
	default:
		where = append(where, `p.status = 'active'`, `p.owner_active = 1`,
			`(a.expires_at IS NULL OR a.expires_at > ?)`, `(a.available_from IS NULL OR a.available_from <= ?)`)
		wargs = append(wargs, now, now)
	}
	if p.Category != "" {
		where = append(where, `c.category_id = ?`)
		wargs = append(wargs, p.Category)
	}
	if p.Type != "" {
		where = append(where, `c.listing_type = ?`)
		wargs = append(wargs, p.Type)
	}
	if p.Kind != "" {
		where = append(where, `p.kind = ?`)
		wargs = append(wargs, p.Kind)
	}
	if p.Condition != "" {
		where = append(where, `c.condition = ?`)
		wargs = append(wargs, p.Condition)
	}
	if p.Availability != "" {
		where = append(where, `COALESCE(NULLIF(a.availability,''),'available') = ?`)
		wargs = append(wargs, p.Availability)
	}
	if p.AreaFilter {
		if len(p.AreaIDs) == 0 {
			where = append(where, `0`)
		} else {
			where = append(where, `c.area_id IN (`+placeholders(len(p.AreaIDs))+`)`)
			for _, a := range p.AreaIDs {
				wargs = append(wargs, a)
			}
		}
	}
	if p.Tag != "" {
		where = append(where, `EXISTS(SELECT 1 FROM listing_tags lt JOIN taxonomy_tag tg ON tg.network_id=lt.network_id AND tg.name=lt.tag WHERE lt.network_id=p.network_id AND lt.listing_id=p.id AND lt.tag=? AND tg.status='approved')`)
		wargs = append(wargs, p.Tag)
	}
	for _, t := range p.Terms {
		where = append(where, `(instr(p.title_fold,?)>0 OR instr(p.desc_fold,?)>0 OR EXISTS(SELECT 1 FROM listing_tags lt JOIN taxonomy_tag tg ON tg.network_id=lt.network_id AND tg.name=lt.tag WHERE lt.network_id=p.network_id AND lt.listing_id=p.id AND tg.status='approved' AND instr(lt.tag,?)>0))`)
		wargs = append(wargs, t, t, t)
	}

	q := `SELECT * FROM (SELECT ` + listingCols + `, (` + score + `) AS score ` + listingJoin + ` WHERE ` + strings.Join(where, " AND ") + `) r`
	args := append(append([]any{}, sargs...), wargs...)
	relevance := p.Sort == d.SortRelevance && len(p.Terms) > 0
	var kw []string
	if a := p.After; a != nil {
		if relevance {
			kw = append(kw, `(score < ? OR (score = ? AND (updated_at < ? OR (updated_at = ? AND id > ?))))`)
			args = append(args, a.Score, a.Score, a.Updated, a.Updated, a.ID)
		} else {
			kw = append(kw, `(updated_at < ? OR (updated_at = ? AND id > ?))`)
			args = append(args, a.Updated, a.Updated, a.ID)
		}
	}
	if len(kw) > 0 {
		q += ` WHERE ` + strings.Join(kw, " AND ")
	}
	if relevance {
		q += ` ORDER BY score DESC, updated_at DESC, id ASC`
	} else {
		q += ` ORDER BY updated_at DESC, id ASC`
	}
	q += ` LIMIT ?`
	args = append(args, p.Limit+1)

	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return page, err
	}
	var items []d.SearchItem
	for rows.Next() {
		var sc int64
		rec, m, err := scanListing(p.Network, rows, &sc)
		if err != nil {
			rows.Close()
			return page, err
		}
		items = append(items, d.SearchItem{Listing: rec, Meta: m, Score: sc})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return page, err
	}
	rows.Close()
	if len(items) > p.Limit {
		items = items[:p.Limit]
		last := items[len(items)-1]
		pos := d.Position{Updated: last.Listing.Updated.Unix(), ID: last.Listing.ID}
		if relevance {
			pos.Score = last.Score
		}
		page.Next = &pos
	}
	if len(items) > 0 {
		if err := s.enrich(ctx, tx, p.Network, items); err != nil {
			return page, err
		}
	}
	page.Items = items
	return page, nil
}

// enrich adds approved tags and a cover thumbnail to the page's items.
func (s *Store) enrich(ctx context.Context, tx *sql.Tx, network string, items []d.SearchItem) error {
	idx := map[string]int{}
	ids := make([]any, 0, len(items)+1)
	ids = append(ids, network)
	for i, it := range items {
		idx[it.Listing.ID] = i
		ids = append(ids, it.Listing.ID)
	}
	ph := placeholders(len(items))
	rows, err := tx.QueryContext(ctx, `SELECT lt.listing_id, lt.tag FROM listing_tags lt JOIN taxonomy_tag tg ON tg.network_id=lt.network_id AND tg.name=lt.tag
	  WHERE lt.network_id=? AND tg.status='approved' AND lt.listing_id IN (`+ph+`) ORDER BY lt.listing_id, lt.tag`, ids...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, tag string
		if err := rows.Scan(&id, &tag); err != nil {
			rows.Close()
			return err
		}
		items[idx[id]].Tags = append(items[idx[id]].Tags, tag)
	}
	rows.Close()
	mrows, err := tx.QueryContext(ctx, mediaSelect+` WHERE network_id=? AND status='active' AND listing_id IN (`+ph+`) ORDER BY listing_id, position, id`, ids...)
	if err != nil {
		return err
	}
	defer mrows.Close()
	for mrows.Next() {
		m, err := scanMedia(mrows)
		if err != nil {
			return err
		}
		i := idx[m.ListingID]
		if items[i].Cover == nil {
			mm := m
			items[i].Cover = &mm
		}
	}
	return mrows.Err()
}
