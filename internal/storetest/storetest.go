// Package storetest is the repository contract suite (ARC-004, ARC-R2). It
// states, against the store.Store and store.Tx interfaces only, the behaviour
// the service layer relies on from any storage implementation: atomic units of
// work, read-only views, coded not-found and conflict errors, per-network
// scoping (ARC-007), the ledger posting guards (LED-1, LED-2, DAT-007),
// integer balances within their bound, serialised check-then-write units
// (ARC-006) and millisecond UTC timestamps (DAT-017).
//
// The SQLite store runs it in internal/store; a later adapter (for example
// PostgreSQL, DEC-001) is acceptable when Run passes against it. The package
// must not import a driver: depguard enforces that.
package storetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Factory returns a new, empty store at the current schema. It registers its
// own cleanup (t.Cleanup) to close the store.
type Factory func(t *testing.T) store.Store

// Run runs every contract as a subtest; each opens a fresh store.
func Run(t *testing.T, open Factory) {
	t.Helper()
	for _, c := range []struct {
		name string
		fn   func(*testing.T, store.Store)
	}{
		{"UpdateCommits", updateCommits},
		{"UpdateRollsBackOnError", updateRollsBackOnError},
		{"UpdateRollsBackOnPanic", updateRollsBackOnPanic},
		{"UpdateHonoursCancelledContext", updateHonoursCancelledContext},
		{"ViewIsReadOnly", viewIsReadOnly},
		{"MissingRowsAreNotFound", missingRowsAreNotFound},
		{"DuplicatesAreConflicts", duplicatesAreConflicts},
		{"NetworkScoping", networkScoping},
		{"LedgerSequencePerNetwork", ledgerSequencePerNetwork},
		{"PostingGuards", postingGuards},
		{"BalanceBound", balanceBound},
		{"AuditAppendAndHead", auditAppendAndHead},
		{"IdempotencyRecords", idempotencyRecords},
		{"ConcurrentUnitsSerialise", concurrentUnitsSerialise},
		{"TimestampsAreUTCMillis", timestampsAreUTCMillis},
	} {
		t.Run(c.name, func(t *testing.T) { c.fn(t, open(t)) })
	}
}

var (
	ctx = context.Background()
	t0  = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// errAbort is returned by units that must roll back.
	errAbort = errors.New("storetest: abort")
)

// fixture names the rows seed creates in one network.
type fixture struct {
	net, sys, accA, accB, memA, memB, tx string
}

// seed creates, in its own unit, network <p>n with a system account, two
// active members (persons <p>pa, <p>pb) with accounts, and one proposed trade
// of 100 from A to B. p keeps fixtures of several networks apart.
func seed(t *testing.T, s store.Store, p string) fixture {
	t.Helper()
	f := fixture{net: p + "n", sys: p + "sys", accA: p + "acA", accB: p + "acB", memA: p + "mA", memB: p + "mB", tx: p + "t1"}
	must(t, s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertNetwork(&domain.Network{ID: f.net, Name: "Net " + p, Slug: p + "-slug", Status: domain.NetworkActive, CreatedAt: t0}); err != nil {
			return err
		}
		if err := tx.AppendPolicy(f.net, 1, domain.DefaultPolicy(), "", "", t0); err != nil {
			return err
		}
		if err := tx.InsertAccount(&domain.Account{ID: f.sys, NetworkID: f.net, IsSystem: true}); err != nil {
			return err
		}
		for _, x := range []struct{ person, member, acc string }{{p + "pa", f.memA, f.accA}, {p + "pb", f.memB, f.accB}} {
			if err := tx.InsertPerson(&domain.Person{ID: x.person, Login: x.person, CreatedAt: t0}, "hash"); err != nil {
				return err
			}
			if err := tx.InsertMember(&domain.Member{ID: x.member, NetworkID: f.net, PersonID: x.person, DisplayName: x.person,
				Status: domain.MemberActive, JoinMethod: "founder", JoinedAt: t0}); err != nil {
				return err
			}
			if err := tx.InsertAccount(&domain.Account{ID: x.acc, NetworkID: f.net, MemberID: x.member}); err != nil {
				return err
			}
		}
		exp := t0.Add(time.Hour)
		return tx.InsertTransaction(&domain.Transaction{ID: f.tx, NetworkID: f.net, Kind: domain.KindTrade,
			PayerAccountID: f.accA, PayeeAccountID: f.accB, Amount: 100, CreatedBy: f.memA, ProposedAt: t0, ExpiresAt: &exp})
	}))
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, what string, err error, code domain.Code) {
	t.Helper()
	if got := domain.CodeOf(err); got != code {
		t.Errorf("%s: code = %q (err %v), want %q", what, got, err, code)
	}
}

