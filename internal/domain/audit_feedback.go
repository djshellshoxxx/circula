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
