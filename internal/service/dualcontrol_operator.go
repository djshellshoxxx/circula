package service

import (
	"context"
	"strconv"

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
		now := s.now()
		v := net.PolicyVersion + 1
		var pending []store.PolicyVersionRow
		g, hasGovernance := tx.(store.GovRepo)
		if hasGovernance {
			latest, err := g.LatestPolicyVersion(net.ID)
			if err != nil {
				return err
			}
			v = latest + 1
			if latest > net.PolicyVersion {
				rows, err := g.ListPolicyVersions(net.ID)
				if err != nil {
					return err
				}
				withdrawn := withdrawnVersions(rows)
				for i, row := range rows {
					if !withdrawn[i] && row.EffectiveAt.After(now) {
						pending = append(pending, row)
					}
				}
			}
		}
		if err := tx.AppendPolicy(net.ID, v, pol, "", "operator: "+reason, now); err != nil {
			return err
		}
		// A higher immediate version supersedes older pending versions. Carry
		// each still-live notice forward as a new immutable version, changing
		// only the operator-controlled threshold. Otherwise this command would
		// silently cancel its unrelated rules or let a later activation undo
		// the operator's threshold. Original rows retain their author/reason;
		// the replacement is explicitly attributed to the operator (A1-08).
		preserved := make([]int, 0, len(pending))
		for _, row := range pending {
			v++
			row.Policy.DualControlThreshold = threshold
			if err := g.AppendPolicyNotice(net.ID, v, row.Policy, "",
				"operator: "+reason+"; preserves scheduled policy version "+strconv.Itoa(row.Version),
				row.AnnouncedAt, row.EffectiveAt, row.NoticeDays, row.NoticeReason); err != nil {
				return err
			}
			preserved = append(preserved, row.Version)
		}
		return s.auditBreakGlass(tx, actor, net.ID, auditEntry{Type: domain.ActorSystem, Action: domain.ActPolicyChanged,
			TargetType: "policy", TargetID: net.ID, Reason: reason,
			Before: map[string]any{"dual_control_threshold": net.Policy.DualControlThreshold},
			After:  map[string]any{"dual_control_threshold": threshold, "preserved_scheduled_versions": preserved}})
	})
}
