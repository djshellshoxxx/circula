package store_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

func openStore(t *testing.T) *store.SQLite {
	t.Helper()
	s, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// seed creates a network with two members, each with an account, and one
// proposed trade of 100 from a to b. It returns the ids.
func seed(t testing.TB, s *store.SQLite) (netID, txID, accA, accB string) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	netID, txID, accA, accB = "n1", "t1", "a1", "a2"
	err := s.Update(ctx, func(tx store.Tx) error {
		for _, p := range []string{"p1", "p2"} {
			if err := tx.InsertPerson(&domain.Person{ID: p, Login: p, CreatedAt: now}, "h"); err != nil {
				return err
			}
		}
		if err := tx.InsertNetwork(&domain.Network{ID: netID, Name: "N", Slug: "n", Status: domain.NetworkActive, CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.AppendPolicy(netID, 1, domain.DefaultPolicy(), "", "", now); err != nil {
			return err
		}
		for i, p := range []string{"p1", "p2"} {
			m := &domain.Member{ID: "m" + p, NetworkID: netID, PersonID: p, DisplayName: p, Status: domain.MemberActive, JoinMethod: "founder", JoinedAt: now}
			if err := tx.InsertMember(m); err != nil {
				return err
			}
			acc := []string{accA, accB}[i]
			if err := tx.InsertAccount(&domain.Account{ID: acc, NetworkID: netID, MemberID: m.ID}); err != nil {
				return err
			}
		}
		exp := now.Add(time.Hour)
		return tx.InsertTransaction(&domain.Transaction{ID: txID, NetworkID: netID, Kind: domain.KindTrade,
			PayerAccountID: accA, PayeeAccountID: accB, Amount: 100, CreatedBy: "mp1", ProposedAt: now, ExpiresAt: &exp})
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestMigrationsIdempotentAndVersioned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.db")
	for i := 0; i < 3; i++ { // reopening re-runs the runner without error
		s, err := store.Open(context.Background(), path)
		if err != nil {
			t.Fatalf("open #%d: %v", i, err)
		}
		var v int
		if err := s.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v); err != nil || v != store.SchemaVersion() {
			t.Fatalf("version = %d err=%v want %d", v, err, store.SchemaVersion())
		}
		_ = s.Close()
	}
	s, _ := store.Open(context.Background(), path)
	defer func() { _ = s.Close() }()
	if _, err := s.DB().Exec(`INSERT INTO schema_migrations(version,name,applied_at) VALUES (99,'future',0)`); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	if _, err := store.Open(context.Background(), path); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("expected refusal for newer schema, got %v", err)
	}
}

// post inserts both entries and completes the transaction.
func post(t *testing.T, s *store.SQLite, netID, txID, accA, accB string) {
	t.Helper()
	err := s.Update(context.Background(), func(tx store.Tx) error {
		now := time.Now()
		if err := tx.InsertEntry(&domain.Entry{ID: "e1", NetworkID: netID, TxID: txID, AccountID: accA, Amount: -100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		if err := tx.InsertEntry(&domain.Entry{ID: "e2", NetworkID: netID, TxID: txID, AccountID: accB, Amount: 100, LedgerSeq: 1, PostedAt: now}); err != nil {
			return err
		}
		x, err := tx.GetTransaction(netID, txID)
		if err != nil {
			return err
		}
		x.Status = domain.TxCompleted
		return tx.UpdateTransaction(x)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTriggersBlockUpdateDeleteOnAppendOnlyTables(t *testing.T) {
	s := openStore(t)
	netID, txID, a, b := seed(t, s)
	post(t, s, netID, txID, a, b)
	ctx := context.Background()
	if err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertStatusEvent(&domain.StatusEvent{ID: "se1", NetworkID: netID, TxID: txID, To: domain.TxCompleted, OccurredAt: time.Now()}); err != nil {
			return err
		}
		return tx.InsertAudit(&domain.AuditEvent{ID: "au1", NetworkID: netID, Seq: 1, OccurredAt: time.Now(), ActorType: domain.ActorSystem,
			Action: "x", TargetType: "t", TargetID: "1", Hash: "h", SubjectIDs: []string{"mp1"}})
	}); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ table, set string }{
		{"entries", "amount = 5"}, {"audit_events", "reason = 'x'"}, {"transaction_status_events", "to_status = 'x'"},
		{"audit_subjects", "member_id = 'zz'"}, {"network_policies", "reason = 'x'"},
	}
	for _, c := range cases {
		if _, err := s.DB().Exec("UPDATE " + c.table + " SET " + c.set); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("UPDATE %s: want append-only error, got %v", c.table, err)
		}
		if _, err := s.DB().Exec("DELETE FROM " + c.table); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("DELETE %s: want append-only error, got %v", c.table, err)
		}
	}
	if _, err := s.DB().Exec("DELETE FROM transactions"); err == nil {
		t.Error("transactions must not be deletable")
	}
	var n int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("entries intact: n=%d err=%v", n, err)
	}
}

func TestTransactionIntegrityTriggers(t *testing.T) {
	s := openStore(t)
	netID, txID, a, b := seed(t, s)
	db := s.DB()
	// Cannot complete without entries (DAT-007).
	if _, err := db.Exec(`UPDATE transactions SET status='completed' WHERE id=?`, txID); err == nil {
		t.Error("completing without entries must fail")
	}
	// Unbalanced / wrong-amount entry refused.
	if _, err := db.Exec(`INSERT INTO entries VALUES ('x',?,?,?,-99,1,0)`, netID, txID, a); err == nil {
		t.Error("entry with wrong amount must fail")
	}
	// A single entry cannot complete the transaction.
	if _, err := db.Exec(`INSERT INTO entries VALUES ('x',?,?,?,-100,1,0)`, netID, txID, a); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE transactions SET status='completed' WHERE id=?`, txID); err == nil {
		t.Error("completing with one entry must fail")
	}
	// Terms immutable (DOM-013).
	for _, set := range []string{"amount = 1", "payer_account_id = '" + b + "'", "kind = 'adjustment'"} {
		if _, err := db.Exec(`UPDATE transactions SET ` + set + ` WHERE id = '` + txID + `'`); err == nil {
			t.Errorf("terms update %q must fail", set)
		}
	}
	// Illegal status jump.
	if _, err := db.Exec(`UPDATE transactions SET status='reversed' WHERE id=?`, txID); err == nil {
		t.Error("proposed -> reversed must fail")
	}
	// Inserting directly as completed is refused.
	if _, err := db.Exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at)
		VALUES ('t9',?,'trade','completed',?,?,5,0)`, netID, a, b); err == nil {
		t.Error("insert as completed must fail")
	}
}

