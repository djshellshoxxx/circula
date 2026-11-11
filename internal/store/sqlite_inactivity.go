package store

import (
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// RecordPersonActivity copies the newest last_seen_at of every person's live
// sessions into person_activity (never moving a value backwards).
func (t *txImpl) RecordPersonActivity() error {
	_, err := t.exec(`INSERT INTO person_activity(person_id, last_seen_at)
		SELECT person_id, MAX(last_seen_at) FROM sessions GROUP BY person_id
		ON CONFLICT(person_id) DO UPDATE SET last_seen_at = MAX(last_seen_at, excluded.last_seen_at)`)
	return err
}

// ListInactiveMembers returns active members of the network whose newest
// activity is older than cutoff, ordered by id. Activity is the latest of:
// joining, the newest recorded sign-in, and the newest transaction (proposed,
// accepted or any other kind) on the member's account.
func (t *txImpl) ListInactiveMembers(netID string, cutoff time.Time, afterID string, limit int) ([]*domain.Member, error) {
	q := `SELECT ` + memberCols + ` FROM members m WHERE m.network_id = ? AND m.status = 'active' AND m.id > ?
		AND MAX(m.joined_at,
			COALESCE((SELECT a.last_seen_at FROM person_activity a WHERE a.person_id = m.person_id), 0),
			COALESCE((SELECT MAX(s.last_seen_at) FROM sessions s WHERE s.person_id = m.person_id), 0),
			COALESCE((SELECT MAX(x.proposed_at) FROM transactions x
				JOIN accounts ac ON ac.id = x.payer_account_id OR ac.id = x.payee_account_id
				WHERE ac.member_id = m.id), 0)) < ?
		ORDER BY m.id LIMIT ?`
	return t.listMembers(netID, q, []any{netID, afterID, ms(cutoff), limitOf(limit)})
}
