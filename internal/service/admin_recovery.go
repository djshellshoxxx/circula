package service

import (
	"context"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	store "github.com/djshellshoxxx/circula/internal/repository"
)

// Administrator recovery when no administrator is available (GOV-021, T-17).
// See docs/ops/admin-recovery.md and docs/audit/consolidation-completion-2026-11/GOV-021.md.
//
//  1. A recorded, active member of an active network that has no active
//     administrator (GOV-020) asks to be appointed (RequestAdminRecovery,
//     HTML and JSON API). Identity is the authenticated account matched
//     against the network's membership, now and again at every later step.
//  2. A platform operator records the first approval (ApproveAdminRecovery,
//     operator CLI, System actor only, documented reason).
//  3. A second operator with a different label approves; that approval applies
//     the request in the same write unit: an ordinary, permanent admin role
//     assignment (grantRoleTx), and nothing else. No balance, entry,
//     transaction or credit limit is touched.
//
// The operator cannot start a request, cannot appoint on its own authority and
// cannot complete a request alone. Every step is audited at platform scope and
// at network scope and every current member of the network is told. Requests
// expire after domain.AdminRecoveryTTL. This flow is separate from ordinary
// break-glass (SEC-013): it neither needs nor creates a break-glass session or
// grant, and break-glass behaviour is unchanged (DEC-014, A3-F02 stay open).

// Notification kinds of administrator recovery (GOV-021).
const (
	NotifAdminRecoveryRequested = "network.admin_recovery_requested"
	NotifAdminRecoveryApproved  = "network.admin_recovery_approved"
	NotifAdminRecoveryApplied   = "network.admin_recovery_applied"
	NotifAdminRecoveryRefused   = "network.admin_recovery_refused"
	NotifAdminRecoveryExpired   = "network.admin_recovery_expired"
)

// adminRecoveryLink is the members' page of the process, inside the network.
const adminRecoveryLink = "/admin-recovery"

// adminRecoveryNotices are the fixed, neutral notice texts: they name no person
// and carry neither the operator's label nor reason (FR-038, SEC-012).
var adminRecoveryNotices = map[string][2]string{
	NotifAdminRecoveryRequested: {"Administrator recovery requested",
		"This network has no active administrator. A member asked the platform operator to appoint them as administrator. Two different operators must approve before anything changes, and balances and history are never changed by this process. See Administrator recovery for details."},
	NotifAdminRecoveryApproved: {"Administrator recovery: first approval",
		"A platform operator gave the first of two approvals to a request to appoint an administrator. A second, different operator must approve before anything changes."},
	NotifAdminRecoveryApplied: {"An administrator was appointed",
		"Two platform operators approved a member's request, and that member is now an administrator of this network. Balances and history are unchanged."},
	NotifAdminRecoveryRefused: {"Administrator recovery refused",
		"A platform operator refused a request to appoint an administrator. Nothing changed."},
	NotifAdminRecoveryExpired: {"Administrator recovery request expired",
		"A request to appoint an administrator was not approved within 7 days and has expired. Nothing changed."},
}

// AdminRecoveryInput is a member's request to be appointed administrator.
// The requester is the nominee; there is no field to name someone else.
type AdminRecoveryInput struct {
	IdempotencyKey string
}

// AdminRecoveryDecision is one operator's approval or refusal (operator CLI).
type AdminRecoveryDecision struct {
	Network   string // slug or id; must be the request's network
	RequestID string
	Operator  string // label of the deciding operator
	Reason    string // documented reason, mandatory; shown to the network's administrators
}

// AdminRecoveryOutcome reports an approval. Applied is true when it was the
// second approval and the requester now holds the administrator role.
type AdminRecoveryOutcome struct {
	Request *domain.AdminRecoveryRequest
	Applied bool
}

// AdminRecoveryView is what members of the network see of a request: who asked
// and when each step happened, without operator labels or reasons (FR-015).
type AdminRecoveryView struct {
	ID              string
	RequesterID     string
	Status          domain.AdminRecoveryStatus // as of now
	RequestedAt     time.Time
	ExpiresAt       time.Time
	FirstApprovedAt *time.Time
	ClosedAt        *time.Time
	Mine            bool
}

// AdminRecoveryListInput is an operator's audited read of recovery requests
// (SEC-013 a, b): a label and a documented reason are mandatory.
type AdminRecoveryListInput struct {
	Network  string // slug or id; "" lists every network
	OpenOnly bool   // only requests that can still be decided
	Operator string // label of the reading operator
	Reason   string // documented reason; shown to the administrators of every network listed
	Page     Page   // keyset: requests strictly before Page.AfterID, newest first
}

