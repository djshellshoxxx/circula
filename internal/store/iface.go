package store

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// Store runs units of work. Update opens a write transaction (BEGIN
// IMMEDIATE, serialised); View opens a consistent read snapshot. fn's
// returned error rolls the whole unit back (LED-4, LED-6).
type Store interface {
	Update(ctx context.Context, fn func(Tx) error) error
	View(ctx context.Context, fn func(Tx) error) error
	Close() error
}

// Tx is the full repository surface available inside a unit of work. The
// service layer is its only caller. Lookups of missing rows return
// domain.ErrNotFound-coded errors; unique violations return ErrConflict.
type Tx interface {
	NetworkRepo
	PersonRepo
	MemberRepo
	AccountRepo
	ListingRepo
	InvitationRepo
	TxnRepo
	DisputeRepo
	AuditRepo
	IdemRepo
	VerifyRepo
	ExportRepo
	BreakGlassRepo
	ErasureRepo
}

// NetworkRepo covers networks, versioned policies and the ledger counter.
type NetworkRepo interface {
	InsertNetwork(n *domain.Network) error
	AppendPolicy(netID string, version int, p domain.Policy, by, reason string, at time.Time) error
	GetNetwork(id string) (*domain.Network, error)
	GetNetworkBySlug(slug string) (*domain.Network, error)
	SetNetworkStatus(id string, s domain.NetworkStatus) error
	// NextLedgerSeq increments and returns the per-network posting counter (DAT-016).
	NextLedgerSeq(netID string) (int64, error)
	LedgerSeq(netID string) (int64, error)
	InsertConsistencyRun(r ConsistencyRun) error
}

// PersonRepo covers identities, login throttling, sessions and API tokens.
type PersonRepo interface {
	InsertPerson(p *domain.Person, passwordHash string) error
	GetPersonByLogin(login string) (*domain.Person, string, error)
	GetPerson(id string) (*domain.Person, string, error)
	SetPasswordHash(id, hash string) error
	GetThrottle(login string) (failures int, lockedUntil time.Time, found bool, err error)
	PutThrottle(login string, failures int, lockedUntil time.Time) error
	DeleteThrottle(login string) error
	// DeleteStaleThrottles purges up to limit rows whose lock ended at or before cutoff (SA-12).
	DeleteStaleThrottles(cutoff time.Time, limit int) (int64, error)

	InsertSession(s *domain.Session) error
	GetSession(idHash string) (*domain.Session, error)
	TouchSession(idHash string, seen time.Time) error
	RevokeSession(idHash string, at time.Time) error
	RevokeSessionsOf(personID string, at time.Time) error
	// DeleteDeadSessions purges up to limit sessions that are expired, idle since
	// idleBefore or revoked at or before revokedBefore (SA-12).
	DeleteDeadSessions(now, idleBefore, revokedBefore time.Time, limit int) (int64, error)

	InsertToken(t *domain.APIToken) error
	GetTokenByHash(hash string) (*domain.APIToken, error)
	TouchToken(id string, at time.Time) error
	RevokeToken(personID, id string, at time.Time) (bool, error)
	ListTokens(personID string) ([]*domain.APIToken, error)
}

// MemberFilter selects members of a network.
type MemberFilter struct {
	Statuses []domain.MemberStatus
	AfterID  string
	Limit    int
}

// MemberRepo covers memberships and role assignments.
type MemberRepo interface {
	InsertMember(m *domain.Member) error
	GetMember(netID, id string) (*domain.Member, error)
	GetMemberByPerson(netID, personID string) (*domain.Member, error)
	ListMembers(netID string, f MemberFilter) ([]*domain.Member, error)
	ListMembershipsOfPerson(personID string) ([]*domain.Member, error)
	SetMemberStatus(netID, id string, s domain.MemberStatus, departedAt *time.Time) error
	SetDefaultFlag(netID, id string, v bool) error
	GrantRole(roleID, netID, memberID string, r domain.Role, by string, at time.Time) error
	RevokeRole(netID, memberID string, r domain.Role, at time.Time) (bool, error)
	// CountActiveAdmins counts members holding admin whose status is active.
	CountActiveAdmins(netID string) (int, error)
}