func view(t *testing.T, s store.Store, fn func(store.Tx) error) {
	t.Helper()
	must(t, s.View(ctx, fn))
}

func balance(t *testing.T, s store.Store, netID, accID string) int64 {
	t.Helper()
	var b int64
	view(t, s, func(tx store.Tx) error {
		a, err := tx.GetAccount(netID, accID)
		if err != nil {
			return err
		}
		b = a.Balance
		return nil
	})
	return b
}

func networkExists(t *testing.T, s store.Store, id string) bool {
	t.Helper()
	var found bool
	view(t, s, func(tx store.Tx) error {
		_, err := tx.GetNetwork(id)
		switch domain.CodeOf(err) {
		case "":
			found = true
		case domain.CodeNotFound:
		default:
			return err
		}
		return nil
	})
	return found
}

// writeNetwork inserts a bare network with its first policy.
func writeNetwork(tx store.Tx, id string) error {
	if err := tx.InsertNetwork(&domain.Network{ID: id, Name: id, Slug: id, Status: domain.NetworkActive, CreatedAt: t0}); err != nil {
		return err
	}
	return tx.AppendPolicy(id, 1, domain.DefaultPolicy(), "", "", t0)
}

func updateCommits(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	view(t, s, func(tx store.Tx) error {
		n, err := tx.GetNetworkBySlug("a-slug")
		if err != nil {
			return err
		}
		if n.ID != f.net || n.Status != domain.NetworkActive || n.PolicyVersion != 1 || n.SystemAccountID != f.sys {
			t.Errorf("network read back = %+v", n)
		}
		x, err := tx.GetTransaction(f.net, f.tx)
		if err != nil {
			return err
		}
		if x.Status != domain.TxProposed || x.Amount != 100 || x.PayerMemberID != f.memA || x.PayeeMemberID != f.memB {
			t.Errorf("transaction read back = %+v", x)
		}
		return nil
	})
}

func updateRollsBackOnError(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	err := s.Update(ctx, func(tx store.Tx) error {
		if err := writeNetwork(tx, "late"); err != nil {
			return err
		}
		if err := tx.AddBalance(f.net, f.accA, -5); err != nil {
			return err
		}
		if err := tx.SetNetworkStatus(f.net, domain.NetworkClosed); err != nil {
			return err
		}
		return errAbort
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("Update returned %v, want the unit's own error", err)
	}
	if networkExists(t, s, "late") {
		t.Error("insert of a failed unit is visible")
	}
	if b := balance(t, s, f.net, f.accA); b != 0 {
		t.Errorf("balance after a failed unit = %d, want 0", b)
	}
	view(t, s, func(tx store.Tx) error {
		n, err := tx.GetNetwork(f.net)
		if err == nil && n.Status != domain.NetworkActive {
			t.Errorf("status change of a failed unit is visible: %s", n.Status)
		}
		return err
	})
}

func updateRollsBackOnPanic(t *testing.T, s store.Store) {
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a panic inside Update was swallowed")
			}
		}()
		_ = s.Update(ctx, func(tx store.Tx) error {
			if err := writeNetwork(tx, "boom"); err != nil {
				return err
			}
			panic("storetest: panic inside a unit")
		})
	}()
	if networkExists(t, s, "boom") {
		t.Error("write of a panicking unit is visible")
	}
	// The store stays usable.
	must(t, s.Update(ctx, func(tx store.Tx) error { return writeNetwork(tx, "after") }))
}

