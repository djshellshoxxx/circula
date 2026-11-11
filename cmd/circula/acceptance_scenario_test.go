package main

// Acceptance scenario: one mutual-credit lifecycle run through the real
// service, store and CLI commands (migrate, backup, restore, verify-ledger),
// with a restart between the business steps and the backup/restore steps.
// The steps are sequential and share state, so they run as ordered subtests.

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

const (
	scenarioPassword = "correct horse battery"
	// scenarioBackupKey signs the backup manifest. Restore authenticates the
	// manifest against it, so the scenario does not need -allow-unsigned.
	scenarioBackupKey  = "acceptance-scenario-backup-key-32-bytes"
	scenarioDefault    = 10000 // network default credit limit, 100.00
	scenarioAliceLimit = 5000  // Alice's override, 50.00
	scenarioTrade      = 3000  // Alice pays Bob, 30.00
	scenarioPartial    = 1000  // partial reversal of the trade
)

// entrySnap is the part of a ledger entry that must survive unchanged.
type entrySnap struct {
	account string
	amount  int64
	seq     int64
}

type scenario struct {
	t   *testing.T
	ctx context.Context
	cfg *Config
	st  *store.SQLite
	svc *service.Service

	netID string
	// Founder is the network administrator (not a party to any trade).
	founder, alice, bob, carol domain.Actor
	founderM, aliceM, bobM     *domain.Member
	carolM                     *domain.Member

	trade        *domain.Transaction
	tradeEntries []entrySnap
	reversal     *domain.Transaction
	restoredDB   string
}

func (sc *scenario) must(err error, what string) {
	sc.t.Helper()
	if err != nil {
		sc.t.Fatalf("%s: %v", what, err)
	}
}

// need stops a step whose earlier step did not leave the state it depends on.
func (sc *scenario) need(ok bool, what string) {
	sc.t.Helper()
	if !ok {
		sc.t.Fatalf("prerequisite missing: %s", what)
	}
}

// register creates a person and returns an actor for them. The person is not
// signed in yet.
func (sc *scenario) register(login string) domain.Actor {
	sc.t.Helper()
	p, err := sc.svc.RegisterPerson(sc.ctx, service.RegisterInput{Login: login, Password: scenarioPassword})
	sc.must(err, "register "+login)
	return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
}

// signIn is the browser/API sign-in path: password login, then the session
// token is resolved to the actor the adapters would build.
func (sc *scenario) signIn(login string) domain.Actor {
	sc.t.Helper()
	token, _, err := sc.svc.Login(sc.ctx, login, scenarioPassword)
	sc.must(err, "sign in "+login)
	actor, err := sc.svc.Authenticate(sc.ctx, token)
	sc.must(err, "authenticate "+login)
	return actor
}

// join invites a person by an administrator and redeems the invitation.
func (sc *scenario) join(person domain.Actor, name string) *domain.Member {
	sc.t.Helper()
	token, _, err := sc.svc.CreateInvitation(sc.ctx, sc.founder, sc.netID, service.InviteInput{})
	sc.must(err, "invite "+name)
	m, err := sc.svc.RedeemInvitation(sc.ctx, person, token, name)
	sc.must(err, "redeem invitation for "+name)
	return m
}

// balance reads a member's balance as the administrator.
func (sc *scenario) balance(m *domain.Member) *service.BalanceView {
	sc.t.Helper()
	b, err := sc.svc.AdminGetMemberBalance(sc.ctx, sc.founder, sc.netID, m.ID)
	sc.must(err, "admin balance "+m.DisplayName)
	return b
}

func (sc *scenario) entries(actor domain.Actor, txID string) []entrySnap {
	sc.t.Helper()
	v, err := sc.svc.GetTransaction(sc.ctx, actor, sc.netID, txID)
	sc.must(err, "get transaction "+txID)
	out := make([]entrySnap, 0, len(v.Entries))
	for _, e := range v.Entries {
		out = append(out, entrySnap{account: e.AccountID, amount: e.Amount, seq: e.LedgerSeq})
	}
	slices.SortFunc(out, func(a, b entrySnap) int { return strings.Compare(a.account, b.account) })
	return out
}

