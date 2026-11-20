package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// QA-024 and QA-028 (OPS-017, OPS-020): a backup taken while trades post
// concurrently restores into a clean instance and into a destroyed one, with
// every posting committed before the backup started present, balances equal
// to an independent replay of the trades in the backup, ledger and audit
// verification passing, and a database.restored audit event in every
// network. A seeded corrupted backup is refused, and a corrupted database put
// in place without restore keeps its network frozen for postings until
// verification passes again.

var qa28Argon = service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}

type qa28Trade struct {
	id, payer, payee string // payer and payee are account ids
	amount           int64
}

type qa28Party struct {
	actor domain.Actor
	m     *domain.Member
	netID string
}

type qa28Instance struct {
	path    string
	st      *store.SQLite
	svc     *service.Service
	nets    []string
	parties [][]qa28Party // per network
	admins  []domain.Actor
}

func qa28Seed(t *testing.T, dir string) *qa28Instance {
	t.Helper()
	ctx := context.Background()
	in := &qa28Instance{path: filepath.Join(dir, "circula.db")}
	var err error
	if in.st, err = store.Open(ctx, in.path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = in.st.Close() })
	if in.svc, err = service.New(in.st, service.Config{Argon: qa28Argon}); err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	person := func(login string) domain.Actor {
		p, err := in.svc.RegisterPerson(ctx, service.RegisterInput{Login: login, Password: "correct horse battery"})
		must(err)
		return domain.Actor{PersonID: p.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}}
	}
	for ni := 0; ni < 2; ni++ {
		admin := person(fmt.Sprintf("admin%d@example.org", ni))
		pol := domain.DefaultPolicy()
		pol.DefaultCreditLimit = 1_000_000
		n, _, err := in.svc.CreateNetwork(ctx, admin, service.CreateNetworkInput{Name: fmt.Sprintf("Load %d", ni),
			Slug: fmt.Sprintf("load-%d", ni), Policy: &pol, CreatorDisplayName: "Founder"})
		must(err)
		_, err = in.svc.ActivateNetwork(ctx, admin, n.ID)
		must(err)
		var ps []qa28Party
		for i := 0; i < 4; i++ {
			login := fmt.Sprintf("n%dm%d@example.org", ni, i)
			a := person(login)
			tok, _, err := in.svc.CreateInvitation(ctx, admin, n.ID, service.InviteInput{})
			must(err)
			m, err := in.svc.RedeemInvitation(ctx, a, tok, login)
			must(err)
			ps = append(ps, qa28Party{a, m, n.ID})
		}
		in.nets, in.parties, in.admins = append(in.nets, n.ID), append(in.parties, ps), append(in.admins, admin)
	}
	return in
}

// load runs one worker per network pair until stop closes, recording every
// trade whose accept returned (committed) in order.
func (in *qa28Instance) load(t *testing.T, stop <-chan struct{}) (wait func() []qa28Trade, committed *atomic.Int64) {
	ctx := context.Background()
	var mu sync.Mutex
	var trades []qa28Trade
	committed = &atomic.Int64{}
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		ps := in.parties[w%2]
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				payer, payee := ps[(w+i)%len(ps)], ps[(w+i+1)%len(ps)]
				amt := int64(1 + (w*7+i*13)%500)
				p, err := in.svc.ProposeTransaction(ctx, payer.actor, payer.netID, service.ProposeInput{CounterpartyID: payee.m.ID,
					Direction: service.ProposerPays, Amount: amt, IdempotencyKey: fmt.Sprintf("qa28-p-%d-%d", w, i)})
				if err != nil {
					t.Errorf("worker %d propose: %v", w, err)
					return
				}
				tx, err := in.svc.AcceptTransaction(ctx, payee.actor, payee.netID, service.TxRef{TxID: p.ID, IdempotencyKey: fmt.Sprintf("qa28-a-%d-%d", w, i)})
				if err != nil {
					t.Errorf("worker %d accept: %v", w, err)
					return
				}
				mu.Lock()
				trades = append(trades, qa28Trade{tx.ID, tx.PayerAccountID, tx.PayeeAccountID, tx.Amount})
				mu.Unlock()
				committed.Add(1)
			}
		}(w)
	}
	return func() []qa28Trade {
		wg.Wait()
		mu.Lock()
		defer mu.Unlock()
		return append([]qa28Trade(nil), trades...)
	}, committed
}

