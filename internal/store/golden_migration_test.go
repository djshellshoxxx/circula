package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Golden migration fixture (DAT-047): 10,000 completed transactions across 60
// member accounts plus the system account, with a valid audit hash chain,
// built at schema v1 (the first released schema) from fixed inputs only, so
// two runs produce the same database content. The test migrates it to the
// current schema, verifies the ledger and the audit chain, and logs the
// duration (DAT-048).
const (
	goldenTxCount   = 10000
	goldenMembers   = 60
	goldenBaseMilli = int64(1_700_000_000_000)
)

type goldenStats struct {
	txs, accounts int
	checksum      string // hash over (tx id, payer, payee, amount) in order
}

func buildGoldenV1(t *testing.T, path string) goldenStats {
	t.Helper()
	ctx := context.Background()
	st, err := store.OpenAtVersion(ctx, path, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if v, err := store.AppliedVersion(ctx, st.DB()); err != nil || v != 1 {
		t.Fatalf("fixture schema = %d err=%v, want 1", v, err)
	}
	dbx, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	must := func(q string, a ...any) {
		t.Helper()
		if _, err := dbx.ExecContext(ctx, q, a...); err != nil {
			_ = dbx.Rollback()
			t.Fatalf("%s: %v", q, err)
		}
	}
	pol, _ := json.Marshal(domain.DefaultPolicy())
	must(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('net1','Golden','golden','active',?,?)`, goldenTxCount, goldenBaseMilli)
	must(`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES ('net1',1,?,?)`, string(pol), goldenBaseMilli)
	must(`INSERT INTO accounts(id,network_id,is_system,balance) VALUES ('acc-sys','net1',1,0)`)
	acc := make([]string, goldenMembers)
	mem := make([]string, goldenMembers)
	bal := make([]int64, goldenMembers)
	for i := 0; i < goldenMembers; i++ {
		p, m, a := fmt.Sprintf("p%03d", i), fmt.Sprintf("m%03d", i), fmt.Sprintf("acc%03d", i)
		mem[i], acc[i] = m, a
		must(`INSERT INTO persons(id,login,password_hash,created_at) VALUES (?,?,'x',?)`, p, p+"@golden.test", goldenBaseMilli)
		must(`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES (?, 'net1', ?, ?, 'active','invitation',?)`,
			m, p, "Member "+p, goldenBaseMilli)
		must(`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES (?, 'net1', ?, 0, 0)`, a, m)
	}

	sum := sha256.New()
	prevHash := ""
	for i := 1; i <= goldenTxCount; i++ {
		// fixed pseudo-random pair and amount, no RNG
		pi := (i * 7) % goldenMembers
		qi := (pi + 1 + (i*13)%(goldenMembers-1)) % goldenMembers
		amt := int64(100 + (i*37)%900)
		id := fmt.Sprintf("tx%05d", i)
		at := goldenBaseMilli + int64(i)*1000
		fmt.Fprintf(sum, "%s|%s|%s|%d\n", id, acc[pi], acc[qi], amt)
		must(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,created_by,proposed_at)
			VALUES (?, 'net1','trade','proposed',?,?,?,?,?)`, id, acc[pi], acc[qi], amt, mem[pi], at)
		must(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?, 'net1', ?, ?, ?, ?, ?)`,
			id+"-d", id, acc[pi], -amt, i, at)
		must(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?, 'net1', ?, ?, ?, ?, ?)`,
			id+"-c", id, acc[qi], amt, i, at)
		must(`UPDATE transactions SET status='completed', accepted_by=?, completed_at=?, ledger_seq=? WHERE id=?`, mem[qi], at, i, id)
		bal[pi] -= amt
		bal[qi] += amt

		ev := &domain.AuditEvent{ID: fmt.Sprintf("aud%05d", i), NetworkID: "net1", Seq: int64(i), OccurredAt: time.UnixMilli(at).UTC(),
			ActorType: domain.ActorMember, ActorID: mem[qi], Action: "transaction.completed", TargetType: "transaction", TargetID: id,
			SubjectIDs: []string{mem[pi], mem[qi]}, PrevHash: prevHash}
		ev.Hash = service.AuditHash(ev)
		prevHash = ev.Hash
		must(`INSERT INTO audit_events(id,network_id,seq,occurred_at,actor_type,actor_id,action,target_type,target_id,prev_hash,hash)
			VALUES (?, 'net1', ?, ?, ?, ?, ?, ?, ?, ?, ?)`, ev.ID, ev.Seq, at, string(ev.ActorType), ev.ActorID, ev.Action,
			ev.TargetType, ev.TargetID, ev.PrevHash, ev.Hash)
		for _, m := range ev.SubjectIDs {
			must(`INSERT INTO audit_subjects(audit_id,network_id,member_id) VALUES (?, 'net1', ?)`, ev.ID, m)
		}
	}
	for i := range acc {
		must(`UPDATE accounts SET balance=? WHERE id=?`, bal[i], acc[i])
	}
	if err := dbx.Commit(); err != nil {
		t.Fatal(err)
	}
	return goldenStats{txs: goldenTxCount, accounts: goldenMembers + 1, checksum: hex.EncodeToString(sum.Sum(nil))}
}