// AccountRepo covers accounts, cached balances and reservations.
type AccountRepo interface {
	InsertAccount(a *domain.Account) error
	GetAccount(netID, id string) (*domain.Account, error)
	GetAccountByMember(netID, memberID string) (*domain.Account, error)
	GetSystemAccount(netID string) (*domain.Account, error)
	// AddBalance applies delta to the cached balance. Only posting and repair may call it (DAT-019).
	AddBalance(netID, id string, delta int64) error
	SetLimitOverride(netID, id string, limit *int64, until *time.Time) error
	// ReservedOutgoing sums unexpired proposed trades the payer proposed themselves
	// (DOM-008, DOM-012). A request proposed by the payee reserves nothing until
	// the payer consents, which posts immediately (SA-08).
	ReservedOutgoing(netID, accountID string, now time.Time) (int64, error)
	PendingIncoming(netID, accountID string, now time.Time) (int64, error)
}

// ListingFilter selects listings.
type ListingFilter struct {
	Type     domain.ListingType
	Status   domain.ListingStatus
	MemberID string
	Query    string
	AfterID  string
	Limit    int
	// ActiveOwnersOnly hides listings whose owner is not an active member.
	ActiveOwnersOnly bool
}

// ListingRepo covers offers and requests.
type ListingRepo interface {
	InsertListing(l *domain.Listing) error
	GetListing(netID, id string) (*domain.Listing, error)
	UpdateListing(l *domain.Listing) error
	ListListings(netID string, f ListingFilter) ([]*domain.Listing, error)
}

// InvitationRepo covers join invitations (token stored hashed only).
type InvitationRepo interface {
	InsertInvitation(inv *domain.Invitation, tokenHash string) error
	GetInvitationByHash(tokenHash string) (*domain.Invitation, error)
	MarkInvitationUsed(id, memberID string, at time.Time) error
	RevokeInvitation(netID, id string, at time.Time) (bool, error)
	ListInvitations(netID string) ([]*domain.Invitation, error)
}

// TxFilter selects transactions of a network.
type TxFilter struct {
	MemberID     string // either party
	Statuses     []domain.TxStatus
	Kind         domain.TxKind
	OriginalTxID string
	AfterID      string // keyset: ids strictly below (newest first)
	Limit        int
	CreatedBy    string    // proposer member id
	NotCreatedBy string    // proposals made by the other party
	LiveAt       time.Time // when set, hides proposals whose expiry is at or before it
}

// TxnRepo covers transactions, status history and ledger entries.
type TxnRepo interface {
	// InsertTransaction stores t as 'proposed'; the DB refuses any other initial status.
	InsertTransaction(t *domain.Transaction) error
	GetTransaction(netID, id string) (*domain.Transaction, error)
	// UpdateTransaction writes status, accepted_by, note, completed_at, ledger_seq and reversed_total only.
	UpdateTransaction(t *domain.Transaction) error
	InsertStatusEvent(e *domain.StatusEvent) error
	ListStatusEvents(netID, txID string) ([]*domain.StatusEvent, error)
	ListTransactions(netID string, f TxFilter) ([]*domain.Transaction, error)
	// ListExpiredProposals spans all networks (system sweep).
	ListExpiredProposals(now time.Time, limit int) ([]*domain.Transaction, error)
	InsertEntry(e *domain.Entry) error
	ListEntries(netID, txID string) ([]*domain.Entry, error)
	SumEntries(accountID string, uptoSeq int64) (int64, error)
	// ListAccountEntries lists an account's entries newest first by ledger_seq; beforeSeq > 0 is the keyset cursor.
	ListAccountEntries(netID, accountID string, beforeSeq int64, limit int) ([]*domain.Entry, error)
	// LiveProposalStats counts and sums unexpired proposed trades created by createdBy
	// (optionally with one counterparty, optionally only those where the creator is the payee).
	LiveProposalStats(netID, createdBy, counterparty string, requestsOnly bool, now time.Time) (count int, sum int64, err error)
}

// DisputeFilter selects disputes.
type DisputeFilter struct {
	Statuses []domain.DisputeStatus
	TxID     string
	PartyID  string // only disputes on transactions this member is a party to
	AfterID  string
	Limit    int
}

