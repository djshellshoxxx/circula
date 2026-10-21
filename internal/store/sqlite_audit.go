package store

import (
	"database/sql"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) LastAudit(netID string) (int64, string, error) {
	var seq int64
	var hash string
	err := t.row(`SELECT seq, hash FROM audit_events WHERE network_id = ? ORDER BY seq DESC LIMIT 1`, netID).Scan(&seq, &hash)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	return seq, hash, err
}

func (t *txImpl) InsertAudit(e *domain.AuditEvent) error {
	if _, err := t.exec(`INSERT INTO audit_events(id,network_id,seq,occurred_at,actor_type,actor_id,action,target_type,
		target_id,request_id,idem_ref,summary_before,summary_after,reason,prev_hash,hash)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.NetworkID, e.Seq, ms(e.OccurredAt), string(e.ActorType),
		e.ActorID, e.Action, e.TargetType, e.TargetID, e.RequestID, e.IdemRef, e.SummaryBefore, e.SummaryAfter,
		e.Reason, e.PrevHash, e.Hash); err != nil {
		return err
	}
	for _, m := range e.SubjectIDs {
		if _, err := t.exec(`INSERT OR IGNORE INTO audit_subjects(audit_id,network_id,member_id) VALUES (?,?,?)`,
			e.ID, e.NetworkID, m); err != nil {
			return err
		}
	}
	return nil
}

func (t *txImpl) ListAudit(netID string, f AuditFilter) ([]*domain.AuditEvent, error) {
	q := `SELECT e.id,e.network_id,e.seq,e.occurred_at,e.actor_type,e.actor_id,e.action,e.target_type,e.target_id,
		e.request_id,e.idem_ref,e.summary_before,e.summary_after,e.reason,e.prev_hash,e.hash FROM audit_events e`
	args := []any{}
	if f.SubjectMember != "" {
		q += ` JOIN audit_subjects s ON s.audit_id = e.id AND s.member_id = ?`
		args = append(args, f.SubjectMember)
	}
	q += ` WHERE e.network_id = ? AND e.seq > ?`
	args = append(args, netID, f.AfterSeq)
	if f.Action != "" {
		q += ` AND e.action = ?`
		args = append(args, f.Action)
	}
	if f.ActorID != "" {
		q += ` AND e.actor_id = ?`
		args = append(args, f.ActorID)
	}
	if f.TargetID != "" {
		q += ` AND e.target_id = ?`
		args = append(args, f.TargetID)
	}
	if len(f.TargetTypes) > 0 {
		in, a := inClause(f.TargetTypes)
		q += ` AND e.target_type IN ` + in
		args = append(args, a...)
	}
	if !f.From.IsZero() {
		q += ` AND e.occurred_at >= ?`
		args = append(args, ms(f.From))
	}
	if !f.To.IsZero() {
		q += ` AND e.occurred_at < ?`
		args = append(args, ms(f.To))
	}
	q += ` ORDER BY e.seq LIMIT ?`
	args = append(args, limitOf(f.Limit))
	rows, err := t.query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.AuditEvent
	for rows.Next() {
		var e domain.AuditEvent
		var at int64
		var at2 string
		if err := rows.Scan(&e.ID, &e.NetworkID, &e.Seq, &at, &at2, &e.ActorID, &e.Action, &e.TargetType,
			&e.TargetID, &e.RequestID, &e.IdemRef, &e.SummaryBefore, &e.SummaryAfter, &e.Reason,
			&e.PrevHash, &e.Hash); err != nil {
			return nil, err
		}
		e.OccurredAt, e.ActorType = fromMS(at), domain.ActorType(at2)
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, t.loadSubjects(out)
}

// loadSubjects attaches the member ids each event concerns.
func (t *txImpl) loadSubjects(evs []*domain.AuditEvent) error {
	if len(evs) == 0 {
		return nil
	}
	idx := map[string]*domain.AuditEvent{}
	ids := make([]string, len(evs))
	for i, e := range evs {
		idx[e.ID], ids[i] = e, e.ID
	}
	in, args := inClause(ids)
	rows, err := t.query(`SELECT audit_id, member_id FROM audit_subjects WHERE audit_id IN `+in+` ORDER BY member_id`, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var a, m string
		if err := rows.Scan(&a, &m); err != nil {
			return err
		}
		idx[a].SubjectIDs = append(idx[a].SubjectIDs, m)
	}
	return rows.Err()
}

func (t *txImpl) GetIdem(netID, actorID, op, key string) (*IdemRecord, error) {
	var r IdemRecord
	var c, x int64
	err := t.row(`SELECT network_id,actor_id,operation,key,fingerprint,result_ref,created_at,expires_at
		FROM idempotency_keys WHERE network_id=? AND actor_id=? AND operation=? AND key=?`,
		netID, actorID, op, key).Scan(&r.NetworkID, &r.ActorID, &r.Operation, &r.Key, &r.Fingerprint, &r.ResultRef, &c, &x)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CreatedAt, r.ExpiresAt = fromMS(c), fromMS(x)
	return &r, nil
}

func (t *txImpl) PutIdem(r IdemRecord) error {
	_, err := t.exec(`INSERT INTO idempotency_keys(network_id,actor_id,operation,key,fingerprint,result_ref,created_at,expires_at)
		VALUES (?,?,?,?,?,?,?,?)`, r.NetworkID, r.ActorID, r.Operation, r.Key, r.Fingerprint, r.ResultRef,
		ms(r.CreatedAt), ms(r.ExpiresAt))
	return err
}

func (t *txImpl) DeleteIdem(netID, actorID, op, key string) error {
	_, err := t.exec(`DELETE FROM idempotency_keys WHERE network_id=? AND actor_id=? AND operation=? AND key=?`,
		netID, actorID, op, key)
	return err
}
