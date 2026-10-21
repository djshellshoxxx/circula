package domain

import "time"

// ActorType classifies who caused an audit event (DAT 7.1).
type ActorType string

// Actor types.
const (
	ActorMember   ActorType = "member"
	ActorAdmin    ActorType = "network_administrator"
	ActorReviewer ActorType = "dispute_reviewer"
	ActorOperator ActorType = "platform_operator"
	ActorSystem   ActorType = "system"
	ActorPerson   ActorType = "person" // pre-membership actions (register, login)
)

// Audit action identifiers (stable verbs).
const (
	ActNetworkCreated     = "network.created"
	ActNetworkActivated   = "network.activated"
	ActNetworkClosed      = "network.closed"
	ActPolicyChanged      = "policy.changed"
	ActMemberJoined       = "member.joined"
	ActMemberRequested    = "member.requested"
	ActMemberApproved     = "member.approved"
	ActMemberDenied       = "member.denied"
	ActMemberSuspended    = "member.suspended"
	ActMemberRestored     = "member.restored"
	ActMemberDeparted     = "member.departed"
	ActRoleGranted        = "role.granted"
	ActRoleRevoked        = "role.revoked"
	ActLimitChanged       = "account.limit_changed"
	ActDefaultFlagSet     = "member.default_flag_set"
	ActDefaultFlagCleared = "member.default_flag_cleared"
	ActAccountViewed      = "account.viewed"
	ActInviteCreated      = "invitation.created"
	ActInviteRevoked      = "invitation.revoked"
	ActInviteRedeemed     = "invitation.redeemed"
	ActListingCreated     = "listing.created"
	ActListingUpdated     = "listing.updated"
	ActTxProposed         = "transaction.proposed"
	ActTxCompleted        = "transaction.completed"
	ActTxDeclined         = "transaction.declined"
	ActTxCancelled        = "transaction.cancelled"
	ActTxExpired          = "transaction.expired"
	ActTxReversed         = "transaction.reversed"
	ActAdjustment         = "adjustment.posted"
	ActGrant              = "grant.posted"
	ActDisputeOpened      = "dispute.opened"
	ActDisputeReview      = "dispute.under_review"
	ActDisputeWithdrawn   = "dispute.withdrawn"
	ActDisputeResolved    = "dispute.resolved"
	ActDisputeBlocked     = "dispute.posting_pending"
	ActAuditRead          = "audit.read"
	ActExport             = "export.created"
	ActVerify             = "ledger.verified"
	ActPersonRegistered   = "person.registered"
	ActRegisterDuplicate  = "person.register_duplicate" // sign-up with a login that already exists (SA-10)
	ActPasswordChanged    = "person.password_changed"
	ActTokenCreated       = "token.created"
	ActTokenRevoked       = "token.revoked"
)

// AuditEvent is an append-only record of a state change (LED-6). Summaries
// are minimal JSON documents and never contain credentials or free text.
type AuditEvent struct {
	ID            string
	NetworkID     string // "" for platform-level events
	Seq           int64
	OccurredAt    time.Time
	ActorType     ActorType
	ActorID       string
	Action        string
	TargetType    string
	TargetID      string
	RequestID     string
	IdemRef       string
	SummaryBefore string
	SummaryAfter  string
	Reason        string
	SubjectIDs    []string // member ids whom the event concerns (member-visible audit)
	PrevHash      string
	Hash          string
}

// Scope is an API-token or session capability.
type Scope string

// Scopes.
const (
	ScopeRead  Scope = "read"
	ScopeWrite Scope = "write"
)

// Actor is the authenticated principal making a service call. The service
// resolves the actor's membership and roles per network on every call; the
// caller never supplies a role.
type Actor struct {
	PersonID     string
	Scopes       []Scope // empty means none; sessions carry read+write
	NetworkScope string  // when non-empty, restricts a token to one network
	RequestID    string
	System       bool // background jobs and CLI operations
	// MustChangePassword is set by Authenticate/AuthenticateToken while the
	// person's password was reset by someone else (SEC-008). Can(ScopeWrite)
	// is then false, so every write refuses until ChangePassword succeeds.
	MustChangePassword bool
}

// SystemActor is used by background jobs (expiry sweep) and operator CLI.
func SystemActor(requestID string) Actor {
	return Actor{System: true, RequestID: requestID, Scopes: []Scope{ScopeRead, ScopeWrite}}
}

// Can reports whether the actor holds scope s.
func (a Actor) Can(s Scope) bool {
	if a.System {
		return true
	}
	if a.MustChangePassword && s == ScopeWrite {
		return false
	}
	for _, x := range a.Scopes {
		if x == s {
			return true
		}
	}
	return false
}
