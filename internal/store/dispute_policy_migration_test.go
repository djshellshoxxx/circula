package store_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Migration 0019 freezes the prior historical-policy interpretation once.
// It changes no posting, balance, outcome, audit row, or exported projection.
func TestDisputePolicyBackfillAndBindingSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v18.db")
	legacy, err := store.OpenAtVersion(ctx, path, 18)
	if err != nil {
		t.Fatal(err)
	}
	seedV1Ledger(t, legacy.DB())
	for _, q := range []string{
		`INSERT INTO network_policies(network_id,version,policy_json,effective_at) VALUES
		 ('n1',2,'{"response_period_days":30,"re_review_window_days":60}',1000),
		 ('n1',3,'{"response_period_days":0,"re_review_window_days":1}',3000),
		 ('n1',4,'{"response_period_days":7,"re_review_window_days":14}',2000)`,
		`INSERT INTO disputes(id,network_id,tx_id,opened_by,category,status,outcome,opened_at,decided_at) VALUES
		 ('old','n1','t1','m1','other','resolved','dismissed',500,600),
		 ('long','n1','t1','m1','other','resolved','dismissed',1500,1600),
		 ('cancelled-notice','n1','t1','m1','other','resolved','dismissed',3500,3600)`,
	} {
		if _, err := legacy.DB().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	before := disputeMigrationProjection(t, legacy)
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		upgraded, err := store.Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		if got := disputeMigrationProjection(t, upgraded); !reflect.DeepEqual(got, before) {
			t.Errorf("migration changed existing exported data: got %+v, want %+v", got, before)
		}
		if err := upgraded.View(ctx, func(tx store.Tx) error {
			for _, c := range []struct {
				id                string
				version, response int
			}{{"old", 1, domain.DefaultPolicy().ResponsePeriodDays}, {"long", 2, 30}, {"cancelled-notice", 4, 7}} {
				d, err := tx.GetDispute("n1", c.id)
				if err != nil {
					return err
				}
				p, err := tx.GetPolicyVersion(d.NetworkID, d.PolicyVersion)
				if err != nil {
					return err
				}
				if d.PolicyVersion != c.version || p.ResponsePeriodDays != c.response {
					t.Errorf("%s: version=%d response=%d, want %d/%d", c.id, d.PolicyVersion, p.ResponsePeriodDays, c.version, c.response)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if pass == 0 {
			// Later data with an earlier effective time must never move a
			// migrated binding, including after another process reopens it.
			if err := upgraded.Update(ctx, func(tx store.Tx) error {
				p := domain.DefaultPolicy()
				p.ResponsePeriodDays = 0
				return tx.AppendPolicy("n1", 5, p, "", "clock rollback", time.UnixMilli(10))
			}); err != nil {
				t.Fatal(err)
			}
		}
		if err := upgraded.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func disputeMigrationProjection(t *testing.T, s *store.SQLite) map[store.ExportTable][][]any {
	t.Helper()
	out := make(map[store.ExportTable][][]any)
	if err := s.View(context.Background(), func(tx store.Tx) error {
		for _, table := range []store.ExportTable{store.TblDisputes, store.TblTransactions, store.TblEntries, store.TblAccounts, store.TblAudit} {
			if err := tx.StreamTable(table, "n1", "", func(cols []string, row []any) error {
				if row == nil {
					values := make([]any, len(cols))
					for i := range cols {
						values[i] = cols[i]
					}
					out[table] = append(out[table], values)
				} else {
					out[table] = append(out[table], append([]any(nil), row...))
				}
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDisputePolicyBackfillStopsWithoutApplicablePolicy(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "unmapped-v18.db")
	legacy, err := store.OpenAtVersion(ctx, path, 18)
	if err != nil {
		t.Fatal(err)
	}
	seedV1Ledger(t, legacy.DB())
	if _, err := legacy.DB().Exec(`INSERT INTO disputes(id,network_id,tx_id,opened_by,category,status,opened_at)
		VALUES ('unmapped','n1','t1','m1','other','open',0)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := store.Open(ctx, path)
	if err == nil {
		_ = upgraded.Close()
		t.Fatal("migration accepted a dispute with no reconstructable policy")
	}
	if !strings.Contains(err.Error(), "dispute_policy_backfill_needs_owner_mapping") {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := store.OpenRaw(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	if got, err := store.AppliedVersion(ctx, raw); err != nil || got != 18 {
		t.Fatalf("failed migration did not roll back: schema %d, err %v", got, err)
	}
	var columns int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('disputes') WHERE name = 'policy_version'`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("failed migration left the new column: count=%d err=%v", columns, err)
	}
}

func TestDisputePolicyBindingGuards(t *testing.T) {
	s := openStore(t)
	net, trade, _, _ := seed(t, s)
	ctx := context.Background()
	at := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := s.Update(ctx, func(tx store.Tx) error {
		return tx.InsertDispute(&domain.Dispute{ID: "bound", NetworkID: net, TxID: trade, OpenedBy: "mp1",
			Category: "other", Status: domain.DisputeOpen, OpenedAt: at, PolicyVersion: 1})
	}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`UPDATE disputes SET policy_version = NULL WHERE id='bound'`,
		`UPDATE disputes SET policy_version = 2 WHERE id='bound'`,
		`UPDATE disputes SET opened_at = opened_at + 1 WHERE id='bound'`,
		`UPDATE disputes SET network_id = 'other' WHERE id='bound'`,
		`INSERT OR REPLACE INTO disputes SELECT * FROM disputes WHERE id='bound'`,
		`INSERT OR REPLACE INTO network_policies SELECT * FROM network_policies WHERE network_id='n1' AND version=1`,
	} {
		if _, err := s.DB().Exec(q); err == nil || !strings.Contains(err.Error(), "immutable") {
			t.Errorf("immutable binding was writable (%s): %v", q, err)
		}
	}
	if _, err := s.DB().Exec(`UPDATE disputes SET status='withdrawn' WHERE id='bound'`); err != nil {
		t.Fatal(err)
	}
	for _, version := range []any{nil, 0, 999} {
		if _, err := s.DB().Exec(`INSERT INTO disputes(id,network_id,tx_id,opened_by,category,status,opened_at,policy_version)
			VALUES ('bad','n1','t1','mp1','other','open',?,?)`, at.UnixMilli(), version); err == nil {
			t.Errorf("accepted missing or unknown policy version %v", version)
		}
	}
	if err := s.Update(ctx, func(tx store.Tx) error {
		return tx.AppendPolicy(net, 2, domain.DefaultPolicy(), "", "future", at.Add(time.Hour))
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`INSERT INTO disputes(id,network_id,tx_id,opened_by,category,status,opened_at,policy_version)
		VALUES ('early','n1','t1','mp1','other','open',?,2)`, at.UnixMilli()); err == nil {
		t.Error("accepted a policy not in force when the dispute opened")
	}
}
