package main

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DOM-031: the hourly governance job reports, once, a scheduled decimal-scale
// change stored by an earlier build and tells the operator in the log. It
// changes nothing: the version still applies on its date with its stored
// scale (neutralising it is an owner or operator decision).
func TestClosuresJobReportsStoredScaleChange(t *testing.T) {
	f := newJobFixture(t)
	var pol domain.Policy
	if err := f.st.Update(f.ctx, func(tx store.Tx) error {
		n, err := tx.GetNetwork(f.netID)
		if err != nil {
			return err
		}
		pol = n.Policy
		pol.Decimals++
		now := f.clk.Now()
		ms, err := tx.ListMembers(f.netID, store.MemberFilter{Limit: 1})
		if err != nil || len(ms) == 0 {
			return err
		}
		return tx.(store.GovRepo).AppendPolicyNotice(f.netID, 2, pol, ms[0].ID, "finer units", now, now.Add(time.Hour), 1, "")
	}); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		var n int
		if err := f.st.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = ?`,
			f.netID, domain.ActPolicyScaleChangeDetected).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := f.runner.closures(f.ctx); err != nil || count() != 1 {
		t.Fatalf("while pending: err %v, notes %d", err, count())
	}
	if !strings.Contains(f.logs.String(), "decimal-scale change detected") {
		t.Errorf("no operator log line:\n%s", f.logs.String())
	}
	f.clk.Advance(2 * time.Hour)
	for range 2 {
		if err := f.runner.closures(f.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := count(); n != 1 {
		t.Fatalf("notes = %d, want 1", n)
	}
	if err := f.st.View(f.ctx, func(tx store.Tx) error {
		tx.(store.GovRepo).SetAsOf(f.clk.Now())
		n, err := tx.GetNetwork(f.netID)
		if err == nil && (n.PolicyVersion != 2 || n.Policy.Decimals != pol.Decimals) {
			t.Errorf("the job changed a read: v%d decimals %d, want v2 decimals %d", n.PolicyVersion, n.Policy.Decimals, pol.Decimals)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