func updateHonoursCancelledContext(t *testing.T, s store.Store) {
	c, cancel := context.WithCancel(ctx)
	cancel()
	err := s.Update(c, func(tx store.Tx) error { return writeNetwork(tx, "cancelled") })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Update with a cancelled context = %v, want context.Canceled", err)
	}
	if networkExists(t, s, "cancelled") {
		t.Error("write under a cancelled context is visible")
	}
}

func viewIsReadOnly(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	err := s.View(ctx, func(tx store.Tx) error {
		if err := writeNetwork(tx, "viewed"); err != nil {
			return err
		}
		return tx.AddBalance(f.net, f.accA, 1)
	})
	if err == nil {
		t.Error("a write inside View succeeded")
	}
	if networkExists(t, s, "viewed") {
		t.Error("write through View is visible")
	}
	if b := balance(t, s, f.net, f.accA); b != 0 {
		t.Errorf("balance after a write through View = %d", b)
	}
}

func missingRowsAreNotFound(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	view(t, s, func(tx store.Tx) error {
		_, err := tx.GetNetwork("nope")
		wantCode(t, "GetNetwork", err, domain.CodeNotFound)
		_, err = tx.GetNetworkBySlug("nope")
		wantCode(t, "GetNetworkBySlug", err, domain.CodeNotFound)
		_, _, err = tx.GetPerson("nope")
		wantCode(t, "GetPerson", err, domain.CodeNotFound)
		_, _, err = tx.GetPersonByLogin("nope")
		wantCode(t, "GetPersonByLogin", err, domain.CodeNotFound)
		_, err = tx.GetMember(f.net, "nope")
		wantCode(t, "GetMember", err, domain.CodeNotFound)
		_, err = tx.GetMemberByPerson(f.net, "nope")
		wantCode(t, "GetMemberByPerson", err, domain.CodeNotFound)
		_, err = tx.GetAccount(f.net, "nope")
		wantCode(t, "GetAccount", err, domain.CodeNotFound)
		_, err = tx.GetAccountByMember(f.net, "nope")
		wantCode(t, "GetAccountByMember", err, domain.CodeNotFound)
		_, err = tx.GetTransaction(f.net, "nope")
		wantCode(t, "GetTransaction", err, domain.CodeNotFound)
		_, err = tx.GetListing(f.net, "nope")
		wantCode(t, "GetListing", err, domain.CodeNotFound)
		_, err = tx.GetDispute(f.net, "nope")
		wantCode(t, "GetDispute", err, domain.CodeNotFound)
		_, err = tx.LedgerSeq("nope")
		wantCode(t, "LedgerSeq", err, domain.CodeNotFound)
		return nil
	})
	err := s.Update(ctx, func(tx store.Tx) error { return tx.AddBalance(f.net, "nope", 1) })
	wantCode(t, "AddBalance", err, domain.CodeNotFound)
	err = s.Update(ctx, func(tx store.Tx) error { _, err := tx.NextLedgerSeq("nope"); return err })
	wantCode(t, "NextLedgerSeq", err, domain.CodeNotFound)
}

