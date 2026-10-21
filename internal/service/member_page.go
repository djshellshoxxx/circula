package service

import (
	"context"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// MemberPageQuery requests one keyset page of members, ordered by id. Set AfterID to page
// forward (members after it) or BeforeID to page backward (members before it). Setting both
// is an error. Limit 0 means 100; the maximum is 500.
type MemberPageQuery struct {
	Name     string // case-insensitive substring of the display name; empty matches all
	AfterID  string
	BeforeID string
	Limit    int
}

// DirectoryPage is one page of the active-member directory, ascending by id.
type DirectoryPage struct {
	Entries          []DirectoryEntry
	HasPrev, HasNext bool
}

// MemberPage is one page of members of any status, ascending by id.
type MemberPage struct {
	Members          []*domain.Member
	HasPrev, HasNext bool
}

// ListDirectoryPage pages the directory of active members for an active member (same
// visibility as ListDirectory).
func (s *Service) ListDirectoryPage(ctx context.Context, actor domain.Actor, netID string, q MemberPageQuery) (DirectoryPage, error) {
	var out DirectoryPage
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireActive(); err != nil {
			return err
		}
		ms, hasPrev, hasNext, err := pageMembers(tx, netID, []domain.MemberStatus{domain.MemberActive}, q)
		if err != nil {
			return err
		}
		for _, m := range ms {
			out.Entries = append(out.Entries, DirectoryEntry{MemberID: m.ID, DisplayName: m.DisplayName})
		}
		out.HasPrev, out.HasNext = hasPrev, hasNext
		return nil
	})
	return out, err
}

// CountDirectory counts the active members matching name, exactly, for an active member.
func (s *Service) CountDirectory(ctx context.Context, actor domain.Actor, netID, name string) (int, error) {
	return s.countMembers(ctx, actor, netID, []domain.MemberStatus{domain.MemberActive}, name, false)
}

// AdminMemberPage pages members of the given statuses (nil means every status) for an
// administrator, with roles and flags.
func (s *Service) AdminMemberPage(ctx context.Context, actor domain.Actor, netID string, statuses []domain.MemberStatus, q MemberPageQuery) (MemberPage, error) {
	var out MemberPage
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		ms, hasPrev, hasNext, err := pageMembers(tx, netID, statuses, q)
		out.Members, out.HasPrev, out.HasNext = ms, hasPrev, hasNext
		return err
	})
	return out, err
}

// CountMembers counts members of the given statuses (nil means every status) exactly, for an
// administrator. Used for totals such as the pending-request count, which is not capped.
func (s *Service) CountMembers(ctx context.Context, actor domain.Actor, netID string, statuses []domain.MemberStatus) (int, error) {
	return s.countMembers(ctx, actor, netID, statuses, "", true)
}

func (s *Service) countMembers(ctx context.Context, actor domain.Actor, netID string, statuses []domain.MemberStatus, name string, admin bool) (int, error) {
	var n int
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if admin {
			err = p.requireAdmin()
		} else {
			err = p.requireActive()
		}
		if err != nil {
			return err
		}
		cm, ok := tx.(store.MemberPager)
		if !ok {
			return domain.Errf(domain.CodeInternal, "store does not support member counts")
		}
		n, err = cm.CountMembers(netID, statuses, name)
		return err
	})
	return n, err
}

// pageMembers runs one keyset page inside tx. It fetches one row more than the page size to
// learn whether a further page exists in the paging direction, and asks for the row
// before/after the page's edge to learn whether a page exists the other way.
func pageMembers(tx store.Tx, netID string, statuses []domain.MemberStatus, q MemberPageQuery) (ms []*domain.Member, hasPrev, hasNext bool, err error) {
	if q.AfterID != "" && q.BeforeID != "" {
		return nil, false, false, domain.Invalid("page", "use either after or before, not both")
	}
	pager, ok := tx.(store.MemberPager)
	if !ok {
		return nil, false, false, domain.Errf(domain.CodeInternal, "store does not support member paging")
	}
	name := strings.TrimSpace(q.Name)
	limit := clampPage(Page{Limit: q.Limit})
	f := store.MemberPageFilter{Statuses: statuses, Name: name, AfterID: q.AfterID, BeforeID: q.BeforeID, Limit: limit + 1}
	ms, err = pager.PageMembers(netID, f)
	if err != nil {
		return nil, false, false, err
	}
	more := len(ms) > limit
	if more {
		if q.BeforeID != "" {
			// Backward: the store returns ascending rows, so the extra row is the farthest one, first.
			ms = ms[1:]
		} else {
			ms = ms[:limit]
		}
	}
	if len(ms) == 0 {
		// An empty page: nothing to page back to from here.
		return ms, false, false, nil
	}
	first, last := ms[0].ID, ms[len(ms)-1].ID
	// probe reports whether any match exists beyond the page edge in the given direction.
	probe := func(cur store.MemberPageFilter) (bool, error) {
		cur.Statuses, cur.Name, cur.Limit = statuses, name, 1
		rows, err := pager.PageMembers(netID, cur)
		return len(rows) > 0, err
	}
	if q.BeforeID != "" {
		// Backward page: `more` says there are older rows; the newer side needs a probe.
		hasNext, err = probe(store.MemberPageFilter{AfterID: last})
		return ms, more, hasNext, err
	}
	// Forward (or first) page: `more` says there are newer rows; the older side needs a probe
	// unless this is the first page.
	if q.AfterID != "" {
		hasPrev, err = probe(store.MemberPageFilter{BeforeID: first})
		if err != nil {
			return nil, false, false, err
		}
	}
	return ms, hasPrev, more, nil
}