// AdminRecoveryOperatorView is the minimum an operator needs to decide a
// request. Each field and why it is there:
//   - ID, NetworkSlug: the arguments of approve and refuse;
//   - Status (as of now), ExpiresAt: whether and until when it can be decided;
//   - AdminRequired: false means approval will be refused (an administrator is active);
//   - Eligible: the requester is an active member (with a second factor when
//     required) now; false means approval will be refused. A yes/no instead of
//     the raw membership status;
//   - RequesterID, RequesterName: to match the request to the out-of-band
//     confirmation (ticket) of the person;
//   - MemberSince (a date, no time): a membership opened just before the
//     request, for example in an open network, is the main takeover signal;
//   - FirstOperator, FirstApprovedAt: whether this is the first or the second
//     approval, and that the second operator is a different one.
//
// No login, person id, contact detail, balance or other member data is read.
type AdminRecoveryOperatorView struct {
	ID              string
	NetworkSlug     string
	Status          domain.AdminRecoveryStatus
	ExpiresAt       time.Time
	AdminRequired   bool
	Eligible        bool
	RequesterID     string
	RequesterName   string
	MemberSince     time.Time // UTC date
	FirstOperator   string
	FirstApprovedAt *time.Time
}

func errRecoveryExpired() error {
	return domain.InvalidState("recovery request has expired; nothing was changed")
}

// auditRecovery writes a step taken by the operator (actor type
// platform_operator) or by the system (expiry, supersession) at platform scope
// and at the network's scope in the caller's unit (SEC-013 decision 5).
// Network administrators read the second copy in the audit log; members read
// it through ListAdminRecoveryEvents.
func (s *Service) auditRecovery(tx store.Tx, actor domain.Actor, netID string, typ domain.ActorType, e auditEntry) error {
	e.Type = typ
	if err := s.emit(tx, "", "", actor.RequestID, e); err != nil {
		return err
	}
	return s.emit(tx, netID, "", actor.RequestID, e)
}

// notifyCurrentMembers sends one inbox entry to every active and suspended
// member of the network, paging through all of them in the caller's unit so
// the notices commit with the step they announce.
func (s *Service) notifyCurrentMembers(tx store.Tx, netID, kind string) error {
	const pageSize = 500
	text := adminRecoveryNotices[kind]
	after := ""
	for {
		ms, err := tx.ListMembers(netID, store.MemberFilter{
			Statuses: []domain.MemberStatus{domain.MemberActive, domain.MemberSuspended}, AfterID: after, Limit: pageSize})
		if err != nil {
			return err
		}
		for _, m := range ms {
			if err := s.notify(tx, netID, m.ID, kind, text[0], text[1], adminRecoveryLink, ""); err != nil {
				return err
			}
		}
		if len(ms) < pageSize {
			return nil
		}
		after = ms[len(ms)-1].ID
	}
}

// recoveryNetworkErr refuses unless the network is active and has no active
// administrator (GOV-020). p.net was derived by open in this unit, so the
// administrator count is current.
func recoveryNetworkErr(n *domain.Network) error {
	if n.Status != domain.NetworkActive {
		return domain.InvalidState("network is %s; administrator recovery needs an active network", n.Status)
	}
	if !n.AdminRequired {
		return domain.InvalidState("the network has an active administrator; administrator recovery is only for a network without one")
	}
	return nil
}

// recoveryRequesterErr checks the requester against the network's recorded
// membership: an active member (not pending, suspended, departed or erased)
// with a second factor when administrators must have one.
func (s *Service) recoveryRequesterErr(tx store.Tx, m *domain.Member) error {
	if m == nil || m.Status != domain.MemberActive {
		return domain.Forbidden("only an active member of the network can be appointed through administrator recovery")
	}
	if s.cfg.RequireMFAForAdmins {
		on, err := s.hasMFA(tx, m.PersonID)
		if err != nil {
			return err
		}
		if !on {
			return domain.Forbidden("mfa-enrolment-required")
		}
	}
	return nil
}

