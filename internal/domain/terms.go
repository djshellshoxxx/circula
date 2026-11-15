package domain

import "time"

// Terms is one published version of a network's onboarding rules and terms
// (GOV-009). Versions are append-only; the highest is current.
type Terms struct {
	Version     int
	Body        string
	PublishedBy string // member id of the administrator
	PublishedAt time.Time
}

// CodeTermsRequired: the member has not accepted the network's current terms,
// so they cannot yet propose or accept a trade (GOV-009). Reading and leaving
// stay open.
const CodeTermsRequired Code = "terms-acceptance-required"

// ErrTermsRequired matches CodeTermsRequired with errors.Is.
var ErrTermsRequired = &Error{Code: CodeTermsRequired}

// Audit actions for terms.
const (
	ActTermsPublished = "terms.published"
	ActTermsAccepted  = "terms.accepted"
)

// Notice period constants for policy changes (GOV-003, DEC-025).
const (
	// DefaultNoticeDays is the minimum notice for an adverse policy change.
	DefaultNoticeDays = 14
	// MaxNoticeDays bounds how far ahead a change may be scheduled.
	MaxNoticeDays = 90
)

// Adverse reports whether moving from old to nw is a change members must be
// told about before it takes effect (GOV-003): less default credit, a shorter
// dispute window, a different departure rule, a less restrictive directory or
// a more open join mode.
func Adverse(old, nw Policy) bool {
	return nw.DefaultCreditLimit < old.DefaultCreditLimit ||
		nw.DisputeWindowDays < old.DisputeWindowDays ||
		nw.DepartureRule != old.DepartureRule ||
		directoryRank(nw.DirectoryVisibility) > directoryRank(old.DirectoryVisibility) ||
		joinRank(nw.JoinMode) > joinRank(old.JoinMode)
}

func joinRank(m JoinMode) int {
	switch m {
	case JoinInvite:
		return 0
	case JoinApproval:
		return 1
	}
	return 2
}

// directoryRank orders visibility values by how widely the directory is shown.
func directoryRank(v string) int {
	if v == "" || v == "members-only" {
		return 0
	}
	return 1
}
