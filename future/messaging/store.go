package messaging

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

// Actor comes from authenticated host context, never request JSON. Authorizer
// must read current network/membership/role/reference state using the supplied
// transaction. It must not retain tx or open another database connection.
type Actor struct{ Network, Member string }
type Authorizer func(context.Context, *sql.Tx, Actor, string, string) error

var (
	ErrInvalid  = errors.New("invalid messaging input")
	ErrConflict = errors.New("idempotency key conflicts with prior request")
	ErrQuota    = errors.New("messaging quota exceeded")
)

//go:embed migrations/001_messaging.sql
var schema string

type Store struct {
	db        *sql.DB
	authorize Authorizer
	now       func() time.Time
}

// Open explicitly opens a private SQLite file, using one connection per store.
// Host integrations should use NewStore with their already serialized DB pool.
// Neither constructor installs schema. Call Migrate explicitly before use.
func Open(path string, auth Authorizer) (*Store, error) {
	if path == "" || path == ":memory:" || strings.HasPrefix(path, "file:") || strings.ContainsAny(path, "?\x00") {
		return nil, ErrInvalid
	}
	// Do not let SQLite's default umask create a world-readable private inbox.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("messaging database must be a private regular file")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"PRAGMA busy_timeout=10000", "PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA secure_delete=ON"} {
		if _, err = db.Exec(q); err != nil {
			db.Close()
			return nil, err
		}
	}
	s, err := NewStore(db, auth)
	if err != nil {
		db.Close()
	}
	return s, err
}

// NewStore borrows db; the host must configure foreign_keys, serialized writes,
// busy_timeout and secure_delete on every connection. Close belongs to the host.
func NewStore(db *sql.DB, auth Authorizer) (*Store, error) {
	if db == nil || auth == nil {
		return nil, ErrInvalid
	}
	return &Store{db: db, authorize: auth, now: time.Now}, nil
}

