package main

import (
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// Unknown coverage in any financial network makes the aggregate unknown.
// Platform audit has no credit-limit coverage. Never infer full coverage from
// a passed check, a missing old field or an empty collection of network checks.
func summariseHistoricalLimits(checks []networkCheck) *service.HistoricalLimitCoverage {
	var out *service.HistoricalLimitCoverage
	for _, check := range checks {
		if check.Platform {
			continue
		}
		coverage := check.HistoricalLimits
		if coverage == nil || (coverage.Status != "complete" && coverage.Status != "partial") {
			return nil
		}
		if out == nil {
			out = &service.HistoricalLimitCoverage{Status: "complete"}
		}
		if coverage.Status == "partial" || coverage.LegacyUnverifiedDebits > 0 || coverage.UnverifiableDebits > 0 {
			out.Status = "partial"
		}
		for _, count := range []struct {
			dst   *int64
			value int64
		}{
			{&out.CheckedDebits, coverage.CheckedDebits},
			{&out.SystemDebits, coverage.SystemDebits},
			{&out.LegacyUnverifiedDebits, coverage.LegacyUnverifiedDebits},
			{&out.UnverifiableDebits, coverage.UnverifiableDebits},
		} {
			if count.value < 0 {
				return nil
			}
			n, ok := domain.AddChecked(*count.dst, count.value)
			if !ok {
				return nil
			}
			*count.dst = n
		}
	}
	return out
}

func printHistoricalLimits(w io.Writer, coverage *service.HistoricalLimitCoverage) {
	if coverage == nil {
		fmt.Fprintln(w, "  historical credit-limit coverage: unknown (not recorded)")
		return
	}
	fmt.Fprintf(w, "  historical credit-limit coverage: %s (checked debits: %d; system debits: %d; legacy unverified debits: %d; unverifiable debits: %d)\n",
		coverage.Status, coverage.CheckedDebits, coverage.SystemDebits, coverage.LegacyUnverifiedDebits, coverage.UnverifiableDebits)
}
