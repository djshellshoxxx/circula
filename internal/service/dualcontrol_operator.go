package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// weakensDualControl reports whether moving the dual-control threshold from
// old to nw makes the control easier to avoid: disabling it, or raising it.
func weakensDualControl(old, nw int64) bool {
	return old > 0 && (nw == 0 || nw > old)
}

// OperatorSetDualControl sets the network's dual-control threshold (0 disables
// it). It is the only way to loosen the control (SA2-01); network
// administrators may enable or lower it. actor must be domain.SystemActor; the
// CLI `circula dual-control` is its caller. A reason is mandatory and audited.
func (s *Service) OperatorSetDualControl(ctx context.Context, actor domain.Actor, network string, threshold int64, reason string) error {
	if err := requireSystem(actor); err != nil {
		return err
	}
	if err := requireReason(reason); err != nil {
		return err
	}
	if threshold < 0 || threshold > domain.MaxPolicyAmount {
		return domain.Invalid("threshold", "must be between 0 and the platform maximum")
	}
	return s.update(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, network)
		if err != nil {
			return err
		}
		pol := net.Policy
		pol.DualControlThreshold = threshold
		if err := pol.Validate(); err != nil {
			return err
		}
		v := net.PolicyVersion + 1
		if err := tx.AppendPolicy(net.ID, v, pol, "", "operator: "+reason, s.now()); err != nil {
			return err
		}
		return s.auditBreakGlass(tx, actor, net.ID, auditEntry{Type: domain.ActorSystem, Action: domain.ActPolicyChanged,
			TargetType: "policy", TargetID: net.ID, Reason: reason,
			Before: map[string]any{"dual_control_threshold": net.Policy.DualControlThreshold},
			After:  map[string]any{"dual_control_threshold": threshold}})
	})
}