func duplicatesAreConflicts(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	cases := []struct {
		what string
		fn   func(store.Tx) error
	}{
		{"network id", func(tx store.Tx) error {
			return tx.InsertNetwork(&domain.Network{ID: f.net, Name: "x", Slug: "fresh", Status: domain.NetworkActive, CreatedAt: t0})
		}},
		{"network slug", func(tx store.Tx) error {
			return tx.InsertNetwork(&domain.Network{ID: "fresh", Name: "x", Slug: "a-slug", Status: domain.NetworkActive, CreatedAt: t0})
		}},
		{"person login", func(tx store.Tx) error {
			return tx.InsertPerson(&domain.Person{ID: "fresh", Login: "apa", CreatedAt: t0}, "h")
		}},
		{"membership of a person", func(tx store.Tx) error {
			return tx.InsertMember(&domain.Member{ID: "fresh", NetworkID: f.net, PersonID: "apa", DisplayName: "x",
				Status: domain.MemberActive, JoinMethod: "founder", JoinedAt: t0})
		}},
		{"transaction id", func(tx store.Tx) error {
			return tx.InsertTransaction(&domain.Transaction{ID: f.tx, NetworkID: f.net, Kind: domain.KindTrade,
				PayerAccountID: f.accA, PayeeAccountID: f.accB, Amount: 1, ProposedAt: t0})
		}},
	}
	for _, c := range cases {
		wantCode(t, "duplicate "+c.what, s.Update(ctx, c.fn), domain.CodeConflict)
	}
}

func networkScoping(t *testing.T, s store.Store) {
	a := seed(t, s, "a")
	b := seed(t, s, "b")
	view(t, s, func(tx store.Tx) error {
		_, err := tx.GetMember(b.net, a.memA)
		wantCode(t, "member of A through B", err, domain.CodeNotFound)
		_, err = tx.GetAccount(b.net, a.accA)
		wantCode(t, "account of A through B", err, domain.CodeNotFound)
		_, err = tx.GetTransaction(b.net, a.tx)
		wantCode(t, "transaction of A through B", err, domain.CodeNotFound)
		ms, err := tx.ListMembers(b.net, store.MemberFilter{})
		if err != nil {
			return err
		}
		for _, m := range ms {
			if m.NetworkID != b.net {
				t.Errorf("ListMembers(B) returned %s of %s", m.ID, m.NetworkID)
			}
		}
		if len(ms) != 2 {
			t.Errorf("ListMembers(B) = %d members, want 2", len(ms))
		}
		txs, err := tx.ListTransactions(b.net, store.TxFilter{})
		if err != nil {
			return err
		}
		if len(txs) != 1 || txs[0].ID != b.tx {
			t.Errorf("ListTransactions(B) = %v, want only %s", txs, b.tx)
		}
		return nil
	})
	err := s.Update(ctx, func(tx store.Tx) error { return tx.AddBalance(b.net, a.accA, 1) })
	wantCode(t, "AddBalance of A through B", err, domain.CodeNotFound)
	if got := balance(t, s, a.net, a.accA); got != 0 {
		t.Errorf("balance of A changed through B: %d", got)
	}
	// A transaction may not join accounts of two networks (DOM-001).
	err = s.Update(ctx, func(tx store.Tx) error {
		return tx.InsertTransaction(&domain.Transaction{ID: "cross", NetworkID: a.net, Kind: domain.KindTrade,
			PayerAccountID: a.accA, PayeeAccountID: b.accB, Amount: 1, ProposedAt: t0})
	})
	if err == nil {
		t.Error("cross-network transaction stored")
	}
}

func ledgerSequencePerNetwork(t *testing.T, s store.Store) {
	a := seed(t, s, "a")
	b := seed(t, s, "b")
	next := func(netID string) int64 {
		t.Helper()
		var v int64
		must(t, s.Update(ctx, func(tx store.Tx) error {
			var err error
			v, err = tx.NextLedgerSeq(netID)
			return err
		}))
		return v
	}
	for want := int64(1); want <= 3; want++ {
		if got := next(a.net); got != want {
			t.Fatalf("NextLedgerSeq(A) = %d, want %d", got, want)
		}
	}
	if got := next(b.net); got != 1 {
		t.Errorf("NextLedgerSeq(B) = %d, want 1: counters are per network (DAT-016)", got)
	}
	_ = s.Update(ctx, func(tx store.Tx) error {
		if _, err := tx.NextLedgerSeq(a.net); err != nil {
			return err
		}
		return errAbort
	})
	view(t, s, func(tx store.Tx) error {
		v, err := tx.LedgerSeq(a.net)
		if err == nil && v != 3 {
			t.Errorf("LedgerSeq(A) = %d after a rolled-back increment, want 3", v)
		}
		return err
	})
}

