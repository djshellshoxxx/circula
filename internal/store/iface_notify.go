package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases preserve the existing storage API and exact type
// identity. Driver-neutral contracts live in internal/repository (ARC-004).
type (
	DisputeResponse    = repository.DisputeResponse
	ReReview           = repository.ReReview
	Report             = repository.Report
	ReportFilter       = repository.ReportFilter
	Notification       = repository.Notification
	NotificationFilter = repository.NotificationFilter
	FeatureRepo        = repository.FeatureRepo
)

const (
	ReReviewRequested = repository.ReReviewRequested
	ReReviewDecided   = repository.ReReviewDecided
)
