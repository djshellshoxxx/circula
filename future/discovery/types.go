package discovery

import (
	"errors"
	"fmt"
	"time"
)

// Role is a trusted adapter fact about the caller inside one network.
type Role string

// Roles.
const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
)

// Actor is the authenticated caller. It is produced by the host's session
// adapter (never from client JSON) and must be re-evaluated for every request.
// Active=false (suspended, departed, network inactive) denies everything.
type Actor struct {
	ID      string
	Network string
	Role    Role
	Active  bool
	// CSRF is the per-session anti-forgery token the host adapter issued.
	CSRF string
}

// Admin reports whether the actor is a network administrator.
func (a Actor) Admin() bool { return a.Role == RoleAdmin }

func (a Actor) valid() bool { return a.Active && a.ID != "" && a.Network != "" }

// Sentinel error kinds. Use errors.Is.
var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrInvalid     = errors.New("invalid input")
	ErrConflict    = errors.New("version conflict")
	ErrQuota       = errors.New("quota exceeded")
	ErrDuplicate   = errors.New("duplicate")
	ErrRateLimited = errors.New("rate limited")
	ErrBadCursor   = errors.New("invalid cursor")
	ErrRejected    = errors.New("rejected by policy")
)

// Error carries a sentinel kind with a client-safe field and message.
type Error struct {
	Kind          error
	Field, Detail string
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%v: %s: %s", e.Kind, e.Field, e.Detail)
	}
	return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
}

// Unwrap returns the sentinel kind.
func (e *Error) Unwrap() error { return e.Kind }

func invalid(field, detail string) error {
	return &Error{Kind: ErrInvalid, Field: field, Detail: detail}
}

// Listing status values (mirrors core, plus an adapter-set suspended state).
const (
	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusArchived  = "archived"
	StatusSuspended = "suspended"
)

// Kind is the core listing direction.
const (
	KindOffer   = "offer"
	KindRequest = "request"
)

// Type is the goods/service classification.
const (
	TypeGoods   = "goods"
	TypeService = "service"
	TypeAny     = "any" // category kind only
)

// Condition values for goods.
var conditions = map[string]bool{"new": true, "like_new": true, "good": true, "fair": true, "for_parts": true}

// Availability values.
const (
	AvailAvailable   = "available"
	AvailLimited     = "limited"
	AvailUnavailable = "unavailable"
)

// ListingRecord is the adapter-synced read model of a core listing. A host
// adapter fills it from the core listings table and the owner's membership
// state; this module never edits core listings.
type ListingRecord struct {
	ID, Network, OwnerID string
	Kind                 string
	Title, Description   string
	Status               string
	OwnerActive          bool
	Created, Updated     time.Time
	ArchivedAt           time.Time // zero unless archived
}

// Meta is the module-owned discovery metadata of a listing. A listing with no
// metadata row is an unclassified legacy listing and stays searchable.
type Meta struct {
	CategoryID    string
	Type          string
	Condition     string
	AreaID        string
	Availability  string
	AvailableFrom time.Time
	ExpiresAt     time.Time
	RenewedAt     time.Time
	RenewCount    int
	// Version is the optimistic-concurrency token; 0 means no row yet.
	Version int64
}

// TagStatus is the moderation state of a tag within a network.
type TagStatus string

// Tag statuses.
const (
	TagPending  TagStatus = "pending"
	TagApproved TagStatus = "approved"
	TagRejected TagStatus = "rejected"
)

// Tag is a network-scoped moderated tag.
type Tag struct {
	Network, Name string
	Status        TagStatus
	ModeratedBy   string
	Updated       time.Time
	Listings      int
}

// ListingTag is a tag attached to a listing together with its status.
type ListingTag struct {
	Name   string
	Status TagStatus
}

// Category is a controlled taxonomy node.
type Category struct {
	ID, Network, Name string
	Kind              string // goods|service|any
	Enabled           bool
	Created           time.Time
	Listings          int
}

