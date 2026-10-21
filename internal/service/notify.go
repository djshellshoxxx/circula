package service

import (
	"context"
	"fmt"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Notification kinds (FR-038). Titles are fixed text; bodies never carry
// member-written free text, so an inbox entry cannot leak a note or statement.
const (
	NotifTradeProposed   = "trade.proposed"
	NotifTradeAccepted   = "trade.accepted"
	NotifTradeDeclined   = "trade.declined"
	NotifTradeCancelled  = "trade.cancelled"
	NotifDisputeOpened   = "dispute.opened"
	NotifDisputeReview   = "dispute.under_review"
	NotifDisputeResponse = "dispute.responded"
	NotifDisputeDecided  = "dispute.decided"
	NotifDisputeWithdraw = "dispute.withdrawn"
	NotifReReviewAsked   = "dispute.rereview_requested"
	NotifReReviewDecided = "dispute.rereview_decided"
	NotifReportHandled   = "report.handled"
	NotifLimitWarning    = "account.limit_warning"
)

// maxMarkIDs bounds one mark-read call.
const maxMarkIDs = 200

// notify writes one inbox entry for a member inside the caller's unit of
// work, so it commits or rolls back with the event it announces. link is a
// path inside the network (for example "/trades/<id>"). A non-empty dedupe key
// makes the call a no-op when the member already has that key.
func (s *Service) notify(tx store.Tx, netID, memberID, kind, title, body, link, dedupe string) error {
	if memberID == "" {
		return nil
	}
	_, err := tx.InsertNotification(store.Notification{ID: s.id(), NetworkID: netID, MemberID: memberID, Kind: kind,
		Title: title, Body: body, Link: link, DedupeKey: dedupe, CreatedAt: s.now()})
	return err
}

// notifyOthers notifies each distinct member id except skip.
func (s *Service) notifyOthers(tx store.Tx, netID string, ids []string, skip, kind, title, body, link string) error {
	seen := map[string]bool{skip: true, "": true}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		if err := s.notify(tx, netID, id, kind, title, body, link, ""); err != nil {
			return err
		}
	}
	return nil
}

// NotificationQuery selects inbox entries, newest first.
type NotificationQuery struct {
	UnreadOnly bool
	Page       Page
}

// inboxMember resolves the caller's own membership; pending and unknown
// members have no inbox.
func (p *principal) inboxMember() (*domain.Member, error) {
	if p.member == nil || p.member.Status == domain.MemberPending {
		return nil, domain.Forbidden("no inbox")
	}
	return p.member, nil
}

// ListNotifications returns the caller's own inbox entries (FR-038), newest
// first, keyset-paged by id.
func (s *Service) ListNotifications(ctx context.Context, actor domain.Actor, netID string, q NotificationQuery) ([]*store.Notification, error) {
	var out []*store.Notification
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		m, err := p.inboxMember()
		if err != nil {
			return err
		}
		out, err = tx.ListNotifications(netID, m.ID, store.NotificationFilter{UnreadOnly: q.UnreadOnly,
			AfterID: q.Page.AfterID, Limit: clampPage(q.Page)})
		return err
	})
	return out, err
}

// CountUnreadNotifications returns how many inbox entries the caller has not read.
func (s *Service) CountUnreadNotifications(ctx context.Context, actor domain.Actor, netID string) (int, error) {
	n := 0
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		m, err := p.inboxMember()
		if err != nil {
			return err
		}
		n, err = tx.CountUnreadNotifications(netID, m.ID)
		return err
	})
	return n, err
}

// MarkNotificationsRead marks the caller's own entries read and returns how
// many changed. An empty ids marks every unread entry. Other members' ids and
// unknown ids are ignored, so it is idempotent and leaks nothing. Reading state
// is a personal convenience, so it is not audited.
func (s *Service) MarkNotificationsRead(ctx context.Context, actor domain.Actor, netID string, ids []string) (int64, error) {
	if len(ids) > maxMarkIDs {
		return 0, domain.Invalid("ids", fmt.Sprintf("at most %d at a time; omit ids to mark everything", maxMarkIDs))
	}
	var n int64
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		m, err := p.inboxMember()
		if err != nil {
			return err
		}
		n, err = tx.MarkNotificationsRead(netID, m.ID, ids, s.now())
		return err
	})
	return n, err
}

// ListDisputesAwaitingMyResponse lists live disputes opened against the caller
// that have no response yet (FR-038 "disputes awaiting response").
func (s *Service) ListDisputesAwaitingMyResponse(ctx context.Context, actor domain.Actor, netID string) ([]*domain.Dispute, error) {
	var out []*domain.Dispute
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		m, err := p.inboxMember()
		if err != nil {
			return err
		}
		out, err = tx.ListDisputesAwaitingResponse(netID, m.ID, 100)
		return err
	})
	return out, err
}

// deciderIDs lists the active members who hold the reviewer or administrator role.
func deciderIDs(tx store.Tx, netID string) ([]string, error) {
	var out []string
	after := ""
	for {
		ms, err := tx.ListMembers(netID, store.MemberFilter{Statuses: []domain.MemberStatus{domain.MemberActive}, AfterID: after, Limit: 500})
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			if m.HasRole(domain.RoleAdmin) || m.HasRole(domain.RoleReviewer) {
				out = append(out, m.ID)
			}
		}
		if len(ms) < 500 {
			return out, nil
		}
		after = ms[len(ms)-1].ID
	}
}
