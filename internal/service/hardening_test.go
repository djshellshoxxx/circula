package service_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// newWorldCfg is newWorld with service Config tweaks.
func newWorldCfg(t *testing.T, mod func(*domain.Policy), cfg func(*service.Config)) *world {
	t.Helper()
	e := tu.NewEnvConfig(t, nil, cfg)
	w := &world{Env: e}
	w.admin = e.Person("admin@example.org")
	w.net, w.adminM = e.Network(w.admin, "demo", mod)
	w.a, w.aM = e.Member(w.admin, w.net, "ana@example.org")
	w.b, w.bM = e.Member(w.admin, w.net, "ben@example.org")
	return w
}

// dbChecksum hashes the content of every table, in rowid order.
func dbChecksum(t *testing.T, e *tu.Env) string {
	t.Helper()
	db := e.Store.DB()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	_ = rows.Close()
	h := sha256.New()
	for _, tb := range tables {
		r, err := db.Query("SELECT * FROM " + tb + " ORDER BY rowid")
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := r.Columns()
		fmt.Fprintf(h, "[%s]", tb)
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(h, "%v\n", vals)
		}
		_ = r.Close()
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// SVC-01 / OPS-016 / FR-052: verification modifies nothing.
func TestVerifyLedgerIsReadOnly(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	before := dbChecksum(t, w.Env)
	auditBefore := countRows(t, w.Env, "audit_events")
	for i := 0; i < 3; i++ {
		rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
		if !rep.OK || rep.RunID != "" {
			t.Fatalf("report %+v", rep)
		}
		tu.Must(w.Svc.VerifyLedger(w.Ctx, domain.SystemActor("cli"), w.net.ID))
	}
	if after := dbChecksum(t, w.Env); after != before {
		t.Error("VerifyLedger changed the database")
	}
	if n := countRows(t, w.Env, "audit_events"); n != auditBefore {
		t.Errorf("audit events %d -> %d", auditBefore, n)
	}
	if n := countRows(t, w.Env, "consistency_runs"); n != 0 {
		t.Errorf("consistency runs = %d", n)
	}
	// Recording is a separate explicit act: exactly one run and one audit event.
	rep := tu.Must(w.Svc.VerifyAndRecordLedger(w.Ctx, w.admin, w.net.ID))
	if rep.RunID == "" || countRows(t, w.Env, "consistency_runs") != 1 || countRows(t, w.Env, "audit_events") != auditBefore+1 {
		t.Errorf("recorded run: %+v", rep)
	}
	// Recording needs an administrator.
	wantCode(t, w.Svc.RecordConsistencyRun(w.Ctx, w.a, rep), domain.ErrForbidden)
}

// A read-only verify does not wait for or block the write gate.
func TestVerifyLedgerDoesNotTakeWriteGate(t *testing.T) {
	w := newWorld(t, nil)
	hold := make(chan struct{})
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- w.Store.Update(w.Ctx, func(store.Tx) error { close(started); <-hold; return nil })
	}()
	<-started
	_, err := w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID) // must complete while the writer is parked
	close(hold)
	if err != nil {
		t.Fatalf("verify blocked or failed behind an open write unit: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// AUTH-01 / SA-04: parallel guesses are each counted; at most throttleAfter
// are ever evaluated, however many run at once.
func TestLoginThrottleIsAtomicUnderConcurrency(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("ana@example.org")
	const n = 40
	var unauth, limited, other atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", "wrong password!!")
			switch domain.CodeOf(err) {
			case domain.CodeUnauthenticated:
				unauth.Add(1)
			case domain.CodeRateLimited:
				limited.Add(1)
			default:
				other.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if unauth.Load() != 5 || limited.Load() != n-5 || other.Load() != 0 {
		t.Fatalf("evaluated=%d throttled=%d other=%d, want 5/%d/0", unauth.Load(), limited.Load(), other.Load(), n-5)
	}
	var failures int
	if err := e.Store.DB().QueryRow(`SELECT failures FROM login_throttle WHERE login = 'ana@example.org'`).Scan(&failures); err != nil || failures != 5 {
		t.Errorf("failures = %d err=%v, want 5", failures, err)
	}
	// even the right password is refused during the lock
	_, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	wantCode(t, err, domain.ErrRateLimited)
}

// AUTH-01: the throttle key is bounded by validLogin; garbage is refused
// uniformly and stores nothing.
func TestLoginThrottleKeyIsBounded(t *testing.T) {
	e := tu.NewEnv(t)
	for _, bad := range []string{strings.Repeat("x", 1<<20), "has space@example.org", "a\x00b@example.org", "ab", ""} {
		_, _, err := e.Svc.Login(e.Ctx, bad, "whatever password")
		wantCode(t, err, domain.ErrUnauthenticated)
	}
	if n := countRows(t, e, "login_throttle"); n != 0 {
		t.Errorf("throttle rows for invalid logins: %d", n)
	}
	_, _, err := e.Svc.Login(e.Ctx, "nobody@example.org", "whatever password")
	wantCode(t, err, domain.ErrUnauthenticated)
	if n := countRows(t, e, "login_throttle"); n != 1 {
		t.Errorf("throttle rows = %d, want 1", n)
	}
}

// failingStore makes every View fail with err.
type failingStore struct {
	store.Store
	err error
}

func (f *failingStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if f.err != nil {
		return f.err
	}
	return f.Store.View(ctx, fn)
}

// AUTH-02: store errors are not "unauthenticated".
func TestAuthenticateStoreErrorIsNotUnauthenticated(t *testing.T) {
	boom := errors.New("disk I/O error")
	var fs *failingStore
	e := tu.NewEnvWith(t, func(s store.Store) store.Store { fs = &failingStore{Store: s}; return fs })
	e.Person("ana@example.org")
	tok, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if err != nil {
		t.Fatal(err)
	}
	apiTok, _ := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, tu.Must(e.Svc.Authenticate(e.Ctx, tok)),
		service.TokenInput{Name: "t", Scopes: []domain.Scope{domain.ScopeRead}}))

	for name, cause := range map[string]error{"internal": boom, "busy": &domain.Error{Code: domain.CodeUnavailable}} {
		fs.err = cause
		_, err1 := e.Svc.Authenticate(e.Ctx, tok)
		_, err2 := e.Svc.AuthenticateToken(e.Ctx, apiTok)
		for _, err := range []error{err1, err2} {
			if err == nil || errors.Is(err, domain.ErrUnauthenticated) {
				t.Errorf("%s: store error became %v", name, err)
			}
		}
		if name == "busy" && (!domain.IsRetryable(err1) || !domain.IsRetryable(err2)) {
			t.Error("busy store error must stay retryable")
		}
	}
	fs.err = nil
	if _, err := e.Svc.Authenticate(e.Ctx, tok); err != nil {
		t.Errorf("recovered store: %v", err)
	}
	// unknown credentials are still unauthenticated
	_, err = e.Svc.Authenticate(e.Ctx, "nope")
	wantCode(t, err, domain.ErrUnauthenticated)
	_, err = e.Svc.AuthenticateToken(e.Ctx, "cir_nope")
	wantCode(t, err, domain.ErrUnauthenticated)
}

// SA-09: deleting the newest audit events (triggers dropped, as a DB admin
// could) is detected through the recorded chain head.
func TestAuditTailTruncationDetected(t *testing.T) {
	w := newWorld(t, nil)
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 1000)
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Fatalf("clean ledger: %v", rep.Findings)
	}
	db := w.Store.DB()
	for _, q := range []string{`DROP TRIGGER audit_no_delete`, `DROP TRIGGER audit_subjects_no_delete`,
		`DELETE FROM audit_subjects WHERE audit_id IN (SELECT id FROM audit_events WHERE network_id = '` + w.net.ID + `' AND seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = '` + w.net.ID + `'))`,
		`DELETE FROM audit_events WHERE network_id = '` + w.net.ID + `' AND seq = (SELECT MAX(seq) FROM audit_events WHERE network_id = '` + w.net.ID + `')`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID))
	if rep.OK || !strings.Contains(strings.Join(rep.Findings, " "), "recorded head") {
		t.Errorf("tail truncation not detected: %+v", rep)
	}
}

