package domain

import (
	"strings"
	"time"
)

// Administrator recovery when no administrator is available (GOV-021, T-17).
//
// A recorded, active member of a network that has no active administrator
// (GOV-020, Network.AdminRequired) asks to be appointed. The platform operator
// cannot start the process and cannot appoint on its own authority: two
// different operators must approve the member's request, the second approval
// applies it as an ordinary administrator role assignment, and nothing in the
// ledger changes. This is a separate, dual-approved process; it does not use or
// change ordinary break-glass (SEC-013, DEC-014, A3-F02).

// AdminRecoveryStatus is the state of a recovery request.
type AdminRecoveryStatus string

// Recovery request states. requested and approved are open; the rest are final.
const (
	AdminRecoveryRequested  AdminRecoveryStatus = "requested"  // waiting for a first operator approval
	AdminRecoveryApproved   AdminRecoveryStatus = "approved"   // one operator approved; a second, different one must approve
	AdminRecoveryApplied    AdminRecoveryStatus = "applied"    // second approval: the requester holds the admin role
	AdminRecoveryRefused    AdminRecoveryStatus = "refused"    // an operator refused it
	AdminRecoveryExpired    AdminRecoveryStatus = "expired"    // not applied within AdminRecoveryTTL
	AdminRecoverySuperseded AdminRecoveryStatus = "superseded" // another request of the network was applied first
)

// AdminRecoveryTTL bounds how long a request stays open (provisional, 7 days;
// the schema enforces the same maximum). Both approvals must happen within it.
const AdminRecoveryTTL = 7 * 24 * time.Hour

// Closing reasons recorded by the system (expiry job, supersession).
const (
	AdminRecoveryClosedExpired    = "expired"
	AdminRecoveryClosedSuperseded = "another recovery request was applied"
)

// Audit actions of administrator recovery (GOV-021). Operator steps are
// written at platform scope and at network scope (SEC-013 decision 5).
const (
	ActAdminRecoveryRequested  = "admin_recovery.requested"
	ActAdminRecoveryApproved   = "admin_recovery.approved" // first of the two approvals
	ActAdminRecoveryApplied    = "admin_recovery.applied"  // second approval, the appointment
	ActAdminRecoveryRefused    = "admin_recovery.refused"
	ActAdminRecoveryExpired    = "admin_recovery.expired"
	ActAdminRecoverySuperseded = "admin_recovery.superseded"
)

// AdminRecoveryRequest is one member's request to be appointed administrator
// of a network that has none.
type AdminRecoveryRequest struct {
	ID              string
	NetworkID       string
	RequesterID     string // member id; the requester is also the nominee
	Status          AdminRecoveryStatus
	RequestedAt     time.Time
	ExpiresAt       time.Time
	FirstOperator   string // label of the operator who gave the first approval
	FirstReason     string
	FirstApprovedAt *time.Time
	ClosedBy        string // operator label (applied, refused); "" when the system closed it
	ClosedReason    string
	ClosedAt        *time.Time
	RoleAssignment  string // role_assignments id created by the appointment (applied only)
}

// Open reports whether the request can still be approved or refused.
func (r *AdminRecoveryRequest) Open() bool {
	return r.Status == AdminRecoveryRequested || r.Status == AdminRecoveryApproved
}

// StatusAt is the status as of t: an open request at or past its expiry reads
// as expired even before the job or an operator call has closed it.
func (r *AdminRecoveryRequest) StatusAt(t time.Time) AdminRecoveryStatus {
	if r.Open() && !t.Before(r.ExpiresAt) {
		return AdminRecoveryExpired
	}
	return r.Status
}

// SameOperator reports whether two operator labels name the same operator:
// equal after trimming, collapsing inner white space and Unicode case folding.
// It is the distinctness rule of the two approvals.
func SameOperator(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}