func TestCrossNetworkReferencesRejected(t *testing.T) {
	s := openStore(t)
	seed(t, s)
	err := s.Update(context.Background(), func(tx store.Tx) error {
		now := time.Now()
		if err := tx.InsertNetwork(&domain.Network{ID: "n2", Name: "Other", Slug: "o", Status: domain.NetworkActive, CreatedAt: now}); err != nil {
			return err
		}
		if err := tx.InsertAccount(&domain.Account{ID: "sys2", NetworkID: "n2", IsSystem: true}); err != nil {
			return err
		}
		// payer in n1, payee in n2: composite FK must reject it (DOM-001, DAT-005).
		return tx.InsertTransaction(&domain.Transaction{ID: "tx", NetworkID: "n1", Kind: domain.KindTrade,
			PayerAccountID: "a1", PayeeAccountID: "sys2", Amount: 1, ProposedAt: now})
	})
	if err == nil {
		t.Fatal("cross-network transaction must be rejected")
	}
}

func TestUpdateRollsBackOnError(t *testing.T) {
	s := openStore(t)
	netID, _, _, _ := seed(t, s)
	boom := context.Canceled
	err := s.Update(context.Background(), func(tx store.Tx) error {
		if err := tx.SetNetworkStatus(netID, domain.NetworkClosed); err != nil {
			return err
		}
		return boom
	})
	if err != boom {
		t.Fatalf("want boom, got %v", err)
	}
	_ = s.View(context.Background(), func(tx store.Tx) error {
		n, _ := tx.GetNetwork(netID)
		if n.Status != domain.NetworkActive {
			t.Errorf("status = %s, rollback failed", n.Status)
		}
		return nil
	})
}
