package governance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrDenied = errors.New("governance operation unavailable")
var ErrConflict = errors.New("governance revision or intent conflict")

type Identity struct{ Network, Member string }
type Principal struct{ Active, Admin, Enabled bool }

// Authority is supplied by the host. All callbacks run inside the very SQLite
// transaction making the decision; identity/roles/electorate never come from JSON.
type Authority interface {
	Principal(context.Context, *sql.Tx, Identity) (Principal, error)
	Electorate(context.Context, *sql.Tx, string) ([]string, error)
	PolicyExists(context.Context, *sql.Tx, Identity, string) (bool, error)
}
type Proposal struct {
	ID, Network, Author, Title, Body, Category, State string
	Opens, Closes                                     time.Time
	Comments                                          bool
	Revision                                          int64
	Eligible                                          int
	Withdrawal                                        string
}
type Comment struct {
	ID, Author, Text string
	At               time.Time
	Hidden           bool
}
type Dispute struct {
	ID, Author, Challenged, Reason, Resolution, Reviewer string
	At                                                   time.Time
}
type Certification struct {
	Member string
	At     time.Time
}
type Result struct {
	Outcome       Outcome
	ClosedAt      time.Time
	Certified     *Certification
	PolicyVersion string
}
type Detail struct {
	Proposal     Proposal
	OwnVote      *Ballot
	Result       *Result
	Comments     []Comment
	Disputes     []Dispute
	OpenDisputes int
	Privacy      string
}

// Command accepts only user-editable fields. IDs, timestamps, current roles and
// frozen eligibility are always generated/read at the service boundary.
type Command struct {
	Action, Proposal, Intent                          string
	Revision                                          int64
	Title, Body, Category                             string
	Opens, Closes                                     time.Time
	Comments                                          bool
	Choice                                            Choice
	Text, Comment, Dispute, Challenged, PolicyVersion string
}
type Reply struct {
	Proposal  string
	Revision  int64
	Reference string
}
type Service struct {
	DB        *sql.DB
	Authority Authority
	Now       func() time.Time
}

