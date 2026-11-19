package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// signalFixture is a network with two members and signals switched off, so
// the service raises nothing on its own and each test writes its own rows.
func signalFixture(t *testing.T) (*testutil.Env, *domain.Network, *domain.Member, *domain.Member) {
	t.Helper()
	e := testutil.NewEnv(t)
	admin := e.Person("admin")
	net, _ := e.Network(admin, "signals", func(p *domain.Policy) { p.ActivitySignalsEnabled = false })
	_, a := e.Member(admin, net, "alice")
	_, b := e.Member(admin, net, "bob")
	return e, net, a, b
}

func signalRow(netID, id, kind, member, other string, period time.Time) store.ActivitySignal {
	return store.ActivitySignal{ID: id, NetworkID: netID, Kind: kind, MemberID: member, OtherMemberID: other,
		PeriodStart: period, Window: time.Hour, Observed: 21, Threshold: 20, RaisedAt: period.Add(time.Minute)}
}

func insertSignal(t *testing.T, e *testutil.Env, s store.ActivitySignal) bool {
	t.Helper()
	var inserted bool
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) (err error) {
		inserted, err = tx.InsertActivitySignal(s)
		return err
	}); err != nil {
		t.Fatalf("insert %s: %v", s.ID, err)
	}
	return inserted
}

