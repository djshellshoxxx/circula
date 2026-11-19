package recurring

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Reminder struct {
	ID, Network, Owner, Template string
	Scheduled, Created           time.Time
}
type Scan struct {
	Next                      string
	Scanned, Fired, Suspended int
	Complete                  bool
}

// RunDue is explicitly invoked by a host job; importing this module never runs
// it. Scan continuation advances over every examined row (including revoked
// owners), so the first batch cannot starve later templates. Caller wraps only
// after Complete. Only today's local window is considered, never a backlog.
func (s Service) RunDue(ctx context.Context, network, after string, limit int) (Scan, error) {
	result := Scan{}
	if s.ready() != nil || !validID(network) || len(after) > 256 || limit < 1 || limit > 100 {
		return result, errors.New("invalid due scan")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id,owner FROM future_trade_templates WHERE network=? AND active=1 AND cancelled=0 AND id>? ORDER BY id LIMIT ?`, network, after, limit)
	if err != nil {
		return result, err
	}
	type item struct{ id, owner string }
	var batch []item
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.id, &i.owner); err != nil {
			rows.Close()
			return result, err
		}
		batch = append(batch, i)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, i := range batch {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		fired, suspended := false, false
		err = s.write(ctx, func(tx *sql.Tx) error {
			a := Actor{network, i.owner}
			r, e := readRecord(ctx, tx, a, i.id)
			if e != nil {
				return e
			}
			if !r.Active || r.Cancelled {
				return nil
			}
			if e = s.Auth(ctx, tx, a, r.Target, r.Amount, "fire"); e != nil {
				if !errors.Is(e, ErrDenied) {
					return e
				}
				r.Active = false
				r.Version++
				r.Updated = s.Now().UTC()
				if e = put(ctx, tx, r); e != nil {
					return e
				}
				suspended = true
				return s.audit(ctx, tx, a, r.ID, "suspended")
			}
			window, due, e := r.Calendar.Window(network, r.ID, s.Now())
			if e != nil {
				return e
			}
			if !due {
				return nil
			}
			// Editing time zones may change the local date key. A conservative
			// 20-hour guard prevents rapid refiring while preserving 23-hour
			// ordinary DST days. This supplements, never replaces, tombstones.
			var recent bool
			if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM future_template_reminders WHERE network=? AND template=? AND julianday(created)>julianday(?))`, network, r.ID, s.Now().Add(-20*time.Hour).UTC().Format(time.RFC3339Nano)).Scan(&recent); e != nil {
				return e
			}
			if recent {
				return nil
			}
			inserted, e := insertFiring(ctx, tx, network, r.ID, window, "reminder")
			if e != nil {
				return e
			}
			if !inserted {
				return nil
			}
			reminder := Reminder{window.Key, network, r.Owner, r.ID, window.Scheduled, s.Now().UTC()}
			if e = execOne(ctx, tx, `INSERT INTO future_template_reminders(network,id,owner,template,created) VALUES(?,?,?,?,?)`, network, window.Key, r.Owner, r.ID, reminder.Created.Format(time.RFC3339Nano)); e != nil {
				return e
			}
			if s.Remind != nil {
				if e = s.Remind(ctx, tx, reminder); e != nil {
					return e
				}
			}
			if e = s.audit(ctx, tx, a, r.ID, "reminder"); e != nil {
				return e
			}
			fired = true
			return nil
		})
		if err != nil {
			return result, err
		}
		result.Scanned++
		result.Next = i.id
		if fired {
			result.Fired++
		}
		if suspended {
			result.Suspended++
		}
	}
	result.Complete = len(batch) < limit
	if result.Complete {
		result.Next = ""
	}
	return result, nil
}