func qa28Run(t *testing.T, what string, wantCode int, args ...string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != wantCode {
		t.Fatalf("%s: exit %d, want %d\nstdout: %s\nstderr: %s", what, code, wantCode, out.String(), errOut.String())
	}
	return out.String() + errOut.String()
}

// qa28Check opens a restored database and checks it against the trades the
// test itself recorded: every trade committed before the backup started is
// present, no unknown trade is, and each cached balance equals the replay of
// the trades present (an oracle independent of the service).
func qa28Check(t *testing.T, path string, before, all []qa28Trade) {
	t.Helper()
	db, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	present := map[string]bool{}
	rows, err := db.Query(`SELECT id FROM transactions WHERE kind = 'trade' AND status = 'completed'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		present[id] = true
	}
	_ = rows.Close()
	for _, tr := range before {
		if !present[tr.id] {
			t.Errorf("trade %s committed before the backup is missing from the restore", tr.id)
		}
	}
	known := map[string]qa28Trade{}
	for _, tr := range all {
		known[tr.id] = tr
	}
	want := map[string]int64{}
	for id := range present {
		tr, ok := known[id]
		if !ok {
			t.Errorf("restored trade %s was never committed by the load", id)
			continue
		}
		want[tr.payer] -= tr.amount
		want[tr.payee] += tr.amount
	}
	rows, err = db.Query(`SELECT id, balance FROM accounts WHERE is_system = 0`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var bal int64
		if err := rows.Scan(&id, &bal); err != nil {
			t.Fatal(err)
		}
		if bal != want[id] {
			t.Errorf("account %s: restored balance %d, replay of the backed-up trades gives %d", id, bal, want[id])
		}
	}
	if len(present) < len(before) {
		t.Errorf("restore holds %d trades, %d were committed before the backup", len(present), len(before))
	}
}

func qa28RestoredEvents(t *testing.T, path string, nets []string) {
	t.Helper()
	db, err := openRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, n := range nets {
		var c int
		if err := db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = 'database.restored'`, n).Scan(&c); err != nil || c != 1 {
			t.Errorf("network %s: %d database.restored events (err=%v), want 1", n, c, err)
		}
	}
}

func TestQA028BackupUnderLoadAndRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	in := qa28Seed(t, dir)
	cfg := testConfigFor(in.path)

	stop := make(chan struct{})
	wait, committed := in.load(t, stop)
	for committed.Load() < 40 { // let the load build up first
		time.Sleep(5 * time.Millisecond)
	}
	// Everything recorded up to here committed before the backup started.
	nBefore := committed.Load()
	mf, backup, err := backupTo(ctx, cfg, cfg.BackupDir, "manual", in.st.DB())
	if err != nil {
		close(stop)
		wait()
		t.Fatalf("backup under load: %v", err)
	}
	during := committed.Load() - nBefore
	for committed.Load() < nBefore+during+20 { // and keep posting after it
		time.Sleep(5 * time.Millisecond)
	}
	close(stop)
	all := wait()
	if t.Failed() {
		t.FailNow()
	}
	before := all[:nBefore]
	t.Logf("load: %d trades before the backup, %d while it ran, %d in total", nBefore, during, len(all))

	// The backup verifies on its own: manifest checksum, integrity, ledger.
	m, err := readManifest(backup + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupFile(backup, m); err != nil {
		t.Fatalf("manifest checksum: %v", err)
	}
	if mf.Integrity != "ok" || mf.Ledger.Status != "ok" || mf.Ledger.Networks != 2 {
		t.Fatalf("manifest = %+v", mf)
	}

	// QA-028: restore into a clean instance.
	clean := filepath.Join(t.TempDir(), "clean.db")
	began := time.Now()
	qa28Run(t, "restore into a clean instance", exitOK, "restore", "-allow-unsigned", "-database", clean, backup)
	t.Logf("QA-028 restore duration (clean instance, %d trades, %d bytes): %s", len(all), m.Bytes, time.Since(began))
	if res, err := integrityCheck(ctx, clean); err != nil || res != "ok" {
		t.Fatalf("restored integrity: %q %v", res, err)
	}
	qa28Run(t, "verify-ledger on the clean restore", exitOK, "verify-ledger", "-database", clean)
	qa28Check(t, clean, before, all)
	qa28RestoredEvents(t, clean, in.nets)

	// QA-024: the instance is destroyed, then restored from the backup.
	if err := in.st.Close(); err != nil {
		t.Fatal(err)
	}
	removeDBFiles(in.path)
	if _, err := os.Stat(in.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("instance not destroyed: %v", err)
	}
	qa28Run(t, "restore into the destroyed instance", exitOK, "restore", "-allow-unsigned", "-database", in.path, backup)
	qa28Run(t, "verify-ledger after restore", exitOK, "verify-ledger", "-database", in.path)
	qa28Check(t, in.path, before, all)
	qa28RestoredEvents(t, in.path, in.nets)

	// The restored instance takes postings again.
	st, err := store.Open(ctx, in.path)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := service.New(st, service.Config{Argon: qa28Argon})
	if err != nil {
		t.Fatal(err)
	}
	a, b := in.parties[0][0], in.parties[0][1]
	if _, err := svc.ProposeTransaction(ctx, a.actor, a.netID, service.ProposeInput{CounterpartyID: b.m.ID, Direction: service.ProposerPays,
		Amount: 5, IdempotencyKey: "after-restore"}); err != nil {
		t.Fatalf("posting after restore: %v", err)
	}
	_ = st.Close()
}

