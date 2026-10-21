package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// CreateNetworkInput creates a network and its founding administrator.
type CreateNetworkInput struct {
	Name               string
	Slug               string
	Policy             *domain.Policy // nil = platform defaults
	CreatorDisplayName string
	// ConsumeSetupSecret makes the network creation also record the one-time
	// setup secret as spent, in the same transaction (SA2-16). It fails with a
	// conflict when the secret is already spent, so a crash can never leave a
	// network created with the secret still valid. Adapters that gate creation
	// on the setup secret must set it and must not call MarkSetupSecretUsed.
	ConsumeSetupSecret bool
}

// CreateNetwork creates a network in `setup` status with its system account
// (DOM-026), policy version 1, and the caller as the first active
// administrator (FR-001, FR-002). It writes no ledger entries. Who may call it
// (setup secret, sign-up policy) is an adapter decision; the service needs
// only an authenticated person.
func (s *Service) CreateNetwork(ctx context.Context, actor domain.Actor, in CreateNetworkInput) (*domain.Network, *domain.Member, error) {
	if actor.MustChangePassword {
		return nil, nil, domain.PasswordChangeRequired() // SEC-008
	}
	if actor.PersonID == "" || !actor.Can(domain.ScopeWrite) || actor.NetworkScope != "" {
		return nil, nil, domain.Forbidden("a full-scope person is required to create a network")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > 100 {
		return nil, nil, domain.Invalid("name", "must be 1 to 100 characters")
	}
	if !slugRE.MatchString(in.Slug) {
		return nil, nil, domain.Invalid("slug", "must be 3 to 40 lowercase letters, digits or hyphens")
	}
	disp, err := cleanName(in.CreatorDisplayName)
	if err != nil {
		return nil, nil, err
	}
	pol := domain.DefaultPolicy()
	if in.Policy != nil {
		pol = *in.Policy
	}
	if err := pol.Validate(); err != nil {
		return nil, nil, err
	}
	now := s.now()
	n := &domain.Network{ID: s.id(), Name: name, Slug: in.Slug, Status: domain.NetworkSetup, Policy: pol, PolicyVersion: 1, CreatedAt: now}
	m := &domain.Member{ID: s.id(), NetworkID: n.ID, PersonID: actor.PersonID, DisplayName: disp,
		Status: domain.MemberActive, JoinMethod: "founder", JoinedAt: now, Roles: []domain.Role{domain.RoleAdmin}}
	err = s.update(ctx, func(tx store.Tx) error {
		if in.ConsumeSetupSecret {
			used, err := setupUsed(tx)
			if err != nil {
				return err
			}
			if used {
				return domain.Errf(domain.CodeConflict, "the setup secret has already been used")
			}
		}
		if _, err := tx.GetNetworkBySlug(in.Slug); err == nil {
			return domain.Invalid("slug", "already in use")
		}
		if err := tx.InsertNetwork(n); err != nil {
			return err
		}
		if err := tx.AppendPolicy(n.ID, 1, pol, m.ID, "initial policy", now); err != nil {
			return err
		}
		sys := &domain.Account{ID: s.id(), NetworkID: n.ID, IsSystem: true}
		if err := tx.InsertAccount(sys); err != nil {
			return err
		}
		n.SystemAccountID = sys.ID
		if err := tx.InsertMember(m); err != nil {
			return err
		}
		m.AccountID = s.id()
		if err := tx.InsertAccount(&domain.Account{ID: m.AccountID, NetworkID: n.ID, MemberID: m.ID}); err != nil {
			return err
		}
		if err := tx.GrantRole(s.id(), n.ID, m.ID, domain.RoleAdmin, m.ID, now); err != nil {
			return err
		}
		if err := s.emit(tx, n.ID, m.ID, actor.RequestID, auditEntry{Type: domain.ActorAdmin, Action: domain.ActNetworkCreated,
			TargetType: "network", TargetID: n.ID, After: map[string]any{"status": n.Status, "slug": n.Slug},
			Subjects: []string{m.ID}}); err != nil {
			return err
		}
		if in.ConsumeSetupSecret {
			return s.auditPlatform(tx, actor, auditEntry{Action: domain.ActSetupSecretUsed, TargetType: "setup_secret"})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return n, m, nil
}

// ActivateNetwork moves setup -> active once an active administrator exists
// (FR-004). Memberships other than the founder cannot become active, and no
// transaction can be proposed, before this.
func (s *Service) ActivateNetwork(ctx context.Context, actor domain.Actor, netID string) (*domain.Network, error) {
	var out *domain.Network
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if p.net.Status != domain.NetworkSetup {
			return domain.InvalidState("network is %s; only a network in setup can be activated", p.net.Status)
		}
		if n, err := tx.CountActiveAdmins(netID); err != nil {
			return err
		} else if n < 1 {
			return domain.InvalidState("network needs at least one active administrator")
		}
		if err := tx.SetNetworkStatus(netID, domain.NetworkActive); err != nil {
			return err
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActNetworkActivated, TargetType: "network",
			TargetID: netID, Before: map[string]any{"status": domain.NetworkSetup}, After: map[string]any{"status": domain.NetworkActive}}); err != nil {
			return err
		}
		out, err = tx.GetNetwork(netID)
		return err
	})
	return out, err
}

// CloseNetwork closes a network (FR-068). It is refused while proposals or
// unresolved disputes exist; the error lists them. History and exports stay available.
func (s *Service) CloseNetwork(ctx context.Context, actor domain.Actor, netID, reason string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := p.requireLasting(); err != nil { // A3-02
			return err
		}
		if p.net.Status == domain.NetworkClosed {
			return domain.InvalidState("network is already closed")
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		var blockers []string
		props, err := tx.ListTransactions(netID, store.TxFilter{Statuses: []domain.TxStatus{domain.TxProposed, domain.TxDisputed}, Limit: 500})
		if err != nil {
			return err
		}
		for _, t := range props {
			blockers = append(blockers, string(t.Status)+" transaction "+t.ID)
		}
		if len(blockers) > 0 {
			return &domain.Error{Code: domain.CodeBlocked, Msg: "network has unresolved items", Blockers: blockers}
		}
		if err := tx.SetNetworkStatus(netID, domain.NetworkClosed); err != nil {
			return err
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActNetworkClosed, TargetType: "network",
			TargetID: netID, Reason: reason, Before: map[string]any{"status": p.net.Status}, After: map[string]any{"status": domain.NetworkClosed}})
	})
}

