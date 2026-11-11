package testutil

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Fixture is the Env network (founder "admin", members "alice" and "bob")
// for callers that hold only a testing.TB, such as the setup of a fuzz target
// (*testing.F), which outlives every fuzz input (ARC-010).
type Fixture struct {
	Ctx                  context.Context
	Store                *store.SQLite
	Svc                  *service.Service
	Clock                *Clock
	Net                  *domain.Network
	Admin, Alice, Bob    domain.Actor
	AdminM, AliceM, BobM *domain.Member
	keys                 atomic.Int64
}

// NewFixture builds a Fixture over a fresh SQLite file, with the same clock,
// ids, Argon2 cost and network policy as NewEnv and Env.Network.
func NewFixture(tb testing.TB) *Fixture {
	tb.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(tb.TempDir(), "circula.db"))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = st.Close() })
	clk := &Clock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	svc, err := service.New(st, service.Config{Clock: clk, IDs: &IDs{},
		Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		tb.Fatal(err)
	}
	fx := &Fixture{Ctx: ctx, Store: st, Svc: svc, Clock: clk}
	must := func(err error) {
		tb.Helper()
		if err != nil {
			tb.Fatal(err)
		}
	}
	person := func(login string) domain.Actor {
		p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: "correct horse battery"})
		must(err)
		return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	}
	fx.Admin = person("admin")
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit = 10000
	pol.ResponsePeriodDays = 0
	n, m, err := svc.CreateNetwork(ctx, fx.Admin, service.CreateNetworkInput{Name: "Net fuzz", Slug: "fuzz",
		Policy: &pol, CreatorDisplayName: "Founder"})
	must(err)
	fx.AdminM = m
	fx.Net, err = svc.ActivateNetwork(ctx, fx.Admin, n.ID)
	must(err)
	member := func(login string) (domain.Actor, *domain.Member) {
		a := person(login)
		tok, _, err := svc.CreateInvitation(ctx, fx.Admin, fx.Net.ID, service.InviteInput{})
		must(err)
		m, err := svc.RedeemInvitation(ctx, a, tok, login)
		must(err)
		return a, m
	}
	fx.Alice, fx.AliceM = member("alice")
	fx.Bob, fx.BobM = member("bob")
	return fx
}

// Key returns a fresh idempotency key.
func (fx *Fixture) Key() string { return fmt.Sprintf("fx-key-%d", fx.keys.Add(1)) }

// volatileTables are stamped by every authenticated request, read or write,
// and so are left out of State. Each entry says why.
var volatileTables = map[string]string{
	"sessions":        "last_seen_at moves on every request a session authenticates",
	"api_tokens":      "last_used_at moves on every request a token authenticates",
	"person_activity": "last_seen_at moves on every authenticated request (inactivity flag, GOV)",
}

// ledgerTables make up State.Ledger.
var ledgerTables = []string{"accounts", "entries", "transactions", "transaction_status_events"}

// State is what a refused command must leave as it was: a checksum of the
// ledger, the number of audit events, and a checksum of every other table.
type State struct {
	Ledger      string            // SHA-256 over the rows of ledgerTables
	AuditEvents int               // rows in audit_events
	Tables      map[string]string // SHA-256 over each table's rows, volatileTables excluded
}

// snapStmts caches the Snapshot statement per database: the schema is fixed
// once the store is open, and preparing the statement costs more than running it.
var snapStmts sync.Map // *sql.DB -> *sql.Stmt

// snapshotStmt builds one statement that returns (table, rows) for every
// table but the volatile ones, each row rendered by SQLite's quote() so that
// NULL, the empty string and blobs stay distinct, plus ("#audit", count).
func snapshotStmt(tb testing.TB, db *sql.DB) *sql.Stmt {
	tb.Helper()
	if st, ok := snapStmts.Load(db); ok {
		return st.(*sql.Stmt)
	}
	rows, err := db.Query(`SELECT m.name, p.name FROM sqlite_master m, pragma_table_info(m.name) p
		WHERE m.type = 'table' AND m.name NOT LIKE 'sqlite_%' ORDER BY m.name, p.cid`)
	if err != nil {
		tb.Fatal(err)
	}
	var tables []string
	cols := map[string][]string{}
	for rows.Next() {
		var t, c string
		if err := rows.Scan(&t, &c); err != nil {
			tb.Fatal(err)
		}
		if _, skip := volatileTables[t]; skip {
			continue
		}
		if cols[t] == nil {
			tables = append(tables, t)
		}
		cols[t] = append(cols[t], `quote("`+c+`")`)
	}
	if err := rows.Close(); err != nil {
		tb.Fatal(err)
	}
	parts := []string{`SELECT '#audit', CAST(COUNT(*) AS TEXT) FROM audit_events`}
	for _, t := range tables { // table and column names come from the schema
		parts = append(parts, `SELECT '`+t+`', (SELECT group_concat(r, char(10)) FROM (SELECT `+
			strings.Join(cols[t], ` || ',' || `)+` AS r FROM "`+t+`" ORDER BY rowid))`)
	}
	st, err := db.Prepare(strings.Join(parts, " UNION ALL "))
	if err != nil {
		tb.Fatal(err)
	}
	if prev, loaded := snapStmts.LoadOrStore(db, st); loaded {
		_ = st.Close()
		return prev.(*sql.Stmt)
	}
	return st
}

// Snapshot reads State with raw SQL, independently of the service, in one
// statement, so every table is read from the same database snapshot.
func Snapshot(tb testing.TB, db *sql.DB) State {
	tb.Helper()
	rows, err := snapshotStmt(tb, db).Query()
	if err != nil {
		tb.Fatal(err)
	}
	s := State{AuditEvents: -1, Tables: map[string]string{}}
	for rows.Next() {
		var t string
		var all sql.NullString
		if err := rows.Scan(&t, &all); err != nil {
			tb.Fatal(err)
		}
		if t == "#audit" {
			if s.AuditEvents, err = strconv.Atoi(all.String); err != nil {
				tb.Fatal(err)
			}
			continue
		}
		sum := sha256.Sum256([]byte(all.String))
		s.Tables[t] = hex.EncodeToString(sum[:])
	}
	if err := rows.Close(); err != nil {
		tb.Fatal(err)
	}
	if s.AuditEvents < 0 {
		tb.Fatal("snapshot: no audit count")
	}
	ledger := sha256.New()
	for _, n := range ledgerTables {
		d, ok := s.Tables[n]
		if !ok {
			tb.Fatalf("ledger table %s missing", n)
		}
		ledger.Write([]byte(n + "=" + d + ";"))
	}
	s.Ledger = hex.EncodeToString(ledger.Sum(nil))
	return s
}

// Changed lists what differs between s and o: "ledger", "audit_events" and
// the names of the tables whose rows changed.
func (s State) Changed(o State) []string {
	var out []string
	if s.Ledger != o.Ledger {
		out = append(out, "ledger")
	}
	if s.AuditEvents != o.AuditEvents {
		out = append(out, fmt.Sprintf("audit_events (%d -> %d)", s.AuditEvents, o.AuditEvents))
	}
	for n, d := range s.Tables {
		if o.Tables[n] != d {
			out = append(out, n)
		}
	}
	for n := range o.Tables {
		if _, ok := s.Tables[n]; !ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