func TestGoldenMigrationV1With10000Transactions(t *testing.T) {
	if testing.Short() {
		t.Skip("10,000-transaction fixture skipped with -short")
	}
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "golden.db")

	genStart := time.Now()
	want := buildGoldenV1(t, path)
	genTime := time.Since(genStart)

	// Determinism: a second build has identical content.
	other := filepath.Join(t.TempDir(), "golden2.db")
	if again := buildGoldenV1(t, other); again.checksum != want.checksum {
		t.Fatalf("fixture is not deterministic: %s vs %s", want.checksum, again.checksum)
	}

	before, err := store.InspectSchema(ctx, path)
	if err != nil || before.Applied != 1 {
		t.Fatalf("before: %+v err=%v", before, err)
	}

	start := time.Now()
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatalf("migrate v1 to v%d: %v", store.SchemaVersion(), err)
	}
	migrateTime := time.Since(start)
	defer func() { _ = st.Close() }()

	if v, _ := store.AppliedVersion(ctx, st.DB()); v != store.SchemaVersion() {
		t.Fatalf("applied = %d, want %d", v, store.SchemaVersion())
	}
	svc, err := service.New(st, service.Config{})
	if err != nil {
		t.Fatal(err)
	}
	vstart := time.Now()
	rep, err := svc.VerifyLedger(ctx, domain.SystemActor("golden"), "net1")
	if err != nil {
		t.Fatal(err)
	}
	verifyTime := time.Since(vstart)
	if !rep.OK || rep.LedgerSeq != goldenTxCount || rep.AuditEvents != goldenTxCount {
		t.Fatalf("verification after migration: %+v", rep)
	}
	var txs, accs int
	if err := st.DB().QueryRow(`SELECT (SELECT COUNT(*) FROM transactions), (SELECT COUNT(*) FROM accounts)`).Scan(&txs, &accs); err != nil {
		t.Fatal(err)
	}
	if txs != want.txs || accs != want.accounts || accs < 50 {
		t.Fatalf("rows after migration: %d tx, %d accounts", txs, accs)
	}

	line := fmt.Sprintf("DAT-048 migration v1->v%d: %d transactions, %d accounts: migrate %s, verify %s (fixture build %s)",
		store.SchemaVersion(), txs, accs, migrateTime.Round(time.Millisecond), verifyTime.Round(time.Millisecond), genTime.Round(time.Millisecond))
	t.Log(line)
	if p := os.Getenv("CIRCULA_MIGRATION_TIMING_FILE"); p != "" {
		f, err := os.OpenFile(filepath.Clean(p), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
			_ = f.Close()
		}
	}
}

// Negative: a fixture whose ledger is damaged is caught after migration.
func TestGoldenMigrationDetectsDamagedFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("fixture skipped with -short")
	}
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "damaged.db")
	buildGoldenV1(t, path)
	raw, err := store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE accounts SET balance = balance + 1 WHERE id = 'acc001'`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	svc, _ := service.New(st, service.Config{})
	rep, err := svc.VerifyLedger(ctx, domain.SystemActor("golden"), "net1")
	if err != nil || rep.OK {
		t.Fatalf("damage not detected: %+v err=%v", rep, err)
	}
}
