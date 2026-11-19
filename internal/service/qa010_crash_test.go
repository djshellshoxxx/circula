package service_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// QA-010 / OPS-033 / LED-4: a process that dies in the middle of a posting
// leaves, after restart, either the fully committed state or the original
// one, never a partial posting. Unlike the FaultStore tests, which return an
// error and let the transaction roll back in-process, these tests kill a real
// child process (os.Exit, no deferred cleanup, the SQLite write transaction
// still open) at each write step of the posting unit, then reopen the file as
// a restarted process would.

const (
	qa010ExitCode = 3
	envCrashDB    = "CIRCULA_QA010_DB"
	envCrashOp    = "CIRCULA_QA010_OP"
	envCrashPoint = "CIRCULA_QA010_POINT"
	envCrashArgs  = "CIRCULA_QA010_ARGS"
)

var qa010Argon = service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}

// crashStore kills the process at a named point of a unit of work: "begin"
// (before any write), "<Method>#<n>" (before the nth call of a write step in
// the unit), "pre-commit" (every write done, before COMMIT) or "after-commit" (once the unit committed, before the caller
// sees the result).
type crashStore struct {
	store.Store
	point string
}

func (c *crashStore) Update(ctx context.Context, fn func(store.Tx) error) error {
	err := c.Store.Update(ctx, func(tx store.Tx) error {
		ct := &crashTx{Tx: tx, point: c.point, n: map[string]int{}}
		ct.hit("begin")
		err := fn(ct)
		if err == nil && c.point == "pre-commit" {
			os.Exit(qa010ExitCode) // every write done, COMMIT not yet issued
		}
		return err
	})
	if err == nil && c.point == "after-commit" {
		os.Exit(qa010ExitCode)
	}
	return err
}

type crashTx struct {
	store.Tx
	point string
	n     map[string]int
}

func (t *crashTx) hit(op string) {
	if op == "begin" {
		if t.point == "begin" {
			os.Exit(qa010ExitCode)
		}
		return
	}
	t.n[op]++
	if t.point == op+"#"+strconv.Itoa(t.n[op]) {
		os.Exit(qa010ExitCode)
	}
}

func (t *crashTx) InsertEntry(e *domain.Entry) error {
	t.hit("InsertEntry")
	return t.Tx.InsertEntry(e)
}
func (t *crashTx) AddBalance(netID, id string, d int64) error {
	t.hit("AddBalance")
	return t.Tx.AddBalance(netID, id, d)
}
func (t *crashTx) UpdateTransaction(x *domain.Transaction) error {
	t.hit("UpdateTransaction")
	return t.Tx.UpdateTransaction(x)
}
func (t *crashTx) InsertStatusEvent(e *domain.StatusEvent) error {
	t.hit("InsertStatusEvent")
	return t.Tx.InsertStatusEvent(e)
}
func (t *crashTx) InsertAudit(e *domain.AuditEvent) error {
	t.hit("InsertAudit")
	return t.Tx.InsertAudit(e)
}
func (t *crashTx) PutIdem(r store.IdemRecord) error { t.hit("PutIdem"); return t.Tx.PutIdem(r) }

