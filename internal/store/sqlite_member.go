package store

import (
	"database/sql"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

const memberCols = `m.id, m.network_id, m.person_id, m.display_name, m.status, m.join_method, m.default_flag,
	COALESCE((SELECT a.id FROM accounts a WHERE a.member_id = m.id), ''), m.joined_at, m.departed_at`

func (t *txImpl) InsertMember(m *domain.Member) error {
	_, err := t.exec(`INSERT INTO members(id,network_id,person_id,display_name,status,join_method,default_flag,joined_at)
		VALUES (?,?,?,?,?,?,?,?)`, m.ID, m.NetworkID, m.PersonID, m.DisplayName, string(m.Status),
		m.JoinMethod, b2i(m.DefaultFlag), ms(m.JoinedAt))
	return err
}

func scanMember(r rowScanner) (*domain.Member, error) {
	var m domain.Member
	var status string
	var flag int
	var joined int64
	var dep sql.NullInt64
	if err := r.Scan(&m.ID, &m.NetworkID, &m.PersonID, &m.DisplayName, &status, &m.JoinMethod,
		&flag, &m.AccountID, &joined, &dep); err != nil {
		return nil, err
	}
	m.Status, m.DefaultFlag, m.JoinedAt, m.DepartedAt = domain.MemberStatus(status), flag == 1, fromMS(joined), ptrTime(dep)
	return &m, nil
}

// loadRoles fills Roles for the given members with one query.
func (t *txImpl) loadRoles(netID string, members []*domain.Member) error {
	if len(members) == 0 {
		return nil
	}
	idx := map[string]*domain.Member{}
	ids := make([]string, len(members))
	for i, m := range members {
		idx[m.ID] = m
		ids[i] = m.ID
	}
	in, args := inClause(ids)
	rows, err := t.query(`SELECT member_id, role FROM role_assignments
		WHERE network_id = ? AND revoked_at IS NULL AND member_id IN `+in+` ORDER BY role`, append([]any{netID}, args...)...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var mid, role string
		if err := rows.Scan(&mid, &role); err != nil {
			return err
		}
		idx[mid].Roles = append(idx[mid].Roles, domain.Role(role))
	}
	return rows.Err()
}

func (t *txImpl) one(netID string, r *sql.Row) (*domain.Member, error) {
	m, err := scanMember(r)
	if err != nil {
		return nil, notFound(err, "member")
	}
	if err := t.loadRoles(netID, []*domain.Member{m}); err != nil {
		return nil, err
	}
	return m, nil
}

func (t *txImpl) GetMember(netID, id string) (*domain.Member, error) {
	return t.one(netID, t.row(`SELECT `+memberCols+` FROM members m WHERE m.network_id = ? AND m.id = ?`, netID, id))
}

func (t *txImpl) GetMemberByPerson(netID, personID string) (*domain.Member, error) {
	return t.one(netID, t.row(`SELECT `+memberCols+` FROM members m WHERE m.network_id = ? AND m.person_id = ?`, netID, personID))
}

func (t *txImpl) ListMembers(netID string, f MemberFilter) ([]*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members m WHERE m.network_id = ?`
	args := []any{netID}
	if len(f.Statuses) > 0 {
		in, a := inClause(f.Statuses)
		q += ` AND m.status IN ` + in
		args = append(args, a...)
	}
	if f.AfterID != "" {
		q += ` AND m.id > ?`
		args = append(args, f.AfterID)
	}
	q += ` ORDER BY m.id LIMIT ?`
	args = append(args, limitOf(f.Limit))
	return t.listMembers(netID, q, args)
}

func (t *txImpl) ListMembershipsOfPerson(personID string) ([]*domain.Member, error) {
	return t.listMembers("", `SELECT `+memberCols+` FROM members m WHERE m.person_id = ? ORDER BY m.id`, []any{personID})
}

func (t *txImpl) listMembers(netID, q string, args []any) ([]*domain.Member, error) {
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	var out []*domain.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if netID != "" {
		return out, t.loadRoles(netID, out)
	}
	byNet := map[string][]*domain.Member{}
	for _, m := range out {
		byNet[m.NetworkID] = append(byNet[m.NetworkID], m)
	}
	for n, ms := range byNet {
		if err := t.loadRoles(n, ms); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (t *txImpl) SetMemberStatus(netID, id string, s domain.MemberStatus, departedAt *time.Time) error {
	_, err := t.exec(`UPDATE members SET status = ?, departed_at = COALESCE(?, departed_at) WHERE network_id = ? AND id = ?`,
		string(s), msPtr(departedAt), netID, id)
	return err
}

func (t *txImpl) SetDefaultFlag(netID, id string, v bool) error {
	_, err := t.exec(`UPDATE members SET default_flag = ? WHERE network_id = ? AND id = ?`, b2i(v), netID, id)
	return err
}

func (t *txImpl) GrantRole(roleID, netID, memberID string, r domain.Role, by string, at time.Time) error {
	_, err := t.exec(`INSERT INTO role_assignments(id,network_id,member_id,role,granted_at,granted_by) VALUES (?,?,?,?,?,?)`,
		roleID, netID, memberID, string(r), ms(at), by)
	return err
}

func (t *txImpl) RevokeRole(netID, memberID string, r domain.Role, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE role_assignments SET revoked_at = ?
		WHERE network_id = ? AND member_id = ? AND role = ? AND revoked_at IS NULL`, ms(at), netID, memberID, string(r))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) CountActiveAdmins(netID string) (int, error) {
	var n int
	err := t.row(`SELECT COUNT(*) FROM role_assignments r JOIN members m ON m.id = r.member_id
		WHERE r.network_id = ? AND r.role = 'admin' AND r.revoked_at IS NULL AND m.status = 'active'`, netID).Scan(&n)
	return n, err
}