// SA-09: the head advances in the same unit as every append, and cannot be
// rolled back or deleted without dropping its triggers.
func TestAuditHeadTracksAppendsAndIsProtected(t *testing.T) {
	w := newWorld(t, nil)
	db := w.Store.DB()
	head := func() (seq int64, hash string) {
		if err := db.QueryRow(`SELECT seq, hash FROM audit_heads WHERE network_id = ?`, w.net.ID).Scan(&seq, &hash); err != nil {
			t.Fatal(err)
		}
		return
	}
	s0, _ := head()
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	s1, h1 := head()
	var lastSeq int64
	var lastHash string
	if err := db.QueryRow(`SELECT seq, hash FROM audit_events WHERE network_id = ? ORDER BY seq DESC LIMIT 1`, w.net.ID).Scan(&lastSeq, &lastHash); err != nil {
		t.Fatal(err)
	}
	if s1 <= s0 || s1 != lastSeq || h1 != lastHash {
		t.Errorf("head %d/%s, chain end %d/%s (was %d)", s1, h1, lastSeq, lastHash, s0)
	}
	for _, q := range []string{`DELETE FROM audit_heads`, `UPDATE audit_heads SET seq = 1`} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("%q succeeded", q)
		}
	}
}

// SA-10: sign-up is uniform for new and taken logins and never alters the
// existing account.
func TestRegisterIsUniform(t *testing.T) {
	e := tu.NewEnv(t)
	first, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: "ana@example.org", Password: pw})
	if err != nil {
		t.Fatal(err)
	}
	dup, err := e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: " ANA@example.org", Password: "a different passphrase"})
	if err != nil || dup == nil {
		t.Fatalf("duplicate sign-up must look like success: %v", err)
	}
	if dup.ID == first.ID {
		t.Error("decoy person must not carry the real id")
	}
	if n := countRows(t, e, "persons"); n != 1 {
		t.Errorf("persons = %d", n)
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", "a different passphrase"); err == nil {
		t.Error("the duplicate sign-up's password works")
	}
	if _, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil {
		t.Errorf("original password broken: %v", err)
	}
	var dups int
	if err := e.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActRegisterDuplicate).Scan(&dups); err != nil || dups != 1 {
		t.Errorf("duplicate attempts audited = %d err=%v", dups, err)
	}
	// field validation still applies equally
	_, err = e.Svc.RegisterPerson(e.Ctx, service.RegisterInput{Login: "ana@example.org", Password: "short"})
	wantCode(t, err, domain.ErrValidation)
}