// Inbox exposes only authenticated owner reminder references; no target, note
// or amount is copied into the background event. Fresh policy is checked.
func (s Service) Inbox(ctx context.Context, a Actor, limit int) ([]Reminder, error) {
	out := []Reminder{}
	if !actorOK(a) || limit < 1 || limit > 100 {
		return nil, ErrDenied
	}
	err := s.write(ctx, func(tx *sql.Tx) error {
		if err := s.Auth(ctx, tx, a, "", nil, "list"); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT r.id,r.template,r.created,f.scheduled FROM future_template_reminders r JOIN future_template_firings f ON f.network=r.network AND f.template=r.template AND f.window=r.id WHERE r.network=? AND r.owner=? ORDER BY `+utcTimestampOrder("r.created")+` DESC,r.id LIMIT ?`, a.Network, a.Member, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r Reminder
			var created, scheduled string
			if err = rows.Scan(&r.ID, &r.Template, &created, &scheduled); err != nil {
				return err
			}
			r.Created, err = time.Parse(time.RFC3339Nano, created)
			if err != nil {
				return err
			}
			r.Scheduled, err = time.Parse(time.RFC3339Nano, scheduled)
			if err != nil {
				return err
			}
			r.Network = a.Network
			r.Owner = a.Member
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// Cleanup never removes firing tombstones or operation keys: deleting them
// would permit retries/replayed windows. It prunes private reminder/audit detail
// only; a separately approved archival policy must retire replay tombstones.
func (s Service) Cleanup(ctx context.Context, network string, before time.Time) error {
	if s.ready() != nil || !validID(network) || before.After(s.Now().Add(-90*24*time.Hour)) {
		return errors.New("retention minimum is 90 days")
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		cutoff := before.UTC().Format("2006-01-02T15:04:05.000000000Z")
		if err := retireResponses(ctx, tx, `network=? AND `+utcTimestampOrder("created")+`<?`, network, cutoff); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT `+recordColumns+` FROM future_trade_templates WHERE network=? AND cancelled=1`, network)
		if err != nil {
			return err
		}
		var expired []Record
		for rows.Next() {
			r, e := scanRecord(rows, network, "")
			if e != nil {
				rows.Close()
				return e
			}
			if !r.Updated.IsZero() && r.Updated.Before(before) {
				r.Note = ""
				r.Amount = nil
				expired = append(expired, r)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, r := range expired {
			if err = put(ctx, tx, r); err != nil {
				return err
			}
		}
		for _, table := range []string{"future_template_reminders", "future_template_audit"} {
			column := "created"
			if table == "future_template_audit" {
				column = "at"
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE network=? AND `+utcTimestampOrder(column)+`<?`, network, cutoff); err != nil {
				return err
			}
		}
		return nil
	})
}

// utcTimestampOrder gives module-generated canonical UTC RFC3339Nano values a
// fixed-width fractional part for exact chronological SQL ordering. Remove the
// terminal Z before padding; neither raw RFC3339Nano text nor julianday retains
// the required nanosecond ordering. This is not a parser or repair for arbitrary
// malformed database timestamps. Column names are static module-owned SQL.
func utcTimestampOrder(column string) string {
	return `substr(` + column + `,1,19)||'.'||substr(CASE WHEN substr(` + column + `,20,1)='.' THEN substr(` + column + `,21,length(` + column + `)-21) ELSE '' END||'000000000',1,9)||'Z'`
}

// Depart is an explicit trusted core-lifecycle hook, never a member endpoint.
// It cancels all related templates without a page-size limit and removes
// private reminder/operation payloads owned by the departing member. Minimal
// replay tombstones remain; no posting or historical ledger record is touched.
func (s Service) Depart(ctx context.Context, network, member string) error {
	if !validID(network) || !validID(member) {
		return ErrDenied
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `SELECT `+recordColumns+` FROM future_trade_templates WHERE network=? AND (owner=? OR target=?)`, network, member, member)
		if err != nil {
			return err
		}
		var affected []Record
		for rows.Next() {
			r, e := scanRecord(rows, network, "")
			if e != nil {
				rows.Close()
				return e
			}
			affected = append(affected, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, r := range affected {
			// Retire every cached private response for an affected template,
			// including drafts whose counterparty is the departing member.
			if err = retireResponses(ctx, tx, `network=? AND owner=? AND (json_extract(CASE WHEN json_valid(response) THEN response ELSE '{}' END,'$.ID')=? OR json_extract(CASE WHEN json_valid(response) THEN response ELSE '{}' END,'$.Template')=?)`, network, r.Owner, r.ID, r.ID); err != nil {
				return err
			}
			r.Active = false
			r.Cancelled = true
			r.Note = ""
			r.Amount = nil
			r.Version++
			r.Updated = s.Now().UTC()
			if err = put(ctx, tx, r); err != nil {
				return err
			}
			if err = s.audit(ctx, tx, Actor{network, r.Owner}, r.ID, "departure"); err != nil {
				return err
			}
		}
		// Old drafts may still reference this target after a template was edited.
		if err = retireResponses(ctx, tx, `network=? AND json_extract(CASE WHEN json_valid(response) THEN response ELSE '{}' END,'$.Target')=?`, network, member); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM future_template_reminders WHERE network=? AND owner=?`, network, member); err != nil {
			return err
		}
		return retireResponses(ctx, tx, `network=? AND owner=?`, network, member)
	})
}
