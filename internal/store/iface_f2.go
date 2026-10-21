package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// The repositories below are optional surfaces of a Tx, in the same way as
// MemberPager: the SQLite transaction implements them and the service asserts
// for them, so the Tx interface and its wrappers do not change.

// PrefsRepo stores per-person display preferences.
type PrefsRepo interface {
	// TimeZone returns the person's IANA zone name, "" when none is set.
	TimeZone(personID string) (string, error)
	SetTimeZone(personID, zone string, at time.Time) error
}

// AnnouncementFilter selects announcements, newest first by id.
type AnnouncementFilter struct {
	// EffectiveBy, when set, hides announcements that take effect after it.
	EffectiveBy time.Time
	// SupersededBefore, when set, hides announcements replaced by one that is live and
	// effective by that time.
	SupersededBefore time.Time
	// IncludeDeleted also returns retracted announcements.
	IncludeDeleted bool
	AfterID        string // keyset: ids strictly below
	Limit          int
}

// AnnouncementRepo covers append-only announcements (API-064).
type AnnouncementRepo interface {
	InsertAnnouncement(a *domain.Announcement) error
	GetAnnouncement(netID, id string) (*domain.Announcement, error)
	// AnnouncementSuperseded reports whether a live announcement already replaces id.
	AnnouncementSuperseded(netID, id string) (bool, error)
	// RetractAnnouncement stamps deleted_at/deleted_by; false when already retracted.
	RetractAnnouncement(netID, id, by string, at time.Time) (bool, error)
	ListAnnouncements(netID string, f AnnouncementFilter) ([]*domain.Announcement, error)
}

// OverviewParams are the thresholds for the administrator overview (FR-067).
type OverviewParams struct {
	Now          time.Time
	DefaultLimit int64         // policy default credit limit, magnitude in minor units
	StaleAfter   time.Duration // a live proposal older than this is "stale"
	InviteWithin time.Duration // an invitation ending within this is "expiring"
}

// OverviewCounts are exact counts computed from the ledger and dispute tables at
// read time (FR-067).
type OverviewCounts struct {
	StaleProposals  int // live proposed trades created at or before Now-StaleAfter
	OpenDisputes    int // open or under review
	PostingPending  int
	NearLimit       int // active members whose credit used is at or beyond their effective limit
	ExpiringInvites int // unused, unrevoked invitations ending within InviteWithin
	LastCheck       *ConsistencyRun
}

// OverviewRepo runs the read-only counts behind the administrator overview.
type OverviewRepo interface {
	OverviewCounts(netID string, p OverviewParams) (OverviewCounts, error)
}

var (
	_ PrefsRepo        = (*txImpl)(nil)
	_ AnnouncementRepo = (*txImpl)(nil)
	_ OverviewRepo     = (*txImpl)(nil)
)