// SA-12 / DATA-01: Purge removes dead rows only.
func TestPurge(t *testing.T) {
	w := newWorld(t, nil)
	tokKeep, _, _ := w.Svc.Login(w.Ctx, "ana@example.org", pw)
	tokOut, _, _ := w.Svc.Login(w.Ctx, "ben@example.org", pw)
	tu.Must(0, w.Svc.Logout(w.Ctx, tokOut))
	_, _, err := w.Svc.Login(w.Ctx, "ghost@example.org", "whatever password")
	wantCode(t, err, domain.ErrUnauthenticated)
	tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100)) // leaves an idempotency record
	ledger := []string{"entries", "audit_events", "transactions", "members", "persons", "accounts"}
	counts := func() (out []int) {
		for _, tb := range ledger {
			out = append(out, countRows(t, w.Env, tb))
		}
		return
	}
	base := counts()

	// nothing is dead yet
	rep := tu.Must(w.Svc.Purge(w.Ctx))
	if rep != (service.PurgeReport{}) {
		t.Fatalf("premature purge: %+v", rep)
	}
	// 25 h: the revoked session and the stale throttle row go; idempotency lives 7 days
	w.Clock.Advance(25 * time.Hour)
	rep = tu.Must(w.Svc.Purge(w.Ctx))
	if rep.Sessions != 1 || rep.Throttles != 1 || rep.IdempotencyKeys != 0 {
		t.Fatalf("after 25h: %+v", rep)
	}
	if _, err := w.Svc.Authenticate(w.Ctx, tokKeep); err != nil {
		t.Errorf("live session purged: %v", err)
	}
	// 8 days: idempotency records are expired
	w.Clock.Advance(7 * 24 * time.Hour)
	rep = tu.Must(w.Svc.Purge(w.Ctx))
	if rep.IdempotencyKeys == 0 || rep.Sessions != 0 {
		t.Fatalf("after 8d: %+v", rep)
	}
	if n := countRows(t, w.Env, "idempotency_keys"); n != 0 {
		t.Errorf("idempotency rows left: %d", n)
	}
	// 100 days: the idle session is gone, and purge is idempotent
	w.Clock.Advance(100 * 24 * time.Hour)
	rep = tu.Must(w.Svc.Purge(w.Ctx))
	if rep.Sessions != 1 {
		t.Fatalf("after 100d: %+v", rep)
	}
	if _, err := w.Svc.Authenticate(w.Ctx, tokKeep); err == nil {
		t.Error("expired session still authenticates")
	}
	if again := tu.Must(w.Svc.Purge(w.Ctx)); again != (service.PurgeReport{}) {
		t.Errorf("second purge: %+v", again)
	}
	for i, c := range counts() {
		if c != base[i] {
			t.Errorf("purge touched %s: %d -> %d", ledger[i], base[i], c)
		}
	}
	if rep := tu.Must(w.Svc.VerifyLedger(w.Ctx, w.admin, w.net.ID)); !rep.OK {
		t.Errorf("ledger: %v", rep.Findings)
	}
}