// requestable applies every rule of a new request for the caller's principal.
func (s *Service) requestable(tx store.Tx, p *principal) error {
	if p.actor.System || p.member == nil {
		return domain.Forbidden("only a member of the network can ask for administrator recovery")
	}
	if err := s.recoveryRequesterErr(tx, p.member); err != nil {
		return err
	}
	if err := recoveryNetworkErr(p.net); err != nil {
		return err
	}
	// The member's own open request, through the one-open-request index: no
	// page of other members' requests can hide it.
	r, err := tx.GetOpenAdminRecovery(p.net.ID, p.member.ID)
	if err != nil {
		return err
	}
	if r != nil && s.now().Before(r.ExpiresAt) {
		return domain.Errf(domain.CodeConflict, "you already have an open administrator recovery request")
	}
	return nil
}

// RequestAdminRecovery records the caller's request to be appointed
// administrator of a network that has none (GOV-021). The caller is the
// nominee. Idempotent on the key: a retry returns the original request.
func (s *Service) RequestAdminRecovery(ctx context.Context, actor domain.Actor, netID string, in AdminRecoveryInput) (*domain.AdminRecoveryRequest, error) {
	var out *domain.AdminRecoveryRequest
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if p.actor.System || p.member == nil {
			return domain.Forbidden("only a member of the network can ask for administrator recovery")
		}
		out, err = idempotent(s, tx, netID, actor, "admin_recovery.request", in.IdempotencyKey,
			map[string]string{"network": netID, "member": p.member.ID},
			func() (string, error) {
				if err := s.requestable(tx, p); err != nil {
					return "", err
				}
				// An open request past its expiry would hold the one-open-request
				// index; close it first so the member can ask again.
				if _, err := s.closeExpiredRecovery(tx, actor, p.net.ID, p.member.ID); err != nil {
					return "", err
				}
				now := s.now()
				r := &domain.AdminRecoveryRequest{ID: s.id(), NetworkID: netID, RequesterID: p.member.ID,
					Status: domain.AdminRecoveryRequested, RequestedAt: now, ExpiresAt: now.Add(domain.AdminRecoveryTTL)}
				if err := tx.InsertAdminRecovery(r); err != nil {
					return "", err
				}
				if err := s.audit(tx, p, auditEntry{Type: domain.ActorMember, Action: domain.ActAdminRecoveryRequested,
					TargetType: "admin_recovery", TargetID: r.ID, IdemRef: keyRef(in.IdempotencyKey), Subjects: []string{p.member.ID},
					After: map[string]any{"status": string(r.Status), "member": p.member.ID, "expires_at": r.ExpiresAt.UnixMilli()}}); err != nil {
					return "", err
				}
				return r.ID, s.notifyCurrentMembers(tx, netID, NotifAdminRecoveryRequested)
			},
			func(ref string) (*domain.AdminRecoveryRequest, error) { return tx.GetAdminRecovery(ref) })
		return err
	})
	return out, err
}

// CanRequestAdminRecovery reports whether the caller may submit a request now,
// by the same rules RequestAdminRecovery applies. Read only.
func (s *Service) CanRequestAdminRecovery(ctx context.Context, actor domain.Actor, netID string) (bool, error) {
	can := false
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		// A request needs write scope, no forced password change and a network
		// that is not frozen; without them the form is simply not offered.
		writable := !actor.MustChangePassword && actor.Can(domain.ScopeWrite)
		if writable && s.checkNotFrozen(netID) == nil {
			can = s.requestable(tx, p) == nil
		}
		return nil
	})
	return can, err
}

// recoveryReader opens a network for the members' read views: any member who
// is not pending. Operators use the CLI listing instead.
func (s *Service) recoveryReader(tx store.Tx, actor domain.Actor, netID string) (*principal, error) {
	p, err := s.open(tx, actor, netID, domain.ScopeRead)
	if err != nil {
		return nil, err
	}
	if p.actor.System || p.member == nil || p.member.Status == domain.MemberPending {
		return nil, domain.Forbidden("no access to administrator recovery")
	}
	return p, nil
}

