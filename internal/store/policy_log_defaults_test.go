package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Older policy JSON omits fields introduced by later builds. The rules log
// must show the same effective values as the network reader, without inventing
// changes from zero or replacing an explicitly stored zero with a default.
func TestPolicyLogUsesDefaultsForMissingHistoricalFields(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "old-policy.db")
	legacy, err := store.OpenAtVersion(ctx, path, 18)
	if err != nil {
		t.Fatal(err)
	}
	seedV1Ledger(t, legacy.DB())
	const secondJSON = `{"dispute_window_days":30,"response_period_days":0}`
	if _, err := legacy.DB().Exec(`INSERT INTO network_policies(network_id,version,policy_json,effective_at)
		VALUES ('n1',2,?,1000)`, secondJSON); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	wantFirst := domain.DefaultPolicy()
	wantSecond := wantFirst
	wantSecond.ResponsePeriodDays = 0
	if err := st.View(ctx, func(tx store.Tx) error {
		net, err := tx.GetNetwork("n1")
		if err != nil {
			return err
		}
		if net.Policy != wantSecond {
			t.Fatalf("network reader = %+v, want %+v", net.Policy, wantSecond)
		}
		g := tx.(store.GovRepo)
		versions, err := g.ListPolicyVersions("n1")
		if err != nil {
			return err
		}
		if len(versions) != 2 || versions[0].Policy != wantFirst || versions[1].Policy != wantSecond {
			t.Errorf("full policy log disagrees with effective defaults: %+v", versions)
		}
		page, err := g.ListPolicyLogPage("n1", 0, 1)
		if err != nil {
			return err
		}
		if len(page) != 1 || page[0].Policy != wantSecond || page[0].Previous == nil || *page[0].Previous != wantFirst {
			t.Errorf("paged policy log or previous values lost defaults: %+v", page)
		}
		first, err := g.ListPolicyLogPage("n1", 2, 1)
		if err != nil {
			return err
		}
		if len(first) != 1 || first[0].Policy != wantFirst || first[0].Previous != nil {
			t.Errorf("first policy page lost defaults or invented a predecessor: %+v", first)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := st.DB().QueryRow(`SELECT policy_json FROM network_policies WHERE network_id = 'n1' AND version = 2`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != secondJSON {
		t.Fatalf("reading defaults rewrote historical policy JSON: %s", stored)
	}
}