// SA-08: a request reserves nothing against the payer (DOM-011/012) and is capped.
func TestUnsolicitedRequestsDoNotFreezeCredit(t *testing.T) {
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 10000 }, nil)
	// B asks A for 90.00 of A's 100.00
	req := tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 9000))
	if got := w.balance(w.a); got.ReservedOutgoing != 0 || got.Available != 10000 {
		t.Errorf("payer view after unsolicited request: %+v", got)
	}
	// A is free to spend all of it elsewhere
	pay := tu.Must(w.propose(w.a, w.adminM, service.ProposerPays, 10000))
	if got := w.balance(w.a); got.ReservedOutgoing != 10000 || got.Available != 0 {
		t.Errorf("own proposal must reserve: %+v", got)
	}
	// ...and the requester's own view: nothing reserved on the payee
	if got := w.balance(w.b); got.ReservedOutgoing != 0 || got.AwaitingFromOthers != 9000 {
		t.Errorf("requester view: %+v", got)
	}
	// consent posts, with the hard limit re-checked (A's 100.00 is already committed
	// to a proposal, but reservations are soft; only balance+limit decides)
	tu.Must(w.accept(w.a, req.ID))
	if got := w.balance(w.a).Balance; got != -9000 {
		t.Errorf("A balance = %d", got)
	}
	// the other proposal now fails its hard check: only 10.00 of limit is left
	_, err := w.accept(w.admin, pay.ID)
	wantCode(t, err, domain.ErrInsufficientCredit)
}

func TestRequestExposureAndCounts(t *testing.T) {
	w := newWorldCfg(t, func(p *domain.Policy) { p.DefaultCreditLimit = 10000 }, nil)
	tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 6000))
	// together the open requests would exceed what A could ever pay
	_, err := w.propose(w.b, w.aM, service.ProposerIsPaid, 6000)
	wantCode(t, err, domain.ErrInsufficientCredit)
	// requests to other payers are independent
	tu.Must(w.propose(w.b, w.adminM, service.ProposerIsPaid, 6000))
	// per-pair count cap (default 5)
	for i := 0; i < 4; i++ {
		tu.Must(w.propose(w.b, w.aM, service.ProposerIsPaid, 1))
	}
	_, err = w.propose(w.b, w.aM, service.ProposerIsPaid, 1)
	wantCode(t, err, domain.ErrRateLimited)
	// the cap applies to requests only: normal payments to the same member are unaffected
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 1))
	tu.Must(w.propose(w.a, w.bM, service.ProposerIsPaid, 1)) // opposite direction pair
}

