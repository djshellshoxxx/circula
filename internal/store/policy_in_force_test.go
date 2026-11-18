package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// GOV-001, DEC-012 (AUD4-F03): a policy version stored by an earlier build
// without the dispute timeline fields reads, at any instant it is in force,
// with the built-in defaults, exactly as GetNetwork reads it. The service
// takes a dispute's response period and re-review window from this read.
func TestPolicyInForceAtReadsLegacyVersionsWithDefaults(t *testing.T) {
	s := openStore(t)
	netID, _, _, _ := seed(t, s) // version 1 effective 2026-01-01
	at := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if _, err := s.DB().Exec(`INSERT INTO network_policies(network_id,version,policy_json,changed_by,reason,effective_at)
		VALUES (?,2,'{"currency_name":"hours","decimals":2}','','legacy',?)`, netID, at.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	def := domain.DefaultPolicy()
	if err := s.View(context.Background(), func(tx store.Tx) error {
		p, v, err := tx.PolicyInForceAt(netID, at)
		if err != nil {
			return err
		}
		if v != 2 || p.CurrencyName != "hours" || p.ResponsePeriodDays != def.ResponsePeriodDays || p.ReReviewWindowDays != def.ReReviewWindowDays {
			t.Errorf("legacy version read as v%d %+v, want v2 with the default timelines %d/%d", v, p, def.ResponsePeriodDays, def.ReReviewWindowDays)
		}
		tx.(store.GovRepo).SetAsOf(at)
		n, err := tx.GetNetwork(netID)
		if err != nil {
			return err
		}
		if n.PolicyVersion != v || n.Policy != p {
			t.Errorf("GetNetwork as of the same instant reads v%d %+v, PolicyInForceAt v%d %+v", n.PolicyVersion, n.Policy, v, p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
