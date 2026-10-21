package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Inactivity rule (GOV-012). A network may set Policy.InactivityDays. A member
// whose newest activity (joining, sign-in, trade) is older than that many days is
// flagged: the member gets an inbox notice, an audit event is recorded, and the
// administrators can list the flagged members. The product never suspends or
// removes anyone for inactivity; an administrator decides with the ordinary
// member actions.

// NotifMemberInactive is the inbox kind for the inactivity notice.
const NotifMemberInactive = "member.inactive"

const inactivityBatch = 200

// inactivityCutoff is the time before which activity counts as inactive.
func inactivityCutoff(now time.Time, days int) time.Time {
	return now.Add(-time.Duration(days) * 24 * time.Hour)
}

// RunInactivityRule saves current sign-in activity, then flags the inactive
// members of every network that has an inactivity rule. A member is notified at
// most once per calendar year (dedupe key). It is the body of a scheduled job:
// it takes no actor, skips closed and frozen networks, and is safe to run often.
// It returns the number of members newly flagged.
func (s *Service) RunInactivityRule(ctx context.Context) (int, error) {
	var ids []string
	if err := s.update(ctx, func(tx store.Tx) (err error) {
		if err = tx.RecordPersonActivity(); err != nil {
			return err
		}
		ids, err = tx.ListNetworkIDs()
		return err
	}); err != nil {
		return 0, err
	}
	total := 0
	for _, netID := range ids {
		after := ""
		for {
			var n, scanned int
			var last string
			err := s.update(ctx, func(tx store.Tx) error {
				net, err := tx.GetNetwork(netID)
				if errors.Is(err, domain.ErrNotFound) {
					return nil
				}
				if err != nil {
					return err
				}
				if net.Policy.InactivityDays == 0 || net.Status != domain.NetworkActive || s.IsFrozen(netID) {
					return nil
				}
				now := s.now()
				ms, err := tx.ListInactiveMembers(netID, inactivityCutoff(now, net.Policy.InactivityDays), after, inactivityBatch)
				if err != nil {
					return err
				}
				scanned = len(ms)
				for _, m := range ms {
					last = m.ID
					created, err := tx.InsertNotification(store.Notification{ID: s.id(), NetworkID: netID, MemberID: m.ID,
						Kind: NotifMemberInactive, Title: "Your membership looks inactive",
						Body: fmt.Sprintf("You have not signed in or traded for %d days. You are still a member. Nothing has changed.",
							net.Policy.InactivityDays),
						Link: "/", DedupeKey: fmt.Sprintf("inactive:%d", now.UTC().Year()), CreatedAt: now})
					if err != nil {
						return err
					}
					if !created {
						continue
					}
					if err := s.emit(tx, netID, "", "job-inactivity", auditEntry{Type: domain.ActorSystem, Action: domain.ActMemberInactive,
						TargetType: "member", TargetID: m.ID, Subjects: []string{m.ID},
						After: map[string]any{"inactivity_days": net.Policy.InactivityDays}}); err != nil {
						return err
					}
					n++
				}
				return nil
			})
			if err != nil {
				return total, err
			}
			total += n
			if scanned < inactivityBatch {
				break
			}
			after = last
		}
	}
	return total, nil
}

// ListInactiveMembers lists the active members that the network's inactivity
// rule currently flags (administrator only). Without a rule it is empty.
func (s *Service) ListInactiveMembers(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]*domain.Member, error) {
	var out []*domain.Member
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		net, err := tx.GetNetwork(netID)
		if err != nil {
			return domain.NotFound("network")
		}
		if net.Policy.InactivityDays == 0 {
			return nil
		}
		out, err = tx.ListInactiveMembers(netID, inactivityCutoff(s.now(), net.Policy.InactivityDays), pg.AfterID, clampPage(pg))
		return err
	})
	return out, err
}
