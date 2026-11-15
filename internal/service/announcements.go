package service

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const (
	maxAnnTitle = 120
	maxAnnBody  = 4000
	// An announcement may be scheduled at most this far ahead.
	maxAnnAhead = 366 * 24 * time.Hour
)

// AnnouncementInput is a new announcement (API-064).
type AnnouncementInput struct {
	Title string
	Body  string
	// EffectiveAt is when members start to see it; zero means now. Not in the past by more than
	// a minute (clock skew), not more than a year ahead.
	EffectiveAt time.Time
	// SupersedesID optionally names a live announcement this one replaces. The old one is
	// hidden once this one is in force; it is never edited.
	SupersedesID   string
	IdempotencyKey string `json:"-"`
}

func announcements(tx store.Tx) (store.AnnouncementRepo, error) {
	r, ok := tx.(store.AnnouncementRepo)
	if !ok {
		return nil, domain.Errf(domain.CodeInternal, "store does not support announcements")
	}
	return r, nil
}

func cleanAnnouncement(in AnnouncementInput, now time.Time) (title, body string, eff time.Time, err error) {
	title, body = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(title); n < 1 || n > maxAnnTitle {
		return "", "", eff, domain.Invalid("title", "must be 1 to 120 characters")
	}
	if n := utf8.RuneCountInString(body); n < 1 || n > maxAnnBody {
		return "", "", eff, domain.Invalid("body", "must be 1 to 4000 characters")
	}
	for _, r := range title {
		if unicode.IsControl(r) {
			return "", "", eff, domain.Invalid("title", "must not contain control characters")
		}
	}
	eff = in.EffectiveAt.UTC().Truncate(time.Millisecond)
	switch {
	case in.EffectiveAt.IsZero():
		eff = now
	case eff.Before(now.Add(-time.Minute)):
		return "", "", eff, domain.Invalid("effective_at", "must not be in the past")
	case eff.After(now.Add(maxAnnAhead)):
		return "", "", eff, domain.Invalid("effective_at", "must be within a year")
	}
	return title, body, eff, nil
}

// CreateAnnouncement publishes an announcement as an administrator (API-064). It is append-only:
// there is no edit, only DeleteAnnouncement, which retracts it from member views.
func (s *Service) CreateAnnouncement(ctx context.Context, actor domain.Actor, netID string, in AnnouncementInput) (*domain.Announcement, error) {
	var out *domain.Announcement
	err := s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		repo, err := announcements(tx)
		if err != nil {
			return err
		}
		out, err = idempotent(s, tx, netID, actor, "announcement.create", in.IdempotencyKey, in, func() (string, error) {
			now := s.now()
			title, body, eff, err := cleanAnnouncement(in, now)
			if err != nil {
				return "", err
			}
			sup := strings.TrimSpace(in.SupersedesID)
			if sup != "" {
				old, err := repo.GetAnnouncement(netID, sup)
				if err != nil || old.DeletedAt != nil {
					return "", domain.Invalid("supersedes_id", "must name a live announcement of this network")
				}
				if taken, err := repo.AnnouncementSuperseded(netID, sup); err != nil {
					return "", err
				} else if taken {
					return "", domain.InvalidState("announcement is already superseded")
				}
			}
			a := &domain.Announcement{ID: s.id(), NetworkID: netID, Title: title, Body: body, CreatedBy: p.memberID(),
				PublishedAt: now, EffectiveAt: eff, SupersedesID: sup}
			if err := repo.InsertAnnouncement(a); err != nil {
				return "", err
			}
			return a.ID, s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAnnouncementCreated,
				TargetType: "announcement", TargetID: a.ID, IdemRef: keyRef(in.IdempotencyKey),
				After: map[string]any{"effective_at": eff.UnixMilli(), "supersedes": sup}})
		}, func(id string) (*domain.Announcement, error) { return repo.GetAnnouncement(netID, id) })
		return err
	})
	return out, err
}

// DeleteAnnouncement retracts an announcement (administrators). The row stays for the audit
// trail; members and the administrator list no longer show it. A second delete is not-found.
func (s *Service) DeleteAnnouncement(ctx context.Context, actor domain.Actor, netID, id string) error {
	return s.update(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeWrite)
		if err != nil {
			return err
		}
		if err := p.requireAdmin(); err != nil {
			return err
		}
		repo, err := announcements(tx)
		if err != nil {
			return err
		}
		done, err := repo.RetractAnnouncement(netID, id, p.memberID(), s.now())
		if err != nil {
			return err
		}
		if !done {
			return domain.NotFound("announcement")
		}
		return s.audit(tx, p, auditEntry{Type: domain.ActorAdmin, Action: domain.ActAnnouncementDeleted,
			TargetType: "announcement", TargetID: id})
	})
}

// AdminListAnnouncements lists live (not retracted) announcements, scheduled ones included,
// newest first, for administrators.
func (s *Service) AdminListAnnouncements(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]*domain.Announcement, error) {
	return s.listAnnouncements(ctx, actor, netID, pg, true)
}

// ListAnnouncements lists the announcements in force (effective, not retracted), newest first,
// for active and suspended members. Administrators who want scheduled ones use
// AdminListAnnouncements.
func (s *Service) ListAnnouncements(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]*domain.Announcement, error) {
	return s.listAnnouncements(ctx, actor, netID, pg, false)
}

func (s *Service) listAnnouncements(ctx context.Context, actor domain.Actor, netID string, pg Page, admin bool) ([]*domain.Announcement, error) {
	var out []*domain.Announcement
	err := s.view(ctx, func(tx store.Tx) error {
		p, err := s.open(tx, actor, netID, domain.ScopeRead)
		if err != nil {
			return err
		}
		switch {
		case admin:
			err = p.requireAdmin()
		case p.actor.System:
		case p.member == nil || (p.member.Status != domain.MemberActive && p.member.Status != domain.MemberSuspended):
			err = domain.Forbidden("membership cannot read announcements")
		}
		if err != nil {
			return err
		}
		repo, err := announcements(tx)
		if err != nil {
			return err
		}
		f := store.AnnouncementFilter{AfterID: pg.AfterID, Limit: clampPage(pg), SupersededBefore: s.now()}
		if !admin {
			f.EffectiveBy = s.now()
		}
		out, err = repo.ListAnnouncements(netID, f)
		return err
	})
	return out, err
}
