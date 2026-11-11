package domain

import "time"

// Audit actions for the operator break-glass trail (SEC-013) and for erasure (SEC-018).
const (
	ActBreakGlassOpened        = "break_glass.opened"
	ActBreakGlassClosed        = "break_glass.closed"
	ActBreakGlassExpired       = "break_glass.expired"
	ActBreakGlassRoleGranted   = "break_glass.role_granted"
	ActMemberAnonymised        = "member.anonymised"
	BreakGlassMaxTTL           = 4 * time.Hour // hard maximum (DEC-014, ARC-009)
	BreakGlassDefaultTTL       = 4 * time.Hour
	BreakGlassClosedByOperator = "closed by operator"
	BreakGlassClosedExpired    = "expired"
)

// BreakGlassSession is one time-boxed operator access record (SEC-013).
type BreakGlassSession struct {
	ID            string
	NetworkID     string
	OperatorLabel string
	Reason        string
	OpenedAt      time.Time
	ExpiresAt     time.Time
	ClosedAt      *time.Time
	ClosedReason  string
}

// LiveAt reports whether the session grants access at t: not closed and t is
// strictly before the expiry (at the boundary the session is already over).
func (b *BreakGlassSession) LiveAt(t time.Time) bool {
	return b.ClosedAt == nil && t.Before(b.ExpiresAt)
}

// Status is "open", "closed" or "expired" as of t.
func (b *BreakGlassSession) Status(t time.Time) string {
	switch {
	case b.ClosedAt != nil && b.ClosedReason == BreakGlassClosedExpired:
		return "expired"
	case b.ClosedAt != nil:
		return "closed"
	case !t.Before(b.ExpiresAt):
		return "expired"
	}
	return "open"
}

// BreakGlassGrant is a temporary role on a member, in force only while its session is live.
type BreakGlassGrant struct {
	ID        string
	SessionID string
	NetworkID string
	MemberID  string
	Role      Role
	GrantedAt time.Time
}

// ErasureRecord notes that a member's personal data was anonymised. It holds no personal data.
type ErasureRecord struct {
	MemberID     string
	NetworkID    string
	PersonID     string
	ErasedAt     time.Time
	PersonErased bool
}
