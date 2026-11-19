package repository

import "github.com/djshellshoxxx/circula/internal/domain"

// MemberPageFilter selects one keyset page of a network's members, ordered by id.
type MemberPageFilter struct {
	Statuses []domain.MemberStatus
	// Name is a case- and accent-insensitive substring of the display name. Empty matches all.
	Name string
	// AfterID selects members with id > AfterID (forward paging).
	AfterID string
	// BeforeID selects members with id < BeforeID (backward paging). Ignored when AfterID is set.
	BeforeID string
	// Limit is the number of rows to return. Callers ask for one more than the page size to learn
	// whether more rows exist.
	Limit int
}

// MemberPager is the keyset and count surface for members. The SQLite transaction implements
// it; the service asserts for it so the Tx interface itself does not change.
type MemberPager interface {
	// PageMembers returns matching members in ascending id order, roles loaded. With BeforeID
	// set it returns the last Limit matches before the cursor, still ascending.
	PageMembers(netID string, f MemberPageFilter) ([]*domain.Member, error)
	// CountMembers counts matching members exactly, without any row cap.
	CountMembers(netID string, statuses []domain.MemberStatus, name string) (int, error)
}