func hasTx(list []*domain.Transaction, id string) bool {
	return slices.ContainsFunc(list, func(t *domain.Transaction) bool { return t.ID == id })
}

// runCLI runs a circula command in-process and fails the step on a non-zero exit.
func (sc *scenario) runCLI(what string, args ...string) (stdout, stderr string) {
	sc.t.Helper()
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != exitOK {
		sc.t.Fatalf("%s: exit %d: %s", what, code, errOut.String())
	}
	return out.String(), errOut.String()
}

func TestAcceptanceScenario(t *testing.T) {
	t.Setenv("CIRCULA_BACKUP_KEY", scenarioBackupKey)
	dir := t.TempDir()
	db := filepath.Join(dir, "circula.db")
	sc := &scenario{t: t, ctx: context.Background(), cfg: &Config{Database: db}}
	t.Cleanup(func() {
		if sc.st != nil {
			_ = sc.st.Close()
		}
	})

	t.Run("01 start with an initialized sqlite database", func(t *testing.T) {
		// Creating the file migrates it (forward-only); the CLI then reports
		// the schema as current, and the application opens it.
		st, err := store.Open(sc.ctx, db)
		sc.must(err, "create database")
		sc.must(st.Close(), "close bootstrap handle")
		out, _ := sc.runCLI("migrate dry-run", "migrate", "-dry-run", "-database", db)
		if !strings.Contains(out, "schema is up to date") {
			t.Fatalf("schema not current after initialisation: %q", out)
		}
		sc.st, sc.svc, err = openDatabase(sc.ctx, sc.cfg)
		sc.must(err, "start application")
	})

	t.Run("02 create a mutual-credit network", func(t *testing.T) {
		sc.need(sc.svc != nil, "application started")
		sc.founder = sc.register("founder01")
		pol := domain.DefaultPolicy()
		pol.DefaultCreditLimit = scenarioDefault
		net, founderM, err := sc.svc.CreateNetwork(sc.ctx, sc.founder, service.CreateNetworkInput{
			Name: "Acceptance network", Slug: "acceptance-net", Policy: &pol, CreatorDisplayName: "Founder"})
		sc.must(err, "create network")
		net, err = sc.svc.ActivateNetwork(sc.ctx, sc.founder, net.ID)
		sc.must(err, "activate network")
		sc.netID, sc.founderM = net.ID, founderM
		if net.ID == "" || founderM.ID == "" {
			t.Fatalf("network or founder membership missing: %+v %+v", net, founderM)
		}
	})

	t.Run("03 create or invite members Alice, Bob and Carol", func(t *testing.T) {
		sc.need(sc.netID != "", "network created")
		alice, bob, carol := sc.register("alice"), sc.register("bob"), sc.register("carol")
		sc.aliceM = sc.join(alice, "Alice")
		sc.bobM = sc.join(bob, "Bob")
		sc.carolM = sc.join(carol, "Carol")
		for _, m := range []*domain.Member{sc.aliceM, sc.bobM, sc.carolM} {
			got, err := sc.svc.GetMember(sc.ctx, sc.founder, sc.netID, m.ID)
			sc.must(err, "get member "+m.DisplayName)
			if got.ID != m.ID {
				t.Fatalf("member %s not visible after joining", m.DisplayName)
			}
		}
	})

	t.Run("04 configure credit limits", func(t *testing.T) {
		sc.need(sc.aliceM != nil, "members joined")
		limit := int64(scenarioAliceLimit)
		sc.must(sc.svc.SetCreditLimit(sc.ctx, sc.founder, sc.netID, service.SetLimitInput{
			MemberID: sc.aliceM.ID, Limit: &limit, Reason: "acceptance scenario: Alice's limit"}), "set Alice limit")
		if got := sc.balance(sc.aliceM).CreditLimit; got != scenarioAliceLimit {
			t.Errorf("Alice credit limit = %d, want %d", got, scenarioAliceLimit)
		}
		if got := sc.balance(sc.bobM).CreditLimit; got != scenarioDefault {
			t.Errorf("Bob credit limit = %d, want the network default %d", got, scenarioDefault)
		}
	})

	t.Run("05 authenticate Alice", func(t *testing.T) {
		sc.need(sc.aliceM != nil, "members joined")
		sc.alice = sc.signIn("alice")
		b, err := sc.svc.GetBalance(sc.ctx, sc.alice, sc.netID)
		sc.must(err, "Alice own balance")
		if b.MemberID != sc.aliceM.ID || b.CreditLimit != scenarioAliceLimit {
			t.Fatalf("Alice signed in to the wrong member or limit: %+v", b)
		}
	})

	t.Run("06 Alice proposes an exchange with Bob", func(t *testing.T) {
		sc.need(sc.alice.PersonID != "", "Alice signed in")
		tx, err := sc.svc.ProposeTransaction(sc.ctx, sc.alice, sc.netID, service.ProposeInput{
			CounterpartyID: sc.bobM.ID, Direction: service.ProposerPays, Amount: scenarioTrade, IdempotencyKey: "alice-propose-1"})
		sc.must(err, "propose")
		if tx.Status != domain.TxProposed {
			t.Fatalf("status after propose = %s, want proposed", tx.Status)
		}
		sc.trade = tx
		if b := sc.balance(sc.aliceM); b.Balance != 0 {
			t.Errorf("proposal moved Alice's balance to %d before acceptance", b.Balance)
		}
	})

	t.Run("07 Bob authenticates and accepts", func(t *testing.T) {
		sc.need(sc.trade != nil, "trade proposed")
		sc.bob = sc.signIn("bob")
		tx, err := sc.svc.AcceptTransaction(sc.ctx, sc.bob, sc.netID, service.TxRef{TxID: sc.trade.ID, IdempotencyKey: "bob-accept-1"})
		sc.must(err, "accept")
		if tx.Status != domain.TxCompleted || tx.ID != sc.trade.ID {
			t.Fatalf("accept returned status %s id %s", tx.Status, tx.ID)
		}
		sc.trade = tx
	})

	t.Run("08 the transaction posts exactly once", func(t *testing.T) {
		sc.need(sc.trade != nil && sc.trade.Status == domain.TxCompleted, "trade accepted")
		sc.tradeEntries = sc.entries(sc.founder, sc.trade.ID)
		if len(sc.tradeEntries) != 2 {
			t.Fatalf("trade has %d entries, want exactly 2 (payer and payee)", len(sc.tradeEntries))
		}
		var sum int64
		for _, e := range sc.tradeEntries {
			sum += e.amount
		}
		if sum != 0 {
			t.Errorf("trade entries sum to %d, want 0", sum)
		}
		done, err := sc.svc.ListTransactions(sc.ctx, sc.bob, sc.netID, []domain.TxStatus{domain.TxCompleted}, service.Page{})
		sc.must(err, "list Bob's completed transactions")
		count := 0
		for _, tx := range done {
			if tx.ID == sc.trade.ID {
				count++
			}
		}
		if count != 1 {
			t.Errorf("trade appears %d times in Bob's completed history, want 1", count)
		}
	})

	t.Run("09 both balances update correctly", func(t *testing.T) {
		sc.need(sc.trade != nil && sc.trade.Status == domain.TxCompleted, "trade accepted")
		alice, err := sc.svc.GetBalance(sc.ctx, sc.alice, sc.netID)
		sc.must(err, "Alice balance")
		bob, err := sc.svc.GetBalance(sc.ctx, sc.bob, sc.netID)
		sc.must(err, "Bob balance")
		if alice.Balance != -scenarioTrade {
			t.Errorf("Alice balance = %d, want %d (she paid)", alice.Balance, -scenarioTrade)
		}
		if bob.Balance != scenarioTrade {
			t.Errorf("Bob balance = %d, want %d (he was paid)", bob.Balance, scenarioTrade)
		}
		if want := int64(scenarioAliceLimit - scenarioTrade); alice.Available != want {
			t.Errorf("Alice available = %d, want %d", alice.Available, want)
		}
	})

	t.Run("10 sum of all network balances is zero", func(t *testing.T) {
		sc.need(sc.founderM != nil && sc.carolM != nil, "members joined")
		var sum int64
		for _, m := range []*domain.Member{sc.founderM, sc.aliceM, sc.bobM, sc.carolM} {
			sum += sc.balance(m).Balance
		}
		if sum != 0 {
			t.Errorf("sum of member balances = %d, want 0", sum)
		}
		rep, err := sc.svc.VerifyLedger(sc.ctx, sc.founder, sc.netID)
		sc.must(err, "verify ledger")
		if !rep.OK || len(rep.Findings) != 0 {
			t.Errorf("ledger verification failed: %v", rep.Findings)
		}
	})

	t.Run("11 Carol cannot read Alice's private transaction details", func(t *testing.T) {
		sc.need(sc.trade != nil, "trade exists")
		sc.carol = sc.signIn("carol")
		_, err := sc.svc.GetTransaction(sc.ctx, sc.carol, sc.netID, sc.trade.ID)
		if domain.CodeOf(err) != domain.CodeNotFound {
			t.Errorf("Carol GetTransaction error = %v, want not-found", err)
		}
		list, err := sc.svc.ListTransactions(sc.ctx, sc.carol, sc.netID, nil, service.Page{})
		sc.must(err, "Carol's transaction list")
		if hasTx(list, sc.trade.ID) {
			t.Errorf("Carol's history includes Alice and Bob's trade")
		}
		_, err = sc.svc.AdminGetMemberBalance(sc.ctx, sc.carol, sc.netID, sc.aliceM.ID)
		if !isForbidden(err) {
			t.Errorf("Carol reading Alice's balance as admin = %v, want forbidden", err)
		}
	})

	t.Run("12 a duplicate acceptance does not post twice", func(t *testing.T) {
		sc.need(sc.trade != nil, "trade exists")
		before := sc.balance(sc.bobM).Balance
		// Same key and payload: the original result is returned, nothing new posts.
		again, err := sc.svc.AcceptTransaction(sc.ctx, sc.bob, sc.netID, service.TxRef{TxID: sc.trade.ID, IdempotencyKey: "bob-accept-1"})
		sc.must(err, "replay acceptance with the same key")
		if again.ID != sc.trade.ID {
			t.Errorf("replay returned transaction %s, want %s", again.ID, sc.trade.ID)
		}
		// A fresh key on a completed trade is refused as an invalid state.
		_, err = sc.svc.AcceptTransaction(sc.ctx, sc.bob, sc.netID, service.TxRef{TxID: sc.trade.ID, IdempotencyKey: "bob-accept-2"})
		if domain.CodeOf(err) != domain.CodeInvalidState {
			t.Errorf("second acceptance error = %v, want invalid-transaction-state", err)
		}
		if got := sc.entries(sc.founder, sc.trade.ID); len(got) != 2 {
			t.Errorf("trade has %d entries after duplicate acceptance, want 2", len(got))
		}
		if after := sc.balance(sc.bobM).Balance; after != before {
			t.Errorf("Bob balance changed from %d to %d on duplicate acceptance", before, after)
		}
	})

	t.Run("13 a transaction exceeding the credit limit is rejected", func(t *testing.T) {
		sc.need(sc.trade != nil, "trade exists")
		// Alice is at -3000 with a limit of 5000, so 2000 remain available.
		// Proposing 2500 would take her to -5500, beyond the limit.
		_, err := sc.svc.ProposeTransaction(sc.ctx, sc.alice, sc.netID, service.ProposeInput{
			CounterpartyID: sc.bobM.ID, Direction: service.ProposerPays, Amount: 2500, IdempotencyKey: "alice-over-limit"})
		if domain.CodeOf(err) != domain.CodeInsufficientCredit {
			t.Fatalf("over-limit proposal error = %v, want insufficient-available-credit", err)
		}
		if b := sc.balance(sc.aliceM); b.Balance != -scenarioTrade {
			t.Errorf("Alice balance = %d after refused proposal, want %d", b.Balance, -scenarioTrade)
		}
	})

	t.Run("14 a valid reversal preserves the audit trail", func(t *testing.T) {
		sc.need(sc.trade != nil, "trade exists")
		rev, err := sc.svc.ReverseTransaction(sc.ctx, sc.founder, sc.netID, service.ReverseInput{
			TxID: sc.trade.ID, Amount: scenarioPartial, Reason: "acceptance scenario: partial reversal", IdempotencyKey: "founder-reverse-1"})
		sc.must(err, "reverse")
		sc.reversal = rev
		if rev.Kind != domain.KindReversal || rev.OriginalTxID != sc.trade.ID {
			t.Fatalf("reversal kind %s original %q, want reversal of %s", rev.Kind, rev.OriginalTxID, sc.trade.ID)
		}
		orig, err := sc.svc.GetTransaction(sc.ctx, sc.founder, sc.netID, sc.trade.ID)
		sc.must(err, "get original")
		if orig.Tx.Amount != scenarioTrade || orig.Tx.PayerMemberID != sc.aliceM.ID || orig.Tx.PayeeMemberID != sc.bobM.ID {
			t.Errorf("original trade changed: amount %d payer %s payee %s", orig.Tx.Amount, orig.Tx.PayerMemberID, orig.Tx.PayeeMemberID)
		}
		if orig.Tx.ReversedTotal != scenarioPartial {
			t.Errorf("original reversed total = %d, want %d", orig.Tx.ReversedTotal, scenarioPartial)
		}
		if got := sc.entries(sc.founder, sc.trade.ID); !slices.Equal(got, sc.tradeEntries) {
			t.Errorf("original postings changed by the reversal: before %v after %v", sc.tradeEntries, got)
		}
		audit, err := sc.svc.ListAudit(sc.ctx, sc.founder, sc.netID, service.AuditQuery{Action: domain.ActTxReversed, TargetID: rev.ID})
		sc.must(err, "audit for reversal")
		if len(audit) != 1 {
			t.Errorf("reversal audit events = %d, want 1", len(audit))
		}
		if a, b := sc.balance(sc.aliceM).Balance, sc.balance(sc.bobM).Balance; a != -(scenarioTrade-scenarioPartial) || b != scenarioTrade-scenarioPartial {
			t.Errorf("balances after reversal: Alice %d Bob %d", a, b)
		}
	})

	t.Run("15 an unauthorised administrative adjustment is rejected", func(t *testing.T) {
		sc.need(sc.bob.PersonID != "", "Bob signed in")
		before := sc.balance(sc.aliceM).Balance
		_, err := sc.svc.CreateAdjustment(sc.ctx, sc.bob, sc.netID, service.AdjustInput{
			MemberID: sc.aliceM.ID, Amount: 500, Direction: service.AdjustCredit,
			Reason: "acceptance scenario: Bob is not an administrator", IdempotencyKey: "bob-adjust-1"})
		if !isForbidden(err) {
			t.Fatalf("adjustment by a member = %v, want forbidden", err)
		}
		if after := sc.balance(sc.aliceM).Balance; after != before {
			t.Errorf("refused adjustment changed Alice's balance from %d to %d", before, after)
		}
	})

	t.Run("16 restart the application", func(t *testing.T) {
		sc.need(sc.st != nil, "application running")
		sc.must(sc.st.Close(), "close store")
		var err error
		sc.st, sc.svc, err = openDatabase(sc.ctx, sc.cfg)
		sc.must(err, "reopen application")
	})

	t.Run("17 balances and transaction history persist", func(t *testing.T) {
		sc.need(sc.reversal != nil, "reversal posted before restart")
		if a, b := sc.balance(sc.aliceM).Balance, sc.balance(sc.bobM).Balance; a != -(scenarioTrade-scenarioPartial) || b != scenarioTrade-scenarioPartial {
			t.Errorf("balances after restart: Alice %d Bob %d", a, b)
		}
		if got := sc.entries(sc.founder, sc.trade.ID); !slices.Equal(got, sc.tradeEntries) {
			t.Errorf("postings changed across restart: before %v after %v", sc.tradeEntries, got)
		}
		list, err := sc.svc.ListTransactions(sc.ctx, sc.alice, sc.netID, nil, service.Page{})
		sc.must(err, "Alice history after restart")
		if !hasTx(list, sc.trade.ID) || !hasTx(list, sc.reversal.ID) {
			t.Errorf("Alice history after restart lacks the trade or its reversal")
		}
		if got := sc.balance(sc.aliceM).Balance; got == 0 {
			t.Errorf("Alice balance unexpectedly zero after restart")
		}
		// Credentials survive the restart too.
		if _, _, err := sc.svc.Login(sc.ctx, "alice", scenarioPassword); err != nil {
			t.Errorf("Alice cannot sign in after restart: %v", err)
		}
	})

	var backupFile string
	restoreDir := t.TempDir()
	t.Run("18 back up the database", func(t *testing.T) {
		backupFile = filepath.Join(dir, "acceptance.bak")
		out, _ := sc.runCLI("backup", "backup", "-database", db, backupFile)
		if !strings.Contains(out, "ledger: ok") {
			t.Fatalf("backup did not report a passing ledger: %q", out)
		}
	})

	t.Run("19 restore into an isolated database", func(t *testing.T) {
		sc.need(backupFile != "", "backup written")
		// A new file in an empty directory, owned by the top-level test so it
		// outlives this subtest (step 20 reads it).
		sc.restoredDB = filepath.Join(restoreDir, "circula.db")
		sc.runCLI("restore", "restore", "-database", sc.restoredDB, backupFile)
	})

	t.Run("20 verify ledger reconciliation and historical integrity on the restored copy", func(t *testing.T) {
		sc.need(sc.restoredDB != "", "restore completed")
		sc.runCLI("verify-ledger", "verify-ledger", "-database", sc.restoredDB)
		// Keep the original store open until the restored copy has been
		// checked: the restored file is a separate database.
		rst, rsvc, err := openDatabase(sc.ctx, &Config{Database: sc.restoredDB})
		sc.must(err, "open restored copy")
		defer func() { _ = rst.Close() }()
		rep, err := rsvc.VerifyLedger(sc.ctx, sc.founder, sc.netID)
		sc.must(err, "verify restored ledger")
		if !rep.OK || len(rep.Findings) != 0 {
			t.Errorf("restored ledger findings: %v", rep.Findings)
		}
		restoredEntries := func() []entrySnap {
			v, err := rsvc.GetTransaction(sc.ctx, sc.founder, sc.netID, sc.trade.ID)
			sc.must(err, "restored trade")
			out := make([]entrySnap, 0, len(v.Entries))
			for _, e := range v.Entries {
				out = append(out, entrySnap{account: e.AccountID, amount: e.Amount, seq: e.LedgerSeq})
			}
			slices.SortFunc(out, func(a, b entrySnap) int { return strings.Compare(a.account, b.account) })
			return out
		}
		if got := restoredEntries(); !slices.Equal(got, sc.tradeEntries) {
			t.Errorf("restored postings differ: want %v got %v", sc.tradeEntries, got)
		}
		orig, err := rsvc.GetTransaction(sc.ctx, sc.founder, sc.netID, sc.trade.ID)
		sc.must(err, "restored original")
		if orig.Tx.ReversedTotal != scenarioPartial {
			t.Errorf("restored reversed total = %d, want %d", orig.Tx.ReversedTotal, scenarioPartial)
		}
		list, err := rsvc.ListTransactions(sc.ctx, sc.alice, sc.netID, nil, service.Page{})
		sc.must(err, "restored Alice history")
		if !hasTx(list, sc.trade.ID) || !hasTx(list, sc.reversal.ID) {
			t.Errorf("restored history lacks the trade or its reversal")
		}
	})
}

// isForbidden reports whether err is the service's forbidden refusal.
func isForbidden(err error) bool { return domain.CodeOf(err) == domain.CodeForbidden }
