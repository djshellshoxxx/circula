package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// DisputeResponse is the other party's answer to a dispute (FR-076).
type DisputeResponse struct {
	ID, NetworkID, DisputeID, ResponderID, Body string
	SubmittedAt                                 time.Time
}

// ReReview is the single re-review of a dispute decision (FR-078). It is a
// record of its own, linked to the dispute; the original decision is never edited.
type ReReview struct {
	ID, NetworkID, DisputeID, RequestedBy, Reason string
	RequestedAt                                   time.Time
	Status                                        string // requested | decided
	ReviewerID                                    string
	Outcome                                       string // confirmed | upheld | partially_upheld
	ReversalAmount                                int64
	Rationale                                     string
	DecidedAt                                     *time.Time
}

// ReReview statuses.
const (
	ReReviewRequested = "requested"
	ReReviewDecided   = "decided"
)

// Report is an abuse report about a listing or member (FR-024).
type Report struct {
	ID, NetworkID, ReporterID, TargetType, TargetID, Category, Note string
	Status                                                          string // open | handled
	CreatedAt                                                       time.Time
	HandledBy, HandlingNote                                         string
	HandledAt                                                       *time.Time
}

// ReportFilter selects reports; newest first by id.
type ReportFilter struct {
	Status  string // "" = all
	AfterID string
	Limit   int
}

// Notification is one inbox entry (FR-038).
type Notification struct {
	ID, NetworkID, MemberID, Kind, Title, Body, Link, DedupeKey string
	CreatedAt                                                   time.Time
	ReadAt                                                      *time.Time
}

// NotificationFilter selects a member's notifications, newest first by id.
type NotificationFilter struct {
	UnreadOnly bool
	AfterID    string
	Limit      int
}

// FeatureRepo covers responses, re-reviews, reports and notifications.
type FeatureRepo interface {
	InsertDisputeResponse(r DisputeResponse) error
	// GetDisputeResponse returns nil, nil when the dispute has no response.
	GetDisputeResponse(netID, disputeID string) (*DisputeResponse, error)
	// ListDisputesAwaitingResponse lists live (open / under review) disputes on
	// trades the member is party to, opened by the other party, without a response.
	ListDisputesAwaitingResponse(netID, memberID string, limit int) ([]*domain.Dispute, error)

	InsertReReview(r ReReview) error
	// GetReReview returns nil, nil when none was requested.
	GetReReview(netID, disputeID string) (*ReReview, error)
	UpdateReReview(r ReReview) error

	InsertReport(r Report) error
	GetReport(netID, id string) (*Report, error)
	UpdateReport(r Report) error
	ListReports(netID string, f ReportFilter) ([]*Report, error)
	CountReportsSince(netID, reporterID string, since time.Time) (int, error)

	// InsertNotification stores n; it returns false, nil when n.DedupeKey
	// already exists for the member (nothing is written).
	InsertNotification(n Notification) (bool, error)
	ListNotifications(netID, memberID string, f NotificationFilter) ([]*Notification, error)
	CountUnreadNotifications(netID, memberID string) (int, error)
	// MarkNotificationsRead marks the given ids (all when ids is empty) read and returns how many changed.
	MarkNotificationsRead(netID, memberID string, ids []string, at time.Time) (int64, error)
	// DeleteReadNotificationsBefore removes up to limit notifications that were read
	// at or before cutoff (retention). Unread entries are never removed.
	DeleteReadNotificationsBefore(cutoff time.Time, limit int) (int64, error)
	// ListNetworkIDs lists every network id (system jobs).
	ListNetworkIDs() ([]string, error)
}
