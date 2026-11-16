package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Notification kinds for suspension and closure (GOV-023..025).
const (
	NotifNetworkSuspended = "network.suspended"
	NotifNetworkResumed   = "network.resumed"
	NotifNetworkClosed    = "network.closed"
)

// Suspension (GOV-023, GOV-024).
//
// A suspension is an overlay on an active network: new proposals, acceptances,
// postings and joins are refused (requireNetworkActive / networkOpenErr), while
// reads, history and exports keep working. An administrator suspension and a
// platform suspension are independent; the administrator cannot lift the
// platform's, and the operator command does not lift the administrator's.

// notifyMembers sends one inbox entry to every active member (adminsOnly: only
// active administrators).
func (s *Service) notifyMembers(tx store.Tx, netID string, adminsOnly bool, kind, title, body string) error {
	after := ""
	for {
		ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive}, AfterID: after, Limit: 500})
		if err != nil {
			return err
		}
		for _, m := range ms {
			if adminsOnly && !m.HasRole(domain.RoleAdmin) {
				continue
			}
			if err := s.notify(tx, netID, m.ID, kind, title, body, "", ""); err != nil {
				return err
			}
		}
		if len(ms) < 500 {
			return nil
		}
		after = ms[len(ms)-1].ID
	}
}

// SuspendNetwork pauses trading in the network (GOV-023). Administrator only.
func (s *Service) SuspendNetwork(ctx context.Context, actor domain.Actor, netID, reason string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		reason = strings.TrimSpace(reason)
		if p.net.Status != domain.NetworkActive {
			return domain.InvalidState("network is %s; only an active network can be suspended", p.net.Status)
		}
		ok, err := tx.PutSuspension(netID, domain.Suspension{Kind: domain.SuspendedByAdmin, Reason: reason, By: p.memberID(), At: s.now()})
		if err != nil {
			return err
		}
		if !ok {
			return domain.InvalidState("network is already suspended by an administrator")
		}
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActNetworkSuspended, TargetType: "network",
			TargetID: netID, Reason: reason, Before: map[string]any{"suspended": p.net.Suspended()}, After: map[string]any{"suspended": true}}); err != nil {
			return err
		}
		return s.notifyMembers(tx, netID, false, NotifNetworkSuspended, "Trading is paused",
			"An administrator has paused trading: "+reason+". You can still read balances and history and download your data.")
	})
}

// ResumeNetwork lifts an administrator suspension. A platform suspension, if
// any, stays and is named in the refusal.
func (s *Service) ResumeNetwork(ctx context.Context, actor domain.Actor, netID, reason string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		if err := requireReason(reason); err != nil {
			return err
		}
		reason = strings.TrimSpace(reason)
		if p.net.SuspensionOf(domain.SuspendedByAdmin) == nil {
			if p.net.SuspensionOf(domain.SuspendedByPlatform) != nil {
				return domain.InvalidState("the platform operator suspended this network; only the operator can lift it")
			}
			return domain.InvalidState("network is not suspended")
		}
		if _, err := tx.DeleteSuspension(netID, domain.SuspendedByAdmin); err != nil {
			return err
		}
		still := p.net.SuspensionOf(domain.SuspendedByPlatform) != nil
		if err := s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActNetworkResumed, TargetType: "network",
			TargetID: netID, Reason: reason, Before: map[string]any{"suspended": true}, After: map[string]any{"suspended": still}}); err != nil {
			return err
		}
		if still {
			return nil // members are told when the platform lifts its suspension
		}
		return s.notifyMembers(tx, netID, false, NotifNetworkResumed, "Trading has resumed",
			"An administrator has resumed trading: "+reason+".")
	})
}

// PlatformSuspendInput is an operator suspension (GOV-024).
type PlatformSuspendInput struct {
	Network  string // slug or id
	Operator string // operator label
	Category string // one of domain.SuspensionCategories
	Reason   string
}

// PlatformSuspendNetwork suspends a network at platform level. Operator CLI
// only (System actor). The category must be one of the documented ones, so a
// reason about members' trades or balances has no category to hide behind. The
// event is written at platform scope and at network scope, where the
// administrators can read it, and every administrator gets a notice.
func (s *Service) PlatformSuspendNetwork(ctx context.Context, actor domain.Actor, in PlatformSuspendInput) error {
	if err := requireSystem(actor); err != nil {
		return err
	}
	cat := strings.TrimSpace(in.Category)
	valid := false
	for _, c := range domain.SuspensionCategories {
		valid = valid || c == cat
	}
	if !valid {
		return domain.Invalid("category", "required, one of "+strings.Join(domain.SuspensionCategories, ", "))
	}
	if err := requireReason(in.Reason); err != nil {
		return err
	}
	op, err := cleanLabel("operator", in.Operator, 80)
	if err != nil {
		return err
	}
	reason := strings.TrimSpace(in.Reason)
	return s.update(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, in.Network)
		if err != nil {
			return err
		}
		if net.Status != domain.NetworkActive {
			return domain.InvalidState("network is %s; only an active network can be suspended", net.Status)
		}
		ok, err := tx.PutSuspension(net.ID, domain.Suspension{Kind: domain.SuspendedByPlatform, Category: cat, Reason: reason, By: op, At: s.now()})
		if err != nil {
			return err
		}
		if !ok {
			return domain.InvalidState("network is already suspended by the platform")
		}
		if err := s.auditBreakGlass(tx, actor, net.ID, auditEntry{Action: domain.ActNetworkPlatformSuspended, TargetType: "network",
			TargetID: net.ID, Reason: reason, After: map[string]any{"category": cat, "operator": op}}); err != nil {
			return err
		}
		return s.notifyMembers(tx, net.ID, true, NotifNetworkSuspended, "The platform has suspended this network",
			fmt.Sprintf("Category: %s. Reason: %s. Members can still read balances and history and download their data.", cat, reason))
	})
}

// PlatformResumeNetwork lifts a platform suspension (operator CLI only).
func (s *Service) PlatformResumeNetwork(ctx context.Context, actor domain.Actor, network, operator, reason string) error {
	if err := requireSystem(actor); err != nil {
		return err
	}
	if err := requireReason(reason); err != nil {
		return err
	}
	op, err := cleanLabel("operator", operator, 80)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	return s.update(ctx, func(tx store.Tx) error {
		net, err := resolveNetwork(tx, network)
		if err != nil {
			return err
		}
		ok, err := tx.DeleteSuspension(net.ID, domain.SuspendedByPlatform)
		if err != nil {
			return err
		}
		if !ok {
			return domain.InvalidState("network is not suspended by the platform")
		}
		if err := s.auditBreakGlass(tx, actor, net.ID, auditEntry{Action: domain.ActNetworkPlatformResumed, TargetType: "network",
			TargetID: net.ID, Reason: reason, After: map[string]any{"operator": op}}); err != nil {
			return err
		}
		if net.SuspensionOf(domain.SuspendedByAdmin) != nil {
			return nil
		}
		return s.notifyMembers(tx, net.ID, false, NotifNetworkResumed, "Trading has resumed", "The platform has lifted its suspension: "+reason+".")
	})
}