func ident(s string) bool {
	if len(s) < 1 || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.In(r, unicode.Cf) {
			return false
		}
	}
	return true
}
func text(s string, max int, line bool) bool {
	if strings.TrimSpace(s) == "" || len(s) > max || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(r == '\n' && !line) || unicode.In(r, unicode.Cf) {
			return false
		}
	}
	return true
}
func randomID() (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(b[:]), nil
}
func (s Service) Migrate(ctx context.Context) error {
	if s.DB == nil {
		return errors.New("missing database")
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var foreignKeys bool
	if e = tx.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); e != nil || !foreignKeys {
		return errors.New("SQLite foreign_keys must be enabled on every connection")
	}
	for _, q := range schemaStatements {
		if _, e = tx.ExecContext(ctx, q); e != nil {
			return e
		}
	}
	if e = validateSchema(ctx, tx); e != nil {
		return e
	}
	var version int
	e = tx.QueryRowContext(ctx, `SELECT version FROM future_gov_schema LIMIT 1`).Scan(&version)
	if errors.Is(e, sql.ErrNoRows) {
		_, e = writeOne(ctx, tx, `INSERT INTO future_gov_schema(version) VALUES(1)`)
	} else if e == nil && version != 1 {
		return errors.New("unsupported governance schema version")
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS future_gov_schema(version INTEGER PRIMARY KEY CHECK(version=1))`,
	`CREATE TABLE IF NOT EXISTS future_gov_proposals(network TEXT NOT NULL,id TEXT NOT NULL,author TEXT NOT NULL,state TEXT NOT NULL,revision INTEGER NOT NULL,body BLOB NOT NULL,PRIMARY KEY(network,id))`,
	`CREATE INDEX IF NOT EXISTS future_gov_owner ON future_gov_proposals(network,author,state,id)`,
	`CREATE TABLE IF NOT EXISTS future_gov_eligible(network TEXT NOT NULL,proposal TEXT NOT NULL,member TEXT NOT NULL,PRIMARY KEY(network,proposal,member),FOREIGN KEY(network,proposal) REFERENCES future_gov_proposals(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_gov_ballots(network TEXT NOT NULL,proposal TEXT NOT NULL,member TEXT NOT NULL,choice TEXT NOT NULL CHECK(choice IN('yes','no','abstain')),cast_at TEXT NOT NULL,PRIMARY KEY(network,proposal,member),FOREIGN KEY(network,proposal,member) REFERENCES future_gov_eligible(network,proposal,member))`,
	`CREATE TABLE IF NOT EXISTS future_gov_results(network TEXT NOT NULL,proposal TEXT NOT NULL,body BLOB NOT NULL,PRIMARY KEY(network,proposal),FOREIGN KEY(network,proposal) REFERENCES future_gov_proposals(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_gov_comments(network TEXT NOT NULL,proposal TEXT NOT NULL,id TEXT NOT NULL,body BLOB NOT NULL,PRIMARY KEY(network,proposal,id),FOREIGN KEY(network,proposal) REFERENCES future_gov_proposals(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_gov_disputes(network TEXT NOT NULL,proposal TEXT NOT NULL,id TEXT NOT NULL,body BLOB NOT NULL,PRIMARY KEY(network,proposal,id),FOREIGN KEY(network,proposal) REFERENCES future_gov_proposals(network,id))`,
	`CREATE TABLE IF NOT EXISTS future_gov_intents(network TEXT NOT NULL,member TEXT NOT NULL,intent TEXT NOT NULL,fingerprint TEXT NOT NULL,reply BLOB NOT NULL,PRIMARY KEY(network,member,intent))`,
	`CREATE TABLE IF NOT EXISTS future_gov_audit(network TEXT NOT NULL,id TEXT NOT NULL,proposal TEXT NOT NULL,member TEXT NOT NULL,action TEXT NOT NULL,at TEXT NOT NULL,reference TEXT NOT NULL,reason TEXT NOT NULL,PRIMARY KEY(network,id))`,
}

func (s Service) begin(ctx context.Context, id Identity) (*sql.Tx, Principal, error) {
	if s.DB == nil || s.Authority == nil || s.Now == nil || !ident(id.Network) || !ident(id.Member) {
		return nil, Principal{}, ErrDenied
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return nil, Principal{}, e
	}
	var foreignKeys bool
	if e = tx.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); e != nil || !foreignKeys {
		tx.Rollback()
		return nil, Principal{}, ErrDenied
	}
	if _, e = writeOne(ctx, tx, `UPDATE future_gov_schema SET version=version WHERE version=1`); e != nil {
		tx.Rollback()
		return nil, Principal{}, e
	}
	p, e := s.Authority.Principal(ctx, tx, id)
	if e != nil || ctx.Err() != nil || !p.Active || !p.Enabled {
		tx.Rollback()
		return nil, Principal{}, ErrDenied
	}
	return tx, p, nil
}
func proposal(ctx context.Context, tx *sql.Tx, network, id string) (Proposal, error) {
	var p Proposal
	var raw []byte
	var author, state string
	var revision int64
	e := tx.QueryRowContext(ctx, `SELECT body,author,state,revision FROM future_gov_proposals WHERE network=? AND id=?`, network, id).Scan(&raw, &author, &state, &revision)
	if e != nil {
		return p, ErrDenied
	}
	e = strictDecode(raw, &p)
	if e != nil || p.Network != network || p.ID != id || p.Author != author || p.State != state || p.Revision != revision || !validProposal(p) {
		return Proposal{}, errors.New("invalid proposal record")
	}
	var electorate int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_gov_eligible WHERE network=? AND proposal=?`, network, id).Scan(&electorate); e != nil {
		return Proposal{}, e
	}
	if electorate != p.Eligible {
		return Proposal{}, errors.New("inconsistent electorate record")
	}
	return p, nil
}
func saveProposal(ctx context.Context, tx *sql.Tx, p Proposal) error {
	raw, e := json.Marshal(p)
	if e != nil {
		return e
	}
	_, e = writeOne(ctx, tx, `INSERT INTO future_gov_proposals(network,id,author,state,revision,body) VALUES(?,?,?,?,?,?) ON CONFLICT(network,id) DO UPDATE SET state=excluded.state,revision=excluded.revision,body=excluded.body`, p.Network, p.ID, p.Author, p.State, p.Revision, raw)
	return e
}
func loadResult(ctx context.Context, tx *sql.Tx, network, pid string) (*Result, error) {
	var raw []byte
	e := tx.QueryRowContext(ctx, `SELECT body FROM future_gov_results WHERE network=? AND proposal=?`, network, pid).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var r Result
	if e = strictDecode(raw, &r); e != nil {
		return nil, e
	}
	if !validResult(r) {
		return nil, errors.New("invalid outcome record")
	}
	return &r, nil
}
func saveResult(ctx context.Context, tx *sql.Tx, network, pid string, r Result, insert bool) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	q := `UPDATE future_gov_results SET body=? WHERE network=? AND proposal=?`
	if insert {
		_, e = writeOne(ctx, tx, `INSERT INTO future_gov_results(network,proposal,body) VALUES(?,?,?)`, network, pid, raw)
	} else {
		_, e = writeOne(ctx, tx, q, raw, network, pid)
	}
	return e
}
func loadComments(ctx context.Context, tx *sql.Tx, network, pid string) ([]Comment, error) {
	rows, e := tx.QueryContext(ctx, `SELECT id,body FROM future_gov_comments WHERE network=? AND proposal=? ORDER BY id LIMIT 1001`, network, pid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		var raw []byte
		var c Comment
		var rowID string
		if e = rows.Scan(&rowID, &raw); e != nil {
			return nil, e
		}
		if e = strictDecode(raw, &c); e != nil || c.ID != rowID || !ident(c.ID) || !ident(c.Author) || c.At.IsZero() || c.Hidden && c.Text != "" || !c.Hidden && !text(c.Text, 2048, false) || len(out) >= 1000 {
			return nil, errors.New("invalid comment record")
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func loadDisputes(ctx context.Context, tx *sql.Tx, network, pid string) ([]Dispute, error) {
	rows, e := tx.QueryContext(ctx, `SELECT id,body FROM future_gov_disputes WHERE network=? AND proposal=? ORDER BY id LIMIT 101`, network, pid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Dispute{}
	for rows.Next() {
		var raw []byte
		var d Dispute
		var rowID string
		if e = rows.Scan(&rowID, &raw); e != nil {
			return nil, e
		}
		if e = strictDecode(raw, &d); e != nil || d.ID != rowID || !ident(d.ID) || !ident(d.Author) || !ident(d.Challenged) || !text(d.Reason, 2000, false) || d.At.IsZero() || len(out) >= 100 || d.Resolution == "" && d.Reviewer != "" || d.Resolution != "" && (!text(d.Resolution, 2000, false) || !ident(d.Reviewer) || d.Reviewer == d.Author || d.Reviewer == d.Challenged) {
			return nil, errors.New("invalid dispute record")
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s Service) Execute(ctx context.Context, id Identity, c Command) (Reply, error) {
	if !ident(c.Intent) || len(c.Action) > 24 || len(c.Proposal) > 128 || len(c.Title) > 200 || len(c.Body) > 12000 || len(c.Category) > 64 || len(c.Text) > 2048 || len(c.Comment) > 128 || len(c.Dispute) > 128 || len(c.Challenged) > 128 || len(c.PolicyVersion) > 128 {
		return Reply{}, ErrDenied
	}
	tx, principal, e := s.begin(ctx, id)
	if e != nil {
		return Reply{}, e
	}
	defer tx.Rollback()
	raw, e := json.Marshal(c)
	if e != nil {
		return Reply{}, e
	}
	sum := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(sum[:])
	var stored string
	var prior []byte
	e = tx.QueryRowContext(ctx, `SELECT fingerprint,reply FROM future_gov_intents WHERE network=? AND member=? AND intent=?`, id.Network, id.Member, c.Intent).Scan(&stored, &prior)
	if e == nil {
		if stored != fingerprint {
			return Reply{}, ErrConflict
		}
		var r Reply
		e = strictDecode(prior, &r)
		if e != nil || !ident(r.Proposal) || r.Revision < 1 {
			return Reply{}, errors.New("invalid replay record")
		}
		previous, err := proposal(ctx, tx, id.Network, r.Proposal)
		if err != nil {
			return Reply{}, err
		}
		if c.Action != "create" && r.Proposal != c.Proposal || r.Revision > previous.Revision {
			return Reply{}, ErrDenied
		}
		if err = authorizeReplay(ctx, tx, id, principal, c, previous); err != nil {
			return Reply{}, err
		}
		if err = ctx.Err(); err != nil {
			return Reply{}, err
		}
		return r, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return Reply{}, e
	}
	if !safetyAction(c.Action) {
		var commands int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_gov_intents WHERE network=? AND member=?`, id.Network, id.Member).Scan(&commands); e != nil {
			return Reply{}, e
		}
		if commands >= 10000 {
			return Reply{}, ErrDenied
		}
	}
	now := s.Now().UTC()
	reply := Reply{}
	p := Proposal{}
	if c.Action == "create" {
		if c.Proposal != "" || !text(c.Title, 200, true) || !text(c.Body, 12000, false) || !text(c.Category, 64, true) {
			return reply, ErrDenied
		}
		var count int
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_gov_proposals WHERE network=? AND author=?`, id.Network, id.Member).Scan(&count); e != nil {
			return reply, e
		}
		if count >= 100 {
			return reply, ErrDenied
		}
		pid, err := randomID()
		if err != nil {
			return reply, err
		}
		p = Proposal{ID: pid, Network: id.Network, Author: id.Member, Title: c.Title, Body: c.Body, Category: c.Category, State: "draft", Comments: c.Comments, Revision: 1}
		if e = saveProposal(ctx, tx, p); e != nil {
			return reply, e
		}
	} else {
		if !ident(c.Proposal) {
			return reply, ErrDenied
		}
		p, e = proposal(ctx, tx, id.Network, c.Proposal)
		if e != nil {
			return reply, e
		}
		if c.Revision != p.Revision {
			return reply, ErrConflict
		}
		switch c.Action {
		case "edit":
			if p.State != "draft" || p.Author != id.Member || !text(c.Title, 200, true) || !text(c.Body, 12000, false) || !text(c.Category, 64, true) {
				return reply, ErrDenied
			}
			p.Title, p.Body, p.Category, p.Comments = c.Title, c.Body, c.Category, c.Comments
			p.Revision++
			e = saveProposal(ctx, tx, p)
		case "open":
			if p.State != "draft" || p.Author != id.Member || c.Opens.Before(now) || !c.Closes.After(c.Opens) || c.Closes.Sub(c.Opens) > 90*24*time.Hour || c.Opens.Sub(now) > 365*24*time.Hour {
				return reply, ErrDenied
			}
			members, err := s.Authority.Electorate(ctx, tx, id.Network)
			if err != nil {
				return reply, err
			}
			if len(members) < 1 || len(members) > 10000 {
				return reply, ErrDenied
			}
			members = append([]string(nil), members...)
			sort.Strings(members)
			for i, m := range members {
				if !ident(m) || i > 0 && members[i-1] == m {
					return reply, ErrDenied
				}
				if _, e = writeOne(ctx, tx, `INSERT INTO future_gov_eligible(network,proposal,member) VALUES(?,?,?)`, id.Network, p.ID, m); e != nil {
					return reply, e
				}
			}
			p.State = "open"
			p.Opens, p.Closes = c.Opens.UTC(), c.Closes.UTC()
			p.Eligible = len(members)
			p.Revision++
			e = saveProposal(ctx, tx, p)
		case "vote":
			if p.State != "open" || now.Before(p.Opens) || !now.Before(p.Closes) || !valid(c.Choice) {
				return reply, ErrDenied
			}
			var eligible bool
			if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM future_gov_eligible WHERE network=? AND proposal=? AND member=?)`, id.Network, p.ID, id.Member).Scan(&eligible); e != nil {
				return reply, e
			}
			if !eligible {
				return reply, ErrDenied
			}
			_, e = writeOne(ctx, tx, `INSERT INTO future_gov_ballots(network,proposal,member,choice,cast_at) VALUES(?,?,?,?,?)`, id.Network, p.ID, id.Member, c.Choice, now.Format(time.RFC3339Nano))
		case "withdraw":
			if (p.Author != id.Member && !principal.Admin) || (p.State != "open" && p.State != "draft") || p.State == "open" && !now.Before(p.Closes) || !text(c.Text, 1000, false) {
				return reply, ErrDenied
			}
			p.State = "withdrawn"
			p.Withdrawal = c.Text
			p.Revision++
			e = saveProposal(ctx, tx, p)
		case "close":
			if p.State != "open" || now.Before(p.Closes) || (p.Author != id.Member && !principal.Admin) {
				return reply, ErrDenied
			}
			outcome := Outcome{Eligible: p.Eligible, Advisory: true}
			rows, err := tx.QueryContext(ctx, `SELECT b.member,b.choice,b.cast_at,e.member FROM future_gov_ballots b LEFT JOIN future_gov_eligible e ON e.network=b.network AND e.proposal=b.proposal AND e.member=b.member WHERE b.network=? AND b.proposal=?`, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			for rows.Next() {
				var choice Choice
				var member, cast string
				var eligible sql.NullString
				if err = rows.Scan(&member, &choice, &cast, &eligible); err != nil {
					rows.Close()
					return reply, err
				}
				castAt, parseErr := time.Parse(time.RFC3339Nano, cast)
				if parseErr != nil || !eligible.Valid || eligible.String != member || !ident(member) || castAt.Before(p.Opens) || !castAt.Before(p.Closes) {
					rows.Close()
					return reply, ErrDenied
				}
				switch choice {
				case Yes:
					outcome.Yes++
				case No:
					outcome.No++
				case Abstain:
					outcome.Abstain++
				default:
					rows.Close()
					return reply, ErrDenied
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return reply, err
			}
			if outcome.Yes+outcome.No+outcome.Abstain > outcome.Eligible {
				return reply, ErrDenied
			}
			if e = saveResult(ctx, tx, id.Network, p.ID, Result{Outcome: outcome, ClosedAt: now}, true); e != nil {
				return reply, e
			}
			p.State = "closed"
			p.Revision++
			e = saveProposal(ctx, tx, p)
		case "comment":
			if !p.Comments || p.State == "draft" || p.State == "withdrawn" || !text(c.Text, 2048, false) {
				return reply, ErrDenied
			}
			var count int
			if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_gov_comments WHERE network=? AND proposal=?`, id.Network, p.ID).Scan(&count); e != nil {
				return reply, e
			}
			if count >= 1000 {
				return reply, ErrDenied
			}
			comments, err := loadComments(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			own := 0
			for _, comment := range comments {
				if comment.Author == id.Member {
					own++
				}
			}
			if own >= 100 {
				return reply, ErrDenied
			}
			cid, err := randomID()
			if err != nil {
				return reply, err
			}
			body, _ := json.Marshal(Comment{cid, id.Member, c.Text, now, false})
			_, e = writeOne(ctx, tx, `INSERT INTO future_gov_comments(network,proposal,id,body) VALUES(?,?,?,?)`, id.Network, p.ID, cid, body)
			reply.Reference = cid
		case "hide":
			if !ident(c.Comment) || !text(c.Text, 1000, false) {
				return reply, ErrDenied
			}
			comments, err := loadComments(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			found := false
			for _, comment := range comments {
				if comment.ID != c.Comment {
					continue
				}
				if comment.Hidden || (comment.Author != id.Member && !principal.Admin) {
					return reply, ErrDenied
				}
				comment.Hidden = true
				comment.Text = ""
				body, _ := json.Marshal(comment)
				_, e = writeOne(ctx, tx, `UPDATE future_gov_comments SET body=? WHERE network=? AND proposal=? AND id=?`, body, id.Network, p.ID, comment.ID)
				found = true
				break
			}
			if !found {
				return reply, ErrDenied
			}
			reply.Reference = c.Comment
		case "dispute":
			if p.State == "draft" || !ident(c.Challenged) || !text(c.Text, 2000, false) {
				return reply, ErrDenied
			}
			disputes, err := loadDisputes(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			if len(disputes) >= 100 {
				return reply, ErrDenied
			}
			own := 0
			for _, dispute := range disputes {
				if dispute.Author == id.Member {
					own++
					if dispute.Challenged == c.Challenged && dispute.Resolution == "" {
						return reply, ErrDenied
					}
				}
			}
			if own >= 10 {
				return reply, ErrDenied
			}
			did, err := randomID()
			if err != nil {
				return reply, err
			}
			body, _ := json.Marshal(Dispute{ID: did, Author: id.Member, Challenged: c.Challenged, Reason: c.Text, At: now})
			_, e = writeOne(ctx, tx, `INSERT INTO future_gov_disputes(network,proposal,id,body) VALUES(?,?,?,?)`, id.Network, p.ID, did, body)
			reply.Reference = did
		case "resolve":
			if !principal.Admin || !ident(c.Dispute) || !text(c.Text, 2000, false) {
				return reply, ErrDenied
			}
			disputes, err := loadDisputes(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			found := false
			for _, d := range disputes {
				if d.ID != c.Dispute {
					continue
				}
				if d.Challenged == id.Member || d.Author == id.Member || d.Resolution != "" {
					return reply, ErrDenied
				}
				d.Resolution, d.Reviewer = c.Text, id.Member
				body, _ := json.Marshal(d)
				_, e = writeOne(ctx, tx, `UPDATE future_gov_disputes SET body=? WHERE network=? AND proposal=? AND id=?`, body, id.Network, p.ID, d.ID)
				found = true
				break
			}
			if !found {
				return reply, ErrDenied
			}
			reply.Reference = c.Dispute
		case "certify", "link":
			if !principal.Admin || p.State != "closed" || p.Author == id.Member {
				return reply, ErrDenied
			}
			r, err := loadResult(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			if r == nil {
				return reply, ErrDenied
			}
			disputes, err := loadDisputes(ctx, tx, id.Network, p.ID)
			if err != nil {
				return reply, err
			}
			for _, d := range disputes {
				if d.Resolution == "" || d.Challenged == id.Member {
					return reply, ErrDenied
				}
			}
			if c.Action == "certify" {
				if r.Certified != nil {
					return reply, ErrConflict
				}
				r.Certified = &Certification{id.Member, now}
			} else {
				if r.Certified == nil || r.PolicyVersion != "" || !ident(c.PolicyVersion) {
					return reply, ErrDenied
				}
				exists, err := s.Authority.PolicyExists(ctx, tx, id, c.PolicyVersion)
				if err != nil {
					return reply, err
				}
				if !exists {
					return reply, ErrDenied
				}
				r.PolicyVersion = c.PolicyVersion
			}
			e = saveResult(ctx, tx, id.Network, p.ID, *r, false)
		default:
			return reply, ErrDenied
		}
	}
	if e != nil {
		return Reply{}, e
	}
	reply.Proposal, reply.Revision = p.ID, p.Revision
	encoded, e := json.Marshal(reply)
	if e != nil {
		return Reply{}, e
	}
	if _, e = writeOne(ctx, tx, `INSERT INTO future_gov_intents(network,member,intent,fingerprint,reply) VALUES(?,?,?,?,?)`, id.Network, id.Member, c.Intent, fingerprint, encoded); e != nil {
		return Reply{}, e
	}
	auditID, e := randomID()
	if e != nil {
		return Reply{}, e
	}
	reason := ""
	if c.Action == "hide" || c.Action == "withdraw" || c.Action == "resolve" {
		reason = c.Text
	}
	if _, e = writeOne(ctx, tx, `INSERT INTO future_gov_audit(network,id,proposal,member,action,at,reference,reason) VALUES(?,?,?,?,?,?,?,?)`, id.Network, auditID, p.ID, id.Member, c.Action, now.Format(time.RFC3339Nano), reply.Reference, reason); e != nil {
		return Reply{}, e
	}
	if e = tx.Commit(); e != nil {
		return Reply{}, e
	}
	return reply, nil
}
func (s Service) Read(ctx context.Context, id Identity, pid string) (Detail, error) {
	tx, principal, e := s.begin(ctx, id)
	if e != nil {
		return Detail{}, e
	}
	defer tx.Rollback()
	p, e := proposal(ctx, tx, id.Network, pid)
	if e != nil {
		return Detail{}, e
	}
	if p.State == "draft" && p.Author != id.Member && !principal.Admin {
		return Detail{}, ErrDenied
	}
	d := Detail{Proposal: p, Privacy: "Ballots are server-readable and accessible to database operators. Members see only their own choice. Results are advisory; no anonymous or coercion-resistant voting claim."}
	var b Ballot
	var cast string
	b.Network, b.Poll, b.Member = id.Network, pid, id.Member
	e = tx.QueryRowContext(ctx, `SELECT choice,cast_at FROM future_gov_ballots WHERE network=? AND proposal=? AND member=?`, id.Network, pid, id.Member).Scan(&b.Choice, &cast)
	if e == nil {
		b.CastAt, e = time.Parse(time.RFC3339Nano, cast)
		if e != nil || !valid(b.Choice) || p.Eligible < 1 || b.CastAt.Before(p.Opens) || !b.CastAt.Before(p.Closes) {
			return Detail{}, errors.New("invalid stored ballot")
		}
		d.OwnVote = &b
	} else if !errors.Is(e, sql.ErrNoRows) {
		return Detail{}, e
	}
	d.Result, e = loadResult(ctx, tx, id.Network, pid)
	if e != nil {
		return Detail{}, e
	}
	if d.Result != nil && (p.State != "closed" || d.Result.Outcome.Eligible != p.Eligible || d.Result.ClosedAt.Before(p.Closes)) {
		return Detail{}, errors.New("inconsistent result record")
	}
	comments, e := loadComments(ctx, tx, id.Network, pid)
	if e != nil {
		return Detail{}, e
	}
	for _, c := range comments {
		if !c.Hidden {
			d.Comments = append(d.Comments, c)
		}
	}
	disputes, e := loadDisputes(ctx, tx, id.Network, pid)
	if e != nil {
		return Detail{}, e
	}
	for _, dispute := range disputes {
		if dispute.Resolution == "" {
			d.OpenDisputes++
		}
		if principal.Admin || dispute.Author == id.Member {
			d.Disputes = append(d.Disputes, dispute)
		}
	}
	if e = tx.Commit(); e != nil {
		return Detail{}, e
	}
	return d, nil
}

// List uses an opaque stable ID keyset. No author/email/eligibility filters exist.
func (s Service) List(ctx context.Context, id Identity, after string, limit int) ([]Proposal, error) {
	if limit < 1 || limit > 100 || after != "" && !ident(after) {
		return nil, ErrDenied
	}
	tx, principal, e := s.begin(ctx, id)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, `SELECT body,author,state,revision,id,(SELECT COUNT(*) FROM future_gov_eligible e WHERE e.network=future_gov_proposals.network AND e.proposal=future_gov_proposals.id) FROM future_gov_proposals WHERE network=? AND id>? AND (state!='draft' OR author=? OR ?=1) ORDER BY id LIMIT ?`, id.Network, after, id.Member, principal.Admin, limit)
	if e != nil {
		return nil, e
	}
	out := []Proposal{}
	for rows.Next() {
		var raw []byte
		var p Proposal
		var author, state, pid string
		var electorate int
		var revision int64
		if e = rows.Scan(&raw, &author, &state, &revision, &pid, &electorate); e != nil {
			rows.Close()
			return nil, e
		}
		if e = strictDecode(raw, &p); e != nil || !validProposal(p) || p.Network != id.Network || p.ID != pid || p.Author != author || p.State != state || p.Revision != revision || p.Eligible != electorate || p.State == "draft" && p.Author != id.Member && !principal.Admin {
			rows.Close()
			if e == nil {
				e = errors.New("inconsistent proposal record")
			}
			return nil, e
		}
		out = append(out, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return out, nil
}

// Maintenance intentionally exposes no ballot deletion or policy write. Retain
// frozen ballots/outcomes/audit for their published governance retention period.
func (s Service) String() string {
	return fmt.Sprintf("isolated advisory governance service (configured=%t)", s.DB != nil && s.Authority != nil)
}