// DisputeEvent is one append-only dispute history row.
type DisputeEvent struct {
	ID, NetworkID, DisputeID, ActorID, Action, Outcome string
	OccurredAt                                         time.Time
}

// DisputeRepo covers disputes and their event log.
type DisputeRepo interface {
	InsertDispute(d *domain.Dispute) error
	GetDispute(netID, id string) (*domain.Dispute, error)
	UpdateDispute(d *domain.Dispute) error
	ListDisputes(netID string, f DisputeFilter) ([]*domain.Dispute, error)
	InsertDisputeEvent(e DisputeEvent) error
	ListDisputeEvents(netID, disputeID string) ([]DisputeEvent, error)
}

// AuditFilter selects audit events.
type AuditFilter struct {
	Action        string
	ActorID       string
	TargetID      string
	TargetTypes   []string
	SubjectMember string // member-visible subset
	From, To      time.Time
	AfterSeq      int64
	Limit         int
}

// AuditRepo covers the append-only audit log.
type AuditRepo interface {
	LastAudit(netID string) (seq int64, hash string, err error)
	InsertAudit(e *domain.AuditEvent) error
	ListAudit(netID string, f AuditFilter) ([]*domain.AuditEvent, error)
	// AuditHead returns the chain head recorded in the same unit as the last append (SA-09).
	AuditHead(netID string) (seq int64, hash string, found bool, err error)
	// CountAuditSince counts events of one action by one actor since a time.
	CountAuditSince(netID, action, actorID string, since time.Time) (int, error)
}

// IdemRecord stores the outcome of an idempotent command (DAT-025..027).
type IdemRecord struct {
	NetworkID, ActorID, Operation, Key string
	Fingerprint, ResultRef             string
	CreatedAt, ExpiresAt               time.Time
}

// IdemRepo covers the idempotency_keys table.
type IdemRepo interface {
	GetIdem(netID, actorID, op, key string) (*IdemRecord, error) // nil, nil when absent
	PutIdem(r IdemRecord) error
	DeleteIdem(netID, actorID, op, key string) error
	// DeleteExpiredIdem purges up to limit records expired at or before now (SA-12).
	DeleteExpiredIdem(now time.Time, limit int) (int64, error)
}

// ConsistencyRun is an append-only record of a ledger verification.
type ConsistencyRun struct {
	ID, NetworkID, Findings string
	Started, Finished       time.Time
	OK                      bool
}

// BalanceMismatch is a cached balance that differs from the entry sum (LED-3).
type BalanceMismatch struct {
	AccountID      string
	Cached, Summed int64
}

// VerifyRepo runs the read-only checks behind VerifyLedger (DAT-020).
type VerifyRepo interface {
	BalanceMismatches(netID string) ([]BalanceMismatch, error)
	// TxProblems lists completed transactions without two balanced entries,
	// and non-completed transactions that have entries.
	TxProblems(netID string) ([]string, error)
	// Totals returns the sum of cached balances and the sum of entries.
	Totals(netID string) (balances, entries int64, err error)
	// ReversalMismatches lists originals whose reversed_total differs from their reversals.
	ReversalMismatches(netID string) ([]string, error)
}

// ExportTable names a table that can be exported.
type ExportTable string

// Exportable tables.
const (
	TblMembers      ExportTable = "members"
	TblAccounts     ExportTable = "accounts"
	TblListings     ExportTable = "listings"
	TblTransactions ExportTable = "transactions"
	TblStatusEvents ExportTable = "transaction_status_events"
	TblEntries      ExportTable = "entries"
	TblDisputes     ExportTable = "disputes"
	TblPolicies     ExportTable = "network_policies"
	TblAudit        ExportTable = "audit_events"
	TblRoles        ExportTable = "role_assignments"
)

// ExportRepo streams rows without loading a table in memory (FR-095).
type ExportRepo interface {
	// StreamTable calls fn once with nil vals (the header) and then per row in a stable order. MemberID, when set,
	// limits to that member's own data (party transactions, own entries...).
	// Credentials and tokens are never reachable through it (SEC-017).
	StreamTable(t ExportTable, netID, memberID string, fn func(cols []string, vals []any) error) error
}
