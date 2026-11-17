package discovery

import (
	"context"
	"time"
)

// SearchParams is the fully authorized, validated query handed to a
// Repository. Visibility has already been decided by the service from the
// trusted Actor; the repository only applies it.
type SearchParams struct {
	Network string
	Viewer  string
	Now     time.Time
	Scope   string // ScopePublic, ScopeMine, ScopeAll

	Terms        []string
	Category     string
	Type         string
	Kind         string
	Condition    string
	Availability string
	AreaIDs      []string // already expanded from area/near filters; nil = no area filter
	AreaFilter   bool     // true when AreaIDs restricts results (even if empty)
	Tag          string
	Sort         string
	Limit        int

	After    *Position // keyset position, nil on first page
	Snapshot int64     // 0 on first page: repository fills with current revision
}

// SearchItem is one result row.
type SearchItem struct {
	Listing ListingRecord
	Meta    Meta
	Score   int64
	Tags    []string // approved tags only
	Cover   *MediaInfo
}

// SearchPage is one page plus the next keyset position when more exist.
type SearchPage struct {
	Items    []SearchItem
	Next     *Position
	Snapshot int64
}

// SaveInput is a validated metadata write for one listing.
type SaveInput struct {
	Network, ListingID, ActorID string
	ActorIsAdmin                bool
	Meta                        Meta
	Tags                        []string // normalized
	ExpectedVersion             int64    // 0 for first write
	MaxNetworkTags              int
	Now                         time.Time
}

// MediaQuota bounds one listing.
type MediaQuota struct {
	MaxCount int
	MaxBytes int64
}

// GCParams controls retention.
type GCParams struct {
	Now               time.Time
	ArchivedRetention time.Duration // media of archived listings
	PendingTTL        time.Duration
	Limit             int
}

// IdemResult is the idempotency lookup outcome.
type IdemResult struct {
	State       string // "new", "replay", "mismatch", "inflight"
	Status      int
	ContentType string
	Body        []byte
}

// AuditEntry is one append-only audit record.
type AuditEntry struct {
	ID, Network, Actor, Action, Target, Detail string
	At                                         time.Time
}

// Repository is the persistence adapter. Every method is network scoped; a
// foreign network's identifier must behave exactly like a missing one.
// Implementations must make each mutating method atomic and durable.
type Repository interface {
	// Trusted adapter sync of the core listing read model.
	UpsertListing(ctx context.Context, rec ListingRecord) error
	GetListing(ctx context.Context, network, id string) (ListingView, error)

	SaveDiscovery(ctx context.Context, in SaveInput) (Meta, error)
	Renew(ctx context.Context, network, id, actor string, expectedVersion int64, newExpires, now time.Time) (Meta, error)
	Search(ctx context.Context, p SearchParams) (SearchPage, error)

	CreateCategory(ctx context.Context, c Category, actor string, max int) error
	UpdateCategory(ctx context.Context, network, id string, name *string, enabled *bool, actor string) error
	ListCategories(ctx context.Context, network string, includeDisabled bool) ([]Category, error)
	CreateArea(ctx context.Context, a Area, actor string, max int) error
	UpdateArea(ctx context.Context, network, id string, name *string, enabled *bool, actor string) error
	ListAreas(ctx context.Context, network string, includeDisabled bool) ([]Area, error)

	ListTags(ctx context.Context, network string, status TagStatus, limit int) ([]Tag, error)
	ModerateTag(ctx context.Context, network, tag string, status TagStatus, actor string, now time.Time) error

	ReserveMedia(ctx context.Context, m MediaInfo, q MediaQuota) error
	ActivateMedia(ctx context.Context, network, listingID, mediaID string, now time.Time) error
	RemoveMedia(ctx context.Context, network, listingID, mediaID, actor string, now, retentionUntil time.Time) error
	GetMedia(ctx context.Context, network, listingID, mediaID string) (MediaInfo, error)

	ClaimGarbage(ctx context.Context, p GCParams) ([]MediaInfo, error)
	PathReferenced(ctx context.Context, path, exceptID string) (bool, error)
	DeleteMediaRow(ctx context.Context, id string) error

	IdemBegin(ctx context.Context, network, actor, key, reqHash string, now time.Time) (IdemResult, error)
	IdemFinish(ctx context.Context, network, actor, key string, status int, contentType string, body []byte) error
	IdemAbort(ctx context.Context, network, actor, key string) error

	Audit(ctx context.Context, network string, limit int) ([]AuditEntry, error)
}

// BlobStore stores sanitized image bytes under server-generated keys.
type BlobStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
