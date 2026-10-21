package domain

import "time"

// Announcement is an append-only notice from the network's administrators to
// its members (API-064). It becomes visible to members at EffectiveAt. Deleting
// retracts it from member views but the row is kept (DeletedAt) so the audit
// trail stays complete.
type Announcement struct {
	ID          string
	NetworkID   string
	Title       string
	Body        string
	CreatedBy   string // member id of the administrator
	PublishedAt time.Time
	EffectiveAt time.Time
	DeletedAt   *time.Time
	DeletedBy   string
}

// Audit actions for announcements and the person time-zone preference.
const (
	ActAnnouncementCreated = "announcement.created"
	ActAnnouncementDeleted = "announcement.deleted"
	ActTimeZoneChanged     = "person.time_zone_changed"
)
