package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// RevokeOtherOwnSessions ends the caller's other sessions in one atomic write.
// It rechecks the current session's owner, effective lifetime and authority in
// that write, then keeps that credential and leaves API tokens unchanged.
// Retained expired rows are also marked revoked, without materializing them.
// Pending MFA logins are removed so a previously issued challenge cannot
// recreate session authority after this operation commits.
//
// Every successful request is audited, including a zero-count request: it may
// still invalidate pending MFA challenges. Repeating the request returns zero
// once no other unrevoked sessions remain. A failed transaction returns zero.
func (s *Service) RevokeOtherOwnSessions(ctx context.Context, actor domain.Actor) (int64, error) {
	var revoked int64
	err := s.update(ctx, func(tx store.Tx) error {
		if _, _, err := s.ownSessionGate(tx, actor, domain.ScopeWrite); err != nil {
			return err
		}
		n, err := tx.RevokeOtherOwnedSessions(actor.PersonID, actor.SessionID, s.now())
		if err != nil {
			return err
		}
		if err := tx.DeleteMFAChallengesOf(actor.PersonID); err != nil {
			return err
		}
		if err := s.auditPlatform(tx, actor, auditEntry{Action: domain.ActOtherSessionsRevoked,
			TargetType: "person", TargetID: actor.PersonID,
			After: map[string]any{"current_retained": true, "revoked": n}}); err != nil {
			return err
		}
		revoked = n
		return nil
	})
	if err != nil {
		return 0, err
	}
	return revoked, nil
}