func TestOpenProposalsPerMemberCap(t *testing.T) {
	w := newWorldCfg(t, nil, func(c *service.Config) { c.MaxOpenProposalsPerMember = 3 })
	for i := 0; i < 3; i++ {
		tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1))
	}
	_, err := w.propose(w.a, w.adminM, service.ProposerPays, 1)
	wantCode(t, err, domain.ErrRateLimited)
	// answering one frees a slot; other members' caps are independent
	list := tu.Must(w.Svc.ListPendingForActor(w.Ctx, w.b, w.net.ID, service.PendingQuery{AwaitingMe: true}))
	tu.Must(w.Svc.DeclineTransaction(w.Ctx, w.b, w.net.ID, service.TxRef{TxID: list[0].ID, IdempotencyKey: w.Key()}))
	tu.Must(w.propose(w.a, w.adminM, service.ProposerPays, 1))
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 1))
}

// SVC-04: joining again after denial, departure or suspension is refused.
func TestRejoinSemantics(t *testing.T) {
	t.Run("denied applicant", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinApproval })
		p, pM := w.joinPending("p@example.org")
		tu.Must(w.Svc.DecideJoinRequest(w.Ctx, w.admin, w.net.ID, pM.ID, false, "no"))
		_, err := w.Svc.JoinNetwork(w.Ctx, p, w.net.ID, "P")
		wantCode(t, err, domain.ErrForbidden)
		got := tu.Must(w.Svc.AdminListMembers(w.Ctx, w.admin, w.net.ID, []domain.MemberStatus{domain.MemberDeparted}, service.Page{}))
		if len(got) != 1 || got[0].ID != pM.ID {
			t.Errorf("denied membership changed: %+v", got)
		}
	})
	t.Run("departed member with a fresh invitation", func(t *testing.T) {
		w := newWorld(t, nil)
		tu.Must(w.Svc.DepartMember(w.Ctx, w.a, w.net.ID, service.DepartInput{}))
		tok, inv := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
		_, err := w.Svc.RedeemInvitation(w.Ctx, w.a, tok, "Ana")
		wantCode(t, err, domain.ErrForbidden)
		invs := tu.Must(w.Svc.ListInvitations(w.Ctx, w.admin, w.net.ID))
		for _, i := range invs {
			if i.ID == inv.ID && i.UsedAt != nil {
				t.Error("token consumed by a refused redemption")
			}
		}
		// the same token still works for someone else
		c := w.Person("c@example.org")
		tu.Must(w.Svc.RedeemInvitation(w.Ctx, c, tok, "C"))
	})
	t.Run("suspended member", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinOpen })
		tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.aM.ID, "pause"))
		_, err := w.Svc.JoinNetwork(w.Ctx, w.a, w.net.ID, "Ana")
		wantCode(t, err, domain.ErrForbidden)
	})
	t.Run("active and pending stay idempotent", func(t *testing.T) {
		w := newWorld(t, func(p *domain.Policy) { p.JoinMode = domain.JoinOpen })
		if again := tu.Must(w.Svc.JoinNetwork(w.Ctx, w.a, w.net.ID, "x")); again.ID != w.aM.ID {
			t.Error("active member join not idempotent")
		}
		tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
		if again := tu.Must(w.Svc.RedeemInvitation(w.Ctx, w.a, tok, "x")); again.ID != w.aM.ID {
			t.Error("active member redeem not idempotent")
		}
	})
}

// SVC-03: the default flag also blocks posting a payment by a flagged payer.
func TestAcceptRechecksDefaultFlag(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultFlagBlocksPayer = true })
	prop := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100))
	tu.Must(0, w.Svc.SetDefaultFlag(w.Ctx, w.admin, w.net.ID, w.aM.ID, true, "missed commitments"))
	_, err := w.accept(w.b, prop.ID)
	wantCode(t, err, domain.ErrForbidden)
	if w.balance(w.a).Balance != 0 {
		t.Error("flagged payer was debited")
	}
	tu.Must(0, w.Svc.SetDefaultFlag(w.Ctx, w.admin, w.net.ID, w.aM.ID, false, "resolved"))
	tu.Must(w.accept(w.b, prop.ID))
	// a flagged PAYEE may still receive
	tu.Must(0, w.Svc.SetDefaultFlag(w.Ctx, w.admin, w.net.ID, w.bM.ID, true, "x"))
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 50)
	// with the policy off the flag is only a marker
	w2 := newWorld(t, func(p *domain.Policy) { p.DefaultFlagBlocksPayer = false })
	p2 := tu.Must(w2.propose(w2.a, w2.bM, service.ProposerPays, 100))
	tu.Must(0, w2.Svc.SetDefaultFlag(w2.Ctx, w2.admin, w2.net.ID, w2.aM.ID, true, "x"))
	tu.Must(w2.accept(w2.b, p2.ID))
}