// GetNetwork returns the network for any of its members (any membership status).
func (s *Service) GetNetwork(ctx context.Context, actor domain.Actor, netID string) (*domain.Network, error) {
	var out *domain.Network
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		out = p.net
		return nil
	})
	return out, err
}

// NetworkInfo is the minimal public description used by join pages.
type NetworkInfo struct {
	ID, Name, Slug string
	Status         domain.NetworkStatus
	JoinMode       domain.JoinMode
}

// NetworkBySlug resolves a slug for the join flow. Unknown slugs return not-found.
func (s *Service) NetworkBySlug(ctx context.Context, slug string) (*NetworkInfo, error) {
	var out *NetworkInfo
	err := s.view(ctx, func(tx store.Tx) error {
		n, err := tx.GetNetworkBySlug(slug)
		if err != nil {
			return domain.NotFound("network")
		}
		out = &NetworkInfo{ID: n.ID, Name: n.Name, Slug: n.Slug, Status: n.Status, JoinMode: n.Policy.JoinMode}
		return nil
	})
	return out, err
}

// MyMemberships lists every membership of the caller across networks.
func (s *Service) MyMemberships(ctx context.Context, actor domain.Actor) ([]*domain.Member, error) {
	if actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return nil, domain.ErrForbidden
	}
	var out []*domain.Member
	err := s.view(ctx, func(tx store.Tx) (err error) {
		all, err := tx.ListMembershipsOfPerson(actor.PersonID)
		for _, m := range all {
			if actor.NetworkScope == "" || actor.NetworkScope == m.NetworkID {
				out = append(out, m)
			}
		}
		return err
	})
	return out, err
}

