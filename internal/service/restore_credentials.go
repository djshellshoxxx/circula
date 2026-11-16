package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const ActRestoreCredentialsInvalidated = "database.restore_credentials_invalidated"

// RestoredCredentialCounts is the non-secret summary recorded by the restore command.
type RestoredCredentialCounts = store.RestoredCredentialCounts

// InvalidateRestoredCredentials ends bearer authority copied from a backup.
// The operator invokes this on the verified staging copy before swapping files.
// A failure, including audit insertion failure, rolls the entire change back.
// Passwords, enrolled MFA secrets and recovery codes retain snapshot semantics;
// operators must reconcile those and membership changes separately after rollback.
func (s *Service) InvalidateRestoredCredentials(ctx context.Context, actor domain.Actor) (RestoredCredentialCounts, error) {
	if err := requireSystem(actor); err != nil {
		return RestoredCredentialCounts{}, err
	}
	var out RestoredCredentialCounts
	err := s.update(ctx, func(tx store.Tx) error {
		var err error
		out, err = tx.InvalidateRestoredCredentials(s.now())
		if err != nil {
			return err
		}
		if err := s.auditPlatform(tx, actor, auditEntry{
			Type: domain.ActorOperator, Action: ActRestoreCredentialsInvalidated,
			TargetType: "credentials", Reason: "database restore: prevent restored bearer authority",
			After: map[string]any{"sessions": out.Sessions, "api_tokens": out.Tokens,
				"mfa_challenges": out.MFAChallenges, "break_glass_sessions": out.BreakGlassSessions},
		}); err != nil {
			return err
		}
		report := &LedgerReport{}
		if err := s.verifyAuditChain(tx, "", report); err != nil {
			return err
		}
		if len(report.Findings) != 0 {
			return domain.InvalidState("restored platform audit chain failed verification")
		}
		return nil
	})
	if err != nil {
		return RestoredCredentialCounts{}, err
	}
	return out, nil
}