// GeoCell is a deliberately coarse centroid in tenths of a degree
// (about 11 km). Listing coordinates are never accepted or stored.
type GeoCell struct{ Lat10, Lon10 int }

// Area is a network-owned coarse area.
type Area struct {
	ID, Network, Name string
	Enabled           bool
	Cell              *GeoCell
}

// MediaStatus values.
type MediaStatus string

// Media states. Only active media is ever served.
const (
	MediaPending  MediaStatus = "pending"
	MediaActive   MediaStatus = "active"
	MediaRemoved  MediaStatus = "removed"
	MediaDeleting MediaStatus = "deleting"
)

// MediaInfo describes one stored photo. Paths are server generated.
type MediaInfo struct {
	ID, Network, ListingID, OwnerID string
	SHA256, MIME, Alt               string
	Width, Height                   int
	ThumbWidth, ThumbHeight         int
	Bytes, ThumbBytes               int64
	StoragePath, ThumbPath          string
	Status                          MediaStatus
	Position                        int
	Created                         time.Time
	RetentionUntil                  time.Time
}

// ListingView is a listing with its discovery state, as shown to one viewer.
type ListingView struct {
	ListingRecord
	Meta
	Tags         []ListingTag
	Media        []MediaInfo
	Unclassified bool
	Expired      bool
	Stale        bool
	CanEdit      bool
}

// Config holds policy. Zero values take the documented provisional defaults.
type Config struct {
	CursorKeys        [][]byte // first signs, all verify (rotation); each >= 32 bytes
	CursorTTL         time.Duration
	MaxPhotos         int
	MaxListingBytes   int64
	MediaRetention    time.Duration // removed/archived media retained this long before GC
	PendingTTL        time.Duration // abandoned uploads reclaimed after this
	UploadsPerHour    int
	RenewExtend       time.Duration
	MaxExpiry         time.Duration
	StaleAfter        time.Duration
	MaxCategories     int
	MaxAreas          int
	MaxTagsPerListing int
	MaxTagsPerNetwork int
	DefaultPage       int
	MaxPage           int
	MaxWithinKM       int
	Now               func() time.Time
	NewID             func(prefix string) string
}

func (c Config) withDefaults() Config {
	if c.CursorTTL <= 0 {
		c.CursorTTL = 15 * time.Minute
	}
	if c.MaxPhotos <= 0 {
		c.MaxPhotos = 6
	}
	if c.MaxPhotos > 6 {
		c.MaxPhotos = 6 // spec ceiling
	}
	if c.MaxListingBytes <= 0 {
		c.MaxListingBytes = 24 << 20
	}
	if c.MediaRetention <= 0 {
		c.MediaRetention = 30 * 24 * time.Hour
	}
	if c.PendingTTL <= 0 {
		c.PendingTTL = time.Hour
	}
	if c.UploadsPerHour <= 0 {
		c.UploadsPerHour = 30
	}
	if c.RenewExtend <= 0 {
		c.RenewExtend = 30 * 24 * time.Hour
	}
	if c.MaxExpiry <= 0 {
		c.MaxExpiry = 365 * 24 * time.Hour
	}
	if c.StaleAfter <= 0 {
		c.StaleAfter = 90 * 24 * time.Hour
	}
	if c.MaxCategories <= 0 {
		c.MaxCategories = 200
	}
	if c.MaxAreas <= 0 {
		c.MaxAreas = 500
	}
	if c.MaxTagsPerListing <= 0 {
		c.MaxTagsPerListing = 8
	}
	if c.MaxTagsPerNetwork <= 0 {
		c.MaxTagsPerNetwork = 2000
	}
	if c.DefaultPage <= 0 {
		c.DefaultPage = 20
	}
	if c.MaxPage <= 0 {
		c.MaxPage = 50
	}
	if c.MaxWithinKM <= 0 {
		c.MaxWithinKM = 500
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	if c.NewID == nil {
		c.NewID = randomID
	}
	return c
}
