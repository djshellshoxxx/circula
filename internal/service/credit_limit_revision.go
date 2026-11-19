package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// ErrCreditLimitChanged means a conditional administrator write lost its
// precondition inside the write transaction. Adapters may ask the caller to
// refresh rather than silently overwrite a concurrent administrator's change.
var ErrCreditLimitChanged = errors.New("credit limit changed since it was read")

// creditLimitRevision fingerprints the limit configuration, not just its
// effective amount (A2-07). The active/inherited marker changes at expiry even
// when both values are equal. A default-policy change matters only while the
// account inherits that default; unrelated postings do not invalidate a limit
// edit. This internal token carries no new API response fields.
func creditLimitRevision(netID string, acc *domain.Account, def int64, now time.Time) string {
	parts := []string{"credit-limit-v1", netID, acc.MemberID}
	if acc.LimitOverride == nil {
		parts = append(parts, "no-override")
	} else {
		parts = append(parts, "override", strconv.FormatInt(*acc.LimitOverride, 10))
	}
	if acc.LimitOverrideExpires == nil {
		parts = append(parts, "no-expiry")
	} else {
		parts = append(parts, "expiry", strconv.FormatInt(acc.LimitOverrideExpires.UnixMilli(), 10))
	}
	if acc.LimitOverride != nil && (acc.LimitOverrideExpires == nil || now.Before(*acc.LimitOverrideExpires)) {
		parts = append(parts, "active")
	} else {
		parts = append(parts, "inherited", strconv.FormatInt(def, 10))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
