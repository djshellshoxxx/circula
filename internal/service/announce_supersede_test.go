package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// An announcement can supersede an earlier one: the old one is hidden once the new one is in force.
func TestAnnouncementSupersedes(t *testing.T) {
	w := newWorld(t, nil)
	mk := func(title string, in service.AnnouncementInput) (*domain.Announcement, error) {
		in.Title, in.Body, in.IdempotencyKey = title, "body", w.Key()
		return w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, in)
	}
	old := tu.Must(mk("Old", service.AnnouncementInput{}))
	_, err := mk("Bad", service.AnnouncementInput{SupersedesID: "nope"})
	wantCode(t, err, domain.ErrValidation)
	repl := tu.Must(mk("New", service.AnnouncementInput{SupersedesID: old.ID, EffectiveAt: w.Clock.Now().Add(24 * time.Hour)}))
	if repl.SupersedesID != old.ID {
		t.Fatalf("supersedes = %q", repl.SupersedesID)
	}
	// before the replacement takes effect the old one stays visible
	if got := tu.Must(w.Svc.ListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{})); len(got) != 1 || got[0].ID != old.ID {
		t.Fatalf("before: %+v", got)
	}
	_, err = mk("Again", service.AnnouncementInput{SupersedesID: old.ID})
	wantCode(t, err, domain.ErrInvalidState) // already superseded
	w.Clock.Advance(25 * time.Hour)
	got := tu.Must(w.Svc.ListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{}))
	if len(got) != 1 || got[0].ID != repl.ID {
		t.Fatalf("after: %+v", got)
	}
}
