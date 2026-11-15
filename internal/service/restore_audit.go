package service

import (
	"context"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// RecordRestore appends a database.restored event to the network's audit chain
// after a restore (OPS-020). System or administrator actors only. The summary
// carries the backup's creation time only (no file names or secrets).
func (s *Service) RecordRestore(ctx context.Context, actor domain.Actor, netID string, backupCreated time.Time) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.openAny(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: p.actorType(domain.ActorAdmin), Action: domain.ActRestore, TargetType: "network", TargetID: netID,
			After: map[string]any{"status": "restored", "backup_created_at": backupCreated.UTC().Format(time.RFC3339)}})
	})
}