// TestQA010CrashChild is the child process body; it does nothing unless the
// parent test started it with the crash environment set.
func TestQA010CrashChild(t *testing.T) {
	path := os.Getenv(envCrashDB)
	if path == "" {
		t.Skip("child process of TestQA010CrashDuringPosting")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(&crashStore{Store: st, point: os.Getenv(envCrashPoint)}, service.Config{Argon: qa010Argon})
	if err != nil {
		t.Fatal(err)
	}
	a := strings.Split(os.Getenv(envCrashArgs), ",")
	actor := domain.Actor{PersonID: a[0], Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	switch os.Getenv(envCrashOp) {
	case "accept":
		_, err = svc.AcceptTransaction(ctx, actor, a[1], service.TxRef{TxID: a[2], IdempotencyKey: a[3]})
	case "adjust":
		_, err = svc.CreateAdjustment(ctx, actor, a[1], service.AdjustInput{MemberID: a[2], Amount: 300,
			Direction: service.AdjustCredit, Reason: "qa010", IdempotencyKey: a[3]})
	default:
		t.Fatalf("unknown op %q", os.Getenv(envCrashOp))
	}
	// Reaching here means the crash point was never hit.
	t.Fatalf("crash point %q not reached (err=%v)", os.Getenv(envCrashPoint), err)
}

type qa010World struct {
	path             string
	adminP, bobP     string
	netID            string
	aliceAcc, bobAcc string
	aliceM, bobM     string
	proposal         string
	auditBefore      int
}

// qa010Setup builds a database file with a network, two members and one
// open proposal from Alice to pay Bob 1000, then closes it.
func qa010Setup(t *testing.T) qa010World {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "circula.db")
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, err := service.New(st, service.Config{Argon: qa010Argon})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	person := func(login string) domain.Actor {
		p, err := svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: "correct horse battery"})
		must(err)
		return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	}
	admin := person("admin@example.org")
	pol := domain.DefaultPolicy()
	pol.DefaultCreditLimit = 10000
	n, _, err := svc.CreateNetwork(ctx, admin, service.CreateNetworkInput{Name: "Crash", Slug: "crash", Policy: &pol, CreatorDisplayName: "Founder"})
	must(err)
	_, err = svc.ActivateNetwork(ctx, admin, n.ID)
	must(err)
	member := func(login string) (domain.Actor, *domain.Member) {
		a := person(login)
		tok, _, err := svc.CreateInvitation(ctx, admin, n.ID, service.InviteInput{})
		must(err)
		m, err := svc.RedeemInvitation(ctx, a, tok, login)
		must(err)
		return a, m
	}
	alice, aliceM := member("alice@example.org")
	bob, bobM := member("bob@example.org")
	p, err := svc.ProposeTransaction(ctx, alice, n.ID, service.ProposeInput{CounterpartyID: bobM.ID, Direction: service.ProposerPays,
		Amount: 1000, IdempotencyKey: "qa010-propose"})
	must(err)
	w := qa010World{path: path, adminP: admin.PersonID, bobP: bob.PersonID, netID: n.ID, aliceM: aliceM.ID, bobM: bobM.ID, proposal: p.ID,
		aliceAcc: p.PayerAccountID, bobAcc: p.PayeeAccountID}
	must(st.DB().QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&w.auditBefore))
	return w
}

// qa010State is what the restarted process reads back.
type qa010State struct {
	entries, audit, idem int
	alice, bob           int64
	status               string
}

func readState(t *testing.T, st *store.SQLite, w qa010World, key string) qa010State {
	t.Helper()
	var s qa010State
	db := st.DB()
	for _, q := range []struct {
		dst any
		sql string
		arg []any
	}{
		{&s.entries, `SELECT COUNT(*) FROM entries`, nil},
		{&s.audit, `SELECT COUNT(*) FROM audit_events`, nil},
		{&s.idem, `SELECT COUNT(*) FROM idempotency_keys WHERE key = ?`, []any{key}},
		{&s.alice, `SELECT balance FROM accounts WHERE member_id = ?`, []any{w.aliceM}},
		{&s.bob, `SELECT balance FROM accounts WHERE member_id = ?`, []any{w.bobM}},
		{&s.status, `SELECT status FROM transactions WHERE id = ?`, []any{w.proposal}},
	} {
		if err := db.QueryRow(q.sql, q.arg...).Scan(q.dst); err != nil {
			t.Fatalf("%s: %v", q.sql, err)
		}
	}
	return s
}

func runCrashChild(t *testing.T, w qa010World, op, point, args string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestQA010CrashChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), envCrashDB+"="+w.path, envCrashOp+"="+op, envCrashPoint+"="+point, envCrashArgs+"="+args)
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != qa010ExitCode {
		t.Fatalf("child did not crash at %s (err=%v):\n%s", point, err, out)
	}
}

