package repository

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// HistoricalLimitCoverage describes available evidence, independently of
// invariant success. A nil coverage pointer on a saved run means not recorded.
type HistoricalLimitCoverage struct {
	Status                 string `json:"status"`
	CheckedDebits          int64  `json:"checked_debits"`
	SystemDebits           int64  `json:"system_debits"`
	LegacyUnverifiedDebits int64  `json:"legacy_unverified_debits"`
	UnverifiableDebits     int64  `json:"unverifiable_debits"`
}

// PostingCreditCheck records the actual inputs observed for a new negative
// entry. Legacy identities are created only by migration, never this API.
type PostingCreditCheck struct {
	EntryID, NetworkID string
	CheckedAt          time.Time
	PolicyVersion      int
	BalanceBefore      int64
	IsSystem           bool
	LimitOverride      *int64
	LimitOverrideUntil *time.Time
	EffectiveLimit     int64
}

// CreditLimitHistoryRow is one immutable entry with optional decision
// evidence. All entries are streamed to reconstruct prior account balances.
type CreditLimitHistoryRow struct {
	Entry            domain.Entry
	AccountIsSystem  bool
	LegacyUnverified bool
	Check            *PostingCreditCheck
	Policy           *domain.Policy
	Problem          string
}
