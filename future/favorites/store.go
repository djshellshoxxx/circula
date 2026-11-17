package favorites

import (
	"context"
	"time"
)

// Store is the transactional repository the Service persists through. Every
// Tx method takes the network and owner explicitly and must apply both in the
// storage predicate; there is no unscoped read or write except the
// maintenance methods PurgeMember and PurgeExpiredIntents.
//
// Update runs fn in one serialized write transaction and commits only if fn
// returns nil; any error (including a panic) must roll back every change made
// by fn. View runs fn in a consistent read-only snapshot.
type Store interface {
	Update(ctx context.Context, fn func(Tx) error) error
	View(ctx context.Context, fn func(Tx) error) error
}

// Tx is one store transaction. Writes called from View must fail.
type Tx interface {
	// Revision returns the owner's list revision (0 before any write).
	Revision(ctx context.Context, network, owner string) (int64, error)
	// BumpRevision increments and returns the owner's list revision.
	BumpRevision(ctx context.Context, network, owner string) (int64, error)
	// ListFavorites returns at most limit entries ordered pinned first, then
	// Order, then Target.
	ListFavorites(ctx context.Context, network, owner string, limit int) ([]Entry, error)
	GetFavorite(ctx context.Context, network, owner, target string) (Entry, bool, error)
	// InsertFavorite returns ErrDuplicate if (network, owner, target) exists.
	InsertFavorite(ctx context.Context, e Entry) error
	// UpdateFavorite changes Nickname, Pinned, Order and UpdatedAt of an
	// existing entry and reports whether a row matched.
	UpdateFavorite(ctx context.Context, e Entry) (bool, error)
	DeleteFavorite(ctx context.Context, network, owner, target string) (bool, error)

	GetIntent(ctx context.Context, network, owner, key string) (Intent, bool, error)
	// InsertIntent returns ErrDuplicate if the key (or generated intent) exists.
	InsertIntent(ctx context.Context, in Intent) error
	DeleteIntent(ctx context.Context, network, owner, key string) error
	// CountIntents counts the owner's drafts that expire after now.
	CountIntents(ctx context.Context, network, owner string, now time.Time) (int, error)

	// PurgeMember removes every row owned by or targeting member in network.
	PurgeMember(ctx context.Context, network, member string) (PurgeResult, error)
	// PurgeExpiredIntents removes drafts that expired at or before cutoff.
	PurgeExpiredIntents(ctx context.Context, cutoff time.Time) (int64, error)
}

// Intent is a durable record of one deliberate quick-repeat request, so a
// retry of the same RequestKey returns the same draft and idempotency key.
type Intent struct {
	Network, Owner, Key string
	// TradeID is empty for drafts started from a favorite.
	TradeID, Target, Description, Intent string
	CreatedAt, ExpiresAt                 time.Time
}

// PurgeResult counts rows removed by PurgeMember.
type PurgeResult struct{ Favorites, Intents, States int64 }

// Policy provides fresh authorization facts. Members returns snapshots of ids
// as seen by viewer (viewer.Member is the authenticated owner): Active is
// current network membership, Visible is whether viewer may see the member,
// and Blocked covers suspension and blocks in either direction. Unknown ids
// must be absent from the map. Any error is treated as a refusal. The result
// for viewer.Member itself must be included when requested.
type Policy interface {
	Members(ctx context.Context, viewer Actor, ids []string) (map[string]Member, error)
}

// TradeSource loads an authoritative historical trade for the requesting
// member. It must return ErrTradeNotFound for unknown trades or trades the
// viewer is not party to.
type TradeSource interface {
	Trade(ctx context.Context, viewer Actor, tradeID string) (HistoricalTrade, error)
}

// Actor is the authenticated member and network resolved by the host
// application's session. It is never taken from request bodies.
type Actor struct{ Network, Member string }
