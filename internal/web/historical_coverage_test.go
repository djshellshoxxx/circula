package web

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
)

func TestHistoricalLimitCoverageCheckPage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ok       bool
		coverage *service.HistoricalLimitCoverage
		wants    []string
	}{
		{"old unknown", true, nil, []string{"Check passed: no problems found.", "Historical credit-limit coverage was not recorded for this check."}},
		{"partial passes", true, &service.HistoricalLimitCoverage{Status: "partial", CheckedDebits: 1, SystemDebits: 0, LegacyUnverifiedDebits: 1000, UnverifiableDebits: 2},
			[]string{"Check passed: no problems found.", "Coverage is partial:", "1 debit was checked against its historical credit limit.",
				"0 system debits were exempt from credit limits.", "1,000 debits predate evidence collection", "2 debits have missing or unusable evidence"}},
		{"complete with finding", false, &service.HistoricalLimitCoverage{Status: "complete", CheckedDebits: 2, SystemDebits: 1},
			[]string{"Check failed: problems found.", "Coverage is complete.", "The check result separately reports any problems found.",
				"2 debits were checked against their historical credit limits.", "1 system debit was exempt from credit limits."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := renderAuditExportPage(t, "admin_check", &page{Slug: "town", D: &service.LedgerReport{OK: tc.ok, HistoricalLimits: tc.coverage}})
			wantAuditExportText(t, body, append([]string{"Historical credit-limit coverage"}, tc.wants...)...)
			if tc.coverage == nil && (strings.Contains(body, "Coverage is complete.") || strings.Contains(body, "0 debits")) {
				t.Fatal("old unknown coverage was presented as fully checked")
			}
		})
	}
}

// Retained checks must distinguish no run, an old run without metadata, and
// a new witnessed run across the repository, service overview and HTML view.
func TestHistoricalLimitCoverageRetainedOverview(t *testing.T) {
	w := newWorld(t)
	admin := w.browser(t)
	admin.login("admin@example.org")
	path := w.base() + "/admin"
	first := admin.get(path)
	admin.mustContain(first, "No ledger check has been recorded yet")
	admin.mustNotContain(first, "Historical credit-limit coverage", "Coverage is complete.")
	when := w.env.Clock.Now()
	if err := w.env.Store.Update(w.env.Ctx, func(tx store.Tx) error {
		return tx.InsertConsistencyRun(store.ConsistencyRun{ID: w.env.Key(), NetworkID: w.net.ID, Started: when, Finished: when, OK: true})
	}); err != nil {
		t.Fatal(err)
	}
	old := admin.get(path)
	admin.mustContain(old, "The last ledger check passed.", "Historical credit-limit coverage was not recorded for this check.")
	admin.mustNotContain(old, "Coverage is complete.", "Coverage is partial:")
	w.env.Clock.Advance(time.Second)
	w.env.Trade(w.net.ID, w.alice, w.bob, w.aliceM, w.bobM, 100)
	checked := admin.post(w.base()+"/admin/check", old, nil)
	if checked.Status != 200 {
		t.Fatalf("check status = %d: %s", checked.Status, checked.Body)
	}
	admin.mustContain(checked, "Check passed: no problems found.", "Coverage is complete.", "1 debit was checked against its historical credit limit.")
	current := admin.get(path)
	admin.mustContain(current, "The last ledger check passed.", "Coverage is complete.", "1 debit was checked against its historical credit limit.")
	admin.mustNotContain(current, "Historical credit-limit coverage was not recorded")
}
