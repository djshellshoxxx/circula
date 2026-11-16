package store_test

// DOM-017 / DAT-051: migration 0016 backfills each completed trade's dispute
// deadline from the policy version in force at completion, stops for an owner
// mapping when none was, and the new triggers keep the deadline fixed.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/store"
)

const day = int64(86_400_000)

type legacyDB struct {
	t   *testing.T
	st  *store.SQLite
	seq int64
}

func (l *legacyDB) exec(q string, a ...any) {
	l.t.Helper()
	if _, err := l.st.DB().Exec(q, a...); err != nil {
		l.t.Fatalf("%s: %v", q, err)
	}
}

// openLegacy builds a database as schema 15 left it: one network, two members
// and a system account.
func openLegacy(t *testing.T, path string, base int64) *legacyDB {
	t.Helper()
	st, err := store.OpenAtVersion(context.Background(), path, 15)
	if err != nil {
		t.Fatal(err)
	}
	l := &legacyDB{t: t, st: st}
	l.exec(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES ('n','Legacy','legacy','active',0,?)`, base)
	l.exec(`INSERT INTO accounts(id,network_id,is_system,balance) VALUES ('sys','n',1,0)`)
	for _, m := range []string{"a", "b"} {
		l.exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES (?,?,'x',?)`, "p"+m, m+"@legacy.test", base)
		l.exec(`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,joined_at) VALUES (?,'n',?,?,'active','invitation',?)`,
			"m"+m, "p"+m, "Member "+m, base)
		l.exec(`INSERT INTO accounts(id,network_id,member_id,is_system,balance) VALUES (?,'n',?,0,0)`, "acc"+m, "m"+m)
	}
	return l
}

func (l *legacyDB) policy(version int, json string, effective int64) {
	l.t.Helper()
	l.exec(`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES ('n',?,?,?)`, version, json, effective)
}

// post inserts a completed transaction the way the service did before 0016.
func (l *legacyDB) post(id, kind, payer, payee string, completed int64) {
	l.t.Helper()
	l.seq++
	reason, createdBy := "", "ma"
	if kind != "trade" {
		reason, createdBy = "legacy adjustment", ""
	}
	l.exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,reason,created_by,proposed_at)
		VALUES (?,'n',?,'proposed',?,?,100,?,NULLIF(?,''),?)`, id, kind, payer, payee, reason, createdBy, completed-1000)
	l.exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?,'n',?,?,-100,?,?)`, id+"-d", id, payer, l.seq, completed)
	l.exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES (?,'n',?,?,100,?,?)`, id+"-c", id, payee, l.seq, completed)
	l.exec(`UPDATE transactions SET status='completed', completed_at=?, ledger_seq=? WHERE id=?`, completed, l.seq, id)
}

func deadlineOf(t *testing.T, st *store.SQLite, id string) (int64, bool) {
	t.Helper()
	var v *int64
	if err := st.DB().QueryRow(`SELECT dispute_deadline_at FROM transactions WHERE id = ?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		return 0, false
	}
	return *v, true
}

func TestDisputeDeadlineBackfillUsesPolicyInForceAtCompletion(t *testing.T) {
	const base = int64(1_700_000_000_000)
	path := filepath.Join(t.TempDir(), "legacy.db")
	l := openLegacy(t, path, base)
	l.policy(1, `{"currency_name":"credits"}`, base)        // no window stored: the built-in 30 days applied
	l.policy(2, `{"dispute_window_days":60}`, base+10*day)  // in force from day 10
	l.policy(3, `{"dispute_window_days":5}`, base+40*day)   // scheduled for day 40 ...
	l.policy(4, `{"dispute_window_days":60}`, base+20*day)  // ... and cancelled on day 20
	l.post("t-day1", "trade", "acca", "accb", base+1*day)   // v1 -> 30
	l.post("t-day15", "trade", "accb", "acca", base+15*day) // v2 -> 60
	l.post("t-day45", "trade", "acca", "accb", base+45*day) // v4, never the withdrawn v3 -> 60
	l.post("adj", "adjustment", "sys", "acca", base+2*day)  // not a trade: no deadline
	l.exec(`INSERT INTO transactions(id,network_id,kind,status,payer_account_id,payee_account_id,amount,created_by,proposed_at)
		VALUES ('t-open','n','trade','proposed','acca','accb',100,'ma',?)`, base+3*day) // open proposal: no deadline
	if err := l.st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer func() { _ = st.Close() }()
	want := map[string]int64{"t-day1": base + 1*day + 30*day, "t-day15": base + 15*day + 60*day, "t-day45": base + 45*day + 60*day}
	for id, w := range want {
		if got, ok := deadlineOf(t, st, id); !ok || got != w {
			t.Errorf("%s deadline = %d (set %v), want %d", id, got, ok, w)
		}
	}
	for id, w := range map[string]int{"t-day1": 1, "t-day15": 2, "t-day45": 4} {
		var v int
		if err := st.DB().QueryRow(`SELECT dispute_policy_version FROM transactions WHERE id = ?`, id).Scan(&v); err != nil || v != w {
			t.Errorf("%s policy version = %d (err %v), want %d", id, v, err, w)
		}
	}
	for _, id := range []string{"adj", "t-open"} {
		if _, ok := deadlineOf(t, st, id); ok {
			t.Errorf("%s got a dispute deadline", id)
		}
	}

	// The deadline is fixed once set.
	if _, err := st.DB().Exec(`UPDATE transactions SET dispute_deadline_at = dispute_deadline_at + 1 WHERE id = 't-day1'`); err == nil ||
		!strings.Contains(err.Error(), "immutable") {
		t.Errorf("changing a stored deadline: err = %v", err)
	}
	// A trade cannot complete without one.
	if _, err := st.DB().Exec(`INSERT INTO entries(id,network_id,tx_id,account_id,amount,ledger_seq,posted_at) VALUES
		('o-d','n','t-open','acca',-100,9,1),('o-c','n','t-open','accb',100,9,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`UPDATE transactions SET status='completed', completed_at=1, ledger_seq=9 WHERE id='t-open'`); err == nil ||
		!strings.Contains(err.Error(), "dispute deadline") {
		t.Errorf("completing a trade without a deadline: err = %v", err)
	}
}

func TestDisputeDeadlineBackfillStopsWithoutApplicablePolicy(t *testing.T) {
	const base = int64(1_700_000_000_000)
	path := filepath.Join(t.TempDir(), "legacy.db")
	l := openLegacy(t, path, base)
	l.policy(1, `{"dispute_window_days":30}`, base+5*day) // first version came into force after the trade completed
	l.post("t-early", "trade", "acca", "accb", base+1*day)
	if err := l.st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(context.Background(), path)
	if err == nil {
		_ = st.Close()
		t.Fatal("migration served writes with an underivable dispute deadline")
	}
	if !strings.Contains(err.Error(), "dispute_deadline_backfill_needs_owner_mapping") {
		t.Fatalf("unexpected error: %v", err)
	}
	// The failed migration rolled back completely: still schema 15, no new column.
	raw, err := store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if v, err := store.AppliedVersion(context.Background(), raw); err != nil || v != 15 {
		t.Fatalf("schema after failed migration = %d (err %v), want 15", v, err)
	}
	var n int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('transactions') WHERE name = 'dispute_deadline_at'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("dispute_deadline_at column present after rollback (n=%d err=%v)", n, err)
	}
}