// postingGuards: the ledger rules the store itself refuses to break, whatever
// the caller does (LED-1, DAT-007, DOM-009, DOM-013).
func postingGuards(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	entry := func(id, acc string, amt int64) *domain.Entry {
		return &domain.Entry{ID: id, NetworkID: f.net, TxID: f.tx, AccountID: acc, Amount: amt, LedgerSeq: 1, PostedAt: t0}
	}
	complete := func(tx store.Tx) error {
		x, err := tx.GetTransaction(f.net, f.tx)
		if err != nil {
			return err
		}
		x.Status = domain.TxCompleted
		done := t0
		x.CompletedAt, x.LedgerSeq = &done, 1
		return tx.UpdateTransaction(x)
	}
	if err := s.Update(ctx, complete); err == nil {
		t.Error("a transaction completed without entries")
	}
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.InsertEntry(entry("e0", f.accA, -99)) }); err == nil {
		t.Error("an entry that does not match the transaction amount was stored")
	}
	if err := s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertEntry(entry("e1", f.accA, -100)); err != nil {
			return err
		}
		return complete(tx)
	}); err == nil {
		t.Error("a transaction completed with one entry")
	}
	if err := s.Update(ctx, func(tx store.Tx) error {
		x, err := tx.GetTransaction(f.net, f.tx)
		if err != nil {
			return err
		}
		x.Status = domain.TxReversed
		return tx.UpdateTransaction(x)
	}); err == nil {
		t.Error("illegal transition proposed -> reversed stored")
	}
	// A balanced pair completes it; the terms stay as proposed.
	must(t, s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertEntry(entry("e1", f.accA, -100)); err != nil {
			return err
		}
		if err := tx.InsertEntry(entry("e2", f.accB, 100)); err != nil {
			return err
		}
		return complete(tx)
	}))
	// Rewriting the terms is either refused or ignored; it never sticks.
	_ = s.Update(ctx, func(tx store.Tx) error {
		x, err := tx.GetTransaction(f.net, f.tx)
		if err != nil {
			return err
		}
		x.Amount, x.PayerAccountID = 1, f.accB
		return tx.UpdateTransaction(x)
	})
	view(t, s, func(tx store.Tx) error {
		x, err := tx.GetTransaction(f.net, f.tx)
		if err != nil {
			return err
		}
		if x.Status != domain.TxCompleted || x.Amount != 100 || x.PayerAccountID != f.accA {
			t.Errorf("after posting and a terms rewrite: %+v (terms are immutable, DOM-013)", x)
		}
		es, err := tx.ListEntries(f.net, f.tx)
		if err != nil {
			return err
		}
		var sum int64
		for _, e := range es {
			sum += e.Amount
		}
		if len(es) != 2 || sum != 0 {
			t.Errorf("entries = %d summing %d, want 2 summing 0 (LED-1)", len(es), sum)
		}
		for acc, want := range map[string]int64{f.accA: -100, f.accB: 100} {
			got, err := tx.SumEntries(acc, 0)
			if err != nil {
				return err
			}
			if got != want {
				t.Errorf("SumEntries(%s) = %d, want %d", acc, got, want)
			}
		}
		return nil
	})
	// A transaction is always inserted as proposed, whatever the caller set.
	must(t, s.Update(ctx, func(tx store.Tx) error {
		return tx.InsertTransaction(&domain.Transaction{ID: "t2", NetworkID: f.net, Kind: domain.KindTrade, Status: domain.TxCompleted,
			PayerAccountID: f.accA, PayeeAccountID: f.accB, Amount: 7, CreatedBy: f.memA, ProposedAt: t0})
	}))
	view(t, s, func(tx store.Tx) error {
		x, err := tx.GetTransaction(f.net, "t2")
		if err == nil && x.Status != domain.TxProposed {
			t.Errorf("inserted transaction status = %s, want proposed", x.Status)
		}
		return err
	})
}

