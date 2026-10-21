package domain

import "time"

// Clock is the injected time source (DOM section 12).
type Clock interface{ Now() time.Time }

// IDGen is the injected identifier generator (DAT-002).
type IDGen interface{ New() string }

// NetworkStatus is the lifecycle state of a network.
type NetworkStatus string

// Network statuses.
const (
	NetworkSetup  NetworkStatus = "setup"
	NetworkActive NetworkStatus = "active"
	NetworkClosed NetworkStatus = "closed"
)

// Network is a tenant. Policy is the current effective policy version.
type Network struct {
	ID              string
	Name            string
	Slug            string
	Status          NetworkStatus
	Policy          Policy
	PolicyVersion   int
	SystemAccountID string
	LedgerSeq       int64
	CreatedAt       time.Time
	Suspensions     []Suspension // GOV-023, GOV-024; empty when trading is open
}

// SuspensionKind says who suspended a network.
type SuspensionKind string

// Suspension kinds.
const (
	SuspendedByAdmin    SuspensionKind = "admin"
	SuspendedByPlatform SuspensionKind = "platform"
)

// Platform suspension categories (GOV-024). Nothing about members' trades or
// balances is a valid category.
const (
	SuspendLegalOrder       = "legal_order"
	SuspendSecurityIncident = "security_incident"
	SuspendPlatformAbuse    = "platform_abuse"
	SuspendHostingAgreement = "hosting_agreement"
)

// SuspensionCategories lists the allowed platform categories.
var SuspensionCategories = []string{SuspendLegalOrder, SuspendSecurityIncident, SuspendPlatformAbuse, SuspendHostingAgreement}

// Suspension pauses trading in a network; reads and exports stay available.
type Suspension struct {
	Kind     SuspensionKind
	Category string // platform suspensions only
	Reason   string
	By       string // member id (admin) or operator label (platform)
	At       time.Time
}

// Suspended reports whether any suspension is in force.
func (n *Network) Suspended() bool { return len(n.Suspensions) > 0 }

// SuspensionOf returns the suspension of the given kind, if any.
func (n *Network) SuspensionOf(k SuspensionKind) *Suspension {
	for i := range n.Suspensions {
		if n.Suspensions[i].Kind == k {
			return &n.Suspensions[i]
		}
	}
	return nil
}

// MemberStatus is the membership state (DOM-003).
type MemberStatus string

// Membership states.
const (
	MemberPending   MemberStatus = "pending"
	MemberActive    MemberStatus = "active"
	MemberSuspended MemberStatus = "suspended"
	MemberDeparted  MemberStatus = "departed"
)

// Role is a network-scoped permission set beyond plain membership.
type Role string

// Roles.
const (
	RoleAdmin    Role = "admin"
	RoleReviewer Role = "reviewer"
)

// Member is a person's membership in one network.
type Member struct {
	ID          string
	NetworkID   string
	PersonID    string
	DisplayName string
	Status      MemberStatus
	JoinMethod  string // invite | approval | open | founder
	DefaultFlag bool
	AccountID   string
	Roles       []Role
	JoinedAt    time.Time
	DepartedAt  *time.Time
}

// HasRole reports whether m holds r.
func (m *Member) HasRole(r Role) bool {
	for _, x := range m.Roles {
		if x == r {
			return true
		}
	}
	return false
}

// Account is a ledger account. System accounts have no member and no limit.
type Account struct {
	ID                   string
	NetworkID            string
	MemberID             string // empty for the system account
	IsSystem             bool
	Balance              int64
	LimitOverride        *int64
	LimitOverrideExpires *time.Time
}

// EffectiveLimit returns the credit limit magnitude in force at now (DOM-007).
func (a *Account) EffectiveLimit(def int64, now time.Time) int64 {
	if a.LimitOverride != nil && (a.LimitOverrideExpires == nil || now.Before(*a.LimitOverrideExpires)) {
		return *a.LimitOverride
	}
	return def
}

// TxKind is the transaction kind (DOM-009).
type TxKind string

// Transaction kinds.
const (
	KindTrade      TxKind = "trade"
	KindReversal   TxKind = "reversal"
	KindAdjustment TxKind = "adjustment"
	KindGrant      TxKind = "grant"
)

// Transaction pays Amount from the payer account to the payee account.
// A completed transaction has exactly two entries: -Amount on the payer
// and +Amount on the payee.
type Transaction struct {
	ID             string
	NetworkID      string
	Kind           TxKind
	Status         TxStatus
	PayerAccountID string
	PayeeAccountID string
	PayerMemberID  string // empty if system account
	PayeeMemberID  string // empty if system account
	Amount         int64
	ListingID      string
	Note           string
	Reason         string
	OriginalTxID   string
	CreatedBy      string // member id of proposer / admin / actor; empty for system
	AcceptedBy     string
	ReversedTotal  int64
	ProposedAt     time.Time
	ExpiresAt      *time.Time
	CompletedAt    *time.Time
	LedgerSeq      int64
}

// IsParty reports whether member id is the payer or payee.
func (t *Transaction) IsParty(memberID string) bool {
	return memberID != "" && (t.PayerMemberID == memberID || t.PayeeMemberID == memberID)
}

// Entry is an immutable ledger line.
type Entry struct {
	ID        string
	NetworkID string
	TxID      string
	AccountID string
	Amount    int64
	LedgerSeq int64
	PostedAt  time.Time
}

// StatusEvent records one transaction status transition (append-only).
type StatusEvent struct {
	ID         string
	NetworkID  string
	TxID       string
	From       TxStatus
	To         TxStatus
	ActorID    string
	RequestID  string
	OccurredAt time.Time
}

// ListingType is offer or request.
type ListingType string

// Listing types and states.
const (
	ListingOffer   ListingType = "offer"
	ListingRequest ListingType = "request"
)

// ListingStatus is the listing state.
type ListingStatus string

// Listing states.
const (
	ListingActive   ListingStatus = "active"
	ListingPaused   ListingStatus = "paused"
	ListingArchived ListingStatus = "archived"
)

// Listing is an offer or request. It never affects balances (DOM-040).
type Listing struct {
	ID          string
	NetworkID   string
	MemberID    string
	Type        ListingType
	Title       string
	Description string
	Price       *int64
	Status      ListingStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Person is a global identity. Credentials never leave the service.
type Person struct {
	ID        string
	Login     string
	CreatedAt time.Time
}

// Invitation is a single-use expiring join token (stored hashed).
type Invitation struct {
	ID        string
	NetworkID string
	CreatedBy string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    string
	RevokedAt *time.Time
	CreatedAt time.Time
}

// Session is a server-side revocable login session. ID is the SHA-256 hex of
// the bearer token; the token itself is never stored.
type Session struct {
	IDHash     string
	PersonID   string
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

// APIToken authenticates non-browser clients. Only the hash is stored.
type APIToken struct {
	ID         string
	PersonID   string
	NetworkID  string // empty = all networks of the person
	Name       string
	TokenHash  string
	Scopes     []Scope
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
}
