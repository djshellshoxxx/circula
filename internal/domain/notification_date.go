package domain

import "time"

// NotificationDate retains the event instant used in a dated inbox message.
// It is an internal snapshot. The API retains the original UTC text. Older
// notices have no snapshot because their date alone cannot recover an instant.
type NotificationDate struct {
	Event         string    `json:"event"`
	At            time.Time `json:"at"`
	ClosureReason string    `json:"closure_reason,omitempty"`
	ReadOnlyDays  int       `json:"read_only_days,omitempty"`
}

const (
	NoticePolicyEffective  = "policy_effective"
	NoticeClosureScheduled = "closure_scheduled"
	// These two values are also the one-time deadline markers in the store.
	NoticeResponseDue = "response"
	NoticeDecisionDue = "decision"
)
