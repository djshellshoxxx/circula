package service

import store "github.com/djshellshoxxx/circula/internal/repository"

// Records the service returns as stored. The adapters name them through these
// aliases so that internal/api and internal/web never import persistence
// packages (ARC-002; depguard rule "adapters" in .golangci.yml). They are value types
// only: reading or changing them goes through the service methods.
type (
	// HistoricalLimitCoverage is evidence coverage, separate from integrity findings.
	HistoricalLimitCoverage = store.HistoricalLimitCoverage
	// Challenge is a member's challenge to an administrator decision (GOV-016).
	Challenge = store.Challenge
	// DisputeResponse is the other party's answer to a dispute (FR-076).
	DisputeResponse = store.DisputeResponse
	// ReReview is the single re-review of a dispute decision (FR-078).
	ReReview = store.ReReview
	// Notification is one inbox entry (FR-038).
	Notification = store.Notification
	// Report is an abuse report about a listing, member or two-member transaction
	// (FR-024, SEC-025).
	Report = store.Report
)

// Re-review statuses (ReReview.Status).
const (
	ReReviewRequested = store.ReReviewRequested
	ReReviewDecided   = store.ReReviewDecided
)