// ListAdminRecoveryRequests lists the network's recovery requests, newest
// first, to its members (GOV-021: visible to the network).
func (s *Service) ListAdminRecoveryRequests(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]AdminRecoveryView, error) {
	var out []AdminRecoveryView
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.recoveryReader(tx, actor, netID)
		if err != nil {
			return err
		}
		if err := validatePageIDs(pg.AfterID); err != nil {
			return err
		}
		rs, err := tx.ListAdminRecovery(netID, false, pg.AfterID, clampPage(pg))
		if err != nil {
			return err
		}
		now := s.now()
		out = make([]AdminRecoveryView, 0, len(rs))
		for _, r := range rs {
			v := AdminRecoveryView{ID: r.ID, RequesterID: r.RequesterID, Status: r.StatusAt(now), RequestedAt: r.RequestedAt,
				ExpiresAt: r.ExpiresAt, FirstApprovedAt: r.FirstApprovedAt, ClosedAt: r.ClosedAt, Mine: r.RequesterID == p.member.ID}
			if v.Status == domain.AdminRecoveryExpired && v.ClosedAt == nil {
				t := r.ExpiresAt
				v.ClosedAt = &t
			}
			out = append(out, v)
		}
		return nil
	})
	return out, err
}

// ListAdminRecoveryEvents returns the network-scope audit events of
// administrator recovery to any member of the network, in chain order. As in
// the members' own audit view, actor identities other than the caller, subject
// lists, hashes and other people's reasons are removed (FR-082, FR-015); the
// full events, with operator labels and reasons, are in the administrators'
// audit log.
func (s *Service) ListAdminRecoveryEvents(ctx context.Context, actor domain.Actor, netID string, q AuditQuery) ([]*domain.AuditEvent, error) {
	var out []*domain.AuditEvent
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.recoveryReader(tx, actor, netID)
		if err != nil {
			return err
		}
		evs, err := tx.ListAudit(netID, store.AuditFilter{TargetTypes: []string{"admin_recovery"}, AfterSeq: q.AfterSeq, Limit: q.Limit})
		if err != nil {
			return err
		}
		for _, e := range evs {
			own := e.ActorID != "" && e.ActorID == p.member.ID
			if !own {
				e.ActorID, e.Reason = "", ""
			}
			e.SubjectIDs, e.PrevHash, e.Hash, e.RequestID, e.IdemRef = nil, "", "", "", ""
			e.SummaryBefore, e.SummaryAfter = memberSafe(e.SummaryBefore), memberSafe(e.SummaryAfter)
		}
		out = evs
		return nil
	})
	return out, err
}

