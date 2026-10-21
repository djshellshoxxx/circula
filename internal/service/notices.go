package service

import (
	"context"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const noticeListMax = 100

// notify records a security notice for personID in the caller's unit of work
// (SEC-012). detail must not contain secrets.
func (s *Service) notify(tx store.Tx, personID, kind, detail string) error {
	if len(detail) > 200 {
		detail = detail[:200]
	}
	return tx.InsertNotice(&domain.SecurityNotice{ID: s.id(), PersonID: personID, Kind: kind,
		CreatedAt: s.now(), Detail: detail})
}

// ListSecurityNotices returns the caller's own notices, newest first (at most
// 100). unseenOnly hides acknowledged ones.
func (s *Service) ListSecurityNotices(ctx context.Context, actor domain.Actor, unseenOnly bool) ([]*domain.SecurityNotice, error) {
	if actor.System || actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return nil, domain.Forbidden("read scope required")
	}
	var out []*domain.SecurityNotice
	err := s.view(ctx, func(tx store.Tx) (err error) {
		out, err = tx.ListNotices(actor.PersonID, unseenOnly, noticeListMax)
		return err
	})
	return out, err
}

// CountUnseenSecurityNotices backs the "new security notices" banner.
func (s *Service) CountUnseenSecurityNotices(ctx context.Context, actor domain.Actor) (int, error) {
	if actor.System || actor.PersonID == "" || !actor.Can(domain.ScopeRead) {
		return 0, domain.Forbidden("read scope required")
	}
	var n int
	err := s.view(ctx, func(tx store.Tx) (err error) { n, err = tx.CountUnseenNotices(actor.PersonID); return })
	return n, err
}

// AckSecurityNotices marks all of the caller's unseen notices as seen and
// returns how many changed. Acknowledging own notices is not audited: it
// changes no right, balance or credential.
func (s *Service) AckSecurityNotices(ctx context.Context, actor domain.Actor) (int64, error) {
	if actor.System || actor.PersonID == "" || !actor.Can(domain.ScopeWrite) {
		return 0, domain.Forbidden("write scope required")
	}
	var n int64
	err := s.update(ctx, func(tx store.Tx) (err error) { n, err = tx.AckNotices(actor.PersonID, s.now()); return })
	return n, err
}
