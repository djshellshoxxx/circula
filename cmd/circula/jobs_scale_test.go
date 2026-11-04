package main

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// DOM-031: the hourly governance job records, once, that a decimal-scale
// change stored by an earlier build came into force without its scale, and
// tells the operator in the log.
func TestClosuresJobNotesNeutralisedScaleChange(t *testing.T) {
	f := newJobFixture(t)
	var founder string
	if err := f.st.Update(f.ctx, func(tx store.Tx) error {
		n, err := tx.GetNetwork(f.netID)
		if err != nil {
			return err
		}
		pol := n.Policy
		pol.Decimals++
		now := f.clk.Now()
		ms, err := tx.ListMembers(f.netID, store.MemberFilter{Limit: 1})
		if err != nil || len(ms) == 0 {
			return err
		}
		founder = ms[0].ID
		return tx.(store.GovRepo).AppendPolicyNotice(f.netID, 2, pol, founder, "finer units", now, now.Add(time.Hour), 1, "")
	}); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		var n int
		if err := f.st.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE network_id = ? AND action = ?`,
			f.netID, domain.ActPolicyScaleNeutralised).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := f.runner.closures(f.ctx); err != nil || count() != 0 {
		t.Fatalf("before the effective time: err %v, notes %d", err, count())
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
	if !strings.Contains(f.logs.String(), "decimal-scale change") {
		t.Errorf("no operator log line:\n%s", f.logs.String())
	}
}
