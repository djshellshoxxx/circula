package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// QA-028 (OPS-017, OPS-020): a backup taken while trades are posting restores
// into a clean instance with zero integrity errors, the ledger verifies, the
// manifest checksum matches, and every trade committed before the backup
// started is present. The restore duration is logged for the performance
// record (DEC-016).
func TestQA028BackupUnderLoadRestores(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}
	dir := t.TempDir()
	db := filepath.Join(dir, "circula.db")
	env := tu.OpenEnvAt(t, db, nil, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 0)
	defer func() { _ = env.Store.Close() }()
	admin := env.Person("admin@example.org")
	net, _ := env.Network(admin, "load", func(p *domain.Policy) { p.DefaultCreditLimit = 1_000_000_000 })
	type pair struct {
		a, b   domain.Actor
		bm     *domain.Member
		posted atomic.Int64
	}
	pairs := make([]*pair, 4)
	for i := range pairs {
		a, _ := env.Member(admin, net, fmt.Sprintf("a%d@example.org", i))
		b, bm := env.Member(admin, net, fmt.Sprintf("b%d@example.org", i))
		pairs[i] = &pair{a: a, b: b, bm: bm}
	}
	trade := func(p *pair) error {
		tx, err := env.Svc.ProposeTransaction(env.Ctx, p.a, net.ID, service.ProposeInput{CounterpartyID: p.bm.ID,
			Direction: service.ProposerPays, Amount: 7, IdempotencyKey: env.Key()})
		if err != nil {
			return err
		}
		if _, err := env.Svc.AcceptTransaction(env.Ctx, p.b, net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: env.Key()}); err != nil {
			return err
		}
		p.posted.Add(1)
		return nil
	}
	for _, p := range pairs { // some history before the load starts
		for i := 0; i < 5; i++ {
			if err := trade(p); err != nil {
				t.Fatal(err)
			}
		}
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var loadErr atomic.Value
	for _, p := range pairs {
		wg.Add(1)
		go func(p *pair) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if err := trade(p); err != nil && domain.CodeOf(err) != domain.CodeUnavailable {
					loadErr.Store(err)
					return
				}
			}
		}(p)
	}
	time.Sleep(200 * time.Millisecond) // the load is running
	committedBefore := completedTrades(t, env.Store.DB(), net.ID)
	dest := filepath.Join(dir, "under-load.db")
	var out, errOut bytes.Buffer
	if code := run([]string{"backup", "-database", db, dest}, &out, &errOut); code != 0 {
		close(stop)
		wg.Wait()
		t.Fatalf("backup under load: exit %d: %s", code, errOut.String())
	}
	time.Sleep(100 * time.Millisecond) // the load continues past the backup
	close(stop)
	wg.Wait()
	if err, _ := loadErr.Load().(error); err != nil {
		t.Fatalf("load: %v", err)
	}
	var total int64
	for _, p := range pairs {
		total += p.posted.Load()
	}
	if total <= committedBefore {
		t.Fatalf("test premise: no trades during the backup (%d before, %d total)", committedBefore, total)
	}

	m, err := readManifest(dest + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupFile(dest, m); err != nil {
		t.Fatalf("manifest checksum: %v", err)
	}
	if m.Integrity != "ok" || m.Ledger.Status != "ok" {
		t.Fatalf("manifest: integrity %q ledger %q", m.Integrity, m.Ledger.Status)
	}

	clean := filepath.Join(t.TempDir(), "restored.db")
	began := time.Now()
	out.Reset()
	errOut.Reset()
	if code := run([]string{"restore", "-allow-unsigned", "-database", clean, dest}, &out, &errOut); code != 0 {
		t.Fatalf("restore: exit %d: %s", code, errOut.String())
	}
	t.Logf("QA-028 restore duration: %s (%d trades in the backup's source while it ran)", time.Since(began), total)
	if res, err := store.IntegrityCheck(context.Background(), clean); err != nil || res != "ok" {
		t.Fatalf("restored integrity: %q %v", res, err)
	}
	if code := run([]string{"verify-ledger", "-database", clean}, &out, &errOut); code != 0 {
		t.Fatalf("verify-ledger on the restored database: exit %d: %s", code, errOut.String())
	}
	raw, err := openRaw(clean)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if got := completedTrades(t, raw, net.ID); got < committedBefore {
		t.Fatalf("restored %d completed trades, %d were committed before the backup began", got, committedBefore)
	}
	var sum int64
	if err := raw.QueryRow(`SELECT COALESCE(SUM(balance), 0) FROM accounts WHERE network_id = ?`, net.ID).Scan(&sum); err != nil || sum != 0 {
		t.Fatalf("restored network balance sum = %d (%v)", sum, err)
	}
}

func completedTrades(t *testing.T, db *sql.DB, netID string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE network_id = ? AND kind = 'trade' AND status = 'completed'`, netID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
