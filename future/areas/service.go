package areas

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Identity struct{ Network, Member string }

// Policy must resolve current active membership, admin/MFA authority and listing
// visibility/ownership from trusted state in tx. A nil policy fails closed.
type Policy interface {
	Check(context.Context, *sql.Tx, Identity, string, string) error
}
type Service struct {
	DB     *sql.DB
	Policy Policy
	Now    func() time.Time
}
type Record struct {
	Area
	Archived bool
	Revision int64
}
type Assignment struct {
	Network, Listing, AreaID string
	Active                   bool
	Revision                 int64
}
type Edit struct {
	ID, Label         string
	Visible, Archived bool
	Revision          int64
}
type Assign struct {
	Listing, AreaID string
	Revision        int64
}

var ErrDenied = errors.New("access denied")
var ErrConflict = errors.New("stale revision or idempotency conflict")
var ErrUnavailable = errors.New("area or listing unavailable")

func validKey(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}
func validIdentity(a Identity) bool { return validKey(a.Network) && validKey(a.Member) }
func (s Service) Migrate(ctx context.Context) error {
	if s.DB == nil {
		return errors.New("missing database")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = requireForeignKeys(ctx, tx); e != nil {
		return e
	}
	schema := []string{
		`CREATE TABLE IF NOT EXISTS future_area_schema(version INTEGER NOT NULL CHECK(version=1))`,

		`CREATE TABLE IF NOT EXISTS future_areas(network TEXT NOT NULL,id TEXT NOT NULL,label TEXT NOT NULL,visible INTEGER NOT NULL CHECK(visible IN(0,1)),archived INTEGER NOT NULL CHECK(archived IN(0,1)),revision INTEGER NOT NULL CHECK(revision>0),updated INTEGER NOT NULL,PRIMARY KEY(network,id))`,
		`CREATE TABLE IF NOT EXISTS future_listing_areas(network TEXT NOT NULL,listing TEXT NOT NULL,area_id TEXT,active INTEGER NOT NULL CHECK(active IN(0,1)),revision INTEGER NOT NULL CHECK(revision>0),updated INTEGER NOT NULL,PRIMARY KEY(network,listing),FOREIGN KEY(network,area_id) REFERENCES future_areas(network,id))`,
		`CREATE INDEX IF NOT EXISTS future_area_listing ON future_listing_areas(network,area_id,active,listing)`,
		`CREATE TABLE IF NOT EXISTS future_area_retries(network TEXT NOT NULL,member TEXT NOT NULL,key TEXT NOT NULL,fingerprint TEXT NOT NULL,result BLOB NOT NULL,created INTEGER NOT NULL,PRIMARY KEY(network,member,key))`,
		`CREATE TABLE IF NOT EXISTS future_area_network_policy(network TEXT PRIMARY KEY,share INTEGER NOT NULL CHECK(share IN(0,1)),revision INTEGER NOT NULL CHECK(revision>0))`,
		`CREATE TABLE IF NOT EXISTS future_area_audit(id INTEGER PRIMARY KEY AUTOINCREMENT,network TEXT NOT NULL,member TEXT NOT NULL,action TEXT NOT NULL,object TEXT NOT NULL,revision INTEGER NOT NULL,created INTEGER NOT NULL)`,
	}
	if e = validateSchema(ctx, tx, schema); e != nil {
		return e
	}
	for _, q := range schema {
		if _, e = tx.ExecContext(ctx, q); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO future_area_schema(version) SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM future_area_schema)`); e != nil {
		return e
	}
	var version, count int
	if e = tx.QueryRowContext(ctx, `SELECT MAX(version),count(*) FROM future_area_schema`).Scan(&version, &count); e != nil {
		return e
	}
	if version != 1 || count != 1 {
		return errors.New("unsupported area schema")
	}
	return tx.Commit()
}
func (s Service) transaction(ctx context.Context, a Identity, action, object, key string, payload any, out any, apply func(*sql.Tx) error) error {
	if s.DB == nil || s.Policy == nil || s.Now == nil || !validIdentity(a) || !validKey(key) {
		return errors.New("invalid service request")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = requireForeignKeys(ctx, tx); e != nil {
		return e
	}
	if e = s.Policy.Check(ctx, tx, a, action, object); e != nil {
		return ErrDenied
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if input, ok := payload.(Assign); ok && input.AreaID != "" {
		var visible, archived bool
		if e = tx.QueryRowContext(ctx, `SELECT visible,archived FROM future_areas WHERE network=? AND id=?`, a.Network, input.AreaID).Scan(&visible, &archived); e != nil || !visible || archived {
			return ErrUnavailable
		}
	}
	body, e := json.Marshal(struct {
		Action  string
		Payload any
	}{action, payload})
	if e != nil {
		return e
	}
	sum := sha256.Sum256(body)
	fp := hex.EncodeToString(sum[:])
	var old string
	var encoded []byte
	e = tx.QueryRowContext(ctx, `SELECT fingerprint,result FROM future_area_retries WHERE network=? AND member=? AND key=?`, a.Network, a.Member, key).Scan(&old, &encoded)
	if e == nil {
		if old != fp {
			return ErrConflict
		}
		if string(encoded) == "null" {
			return ErrUnavailable
		}
		if len(encoded) > 65536 {
			return errors.New("invalid stored retry")
		}
		if e = strictJSON(encoded, out); e != nil {
			return e
		}
		if e = validateReplay(a, object, payload, out); e != nil {
			return e
		}
		if reply, ok := out.(*Assignment); ok {
			var currentArea sql.NullString
			var active bool
			var revision int64
			if e = tx.QueryRowContext(ctx, `SELECT area_id,active,revision FROM future_listing_areas WHERE network=? AND listing=?`, a.Network, object).Scan(&currentArea, &active, &revision); e != nil {
				return ErrUnavailable
			}
			if currentArea.String != reply.AreaID || active != reply.Active || revision != reply.Revision {
				return ErrConflict
			}
		}
		if reply, ok := out.(*Sharing); ok {
			var current Sharing
			if e = tx.QueryRowContext(ctx, `SELECT share,revision FROM future_area_network_policy WHERE network=?`, a.Network).Scan(&current.Enabled, &current.Revision); e != nil {
				return ErrUnavailable
			}
			if current != *reply {
				return ErrConflict
			}
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		return tx.Commit()
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	var retryCount int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM future_area_retries WHERE network=? AND member=?`, a.Network, a.Member).Scan(&retryCount); e != nil {
		return e
	}
	if retryCount >= 10000 {
		return errors.New("idempotency quota reached; archive feature data through host retention policy")
	}
	if e = apply(tx); e != nil {
		return e
	}
	encoded, e = json.Marshal(out)
	if e != nil {
		return e
	}
	if _, e = execOne(ctx, tx, `INSERT INTO future_area_retries(network,member,key,fingerprint,result,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, key, fp, encoded, s.Now().Unix()); e != nil {
		return e
	}
	return tx.Commit()
}
func (s Service) Edit(ctx context.Context, a Identity, key string, input Edit) (Record, error) {
	var out Record
	if !validKey(input.ID) || input.Revision < 0 || input.Revision == int64(^uint64(0)>>1) {
		return out, errors.New("invalid area edit")
	}
	candidate := Area{a.Network, input.ID, strings.TrimSpace(input.Label), input.Visible}
	if e := Validate(candidate); e != nil {
		return out, e
	}
	e := s.transaction(ctx, a, "admin", input.ID, key, input, &out, func(tx *sql.Tx) error {
		var prior int64
		err := tx.QueryRowContext(ctx, `SELECT revision FROM future_areas WHERE network=? AND id=?`, a.Network, input.ID).Scan(&prior)
		if errors.Is(err, sql.ErrNoRows) {
			if input.Revision != 0 {
				return ErrConflict
			}
			var count int
			if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM future_areas WHERE network=?`, a.Network).Scan(&count); err != nil {
				return err
			}
			if count >= 1000 {
				return errors.New("area quota reached")
			}
			_, err = execOne(ctx, tx, `INSERT INTO future_areas(network,id,label,visible,archived,revision,updated) VALUES(?,?,?,?,?,1,?)`, a.Network, input.ID, candidate.Label, input.Visible, input.Archived, s.Now().Unix())
		} else if err == nil {
			if prior != input.Revision {
				return ErrConflict
			}
			_, err = execOne(ctx, tx, `UPDATE future_areas SET label=?,visible=?,archived=?,revision=revision+1,updated=? WHERE network=? AND id=? AND revision=?`, candidate.Label, input.Visible, input.Archived, s.Now().Unix(), a.Network, input.ID, input.Revision)
		}
		if err != nil {
			return err
		}
		out = Record{candidate, input.Archived, prior + 1}
		if input.Archived {
			if _, err = tx.ExecContext(ctx, `UPDATE future_listing_areas SET active=0,revision=revision+1,updated=? WHERE network=? AND area_id=? AND active=1`, s.Now().Unix(), a.Network, input.ID); err != nil {
				return err
			}
			var remaining int
			if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM future_listing_areas WHERE network=? AND area_id=? AND active=1`, a.Network, input.ID).Scan(&remaining); err != nil {
				return err
			}
			if remaining != 0 {
				return errors.New("archive association cleanup incomplete")
			}
		}
		_, err = execOne(ctx, tx, `INSERT INTO future_area_audit(network,member,action,object,revision,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, "edit", input.ID, out.Revision, s.Now().Unix())
		return err
	})
	if e != nil {
		return Record{}, e
	}
	return out, nil
}
func (s Service) Assign(ctx context.Context, a Identity, key string, input Assign) (Assignment, error) {
	out := Assignment{}
	if !validKey(input.Listing) || (input.AreaID != "" && !validKey(input.AreaID)) || input.Revision < 0 || input.Revision == int64(^uint64(0)>>1) {
		return out, errors.New("invalid assignment")
	}
	e := s.transaction(ctx, a, "assign", input.Listing, key, input, &out, func(tx *sql.Tx) error {
		if input.AreaID != "" {
			var visible, archived bool
			if e := tx.QueryRowContext(ctx, `SELECT visible,archived FROM future_areas WHERE network=? AND id=?`, a.Network, input.AreaID).Scan(&visible, &archived); e != nil || !visible || archived {
				return ErrUnavailable
			}
		}
		var prior int64
		e := tx.QueryRowContext(ctx, `SELECT revision FROM future_listing_areas WHERE network=? AND listing=?`, a.Network, input.Listing).Scan(&prior)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if prior != input.Revision {
			return ErrConflict
		}
		var area any
		if input.AreaID != "" {
			area = input.AreaID
		}
		_, e = execOne(ctx, tx, `INSERT INTO future_listing_areas(network,listing,area_id,active,revision,updated) VALUES(?,?,?,?,1,?) ON CONFLICT(network,listing) DO UPDATE SET area_id=excluded.area_id,active=excluded.active,revision=future_listing_areas.revision+1,updated=excluded.updated`, a.Network, input.Listing, area, input.AreaID != "", s.Now().Unix())
		if e != nil {
			return e
		}
		out = Assignment{a.Network, input.Listing, input.AreaID, input.AreaID != "", prior + 1}
		_, e = execOne(ctx, tx, `INSERT INTO future_area_audit(network,member,action,object,revision,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, "assign", input.Listing, out.Revision, s.Now().Unix())
		return e
	})
	if e != nil {
		return Assignment{}, e
	}
	return out, nil
}
func (s Service) Catalog(ctx context.Context, a Identity) ([]Record, error) {
	return s.catalog(ctx, a, false)
}
func (s Service) AdminCatalog(ctx context.Context, a Identity) ([]Record, error) {
	return s.catalog(ctx, a, true)
}
func (s Service) catalog(ctx context.Context, a Identity, admin bool) ([]Record, error) {
	if s.DB == nil || s.Policy == nil || !validIdentity(a) {
		return nil, errors.New("invalid service context")
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	action := "read"
	if admin {
		action = "admin"
	}
	if e = s.Policy.Check(ctx, tx, a, action, ""); e != nil {
		return nil, e
	}
	query := `SELECT id,label,visible,archived,revision FROM future_areas WHERE network=?`
	if !admin {
		query += ` AND visible=1 AND archived=0`
	}
	query += ` ORDER BY label,id LIMIT 1001`
	rows, e := tx.QueryContext(ctx, query, a.Network)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r := Record{Area: Area{Network: a.Network}}
		if e = rows.Scan(&r.ID, &r.Label, &r.Visible, &r.Archived, &r.Revision); e != nil {
			return nil, e
		}
		if r.Revision < 1 {
			return nil, errors.New("invalid stored revision")
		}
		if e = Validate(r.Area); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	if len(out) > 1000 {
		return nil, errors.New("catalog quota exceeded")
	}
	return out, rows.Err()
}

// Filter returns only freshly authorized listing IDs, never addresses, owner IDs,
// coordinates or inferred positions. Paging is deliberately bounded to 1000 IDs.
func (s Service) Filter(ctx context.Context, a Identity, area string) ([]string, error) {
	if !validKey(area) || s.DB == nil || s.Policy == nil || !validIdentity(a) {
		return nil, errors.New("invalid area filter")
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if e = s.Policy.Check(ctx, tx, a, "read", ""); e != nil {
		return nil, e
	}
	var visible, archived bool
	if e = tx.QueryRowContext(ctx, `SELECT visible,archived FROM future_areas WHERE network=? AND id=?`, a.Network, area).Scan(&visible, &archived); e != nil || !visible || archived {
		return nil, ErrUnavailable
	}
	rows, e := tx.QueryContext(ctx, `SELECT listing FROM future_listing_areas WHERE network=? AND area_id=? AND active=1 ORDER BY listing LIMIT 1001`, a.Network, area)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if len(ids) > 1000 {
		return nil, errors.New("filter too large; host must provide cursor paging")
	}
	out := []string{}
	for _, id := range ids {
		if s.Policy.Check(ctx, tx, a, "listing", id) == nil {
			out = append(out, id)
		}
		if e = ctx.Err(); e != nil {
			return nil, e
		}
	}
	return out, nil
}

type Sharing struct {
	Enabled  bool
	Revision int64
}

func (s Service) SetSharing(ctx context.Context, a Identity, key string, enabled bool, revision int64) (Sharing, error) {
	out := Sharing{}
	if revision < 0 || revision == int64(^uint64(0)>>1) {
		return out, errors.New("invalid sharing revision")
	}
	e := s.transaction(ctx, a, "admin", "sharing", key, struct {
		Enabled  bool
		Revision int64
	}{enabled, revision}, &out, func(tx *sql.Tx) error {
		var prior int64
		e := tx.QueryRowContext(ctx, `SELECT revision FROM future_area_network_policy WHERE network=?`, a.Network).Scan(&prior)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if prior != revision {
			return ErrConflict
		}
		_, e = execOne(ctx, tx, `INSERT INTO future_area_network_policy(network,share,revision) VALUES(?,?,1) ON CONFLICT(network) DO UPDATE SET share=excluded.share,revision=future_area_network_policy.revision+1`, a.Network, enabled)
		out = Sharing{enabled, prior + 1}
		return e
	})
	if e != nil {
		return Sharing{}, e
	}
	return out, nil
}

// ExportSharing consults the durable per-network opt-in. It never exports
// listing associations, member IDs, addresses or coordinates.
func (s Service) ExportSharing(ctx context.Context, a Identity) ([]Area, error) {
	if s.DB == nil || s.Policy == nil || !validIdentity(a) {
		return nil, errors.New("invalid export context")
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if e = s.Policy.Check(ctx, tx, a, "read", ""); e != nil {
		return nil, e
	}
	var enabled bool
	e = tx.QueryRowContext(ctx, `SELECT share FROM future_area_network_policy WHERE network=?`, a.Network).Scan(&enabled)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	out := []Area{}
	if !enabled {
		return out, nil
	}
	rows, e := tx.QueryContext(ctx, `SELECT id,label FROM future_areas WHERE network=? AND visible=1 AND archived=0 ORDER BY label,id LIMIT 1001`, a.Network)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		a := Area{Network: a.Network, Visible: true}
		if e = rows.Scan(&a.ID, &a.Label); e != nil {
			return nil, e
		}
		if e = Validate(a); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	if len(out) > 1000 {
		return nil, errors.New("catalog quota exceeded")
	}
	return out, rows.Err()
}

// CleanupListing removes a current association after fresh host ownership or
// departure authority. Historical audit entries remain, without a public label.
func (s Service) CleanupListing(ctx context.Context, a Identity, key, listing string) error {
	if s.DB == nil || s.Policy == nil || s.Now == nil || !validIdentity(a) || !validKey(key) || !validKey(listing) {
		return errors.New("invalid cleanup request")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = requireForeignKeys(ctx, tx); e != nil {
		return e
	}
	if e = s.Policy.Check(ctx, tx, a, "cleanup", listing); e != nil {
		return ErrDenied
	}
	result, e := tx.ExecContext(ctx, `DELETE FROM future_listing_areas WHERE network=? AND listing=?`, a.Network, listing)
	if e != nil {
		return e
	}
	var remaining int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM future_listing_areas WHERE network=? AND listing=?`, a.Network, listing).Scan(&remaining); e != nil {
		return e
	}
	if remaining != 0 {
		return errors.New("cleanup deletion incomplete")
	}
	removed, e := result.RowsAffected()
	if e != nil {
		return e
	}
	// Retire cached coarse-location replies but retain fingerprints as replay tombstones.
	const refs = `network=? AND CASE WHEN json_valid(result) THEN json_extract(result,'$.Listing') END=?`
	scrub, e := tx.ExecContext(ctx, `UPDATE future_area_retries SET result='null' WHERE `+refs, a.Network, listing)
	if e != nil {
		return e
	}
	scrubbed, e := scrub.RowsAffected()
	if e != nil {
		return e
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM future_area_retries WHERE `+refs, a.Network, listing).Scan(&remaining); e != nil {
		return e
	}
	if remaining != 0 {
		return errors.New("cleanup cached reply retirement incomplete")
	}
	if removed > 0 || scrubbed > 0 {
		if _, e = execOne(ctx, tx, `INSERT INTO future_area_audit(network,member,action,object,revision,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, "cleanup", listing, 0, s.Now().Unix()); e != nil {
			return e
		}
	}
	return tx.Commit()
}

// GetSharing is admin-only, since federation policy is a network decision.
func (s Service) GetSharing(ctx context.Context, a Identity) (Sharing, error) {
	out := Sharing{}
	if s.DB == nil || s.Policy == nil || !validIdentity(a) {
		return out, errors.New("invalid sharing context")
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = s.Policy.Check(ctx, tx, a, "admin", ""); e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, `SELECT share,revision FROM future_area_network_policy WHERE network=?`, a.Network).Scan(&out.Enabled, &out.Revision)
	if errors.Is(e, sql.ErrNoRows) {
		e = nil
	}
	if e != nil {
		return Sharing{}, e
	}
	return out, nil
}

// Current returns only an owner's current association and revision. Hidden or
// archived taxonomy is projected as unset without exposing its historical label.
func (s Service) Current(ctx context.Context, a Identity, listing string) (Assignment, error) {
	out := Assignment{Network: a.Network, Listing: listing}
	if s.DB == nil || s.Policy == nil || !validIdentity(a) || !validKey(listing) {
		return out, errors.New("invalid association context")
	}
	tx, e := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = s.Policy.Check(ctx, tx, a, "assign", listing); e != nil {
		return out, ErrDenied
	}
	var area sql.NullString
	var active bool
	e = tx.QueryRowContext(ctx, `SELECT area_id,active,revision FROM future_listing_areas WHERE network=? AND listing=?`, a.Network, listing).Scan(&area, &active, &out.Revision)
	if errors.Is(e, sql.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	if active && area.Valid {
		var visible, archived bool
		e = tx.QueryRowContext(ctx, `SELECT visible,archived FROM future_areas WHERE network=? AND id=?`, a.Network, area.String).Scan(&visible, &archived)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return out, e
		}
		if e == nil && visible && !archived {
			out.AreaID = area.String
			out.Active = true
		}
	}
	return out, nil
}

// DisableSharing is a privacy safety action. It remains available after retry
// quota exhaustion and allocates no retry record; repeating it is a no-op.
func (s Service) DisableSharing(ctx context.Context, a Identity) (Sharing, error) {
	out := Sharing{}
	if s.DB == nil || s.Policy == nil || s.Now == nil || !validIdentity(a) {
		return out, errors.New("invalid sharing context")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = requireForeignKeys(ctx, tx); e != nil {
		return out, e
	}
	if e = s.Policy.Check(ctx, tx, a, "admin", "sharing"); e != nil {
		return out, ErrDenied
	}
	if e = ctx.Err(); e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, `SELECT share,revision FROM future_area_network_policy WHERE network=?`, a.Network).Scan(&out.Enabled, &out.Revision)
	if errors.Is(e, sql.ErrNoRows) {
		return Sharing{}, nil
	}
	if e != nil {
		return Sharing{}, e
	}
	if !out.Enabled {
		return out, nil
	}
	if out.Revision < 1 || out.Revision == int64(^uint64(0)>>1) {
		return Sharing{}, errors.New("invalid stored sharing revision")
	}
	if _, e = execOne(ctx, tx, `UPDATE future_area_network_policy SET share=0,revision=revision+1 WHERE network=? AND revision=?`, a.Network, out.Revision); e != nil {
		return Sharing{}, e
	}
	out.Enabled = false
	out.Revision++
	if _, e = execOne(ctx, tx, `INSERT INTO future_area_audit(network,member,action,object,revision,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, "disable-sharing", "sharing", out.Revision, s.Now().Unix()); e != nil {
		return Sharing{}, e
	}
	if e = tx.Commit(); e != nil {
		return Sharing{}, e
	}
	return out, nil
}