// SVC-05: the limit PUT path writes one limit.changed and no account.viewed.
func TestSetCreditLimitViewAuditsOnce(t *testing.T) {
	w := newWorld(t, nil)
	count := func(action string) (n int) {
		if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, action).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return
	}
	before := tu.Must(w.Svc.PeekMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	lim := int64(2500)
	after := tu.Must(w.Svc.SetCreditLimitView(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lim, Reason: "review"}))
	if before.CreditLimit == after.CreditLimit || after.CreditLimit != 2500 || after.Available != 2500 {
		t.Errorf("before %+v after %+v", before, after)
	}
	if count(domain.ActAccountViewed) != 0 || count(domain.ActLimitChanged) != 1 {
		t.Errorf("viewed=%d changed=%d", count(domain.ActAccountViewed), count(domain.ActLimitChanged))
	}
	// Peek needs an admin and does not leak across networks
	_, err := w.Svc.PeekMemberBalance(w.Ctx, w.a, w.net.ID, w.bM.ID)
	wantCode(t, err, domain.ErrForbidden)
	// the audited display path still audits
	tu.Must(w.Svc.AdminGetMemberBalance(w.Ctx, w.admin, w.net.ID, w.aM.ID))
	if count(domain.ActAccountViewed) != 1 {
		t.Error("AdminGetMemberBalance must stay audited")
	}
	// failed validation changes nothing
	neg := int64(-1)
	_, err = w.Svc.SetCreditLimitView(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &neg, Reason: "x"})
	wantCode(t, err, domain.ErrValidation)
	if count(domain.ActLimitChanged) != 1 {
		t.Error("failed limit change was audited")
	}
}

// API-04: single-object reads with the right visibility.
func TestSingleObjectReads(t *testing.T) {
	w := newWorld(t, nil)
	oAdmin := w.Person("o@example.org") // admin of a second network in the same database
	oNet, _ := w.Network(oAdmin, "other-net", nil)

	// GetMember
	if m := tu.Must(w.Svc.GetMember(w.Ctx, w.admin, w.net.ID, w.aM.ID)); m.PersonID == "" || m.JoinMethod == "" {
		t.Errorf("admin must see the full record: %+v", m)
	}
	if m := tu.Must(w.Svc.GetMember(w.Ctx, w.a, w.net.ID, w.aM.ID)); m.PersonID == "" {
		t.Error("own record must be full")
	}
	m := tu.Must(w.Svc.GetMember(w.Ctx, w.a, w.net.ID, w.bM.ID))
	if m.DisplayName == "" || m.PersonID != "" || m.JoinMethod != "" || len(m.Roles) != 0 || m.AccountID != "" {
		t.Errorf("peer view leaks: %+v", m)
	}
	tu.Must(0, w.Svc.SuspendMember(w.Ctx, w.admin, w.net.ID, w.bM.ID, "x"))
	_, err := w.Svc.GetMember(w.Ctx, w.a, w.net.ID, w.bM.ID) // not active any more
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.GetMember(w.Ctx, w.a, w.net.ID, "missing")
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.GetMember(w.Ctx, oAdmin, oNet.ID, w.aM.ID) // id from another network
	wantCode(t, err, domain.ErrNotFound)
	_, err = w.Svc.GetMember(w.Ctx, oAdmin, w.net.ID, w.aM.ID) // outsider
	wantCode(t, err, domain.ErrNotFound)

	// GetListing
	l := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{Type: domain.ListingOffer, Title: "Bread", IdempotencyKey: w.Key()}))
	if got := tu.Must(w.Svc.GetListing(w.Ctx, w.admin, w.net.ID, l.ID)); got.ID != l.ID {
		t.Error("listing not returned")
	}
	paused := domain.ListingPaused
	tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: l.ID, Status: &paused}))
	_, err = w.Svc.GetListing(w.Ctx, w.admin, w.net.ID, l.ID)
	wantCode(t, err, domain.ErrNotFound)
	if got := tu.Must(w.Svc.GetListing(w.Ctx, w.a, w.net.ID, l.ID)); got.Status != domain.ListingPaused {
		t.Error("owner must see own paused listing")
	}
	_, err = w.Svc.GetListing(w.Ctx, oAdmin, oNet.ID, l.ID)
	wantCode(t, err, domain.ErrNotFound)
}

