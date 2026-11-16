package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// BreakGlassRepo covers operator break-glass sessions and the temporary role
// grants tied to them (SEC-013).
type BreakGlassRepo interface {
	InsertBreakGlass(b *domain.BreakGlassSession) error
	// GetBreakGlass finds a session by id in any network (operator CLI only).
	GetBreakGlass(id string) (*domain.BreakGlassSession, error)
	// CloseBreakGlass closes an open session; false when it was already closed.
	CloseBreakGlass(id string, at time.Time, reason string) (bool, error)
	// ListBreakGlass lists a network's sessions, newest first; afterID is a keyset cursor.
	ListBreakGlass(netID, afterID string, limit int) ([]*domain.BreakGlassSession, error)
	// LiveBreakGlass lists the network's open, unexpired sessions, newest first.
	LiveBreakGlass(netID string, now time.Time) ([]*domain.BreakGlassSession, error)
	// ExpiredOpenBreakGlass lists open sessions (any network) whose expiry is at or before now.
	ExpiredOpenBreakGlass(now time.Time, limit int) ([]*domain.BreakGlassSession, error)
	InsertBreakGlassGrant(g *domain.BreakGlassGrant) error
	// LiveBreakGlassRoles lists the roles a member holds through live sessions.
	LiveBreakGlassRoles(netID, memberID string, now time.Time) ([]domain.Role, error)
}

// ErasureRepo covers SEC-018 anonymisation of a departed member's personal data.
type ErasureRepo interface {
	SetMemberDisplayName(netID, memberID, name string) error
	// AnonymiseListings archives the member's listings and replaces their text.
	AnonymiseListings(netID, memberID, title string, at time.Time) (int64, error)
	// RedactTransactionNotes clears the free-text memo of transactions the member proposed.
	RedactTransactionNotes(netID, memberID string) (int64, error)
	// RedactDisputeStatements clears the statements of disputes the member opened.
	RedactDisputeStatements(netID, memberID string) (int64, error)
	// AnonymisePerson replaces the login and sets an unusable password hash.
	AnonymisePerson(personID, login, unusableHash string) error
	// RedactTokensOf revokes the person's API tokens and clears their names.
	RedactTokensOf(personID string, at time.Time) error
	InsertErasure(e *domain.ErasureRecord) error
	// GetErasure returns nil, nil when the member was never erased.
	GetErasure(netID, memberID string) (*domain.ErasureRecord, error)
}