// ListAdminRecoveryOperator lists requests for the operator CLI: one network
// (slug or id) or, with Network "", every network, newest first, one keyset
// page at a time. It is operator access to member data, so it needs a label
// and a documented reason and is audited as admin_recovery.listed: once at
// platform scope and once in each network whose requests were shown, with the
// shown requesters as subjects (SEC-013 a, b, decision 5). It shows only the
// fields of AdminRecoveryOperatorView. System actor only.
func (s *Service) ListAdminRecoveryOperator(ctx context.Context, actor domain.Actor, in AdminRecoveryListInput) ([]AdminRecoveryOperatorView, error) {
	op, reason, err := recoveryDecision(actor, AdminRecoveryDecision{Operator: in.Operator, Reason: in.Reason})
	if err != nil {
		return nil, err
	}
	if err := validatePageIDs(in.Page.AfterID); err != nil {
		return nil, err
	}
	var out []AdminRecoveryOperatorView
	err = s.update(ctx, func(tx store.Tx) error {
		netID := ""
		if in.Network != "" {
			net, err := resolveNetwork(tx, in.Network)
			if err != nil {
				return err
			}
			netID = net.ID
		}
		rs, err := tx.ListAdminRecovery(netID, in.OpenOnly, in.Page.AfterID, clampPage(in.Page))
		if err != nil {
			return err
		}
		now := s.now()
		nets := map[string]*domain.Network{}
		var order []string                // networks in the order first shown
		shown := map[string][]string{}    // network -> requests shown
		subjects := map[string][]string{} // network -> requesters shown
		out = make([]AdminRecoveryOperatorView, 0, len(rs))
		for _, r := range rs {
			n, ok := nets[r.NetworkID]
			if !ok {
				if n, err = tx.GetNetwork(r.NetworkID); err != nil {
					return err
				}
				if err := s.lifecycle(tx, n); err != nil {
					return err
				}
				nets[r.NetworkID] = n
				order = append(order, r.NetworkID)
			}
			v := AdminRecoveryOperatorView{ID: r.ID, NetworkSlug: n.Slug, Status: r.StatusAt(now), ExpiresAt: r.ExpiresAt,
				AdminRequired: n.Status == domain.NetworkActive && n.AdminRequired, RequesterID: r.RequesterID,
				FirstOperator: r.FirstOperator, FirstApprovedAt: r.FirstApprovedAt}
			if m, err := tx.GetMember(r.NetworkID, r.RequesterID); err == nil {
				v.RequesterName = m.DisplayName
				v.MemberSince = m.JoinedAt.UTC().Truncate(24 * time.Hour)
				v.Eligible = s.recoveryRequesterErr(tx, m) == nil
			}
			shown[r.NetworkID] = append(shown[r.NetworkID], r.ID)
			subjects[r.NetworkID] = append(subjects[r.NetworkID], r.RequesterID)
			out = append(out, v)
		}
		scope := in.Network
		if scope == "" {
			scope = "all networks"
		}
		if err := s.emit(tx, "", "", actor.RequestID, auditEntry{Type: domain.ActorOperator, Action: domain.ActAdminRecoveryListed,
			TargetType: "admin_recovery", TargetID: netID, Reason: reason,
			After: map[string]any{"operator": op, "scope": scope, "open_only": in.OpenOnly, "requests": len(out), "networks": len(order)}}); err != nil {
			return err
		}
		for _, id := range order {
			if err := s.emit(tx, id, "", actor.RequestID, auditEntry{Type: domain.ActorOperator, Action: domain.ActAdminRecoveryListed,
				TargetType: "admin_recovery", TargetID: id, Reason: reason, Subjects: subjects[id],
				After: map[string]any{"operator": op, "requests": len(shown[id])}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// recoveryDecision validates the operator inputs shared by approval and refusal.
func recoveryDecision(actor domain.Actor, in AdminRecoveryDecision) (string, string, error) {
	if err := requireSystem(actor); err != nil {
		return "", "", err
	}
	op, err := cleanLabel("operator", in.Operator, 80)
	if err != nil {
		return "", "", err
	}
	if err := requireReason(in.Reason); err != nil {
		return "", "", err
	}
	return op, strings.TrimSpace(in.Reason), nil
}

// recoveryOf loads a request of the named network. A request of another
// network is not found, exactly like an unknown id (no cross-network reach).
func recoveryOf(tx store.Tx, network, id string) (*domain.Network, *domain.AdminRecoveryRequest, error) {
	net, err := resolveNetwork(tx, network)
	if err != nil {
		return nil, nil, err
	}
	if !domain.ValidRecordID(id) {
		return nil, nil, domain.NotFound("recovery request")
	}
	r, err := tx.GetAdminRecovery(id)
	if err != nil || r.NetworkID != net.ID {
		return nil, nil, domain.NotFound("recovery request")
	}
	return net, r, nil
}

// expireOne closes an open request at or past its expiry, audits it and tells
// the members. The caller's unit must then commit.
func (s *Service) expireOne(tx store.Tx, actor domain.Actor, r *domain.AdminRecoveryRequest) (bool, error) {
	if !r.Open() || s.now().Before(r.ExpiresAt) {
		return false, nil
	}
	ok, err := tx.CloseAdminRecovery(r.ID, r.Status, domain.AdminRecoveryExpired, "", domain.AdminRecoveryClosedExpired, r.ExpiresAt, "")
	if err != nil || !ok {
		return false, err
	}
	if err := s.auditRecovery(tx, actor, r.NetworkID, domain.ActorSystem, auditEntry{Action: domain.ActAdminRecoveryExpired, TargetType: "admin_recovery",
		TargetID: r.ID, Subjects: []string{r.RequesterID}, Before: map[string]any{"status": string(r.Status)},
		After: map[string]any{"status": string(domain.AdminRecoveryExpired), "closed_at": r.ExpiresAt.UnixMilli()}}); err != nil {
		return false, err
	}
	return true, s.notifyCurrentMembers(tx, r.NetworkID, NotifAdminRecoveryExpired)
}

// closeExpiredRecovery expires the member's own open request in the network,
// if it is past its expiry, found through the one-open-request index.
func (s *Service) closeExpiredRecovery(tx store.Tx, actor domain.Actor, netID, memberID string) (bool, error) {
	r, err := tx.GetOpenAdminRecovery(netID, memberID)
	if err != nil || r == nil {
		return false, err
	}
	return s.expireOne(tx, actor, r)
}

// ExpireAdminRecovery closes open requests past their expiry, auditing each and
// telling the members. Call it from the job runner. Idempotent; System only.
func (s *Service) ExpireAdminRecovery(ctx context.Context, actor domain.Actor) (int, error) {
	if err := requireSystem(actor); err != nil {
		return 0, err
	}
	// One short write unit per batch, walking the expired rows by ascending id
	// (keyset), so every expired request is reached even if one in a batch
	// could not be closed, and no unit grows with the backlog.
	const batch = 100
	total, after := 0, ""
	for {
		n, last, full := 0, "", false
		err := s.update(ctx, func(tx store.Tx) error {
			n = 0
			rs, err := tx.ExpiredOpenAdminRecovery(s.now(), after, batch)
			if err != nil {
				return err
			}
			full = len(rs) == batch
			for _, r := range rs {
				done, err := s.expireOne(tx, actor, r)
				if err != nil {
					return err
				}
				if done {
					n++
				}
				last = r.ID
			}
			return nil
		})
		if err != nil {
			return total, err
		}
		total += n
		if !full {
			return total, nil
		}
		after = last
	}
}

// ApproveAdminRecovery records one operator approval of a request (operator
// CLI, System actor only). The first approval only records; the second, by an
// operator whose label differs from the first one's, applies the request:
// the requester receives the permanent administrator role through the
// ordinary role-assignment path, other open requests of the network are
// superseded, and nothing else changes. Every rule is checked again in the
// same write unit: the network is still active and without an administrator,
// the requester is still an active member. A request past its expiry is
// closed and audited by the attempt and nothing is approved.
func (s *Service) ApproveAdminRecovery(ctx context.Context, actor domain.Actor, in AdminRecoveryDecision) (*AdminRecoveryOutcome, error) {
	op, reason, err := recoveryDecision(actor, in)
	if err != nil {
		return nil, err
	}
	var out *AdminRecoveryOutcome
	expired := false
	err = s.update(ctx, func(tx store.Tx) error {
		net, r, err := recoveryOf(tx, in.Network, in.RequestID)
		if err != nil {
			return err
		}
		if expired, err = s.expireOne(tx, actor, r); err != nil || expired {
			return err // an expiry close commits: nothing is approved but the closure is audited
		}
		if !r.Open() {
			return domain.InvalidState("recovery request is %s", r.Status)
		}
		p, err := s.open(tx, actor, net.ID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := recoveryNetworkErr(p.net); err != nil {
			return err
		}
		m, err := tx.GetMember(net.ID, r.RequesterID)
		if err != nil {
			return domain.InvalidState("the requester is no longer a member of the network")
		}
		if err := s.recoveryRequesterErr(tx, m); err != nil {
			return domain.InvalidState("the requester is no longer eligible: %v", err)
		}
		now := s.now()
		if r.Status == domain.AdminRecoveryRequested {
			ok, err := tx.ApproveAdminRecoveryFirst(r.ID, op, reason, now)
			if err != nil {
				return err
			}
			if !ok {
				return domain.InvalidState("recovery request is no longer waiting for a first approval")
			}
			if err := s.auditRecovery(tx, actor, net.ID, domain.ActorOperator, auditEntry{Action: domain.ActAdminRecoveryApproved, TargetType: "admin_recovery",
				TargetID: r.ID, Reason: reason, Subjects: []string{m.ID},
				Before: map[string]any{"status": string(domain.AdminRecoveryRequested)},
				After:  map[string]any{"status": string(domain.AdminRecoveryApproved), "operator": op, "approval": 1}}); err != nil {
				return err
			}
			if err := s.notifyCurrentMembers(tx, net.ID, NotifAdminRecoveryApproved); err != nil {
				return err
			}
			r, err = tx.GetAdminRecovery(r.ID)
			out = &AdminRecoveryOutcome{Request: r}
			return err
		}
		// Second approval: a different operator, then the appointment.
		if domain.SameOperator(op, r.FirstOperator) {
			return domain.Forbidden("the second approval must come from a different operator than the first approval (%s)", r.FirstOperator)
		}
		roleID, err := s.grantRoleTx(tx, p, m, domain.RoleAdmin, "", now)
		if err != nil {
			return err
		}
		ok, err := tx.CloseAdminRecovery(r.ID, domain.AdminRecoveryApproved, domain.AdminRecoveryApplied, op, reason, now, roleID)
		if err != nil {
			return err
		}
		if !ok { // cannot happen in one serialised write unit; rolls the grant back if it ever does
			return domain.InvalidState("recovery request is no longer waiting for a second approval")
		}
		if err := s.auditRecovery(tx, actor, net.ID, domain.ActorOperator, auditEntry{Action: domain.ActAdminRecoveryApplied, TargetType: "admin_recovery",
			TargetID: r.ID, Reason: reason, Subjects: []string{m.ID},
			Before: map[string]any{"status": string(domain.AdminRecoveryApproved)},
			After: map[string]any{"status": string(domain.AdminRecoveryApplied), "operator": op, "first_operator": r.FirstOperator,
				"approval": 2, "member": m.ID, "role": string(domain.RoleAdmin), "role_assignment": roleID}}); err != nil {
			return err
		}
		if err := s.emit(tx, net.ID, "", actor.RequestID, auditEntry{Type: domain.ActorOperator, Action: domain.ActRoleGranted,
			TargetType: "member", TargetID: m.ID, Reason: reason, Subjects: []string{m.ID},
			After: map[string]any{"role": string(domain.RoleAdmin), "admin_recovery": r.ID}}); err != nil {
			return err
		}
		if err := s.supersedeRecovery(tx, actor, net.ID, r.ID); err != nil {
			return err
		}
		if err := s.notifyCurrentMembers(tx, net.ID, NotifAdminRecoveryApplied); err != nil {
			return err
		}
		r, err = tx.GetAdminRecovery(r.ID)
		out = &AdminRecoveryOutcome{Request: r, Applied: true}
		return err
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, errRecoveryExpired()
	}
	return out, nil
}

// supersedeRecovery closes every other open request of the network once one
// has been applied, so a later loss of administrators cannot revive them.
func (s *Service) supersedeRecovery(tx store.Tx, actor domain.Actor, netID, appliedID string) error {
	const pageSize = 500
	after := "" // keyset over the network's open requests, id descending
	for {
		open, err := tx.ListAdminRecovery(netID, true, after, pageSize)
		if err != nil {
			return err
		}
		for _, o := range open {
			if o.ID == appliedID {
				continue
			}
			ok, err := tx.CloseAdminRecovery(o.ID, o.Status, domain.AdminRecoverySuperseded, "", domain.AdminRecoveryClosedSuperseded, s.now(), "")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if err := s.auditRecovery(tx, actor, netID, domain.ActorSystem, auditEntry{Action: domain.ActAdminRecoverySuperseded, TargetType: "admin_recovery",
				TargetID: o.ID, Subjects: []string{o.RequesterID}, Before: map[string]any{"status": string(o.Status)},
				After: map[string]any{"status": string(domain.AdminRecoverySuperseded), "applied": appliedID}}); err != nil {
				return err
			}
		}
		if len(open) < pageSize {
			return nil
		}
		after = open[len(open)-1].ID
	}
}

// RefuseAdminRecovery closes an open request without any change of rights
// (operator CLI, System actor only, documented reason). Any operator may
// refuse, including the one who gave the first approval.
func (s *Service) RefuseAdminRecovery(ctx context.Context, actor domain.Actor, in AdminRecoveryDecision) (*domain.AdminRecoveryRequest, error) {
	op, reason, err := recoveryDecision(actor, in)
	if err != nil {
		return nil, err
	}
	var out *domain.AdminRecoveryRequest
	expired := false
	err = s.update(ctx, func(tx store.Tx) error {
		net, r, err := recoveryOf(tx, in.Network, in.RequestID)
		if err != nil {
			return err
		}
		if expired, err = s.expireOne(tx, actor, r); err != nil || expired {
			return err
		}
		if !r.Open() {
			return domain.InvalidState("recovery request is %s", r.Status)
		}
		ok, err := tx.CloseAdminRecovery(r.ID, r.Status, domain.AdminRecoveryRefused, op, reason, s.now(), "")
		if err != nil {
			return err
		}
		if !ok {
			return domain.InvalidState("recovery request is no longer open")
		}
		if err := s.auditRecovery(tx, actor, net.ID, domain.ActorOperator, auditEntry{Action: domain.ActAdminRecoveryRefused, TargetType: "admin_recovery",
			TargetID: r.ID, Reason: reason, Subjects: []string{r.RequesterID},
			Before: map[string]any{"status": string(r.Status)},
			After:  map[string]any{"status": string(domain.AdminRecoveryRefused), "operator": op}}); err != nil {
			return err
		}
		if err := s.notifyCurrentMembers(tx, net.ID, NotifAdminRecoveryRefused); err != nil {
			return err
		}
		out, err = tx.GetAdminRecovery(r.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, errRecoveryExpired()
	}
	return out, nil
}