// PolicyChange replaces the network policy (all keys at once, validated).
type PolicyChange struct {
	Policy         domain.Policy
	Reason         string
	IdempotencyKey string `json:"-"`
}

// UpdatePolicy appends a new policy version (DOM-029, DOM-030). Changes apply
// prospectively; decimals cannot change once any posting exists (DOM-031).
// Out-of-bounds values are refused with the field and range.
func (s *Service) UpdatePolicy(ctx context.Context, actor domain.Actor, netID string, in PolicyChange) (*domain.Network, error) {
	var out *domain.Network
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := p.requireLasting(); err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "policy.update", in.IdempotencyKey, in, func() (string, error) {
			if p.net.Status == domain.NetworkClosed {
				return "", domain.InvalidState("network is closed")
			}
			if err := requireReason(in.Reason); err != nil {
				return "", err
			}
			if err := in.Policy.Validate(); err != nil {
				return "", err
			}
			if in.Policy.Decimals != p.net.Policy.Decimals && p.net.LedgerSeq > 0 {
				return "", domain.Invalid("decimals", "cannot change after the first posting")
			}
			if weakensDualControl(p.net.Policy.DualControlThreshold, in.Policy.DualControlThreshold) {
				// SA2-01: one administrator must not switch off or loosen the control
				// that exists to stop one administrator; the operator CLI can.
				return "", domain.Forbidden("dual control can only be loosened or disabled by the platform operator")
			}
			if t := in.Policy.DualControlThreshold; t > 0 && t != p.net.Policy.DualControlThreshold {
				// SEC-011: dual control needs a second approver; refuse a control nobody can satisfy.
				if n, err := tx.CountActiveAdmins(netID); err != nil {
					return "", err
				} else if n < 2 {
					return "", domain.Invalid("dual_control_threshold", "needs at least two active administrators")
				}
			}
			v := p.net.PolicyVersion + 1
			if err := tx.AppendPolicy(netID, v, in.Policy, p.memberID(), in.Reason, s.now()); err != nil {
				return "", err
			}
			if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActPolicyChanged, TargetType: "policy",
				TargetID: netID, Reason: in.Reason, IdemRef: keyRef(in.IdempotencyKey),
				Before: policyDiff(p.net.Policy, in.Policy, true), After: policyDiff(p.net.Policy, in.Policy, false)}); err != nil {
				return "", err
			}
			return netID, nil
		}, func(id string) (*domain.Network, error) { return tx.GetNetwork(id) })
		return err
	})
	return out, err
}

// policyDiff returns only the changed keys (old or new values).
func policyDiff(old, nw domain.Policy, before bool) map[string]any {
	o, n := policyMap(old), policyMap(nw)
	out := map[string]any{}
	for k, v := range n {
		if o[k] != v {
			if before {
				out[k] = o[k]
			} else {
				out[k] = v
			}
		}
	}
	return out
}

func policyMap(p domain.Policy) map[string]any {
	return map[string]any{
		"currency_name": p.CurrencyName, "decimals": p.Decimals, "default_credit_limit": p.DefaultCreditLimit,
		"max_tx_amount": p.MaxTxAmount, "proposal_expiry_days": p.ProposalExpiryDays,
		"dispute_window_days": p.DisputeWindowDays, "join_mode": string(p.JoinMode),
		"departure_rule": string(p.DepartureRule), "signup_grant": p.SignupGrant, "dual_control_threshold": p.DualControlThreshold,
		"limit_warning_days": p.LimitWarningDays, "default_flag_blocks_payer": p.DefaultFlagBlocksPayer,
		"directory_visibility": p.DirectoryVisibility,
	}
}