// Close closes the underlying pool. Do not call when a host owns the pool.
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Migrate(ctx context.Context) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, schema); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) unit(ctx context.Context, a Actor, action, target string, fn func(*sql.Tx) error) error {
	if !validID(a.Network) || !validID(a.Member) {
		return ErrDenied
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Acquire SQLite's writer reservation before the authorization snapshot. This
	// avoids deferred-transaction read/write upgrades under competing processes.
	if _, e = tx.ExecContext(ctx, "UPDATE message_threads SET sequence=sequence WHERE 0"); e != nil {
		return e
	}
	if e = s.authorize(ctx, tx, a, action, target); e != nil {
		return ErrDenied
	}
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit()
}
func validID(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_.:-", c)) {
			return false
		}
	}
	return true
}
func validKey(s string) bool { return len(s) >= 8 && validID(s) }
func newID() (string, error) {
	b := make([]byte, 16)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func requestHash(v ...string) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type Conversation struct {
	ID        string `json:"id"`
	Network   string `json:"network"`
	A         string `json:"a"`
	B         string `json:"b"`
	Reference string `json:"reference,omitempty"`
	Created   int64  `json:"created"`
	Activity  int64  `json:"activity"`
	Retention int    `json:"retention_days"`
	Sequence  int64  `json:"sequence"`
	Closed    bool   `json:"closed"`
	Held      bool   `json:"-"`
	Purged    bool   `json:"-"`
	Unread    int64  `json:"unread"`
}
type Message struct {
	ID       string `json:"id"`
	Sequence int64  `json:"sequence"`
	Sender   string `json:"sender"`
	Body     string `json:"body"`
	Created  int64  `json:"created"`
}

func (s *Store) thread(ctx context.Context, tx *sql.Tx, a Actor, id string, moderator bool) (Conversation, error) {
	var t Conversation
	if !validID(id) {
		return t, ErrDenied
	}
	e := tx.QueryRowContext(ctx, `SELECT id,network,a,b,reference,created,activity,retention,sequence,closed,held,purged FROM message_threads WHERE network=? AND id=?`, a.Network, id).Scan(&t.ID, &t.Network, &t.A, &t.B, &t.Reference, &t.Created, &t.Activity, &t.Retention, &t.Sequence, &t.Closed, &t.Held, &t.Purged)
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return t, ErrDenied
		}
		return t, e
	}
	if !moderator && (a.Member != t.A && a.Member != t.B || t.Purged || s.expired(t)) {
		return t, ErrDenied
	}
	return t, nil
}
func (s *Store) expired(t Conversation) bool {
	return !s.now().Before(time.Unix(0, t.Activity).Add(time.Duration(t.Retention) * 24 * time.Hour))
}
func blocked(ctx context.Context, tx *sql.Tx, n, a, b string) error {
	var count int
	e := tx.QueryRowContext(ctx, `SELECT count(*) FROM message_blocks WHERE network=? AND ((actor=? AND target=?) OR (actor=? AND target=?))`, n, a, b, b, a).Scan(&count)
	if e != nil {
		return e
	}
	if count > 0 {
		return ErrDenied
	}
	return nil
}
func idem(ctx context.Context, tx *sql.Tx, a Actor, op, key, hash string) (string, error) {
	var h, result string
	e := tx.QueryRowContext(ctx, `SELECT hash,result FROM message_idempotency WHERE network=? AND actor=? AND operation=? AND key=?`, a.Network, a.Member, op, key).Scan(&h, &result)
	if errors.Is(e, sql.ErrNoRows) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	if h != hash {
		return "", ErrConflict
	}
	return result, nil
}
func saveIdem(ctx context.Context, tx *sql.Tx, a Actor, op, key, hash, result, thread string) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO message_idempotency(network,actor,operation,key,hash,result,thread) VALUES(?,?,?,?,?,?,?)`, a.Network, a.Member, op, key, hash, result, thread)
	return e
}

func (s *Store) Create(ctx context.Context, a Actor, target, reference, key string, days int) (Conversation, error) {
	var out Conversation
	if !validID(target) || target == a.Member || !validKey(key) || days < 1 || days > 30 || len(reference) > 128 || reference != "" && !validID(reference) {
		return out, ErrInvalid
	}
	e := s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if e := s.authorize(ctx, tx, a, "conversation", target); e != nil {
			return ErrDenied
		}
		if e := s.authorize(ctx, tx, Actor{a.Network, target}, "member", target); e != nil {
			return ErrDenied
		}
		if e := s.authorize(ctx, tx, a, "retention-days", fmt.Sprint(days)); e != nil {
			return ErrDenied
		}
		if reference != "" {
			if e := s.authorize(ctx, tx, a, "reference", reference); e != nil {
				return ErrDenied
			}
		}
		if e := blocked(ctx, tx, a.Network, a.Member, target); e != nil {
			return e
		}
		hash := requestHash(target, reference, fmt.Sprint(days))
		id, e := idem(ctx, tx, a, "create", key, hash)
		if e != nil {
			return e
		}
		if id != "" {
			out, e = s.thread(ctx, tx, a, id, false)
			return e
		}
		var count int
		now := s.now().UnixNano()
		// Closed/purged tombstones still count toward initiation budgets, so
		// closing a conversation cannot be used to bypass admission quotas.
		for _, budget := range []struct {
			window time.Duration
			max    int
		}{{time.Minute, 60}, {24 * time.Hour, 1000}} {
			if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_threads WHERE network=? AND a=? AND created>?`, a.Network, a.Member, now-int64(budget.window)).Scan(&count); e != nil {
				return e
			}
			if count >= budget.max {
				return ErrQuota
			}
		}
		cutoff := now
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_threads WHERE network=? AND (a=? OR b=?) AND closed=0 AND purged=0 AND activity + retention*86400000000000 > ?`, a.Network, a.Member, a.Member, cutoff).Scan(&count); e != nil {
			return e
		}
		if count >= 100 {
			return ErrQuota
		}
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_threads WHERE network=? AND (a=? OR b=?) AND closed=0 AND purged=0 AND activity + retention*86400000000000 > ?`, a.Network, target, target, cutoff).Scan(&count); e != nil {
			return e
		}
		if count >= 100 {
			return ErrQuota
		}
		id, e = newID()
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO message_threads(network,id,a,b,reference,created,activity,retention) VALUES(?,?,?,?,?,?,?,?)`, a.Network, id, a.Member, target, reference, now, now, days)
		if e != nil {
			return e
		}
		for _, m := range []string{a.Member, target} {
			if _, e = tx.ExecContext(ctx, `INSERT INTO message_positions(network,thread,member) VALUES(?,?,?)`, a.Network, id, m); e != nil {
				return e
			}
		}
		if e = saveIdem(ctx, tx, a, "create", key, hash, id, id); e != nil {
			return e
		}
		out, e = s.thread(ctx, tx, a, id, false)
		return e
	})
	return out, e
}

func (s *Store) Send(ctx context.Context, a Actor, id, body, key string) (Message, error) {
	var out Message
	if !validKey(key) {
		return out, ErrInvalid
	}
	e := s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, false)
		if e != nil {
			return e
		}
		other := t.A
		if other == a.Member {
			other = t.B
		}
		if e = s.authorize(ctx, tx, a, "conversation", other); e != nil {
			return ErrDenied
		}
		if e = s.authorize(ctx, tx, Actor{a.Network, other}, "member", other); e != nil {
			return ErrDenied
		}
		if e = blocked(ctx, tx, a.Network, t.A, t.B); e != nil {
			return e
		}
		text, e := Prepare(Thread{Network: t.Network, A: t.A, B: t.B}, Access{a.Network, a.Member, true}, body)
		if e != nil {
			return ErrInvalid
		}
		hash := requestHash(id, text)
		result, e := idem(ctx, tx, a, "send", key, hash)
		if e != nil {
			return e
		}
		if result != "" {
			if e = tx.QueryRowContext(ctx, `SELECT id,sequence,sender,body,created FROM message_entries WHERE network=? AND id=?`, a.Network, result).Scan(&out.ID, &out.Sequence, &out.Sender, &out.Body, &out.Created); e != nil {
				return e
			}
			var hidden int64
			if e = tx.QueryRowContext(ctx, `SELECT hidden FROM message_positions WHERE network=? AND thread=? AND member=?`, a.Network, id, a.Member).Scan(&hidden); e != nil {
				return e
			}
			if out.Sequence <= hidden {
				out.Body = ""
			}
			return nil
		}
		if t.Closed {
			return ErrDenied
		}
		var count int
		now := s.now().UnixNano()
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_entries WHERE network=? AND sender=? AND created>?`, a.Network, a.Member, now-int64(time.Minute)).Scan(&count); e != nil {
			return e
		}
		if count >= 60 {
			return ErrQuota
		}
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_entries WHERE network=? AND sender=? AND created>?`, a.Network, a.Member, now-int64(24*time.Hour)).Scan(&count); e != nil {
			return e
		}
		if count >= 1000 {
			return ErrQuota
		}
		mid, e := newID()
		if e != nil {
			return e
		}
		out = Message{mid, t.Sequence + 1, a.Member, text, now}
		_, e = tx.ExecContext(ctx, `INSERT INTO message_entries(network,thread,sequence,id,sender,body,created) VALUES(?,?,?,?,?,?,?)`, a.Network, id, out.Sequence, mid, a.Member, text, now)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE message_threads SET sequence=?,activity=? WHERE network=? AND id=?`, out.Sequence, now, a.Network, id); e != nil {
			return e
		}
		return saveIdem(ctx, tx, a, "send", key, hash, mid, id)
	})
	return out, e
}