func TestGetDisputeForTx(t *testing.T) {
	w := newWorld(t, nil)
	c, _ := w.Member(w.admin, w.net, "cy@example.org")
	tx := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 500)
	_, err := w.Svc.GetDisputeForTx(w.Ctx, w.a, w.net.ID, tx.ID)
	wantCode(t, err, domain.ErrNotFound) // none yet
	d := tu.Must(w.Svc.OpenDispute(w.Ctx, w.b, w.net.ID, service.OpenDisputeInput{TxID: tx.ID, Category: "quality", IdempotencyKey: w.Key()}))
	for name, who := range map[string]domain.Actor{"party": w.a, "opener": w.b, "admin": w.admin} {
		if v := tu.Must(w.Svc.GetDisputeForTx(w.Ctx, who, w.net.ID, tx.ID)); v.Dispute.ID != d.ID {
			t.Errorf("%s: wrong dispute", name)
		}
	}
	_, err = w.Svc.GetDisputeForTx(w.Ctx, c, w.net.ID, tx.ID)
	wantCode(t, err, domain.ErrNotFound)
	// GetDispute (existing) is tenant scoped too
	oa := w.Person("o@example.org")
	on, _ := w.Network(oa, "other-net", nil)
	_, err = w.Svc.GetDispute(w.Ctx, oa, on.ID, d.ID)
	wantCode(t, err, domain.ErrNotFound)
}

// API-03: entries by ledger_seq descending, keyset paged. Proposals made first
// complete last, so id order and posting order differ.
func TestListEntriesOrderedByLedgerSeq(t *testing.T) {
	w := newWorld(t, nil)
	t1 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 100)) // proposed first...
	t2 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 200))
	t3 := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 300))
	tu.Must(w.accept(w.b, t3.ID)) // ...posted in the order 3, 1, 2
	tu.Must(w.accept(w.b, t1.ID))
	tu.Must(w.accept(w.b, t2.ID))

	rows := tu.Must(w.Svc.ListEntries(w.Ctx, w.a, w.net.ID, service.EntryQuery{}))
	var gotSeq []int64
	var gotTx []string
	for _, r := range rows {
		gotSeq = append(gotSeq, r.Entry.LedgerSeq)
		gotTx = append(gotTx, r.Tx.ID)
		if r.Entry.TxID != r.Tx.ID || r.Entry.Amount >= 0 {
			t.Errorf("row %+v", r)
		}
	}
	if fmt.Sprint(gotSeq) != "[3 2 1]" || fmt.Sprint(gotTx) != fmt.Sprint([]string{t2.ID, t1.ID, t3.ID}) {
		t.Errorf("seq %v tx %v", gotSeq, gotTx)
	}
	// keyset paging
	p1 := tu.Must(w.Svc.ListEntries(w.Ctx, w.a, w.net.ID, service.EntryQuery{Limit: 2}))
	p2 := tu.Must(w.Svc.ListEntries(w.Ctx, w.a, w.net.ID, service.EntryQuery{Limit: 2, BeforeSeq: p1[len(p1)-1].Entry.LedgerSeq}))
	if len(p1) != 2 || len(p2) != 1 || p2[0].Entry.LedgerSeq != 1 {
		t.Errorf("pages %d/%d", len(p1), len(p2))
	}
	// only the caller's own entries; payee sees the credits
	if rb := tu.Must(w.Svc.ListEntries(w.Ctx, w.b, w.net.ID, service.EntryQuery{})); len(rb) != 3 || rb[0].Entry.Amount != 200 {
		t.Errorf("payee rows: %+v", rb)
	}
	if rc := tu.Must(w.Svc.ListEntries(w.Ctx, w.admin, w.net.ID, service.EntryQuery{})); len(rc) != 0 {
		t.Errorf("admin sees entries that are not his: %d", len(rc))
	}
	_, err := w.Svc.ListEntries(w.Ctx, domain.SystemActor("x"), w.net.ID, service.EntryQuery{})
	wantCode(t, err, domain.ErrForbidden)
}

