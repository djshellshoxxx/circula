package domain

// Audit action identifiers for dispute responses and re-reviews (FR-076,
// FR-078) and abuse reports (FR-024).
const (
	ActDisputeResponded      = "dispute.responded"
	ActDisputeReReviewAsked  = "dispute.rereview_requested"
	ActDisputeReReviewDecide = "dispute.rereview_decided"
	ActReportCreated         = "report.created"
	ActReportHandled         = "report.handled"
)

// ActActivitySignalRaised records a velocity or reciprocal-pattern signal
// (SEC-026). The event has no member subjects, so the members it concerns do
// not see it in their own audit view; its summary carries counts only.
const ActActivitySignalRaised = "activity_signal.raised"