// balanceBound: cached balances are integers held within +-2^62 (A3-03), so
// network totals cannot overflow int64.
func balanceBound(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	const bound = int64(1) << 62
	must(t, s.Update(ctx, func(tx store.Tx) error { return tx.AddBalance(f.net, f.accA, bound) }))
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.AddBalance(f.net, f.accA, 1) }); err == nil {
		t.Error("balance above +2^62 stored")
	}
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.AddBalance(f.net, f.accB, -bound-1) }); err == nil {
		t.Error("balance below -2^62 stored")
	}
	if got := balance(t, s, f.net, f.accA); got != bound {
		t.Errorf("balance = %d, want %d", got, bound)
	}
	if got := balance(t, s, f.net, f.accB); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
}

func audit(netID string, seq int64) *domain.AuditEvent {
	return &domain.AuditEvent{ID: fmt.Sprintf("%s-au%d", netID, seq), NetworkID: netID, Seq: seq, OccurredAt: t0,
		ActorType: domain.ActorSystem, Action: "storetest", TargetType: "network", TargetID: netID,
		PrevHash: fmt.Sprintf("h%d", seq-1), Hash: fmt.Sprintf("h%d", seq)}
}

func auditAppendAndHead(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	view(t, s, func(tx store.Tx) error {
		seq, _, err := tx.LastAudit(f.net)
		if err == nil && seq != 0 {
			t.Errorf("LastAudit of an empty chain = %d", seq)
		}
		_, _, found, err2 := tx.AuditHead(f.net)
		if err2 == nil && found {
			t.Error("AuditHead found on an empty chain")
		}
		return errors.Join(err, err2)
	})
	for seq := int64(1); seq <= 2; seq++ {
		must(t, s.Update(ctx, func(tx store.Tx) error { return tx.InsertAudit(audit(f.net, seq)) }))
	}
	dup := audit(f.net, 2)
	dup.ID = "other-id"
	if err := s.Update(ctx, func(tx store.Tx) error { return tx.InsertAudit(dup) }); err == nil {
		t.Error("a second event with an existing sequence number was stored")
	}
	view(t, s, func(tx store.Tx) error {
		seq, hash, err := tx.LastAudit(f.net)
		if err != nil {
			return err
		}
		hs, hh, found, err := tx.AuditHead(f.net)
		if err != nil {
			return err
		}
		if seq != 2 || hash != "h2" || !found || hs != 2 || hh != "h2" {
			t.Errorf("last = (%d,%s), head = (%d,%s,%v), want 2/h2 for both (SA-09)", seq, hash, hs, hh, found)
		}
		evs, err := tx.ListAudit(f.net, store.AuditFilter{})
		if err != nil {
			return err
		}
		if len(evs) != 2 || evs[0].Seq >= evs[1].Seq {
			t.Errorf("ListAudit = %d events (want 2, ascending seq)", len(evs))
		}
		return nil
	})
}

