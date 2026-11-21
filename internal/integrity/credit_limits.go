package integrity

import (
	"fmt"
	"math/big"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
)

func checkHistoricalLimits(tx repository.VerifyRepo, netID string, rep *Report) error {
	coverage := &repository.HistoricalLimitCoverage{Status: "complete"}
	var accountID string
	var balance, amount, next, minimum big.Int
	err := tx.CreditLimitHistory(netID, func(r repository.CreditLimitHistoryRow) error {
		entry := r.Entry
		if entry.AccountID != accountID {
			accountID = entry.AccountID
			balance.SetInt64(0)
		}
		next.Add(&balance, amount.SetInt64(entry.Amount))
		defer balance.Set(&next)
		add := func(message string) {
			rep.Findings = append(rep.Findings, fmt.Sprintf("entry %s historical credit check: %s", entry.ID, message))
		}
		if entry.Amount >= 0 {
			if r.Problem != "" {
				add(r.Problem)
			}
			if r.Check != nil || r.LegacyUnverified {
				add("credit evidence is attached to a non-debit entry")
			}
			return nil // incoming credits improve the position without a limit test
		}
		if r.Problem != "" {
			coverage.UnverifiableDebits++
			add(r.Problem)
			return nil
		}
		if r.LegacyUnverified {
			coverage.LegacyUnverifiedDebits++
			return nil
		}
		w := r.Check
		if w == nil || r.Policy == nil {
			coverage.UnverifiableDebits++
			if w == nil {
				add("forward debit has no immutable witness")
			} else {
				add("witness policy version is missing")
			}
			return nil
		}
		if w.EntryID != entry.ID || w.NetworkID != netID || w.CheckedAt.UnixMilli() != entry.PostedAt.UnixMilli() || w.IsSystem != r.AccountIsSystem {
			coverage.UnverifiableDebits++
			add("witness does not match its entry, posting time or account")
			return nil
		}
		payer := domain.Account{LimitOverride: w.LimitOverride, LimitOverrideExpires: w.LimitOverrideUntil}
		limit := payer.EffectiveLimit(r.Policy.DefaultCreditLimit, w.CheckedAt)
		if limit < 0 || limit != w.EffectiveLimit {
			coverage.UnverifiableDebits++
			add("recorded effective limit does not match the immutable policy and override inputs")
			return nil
		}
		if !balance.IsInt64() || balance.Int64() != w.BalanceBefore {
			add(fmt.Sprintf("recorded prior balance %d does not match entry history %s", w.BalanceBefore, balance.String()))
		}
		if w.IsSystem {
			coverage.SystemDebits++
			return nil
		}
		coverage.CheckedDebits++
		minimum.SetInt64(-limit)
		if next.Cmp(&minimum) < 0 {
			add(fmt.Sprintf("debit leaves account %s at %s below its then-effective limit %d", accountID, next.String(), limit))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if coverage.LegacyUnverifiedDebits != 0 || coverage.UnverifiableDebits != 0 {
		coverage.Status = "partial"
	}
	rep.HistoricalLimits = coverage
	return nil
}
