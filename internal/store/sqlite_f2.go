package store

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	sqlite "modernc.org/sqlite"
)

// circula_fold(text) is the SQL form of domain.FoldName: case and accent
// folding for directory search. SQLite's own lower() only folds ASCII.
func init() {
	sqlite.MustRegisterDeterministicScalarFunction("circula_fold", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			switch v := args[0].(type) {
			case string:
				return domain.FoldName(v), nil
			case []byte:
				return domain.FoldName(string(v)), nil
			default:
				return v, nil
			}
		})
}

func (t *txImpl) TimeZone(personID string) (string, error) {
	var z string
	err := t.row(`SELECT time_zone FROM person_prefs WHERE person_id = ?`, personID).Scan(&z)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return z, mapErr(err)
}

func (t *txImpl) SetTimeZone(personID, zone string, at time.Time) error {
	_, err := t.exec(`INSERT INTO person_prefs(person_id,time_zone,updated_at) VALUES (?,?,?)
		ON CONFLICT(person_id) DO UPDATE SET time_zone = excluded.time_zone, updated_at = excluded.updated_at`,
		personID, zone, ms(at))
	return err
}

const announcementCols = `id,network_id,title,body,created_by,published_at,effective_at,deleted_at,deleted_by,supersedes_id`

func scanAnnouncement(r rowScanner) (*domain.Announcement, error) {
	var a domain.Announcement
	var pub, eff int64
	var del sql.NullInt64
	if err := r.Scan(&a.ID, &a.NetworkID, &a.Title, &a.Body, &a.CreatedBy, &pub, &eff, &del, &a.DeletedBy, &a.SupersedesID); err != nil {
		return nil, err
	}
	a.PublishedAt, a.EffectiveAt, a.DeletedAt = fromMS(pub), fromMS(eff), ptrTime(del)
	return &a, nil
}

func (t *txImpl) InsertAnnouncement(a *domain.Announcement) error {
	_, err := t.exec(`INSERT INTO announcements(`+announcementCols+`) VALUES (?,?,?,?,?,?,?,NULL,'',?)`,
		a.ID, a.NetworkID, a.Title, a.Body, a.CreatedBy, ms(a.PublishedAt), ms(a.EffectiveAt), a.SupersedesID)
	return err
}

func (t *txImpl) GetAnnouncement(netID, id string) (*domain.Announcement, error) {
	a, err := scanAnnouncement(t.row(`SELECT `+announcementCols+` FROM announcements WHERE network_id = ? AND id = ?`, netID, id))
	return a, notFound(err, "announcement")
}

func (t *txImpl) AnnouncementSuperseded(netID, id string) (bool, error) {
	var n int
	err := mapErr(t.row(`SELECT COUNT(*) FROM announcements WHERE network_id = ? AND supersedes_id = ? AND deleted_at IS NULL`, netID, id).Scan(&n))
	return n > 0, err
}

func (t *txImpl) RetractAnnouncement(netID, id, by string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE announcements SET deleted_at = ?, deleted_by = ?
		WHERE network_id = ? AND id = ? AND deleted_at IS NULL`, ms(at), by, netID, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (t *txImpl) ListAnnouncements(netID string, f AnnouncementFilter) ([]*domain.Announcement, error) {
	q := `SELECT ` + announcementCols + ` FROM announcements WHERE network_id = ?`
	args := []any{netID}
	if !f.IncludeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	if !f.SupersededBefore.IsZero() {
		q += ` AND NOT EXISTS (SELECT 1 FROM announcements s WHERE s.network_id = announcements.network_id
			AND s.supersedes_id = announcements.id AND s.deleted_at IS NULL AND s.effective_at <= ?)`
		args = append(args, ms(f.SupersededBefore))
	}
	if !f.EffectiveBy.IsZero() {
		q += ` AND effective_at <= ?`
		args = append(args, ms(f.EffectiveBy))
	}
	if f.AfterID != "" {
		q += ` AND id < ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limitOf(f.Limit))
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Announcement
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (t *txImpl) OverviewCounts(netID string, p OverviewParams) (OverviewCounts, error) {
	var c OverviewCounts
	now := ms(p.Now)
	count := func(dst *int, q string, args ...any) error {
		return mapErr(t.row(q, args...).Scan(dst))
	}
	if err := count(&c.StaleProposals, `SELECT COUNT(*) FROM transactions WHERE network_id = ? AND kind = 'trade'
		AND status = 'proposed' AND proposed_at <= ? AND (expires_at IS NULL OR expires_at > ?)`,
		netID, ms(p.Now.Add(-p.StaleAfter)), now); err != nil {
		return c, err
	}
	if err := count(&c.OpenDisputes, `SELECT COUNT(*) FROM disputes WHERE network_id = ? AND status IN ('open','under_review')`, netID); err != nil {
		return c, err
	}
	if err := count(&c.PostingPending, `SELECT COUNT(*) FROM disputes WHERE network_id = ? AND status = 'posting_pending'`, netID); err != nil {
		return c, err
	}
	if err := count(&c.NearLimit, `SELECT COUNT(*) FROM accounts a
		JOIN members m ON m.network_id = a.network_id AND m.id = a.member_id
		WHERE a.network_id = ? AND a.is_system = 0 AND m.status = 'active' AND a.balance < 0
		AND -a.balance >= CASE WHEN a.limit_override IS NOT NULL
			AND (a.limit_override_until IS NULL OR a.limit_override_until > ?)
			THEN a.limit_override ELSE ? END`, netID, now, p.DefaultLimit); err != nil {
		return c, err
	}
	if err := count(&c.ExpiringInvites, `SELECT COUNT(*) FROM invitations WHERE network_id = ?
		AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ? AND expires_at <= ?`,
		netID, now, ms(p.Now.Add(p.InviteWithin))); err != nil {
		return c, err
	}
	var run ConsistencyRun
	var started, finished int64
	var ok int
	err := t.row(`SELECT id,network_id,started_at,finished_at,ok,findings FROM consistency_runs
		WHERE network_id = ? ORDER BY finished_at DESC, id DESC LIMIT 1`, netID).
		Scan(&run.ID, &run.NetworkID, &started, &finished, &ok, &run.Findings)
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return c, mapErr(err)
	default:
		run.Started, run.Finished, run.OK = fromMS(started), fromMS(finished), ok == 1
		c.LastCheck = &run
	}
	return c, nil
}
