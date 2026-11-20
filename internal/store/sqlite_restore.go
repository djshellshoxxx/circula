package store

import (
	"fmt"
	"time"

	"github.com/djshellshoxxx/circula/internal/repository"
)

// RestoredCredentialCounts retains the old public name with exact type identity.
type RestoredCredentialCounts = repository.RestoredCredentialCounts

// InvalidateRestoredCredentials is called inside one service write unit on a
// staging copy. Revocation does not depend on expiry timestamps: clock rollback
// must not revive a bearer credential or a temporary role grant (A3-F09).
func (t *txImpl) InvalidateRestoredCredentials(at time.Time) (RestoredCredentialCounts, error) {
	var out RestoredCredentialCounts
	for _, step := range []struct {
		query string
		args  []any
		count *int64
	}{
		{`UPDATE sessions SET revoked_at=? WHERE revoked_at IS NULL`, []any{ms(at)}, &out.Sessions},
		{`UPDATE api_tokens SET revoked_at=? WHERE revoked_at IS NULL`, []any{ms(at)}, &out.Tokens},
		{`DELETE FROM mfa_challenges`, nil, &out.MFAChallenges},
		{`UPDATE break_glass_sessions SET closed_at=?, closed_reason='database restore: credential invalidation' WHERE closed_at IS NULL`, []any{ms(at)}, &out.BreakGlassSessions},
	} {
		result, err := t.exec(step.query, step.args...)
		if err != nil {
			return RestoredCredentialCounts{}, err
		}
		*step.count, err = result.RowsAffected()
		if err != nil {
			return RestoredCredentialCounts{}, err
		}
	}
	// A trigger that silently ignores a write must fail the restore as well.
	var remaining int64
	if err := t.row(`SELECT
  (SELECT count(*) FROM sessions WHERE revoked_at IS NULL) +
  (SELECT count(*) FROM api_tokens WHERE revoked_at IS NULL) +
  (SELECT count(*) FROM mfa_challenges) +
  (SELECT count(*) FROM break_glass_sessions WHERE closed_at IS NULL)`).Scan(&remaining); err != nil {
		return RestoredCredentialCounts{}, err
	}
	if remaining != 0 {
		return RestoredCredentialCounts{}, fmt.Errorf("restore credential invalidation left %d credentials active", remaining)
	}
	return out, nil
}
