package store

import (
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// MemberPageFilter selects one keyset page of a network's members, ordered by id.
type MemberPageFilter struct {
	Statuses []domain.MemberStatus
	// Name is a case-insensitive substring of the display name. Empty matches all.
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

// memberWhere builds the shared WHERE clause (without the cursor) for member queries.
func memberWhere(netID string, statuses []domain.MemberStatus, name string) (string, []any) {
	q := ` WHERE m.network_id = ?`
	args := []any{netID}
	if len(statuses) > 0 {
		in, a := inClause(statuses)
		q += ` AND m.status IN ` + in
		args = append(args, a...)
	}
	if name = strings.TrimSpace(name); name != "" {
		// instr over lower() keeps the match a plain substring test, with no LIKE escaping needed.
		q += ` AND instr(lower(m.display_name), lower(?)) > 0`
		args = append(args, name)
	}
	return q, args
}

func (t *txImpl) PageMembers(netID string, f MemberPageFilter) ([]*domain.Member, error) {
	where, args := memberWhere(netID, f.Statuses, f.Name)
	q := `SELECT ` + memberCols + ` FROM members m` + where
	limit := f.Limit
	if limit <= 0 {
		limit = 1
	}
	backward := f.AfterID == "" && f.BeforeID != ""
	switch {
	case f.AfterID != "":
		q += ` AND m.id > ? ORDER BY m.id ASC LIMIT ?`
		args = append(args, f.AfterID, limit)
	case backward:
		q += ` AND m.id < ? ORDER BY m.id DESC LIMIT ?`
		args = append(args, f.BeforeID, limit)
	default:
		q += ` ORDER BY m.id ASC LIMIT ?`
		args = append(args, limit)
	}
	ms, err := t.listMembers(netID, q, args)
	if err != nil {
		return nil, err
	}
	if backward {
		for i, j := 0, len(ms)-1; i < j; i, j = i+1, j-1 {
			ms[i], ms[j] = ms[j], ms[i]
		}
	}
	return ms, nil
}

func (t *txImpl) CountMembers(netID string, statuses []domain.MemberStatus, name string) (int, error) {
	where, args := memberWhere(netID, statuses, name)
	var n int
	if err := t.row(`SELECT COUNT(*) FROM members m`+where, args...).Scan(&n); err != nil {
		return 0, mapErr(err)
	}
	return n, nil
}
