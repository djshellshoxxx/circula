package recurring

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrDenied = errors.New("recurring access denied")
var ErrConflict = errors.New("recurring revision or idempotency conflict")
var ErrStorage = errors.New("recurring storage invariant failed")

type Actor struct{ Network, Member string }
type Record struct {
	ID, Network, Owner, Target, Note string
	Amount                           *int64
	Calendar                         Calendar
	Active, Cancelled                bool
	Version                          int64
	Created, Updated                 time.Time
}
type Draft struct {
	Intent, Template, Target, Note string
	Amount                         *int64
	Created                        time.Time
}

// Authorize must check current network/member/target activity and, for draft,
// current terms and capacity using the supplied transaction. It cannot be a
// boolean copied from HTTP input. A membership denial must return ErrDenied.
type Authorize func(context.Context, *sql.Tx, Actor, string, *int64, string) error

// Service opens no connection and starts no job. The supplied SQLite connection
// must use foreign_keys and a bounded busy_timeout on every connection.
type Service struct {
	DB      *sql.DB
	Auth    Authorize
	Now     func() time.Time
	Enabled bool
	Remind  func(context.Context, *sql.Tx, Reminder) error
}

func validID(s string) bool {
	return s != "" && len(s) <= 256 && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func actorOK(a Actor) bool { return validID(a.Network) && validID(a.Member) }
func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func digest(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func (s Service) ready() error {
	if !s.Enabled || s.DB == nil || s.Auth == nil || s.Now == nil {
		return ErrDenied
	}
	return nil
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS future_recurring_lock(id INTEGER PRIMARY KEY CHECK(id=1),v INTEGER NOT NULL)`,
	`INSERT OR IGNORE INTO future_recurring_lock VALUES(1,0)`,
	`CREATE TABLE IF NOT EXISTS future_trade_templates(network TEXT NOT NULL,id TEXT NOT NULL,owner TEXT NOT NULL,target TEXT NOT NULL,active INTEGER NOT NULL CHECK(active IN(0,1)),cancelled INTEGER NOT NULL CHECK(cancelled IN(0,1)),version INTEGER NOT NULL CHECK(version>0),payload TEXT NOT NULL,PRIMARY KEY(network,id))`,
	`CREATE INDEX IF NOT EXISTS future_template_owner ON future_trade_templates(network,owner,id)`,
	`CREATE INDEX IF NOT EXISTS future_template_due ON future_trade_templates(network,active,id)`,
	`CREATE TABLE IF NOT EXISTS future_template_firings(network TEXT NOT NULL,template TEXT NOT NULL,window TEXT NOT NULL,scheduled TEXT NOT NULL,state TEXT NOT NULL CHECK(state IN('reminder','skipped')),PRIMARY KEY(network,template,window),FOREIGN KEY(network,template) REFERENCES future_trade_templates(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_template_reminders(network TEXT NOT NULL,id TEXT NOT NULL,owner TEXT NOT NULL,template TEXT NOT NULL,created TEXT NOT NULL,PRIMARY KEY(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_template_operations(network TEXT NOT NULL,owner TEXT NOT NULL,key TEXT NOT NULL,fingerprint TEXT NOT NULL,response TEXT NOT NULL,created TEXT NOT NULL,PRIMARY KEY(network,owner,key))`,
	`CREATE TABLE IF NOT EXISTS future_template_audit(seq INTEGER PRIMARY KEY AUTOINCREMENT,network TEXT NOT NULL,owner TEXT NOT NULL,template TEXT NOT NULL,action TEXT NOT NULL,at TEXT NOT NULL)`,
}

func (s Service) Migrate(ctx context.Context) error {
	if s.DB == nil {
		return errors.New("missing database")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range schemaStatements {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	if err = checkSchema(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
func (s Service) write(ctx context.Context, fn func(*sql.Tx) error) error {
	if err := s.ready(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = execOne(ctx, tx, `UPDATE future_recurring_lock SET v=v WHERE id=1`); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

const recordColumns = "owner,id,target,active,cancelled,version,payload"

type recordScanner interface{ Scan(...any) error }

func scanRecord(scanner recordScanner, network, expectedOwner string) (Record, error) {
	var r Record
	var owner, id, target, payload string
	var active, cancelled bool
	var version int64
	if err := scanner.Scan(&owner, &id, &target, &active, &cancelled, &version, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrDenied
		}
		return r, err
	}
	if err := decodeJSON([]byte(payload), &r); err != nil {
		return Record{}, ErrStorage
	}
	if !validID(owner) || !validID(id) || expectedOwner != "" && owner != expectedOwner || r.Network != network || r.Owner != owner || r.ID != id || r.Target != target || r.Active != active || r.Cancelled != cancelled || r.Version != version || r.Target == r.Owner || version < 1 || r.Created.IsZero() || r.Updated.IsZero() || validate(r) != nil {
		return Record{}, ErrStorage
	}
	return r, nil
}
func readRecord(ctx context.Context, tx *sql.Tx, a Actor, id string) (Record, error) {
	return scanRecord(tx.QueryRowContext(ctx, `SELECT `+recordColumns+` FROM future_trade_templates WHERE network=? AND owner=? AND id=?`, a.Network, a.Member, id), a.Network, a.Member)
}
func put(ctx context.Context, tx *sql.Tx, r Record) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	err = execOne(ctx, tx, `INSERT INTO future_trade_templates(network,id,owner,target,active,cancelled,version,payload) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(network,id) DO UPDATE SET target=excluded.target,active=excluded.active,cancelled=excluded.cancelled,version=excluded.version,payload=excluded.payload`, r.Network, r.ID, r.Owner, r.Target, r.Active, r.Cancelled, r.Version, string(payload))
	return err
}
func (s Service) audit(ctx context.Context, tx *sql.Tx, a Actor, id, action string) error {
	err := execOne(ctx, tx, `INSERT INTO future_template_audit(network,owner,template,action,at) VALUES(?,?,?,?,?)`, a.Network, a.Member, id, action, s.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func validate(r Record) error {
	if !validID(r.Target) || !utf8.ValidString(r.Note) || len(r.Note) > 1024 || r.Amount != nil && (*r.Amount <= 0 || *r.Amount > 1<<50) {
		return errors.New("invalid template fields")
	}
	_, err := r.Calendar.parse()
	return err
}
func opRead(ctx context.Context, tx *sql.Tx, a Actor, key, fp string, out any) (bool, error) {
	if !validID(key) {
		return false, errors.New("invalid idempotency key")
	}
	var stored, payload string
	err := tx.QueryRowContext(ctx, `SELECT fingerprint,response FROM future_template_operations WHERE network=? AND owner=? AND key=?`, a.Network, a.Member, key).Scan(&stored, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if payload == "" {
		return false, ErrConflict
	}
	if fp != stored {
		return false, ErrConflict
	}
	if err = decodeJSON([]byte(payload), out); err != nil {
		return false, ErrStorage
	}
	switch value := out.(type) {
	case *Record:
		if value.Network != a.Network || value.Owner != a.Member || !validID(value.ID) || value.Version < 1 || value.Target == value.Owner || value.Created.IsZero() || value.Updated.IsZero() || validate(*value) != nil {
			return false, ErrStorage
		}
	case *Draft:
		if !validID(value.Intent) || !validID(value.Template) || !validID(value.Target) || value.Target == a.Member || !utf8.ValidString(value.Note) || len(value.Note) > 1024 || value.Amount != nil && (*value.Amount <= 0 || *value.Amount > 1<<50) || value.Created.IsZero() {
			return false, ErrStorage
		}
	}
	return true, nil
}
func (s Service) opPut(ctx context.Context, tx *sql.Tx, a Actor, key, fp string, response any) error {
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}
	err = execOne(ctx, tx, `INSERT INTO future_template_operations(network,owner,key,fingerprint,response,created) VALUES(?,?,?,?,?,?)`, a.Network, a.Member, key, fp, string(b), s.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// Save supports create (ID empty, expected=0) and optimistic edits. Network,
// owner/version/cancelled values are server-owned; clients cannot supply them.
func (s Service) Save(ctx context.Context, a Actor, input Record, expected int64, key string) (Record, error) {
	var out Record
	if !actorOK(a) || (input.ID != "" && !validID(input.ID)) || expected < 0 || validate(input) != nil || input.Target == a.Member || input.Network != "" || input.Owner != "" || input.Version != 0 || input.Cancelled || !input.Created.IsZero() || !input.Updated.IsZero() {
		return out, errors.New("invalid template request")
	}
	fp := digest(struct {
		Action   string
		Input    Record
		Expected int64
	}{"save", input, expected})
	err := s.write(ctx, func(tx *sql.Tx) error {
		if err := s.Auth(ctx, tx, a, input.Target, input.Amount, "save"); err != nil {
			return err
		}
		found, err := opRead(ctx, tx, a, key, fp, &out)
		if err != nil || found {
			return err
		}
		r := input
		r.Network = a.Network
		r.Owner = a.Member
		r.Version = 1
		r.Created = s.Now().UTC()
		r.Updated = r.Created
		if r.ID == "" {
			if expected != 0 {
				return ErrConflict
			}
			var count int
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_trade_templates WHERE network=? AND owner=? AND cancelled=0`, a.Network, a.Member).Scan(&count); err != nil {
				return err
			}
			if count >= 20 {
				return errors.New("template quota reached")
			}
			r.ID, err = randomID()
			if err != nil {
				return err
			}
		} else {
			old, e := readRecord(ctx, tx, a, r.ID)
			if e != nil {
				return e
			}
			if old.Version != expected || old.Cancelled {
				return ErrConflict
			}
			r.Version = old.Version + 1
			r.Created = old.Created
		}
		if err = put(ctx, tx, r); err != nil {
			return err
		}
		if err = s.audit(ctx, tx, a, r.ID, "save"); err != nil {
			return err
		}
		out = r
		return s.opPut(ctx, tx, a, key, fp, out)
	})
	return out, err
}

// Get performs a direct scoped lookup; callers need not page through cancelled
// history to preview a currently owned template.
func (s Service) Get(ctx context.Context, a Actor, id string) (Record, error) {
	var out Record
	if !actorOK(a) || !validID(id) {
		return out, ErrDenied
	}
	err := s.write(ctx, func(tx *sql.Tx) error {
		r, err := readRecord(ctx, tx, a, id)
		if err != nil {
			return err
		}
		if err = s.Auth(ctx, tx, a, r.Target, r.Amount, "read"); err != nil {
			return err
		}
		out = r
		return nil
	})
	return out, err
}

// List always rechecks current membership and target policy. A revoked target is
// suppressed without exposing a reason. Cursor is an opaque template ID scoped
// by SQL to this owner; it is not an authorization token.
func (s Service) List(ctx context.Context, a Actor, after string, limit int) ([]Record, string, error) {
	if !actorOK(a) || len(after) > 256 || limit < 1 || limit > 100 {
		return nil, "", errors.New("invalid list request")
	}
	out := []Record{}
	next := ""
	err := s.write(ctx, func(tx *sql.Tx) error {
		if err := s.Auth(ctx, tx, a, "", nil, "list"); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT `+recordColumns+` FROM future_trade_templates WHERE network=? AND owner=? AND id>? ORDER BY id LIMIT ?`, a.Network, a.Member, after, limit)
		if err != nil {
			return err
		}
		var all []Record
		for rows.Next() {
			r, e := scanRecord(rows, a.Network, a.Member)
			if e != nil {
				rows.Close()
				return e
			}
			all = append(all, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, r := range all {
			next = r.ID
			if err = s.Auth(ctx, tx, a, r.Target, r.Amount, "read"); errors.Is(err, ErrDenied) {
				continue
			} else if err != nil {
				return err
			}
			out = append(out, r)
		}
		return nil
	})
	return out, next, err
}

// Change mutates explicit state with an optimistic revision and durable retry key.
func (s Service) Change(ctx context.Context, a Actor, id, action string, expected int64, key string) (Record, error) {
	var out Record
	if !actorOK(a) || !validID(id) {
		return out, ErrDenied
	}
	fp := digest([]any{"change", id, action, expected})
	err := s.write(ctx, func(tx *sql.Tx) error {
		r, err := readRecord(ctx, tx, a, id)
		if err != nil {
			return err
		}
		authTarget := r.Target
		authAmount := r.Amount
		if action == "pause" || action == "cancel" || action == "skip" {
			authTarget = ""
			authAmount = nil
		}
		if err = s.Auth(ctx, tx, a, authTarget, authAmount, action); err != nil {
			return err
		}
		found, err := opRead(ctx, tx, a, key, fp, &out)
		if err != nil || found {
			return err
		}
		if r.Version != expected || r.Cancelled {
			return ErrConflict
		}
		switch action {
		case "pause":
			r.Active = false
		case "resume":
			r.Active = true
		case "cancel":
			r.Active = false
			r.Cancelled = true
		case "skip":
			window, ok, e := r.Calendar.Next(a.Network, r.ID, s.Now())
			if e != nil {
				return e
			}
			if !ok {
				return errors.New("no due occurrence to skip")
			}
			if _, err = insertFiring(ctx, tx, a.Network, id, window, "skipped"); err != nil {
				return err
			}
		default:
			return errors.New("invalid template action")
		}
		r.Version++
		r.Updated = s.Now().UTC()
		if err = put(ctx, tx, r); err != nil {
			return err
		}
		if err = s.audit(ctx, tx, a, id, action); err != nil {
			return err
		}
		out = r
		return s.opPut(ctx, tx, a, key, fp, out)
	})
	return out, err
}

// CreateDraft is a deliberate member action. It persists only a prefill and a
// random intent, never a proposal, acceptance or ledger entry.
func (s Service) CreateDraft(ctx context.Context, a Actor, id, key string) (Draft, error) {
	var out Draft
	if !actorOK(a) || !validID(id) {
		return out, ErrDenied
	}
	err := s.write(ctx, func(tx *sql.Tx) error {
		r, err := readRecord(ctx, tx, a, id)
		if err != nil {
			return err
		}
		if err = s.Auth(ctx, tx, a, r.Target, r.Amount, "draft"); err != nil {
			return err
		}
		if r.Cancelled {
			return ErrDenied
		}
		fp := digest([]any{"draft", id})
		found, err := opRead(ctx, tx, a, key, fp, &out)
		if err != nil {
			return err
		}
		if found {
			if out.Template != id {
				return ErrStorage
			}
			return s.Auth(ctx, tx, a, out.Target, out.Amount, "draft")
		}
		intent, err := randomID()
		if err != nil {
			return err
		}
		out = Draft{intent, r.ID, r.Target, r.Note, r.Amount, s.Now().UTC()}
		if err = s.audit(ctx, tx, a, id, "draft"); err != nil {
			return err
		}
		return s.opPut(ctx, tx, a, key, fp, out)
	})
	return out, err
}
