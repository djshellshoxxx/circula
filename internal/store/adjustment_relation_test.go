package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

func TestAdjustmentRelationDatabaseGuards(t *testing.T) {
	e := testutil.NewEnv(t)
	admin := e.Person("admin")
	net, am := e.Network(admin, "relations", nil)
	a, ma := e.Member(admin, net, "alice")
	b, mb := e.Member(admin, net, "bob")
	original := e.Trade(net.ID, a, b, ma, mb, 100)
	unrelated := e.Trade(net.ID, admin, b, am, mb, 100)
	unposted := testutil.Must(e.Svc.ProposeTransaction(e.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: mb.ID, Direction: service.ProposerPays, Amount: 100, IdempotencyKey: e.Key()}))
	// A sequence alone does not make an unposted proposal a valid original.
	testutil.Must(0, e.Store.DB().QueryRow(`UPDATE transactions SET ledger_seq=999 WHERE id=? RETURNING ledger_seq`, unposted.ID).Scan(new(int64)))
	other, otherAdmin := e.Network(admin, "other", nil)
	c, mc := e.Member(admin, other, "charlie")
	cross := e.Trade(other.ID, admin, c, otherAdmin, mc, 100)
	var sys, account string
	testutil.Must(0, e.Store.DB().QueryRow(`SELECT id FROM accounts WHERE network_id=? AND is_system=1`, net.ID).Scan(&sys))
	testutil.Must(0, e.Store.DB().QueryRow(`SELECT id FROM accounts WHERE network_id=? AND member_id=?`, net.ID, ma.ID).Scan(&account))
	for _, id := range []string{"missing", unrelated.ID, unposted.ID, cross.ID} {
		_, err := e.Store.DB().Exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,reason,proposed_at,related_tx_id) VALUES ('bad',?,'adjustment','proposed',?,?,1,'correction',1,?)`, net.ID, sys, account, id)
		if err == nil {
			t.Fatalf("database accepted invalid relation %s", id)
		}
		_, err = e.Store.DB().Exec(`INSERT INTO pending_adjustments(id,network_id,proposer_member_id,target_member_id,amount,direction,reason,created_at,expires_at,related_tx_id) VALUES ('bad-pending',?,?,?,1,'credit','correction',1,2,?)`, net.ID, am.ID, ma.ID, id)
		if err == nil {
			t.Fatalf("database accepted invalid pending relation %s", id)
		}
	}
	_, err := e.Store.DB().Exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,proposed_at,related_tx_id) VALUES ('bad-kind',?,'trade','proposed',?,?,1,1,?)`, net.ID, sys, account, original.ID)
	if err == nil {
		t.Fatal("non-adjustment relation accepted")
	}
	// A malformed member-to-member adjustment must not reveal the relation to
	// an extra member merely because its other account is a related party.
	var adminAccount string
	testutil.Must(0, e.Store.DB().QueryRow(`SELECT id FROM accounts WHERE network_id=? AND member_id=?`, net.ID, am.ID).Scan(&adminAccount))
	_, err = e.Store.DB().Exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,reason,proposed_at,related_tx_id) VALUES ('bad-extra-party',?,'adjustment','proposed',?,?,1,'correction',1,?)`, net.ID, account, adminAccount, original.ID)
	if err == nil {
		t.Fatal("relation exposed to an unrelated adjustment party")
	}
	// A valid reference remains insertable; the guards cannot simply reject every link.
	_, err = e.Store.DB().Exec(`INSERT INTO pending_adjustments(id,network_id,proposer_member_id,target_member_id,amount,direction,reason,created_at,expires_at,related_tx_id) VALUES ('valid-pending',?,?,?,1,'credit','correction',1,2,?)`, net.ID, am.ID, ma.ID, original.ID)
	if err != nil {
		t.Fatalf("valid pending relation: %v", err)
	}
	_, err = e.Store.DB().Exec(`INSERT OR REPLACE INTO pending_adjustments(id,network_id,proposer_member_id,target_member_id,amount,direction,reason,created_at,expires_at,related_tx_id) VALUES ('valid-pending',?,?,?,1,'credit','rewritten',1,2,NULL)`, net.ID, am.ID, ma.ID)
	if err == nil {
		t.Fatal("REPLACE rewrote pending relation")
	}
	posted := testutil.Must(e.Svc.CreateAdjustment(e.Ctx, admin, net.ID, service.AdjustInput{MemberID: ma.ID, Amount: 1, Direction: service.AdjustCredit, Reason: "linked", RelatedTxID: original.ID, IdempotencyKey: e.Key()}))
	_, err = e.Store.DB().Exec(`INSERT OR REPLACE INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,reason,proposed_at,related_tx_id) VALUES (?,?,'adjustment','proposed',?,?,1,'rewritten',1,NULL)`, posted.ID, net.ID, sys, account)
	if err == nil {
		t.Fatal("REPLACE rewrote posted relation")
	}
	for _, row := range []struct{ table, id string }{{"transactions", posted.ID}, {"pending_adjustments", "valid-pending"}} {
		var related string
		testutil.Must(0, e.Store.DB().QueryRow(`SELECT related_tx_id FROM `+row.table+` WHERE id=?`, row.id).Scan(&related))
		if related != original.ID {
			t.Fatalf("%s relation changed after REPLACE", row.table)
		}
	}
}

func TestAdjustmentRelationMigrationLeavesExistingLedgerUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	ctx := context.Background()
	legacy := testutil.Must(store.OpenAtVersion(ctx, path, 17))
	seedV1Ledger(t, legacy.DB())
	testutil.Must(0, legacy.Close())
	current := testutil.Must(store.Open(ctx, path))
	defer func() { _ = current.Close() }()
	var id, status string
	var amount, seq, total int64
	var related, original *string
	err := current.DB().QueryRow(`SELECT id,status,amount,ledger_seq,reversed_total,original_tx_id,related_tx_id FROM transactions`).Scan(&id, &status, &amount, &seq, &total, &original, &related)
	if err != nil || id != "t1" || status != "completed" || amount != 2500 || seq != 1 || total != 0 || original != nil || related != nil {
		t.Fatalf("migration changed historical transaction: %s %s %d %d %d %v %v err=%v", id, status, amount, seq, total, original, related, err)
	}
	var count int
	testutil.Must(0, current.DB().QueryRow(`SELECT count(*) FROM entries`).Scan(&count))
	if count != 2 {
		t.Fatal("migration changed ledger")
	}
	var integrity string
	testutil.Must(0, current.DB().QueryRow(`PRAGMA integrity_check`).Scan(&integrity))
	if integrity != "ok" {
		t.Fatal(integrity)
	}
}
