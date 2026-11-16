package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// setupUsed reports whether the newest platform setup event is a use. A reset
// after the last use re-arms the secret.
func setupUsed(tx store.Tx) (bool, error) {
	used, usedFound, err := tx.LastPlatformAuditSeq(domain.ActSetupSecretUsed)
	if err != nil || !usedFound {
		return false, err
	}
	reset, resetFound, err := tx.LastPlatformAuditSeq(domain.ActSetupSecretReset)
	if err != nil {
		return false, err
	}
	return !resetFound || reset < used, nil
}

// SetupSecretUsed reports whether the one-time setup secret has been used and
// not since reset by the operator (SEC-028). The state survives restarts.
func (s *Service) SetupSecretUsed(ctx context.Context) (bool, error) {
	var used bool
	err := s.view(ctx, func(tx store.Tx) error {
		var err error
		used, err = setupUsed(tx)
		return err
	})
	return used, err
}

// MarkSetupSecretUsed records, with an audit event in the same unit of work,
// that the setup secret created a network. It fails with a conflict when the
// secret is already used, so exactly one caller wins a race.
func (s *Service) MarkSetupSecretUsed(ctx context.Context, actor domain.Actor) error {
	if actor.PersonID == "" && !actor.System {
		return domain.Forbidden("sign-in required")
	}
	return s.update(ctx, func(tx store.Tx) error {
		used, err := setupUsed(tx)
		if err != nil {
			return err
		}
		if used {
			return domain.Errf(domain.CodeConflict, "the setup secret has already been used")
		}
		return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActSetupSecretUsed, TargetType: "setup_secret"})
	})
}

// ResetSetupSecret re-arms the setup secret. Operator CLI only: the actor must
// be the system actor. Resetting an unused secret is a no-op without an event.
func (s *Service) ResetSetupSecret(ctx context.Context, actor domain.Actor) (wasUsed bool, err error) {
	if !actor.System {
		return false, domain.Forbidden("operator command only")
	}
	err = s.update(ctx, func(tx store.Tx) error {
		used, err := setupUsed(tx)
		if err != nil || !used {
			return err
		}
		wasUsed = true
		return s.auditPlatform(tx, actor, auditEntry{Type: domain.ActorSystem, Action: domain.ActSetupSecretReset, TargetType: "setup_secret"})
	})
	return wasUsed, err
}
