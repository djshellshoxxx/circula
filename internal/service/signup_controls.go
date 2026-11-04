package service

import (
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// checkNetworkJoinLimit refuses a new membership of netID with rate-limited
// once Config.NetworkJoinLimit memberships (invitation redemptions and join
// requests together, any status) were created within NetworkJoinWindow
// (SEC-006: an aggregate limit per network on top of the adapters' per-person
// and per-address limits). The count is read inside the caller's write unit,
// so concurrent joins cannot overshoot it. Off by default.
func (s *Service) checkNetworkJoinLimit(tx store.Tx, netID string) error {
	if s.cfg.NetworkJoinLimit <= 0 {
		return nil
	}
	n, err := tx.CountMembersJoinedSince(netID, s.now().Add(-s.cfg.NetworkJoinWindow))
	if err != nil {
		return err
	}
	if n >= s.cfg.NetworkJoinLimit {
		return domain.Errf(domain.CodeRateLimited, "this network is not accepting more members right now; try again later")
	}
	return nil
}

// openGrantRisk reports whether p combines open join mode with a sign-up grant.
func openGrantRisk(p domain.Policy) bool {
	return p.JoinMode == domain.JoinOpen && p.SignupGrant > 0
}

// checkGrantPolicy applies the sign-up grant controls of SEC-032 to a policy
// that is about to be stored; old is the policy in force (nil at creation).
//   - The grant must not exceed the platform cap (Config.MaxSignupGrant, minor
//     units; by default the general policy bound, so nothing changes until the
//     operator lowers it). postSignupGrant applies the same cap when a grant
//     is posted, so a grant stored before the cap was lowered pays at most
//     the cap, and nothing with a cap of zero.
//   - Combining open join mode with a grant needs confirm: a change that turns
//     the combination on, or raises the grant while it is on, without it is
//     refused at confirm_open_grant_risk. Other changes to a network already in
//     that state do not ask again.
//
// The second result is true when the confirmation was needed and given, so
// the caller records the audited risk confirmation.
func (s *Service) checkGrantPolicy(old *domain.Policy, nw domain.Policy, confirm bool) (bool, error) {
	if c := s.cfg.MaxSignupGrant; nw.SignupGrant > c {
		if c == 0 {
			return false, domain.Invalid("signup_grant", "must be 0: sign-up grants are turned off on this platform")
		}
		return false, domain.Invalid("signup_grant", "must not exceed the platform cap of "+
			domain.FormatAmount(c, nw.Decimals)+" "+nw.CurrencyName)
	}
	if !openGrantRisk(nw) {
		return false, nil
	}
	if old != nil && openGrantRisk(*old) && nw.SignupGrant <= old.SignupGrant {
		return false, nil
	}
	if !confirm {
		return false, domain.Invalid("confirm_open_grant_risk",
			"open join mode with a sign-up grant lets anyone collect the grant with new accounts; confirm the risk to save it")
	}
	return true, nil
}

// grantRiskAudit is the audit entry of a confirmed open-mode grant (SEC-032).
func grantRiskAudit(netID string, p domain.Policy, reason string) auditEntry {
	return auditEntry{Type: domain.ActorAdmin, Action: domain.ActOpenGrantRiskConfirmed, TargetType: "policy",
		TargetID: netID, Reason: reason, After: map[string]any{"join_mode": string(p.JoinMode), "signup_grant": p.SignupGrant}}
}