// SEC-026: one signal per network, kind, member (or pair) and period is a
// database fact; a second insert for the same key writes nothing, in the same
// unit or a later one, and other periods, kinds and members are independent.
func TestSEC026SignalStoreOncePerPeriod(t *testing.T) {
	e, net, a, b := signalFixture(t)
	period := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	lo, hi := a.ID, b.ID
	if hi < lo {
		lo, hi = hi, lo
	}
	if !insertSignal(t, e, signalRow(net.ID, "sig-1", domain.SignalVelocity, a.ID, "", period)) {
		t.Fatal("first velocity signal not inserted")
	}
	if insertSignal(t, e, signalRow(net.ID, "sig-2", domain.SignalVelocity, a.ID, "", period)) {
		t.Fatal("second velocity signal for the same period inserted")
	}
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		for _, id := range []string{"sig-3", "sig-4"} {
			ok, err := tx.InsertActivitySignal(signalRow(net.ID, id, domain.SignalReciprocal, lo, hi, period))
			if err != nil {
				return err
			}
			if ok != (id == "sig-3") {
				t.Errorf("reciprocal %s inserted=%v", id, ok)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, s := range []store.ActivitySignal{
		signalRow(net.ID, "sig-5", domain.SignalVelocity, a.ID, "", period.Add(time.Hour)), // next period
		signalRow(net.ID, "sig-6", domain.SignalVelocity, b.ID, "", period),                // other member
	} {
		if !insertSignal(t, e, s) {
			t.Errorf("%s not inserted", s.ID)
		}
	}
	var got []string
	if err := e.Store.View(e.Ctx, func(tx store.Tx) error {
		rows, err := tx.ListActivitySignals(net.ID, store.ActivitySignalFilter{Limit: 50})
		for _, r := range rows {
			got = append(got, r.ID)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"sig-6", "sig-5", "sig-3", "sig-1"}; !equalStrings(got, want) {
		t.Fatalf("stored signals %v, want %v (newest id first)", got, want)
	}
}

// SEC-026: signals are append-only. UPDATE and DELETE are refused, REPLACE
// cannot overwrite a stored signal by id or by key, and the CHECK and
// foreign-key constraints refuse malformed rows.
func TestSEC026SignalRowsAppendOnly(t *testing.T) {
	e, net, a, b := signalFixture(t)
	period := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	insertSignal(t, e, signalRow(net.ID, "sig-1", domain.SignalVelocity, a.ID, "", period))
	db := e.Store.DB()
	for _, q := range []string{
		`UPDATE activity_signals SET observed = 99 WHERE id = 'sig-1'`,
		`DELETE FROM activity_signals WHERE id = 'sig-1'`,
		`INSERT OR REPLACE INTO activity_signals(id,network_id,kind,member_id,period_start,window_ms,observed,threshold,raised_at)
			VALUES ('sig-1','` + net.ID + `','velocity','` + b.ID + `',0,3600000,99,1,1)`,
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("accepted: %s", q)
		}
	}
	// REPLACE on the deduplication key is a no-op, not an overwrite.
	if _, err := db.Exec(`INSERT OR REPLACE INTO activity_signals(id,network_id,kind,member_id,period_start,window_ms,observed,threshold,raised_at)
		VALUES ('sig-other',?,'velocity',?,?,3600000,99,1,?)`, net.ID, a.ID, period.UnixMilli(), period.UnixMilli()+1); err != nil {
		t.Fatalf("key REPLACE: %v", err)
	}
	var id string
	var observed, n int
	testutil.Must(0, db.QueryRow(`SELECT id, observed FROM activity_signals`).Scan(&id, &observed))
	testutil.Must(0, db.QueryRow(`SELECT COUNT(*) FROM activity_signals`).Scan(&n))
	if n != 1 || id != "sig-1" || observed != 21 {
		t.Fatalf("stored signal changed: n=%d id=%s observed=%d", n, id, observed)
	}
	lo, hi := a.ID, b.ID
	if hi < lo {
		lo, hi = hi, lo
	}
	other, _ := e.Network(e.Person("carol"), "elsewhere", nil)
	bad := []store.ActivitySignal{
		func() store.ActivitySignal {
			s := signalRow(net.ID, "b1", domain.SignalVelocity, a.ID, "", period.Add(3*time.Hour))
			s.Observed = 20 // not above the threshold
			return s
		}(),
		signalRow(net.ID, "b2", domain.SignalReciprocal, hi, lo, period),    // pair not ordered
		signalRow(net.ID, "b3", domain.SignalVelocity, a.ID, b.ID, period),  // velocity names one member
		signalRow(net.ID, "b4", domain.SignalReciprocal, a.ID, "", period),  // reciprocal needs two
		signalRow(other.ID, "b5", domain.SignalVelocity, a.ID, "", period),  // member of another network
		signalRow(net.ID, "b6", "bogus", a.ID, "", period.Add(2*time.Hour)), // unknown kind
		func() store.ActivitySignal {
			s := signalRow(net.ID, "b7", domain.SignalVelocity, b.ID, "", period)
			s.RaisedAt = period.Add(2 * time.Hour)
			return s
		}(),
	}
	for _, s := range bad {
		err := e.Store.Update(e.Ctx, func(tx store.Tx) error { _, err := tx.InsertActivitySignal(s); return err })
		if err == nil {
			t.Errorf("malformed signal %s accepted", s.ID)
		}
	}
}

// SEC-026: a savepoint rolls back only its own writes, including the audit
// chain head and the post-commit audit sink, and the unit stays usable.
func TestSEC026SavepointRollsBackOnlyItsWrites(t *testing.T) {
	e, net, a, _ := signalFixture(t)
	var sunk []string
	e.Store.SetAuditSink(func(ev *domain.AuditEvent) { sunk = append(sunk, ev.ID) })
	defer e.Store.SetAuditSink(nil)
	period := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	boom := errors.New("signal failed")
	err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		seq, prev, err := tx.LastAudit(net.ID)
		if err != nil {
			return err
		}
		ev := func(id string) *domain.AuditEvent {
			return &domain.AuditEvent{ID: id, NetworkID: net.ID, Seq: seq + 1, OccurredAt: period, ActorType: domain.ActorSystem,
				Action: "x.y", PrevHash: prev, Hash: "h-" + id}
		}
		got := tx.Savepoint(func() error {
			if _, err := tx.InsertActivitySignal(signalRow(net.ID, "sig-rolled-back", domain.SignalVelocity, a.ID, "", period)); err != nil {
				return err
			}
			if err := tx.InsertAudit(ev("audit-rolled-back")); err != nil {
				return err
			}
			return boom
		})
		if !errors.Is(got, boom) {
			t.Errorf("Savepoint returned %v, want the callback error", got)
		}
		// The rolled-back head no longer blocks the same sequence number.
		if err := tx.InsertAudit(ev("audit-kept")); err != nil {
			return err
		}
		return tx.Savepoint(func() error {
			_, err := tx.InsertActivitySignal(signalRow(net.ID, "sig-kept", domain.SignalVelocity, a.ID, "", period))
			return err
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	rows := testutil.Must(e.Store.DB().Query(`SELECT id FROM activity_signals UNION ALL SELECT id FROM audit_events WHERE id LIKE 'audit-%'`))
	for rows.Next() {
		var id string
		testutil.Must(0, rows.Scan(&id))
		ids = append(ids, id)
	}
	_ = rows.Close()
	sort.Strings(ids)
	if want := []string{"audit-kept", "sig-kept"}; !equalStrings(ids, want) {
		t.Fatalf("committed rows %v, want %v", ids, want)
	}
	if !equalStrings(sunk, []string{"audit-kept"}) {
		t.Fatalf("audit sink got %v, want only the committed event", sunk)
	}
}

// SEC-026: the counts behind the signals. Proposals count by proposer in any
// later status; completed trades count per direction, by completion time.
func TestSEC026SignalCounts(t *testing.T) {
	e := testutil.NewEnv(t)
	admin := e.Person("admin")
	net, _ := e.Network(admin, "counts", func(p *domain.Policy) { p.ActivitySignalsEnabled = false })
	a, am := e.Member(admin, net, "alice")
	b, bm := e.Member(admin, net, "bob")
	start := e.Clock.Now()
	e.Trade(net.ID, a, b, am, bm, 100)
	e.Trade(net.ID, a, b, am, bm, 100)
	e.Clock.Advance(time.Minute)
	e.Trade(net.ID, b, a, bm, am, 100)
	open := testutil.Must(e.Svc.ProposeTransaction(e.Ctx, a, net.ID, service.ProposeInput{CounterpartyID: bm.ID,
		Direction: service.ProposerIsPaid, Amount: 1, IdempotencyKey: e.Key()}))
	testutil.Must(e.Svc.CancelTransaction(e.Ctx, a, net.ID, service.CancelInput{TxID: open.ID, IdempotencyKey: e.Key()}))
	if err := e.Store.View(context.Background(), func(tx store.Tx) error {
		check := func(what string, got, want int) {
			if got != want {
				t.Errorf("%s = %d, want %d", what, got, want)
			}
		}
		n, err := tx.CountProposalsSince(net.ID, am.ID, start.Add(-time.Millisecond), e.Clock.Now())
		if err != nil {
			return err
		}
		check("alice's proposals", n, 3)
		n, err = tx.CountProposalsSince(net.ID, am.ID, start, e.Clock.Now())
		if err != nil {
			return err
		}
		check("alice's proposals strictly after the first instant", n, 1)
		ab, ba, err := tx.CountCompletedTradesBetween(net.ID, am.AccountID, bm.AccountID, start.Add(-time.Millisecond), e.Clock.Now())
		if err != nil {
			return err
		}
		check("alice to bob", ab, 2)
		check("bob to alice", ba, 1)
		ab, ba, err = tx.CountCompletedTradesBetween(net.ID, am.AccountID, bm.AccountID, start, e.Clock.Now())
		if err != nil {
			return err
		}
		check("alice to bob after start", ab, 0)
		check("bob to alice after start", ba, 1)
		// A clock rollback must exclude events later than the triggering
		// instant, while still including events exactly at that instant.
		n, err = tx.CountProposalsSince(net.ID, am.ID, start.Add(-time.Millisecond), start)
		if err != nil {
			return err
		}
		check("proposals through the first instant", n, 2)
		ab, ba, err = tx.CountCompletedTradesBetween(net.ID, am.AccountID, bm.AccountID, start.Add(-time.Millisecond), start)
		if err != nil {
			return err
		}
		check("alice to bob through the first instant", ab, 2)
		check("bob to alice through the first instant", ba, 0)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// SEC-026: migration 0022 is additive; a database at v18 upgrades with its
// existing rows unchanged and starts with no signals.
func TestSEC026SignalMigrationIsAdditive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	ctx := context.Background()
	legacy := testutil.Must(store.OpenAtVersion(ctx, path, 18))
	seedV1Ledger(t, legacy.DB())
	before := rowCounts(t, legacy.DB(), "transactions", "entries", "audit_events", "members", "network_policies")
	testutil.Must(0, legacy.Close())
	st := testutil.Must(store.Open(ctx, path))
	defer func() { _ = st.Close() }()
	if after := rowCounts(t, st.DB(), "transactions", "entries", "audit_events", "members", "network_policies"); !equalCounts(before, after) {
		t.Fatalf("row counts changed by the migration: %v -> %v", before, after)
	}
	var n int
	testutil.Must(0, st.DB().QueryRow(`SELECT COUNT(*) FROM activity_signals`).Scan(&n))
	if n != 0 {
		t.Fatalf("migration created %d signals", n)
	}
	// The legacy policy has no signal fields. Upgrading must not silently
	// activate surveillance under provisional thresholds, or rewrite that
	// historical policy to add the new defaults.
	if err := st.View(ctx, func(tx store.Tx) error {
		net, err := tx.GetNetwork("n1")
		if err != nil {
			return err
		}
		if net.Policy.ActivitySignalsEnabled {
			t.Fatal("upgrading a legacy policy enabled activity signals")
		}
		if net.Policy.VelocitySignalProposals != domain.DefaultVelocitySignalProposals ||
			net.Policy.ReciprocalSignalTrades != domain.DefaultReciprocalSignalTrades {
			t.Fatalf("legacy policy lacks usable opt-in thresholds: %+v", net.Policy)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var policyJSON string
	testutil.Must(0, st.DB().QueryRow(`SELECT policy_json FROM network_policies WHERE network_id = 'n1' AND version = 1`).Scan(&policyJSON))
	if policyJSON != `{"dispute_window_days":30}` {
		t.Fatalf("migration rewrote historical policy: %s", policyJSON)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