// Threads uses a stable ID cursor. Expired/hidden conversations are omitted.
func (s *Store) Threads(ctx context.Context, a Actor, after string, limit int) ([]Conversation, error) {
	out := []Conversation{}
	if limit < 1 || limit > 100 || after != "" && !validID(after) {
		return nil, ErrInvalid
	}
	e := s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		rows, e := tx.QueryContext(ctx, `SELECT t.id FROM message_threads t JOIN message_positions p ON p.network=t.network AND p.thread=t.id AND p.member=? WHERE t.network=? AND (t.a=? OR t.b=?) AND t.id>? AND t.purged=0 AND t.activity+t.retention*86400000000000>? AND (p.hidden_view=0 OR t.sequence>p.hidden) ORDER BY t.id LIMIT ?`, a.Member, a.Network, a.Member, a.Member, after, s.now().UnixNano(), limit)
		if e != nil {
			return e
		}
		var ids []string
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			t, e := s.thread(ctx, tx, a, id, false)
			if e != nil {
				return e
			}
			if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM message_entries e JOIN message_positions p ON p.network=e.network AND p.thread=e.thread AND p.member=? WHERE e.network=? AND e.thread=? AND e.sequence>max(p.seen,p.hidden) AND e.sender<>?`, a.Member, a.Network, id, a.Member).Scan(&t.Unread); e != nil {
				return e
			}
			out = append(out, t)
		}
		return nil
	})
	return out, e
}
func messages(ctx context.Context, tx *sql.Tx, a Actor, id string, after int64, limit int, moderator bool) ([]Message, error) {
	hidden := int64(0)
	if !moderator {
		if e := tx.QueryRowContext(ctx, `SELECT hidden FROM message_positions WHERE network=? AND thread=? AND member=?`, a.Network, id, a.Member).Scan(&hidden); e != nil {
			return nil, e
		}
	}
	if after < hidden {
		after = hidden
	}
	rows, e := tx.QueryContext(ctx, `SELECT id,sequence,sender,body,created FROM message_entries WHERE network=? AND thread=? AND sequence>? ORDER BY sequence LIMIT ?`, a.Network, id, after, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if e = rows.Scan(&m.ID, &m.Sequence, &m.Sender, &m.Body, &m.Created); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) Messages(ctx context.Context, a Actor, id string, after int64, limit int) ([]Message, error) {
	if after < 0 || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	var out []Message
	e := s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if _, e := s.thread(ctx, tx, a, id, false); e != nil {
			return e
		}
		var e error
		out, e = messages(ctx, tx, a, id, after, limit, false)
		return e
	})
	return out, e
}
func (s *Store) MarkRead(ctx context.Context, a Actor, id string, sequence int64) error {
	return s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, false)
		if e != nil {
			return e
		}
		if sequence < 0 || sequence > t.Sequence {
			return ErrInvalid
		}
		_, e = tx.ExecContext(ctx, `UPDATE message_positions SET seen=max(seen,?) WHERE network=? AND thread=? AND member=?`, sequence, a.Network, id, a.Member)
		return e
	})
}
func (s *Store) Hide(ctx context.Context, a Actor, id string) error {
	return s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, false)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `UPDATE message_positions SET hidden_view=1,hidden=?,seen=max(seen,?) WHERE network=? AND thread=? AND member=?`, t.Sequence, t.Sequence, a.Network, id, a.Member)
		return e
	})
}
func (s *Store) CloseThread(ctx context.Context, a Actor, id string) error {
	return s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if _, e := s.thread(ctx, tx, a, id, false); e != nil {
			return e
		}
		_, e := tx.ExecContext(ctx, `UPDATE message_threads SET closed=1 WHERE network=? AND id=?`, a.Network, id)
		return e
	})
}
func (s *Store) Block(ctx context.Context, a Actor, target string, enabled bool) error {
	if !validID(target) || target == a.Member {
		return ErrInvalid
	}
	return s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if enabled {
			if e := s.authorize(ctx, tx, Actor{a.Network, target}, "member", target); e != nil {
				return ErrDenied
			}
			var count, exists int
			if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM message_blocks WHERE network=? AND actor=?`, a.Network, a.Member).Scan(&count); e != nil {
				return e
			}
			if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM message_blocks WHERE network=? AND actor=? AND target=?`, a.Network, a.Member, target).Scan(&exists); e != nil {
				return e
			}
			if count >= 1000 && exists == 0 {
				return ErrQuota
			}
		}
		q := `DELETE FROM message_blocks WHERE network=? AND actor=? AND target=?`
		if enabled {
			q = `INSERT OR IGNORE INTO message_blocks(network,actor,target) VALUES(?,?,?)`
		}
		_, e := tx.ExecContext(ctx, q, a.Network, a.Member, target)
		return e
	})
}
func (s *Store) Report(ctx context.Context, a Actor, id string, sequence int64, reason string) error {
	if sequence < 1 || len(reason) < 1 || len(reason) > 1024 || !utf8.ValidString(reason) || strings.TrimSpace(reason) == "" || strings.ContainsRune(reason, 0) {
		return ErrInvalid
	}
	return s.unit(ctx, a, "member", a.Member, func(tx *sql.Tx) error {
		if _, e := s.thread(ctx, tx, a, id, false); e != nil {
			return e
		}
		var hidden int64
		if e := tx.QueryRowContext(ctx, `SELECT hidden FROM message_positions WHERE network=? AND thread=? AND member=?`, a.Network, id, a.Member).Scan(&hidden); e != nil {
			return e
		}
		if sequence <= hidden {
			return ErrDenied
		}
		var sender string
		if e := tx.QueryRowContext(ctx, `SELECT sender FROM message_entries WHERE network=? AND thread=? AND sequence=?`, a.Network, id, sequence).Scan(&sender); e != nil {
			if errors.Is(e, sql.ErrNoRows) {
				return ErrDenied
			}
			return e
		}
		if sender == a.Member {
			return ErrInvalid
		}
		var count int
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM message_reports WHERE network=? AND reporter=? AND created>?`, a.Network, a.Member, s.now().Add(-24*time.Hour).UnixNano()).Scan(&count); e != nil {
			return e
		}
		var existing int
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM message_reports WHERE network=? AND thread=? AND sequence=? AND reporter=?`, a.Network, id, sequence, a.Member).Scan(&existing); e != nil {
			return e
		}
		if existing > 0 {
			return nil
		}
		if count >= 20 {
			return ErrQuota
		}
		_, e := tx.ExecContext(ctx, `INSERT INTO message_reports(network,thread,sequence,reporter,reason,created) VALUES(?,?,?,?,?,?)`, a.Network, id, sequence, a.Member, reason, s.now().UnixNano())
		return e
	})
}

type Report struct {
	Thread   string `json:"thread"`
	Sequence int64  `json:"sequence"`
	Reporter string `json:"reporter"`
	Reason   string `json:"reason"`
	State    string `json:"state"`
}

func (s *Store) Reports(ctx context.Context, a Actor, id string) ([]Report, error) {
	return s.ReportsPage(ctx, a, id, 0, "", 100)
}
func (s *Store) ReportsPage(ctx context.Context, a Actor, id string, after int64, reporter string, limit int) ([]Report, error) {
	if after < 0 || limit < 1 || limit > 100 || reporter != "" && !validID(reporter) {
		return nil, ErrInvalid
	}
	out := []Report{}
	e := s.unit(ctx, a, "moderate", id, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, true)
		if e != nil {
			return e
		}
		if t.Purged || s.expired(t) && !t.Held {
			return ErrDenied
		}
		rows, e := tx.QueryContext(ctx, `SELECT thread,sequence,reporter,reason,state FROM message_reports WHERE network=? AND thread=? AND (sequence>? OR (sequence=? AND reporter>?)) ORDER BY sequence,reporter LIMIT ?`, a.Network, id, after, after, reporter, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var r Report
			if e = rows.Scan(&r.Thread, &r.Sequence, &r.Reporter, &r.Reason, &r.State); e != nil {
				return e
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, e
}

// Moderate is the only route to held material. action hold/release changes a
// separate legal-hold flag; resolve/dismiss updates a particular report.
func (s *Store) Moderate(ctx context.Context, a Actor, id, action string, sequence int64, reporter string) error {
	return s.unit(ctx, a, "moderate", id, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, true)
		if e != nil {
			return e
		}
		if t.Purged {
			return ErrDenied
		}
		switch action {
		case "hold", "release":
			if action == "hold" && s.expired(t) && !t.Held {
				return ErrDenied
			}
			_, e = tx.ExecContext(ctx, `UPDATE message_threads SET held=? WHERE network=? AND id=?`, action == "hold", a.Network, id)
		case "resolve", "dismiss":
			if s.expired(t) && !t.Held {
				return ErrDenied
			}
			var result sql.Result
			result, e = tx.ExecContext(ctx, `UPDATE message_reports SET state=? WHERE network=? AND thread=? AND sequence=? AND reporter=?`, action, a.Network, id, sequence, reporter)
			if e == nil {
				n, _ := result.RowsAffected()
				if n != 1 {
					return ErrDenied
				}
			}
		default:
			return ErrInvalid
		}
		return e
	})
}
func (s *Store) HeldMessages(ctx context.Context, a Actor, id string, after int64, limit int) ([]Message, error) {
	if after < 0 || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	var out []Message
	e := s.unit(ctx, a, "moderate", id, func(tx *sql.Tx) error {
		t, e := s.thread(ctx, tx, a, id, true)
		if e != nil {
			return e
		}
		if t.Purged || s.expired(t) && !t.Held {
			return ErrDenied
		}
		out, e = messages(ctx, tx, a, id, after, limit, true)
		return e
	})
	return out, e
}

// Purge removes content, report reasons and keyed content hashes atomically.
// Thread tombstones remain to avoid resurrecting an expired conversation ID.
func (s *Store) Purge(ctx context.Context, a Actor) (int64, error) {
	var count int64
	e := s.unit(ctx, a, "retention", "", func(tx *sql.Tx) error {
		filter := `SELECT id FROM message_threads WHERE network=? AND purged=0 AND held=0 AND activity+retention*86400000000000<=?`
		now := s.now().UnixNano()
		for _, table := range []string{"message_reports", "message_entries", "message_idempotency"} {
			if _, e := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE network=? AND thread IN (`+filter+`)`, a.Network, a.Network, now); e != nil {
				return e
			}
		}
		result, e := tx.ExecContext(ctx, `UPDATE message_threads SET purged=1,closed=1,reference='' WHERE network=? AND purged=0 AND held=0 AND activity+retention*86400000000000<=?`, a.Network, now)
		if e != nil {
			return e
		}
		count, e = result.RowsAffected()
		return e
	})
	return count, e
}

// Export writes the first bounded page. ExportPage supports sequence cursors.
// Output remains server-readable plaintext and must be handled as private data.
func (s *Store) Export(ctx context.Context, a Actor, id string, w io.Writer) error {
	return s.ExportPage(ctx, a, id, 0, w)
}
func (s *Store) ExportPage(ctx context.Context, a Actor, id string, after int64, w io.Writer) error {
	out, e := s.Messages(ctx, a, id, after, 100)
	if e != nil {
		return e
	}
	return json.NewEncoder(w).Encode(out)
}

// ShortenRetention applies a newly approved network maximum to existing threads.
// It never increases retention or changes activity and requires both maintenance
// permission and fresh validation of the network's requested retention policy.
func (s *Store) ShortenRetention(ctx context.Context, a Actor, days int) error {
	if days < 1 || days > 30 {
		return ErrInvalid
	}
	return s.unit(ctx, a, "retention", "", func(tx *sql.Tx) error {
		if e := s.authorize(ctx, tx, a, "retention-days", fmt.Sprint(days)); e != nil {
			return ErrDenied
		}
		_, e := tx.ExecContext(ctx, `UPDATE message_threads SET retention=min(retention,?) WHERE network=? AND purged=0`, days, a.Network)
		return e
	})
}