// WEB-02: pending proposals are paged without a hidden window.
func TestListPendingForActorPagesBeyondOneHundred(t *testing.T) {
	w := newWorldCfg(t, nil, nil)
	const n = 130
	want := map[string]bool{}
	for i := 0; i < n; i++ {
		want[tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1)).ID] = true
	}
	done := tu.Must(w.propose(w.a, w.bM, service.ProposerPays, 1))
	tu.Must(w.accept(w.b, done.ID)) // completed ones are excluded
	mine := tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 1))

	got := map[string]bool{}
	after := ""
	pages := 0
	for {
		page := tu.Must(w.Svc.ListPendingForActor(w.Ctx, w.b, w.net.ID, service.PendingQuery{AwaitingMe: true, Page: service.Page{AfterID: after, Limit: 100}}))
		if len(page) == 0 {
			break
		}
		pages++
		for _, tx := range page {
			if got[tx.ID] || tx.Status != domain.TxProposed || tx.CreatedBy == w.bM.ID {
				t.Fatalf("bad row %+v", tx)
			}
			got[tx.ID] = true
		}
		after = page[len(page)-1].ID
	}
	if len(got) != n || pages != 2 {
		t.Fatalf("got %d rows in %d pages, want %d in 2", len(got), pages, n)
	}
	for id := range want {
		if !got[id] {
			t.Fatalf("missing %s", id)
		}
	}
	// without AwaitingMe the caller's own open proposal is included
	all := tu.Must(w.Svc.ListPendingForActor(w.Ctx, w.b, w.net.ID, service.PendingQuery{Page: service.Page{Limit: 500}}))
	if len(all) != n+1 || all[0].ID != mine.ID {
		t.Errorf("all=%d first=%v", len(all), all[0].ID)
	}
	// expired proposals are not pending
	w.Clock.Advance(30 * 24 * time.Hour)
	if left := tu.Must(w.Svc.ListPendingForActor(w.Ctx, w.b, w.net.ID, service.PendingQuery{})); len(left) != 0 {
		t.Errorf("expired proposals listed: %d", len(left))
	}
	_, err := w.Svc.ListPendingForActor(w.Ctx, w.Person("zz@example.org"), w.net.ID, service.PendingQuery{})
	wantCode(t, err, domain.ErrNotFound)
}

// SA-11: per-actor export quota, checked before anything is audited.
func TestExportQuotaPerActor(t *testing.T) {
	w := newWorldCfg(t, nil, func(c *service.Config) { c.ExportsPerHour = 2 })
	exp := func(who domain.Actor) error {
		_, err := w.Svc.Export(w.Ctx, who, w.net.ID, service.ExportMine, service.FormatCSV, &service.MemSink{})
		return err
	}
	tu.Must(0, exp(w.a))
	tu.Must(0, exp(w.a))
	audits := countRows(t, w.Env, "audit_events")
	wantCode(t, exp(w.a), domain.ErrRateLimited)
	if countRows(t, w.Env, "audit_events") != audits {
		t.Error("a refused export was audited")
	}
	tu.Must(0, exp(w.b)) // other members are unaffected
	w.Clock.Advance(61 * time.Minute)
	tu.Must(0, exp(w.a))
}

func TestRedactSecrets(t *testing.T) {
	in := "POST /api/v1/invitations/inv_AbCdEf123456-xyz_9/accept token cir_0123456789abcdef"
	out := service.RedactSecrets(in)
	if strings.Contains(out, "AbCdEf") || strings.Contains(out, "0123456789abcdef") || !strings.Contains(out, "inv_REDACTED") || !strings.Contains(out, "cir_REDACTED") {
		t.Errorf("redacted = %q", out)
	}
	if service.RedactSecrets("/api/v1/members") != "/api/v1/members" {
		t.Error("plain paths must pass through")
	}
}
