package integrity

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
)

type creditHistoryFixture struct {
	repository.VerifyRepo
	rows []repository.CreditLimitHistoryRow
}

func (f creditHistoryFixture) CreditLimitHistory(_ string, visit func(repository.CreditLimitHistoryRow) error) error {
	for _, row := range f.rows {
		if err := visit(row); err != nil {
			return err
		}
	}
	return nil
}

func TestHistoricalCreditCheckUsesExactExpiryBoundary(t *testing.T) {
	expires := time.UnixMilli(1000)
	override := int64(9000)
	for _, tc := range []struct {
		name      string
		at        time.Time
		effective int64
		finding   bool
	}{
		{"before", expires.Add(-time.Millisecond), 9000, false},
		{"exactly-at", expires, 100, true},
		{"after", expires.Add(time.Millisecond), 100, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := repository.CreditLimitHistoryRow{
				Entry: domain.Entry{ID: "e", NetworkID: "n", AccountID: "a", Amount: -101, LedgerSeq: 1, PostedAt: tc.at},
				Check: &repository.PostingCreditCheck{EntryID: "e", NetworkID: "n", CheckedAt: tc.at, PolicyVersion: 1,
					LimitOverride: &override, LimitOverrideUntil: &expires, EffectiveLimit: tc.effective},
				Policy: &domain.Policy{DefaultCreditLimit: 100},
			}
			report := &Report{}
			if err := checkHistoricalLimits(creditHistoryFixture{rows: []repository.CreditLimitHistoryRow{row}}, "n", report); err != nil {
				t.Fatal(err)
			}
			if found := strings.Contains(strings.Join(report.Findings, "\n"), "below its then-effective limit"); found != tc.finding {
				t.Fatalf("expiry finding = %t, want %t: %v", found, tc.finding, report.Findings)
			}
			if report.HistoricalLimits.Status != "complete" || report.HistoricalLimits.CheckedDebits != 1 || report.HistoricalLimits.UnverifiableDebits != 0 {
				t.Fatalf("a valid witness did not retain complete coverage: %+v", report.HistoricalLimits)
			}
		})
	}
}

func TestHistoricalCreditCheckReconstructsPrefixAndSeparatesCoverage(t *testing.T) {
	at := time.UnixMilli(1000)
	rows := []repository.CreditLimitHistoryRow{
		{Entry: domain.Entry{ID: "legacy", NetworkID: "n", AccountID: "a", Amount: -4000, LedgerSeq: 1}, LegacyUnverified: true},
		// An incoming credit improves the position even when it is still below
		// today's limit; it must not be reinterpreted as a failed debit.
		{Entry: domain.Entry{ID: "credit", NetworkID: "n", AccountID: "a", Amount: 500, LedgerSeq: 2}},
		{Entry: domain.Entry{ID: "forward", NetworkID: "n", AccountID: "a", Amount: -1, LedgerSeq: 3, PostedAt: at},
			Check: &repository.PostingCreditCheck{EntryID: "forward", NetworkID: "n", CheckedAt: at, PolicyVersion: 1,
				BalanceBefore: -3400, EffectiveLimit: 4000}, Policy: &domain.Policy{DefaultCreditLimit: 4000}},
	}
	report := &Report{}
	if err := checkHistoricalLimits(creditHistoryFixture{rows: rows}, "n", report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || !strings.Contains(report.Findings[0], "recorded prior balance -3400 does not match entry history -3500") {
		t.Fatalf("prefix reconstruction or incoming-credit rule changed: %v", report.Findings)
	}
	if coverage := report.HistoricalLimits; coverage.Status != "partial" || coverage.CheckedDebits != 1 || coverage.LegacyUnverifiedDebits != 1 || coverage.UnverifiableDebits != 0 {
		t.Fatalf("integrity findings and historical coverage were conflated: %+v", coverage)
	}
}
