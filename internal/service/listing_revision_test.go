package service_test

import (
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// API-027 / WAPI-02: overlapping owner edits with the same explicit revision
// can commit only once, even when the injected clock has not advanced.
func TestListingConcurrentConditionalOwnerEdits(t *testing.T) {
	w := newWorld(t, nil)
	price := int64(500)
	l := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Bike repair", Description: "Bring the bicycle", Price: &price, IdempotencyKey: w.Key(),
	}))
	revision := service.ListingRevision(l)
	before := tu.Snapshot(t, w.Store.DB())
	type result struct {
		listing *domain.Listing
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for _, title := range []string{"First owner edit", "Second owner edit"} {
		go func(title string) {
			<-start
			got, err := w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{
				ListingID: l.ID, Title: &title, ExpectedRevision: revision,
			})
			results <- result{got, err}
		}(title)
	}
	close(start)
	var winner *domain.Listing
	refused := 0
	for _, r := range []result{<-results, <-results} {
		switch {
		case r.err == nil:
			if winner != nil {
				t.Fatal("both owner edits committed with the same revision")
			}
			winner = r.listing
		case errors.Is(r.err, service.ErrListingChanged):
			if r.listing != nil {
				t.Fatal("refused edit returned a listing result")
			}
			refused++
		default:
			t.Fatalf("unexpected owner edit error: %v", r.err)
		}
	}
	if winner == nil || refused != 1 {
		t.Fatalf("want one commit and one stale refusal, got winner=%v refused=%d", winner != nil, refused)
	}
	got := tu.Must(w.Svc.GetListing(w.Ctx, w.a, w.net.ID, l.ID))
	if got.Title != winner.Title || got.Description != l.Description || got.Price == nil || *got.Price != price ||
		got.Status != domain.ListingActive || !got.UpdatedAt.Equal(l.UpdatedAt) {
		t.Fatalf("overlapping edits did not retain the winning field update: %+v", got)
	}
	after := tu.Snapshot(t, w.Store.DB())
	if after.Ledger != before.Ledger || after.AuditEvents != before.AuditEvents+1 {
		t.Fatal("one owner edit must append one audit and leave the ledger unchanged")
	}
}

func TestListingConditionalUpdatePreservesOwnerAndArchiveRules(t *testing.T) {
	w := newWorld(t, nil)
	l := tu.Must(w.Svc.CreateListing(w.Ctx, w.a, w.net.ID, service.ListingInput{
		Type: domain.ListingOffer, Title: "Bike repair", IdempotencyKey: w.Key(),
	}))
	original := service.ListingRevision(l)
	title := "Owner's newer title"
	l = tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{ListingID: l.ID, Title: &title}))
	refuse := func(name string, actor domain.Actor, expected string, want error) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			before := tu.Snapshot(t, w.Store.DB())
			got, err := w.Svc.UpdateListing(w.Ctx, actor, w.net.ID, service.ListingUpdate{
				ListingID: l.ID, Title: &title, ExpectedRevision: expected,
			})
			if !errors.Is(err, want) || got != nil {
				t.Fatalf("refused edit = %+v, %v; want nil, %v", got, err, want)
			}
			if changed := before.Changed(tu.Snapshot(t, w.Store.DB())); len(changed) != 0 {
				t.Fatalf("refused listing edit changed durable state: %v", changed)
			}
		})
	}
	refuse("other-owner", w.b, original, domain.ErrNotFound)
	refuse("stale-owner", w.a, original, service.ErrListingChanged)

	current := service.ListingRevision(l)
	archived := domain.ListingArchived
	l = tu.Must(w.Svc.UpdateListing(w.Ctx, w.a, w.net.ID, service.ListingUpdate{
		ListingID: l.ID, Status: &archived, ExpectedRevision: current,
	}))
	if l.Status != domain.ListingArchived {
		t.Fatal("fresh conditional archive did not commit")
	}
	refuse("stale-archived", w.a, current, service.ErrListingChanged)
	refuse("fresh-archived", w.a, service.ListingRevision(l), domain.ErrInvalidState)
	refuse("unconditional-archived", w.a, "", domain.ErrInvalidState)
}