// qa28Corrupt copies a backup, raises one cached balance in the copy and
// writes a manifest whose checksum matches the corrupted bytes, so only the
// ledger check inside the backup can catch it.
func qa28Corrupt(t *testing.T, ctx context.Context, src, dst string) {
	t.Helper()
	if err := copyFileSync(src, dst); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE accounts SET balance = balance + 1 WHERE rowid = (SELECT MIN(rowid) FROM accounts WHERE is_system = 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	removeSideFiles(dst)
	m, err := readManifest(src + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	m.File = filepath.Base(dst)
	if m.SHA256, m.Bytes, err = fileSHA256(dst); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(dst+".manifest.json", m); err != nil {
		t.Fatal(err)
	}
}

func TestQA028CorruptedBackupRefusedAndPostingsFrozen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	in := qa28Seed(t, dir)
	a, b := in.parties[0][0], in.parties[0][1]
	propose := func(svc *service.Service, key string) error {
		_, err := svc.ProposeTransaction(ctx, a.actor, a.netID, service.ProposeInput{CounterpartyID: b.m.ID, Direction: service.ProposerPays,
			Amount: 10, IdempotencyKey: key})
		return err
	}
	if err := propose(in.svc, "seed"); err != nil {
		t.Fatal(err)
	}
	cfg := testConfigFor(in.path)
	_, good, err := backupTo(ctx, cfg, cfg.BackupDir, "manual", in.st.DB())
	if err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "corrupted.db")
	qa28Corrupt(t, ctx, good, bad)

	// The restore refuses the corrupted backup and creates nothing.
	target := filepath.Join(t.TempDir(), "target.db")
	out := qa28Run(t, "restore of a corrupted backup", exitLedgerFailed, "restore", "-allow-unsigned", "-database", target, bad)
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupted restore left a database behind (%v): %s", err, out)
	}

	// Copied in place by hand instead, the corrupted file fails verification
	// at start-up and its network is frozen: postings are refused...
	if err := copyFileSync(bad, target); err != nil {
		t.Fatal(err)
	}
	tcfg := testConfigFor(target)
	open := func() (*store.SQLite, *service.Service) {
		st, err := store.Open(ctx, target)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		svc, err := newService(st, tcfg)
		if err != nil {
			t.Fatal(err)
		}
		checks, err := verifyAllNetworks(ctx, svc, "startup", "")
		if err != nil {
			t.Fatal(err)
		}
		applyFreeze(svc, checks, discardLogger())
		return st, svc
	}
	st, svc := open()
	if !svc.IsFrozen(a.netID) {
		t.Fatal("network with a corrupted balance is not frozen")
	}
	if err := propose(svc, "frozen"); !errors.Is(err, domain.ErrNetworkFrozen) {
		t.Fatalf("posting on a corrupted ledger: %v, want network-frozen", err)
	}
	if svc.IsFrozen(in.nets[1]) {
		t.Error("the healthy network was frozen too")
	}
	_ = st.Close()

	// ...until the verified backup is restored and verification passes.
	qa28Run(t, "restore of the verified backup", exitOK, "restore", "-allow-unsigned", "-overwrite", "-database", target, good)
	st, svc = open()
	defer func() { _ = st.Close() }()
	if svc.IsFrozen(a.netID) {
		t.Fatal("network still frozen after a verified restore")
	}
	if err := propose(svc, "thawed"); err != nil {
		t.Fatalf("posting after verification passes: %v", err)
	}
}