// restart reopens the file as a new process would and checks the ledger.
func restart(t *testing.T, w qa010World) (*store.SQLite, *service.Service) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, w.path)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if res, err := store.IntegrityCheck(ctx, w.path); err != nil || res != "ok" {
		t.Fatalf("integrity after crash: %q %v", res, err)
	}
	svc, err := service.New(st, service.Config{Argon: qa010Argon})
	if err != nil {
		t.Fatal(err)
	}
	rep, err := svc.VerifyLedger(ctx, domain.SystemActor("qa010"), w.netID)
	if err != nil || !rep.OK {
		t.Fatalf("verify after restart: %v %+v", err, rep)
	}
	return st, svc
}

var qa010Points = []string{"begin", "InsertEntry#1", "InsertEntry#2", "AddBalance#1", "AddBalance#2",
	"UpdateTransaction#1", "InsertStatusEvent#1", "InsertAudit#1", "PutIdem#1", "pre-commit", "after-commit"}

func TestQA010CrashDuringPosting(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns child processes")
	}
	for _, point := range qa010Points {
		t.Run("accept/"+strings.ReplaceAll(point, "#", "-"), func(t *testing.T) {
			w := qa010Setup(t)
			key := "qa010-accept"
			runCrashChild(t, w, "accept", point, strings.Join([]string{w.bobP, w.netID, w.proposal, key}, ","))
			st, svc := restart(t, w)
			got := readState(t, st, w, key)
			original := qa010State{entries: 0, audit: w.auditBefore, idem: 0, alice: 0, bob: 0, status: string(domain.TxProposed)}
			committed := got.entries == 2 && got.alice == -1000 && got.bob == 1000 && got.status == string(domain.TxCompleted) && got.idem == 1
			switch {
			case point == "after-commit":
				if !committed || got.audit <= w.auditBefore {
					t.Fatalf("crash after commit lost the posting: %+v", got)
				}
			case got != original:
				t.Fatalf("crash at %s left a partial state: %+v, want original %+v", point, got, original)
			}
			// The client's retry with the same key now yields exactly one posting.
			bob := domain.Actor{PersonID: w.bobP, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
			tx, err := svc.AcceptTransaction(context.Background(), bob, w.netID, service.TxRef{TxID: w.proposal, IdempotencyKey: key})
			if err != nil || tx.Status != domain.TxCompleted {
				t.Fatalf("retry after restart: %v %+v", err, tx)
			}
			if after := readState(t, st, w, key); after.entries != 2 || after.alice != -1000 || after.bob != 1000 {
				t.Fatalf("after retry: %+v", after)
			}
			if rep, err := svc.VerifyLedger(context.Background(), domain.SystemActor("qa010"), w.netID); err != nil || !rep.OK {
				t.Fatalf("verify after retry: %v %+v", err, rep)
			}
		})
	}
	// Adjustments post to the system account through the same unit.
	for _, point := range qa010Points {
		t.Run("adjust/"+strings.ReplaceAll(point, "#", "-"), func(t *testing.T) {
			w := qa010Setup(t)
			key := "qa010-adjust"
			runCrashChild(t, w, "adjust", point, strings.Join([]string{w.adminP, w.netID, w.aliceM, key}, ","))
			st, _ := restart(t, w)
			got := readState(t, st, w, key)
			switch point {
			case "after-commit":
				if got.entries != 2 || got.alice != 300 || got.idem != 1 {
					t.Fatalf("crash after commit lost the adjustment: %+v", got)
				}
			default:
				if got.entries != 0 || got.alice != 0 || got.idem != 0 || got.audit != w.auditBefore {
					t.Fatalf("crash at %s left a partial adjustment: %+v", point, got)
				}
			}
		})
	}
}
