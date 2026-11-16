// Package analytics calculates a bounded, read-only gross-turnover snapshot.
package analytics

import (
	"errors"
	"math"
	"time"
)

type Trade struct {
	ID, Network, Payer, Payee string
	Amount                    int64
	PostedAt                  time.Time
	Completed                 bool
}
type Result struct {
	Suppressed                bool
	TradeCount, ActiveTraders int
	GrossTurnover             int64
}

// Aggregate counts each completed transfer once (not both ledger entries).
// This is gross historical volume, NOT net flow or outstanding balance. Reversal
// adjustments must be provided by a later ledger adapter, not silently subtracted.
// k suppression alone does not prevent differencing; only fixed approved periods
// may be exposed by a future authorization adapter.
func Aggregate(network string, from, to time.Time, k int, rows []Trade) (Result, error) {
	if network == "" || !from.Before(to) || k < 5 || len(rows) > 100000 {
		return Result{}, errors.New("invalid aggregate request")
	}
	result := Result{}
	members := map[string]bool{}
	seen := map[string]bool{}
	for _, r := range rows {
		if r.Network != network || !r.Completed || r.PostedAt.Before(from) || !r.PostedAt.Before(to) {
			continue
		}
		if r.ID == "" || r.Payer == "" || r.Payee == "" || r.Payer == r.Payee || r.Amount <= 0 || seen[r.ID] || r.Amount > math.MaxInt64-result.GrossTurnover {
			return Result{}, errors.New("invalid or overflowing snapshot")
		}
		seen[r.ID] = true
		members[r.Payer] = true
		members[r.Payee] = true
		result.TradeCount++
		result.GrossTurnover += r.Amount
	}
	if len(members) < k {
		return Result{Suppressed: true}, nil
	}
	result.ActiveTraders = len(members)
	return result, nil
}
