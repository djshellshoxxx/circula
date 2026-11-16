package store

import (
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
)

// Compatibility aliases keep existing callers source-compatible.
type MemberPageFilter = repository.MemberPageFilter
type MemberPager = repository.MemberPager

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
		// instr over the folded name keeps the match a plain substring test, with no LIKE escaping
		// needed, and ignores case and accents beyond ASCII (see domain.FoldName).
		q += ` AND instr(circula_fold(m.display_name), ?) > 0`
		args = append(args, domain.FoldName(name))
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