func idempotencyRecords(t *testing.T, s store.Store) {
	a := seed(t, s, "a")
	b := seed(t, s, "b")
	rec := store.IdemRecord{NetworkID: a.net, ActorID: a.memA, Operation: "propose", Key: "k1",
		Fingerprint: "fp", ResultRef: "r1", CreatedAt: t0, ExpiresAt: t0.Add(24 * time.Hour)}
	view(t, s, func(tx store.Tx) error {
		r, err := tx.GetIdem(a.net, a.memA, "propose", "k1")
		if err == nil && r != nil {
			t.Error("GetIdem of an absent key returned a record")
		}
		return err
	})
	must(t, s.Update(ctx, func(tx store.Tx) error { return tx.PutIdem(rec) }))
	again := rec
	again.ResultRef = "r2"
	wantCode(t, "second PutIdem", s.Update(ctx, func(tx store.Tx) error { return tx.PutIdem(again) }), domain.CodeConflict)
	view(t, s, func(tx store.Tx) error {
		r, err := tx.GetIdem(a.net, a.memA, "propose", "k1")
		if err != nil {
			return err
		}
		if r == nil || r.ResultRef != "r1" || r.Fingerprint != "fp" || !r.ExpiresAt.Equal(rec.ExpiresAt) {
			t.Errorf("GetIdem = %+v, want the first record", r)
		}
		other, err := tx.GetIdem(b.net, a.memA, "propose", "k1")
		if err == nil && other != nil {
			t.Error("an idempotency record is visible from another network")
		}
		return err
	})
	must(t, s.Update(ctx, func(tx store.Tx) error { return tx.DeleteIdem(a.net, a.memA, "propose", "k1") }))
	view(t, s, func(tx store.Tx) error {
		r, err := tx.GetIdem(a.net, a.memA, "propose", "k1")
		if err == nil && r != nil {
			t.Error("deleted idempotency record still returned")
		}
		return err
	})
}

// concurrentUnitsSerialise: N concurrent read-check-write units against one
// account whose bound admits only k of them commit exactly k (ARC-006), and
// concurrent counter increments never repeat a value.
func concurrentUnitsSerialise(t *testing.T, s store.Store) {
	f := seed(t, s, "a")
	const n, step, floor = 50, 10, -100 // floor/step = 10 may pass
	var wg sync.WaitGroup
	var mu sync.Mutex
	committed, seqs := 0, map[int64]bool{}
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var seq int64
			err := s.Update(ctx, func(tx store.Tx) error {
				acc, err := tx.GetAccount(f.net, f.accA)
				if err != nil {
					return err
				}
				if acc.Balance-step < floor {
					return errAbort
				}
				if seq, err = tx.NextLedgerSeq(f.net); err != nil {
					return err
				}
				if err := tx.AddBalance(f.net, f.accA, -step); err != nil {
					return err
				}
				return tx.AddBalance(f.net, f.sys, step)
			})
			switch {
			case err == nil:
				mu.Lock()
				committed++
				if seqs[seq] {
					errs <- fmt.Errorf("ledger sequence %d handed out twice", seq)
				}
				seqs[seq] = true
				mu.Unlock()
			case errors.Is(err, errAbort):
			default:
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if want := -floor / step; committed != want {
		t.Errorf("committed %d units, want exactly %d", committed, want)
	}
	if got := balance(t, s, f.net, f.accA); got != floor {
		t.Errorf("balance = %d, want %d", got, floor)
	}
	if got := balance(t, s, f.net, f.sys); got != -floor {
		t.Errorf("system balance = %d, want %d (postings are equal and opposite)", got, -floor)
	}
}

func timestampsAreUTCMillis(t *testing.T, s store.Store) {
	at := time.Date(2026, 3, 29, 1, 30, 0, 123456789, time.FixedZone("CET", 3600))
	must(t, s.Update(ctx, func(tx store.Tx) error {
		if err := tx.InsertNetwork(&domain.Network{ID: "tz", Name: "tz", Slug: "tz", Status: domain.NetworkActive, CreatedAt: at}); err != nil {
			return err
		}
		return tx.AppendPolicy("tz", 1, domain.DefaultPolicy(), "", "", at)
	}))
	view(t, s, func(tx store.Tx) error {
		n, err := tx.GetNetwork("tz")
		if err != nil {
			return err
		}
		want := at.UTC().Truncate(time.Millisecond)
		if !n.CreatedAt.Equal(want) || n.CreatedAt.Location() != time.UTC {
			t.Errorf("CreatedAt = %v, want %v in UTC (DAT-017)", n.CreatedAt, want)
		}
		return nil
	})
}
