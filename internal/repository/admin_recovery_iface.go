package repository

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// AdminRecoveryRepo covers administrator recovery requests (GOV-021). Rows are
// never deleted; each state change is a compare-and-set on the current status,
// so of any number of concurrent callers exactly one wins a transition.
type AdminRecoveryRepo interface {
	InsertAdminRecovery(r *domain.AdminRecoveryRequest) error
	// GetAdminRecovery finds a request by id in any network; callers check the network.
	GetAdminRecovery(id string) (*domain.AdminRecoveryRequest, error)
	// ListAdminRecovery lists requests newest first (id descending). netID ""
	// spans every network (operator CLI); openOnly keeps requested and approved
	// requests; afterID is a keyset cursor.
	ListAdminRecovery(netID string, openOnly bool, afterID string, limit int) ([]*domain.AdminRecoveryRequest, error)
	// ApproveAdminRecoveryFirst records the first approval of a requested
	// request; false when the request was not in the requested state.
	ApproveAdminRecoveryFirst(id, operator, reason string, at time.Time) (bool, error)
	// CloseAdminRecovery moves a request from status from to the final status
	// to; false when it was not in from. roleAssignmentID is set only for applied.
	CloseAdminRecovery(id string, from, to domain.AdminRecoveryStatus, by, reason string, at time.Time, roleAssignmentID string) (bool, error)
	// GetOpenAdminRecovery returns the member's open (requested or approved)
	// request in the network, or nil when there is none. The schema allows at
	// most one; the lookup uses that partial unique index, never a page scan.
	GetOpenAdminRecovery(netID, memberID string) (*domain.AdminRecoveryRequest, error)
	// ExpiredOpenAdminRecovery lists open requests of any network whose expiry
	// is at or before now, ascending by id strictly after afterID (keyset).
	ExpiredOpenAdminRecovery(now time.Time, afterID string, limit int) ([]*domain.AdminRecoveryRequest, error)
}
