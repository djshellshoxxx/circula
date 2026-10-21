package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// PolicyVersionRow is one stored policy version with its notice metadata
// (GOV-002, GOV-003). EffectiveAt may lie in the future (a pending change).
type PolicyVersionRow struct {
	Version      int
	Policy       domain.Policy
	ChangedBy    string
	Reason       string
	AnnouncedAt  time.Time // equals EffectiveAt for versions written without notice
	EffectiveAt  time.Time
	NoticeDays   int
	NoticeReason string
}

// GovRepo covers policy notice periods and terms (GOV-003, GOV-009). Like the
// other optional repositories it is asserted on the Tx by the service.
type GovRepo interface {
	// SetAsOf makes GetNetwork return the newest policy version whose
	// effective time is not after t. The service sets it to its clock at the
	// start of every unit of work.
	SetAsOf(t time.Time)
	// LatestPolicyVersion is the highest stored version, effective or pending.
	LatestPolicyVersion(netID string) (int, error)
	AppendPolicyNotice(netID string, version int, p domain.Policy, by, reason string, announced, effective time.Time, noticeDays int, noticeReason string) error
	// ListPolicyVersions returns every version, oldest first.
	ListPolicyVersions(netID string) ([]PolicyVersionRow, error)

	InsertTerms(netID string, t domain.Terms) error
	// LatestTerms returns the current terms, or nil when none are published.
	LatestTerms(netID string) (*domain.Terms, error)
	GetTerms(netID string, version int) (*domain.Terms, error)
	// RecordTermsAcceptance is idempotent per (member, version).
	RecordTermsAcceptance(netID, memberID string, version int, at time.Time) error
	// AcceptedTermsVersion is the highest version the member accepted, 0 for none.
	AcceptedTermsVersion(netID, memberID string) (int, error)
}

var _ GovRepo = (*txImpl)(nil)
