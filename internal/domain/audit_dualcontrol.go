package domain

// Audit actions for dual control of administrator adjustments (SEC-011).
const (
	ActAdjustmentPending   = "adjustment.pending"   // request recorded, nothing posted
	ActAdjustmentApproved  = "adjustment.approved"  // second administrator approved; posting follows in the same unit
	ActAdjustmentRejected  = "adjustment.rejected"  // second administrator rejected
	ActAdjustmentCancelled = "adjustment.cancelled" // proposer withdrew the request
	ActAdjustmentExpired   = "adjustment.expired"   // not decided in time; nothing posted
)

// CodeApprovalRequired is returned by the legacy single-step adjustment call
// when the amount is above the network's dual-control threshold. The caller
// must use the request/approve flow instead.
const CodeApprovalRequired Code = "approval-required"

// ErrApprovalRequired matches errors.Is on the approval-required code.
var ErrApprovalRequired = &Error{Code: CodeApprovalRequired}
